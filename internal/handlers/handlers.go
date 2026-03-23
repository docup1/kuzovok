package handlers

import (
	"kusovok/internal/infrastructure/auth"
	"kusovok/internal/infrastructure/config"
	"kusovok/internal/infrastructure/storage"
)

type UserMessages struct {
	RegisterSuccess     string
	LoginSuccess        string
	LogoutSuccess       string
	ErrorInvalidData    string
	ErrorRequiredFields string
	ErrorPasswordShort  string
	ErrorUserExists     string
	ErrorInvalidCreds   string
	ErrorUnauthorized   string
}

type PostMessages struct {
	PostCreated        string
	ErrorInvalidData   string
	ErrorPostEmpty     string
	ErrorPostTooLong   string
	ErrorImageTooLarge string
	ErrorInvalidImage  string
	ErrorServer        string
}

type PostConfig struct {
	MaxImageSize      int64
	MultipartBodySize int64
}

type LikeMessages struct {
	ErrorInvalidData   string
	ErrorInvalidPostID string
	ErrorServer        string
}

type AdminMessages struct {
	UserAdded          string
	RoleUpdated        string
	UserRemoved        string
	ErrorInvalidData   string
	ErrorInvalidUserID string
	ErrorInvalidRole   string
	ErrorRouteNotFound string
	ErrorServer        string
}

type AuthMessages struct {
	ErrorUnauthorized string
	ErrorInvalidToken string
	AccessDenied      string
	AdminDenied       string
}

func GetMessages(cfg *config.Config) UserMessages {
	return UserMessages{
		RegisterSuccess:     cfg.Messages.RegisterSuccess,
		LoginSuccess:        cfg.Messages.LoginSuccess,
		LogoutSuccess:       cfg.Messages.LogoutSuccess,
		ErrorInvalidData:    cfg.Messages.ErrorInvalidData,
		ErrorRequiredFields: cfg.Messages.ErrorRequiredFields,
		ErrorPasswordShort:  cfg.Messages.ErrorPasswordShort,
		ErrorUserExists:     cfg.Messages.ErrorUserExists,
		ErrorInvalidCreds:   cfg.Messages.ErrorInvalidCreds,
		ErrorUnauthorized:   cfg.Messages.ErrorUnauthorized,
	}
}

func GetPostMessages(cfg *config.Config) PostMessages {
	return PostMessages{
		PostCreated:        cfg.Messages.PostCreated,
		ErrorInvalidData:   cfg.Messages.ErrorInvalidData,
		ErrorPostEmpty:     cfg.Messages.ErrorPostEmpty,
		ErrorPostTooLong:   cfg.Messages.ErrorPostTooLong,
		ErrorImageTooLarge: cfg.Messages.ErrorImageTooLarge,
		ErrorInvalidImage:  cfg.Messages.ErrorInvalidImage,
		ErrorServer:        cfg.Messages.ErrorServer,
	}
}

func GetPostConfig(cfg *config.Config) PostConfig {
	return PostConfig{
		MaxImageSize:      int64(cfg.Images.MaxSizeMB) << 20,
		MultipartBodySize: int64(cfg.Limits.MultipartBodySizeMB) << 20,
	}
}

func GetLikeMessages(cfg *config.Config) LikeMessages {
	return LikeMessages{
		ErrorInvalidData:   cfg.Messages.ErrorInvalidData,
		ErrorInvalidPostID: cfg.Messages.ErrorInvalidPostID,
		ErrorServer:        cfg.Messages.ErrorServer,
	}
}

func GetAdminMessages(cfg *config.Config) AdminMessages {
	return AdminMessages{
		UserAdded:          cfg.Messages.UserAdded,
		RoleUpdated:        cfg.Messages.RoleUpdated,
		UserRemoved:        cfg.Messages.UserRemoved,
		ErrorInvalidData:   cfg.Messages.ErrorInvalidData,
		ErrorInvalidUserID: cfg.Messages.ErrorInvalidUserID,
		ErrorInvalidRole:   cfg.Messages.ErrorInvalidRole,
		ErrorRouteNotFound: cfg.Messages.ErrorRouteNotFound,
		ErrorServer:        cfg.Messages.ErrorServer,
	}
}

func NewCookieService(cfg *config.Config) *auth.CookieService {
	return auth.NewCookieService(
		cfg.Auth.CookieName,
		cfg.Auth.CookiePath,
		cfg.Auth.SecureCookies,
		cfg.Auth.JWTExpireHours,
	)
}

func NewImageStorage(cfg *config.Config) *storage.ImageStorage {
	return storage.NewImageStorage(
		cfg.Images.Dir,
		cfg.Images.PublicPrefix,
		cfg.Images.AllowedTypes,
		cfg.Images.MaxSizeMB,
	)
}

func GetAuthMessages(cfg *config.Config) AuthMessages {
	return AuthMessages{
		ErrorUnauthorized: cfg.Messages.ErrorUnauthorized,
		ErrorInvalidToken: cfg.Messages.ErrorInvalidToken,
		AccessDenied:      cfg.Messages.AccessDenied,
		AdminDenied:       cfg.Messages.AdminDenied,
	}
}
