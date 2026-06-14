package chat

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"

	StatusComplete = "complete"
	StatusError    = "error"
)

var ErrNotFound = errors.New("chat resource not found")

type User struct {
	ID       int64  `json:"id"`
	PublicID string `json:"publicId"`
}

type APIKey struct {
	UserID       int64
	EncryptedKey string
	KeyHint      string
}

type Session struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Message struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"sessionId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func NewPublicID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func (r *Repository) FindUserByPublicID(ctx context.Context, publicID string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, "SELECT id, public_id FROM users WHERE public_id = ?", publicID).Scan(&user.ID, &user.PublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (r *Repository) CreateUser(ctx context.Context, publicID string) (User, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO users (public_id) VALUES (?)", publicID)
	if err != nil {
		return User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, PublicID: publicID}, nil
}

func (r *Repository) UpsertAPIKey(ctx context.Context, userID int64, encryptedKey, hint string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_api_keys (user_id, encrypted_key, key_hint)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE encrypted_key = VALUES(encrypted_key), key_hint = VALUES(key_hint)
	`, userID, encryptedKey, hint)
	return err
}

func (r *Repository) GetAPIKey(ctx context.Context, userID int64) (APIKey, error) {
	var key APIKey
	err := r.db.QueryRowContext(ctx, "SELECT user_id, encrypted_key, key_hint FROM user_api_keys WHERE user_id = ?", userID).
		Scan(&key.UserID, &key.EncryptedKey, &key.KeyHint)
	if errors.Is(err, sql.ErrNoRows) {
		return APIKey{}, ErrNotFound
	}
	return key, err
}

func (r *Repository) DeleteAPIKey(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM user_api_keys WHERE user_id = ?", userID)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, userID int64) ([]Session, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, created_at, updated_at
		FROM chat_sessions
		WHERE user_id = ?
		ORDER BY updated_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.Title, &session.CreatedAt, &session.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *Repository) CreateSession(ctx context.Context, userID int64, title string) (Session, error) {
	title = normalizeTitle(title)
	result, err := r.db.ExecContext(ctx, "INSERT INTO chat_sessions (user_id, title) VALUES (?, ?)", userID, title)
	if err != nil {
		return Session{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Session{}, err
	}
	return r.GetSession(ctx, userID, id)
}

func (r *Repository) GetSession(ctx context.Context, userID, sessionID int64) (Session, error) {
	var session Session
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, created_at, updated_at
		FROM chat_sessions
		WHERE id = ? AND user_id = ?
	`, sessionID, userID).Scan(&session.ID, &session.Title, &session.CreatedAt, &session.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return session, err
}

func (r *Repository) RenameSession(ctx context.Context, userID, sessionID int64, title string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE chat_sessions SET title = ? WHERE id = ? AND user_id = ?", normalizeTitle(title), sessionID, userID)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (r *Repository) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM chat_sessions WHERE id = ? AND user_id = ?", sessionID, userID)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (r *Repository) TouchSession(ctx context.Context, userID, sessionID int64) error {
	result, err := r.db.ExecContext(ctx, "UPDATE chat_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?", sessionID, userID)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (r *Repository) ListMessages(ctx context.Context, userID, sessionID int64, limit int) ([]Message, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 1000
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, role, content, model, status, COALESCE(error, ''), created_at
		FROM (
			SELECT id, session_id, role, content, model, status, error, created_at
			FROM chat_messages
			WHERE session_id = ?
			ORDER BY id DESC
			LIMIT ?
		) recent
		ORDER BY id ASC
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.Model, &message.Status, &message.Error, &message.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (r *Repository) AddMessage(ctx context.Context, userID, sessionID int64, role, content, model, status, messageError string) (Message, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return Message{}, err
	}

	role = normalizeRole(role)
	status = normalizeStatus(status)
	content = strings.TrimSpace(content)
	if role == RoleUser && content == "" {
		return Message{}, errors.New("message content is required")
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (session_id, role, content, model, status, error)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''))
	`, sessionID, role, content, strings.TrimSpace(model), status, strings.TrimSpace(messageError))
	if err != nil {
		return Message{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Message{}, err
	}
	if err := r.TouchSession(ctx, userID, sessionID); err != nil {
		return Message{}, err
	}
	return r.GetMessage(ctx, userID, sessionID, id)
}

func (r *Repository) GetMessage(ctx context.Context, userID, sessionID, messageID int64) (Message, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return Message{}, err
	}

	var message Message
	err := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, role, content, model, status, COALESCE(error, ''), created_at
		FROM chat_messages
		WHERE id = ? AND session_id = ?
	`, messageID, sessionID).Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.Model, &message.Status, &message.Error, &message.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	return message, err
}

func normalizeTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "New chat"
	}
	runes := []rune(title)
	if len(runes) > 255 {
		return string(runes[:255])
	}
	return title
}

func normalizeRole(role string) string {
	if role == RoleAssistant {
		return RoleAssistant
	}
	return RoleUser
}

func normalizeStatus(status string) string {
	if status == StatusError {
		return StatusError
	}
	return StatusComplete
}

func requireAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
