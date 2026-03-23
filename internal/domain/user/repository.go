package user

import "context"

type Repository interface {
	Create(ctx context.Context, username, hashedPassword string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetSummaryByID(ctx context.Context, id int64) (*UserSummary, error)
	GetAllSummaries(ctx context.Context) ([]UserSummary, error)
	CountPosts(ctx context.Context, userID int64) (int, error)
}
