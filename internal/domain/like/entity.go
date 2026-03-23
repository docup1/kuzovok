package like

type Like struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	PostID    int64  `json:"post_id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type PostLikes struct {
	PostID         int64   `json:"post_id"`
	AuthorUsername string  `json:"author_username"`
	Content        string  `json:"content"`
	ImageURL       *string `json:"image_url,omitempty"`
	CreatedAt      string  `json:"created_at"`
	LikeCount      int     `json:"like_count"`
	LikedUsers     []Like  `json:"liked_users"`
}

type LikeRequest struct {
	PostID int64 `json:"post_id"`
}
