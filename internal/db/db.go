package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nthung2499/fake-gk/internal/config"
)

type Store struct {
	DB *sql.DB
}

func Open(cfg config.Config) (*Store, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=false",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(cfg.MySQL.ConnectionLimit)
	conn.SetMaxIdleConns(cfg.MySQL.ConnectionLimit)
	conn.SetConnMaxLifetime(30 * time.Minute)

	return &Store{DB: conn}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) Ping() error {
	return s.DB.Ping()
}

func (s *Store) Migrate() error {
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			title VARCHAR(255) NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			color VARCHAR(32) NOT NULL DEFAULT 'yellow',
			is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
			is_archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			INDEX idx_notes_pinned_updated (is_pinned, updated_at),
			INDEX idx_notes_archived_updated (is_archived, updated_at)
		)
	`)
	return err
}
