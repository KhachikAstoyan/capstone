package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/google/uuid"
)

func (r *userRepository) ListAdminUsers(ctx context.Context, query, sortBy string, limit, offset int) ([]domain.AdminUserSummary, int, error) {
	orderCol := "violation_count"
	switch sortBy {
	case "handle":
		orderCol = "u.handle"
	case "submissions":
		orderCol = "submission_count"
	case "created_at":
		orderCol = "u.created_at"
	}

	const baseFrom = `
		FROM users u
		LEFT JOIN submissions s ON s.user_id = u.id AND s.kind = 'submit'
		LEFT JOIN security_events se ON se.submission_id IN (
			SELECT id FROM submissions WHERE user_id = u.id
		)
		WHERE ($1 = '' OR u.handle ILIKE '%' || $1 || '%' OR u.email::text ILIKE '%' || $1 || '%')
		GROUP BY u.id, u.handle, u.email, u.display_name, u.status, u.created_at
	`

	countQ := `SELECT COUNT(*) FROM (SELECT u.id ` + baseFrom + `) sub`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin users: %w", err)
	}

	listQ := `
		SELECT u.id, u.handle, u.email, u.display_name, u.status, u.created_at,
		       COUNT(DISTINCT se.id) AS violation_count,
		       COUNT(DISTINCT s.id) AS submission_count
		` + baseFrom + `
		ORDER BY ` + orderCol + ` DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, listQ, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	var users []domain.AdminUserSummary
	for rows.Next() {
		var u domain.AdminUserSummary
		if err := rows.Scan(
			&u.ID, &u.Handle, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt,
			&u.ViolationCount, &u.SubmissionCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan admin user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *userRepository) GetUserSecurityEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.SecurityEvent, int, error) {
	const countQ = `
		SELECT COUNT(*) FROM security_events se
		WHERE se.submission_id IN (SELECT id FROM submissions WHERE user_id = $1)`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count security events: %w", err)
	}

	const listQ = `
		SELECT se.id, se.submission_id, se.category, se.severity, se.detail_json, se.created_at
		FROM security_events se
		WHERE se.submission_id IN (SELECT id FROM submissions WHERE user_id = $1)
		ORDER BY se.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, listQ, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list security events: %w", err)
	}
	defer rows.Close()

	var events []domain.SecurityEvent
	for rows.Next() {
		var e domain.SecurityEvent
		var detailRaw []byte
		if err := rows.Scan(&e.ID, &e.SubmissionID, &e.Category, &e.Severity, &detailRaw, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan security event: %w", err)
		}
		if len(detailRaw) > 0 {
			e.DetailJSON = json.RawMessage(detailRaw)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return events, total, nil
}
