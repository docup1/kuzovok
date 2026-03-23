package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"kusovok/internal/application/admin"
	appauth "kusovok/internal/application/authapp"
	likeapp "kusovok/internal/application/likeapp"
	apppost "kusovok/internal/application/postapp"
	profileapp "kusovok/internal/application/profileapp"
	"kusovok/internal/domain/access"
	"kusovok/internal/domain/like"
	"kusovok/internal/domain/post"
	"kusovok/internal/domain/reply"
	userdomain "kusovok/internal/domain/user"
	"kusovok/internal/handlers"
	"kusovok/internal/infrastructure/auth"
	"kusovok/internal/infrastructure/config"
	"kusovok/internal/infrastructure/database"
	"kusovok/internal/infrastructure/storage"
	apperrors "kusovok/pkg/errors"
)

type App struct {
	cfg     *config.Config
	db      *sql.DB
	router  *handlers.Router
	cleanup *CleanupService
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.EnsureDir(cfg.Database.Path); err != nil {
		log.Fatalf("ensure db dir: %v", err)
	}

	db, err := database.Open(cfg.Database.Path, cfg.Database.MaxOpenConns)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrator := database.NewMigrator(db)
	if err := migrator.Backup(cfg.Database.Path, cfg.Database.BackupDir); err != nil {
		log.Printf("backup warning: %v", err)
	}
	if err := migrator.Run(); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	app := NewApp(cfg, db)
	defer app.Stop()

	fmt.Printf("Кузовок запущен на %s\n", handlers.PublicURL(cfg.Server.Addr))
	log.Fatal(http.ListenAndServe(cfg.Server.Addr, app.router))
}

func NewApp(cfg *config.Config, db *sql.DB) *App {
	userRepo := database.NewUserRepository(db)
	postRepo := database.NewPostRepository(db)
	likeRepo := database.NewLikeRepository(db)
	accessRepo := database.NewAccessRepository(db)
	replyRepo := database.NewReplyRepository(db)

	userDomainService := userdomain.NewService(userRepo)
	postDomainService := post.NewService(postRepo, accessRepo)
	likeDomainService := like.NewService(likeRepo)
	accessDomainService := access.NewService(accessRepo, userRepo)
	replyDomainService := reply.NewService(replyRepo, postRepo)

	jwtService := auth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.JWTExpireHours)
	cookieService := handlers.NewCookieService(cfg)
	imageStorage := handlers.NewImageStorage(cfg)

	registerUC := appauth.NewRegisterUseCase(userDomainService, jwtService)
	loginUC := appauth.NewLoginUseCase(userDomainService, jwtService)
	meUC := appauth.NewMeUseCase(userRepo, accessDomainService)
	appauth.SetAccessDeniedMessage(cfg.Messages.AccessDenied)

	createPostUC := apppost.NewCreatePostUseCase(postDomainService, replyDomainService, imageStorage, cfg.Images.LifetimeHours, cfg.Limits.PostContentMaxLength)
	feedUC := apppost.NewFeedUseCase(postDomainService)
	userPostsUC := apppost.NewUserPostsUseCase(postDomainService)

	toggleLikeUC := likeapp.NewToggleLikeUseCase(likeDomainService, postDomainService)

	getProfileUC := profileapp.NewGetProfileUseCase(userRepo)
	getProfileByUsernameUC := profileapp.NewGetProfileByUsernameUseCase(userRepo)
	updateProfileUC := profileapp.NewUpdateProfileUseCase(userRepo)

	getUsersUC := admin.NewGetUsersUseCase(userRepo)
	getLikesUC := admin.NewGetLikesUseCase(likeRepo)
	manageAllowedUC := admin.NewManageAllowedUsersUseCase(accessDomainService)

	userHandler := handlers.NewUserHandler(registerUC, loginUC, meUC, cookieService, handlers.GetMessages(cfg))
	postHandler := handlers.NewPostHandler(createPostUC, feedUC, userPostsUC, handlers.GetPostMessages(cfg), handlers.GetPostConfig(cfg))
	likeHandler := handlers.NewLikeHandler(toggleLikeUC, handlers.GetLikeMessages(cfg))
	adminHandler := handlers.NewAdminHandler(getUsersUC, getLikesUC, manageAllowedUC, handlers.GetAdminMessages(cfg))
	imageHandler := handlers.NewImageHandler(imageStorage)
	profileHandler := handlers.NewProfileHandler(getProfileUC, getProfileByUsernameUC, updateProfileUC)

	authMiddleware := handlers.NewAuthMiddleware(jwtService, cfg.Auth.CookieName, accessDomainService, handlers.GetAuthMessages(cfg))

	router := handlers.NewRouter(handlers.RouterDeps{
		UserHandler:    userHandler,
		PostHandler:    postHandler,
		LikeHandler:    likeHandler,
		AdminHandler:   adminHandler,
		ImageHandler:   imageHandler,
		ProfileHandler: profileHandler,
		AuthMiddleware: authMiddleware,
		StaticDir:      "static",
	})

	cleanup := NewCleanupService(postRepo, imageStorage, cfg.Cleanup.IntervalMinutes)
	cleanup.Start()

	return &App{
		cfg:     cfg,
		db:      db,
		router:  router,
		cleanup: cleanup,
	}
}

func (a *App) Stop() {
	if a.cleanup != nil {
		a.cleanup.Stop()
	}
}

type CleanupService struct {
	postRepo     *database.PostRepository
	imageStorage *storage.ImageStorage
	interval     int
	stopCh       chan struct{}
}

func NewCleanupService(postRepo *database.PostRepository, imageStorage *storage.ImageStorage, intervalMinutes int) *CleanupService {
	return &CleanupService{
		postRepo:     postRepo,
		imageStorage: imageStorage,
		interval:     intervalMinutes,
		stopCh:       make(chan struct{}),
	}
}

func (s *CleanupService) Start() {
	go func() {
		ticker := time.NewTicker(time.Duration(s.interval) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *CleanupService) Stop() {
	close(s.stopCh)
}

func (s *CleanupService) cleanup() {
	ctx := context.Background()
	now := time.Now().UTC()

	expired, err := s.postRepo.GetExpiredImages(ctx, now)
	if err != nil {
		log.Printf("cleanup: get expired images: %v", err)
		return
	}

	for _, img := range expired {
		if err := s.imageStorage.Delete(img.ImageURL); err != nil {
			log.Printf("cleanup: delete image %s: %v", img.ImageURL, err)
		}
		if err := s.postRepo.ClearImage(ctx, img.PostID); err != nil {
			log.Printf("cleanup: clear image db record %d: %v", img.PostID, err)
		}
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"message":"` + message + `"}`))
}

func writeAppError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		writeError(w, appErr.StatusCode, appErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
