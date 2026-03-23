package admin

import (
	"context"
	"strings"

	"kusovok/internal/domain/access"
	"kusovok/internal/domain/shared"
	apperrors "kusovok/pkg/errors"
)

type ManageAllowedUsersUseCase struct {
	accessService *access.Service
}

func NewManageAllowedUsersUseCase(accessService *access.Service) *ManageAllowedUsersUseCase {
	return &ManageAllowedUsersUseCase{accessService: accessService}
}

func (uc *ManageAllowedUsersUseCase) AddUser(ctx context.Context, userID int64, role string) (*shared.UserSummary, error) {
	if userID <= 0 {
		return nil, apperrors.BadRequest("invalid user id")
	}

	parsedRole := parseRole(role)
	if parsedRole == "" {
		return nil, apperrors.BadRequest("invalid role")
	}

	summary, err := uc.accessService.AddAllowedUser(ctx, userID, parsedRole)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, apperrors.Conflict("user already allowed")
		}
		return nil, err
	}

	return summary, nil
}

func (uc *ManageAllowedUsersUseCase) UpdateRole(ctx context.Context, userID int64, role string) (*shared.UserSummary, error) {
	if userID <= 0 {
		return nil, apperrors.BadRequest("invalid user id")
	}

	parsedRole := parseRole(role)
	if parsedRole == "" {
		return nil, apperrors.BadRequest("invalid role")
	}

	summary, err := uc.accessService.UpdateRole(ctx, userID, parsedRole)
	if err != nil {
		if access.ErrNotInAllowedList == err {
			return nil, apperrors.NotFound("user not in allowed list")
		}
		if access.ErrLastAdmin == err {
			return nil, apperrors.Conflict("cannot demote last admin")
		}
		return nil, err
	}

	return summary, nil
}

func (uc *ManageAllowedUsersUseCase) RemoveUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return apperrors.BadRequest("invalid user id")
	}

	err := uc.accessService.RemoveAllowedUser(ctx, userID)
	if err != nil {
		if access.ErrNotInAllowedList == err {
			return apperrors.NotFound("user not in allowed list")
		}
		if access.ErrCannotRemoveAdmin == err {
			return apperrors.Conflict("demote admin first")
		}
		return err
	}

	return nil
}

func parseRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "", "user":
		return "user"
	case "admin":
		return "admin"
	default:
		return ""
	}
}
