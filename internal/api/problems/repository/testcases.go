package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/google/uuid"
)

var ErrTestCaseNotFound = errors.New("test case not found")

func (r *repository) getTestCaseByID(ctx context.Context, id uuid.UUID) (*domain.TestCase, error) {
	const q = `
		SELECT id, problem_id, external_id, input_data, expected_data,
		       order_index, is_active, is_hidden, created_at
		FROM problem_test_cases WHERE id = $1 AND is_active = TRUE`
	tc := &domain.TestCase{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&tc.ID, &tc.ProblemID, &tc.ExternalID,
		&tc.InputData, &tc.ExpectedData,
		&tc.OrderIndex, &tc.IsActive, &tc.IsHidden, &tc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

func (r *repository) ListTestCases(ctx context.Context, problemID uuid.UUID) ([]*domain.TestCase, error) {
	const q = `
		SELECT id, problem_id, external_id, input_data, expected_data,
		       order_index, is_active, is_hidden, created_at
		FROM problem_test_cases
		WHERE problem_id = $1 AND is_active = TRUE
		ORDER BY order_index ASC`

	rows, err := r.db.QueryContext(ctx, q, problemID)
	if err != nil {
		return nil, fmt.Errorf("list test cases: %w", err)
	}
	defer rows.Close()

	var tcs []*domain.TestCase
	for rows.Next() {
		tc := &domain.TestCase{}
		if err := rows.Scan(
			&tc.ID, &tc.ProblemID, &tc.ExternalID,
			&tc.InputData, &tc.ExpectedData,
			&tc.OrderIndex, &tc.IsActive, &tc.IsHidden, &tc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan test case: %w", err)
		}
		tcs = append(tcs, tc)
	}
	return tcs, rows.Err()
}

func (r *repository) CreateTestCase(ctx context.Context, problemID uuid.UUID, req domain.CreateTestCaseRequest) (*domain.TestCase, error) {
	const q = `
		INSERT INTO problem_test_cases
		    (problem_id, external_id, input_data, expected_data, order_index, is_hidden)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, problem_id, external_id, input_data, expected_data,
		          order_index, is_active, is_hidden, created_at`

	tc := &domain.TestCase{}
	err := r.db.QueryRowContext(ctx, q,
		problemID,
		uuid.New().String(),
		req.InputData,
		req.ExpectedData,
		req.OrderIndex,
		req.IsHidden,
	).Scan(
		&tc.ID, &tc.ProblemID, &tc.ExternalID,
		&tc.InputData, &tc.ExpectedData,
		&tc.OrderIndex, &tc.IsActive, &tc.IsHidden, &tc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create test case: %w", err)
	}
	return tc, nil
}

func (r *repository) UpdateTestCase(ctx context.Context, id uuid.UUID, req domain.UpdateTestCaseRequest) (*domain.TestCase, error) {
	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	if len(req.InputData) > 0 {
		setClauses = append(setClauses, fmt.Sprintf("input_data = $%d", argNum))
		args = append(args, req.InputData)
		argNum++
	}
	if len(req.ExpectedData) > 0 {
		setClauses = append(setClauses, fmt.Sprintf("expected_data = $%d", argNum))
		args = append(args, req.ExpectedData)
		argNum++
	}
	if req.OrderIndex != nil {
		setClauses = append(setClauses, fmt.Sprintf("order_index = $%d", argNum))
		args = append(args, *req.OrderIndex)
		argNum++
	}
	if req.IsHidden != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_hidden = $%d", argNum))
		args = append(args, *req.IsHidden)
		argNum++
	}
	if len(setClauses) == 0 {
		return r.getTestCaseByID(ctx, id)
	}

	query := "UPDATE problem_test_cases SET "
	for i, c := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += c
	}
	query += fmt.Sprintf(` WHERE id = $%d AND is_active = TRUE
		RETURNING id, problem_id, external_id, input_data, expected_data,
		          order_index, is_active, is_hidden, created_at`, argNum)
	args = append(args, id)

	tc := &domain.TestCase{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&tc.ID, &tc.ProblemID, &tc.ExternalID,
		&tc.InputData, &tc.ExpectedData,
		&tc.OrderIndex, &tc.IsActive, &tc.IsHidden, &tc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update test case: %w", err)
	}
	return tc, nil
}

func (r *repository) DeleteTestCase(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE problem_test_cases SET is_active = FALSE WHERE id = $1 AND is_active = TRUE`, id)
	if err != nil {
		return fmt.Errorf("delete test case: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTestCaseNotFound
	}
	return nil
}
