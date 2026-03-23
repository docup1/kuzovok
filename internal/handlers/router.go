package handlers

import (
	"net/http"
	"strings"
)

type Router struct {
	mux            *http.ServeMux
	userHandler    *UserHandler
	postHandler    *PostHandler
	likeHandler    *LikeHandler
	adminHandler   *AdminHandler
	imageHandler   *ImageHandler
	authMiddleware *AuthMiddleware
	staticDir      string
}

type RouterDeps struct {
	UserHandler    *UserHandler
	PostHandler    *PostHandler
	LikeHandler    *LikeHandler
	AdminHandler   *AdminHandler
	ImageHandler   *ImageHandler
	AuthMiddleware *AuthMiddleware
	StaticDir      string
}

func NewRouter(deps RouterDeps) *Router {
	mux := http.NewServeMux()

	r := &Router{
		mux:            mux,
		userHandler:    deps.UserHandler,
		postHandler:    deps.PostHandler,
		likeHandler:    deps.LikeHandler,
		adminHandler:   deps.AdminHandler,
		imageHandler:   deps.ImageHandler,
		authMiddleware: deps.AuthMiddleware,
		staticDir:      deps.StaticDir,
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.mux.Handle("/img/", r.imageHandler)

	r.mux.HandleFunc("/api/register", r.userHandler.Register)
	r.mux.HandleFunc("/api/login", r.userHandler.Login)
	r.mux.HandleFunc("/api/logout", r.userHandler.Logout)

	r.mux.HandleFunc("/api/me", r.authMiddleware.Authenticate(r.userHandler.Me))
	r.mux.HandleFunc("/api/posts", r.authMiddleware.Authenticate(r.authMiddleware.RequireAllowedUser(r.postHandler.Posts)))
	r.mux.HandleFunc("/api/feed", r.authMiddleware.Authenticate(r.authMiddleware.RequireAllowedUser(r.postHandler.Feed)))
	r.mux.HandleFunc("/api/like", r.authMiddleware.Authenticate(r.authMiddleware.RequireAllowedUser(r.likeHandler.Toggle)))

	r.mux.HandleFunc("/api/admin/users", r.authMiddleware.Authenticate(r.authMiddleware.RequireAdmin(r.adminHandler.Users)))
	r.mux.HandleFunc("/api/admin/likes", r.authMiddleware.Authenticate(r.authMiddleware.RequireAdmin(r.adminHandler.Likes)))
	r.mux.HandleFunc("/api/admin/allowed-users", r.authMiddleware.Authenticate(r.authMiddleware.RequireAdmin(r.adminHandler.AllowedUsers)))
	r.mux.HandleFunc("/api/admin/allowed-users/", r.authMiddleware.Authenticate(r.authMiddleware.RequireAdmin(r.adminHandler.AllowedUserItem)))

	r.mux.HandleFunc("/", r.serveStatic)
}

func (r *Router) serveStatic(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimSuffix(req.URL.Path, "/")

	if path == "/" || path == "" {
		http.ServeFile(w, req, r.staticDir+"/index.html")
		return
	}
	if path == "/admin" || path == "/admin.html" {
		http.ServeFile(w, req, r.staticDir+"/admin.html")
		return
	}

	http.ServeFile(w, req, r.staticDir+req.URL.Path)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) ServeMux() *http.ServeMux {
	return r.mux
}

func PublicURL(addr string) string {
	if addr == "" {
		return "http://localhost:8080"
	}
	if len(addr) > 0 && addr[0] == ':' {
		return "http://localhost" + addr
	}
	if len(addr) > 7 && (addr[:7] == "http://" || addr[:8] == "https://") {
		return addr
	}
	return "http://" + addr
}
