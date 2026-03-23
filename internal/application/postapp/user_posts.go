package post

import (
	"context"

	"kusovok/internal/domain/post"
)

type UserPostsUseCase struct {
	postService *post.Service
}

func NewUserPostsUseCase(postService *post.Service) *UserPostsUseCase {
	return &UserPostsUseCase{postService: postService}
}

func (uc *UserPostsUseCase) Execute(ctx context.Context, userID, currentUserID int64) ([]post.Post, error) {
	return uc.postService.GetUserPosts(ctx, userID, currentUserID)
}
