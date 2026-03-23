package database

import (
	"context"
	"database/sql"
	"kusovok/internal/domain/post"
	"time"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(ctx context.Context, userID int64, content, imageURL string, imageExpiresAt *time.Time) (*post.Post, error) {
	var iea interface{}
	if imageExpiresAt != nil {
		iea = imageExpiresAt.Format(time.RFC3339)
	}

	result, err := r.db.ExecContext(ctx,
		"INSERT INTO posts (user_id, content, created_at, image_url, image_expires_at) VALUES (?, ?, ?, ?, ?)",
		userID, content, time.Now().UTC().Format(time.RFC3339), imageURL, iea,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *PostRepository) GetByID(ctx context.Context, id int64) (*post.Post, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.user_id, u.username, p.content, p.created_at,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes,
		       0 AS liked, p.image_url, p.image_expires_at
		  FROM posts p
		  JOIN users u ON p.user_id = u.id
		 WHERE p.id = ?`,
		id,
	)

	var p post.Post
	var createdAt string
	var likes int
	var liked int
	var imageURL sql.NullString
	var imageExpiresAt sql.NullString

	if err := row.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &createdAt, &likes, &liked, &imageURL, &imageExpiresAt); err != nil {
		return nil, err
	}

	p.CreatedAt = parseTimestamp(createdAt)
	p.Likes = likes
	p.Liked = liked == 1
	p.ImageURL = nullableStringPointer(imageURL)
	p.ImageExpiresAt = nullableTimePointer(imageExpiresAt)

	return &p, nil
}

func (r *PostRepository) GetUserPosts(ctx context.Context, userID, currentUserID int64) ([]post.Post, error) {
	return r.queryPosts(ctx, `
		SELECT p.id, p.user_id, u.username, p.content, p.created_at,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes,
		       EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked,
		       p.image_url, p.image_expires_at
		  FROM posts p
		  JOIN users u ON p.user_id = u.id
		 WHERE p.user_id = ?
		 ORDER BY p.created_at DESC`,
		currentUserID, userID,
	)
}

func (r *PostRepository) GetFeed(ctx context.Context, currentUserID int64, limit int) ([]post.Post, error) {
	return r.queryPosts(ctx, `
		SELECT p.id, p.user_id, u.username, p.content, p.created_at,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes,
		       EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked,
		       p.image_url, p.image_expires_at
		  FROM posts p
		  JOIN users u ON p.user_id = u.id
		 ORDER BY p.created_at DESC
		 LIMIT ?`,
		currentUserID, limit,
	)
}

func (r *PostRepository) queryPosts(ctx context.Context, query string, args ...interface{}) ([]post.Post, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []post.Post
	for rows.Next() {
		var p post.Post
		var createdAt string
		var likes int
		var liked int
		var imageURL sql.NullString
		var imageExpiresAt sql.NullString

		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &createdAt, &likes, &liked, &imageURL, &imageExpiresAt); err != nil {
			continue
		}

		p.CreatedAt = parseTimestamp(createdAt)
		p.Likes = likes
		p.Liked = liked == 1
		p.ImageURL = nullableStringPointer(imageURL)
		p.ImageExpiresAt = nullableTimePointer(imageExpiresAt)

		posts = append(posts, p)
	}

	return posts, rows.Err()
}

func (r *PostRepository) GetExpiredImages(ctx context.Context, now time.Time) ([]post.ExpiredImage, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, image_url FROM posts WHERE image_url IS NOT NULL AND image_url != '' AND image_expires_at IS NOT NULL AND image_expires_at <= ?",
		now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expired []post.ExpiredImage
	for rows.Next() {
		var ei post.ExpiredImage
		if err := rows.Scan(&ei.PostID, &ei.ImageURL); err != nil {
			continue
		}
		expired = append(expired, ei)
	}

	return expired, rows.Err()
}

func (r *PostRepository) ClearImage(ctx context.Context, postID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE posts SET image_url = NULL, image_expires_at = NULL WHERE id = ?",
		postID,
	)
	return err
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}

func nullableTimePointer(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed := parseTimestamp(value.String)
	return &parsed
}
