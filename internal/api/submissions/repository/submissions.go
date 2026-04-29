package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/submissions/domain"
	"github.com/google/uuid"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrLanguageNotFound   = errors.New("language not found or not enabled")
	ErrLanguageNotAllowed = errors.New("language not allowed for this problem")
)

type Repository interface {
	Create(ctx context.Context, userID, problemID, languageID uuid.UUID, sourceText string, kind domain.SubmissionKind) (*domain.Submission, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
	List(ctx context.Context, filters domain.ListFilters, limit, offset int) ([]*domain.Submission, int, error)
	UpdateCPJobID(ctx context.Context, id uuid.UUID, cpJobID uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SubmissionStatus) error
	SaveResult(ctx context.Context, result domain.SubmissionResult, status domain.SubmissionStatus) error
	GetResult(ctx context.Context, submissionID uuid.UUID) (*domain.SubmissionResult, error)
	ResolveLanguage(ctx context.Context, languageKey string) (id uuid.UUID, key string, err error)
	IsLanguageAllowed(ctx context.Context, problemID, languageID uuid.UUID) (bool, error)
	GetTestCasesForProblem(ctx context.Context, problemID uuid.UUID) ([]*domain.ProblemTestCase, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, userID, problemID, languageID uuid.UUID, sourceText string, kind domain.SubmissionKind) (*domain.Submission, error) {
	if kind == "" {
		kind = domain.KindSubmit
	}
	const q = `
		INSERT INTO submissions (user_id, problem_id, language_id, source_text, status, kind)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, user_id, problem_id, language_id, source_text, status, kind, cp_job_id, created_at`

	sub := &domain.Submission{}
	var cpJobID *uuid.UUID
	err := r.db.QueryRowContext(ctx, q, userID, problemID, languageID, sourceText, kind).Scan(
		&sub.ID, &sub.UserID, &sub.ProblemID, &sub.LanguageID,
		&sub.SourceText, &sub.Status, &sub.Kind, &cpJobID, &sub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	sub.CPJobID = cpJobID

	var key string
	err = r.db.QueryRowContext(ctx, `SELECT key FROM languages WHERE id = $1`, languageID).Scan(&key)
	if err != nil {
		return nil, err
	}
	sub.LanguageKey = key
	return sub, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	const q = `
		SELECT s.id, s.user_id, s.problem_id, s.language_id, l.key,
		       s.source_text, s.status, s.kind, s.cp_job_id, s.created_at
		FROM submissions s
		JOIN languages l ON l.id = s.language_id
		WHERE s.id = $1`

	sub := &domain.Submission{}
	var cpJobID *uuid.UUID
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&sub.ID, &sub.UserID, &sub.ProblemID, &sub.LanguageID, &sub.LanguageKey,
		&sub.SourceText, &sub.Status, &sub.Kind, &cpJobID, &sub.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	sub.CPJobID = cpJobID
	return sub, nil
}

func (r *repository) List(ctx context.Context, filters domain.ListFilters, limit, offset int) ([]*domain.Submission, int, error) {
	args := []any{filters.UserID}
	where := `s.user_id = $1 AND s.kind = 'submit'`
	if filters.ProblemID != nil {
		args = append(args, *filters.ProblemID)
		where += ` AND s.problem_id = $2`
	}

	countQ := `SELECT COUNT(*) FROM submissions s WHERE ` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, limit, offset)
	listQ := `
		SELECT s.id, s.user_id, s.problem_id, s.language_id, l.key,
		       s.source_text, s.status, s.kind, s.cp_job_id, s.created_at
		FROM submissions s
		JOIN languages l ON l.id = s.language_id
		WHERE ` + where + `
		ORDER BY s.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", limitIdx) + ` OFFSET $` + fmt.Sprintf("%d", offsetIdx)

	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subs []*domain.Submission
	for rows.Next() {
		sub := &domain.Submission{}
		var cpJobID *uuid.UUID
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.ProblemID, &sub.LanguageID, &sub.LanguageKey,
			&sub.SourceText, &sub.Status, &sub.Kind, &cpJobID, &sub.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		sub.CPJobID = cpJobID
		subs = append(subs, sub)
	}
	return subs, total, rows.Err()
}

func (r *repository) UpdateCPJobID(ctx context.Context, id uuid.UUID, cpJobID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE submissions SET cp_job_id = $2 WHERE id = $1`, id, cpJobID)
	return err
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SubmissionStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE submissions SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *repository) SaveResult(ctx context.Context, result domain.SubmissionResult, status domain.SubmissionStatus) error {
	tcJSON, err := json.Marshal(result.TestcaseResults)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO submission_results
		    (submission_id, overall_verdict, total_time_ms, max_memory_kb,
		     wall_time_ms, compiler_output, testcase_results)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (submission_id) DO NOTHING`,
		result.SubmissionID, result.OverallVerdict, result.TotalTimeMs,
		result.MaxMemoryKb, result.WallTimeMs, result.CompilerOutput, tcJSON,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE submissions SET status = $2 WHERE id = $1`,
		result.SubmissionID, status,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *repository) GetResult(ctx context.Context, submissionID uuid.UUID) (*domain.SubmissionResult, error) {
	const q = `
		SELECT submission_id, overall_verdict, total_time_ms, max_memory_kb,
		       wall_time_ms, compiler_output, testcase_results, created_at
		FROM submission_results
		WHERE submission_id = $1`

	res := &domain.SubmissionResult{}
	var tcRaw []byte
	err := r.db.QueryRowContext(ctx, q, submissionID).Scan(
		&res.SubmissionID, &res.OverallVerdict, &res.TotalTimeMs,
		&res.MaxMemoryKb, &res.WallTimeMs, &res.CompilerOutput,
		&tcRaw, &res.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tcRaw, &res.TestcaseResults); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *repository) ResolveLanguage(ctx context.Context, languageKey string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var key string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, key FROM languages WHERE key = $1 AND is_enabled = TRUE`,
		languageKey,
	).Scan(&id, &key)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, "", ErrLanguageNotFound
	}
	return id, key, err
}

func (r *repository) IsLanguageAllowed(ctx context.Context, problemID, languageID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM problem_languages
		    WHERE problem_id = $1 AND language_id = $2
		)`, problemID, languageID,
	).Scan(&ok)
	return ok, err
}

func (r *repository) GetTestCasesForProblem(ctx context.Context, problemID uuid.UUID) ([]*domain.ProblemTestCase, error) {
	const q = `
		SELECT id, problem_id, external_id, input_data, expected_data, order_index, is_active, is_hidden
		FROM problem_test_cases
		WHERE problem_id = $1 AND is_active = TRUE
		ORDER BY order_index ASC`

	rows, err := r.db.QueryContext(ctx, q, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tcs []*domain.ProblemTestCase
	for rows.Next() {
		tc := &domain.ProblemTestCase{}
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.ExternalID, &tc.InputData, &tc.ExpectedData, &tc.OrderIndex, &tc.IsActive, &tc.IsHidden); err != nil {
			return nil, err
		}
		tcs = append(tcs, tc)
	}
	return tcs, rows.Err()
}

