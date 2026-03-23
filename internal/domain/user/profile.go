package user

import "time"

type Profile struct {
	UserID    int64     `json:"user_id"`
	Avatar    string    `json:"avatar"`
	Name      string    `json:"name,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProfileWithStats struct {
	Profile
	Username  string `json:"username"`
	PostCount int    `json:"post_count"`
	LikeCount int    `json:"like_count"`
	CreatedAt string `json:"created_at"`
}

type UpdateProfileRequest struct {
	Avatar string `json:"avatar,omitempty"`
	Name   string `json:"name,omitempty"`
	Bio    string `json:"bio,omitempty"`
}
