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
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			public_id VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY idx_users_public_id (public_id)
		)`,

		`CREATE TABLE IF NOT EXISTS user_api_keys (
			user_id BIGINT UNSIGNED NOT NULL,
			encrypted_key TEXT NOT NULL,
			key_hint VARCHAR(32) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id),
			CONSTRAINT fk_user_api_keys_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			user_id BIGINT UNSIGNED NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT 'New chat',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			INDEX idx_chat_sessions_user_updated (user_id, updated_at),
			CONSTRAINT fk_chat_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS chat_messages (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			session_id BIGINT UNSIGNED NOT NULL,
			role VARCHAR(16) NOT NULL,
			content MEDIUMTEXT NOT NULL,
			model VARCHAR(128) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'complete',
			error TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			INDEX idx_chat_messages_session_id (session_id, id),
			CONSTRAINT fk_chat_messages_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
		)`,
	}

	for _, statement := range statements {
		if _, err := s.DB.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
