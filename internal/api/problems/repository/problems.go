package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/driver"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrProblemNotFound     = errors.New("problem not found")
	ErrProblemSlugConflict = errors.New("problem with this slug already exists")
)

type ListFilters struct {
	Visibility *domain.ProblemVisibility
	Difficulty *domain.ProblemDifficulty
	Tags       []string
	Search     string
}

type Repository interface {
	Create(ctx context.Context, req domain.CreateProblemRequest, slug string, createdByUserID uuid.UUID) (*domain.Problem, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Problem, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Problem, error)
	List(ctx context.Context, filters ListFilters, limit, offset int) ([]*domain.Problem, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, filters ListFilters) (int, error)
	GetSolvedProblemIDs(ctx context.Context, userID uuid.UUID, problemIDs []uuid.UUID) ([]uuid.UUID, error)
	GetAcceptanceRates(ctx context.Context, problemIDs []uuid.UUID) (map[uuid.UUID]float64, error)
	GetProblemTags(ctx context.Context, problemIDs []uuid.UUID) (map[uuid.UUID][]string, error)

	ListTestCases(ctx context.Context, problemID uuid.UUID) ([]*domain.TestCase, error)
	CreateTestCase(ctx context.Context, problemID uuid.UUID, req domain.CreateTestCaseRequest) (*domain.TestCase, error)
	UpdateTestCase(ctx context.Context, id uuid.UUID, req domain.UpdateTestCaseRequest) (*domain.TestCase, error)
	DeleteTestCase(ctx context.Context, id uuid.UUID) error
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
			slug, title, summary, statement_markdown, time_limit_ms, memory_limit_mb,
			tests_ref, visibility, difficulty, created_by_user_id, function_spec
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, slug, title, summary, statement_markdown, time_limit_ms, memory_limit_mb,
		          tests_ref, tests_hash, visibility, difficulty, created_by_user_id, created_at, updated_at, function_spec
	`

	var fsParam interface{}
	if req.FunctionSpec != nil {
		fsParam, _ = json.Marshal(req.FunctionSpec)
	}

	problem := &domain.Problem{}
	var fsRaw []byte
	err := r.db.QueryRowContext(
		ctx, query,
		slug, req.Title, req.Summary, req.StatementMarkdown, req.TimeLimitMs, req.MemoryLimitMb,
		req.TestsRef, req.Visibility, req.Difficulty, createdByUserID, fsParam,
	).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.Summary, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.Difficulty, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
		&fsRaw,
	)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "problems_slug_key"` {
			return nil, ErrProblemSlugConflict
		}
		return nil, fmt.Errorf("failed to create problem: %w", err)
	}

	if len(fsRaw) > 0 && string(fsRaw) != "null" {
		var fs driver.FunctionSpec
		if json.Unmarshal(fsRaw, &fs) == nil {
			problem.FunctionSpec = &fs
		}
	}

	return problem, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Problem, error) {
	query := `
		SELECT id, slug, title, summary, statement_markdown, time_limit_ms, memory_limit_mb,
		       tests_ref, tests_hash, visibility, difficulty, created_by_user_id, created_at, updated_at,
		       function_spec
		FROM problems
		WHERE id = $1
	`

	problem := &domain.Problem{}
	var fsRaw []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.Summary, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.Difficulty, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
		&fsRaw,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}

	if len(fsRaw) > 0 && string(fsRaw) != "null" {
		var fs driver.FunctionSpec
		if json.Unmarshal(fsRaw, &fs) == nil {
			problem.FunctionSpec = &fs
		}
	}

	return problem, nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*domain.Problem, error) {
	query := `
		SELECT id, slug, title, summary, statement_markdown, time_limit_ms, memory_limit_mb,
		       tests_ref, tests_hash, visibility, difficulty, created_by_user_id, created_at, updated_at,
		       function_spec
		FROM problems
		WHERE slug = $1
	`

	problem := &domain.Problem{}
	var fsRaw []byte
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.Summary, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.Difficulty, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
		&fsRaw,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}

	if len(fsRaw) > 0 && string(fsRaw) != "null" {
		var fs driver.FunctionSpec
		if json.Unmarshal(fsRaw, &fs) == nil {
			problem.FunctionSpec = &fs
		}
	}

	return problem, nil
}

