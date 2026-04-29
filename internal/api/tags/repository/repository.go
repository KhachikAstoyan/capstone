package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/KhachikAstoyan/capstone/internal/api/tags/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, name string) (*domain.Tag, error) {
	query := `
		INSERT INTO tags (name)
		VALUES ($1)
		RETURNING id, name, created_at
	`

	var tag domain.Tag
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tag.ID,
		&tag.Name,
		&tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Tag, error) {
	query := `
		SELECT id, name, created_at
		FROM tags
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	query := `
		SELECT id, name, created_at
		FROM tags
		WHERE id = $1
	`

	var tag domain.Tag
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tag.ID,
		&tag.Name,
		&tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*domain.Tag, error) {
	query := `
		SELECT id, name, created_at
		FROM tags
		WHERE name = $1
	`

	var tag domain.Tag
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tag.ID,
		&tag.Name,
		&tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *Repository) SetProblemTags(ctx context.Context, problemID uuid.UUID, tagIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	deleteQuery := `DELETE FROM problem_tags WHERE problem_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQuery, problemID); err != nil {
		return err
	}

	if len(tagIDs) > 0 {
		insertQuery := `INSERT INTO problem_tags (problem_id, tag_id) VALUES ($1, $2)`
		for _, tagID := range tagIDs {
			if _, err := tx.ExecContext(ctx, insertQuery, problemID, tagID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) GetProblemTags(ctx context.Context, problemID uuid.UUID) ([]domain.Tag, error) {
	query := `
		SELECT t.id, t.name, t.created_at
		FROM tags t
		INNER JOIN problem_tags pt ON pt.tag_id = t.id
		WHERE pt.problem_id = $1
		ORDER BY t.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
