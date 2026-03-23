package admin

import (
	"context"

	"kusovok/internal/domain/like"
)

type GetLikesUseCase struct {
	likeRepo like.Repository
}

func NewGetLikesUseCase(likeRepo like.Repository) *GetLikesUseCase {
	return &GetLikesUseCase{likeRepo: likeRepo}
}

func (uc *GetLikesUseCase) Execute(ctx context.Context) ([]like.PostLikes, error) {
	return uc.likeRepo.GetAllWithDetails(ctx)
}