func (r *repository) List(ctx context.Context, filters ListFilters, limit, offset int) ([]*domain.Problem, error) {
	query := `
		SELECT DISTINCT p.id, p.slug, p.title, p.summary, p.statement_markdown, p.time_limit_ms, p.memory_limit_mb,
		       p.tests_ref, p.tests_hash, p.visibility, p.difficulty, p.created_by_user_id, p.created_at, p.updated_at,
		       p.function_spec
		FROM problems p
	`

	args := []interface{}{}
	argNum := 1
	whereClauses := []string{}

	if len(filters.Tags) > 0 {
		query += `
		INNER JOIN problem_tags pt ON pt.problem_id = p.id
		INNER JOIN tags t ON t.id = pt.tag_id
		`
		placeholders := []string{}
		for _, tag := range filters.Tags {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
			args = append(args, tag)
			argNum++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("t.name IN (%s)", strings.Join(placeholders, ", ")))
	}

	if filters.Visibility != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.visibility = $%d", argNum))
		args = append(args, *filters.Visibility)
		argNum++
	}

	if filters.Difficulty != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.difficulty = $%d", argNum))
		args = append(args, *filters.Difficulty)
		argNum++
	}

	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(p.title ILIKE $%d OR p.slug ILIKE $%d OR p.summary ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+filters.Search+"%")
		argNum++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			query += " AND " + whereClauses[i]
		}
	}

	query += " ORDER BY p.created_at DESC"

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
		var fsRaw []byte
		err := rows.Scan(
			&problem.ID, &problem.Slug, &problem.Title, &problem.Summary, &problem.StatementMarkdown,
			&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
			&problem.Visibility, &problem.Difficulty, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
			&fsRaw,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan problem: %w", err)
		}
		if len(fsRaw) > 0 && string(fsRaw) != "null" {
			var fs driver.FunctionSpec
			if json.Unmarshal(fsRaw, &fs) == nil {
				problem.FunctionSpec = &fs
			}
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

	if req.Summary != nil {
		query += fmt.Sprintf(", summary = $%d", argNum)
		args = append(args, *req.Summary)
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

	if req.Difficulty != nil {
		query += fmt.Sprintf(", difficulty = $%d", argNum)
		args = append(args, *req.Difficulty)
		argNum++
	}

	if req.FunctionSpec != nil {
		fsBytes, _ := json.Marshal(req.FunctionSpec)
		query += fmt.Sprintf(", function_spec = $%d", argNum)
		args = append(args, fsBytes)
		argNum++
	}

	query += fmt.Sprintf(` WHERE id = $%d
		RETURNING id, slug, title, summary, statement_markdown, time_limit_ms, memory_limit_mb,
		          tests_ref, tests_hash, visibility, difficulty, created_by_user_id, created_at, updated_at,
		          function_spec`, argNum)
	args = append(args, id)

	problem := &domain.Problem{}
	var fsRaw []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&problem.ID, &problem.Slug, &problem.Title, &problem.Summary, &problem.StatementMarkdown,
		&problem.TimeLimitMs, &problem.MemoryLimitMb, &problem.TestsRef, &problem.TestsHash,
		&problem.Visibility, &problem.Difficulty, &problem.CreatedByUserID, &problem.CreatedAt, &problem.UpdatedAt,
		&fsRaw,
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

	if len(fsRaw) > 0 && string(fsRaw) != "null" {
		var fs driver.FunctionSpec
		if json.Unmarshal(fsRaw, &fs) == nil {
			problem.FunctionSpec = &fs
		}
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

func (r *repository) Count(ctx context.Context, filters ListFilters) (int, error) {
	query := `SELECT COUNT(DISTINCT p.id) FROM problems p`
	args := []interface{}{}
	argNum := 1
	whereClauses := []string{}

	if len(filters.Tags) > 0 {
		query += `
		INNER JOIN problem_tags pt ON pt.problem_id = p.id
		INNER JOIN tags t ON t.id = pt.tag_id
		`
		placeholders := []string{}
		for _, tag := range filters.Tags {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
			args = append(args, tag)
			argNum++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("t.name IN (%s)", strings.Join(placeholders, ", ")))
	}

	if filters.Visibility != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.visibility = $%d", argNum))
		args = append(args, *filters.Visibility)
		argNum++
	}

	if filters.Difficulty != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.difficulty = $%d", argNum))
		args = append(args, *filters.Difficulty)
		argNum++
	}

	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(p.title ILIKE $%d OR p.slug ILIKE $%d OR p.summary ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+filters.Search+"%")
		argNum++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			query += " AND " + whereClauses[i]
		}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count problems: %w", err)
	}

	return count, nil
}

