package access

import "context"

type Repository interface {
	GetByUserID(ctx context.Context, userID int64) (*AllowedUser, error)
	Create(ctx context.Context, userID int64, role string) error
	UpdateRole(ctx context.Context, userID int64, role string) error
	Delete(ctx context.Context, userID int64) error
	CountAdmins(ctx context.Context) (int, error)
	UserExists(ctx context.Context, userID int64) (bool, error)
}
