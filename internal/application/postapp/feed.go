package post

import (
	"context"

	"kusovok/internal/domain/post"
)

type FeedUseCase struct {
	postService *post.Service
}

func NewFeedUseCase(postService *post.Service) *FeedUseCase {
	return &FeedUseCase{postService: postService}
}

func (uc *FeedUseCase) Execute(ctx context.Context, currentUserID int64, limit int) ([]post.Post, error) {
	return uc.postService.GetFeed(ctx, currentUserID, limit)
}
