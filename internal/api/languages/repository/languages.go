package repository

import (
	"context"
	"database/sql"

	"github.com/KhachikAstoyan/capstone/internal/api/languages/domain"
	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, search string) ([]domain.Language, error) {
	args := []any{}
	where := ""
	if search != "" {
		where = `WHERE key ILIKE $1 OR display_name ILIKE $1`
		args = append(args, "%"+search+"%")
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, key, display_name, is_enabled, created_at, updated_at
		FROM languages
		`+where+`
		ORDER BY key ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := []domain.Language{}
	for rows.Next() {
		var lang domain.Language
		if err := rows.Scan(
			&lang.ID,
			&lang.Key,
			&lang.DisplayName,
			&lang.IsEnabled,
			&lang.CreatedAt,
			&lang.UpdatedAt,
		); err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}
	return languages, rows.Err()
}

func (r *Repository) Upsert(ctx context.Context, key, displayName string, enabled bool) (*domain.Language, error) {
	var lang domain.Language
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO languages (key, display_name, is_enabled)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()
		RETURNING id, key, display_name, is_enabled, created_at, updated_at`,
		key, displayName, enabled,
	).Scan(
		&lang.ID,
		&lang.Key,
		&lang.DisplayName,
		&lang.IsEnabled,
		&lang.CreatedAt,
		&lang.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *Repository) ListForProblem(ctx context.Context, problemID uuid.UUID) ([]domain.Language, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.key, l.display_name, l.is_enabled, l.created_at, l.updated_at
		FROM languages l
		INNER JOIN problem_languages pl ON pl.language_id = l.id
		WHERE pl.problem_id = $1 AND l.is_enabled = TRUE
		ORDER BY l.key ASC`, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := []domain.Language{}
	for rows.Next() {
		var lang domain.Language
		if err := rows.Scan(
			&lang.ID,
			&lang.Key,
			&lang.DisplayName,
			&lang.IsEnabled,
			&lang.CreatedAt,
			&lang.UpdatedAt,
		); err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}
	return languages, rows.Err()
}

func (r *Repository) SetForProblem(ctx context.Context, problemID uuid.UUID, languageIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM problem_languages WHERE problem_id = $1`, problemID); err != nil {
		return err
	}

	for _, languageID := range languageIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO problem_languages (problem_id, language_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, problemID, languageID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
