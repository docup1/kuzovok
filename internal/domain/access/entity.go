package access

import (
	"time"

	"kusovok/internal/domain/shared"
)

type AllowedUser struct {
	UserID    int64     `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type AddAllowedUserRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

type AccessInfo = shared.AccessInfo
