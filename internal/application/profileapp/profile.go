package profileapp

import (
	"context"

	"kusovok/internal/domain/user"
	apperrors "kusovok/pkg/errors"
)

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID int64) (*user.Profile, error)
	CreateProfile(ctx context.Context, userID int64) error
	UpdateProfile(ctx context.Context, userID int64, profile *user.Profile) error
	GetProfileWithStats(ctx context.Context, userID int64) (*user.ProfileWithStats, error)
	GetProfileWithStatsByUsername(ctx context.Context, username string) (*user.ProfileWithStats, error)
}

type GetProfileUseCase struct {
	repo ProfileRepository
}

func NewGetProfileUseCase(repo ProfileRepository) *GetProfileUseCase {
	return &GetProfileUseCase{repo: repo}
}

func (uc *GetProfileUseCase) Execute(ctx context.Context, userID int64) (*user.ProfileWithStats, error) {
	profile, err := uc.repo.GetProfileWithStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile.UserID == 0 {
		if err := uc.repo.CreateProfile(ctx, userID); err != nil {
			return nil, err
		}
		profile, err = uc.repo.GetProfileWithStats(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	return profile, nil
}

type GetProfileByUsernameUseCase struct {
	repo ProfileRepository
}

func NewGetProfileByUsernameUseCase(repo ProfileRepository) *GetProfileByUsernameUseCase {
	return &GetProfileByUsernameUseCase{repo: repo}
}

func (uc *GetProfileByUsernameUseCase) Execute(ctx context.Context, username string) (*user.ProfileWithStats, error) {
	return uc.repo.GetProfileWithStatsByUsername(ctx, username)
}

type UpdateProfileUseCase struct {
	repo ProfileRepository
}

func NewUpdateProfileUseCase(repo ProfileRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, userID int64, req *user.UpdateProfileRequest) (*user.Profile, error) {
	if req.Avatar == "" {
		req.Avatar = "🐠"
	}

	existing, err := uc.repo.GetProfile(ctx, userID)
	if err != nil && !apperrors.IsNoRows(err) {
		return nil, err
	}

	if apperrors.IsNoRows(err) {
		if err := uc.repo.CreateProfile(ctx, userID); err != nil {
			return nil, err
		}
		existing = &user.Profile{UserID: userID, Avatar: "🐠"}
	}

	existing.Avatar = req.Avatar
	existing.Name = req.Name
	existing.Bio = req.Bio

	if err := uc.repo.UpdateProfile(ctx, userID, existing); err != nil {
		return nil, err
	}

	return existing, nil
}
