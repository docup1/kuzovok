package post

import (
	"context"
	"time"

	"kusovok/internal/domain/post"
	"kusovok/internal/domain/reply"
	apperrors "kusovok/pkg/errors"
)

type CreatePostUseCase struct {
	postService   *post.Service
	replyService  *reply.Service
	imageStorage  ImageStorage
	lifetime      time.Duration
	maxContentLen int
}

type ImageStorage interface {
	Save(data []byte, contentType string) (url string, expiresAt string, err error)
	Delete(imageURL string) error
}

func NewCreatePostUseCase(postService *post.Service, replyService *reply.Service, imageStorage ImageStorage, lifetimeHours, maxContentLen int) *CreatePostUseCase {
	return &CreatePostUseCase{
		postService:   postService,
		replyService:  replyService,
		imageStorage:  imageStorage,
		lifetime:      time.Duration(lifetimeHours) * time.Hour,
		maxContentLen: maxContentLen,
	}
}

func (uc *CreatePostUseCase) Execute(ctx context.Context, userID int64, username string, content string, imageData []byte, imageContentType string, parentPostID *int64) (*post.Post, error) {
	content = trimSpace(content)

	if content == "" && len(imageData) == 0 {
		return nil, apperrors.BadRequest("post must have content or image")
	}

	if len(content) > uc.maxContentLen {
		return nil, apperrors.BadRequest("post too long")
	}

	createdAt := time.Now().UTC()
	var imageURL string
	var imageExpiresAt *time.Time

	if len(imageData) > 0 {
		url, _, err := uc.imageStorage.Save(imageData, imageContentType)
		if err != nil {
			return nil, apperrors.BadRequest("failed to save image")
		}
		imageURL = url
		expiresAt := createdAt.Add(uc.lifetime)
		imageExpiresAt = &expiresAt
	}

	p, err := uc.postService.Create(ctx, userID, content, imageURL, imageExpiresAt)
	if err != nil {
		if imageURL != "" {
			_ = uc.imageStorage.Delete(imageURL)
		}
		return nil, err
	}

	if parentPostID != nil && *parentPostID > 0 {
		if err := uc.replyService.CreateReply(ctx, p.ID, *parentPostID); err != nil {
			if imageURL != "" {
				_ = uc.imageStorage.Delete(imageURL)
			}
			return nil, err
		}
		reloaded, err := uc.postService.GetByID(ctx, p.ID)
		if err == nil {
			p = reloaded
		}
	}

	p.Username = username
	return p, nil
}

func trimSpace(s string) string {
	return s
}
