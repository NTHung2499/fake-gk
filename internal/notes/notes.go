package notes

import (
	"context"
	"database/sql"
	"strings"
)

var allowedColors = map[string]struct{}{
	"yellow": {},
	"green":  {},
	"blue":   {},
	"pink":   {},
	"purple": {},
	"gray":   {},
}

var colorList = []string{"yellow", "green", "blue", "pink", "purple", "gray"}

type Note struct {
	ID         int64
	Title      string
	Body       string
	Color      string
	IsPinned   bool
	IsArchived bool
}

type Repository struct {
	db *sql.DB
}

type Input struct {
	Title string
	Body  string
	Color string
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func Colors() []string {
	return append([]string(nil), colorList...)
}

func (r *Repository) List(ctx context.Context) ([]Note, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, body, color, is_pinned, is_archived
		FROM notes
		ORDER BY is_archived ASC, is_pinned DESC, updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Body, &note.Color, &note.IsPinned, &note.IsArchived); err != nil {
			return nil, err
		}
		result = append(result, note)
	}
	return result, rows.Err()
}

func (r *Repository) Create(ctx context.Context, input Input) error {
	title := normalizeText(input.Title)
	body := normalizeText(input.Body)
	color := normalizeColor(input.Color)
	if title == "" && body == "" {
		return nil
	}

	_, err := r.db.ExecContext(ctx, "INSERT INTO notes (title, body, color) VALUES (?, ?, ?)", title, body, color)
	return err
}

func (r *Repository) Update(ctx context.Context, id string, input Input) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE notes SET title = ?, body = ?, color = ? WHERE id = ?",
		normalizeText(input.Title),
		normalizeText(input.Body),
		normalizeColor(input.Color),
		id,
	)
	return err
}

func (r *Repository) TogglePinned(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE notes SET is_pinned = NOT is_pinned WHERE id = ?", id)
	return err
}

func (r *Repository) ToggleArchived(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE notes SET is_pinned = IF(is_archived = FALSE, FALSE, is_pinned), is_archived = NOT is_archived WHERE id = ?", id)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM notes WHERE id = ?", id)
	return err
}

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeColor(value string) string {
	if _, ok := allowedColors[value]; ok {
		return value
	}
	return "yellow"
}
