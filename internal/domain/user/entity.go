package user

import (
	"time"

	"kusovok/internal/domain/shared"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSummary = shared.UserSummary

type MeResponse struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	PostCount     int    `json:"post_count"`
	IsAllowed     bool   `json:"is_allowed"`
	Role          string `json:"role"`
	AccessMessage string `json:"access_message,omitempty"`
}

type AccessInfo struct {
	IsAllowed bool
	Role      shared.Role
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}
