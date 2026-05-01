package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/ai/domain"
	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateValidation(ctx context.Context, req domain.ValidateCodeRequest, resp domain.ValidateCodeResponse) (*domain.CodeValidation, error) {
	validation := &domain.CodeValidation{
		ID:           uuid.New(),
		SubmissionID: req.SubmissionID,
		UserID:       req.UserID,
		ProblemID:    req.ProblemID,
		Code:         req.Code,
		LanguageID:   req.LanguageID,
		IsAllowed:    resp.IsAllowed,
	}

	if resp.Severity != "" {
		validation.Severity = &resp.Severity
	}
	if resp.Reason != "" {
		validation.Reason = &resp.Reason
	}

	if resp.Details != nil {
		validation.ValidationMeta = resp.Details
	}

	metaJSON, _ := json.Marshal(validation.ValidationMeta)

	query := `
		INSERT INTO ai_code_validations (
			id, submission_id, user_id, problem_id, code, language_id,
			is_allowed, severity, reason, validation_metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, submission_id, user_id, problem_id, code, language_id,
		          is_allowed, severity, reason, validation_metadata, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		validation.ID,
		validation.SubmissionID,
		validation.UserID,
		validation.ProblemID,
		validation.Code,
		validation.LanguageID,
		validation.IsAllowed,
		validation.Severity,
		validation.Reason,
		metaJSON,
	).Scan(
		&validation.ID,
		&validation.SubmissionID,
		&validation.UserID,
		&validation.ProblemID,
		&validation.Code,
		&validation.LanguageID,
		&validation.IsAllowed,
		&validation.Severity,
		&validation.Reason,
		&metaJSON,
		&validation.CreatedAt,
		&validation.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create validation: %w", err)
	}

	if len(metaJSON) > 0 {
		json.Unmarshal(metaJSON, &validation.ValidationMeta)
	}

	return validation, nil
}

func (r *Repository) GetValidationBySubmission(ctx context.Context, submissionID uuid.UUID) (*domain.CodeValidation, error) {
	query := `
		SELECT id, submission_id, user_id, problem_id, code, language_id,
		       is_allowed, severity, reason, validation_metadata, created_at, updated_at
		FROM ai_code_validations
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	validation := &domain.CodeValidation{}
	var metaJSON []byte

	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&validation.ID,
		&validation.SubmissionID,
		&validation.UserID,
		&validation.ProblemID,
		&validation.Code,
		&validation.LanguageID,
		&validation.IsAllowed,
		&validation.Severity,
		&validation.Reason,
		&metaJSON,
		&validation.CreatedAt,
		&validation.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get validation: %w", err)
	}

	if len(metaJSON) > 0 {
		json.Unmarshal(metaJSON, &validation.ValidationMeta)
	}
	return validation, nil
}

func (r *Repository) LogValidationRequest(ctx context.Context, validationID uuid.UUID, requestBody, responseBody map[string]interface{}, errorMsg *string, tokensUsed, responseTime *int) (*domain.ValidationLog, error) {
	log := &domain.ValidationLog{
		ID:            uuid.New(),
		ValidationID:  validationID,
		RequestBody:   requestBody,
		ResponseBody:  responseBody,
		ErrorMessage:  errorMsg,
		TokensUsed:    tokensUsed,
		ResponseTimeMs: responseTime,
	}

	reqJSON, _ := json.Marshal(requestBody)
	respJSON, _ := json.Marshal(responseBody)

	query := `
		INSERT INTO ai_validation_logs (
			id, validation_id, request_body, response_body, error_message,
			tokens_used, response_time_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, validation_id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		log.ID,
		log.ValidationID,
		reqJSON,
		respJSON,
		log.ErrorMessage,
		log.TokensUsed,
		log.ResponseTimeMs,
	).Scan(&log.ID, &log.ValidationID, &log.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("log validation: %w", err)
	}

	return log, nil
}
