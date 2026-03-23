package appauth

import (
	"context"

	"kusovok/internal/domain/access"
	"kusovok/internal/domain/user"
)

type MeUseCase struct {
	userRepo      UserRepository
	accessService AccessService
}

type UserRepository interface {
	CountPosts(ctx context.Context, userID int64) (int, error)
}

type AccessService interface {
	GetAccessInfo(ctx context.Context, userID int64) (access.AccessInfo, error)
}

func NewMeUseCase(userRepo UserRepository, accessService AccessService) *MeUseCase {
	return &MeUseCase{
		userRepo:      userRepo,
		accessService: accessService,
	}
}

func (uc *MeUseCase) Execute(ctx context.Context, userID int64, username string) (*user.MeResponse, error) {
	postCount, err := uc.userRepo.CountPosts(ctx, userID)
	if err != nil {
		return nil, err
	}

	access, err := uc.accessService.GetAccessInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := &user.MeResponse{
		ID:        userID,
		Username:  username,
		PostCount: postCount,
		IsAllowed: access.IsAllowed,
		Role:      string(access.Role),
	}

	if !access.IsAllowed {
		response.AccessMessage = accessDeniedMessage
	}

	return response, nil
}
