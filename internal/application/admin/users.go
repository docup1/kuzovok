package admin

import (
	"context"

	"kusovok/internal/domain/shared"
	"kusovok/internal/domain/user"
)

type GetUsersUseCase struct {
	userRepo user.Repository
}

func NewGetUsersUseCase(userRepo user.Repository) *GetUsersUseCase {
	return &GetUsersUseCase{userRepo: userRepo}
}

func (uc *GetUsersUseCase) Execute(ctx context.Context) ([]*shared.UserSummary, error) {
	summaries, err := uc.userRepo.GetAllSummaries(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*shared.UserSummary, len(summaries))
	for i, s := range summaries {
		result[i] = &shared.UserSummary{
			ID:        s.ID,
			Username:  s.Username,
			PostCount: s.PostCount,
			IsAllowed: s.IsAllowed,
			Role:      s.Role,
			CreatedAt: s.CreatedAt,
		}
	}
	return result, nil
}
