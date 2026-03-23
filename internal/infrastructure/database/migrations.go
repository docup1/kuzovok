package database

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) Run() error {
	if err := m.createTables(); err != nil {
		return err
	}
	if err := m.addPostImageColumns(); err != nil {
		return err
	}
	if err := m.addAllowedUsersRoleColumn(); err != nil {
		return err
	}
	if err := m.createPostRepliesTable(); err != nil {
		return err
	}
	if err := m.createIndexes(); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) Backup(dbPath, backupDir string) error {
	info, err := os.Stat(dbPath)
	if err != nil || info.Size() == 0 {
		return nil
	}

	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	backupPath := fmt.Sprintf("%s/kusovok_%s.db", backupDir, timestamp)

	src, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	fmt.Printf("Бэкап базы создан: %s\n", backupPath)
	return nil
}

func (m *Migrator) createTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			UNIQUE(user_id, post_id)
		)`,
		`CREATE TABLE IF NOT EXISTS allowed_users (
			user_id INTEGER PRIMARY KEY,
			role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user', 'admin')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}
	for _, stmt := range statements {
		if _, err := m.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) addPostImageColumns() error {
	columns, err := m.tableColumns("posts")
	if err != nil {
		return err
	}

	if !columns["image_url"] {
		if _, err := m.db.Exec("ALTER TABLE posts ADD COLUMN image_url TEXT"); err != nil {
			return err
		}
	}
	if !columns["image_expires_at"] {
		if _, err := m.db.Exec("ALTER TABLE posts ADD COLUMN image_expires_at DATETIME"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) addAllowedUsersRoleColumn() error {
	columns, err := m.tableColumns("allowed_users")
	if err != nil {
		return err
	}

	if !columns["role"] {
		if _, err := m.db.Exec("ALTER TABLE allowed_users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'"); err != nil {
			return err
		}
	}

	_, err = m.db.Exec("UPDATE allowed_users SET role = 'user' WHERE role IS NULL OR role = ''")
	return err
}

func (m *Migrator) createPostRepliesTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS post_replies (
			post_id INTEGER PRIMARY KEY,
			parent_post_id INTEGER NOT NULL,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY (parent_post_id) REFERENCES posts(id) ON DELETE CASCADE,
			UNIQUE(post_id)
		)
	`)
	return err
}

func (m *Migrator) createIndexes() error {
	_, err := m.db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_image_expires_at ON posts(image_expires_at)")
	return err
}

func (m *Migrator) tableColumns(table string) (map[string]bool, error) {
	rows, err := m.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

func IsUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func IsForeignKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func IsNoRowsError(err error) bool {
	return err == sql.ErrNoRows
}

func NewConstraintError(message string) error {
	return fmt.Errorf("%s", message)
}
