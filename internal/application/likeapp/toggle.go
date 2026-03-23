package likeapp

import (
	"context"

	"kusovok/internal/domain/like"
	apppost "kusovok/internal/domain/post"
	apperrors "kusovok/pkg/errors"
)

type ToggleLikeUseCase struct {
	likeService *like.Service
	postService *apppost.Service
}

func NewToggleLikeUseCase(likeService *like.Service, postService *apppost.Service) *ToggleLikeUseCase {
	return &ToggleLikeUseCase{
		likeService: likeService,
		postService: postService,
	}
}

type LikeInfo struct {
	Likes int  `json:"likes"`
	Liked bool `json:"liked"`
}

func (uc *ToggleLikeUseCase) Execute(ctx context.Context, userID, postID int64) (*LikeInfo, error) {
	if postID <= 0 {
		return nil, apperrors.BadRequest("invalid post id")
	}

	exists, err := uc.postService.PostExists(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperrors.NotFound("post not found")
	}

	liked, count, err := uc.likeService.Toggle(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	return &LikeInfo{
		Likes: count,
		Liked: liked,
	}, nil
}
