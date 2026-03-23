package like

import "context"

type Service struct {
	repo likeRepository
}

type likeRepository interface {
	Exists(ctx context.Context, userID, postID int64) (bool, error)
	Create(ctx context.Context, userID, postID int64) error
	Delete(ctx context.Context, userID, postID int64) error
	CountByPostID(ctx context.Context, postID int64) (int, error)
}

func NewService(repo likeRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Toggle(ctx context.Context, userID, postID int64) (liked bool, likes int, err error) {
	exists, err := s.repo.Exists(ctx, userID, postID)
	if err != nil {
		return false, 0, err
	}

	if exists {
		if err := s.repo.Delete(ctx, userID, postID); err != nil {
			return false, 0, err
		}
		liked = false
	} else {
		if err := s.repo.Create(ctx, userID, postID); err != nil {
			return false, 0, err
		}
		liked = true
	}

	count, err := s.repo.CountByPostID(ctx, postID)
	if err != nil {
		return false, 0, err
	}

	return liked, count, nil
}
