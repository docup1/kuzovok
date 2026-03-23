package post

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, userID int64, content, imageURL string, imageExpiresAt *time.Time) (*Post, error)
	GetByID(ctx context.Context, id int64) (*Post, error)
	GetUserPosts(ctx context.Context, userID, currentUserID int64) ([]Post, error)
	GetFeed(ctx context.Context, currentUserID int64, limit int) ([]Post, error)
	GetExpiredImages(ctx context.Context, now time.Time) ([]ExpiredImage, error)
	ClearImage(ctx context.Context, postID int64) error
}

type ExpiredImage struct {
	PostID   int64
	ImageURL string
}
