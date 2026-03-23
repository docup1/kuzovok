package post

import (
	"context"
	"time"

	"kusovok/internal/domain/access"
)

type Service struct {
	postRepo   postRepository
	accessRepo accessRepository
}

type postRepository interface {
	Create(ctx context.Context, userID int64, content, imageURL string, imageExpiresAt *time.Time) (*Post, error)
	GetByID(ctx context.Context, id int64) (*Post, error)
	GetUserPosts(ctx context.Context, userID, currentUserID int64) ([]Post, error)
	GetFeed(ctx context.Context, currentUserID int64, limit int) ([]Post, error)
}

type accessRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*access.AllowedUser, error)
}

func NewService(postRepo postRepository, accessRepo accessRepository) *Service {
	return &Service{
		postRepo:   postRepo,
		accessRepo: accessRepo,
	}
}

func (s *Service) Create(ctx context.Context, userID int64, content, imageURL string, imageExpiresAt *time.Time) (*Post, error) {
	return s.postRepo.Create(ctx, userID, content, imageURL, imageExpiresAt)
}

func (s *Service) GetUserPosts(ctx context.Context, userID, currentUserID int64) ([]Post, error) {
	return s.postRepo.GetUserPosts(ctx, userID, currentUserID)
}

func (s *Service) GetFeed(ctx context.Context, currentUserID int64, limit int) ([]Post, error) {
	return s.postRepo.GetFeed(ctx, currentUserID, limit)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Post, error) {
	return s.postRepo.GetByID(ctx, id)
}

func (s *Service) PostExists(ctx context.Context, postID int64) (bool, error) {
	_, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return false, nil
	}
	return true, nil
}