func (r *repository) GetSolvedProblemIDs(ctx context.Context, userID uuid.UUID, problemIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(problemIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	// Convert UUID slice to string slice for pq.Array
	problemIDStrs := make([]string, len(problemIDs))
	for i, id := range problemIDs {
		problemIDStrs[i] = id.String()
	}

	query := `
		SELECT DISTINCT s.problem_id
		FROM submissions s
		WHERE s.user_id = $1
		  AND s.status = 'accepted'
		  AND s.kind = 'submit'
		  AND s.problem_id = ANY($2)
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pq.Array(problemIDStrs))
	if err != nil {
		return nil, fmt.Errorf("failed to get solved problem IDs: %w", err)
	}
	defer rows.Close()

	var solvedIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan solved problem ID: %w", err)
		}
		solvedIDs = append(solvedIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating solved problem IDs: %w", err)
	}

	return solvedIDs, nil
}

func (r *repository) GetAcceptanceRates(ctx context.Context, problemIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	if len(problemIDs) == 0 {
		return map[uuid.UUID]float64{}, nil
	}

	// Convert UUID slice to string slice for pq.Array
	problemIDStrs := make([]string, len(problemIDs))
	for i, id := range problemIDs {
		problemIDStrs[i] = id.String()
	}

	query := `
		SELECT
			s.problem_id,
			COUNT(*)                                       AS total_submissions,
			COUNT(*) FILTER (WHERE s.status = 'accepted') AS accepted_submissions
		FROM submissions s
		WHERE s.problem_id = ANY($1)
		  AND s.kind = 'submit'
		GROUP BY s.problem_id
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(problemIDStrs))
	if err != nil {
		return nil, fmt.Errorf("failed to get acceptance rates: %w", err)
	}
	defer rows.Close()

	rates := make(map[uuid.UUID]float64)
	for rows.Next() {
		var problemID uuid.UUID
		var total, accepted int
		if err := rows.Scan(&problemID, &total, &accepted); err != nil {
			return nil, fmt.Errorf("failed to scan acceptance rate: %w", err)
		}
		if total > 0 {
			rates[problemID] = (float64(accepted) / float64(total)) * 100
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating acceptance rates: %w", err)
	}

	return rates, nil
}

func (r *repository) GetProblemTags(ctx context.Context, problemIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(problemIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}

	// Convert UUID slice to string slice for pq.Array
	problemIDStrs := make([]string, len(problemIDs))
	for i, id := range problemIDs {
		problemIDStrs[i] = id.String()
	}

	query := `
		SELECT pt.problem_id, t.name
		FROM problem_tags pt
		INNER JOIN tags t ON t.id = pt.tag_id
		WHERE pt.problem_id = ANY($1)
		ORDER BY pt.problem_id, t.name
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(problemIDStrs))
	if err != nil {
		return nil, fmt.Errorf("failed to get problem tags: %w", err)
	}
	defer rows.Close()

	tagsMap := make(map[uuid.UUID][]string)
	for rows.Next() {
		var problemID uuid.UUID
		var tagName string
		if err := rows.Scan(&problemID, &tagName); err != nil {
			return nil, fmt.Errorf("failed to scan problem tag: %w", err)
		}
		tagsMap[problemID] = append(tagsMap[problemID], tagName)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating problem tags: %w", err)
	}

	return tagsMap, nil
}
