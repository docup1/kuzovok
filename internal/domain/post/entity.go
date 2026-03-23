package post

import (
	"time"

	"kusovok/internal/domain/shared"
)

type Post struct {
	ID             int64                  `json:"id"`
	UserID         int64                  `json:"user_id"`
	Username       string                 `json:"username"`
	Avatar         string                 `json:"avatar"`
	Name           string                 `json:"name"`
	Content        string                 `json:"content"`
	CreatedAt      time.Time              `json:"created_at"`
	Likes          int                    `json:"likes"`
	Liked          bool                   `json:"liked"`
	ImageURL       *string                `json:"image_url,omitempty"`
	ImageExpiresAt *time.Time             `json:"image_expires_at,omitempty"`
	ParentPostID   *int64                 `json:"parent_post_id,omitempty"`
	ParentPost     *shared.ParentPostInfo `json:"parent_post,omitempty"`
}

type CreatePostRequest struct {
	Content string `json:"content"`
	Image   *ImageUpload
}

type ImageUpload struct {
	Bytes       []byte
	ContentType string
	Extension   string
}

type StoredImage struct {
	URL       string
	ExpiresAt time.Time
}

type LikeInfo struct {
	Likes int  `json:"likes"`
	Liked bool `json:"liked"`
}
