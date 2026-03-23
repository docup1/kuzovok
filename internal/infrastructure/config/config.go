package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Auth     AuthConfig     `json:"auth"`
	Images   ImagesConfig   `json:"images"`
	Messages MessagesConfig `json:"messages"`
	Limits   LimitsConfig   `json:"limits"`
	Cleanup  CleanupConfig  `json:"cleanup"`
}

type ServerConfig struct {
	Addr    string `json:"addr"`
	BaseURL string `json:"base_url"`
}

type DatabaseConfig struct {
	Path         string `json:"path"`
	MaxOpenConns int    `json:"max_open_conns"`
	BackupDir    string `json:"backup_dir"`
}

type AuthConfig struct {
	JWTSecret      string `json:"jwt_secret"`
	JWTExpireHours int    `json:"jwt_expire_hours"`
	CookieName     string `json:"cookie_name"`
	CookiePath     string `json:"cookie_path"`
	SecureCookies  bool   `json:"secure_cookies"`
}

type ImagesConfig struct {
	Dir           string   `json:"dir"`
	PublicPrefix  string   `json:"public_prefix"`
	AllowedTypes  []string `json:"allowed_types"`
	MaxSizeMB     int      `json:"max_size_mb"`
	LifetimeHours int      `json:"lifetime_hours"`
}

type MessagesConfig struct {
	AccessDenied        string `json:"access_denied"`
	AdminDenied         string `json:"admin_denied"`
	RegisterSuccess     string `json:"register_success"`
	LoginSuccess        string `json:"login_success"`
	LogoutSuccess       string `json:"logout_success"`
	PostCreated         string `json:"post_created"`
	RoleUpdated         string `json:"role_updated"`
	UserAdded           string `json:"user_added"`
	UserRemoved         string `json:"user_removed"`
	ErrorInvalidData    string `json:"error_invalid_data"`
	ErrorRequiredFields string `json:"error_required_fields"`
	ErrorPasswordShort  string `json:"error_password_short"`
	ErrorUserExists     string `json:"error_user_exists"`
	ErrorInvalidCreds   string `json:"error_invalid_credentials"`
	ErrorUnauthorized   string `json:"error_unauthorized"`
	ErrorInvalidToken   string `json:"error_invalid_token"`
	ErrorMethodNotAllow string `json:"error_method_not_allowed"`
	ErrorPostEmpty      string `json:"error_post_empty"`
	ErrorPostTooLong    string `json:"error_post_too_long"`
	ErrorImageTooLarge  string `json:"error_image_too_large"`
	ErrorInvalidImage   string `json:"error_invalid_image"`
	ErrorPostNotFound   string `json:"error_post_not_found"`
	ErrorInvalidPostID  string `json:"error_invalid_post_id"`
	ErrorInvalidUserID  string `json:"error_invalid_user_id"`
	ErrorUserNotFound   string `json:"error_user_not_found"`
	ErrorInvalidRole    string `json:"error_invalid_role"`
	ErrorAlreadyAllowed string `json:"error_already_allowed"`
	ErrorLastAdmin      string `json:"error_last_admin"`
	ErrorDemoteAdmin    string `json:"error_demote_admin_first"`
	ErrorNotInAllowed   string `json:"error_not_found_in_allowed"`
	ErrorRouteNotFound  string `json:"error_route_not_found"`
	ErrorServer         string `json:"error_server"`
}

type LimitsConfig struct {
	PostContentMaxLength int `json:"post_content_max_length"`
	MultipartBodySizeMB  int `json:"multipart_body_size_mb"`
}

type CleanupConfig struct {
	IntervalMinutes int `json:"interval_minutes"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	setDefaults(&cfg)
	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Server.BaseURL == "" {
		cfg.Server.BaseURL = "http://localhost:8080"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./kusovok.db"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 1
	}
	if cfg.Database.BackupDir == "" {
		cfg.Database.BackupDir = "./backups"
	}
	if cfg.Auth.JWTExpireHours == 0 {
		cfg.Auth.JWTExpireHours = 24
	}
	if cfg.Auth.CookieName == "" {
		cfg.Auth.CookieName = "token"
	}
	if cfg.Auth.CookiePath == "" {
		cfg.Auth.CookiePath = "/"
	}
	if cfg.Images.Dir == "" {
		cfg.Images.Dir = "./img"
	}
	if cfg.Images.PublicPrefix == "" {
		cfg.Images.PublicPrefix = "/img/"
	}
	if cfg.Images.MaxSizeMB == 0 {
		cfg.Images.MaxSizeMB = 10
	}
	if cfg.Images.LifetimeHours == 0 {
		cfg.Images.LifetimeHours = 24
	}
	if cfg.Limits.PostContentMaxLength == 0 {
		cfg.Limits.PostContentMaxLength = 1000
	}
	if cfg.Limits.MultipartBodySizeMB == 0 {
		cfg.Limits.MultipartBodySizeMB = 11
	}
	if cfg.Cleanup.IntervalMinutes == 0 {
		cfg.Cleanup.IntervalMinutes = 1
	}
}
