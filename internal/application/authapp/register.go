package appauth

import (
	"context"

	"kusovok/internal/domain/user"
	apperrors "kusovok/pkg/errors"
)

type RegisterUseCase struct {
	userService *user.Service
	jwtService  JWTService
}

type JWTService interface {
	GenerateToken(userID int64, username string) (string, error)
}

func NewRegisterUseCase(userService *user.Service, jwtService JWTService) *RegisterUseCase {
	return &RegisterUseCase{
		userService: userService,
		jwtService:  jwtService,
	}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, username, password string) (string, *user.AuthResponse, error) {
	if username == "" || password == "" {
		return "", nil, apperrors.BadRequest("all fields required")
	}
	if len(password) < 6 {
		return "", nil, apperrors.BadRequest("password must be at least 6 characters")
	}

	u, err := uc.userService.Register(ctx, username, password)
	if err != nil {
		return "", nil, err
	}

	token, err := uc.jwtService.GenerateToken(u.ID, u.Username)
	if err != nil {
		return "", nil, err
	}

	return token, &user.AuthResponse{
		ID:       u.ID,
		Username: u.Username,
	}, nil
}
