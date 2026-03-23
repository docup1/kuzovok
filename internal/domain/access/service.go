package access

import (
	"context"
	"errors"

	"kusovok/internal/domain/shared"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrNotInAllowedList  = errors.New("user not in allowed list")
	ErrLastAdmin         = errors.New("cannot demote last admin")
	ErrCannotRemoveAdmin = errors.New("cannot remove admin, demote first")
)

type Service struct {
	accessRepo accessRepository
	userRepo   userRepository
}

type accessRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*AllowedUser, error)
	Create(ctx context.Context, userID int64, role string) error
	UpdateRole(ctx context.Context, userID int64, role string) error
	Delete(ctx context.Context, userID int64) error
	CountAdmins(ctx context.Context) (int, error)
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type userRepository interface {
	GetSummaryByID(ctx context.Context, id int64) (*shared.UserSummary, error)
}

func NewService(accessRepo accessRepository, userRepo userRepository) *Service {
	return &Service{
		accessRepo: accessRepo,
		userRepo:   userRepo,
	}
}

func (s *Service) GetAccessInfo(ctx context.Context, userID int64) (AccessInfo, error) {
	au, err := s.accessRepo.GetByUserID(ctx, userID)
	if err != nil {
		return AccessInfo{IsAllowed: false}, nil
	}
	return AccessInfo{
		IsAllowed: true,
		Role:      shared.ParseRole(au.Role),
	}, nil
}

func (s *Service) AddAllowedUser(ctx context.Context, userID int64, role string) (*shared.UserSummary, error) {
	exists, err := s.accessRepo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	if err := s.accessRepo.Create(ctx, userID, role); err != nil {
		return nil, err
	}

	return s.userRepo.GetSummaryByID(ctx, userID)
}

func (s *Service) UpdateRole(ctx context.Context, userID int64, role string) (*shared.UserSummary, error) {
	summary, err := s.userRepo.GetSummaryByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if !summary.IsAllowed {
		return nil, ErrNotInAllowedList
	}

	currentRole := shared.ParseRole(summary.Role)
	newRole := shared.ParseRole(role)

	if currentRole == newRole {
		return summary, nil
	}

	if currentRole == shared.RoleAdmin && newRole == shared.RoleUser {
		count, err := s.accessRepo.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, ErrLastAdmin
		}
	}

	if err := s.accessRepo.UpdateRole(ctx, userID, role); err != nil {
		return nil, err
	}

	return s.userRepo.GetSummaryByID(ctx, userID)
}

func (s *Service) RemoveAllowedUser(ctx context.Context, userID int64) error {
	summary, err := s.userRepo.GetSummaryByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if !summary.IsAllowed {
		return ErrNotInAllowedList
	}
	if summary.Role == string(shared.RoleAdmin) {
		return ErrCannotRemoveAdmin
	}

	return s.accessRepo.Delete(ctx, userID)
}
