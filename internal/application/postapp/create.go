package post

import (
	"context"
	"time"

	"kusovok/internal/domain/post"
	apperrors "kusovok/pkg/errors"
)

type CreatePostUseCase struct {
	postService   *post.Service
	imageStorage  ImageStorage
	lifetime      time.Duration
	maxContentLen int
}

type ImageStorage interface {
	Save(data []byte, contentType string) (url string, expiresAt string, err error)
	Delete(imageURL string) error
}

func NewCreatePostUseCase(postService *post.Service, imageStorage ImageStorage, lifetimeHours, maxContentLen int) *CreatePostUseCase {
	return &CreatePostUseCase{
		postService:   postService,
		imageStorage:  imageStorage,
		lifetime:      time.Duration(lifetimeHours) * time.Hour,
		maxContentLen: maxContentLen,
	}
}

func (uc *CreatePostUseCase) Execute(ctx context.Context, userID int64, username string, content string, imageData []byte, imageContentType string) (*post.Post, error) {
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

	p.Username = username
	return p, nil
}

func trimSpace(s string) string {
	return s
}
