package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/google/uuid"
)

var (
	ErrProblemNotFound     = errors.New("problem not found")
	ErrProblemSlugConflict = errors.New("problem with this slug already exists")
)

type Repository interface {
	Create(ctx context.Context, req domain.CreateProblemRequest, slug string, createdByUserID uuid.UUID) (*domain.Problem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Problem, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Problem, error)
	List(ctx context.Context, visibility *domain.ProblemVisibility, limit, offset int) ([]*domain.Problem, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, visibility *domain.ProblemVisibility) (int, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, req domain.CreateProblemRequest, slug string, createdByUserID uuid.UUID) (*domain.Problem, error) {
	query := `
		INSERT INTO problems (
			slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
			tests_ref, visibility, created_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
		          tests_ref, tests_hash, visibility, created_by_user_id, created_at, updated_at
	`

	problem := &domain.Problem{}
	err := r.db.QueryRowContext(
		ctx, query,
		slug, req.Title, req.StatementMarkdown, req.TimeLimitMs, req.MemoryLimitMb,
		req.TestsRef, req.Visibility, createdByUserID,
	).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
	)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "problems_slug_key"` {
			return nil, ErrProblemSlugConflict
		}
		return nil, fmt.Errorf("failed to create problem: %w", err)
	}

	return problem, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Problem, error) {
	query := `
		SELECT id, slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
		       tests_ref, tests_hash, visibility, created_by_user_id, created_at, updated_at
		FROM problems
		WHERE id = $1
	`

	problem := &domain.Problem{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}

	return problem, nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*domain.Problem, error) {
	query := `
		SELECT id, slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
		       tests_ref, tests_hash, visibility, created_by_user_id, created_at, updated_at
		FROM problems
		WHERE slug = $1
	`

	problem := &domain.Problem{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}

	return problem, nil
}

func (r *repository) List(ctx context.Context, visibility *domain.ProblemVisibility, limit, offset int) ([]*domain.Problem, error) {
	query := `
		SELECT id, slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
		       tests_ref, tests_hash, visibility, created_by_user_id, created_at, updated_at
		FROM problems
	`

	args := []interface{}{}
	argNum := 1

	if visibility != nil {
		query += fmt.Sprintf(" WHERE visibility = $%d", argNum)
		args = append(args, *visibility)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
		argNum++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list problems: %w", err)
	}
	defer rows.Close()

	problems := []*domain.Problem{}
	for rows.Next() {
		problem := &domain.Problem{}
		err := rows.Scan(
			&problem.ID, &problem.Slug, &problem.Title, &problem.StatementMarkdown,
			&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
			&problem.Visibility, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan problem: %w", err)
		}
		problems = append(problems, problem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating problems: %w", err)
	}

	return problems, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error) {
	query := `UPDATE problems SET updated_at = NOW()`
	args := []interface{}{}
	argNum := 1

	if req.Slug != nil {
		query += fmt.Sprintf(", slug = $%d", argNum)
		args = append(args, *req.Slug)
		argNum++
	}

	if req.Title != nil {
		query += fmt.Sprintf(", title = $%d", argNum)
		args = append(args, *req.Title)
		argNum++
	}

	if req.StatementMarkdown != nil {
		query += fmt.Sprintf(", statement_markdown = $%d", argNum)
		args = append(args, *req.StatementMarkdown)
		argNum++
	}

	if req.TimeLimitMs != nil {
		query += fmt.Sprintf(", time_limit_ms = $%d", argNum)
		args = append(args, *req.TimeLimitMs)
		argNum++
	}

	if req.MemoryLimitMb != nil {
		query += fmt.Sprintf(", memory_limit_mb = $%d", argNum)
		args = append(args, *req.MemoryLimitMb)
		argNum++
	}

	if req.TestsRef != nil {
		query += fmt.Sprintf(", tests_ref = $%d", argNum)
		args = append(args, *req.TestsRef)
		argNum++
	}

	if req.Visibility != nil {
		query += fmt.Sprintf(", visibility = $%d", argNum)
		args = append(args, *req.Visibility)
		argNum++
	}

	query += fmt.Sprintf(` WHERE id = $%d
		RETURNING id, slug, title, statement_markdown, time_limit_ms, memory_limit_mb,
		          tests_ref, tests_hash, visibility, created_by_user_id, created_at, updated_at`, argNum)
	args = append(args, id)

	problem := &domain.Problem{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProblemNotFound
		}
		if err.Error() == `pq: duplicate key value violates unique constraint "problems_slug_key"` {
			return nil, ErrProblemSlugConflict
		}
		return nil, fmt.Errorf("failed to update problem: %w", err)
	}

	return problem, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM problems WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete problem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProblemNotFound
	}

	return nil
}

func (r *repository) Count(ctx context.Context, visibility *domain.ProblemVisibility) (int, error) {
	query := `SELECT COUNT(*) FROM problems`
	args := []interface{}{}

	if visibility != nil {
		query += " WHERE visibility = $1"
		args = append(args, *visibility)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count problems: %w", err)
	}

	return count, nil
}
