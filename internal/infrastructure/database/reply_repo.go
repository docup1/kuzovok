package database

import (
	"context"
	"database/sql"
)

type ReplyRepository struct {
	db *sql.DB
}

func NewReplyRepository(db *sql.DB) *ReplyRepository {
	return &ReplyRepository{db: db}
}

func (r *ReplyRepository) Create(ctx context.Context, postID, parentPostID int64) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO post_replies (post_id, parent_post_id) VALUES (?, ?)",
		postID, parentPostID,
	)
	return err
}

func (r *ReplyRepository) GetParentPostID(ctx context.Context, postID int64) (int64, error) {
	var parentPostID int64
	err := r.db.QueryRowContext(ctx,
		"SELECT parent_post_id FROM post_replies WHERE post_id = ?",
		postID,
	).Scan(&parentPostID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return parentPostID, err
}
