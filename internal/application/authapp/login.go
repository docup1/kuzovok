package appauth

import (
	"context"

	"kusovok/internal/domain/user"
	apperrors "kusovok/pkg/errors"
)

type LoginUseCase struct {
	userService *user.Service
	jwtService  JWTService
}

func NewLoginUseCase(userService *user.Service, jwtService JWTService) *LoginUseCase {
	return &LoginUseCase{
		userService: userService,
		jwtService:  jwtService,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, username, password string) (string, *user.AuthResponse, error) {
	if username == "" || password == "" {
		return "", nil, apperrors.BadRequest("all fields required")
	}

	u, err := uc.userService.Authenticate(ctx, username, password)
	if err != nil {
		return "", nil, apperrors.Unauthorized("Неверный логин или пароль")
	}

	token, err := uc.jwtService.GenerateToken(u.ID, u.Username)
	if err != nil {
		return "", nil, apperrors.Internal("failed to generate token")
	}

	return token, &user.AuthResponse{
		ID:       u.ID,
		Username: u.Username,
	}, nil
}
