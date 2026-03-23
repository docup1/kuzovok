package like

import "context"

type Repository interface {
	Exists(ctx context.Context, userID, postID int64) (bool, error)
	Create(ctx context.Context, userID, postID int64) error
	Delete(ctx context.Context, userID, postID int64) error
	CountByPostID(ctx context.Context, postID int64) (int, error)
	GetAllWithDetails(ctx context.Context) ([]PostLikes, error)
}
