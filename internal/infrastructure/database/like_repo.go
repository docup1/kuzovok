package database

import (
	"context"
	"database/sql"
	"kusovok/internal/domain/like"
	"strings"
	"time"
)

type LikeRepository struct {
	db *sql.DB
}

func NewLikeRepository(db *sql.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) Exists(ctx context.Context, userID, postID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM likes WHERE user_id = ? AND post_id = ?",
		userID, postID,
	).Scan(&count)
	return count > 0, err
}

func (r *LikeRepository) Create(ctx context.Context, userID, postID int64) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO likes (user_id, post_id) VALUES (?, ?)",
		userID, postID,
	)
	return err
}

func (r *LikeRepository) Delete(ctx context.Context, userID, postID int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM likes WHERE user_id = ? AND post_id = ?",
		userID, postID,
	)
	return err
}

func (r *LikeRepository) CountByPostID(ctx context.Context, postID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM likes WHERE post_id = ?",
		postID,
	).Scan(&count)
	return count, err
}

func (r *LikeRepository) GetAllWithDetails(ctx context.Context) ([]like.PostLikes, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, author.username, p.content, p.image_url, p.created_at,
		       liker.id, liker.username, l.created_at
		  FROM likes l
		  JOIN posts p ON p.id = l.post_id
		  JOIN users author ON author.id = p.user_id
		  JOIN users liker ON liker.id = l.user_id
		 ORDER BY p.created_at DESC, p.id DESC, l.created_at DESC, l.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []like.PostLikes{}
	postIndex := map[int64]int{}

	for rows.Next() {
		var (
			postID         int64
			authorUsername string
			content        string
			imageURL       sql.NullString
			createdAt      string
			likerID        int64
			likerUsername  string
			likedAt        string
		)
		if err := rows.Scan(&postID, &authorUsername, &content, &imageURL, &createdAt, &likerID, &likerUsername, &likedAt); err != nil {
			continue
		}

		idx, ok := postIndex[postID]
		if !ok {
			postIndex[postID] = len(posts)
			posts = append(posts, like.PostLikes{
				PostID:         postID,
				AuthorUsername: authorUsername,
				Content:        content,
				ImageURL:       nullableStringPointer(imageURL),
				CreatedAt:      parseTimestamp(createdAt).Format(time.RFC3339),
			})
			idx = len(posts) - 1
		}

		p := &posts[idx]
		p.LikeCount++
		p.LikedUsers = append(p.LikedUsers, like.Like{
			UserID:    likerID,
			Username:  likerUsername,
			CreatedAt: strings.TrimSpace(likedAt),
		})
	}

	return posts, rows.Err()
}
