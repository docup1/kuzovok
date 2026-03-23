package reply

import "context"

type Service struct {
	replyRepo replyRepository
	postRepo  postGetter
}

type replyRepository interface {
	Create(ctx context.Context, postID, parentPostID int64) error
	GetParentPostID(ctx context.Context, postID int64) (int64, error)
}

type postGetter interface {
	Exists(ctx context.Context, postID int64) (bool, error)
}

func NewService(replyRepo replyRepository, postRepo postGetter) *Service {
	return &Service{
		replyRepo: replyRepo,
		postRepo:  postRepo,
	}
}

func (s *Service) CreateReply(ctx context.Context, postID, parentPostID int64) error {
	exists, err := s.postRepo.Exists(ctx, parentPostID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrParentPostNotFound
	}

	return s.replyRepo.Create(ctx, postID, parentPostID)
}

func (s *Service) GetParentPostID(ctx context.Context, postID int64) (int64, error) {
	return s.replyRepo.GetParentPostID(ctx, postID)
}
