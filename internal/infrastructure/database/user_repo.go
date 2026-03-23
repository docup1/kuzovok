package database

import (
	"context"
	"database/sql"
	"kusovok/internal/domain/user"
	"strings"
	"time"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, username, hashedPassword string) (*user.User, error) {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO users (username, password) VALUES (?, ?)",
		username, hashedPassword,
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

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT id, username, password, created_at FROM users WHERE username = ?",
		username,
	)

	var u user.User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Username, &u.Password, &createdAt); err != nil {
		return nil, err
	}

	u.CreatedAt = parseTimestamp(createdAt)
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT id, username, password, created_at FROM users WHERE id = ?",
		id,
	)

	var u user.User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Username, &u.Password, &createdAt); err != nil {
		return nil, err
	}

	u.CreatedAt = parseTimestamp(createdAt)
	return &u, nil
}

func (r *UserRepository) GetSummaryByID(ctx context.Context, id int64) (*user.UserSummary, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.created_at, COUNT(p.id) AS post_count,
		       CASE WHEN au.user_id IS NULL THEN 0 ELSE 1 END AS is_allowed,
		       COALESCE(au.role, '')
		  FROM users u
		  LEFT JOIN allowed_users au ON au.user_id = u.id
		  LEFT JOIN posts p ON p.user_id = u.id
		 WHERE u.id = ?
		 GROUP BY u.id, u.username, u.created_at, au.user_id, au.role`,
		id,
	)

	var summary user.UserSummary
	var createdAt string
	var isAllowed int
	var role string

	if err := row.Scan(&summary.ID, &summary.Username, &createdAt, &summary.PostCount, &isAllowed, &role); err != nil {
		return nil, err
	}

	summary.CreatedAt = parseTimestamp(createdAt).Format(time.RFC3339)
	summary.IsAllowed = isAllowed == 1
	if summary.IsAllowed {
		summary.Role = strings.TrimSpace(role)
	}

	return &summary, nil
}

func (r *UserRepository) GetAllSummaries(ctx context.Context) ([]user.UserSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.created_at, COUNT(p.id) AS post_count,
		       CASE WHEN au.user_id IS NULL THEN 0 ELSE 1 END AS is_allowed,
		       COALESCE(au.role, '')
		  FROM users u
		  LEFT JOIN allowed_users au ON au.user_id = u.id
		  LEFT JOIN posts p ON p.user_id = u.id
		 GROUP BY u.id, u.username, u.created_at, au.user_id, au.role
		 ORDER BY u.created_at DESC, u.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []user.UserSummary
	for rows.Next() {
		var s user.UserSummary
		var createdAt string
		var isAllowed int
		var role string

		if err := rows.Scan(&s.ID, &s.Username, &createdAt, &s.PostCount, &isAllowed, &role); err != nil {
			return nil, err
		}

		s.CreatedAt = parseTimestamp(createdAt).Format(time.RFC3339)
		s.IsAllowed = isAllowed == 1
		if s.IsAllowed {
			s.Role = strings.TrimSpace(role)
		}

		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

func (r *UserRepository) CountPosts(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE user_id = ?",
		userID,
	).Scan(&count)
	return count, err
}

func parseTimestamp(value string) time.Time {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Now()
}
