package database

import (
	"context"
	"database/sql"
	"kusovok/internal/domain/access"
	"strings"
)

type AccessRepository struct {
	db *sql.DB
}

func NewAccessRepository(db *sql.DB) *AccessRepository {
	return &AccessRepository{db: db}
}

func (r *AccessRepository) GetByUserID(ctx context.Context, userID int64) (*access.AllowedUser, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT user_id, role, created_at FROM allowed_users WHERE user_id = ?",
		userID,
	)

	var au access.AllowedUser
	var createdAt string
	if err := row.Scan(&au.UserID, &au.Role, &createdAt); err != nil {
		return nil, err
	}

	au.CreatedAt = parseTimestamp(createdAt)
	au.Role = strings.TrimSpace(au.Role)
	return &au, nil
}

func (r *AccessRepository) Create(ctx context.Context, userID int64, role string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO allowed_users (user_id, role) VALUES (?, ?)",
		userID, role,
	)
	return err
}

func (r *AccessRepository) UpdateRole(ctx context.Context, userID int64, role string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE allowed_users SET role = ? WHERE user_id = ?",
		role, userID,
	)
	return err
}

func (r *AccessRepository) Delete(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM allowed_users WHERE user_id = ?",
		userID,
	)
	return err
}

func (r *AccessRepository) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM allowed_users WHERE role = 'admin'",
	).Scan(&count)
	return count, err
}

func (r *AccessRepository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE id = ?",
		userID,
	).Scan(&count)
	return count > 0, err
}
