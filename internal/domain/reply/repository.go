package reply

import "context"

type Repository interface {
	Create(ctx context.Context, postID, parentPostID int64) error
	GetParentPostID(ctx context.Context, postID int64) (int64, error)
}
