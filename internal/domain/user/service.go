package user

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	repo userRepository
}

type userRepository interface {
	Create(ctx context.Context, username, hashedPassword string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
}

func NewService(repo userRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, username, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, username, string(hashedPassword))
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
