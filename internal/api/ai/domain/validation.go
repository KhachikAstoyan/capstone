package domain

import (
	"time"

	"github.com/google/uuid"
)

type ValidationSeverity string

const (
	SeverityInfo  ValidationSeverity = "info"
	SeverityWarn  ValidationSeverity = "warn"
	SeverityHigh  ValidationSeverity = "high"
	SeverityBlock ValidationSeverity = "block"
)

type CodeValidation struct {
	ID             uuid.UUID
	SubmissionID   uuid.UUID
	UserID         uuid.UUID
	ProblemID      uuid.UUID
	Code           string
	LanguageID     uuid.UUID
	IsAllowed      bool
	Severity       *ValidationSeverity
	Reason         *string
	ValidationMeta map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ValidateCodeRequest struct {
	SubmissionID uuid.UUID
	UserID       uuid.UUID
	ProblemID    uuid.UUID
	Code         string
	LanguageID   uuid.UUID
	LanguageKey  string
}

type ValidateCodeResponse struct {
	IsAllowed bool
	Severity  ValidationSeverity
	Reason    string
	Details   map[string]interface{}
}

type ValidationLog struct {
	ID             uuid.UUID
	ValidationID   uuid.UUID
	RequestBody    map[string]interface{}
	ResponseBody   map[string]interface{}
	ErrorMessage   *string
	TokensUsed     *int
	ResponseTimeMs *int
	CreatedAt      time.Time
}
