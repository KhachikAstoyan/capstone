package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SubmissionStatus string

const (
	StatusPending             SubmissionStatus = "pending"
	StatusQueued              SubmissionStatus = "queued"
	StatusRunning             SubmissionStatus = "running"
	StatusAccepted            SubmissionStatus = "accepted"
	StatusWrongAnswer         SubmissionStatus = "wrong_answer"
	StatusTimeLimitExceeded   SubmissionStatus = "time_limit_exceeded"
	StatusMemoryLimitExceeded SubmissionStatus = "memory_limit_exceeded"
	StatusRuntimeError        SubmissionStatus = "runtime_error"
	StatusCompilationError    SubmissionStatus = "compilation_error"
	StatusInternalError       SubmissionStatus = "internal_error"
	StatusBlocked             SubmissionStatus = "blocked"
)

type SubmissionKind string

const (
	KindRun    SubmissionKind = "run"
	KindSubmit SubmissionKind = "submit"
)

func (s SubmissionStatus) IsTerminal() bool {
	switch s {
	case StatusAccepted, StatusWrongAnswer, StatusTimeLimitExceeded,
		StatusMemoryLimitExceeded, StatusRuntimeError,
		StatusCompilationError, StatusInternalError, StatusBlocked:
		return true
	}
	return false
}

type Submission struct {
	ID          uuid.UUID         `json:"id"`
	UserID      uuid.UUID         `json:"user_id"`
	ProblemID   uuid.UUID         `json:"problem_id"`
	LanguageID  uuid.UUID         `json:"language_id"`
	LanguageKey string            `json:"language_key"`
	SourceText  *string           `json:"source_text,omitempty"`
	Status      SubmissionStatus  `json:"status"`
	Kind        SubmissionKind    `json:"kind"`
	CPJobID     *uuid.UUID        `json:"cp_job_id,omitempty"`
	Result      *SubmissionResult `json:"result,omitempty"`
	Validation  *CodeValidation   `json:"validation,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type CodeValidation struct {
	IsAllowed bool                   `json:"is_allowed"`
	Severity  string                 `json:"severity"`
	Reason    string                 `json:"reason"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type SubmissionResult struct {
	SubmissionID    uuid.UUID             `json:"submission_id"`
	OverallVerdict  string                `json:"overall_verdict"`
	TotalTimeMs     *int                  `json:"total_time_ms,omitempty"`
	MaxMemoryKb     *int                  `json:"max_memory_kb,omitempty"`
	WallTimeMs      *int                  `json:"wall_time_ms,omitempty"`
	CompilerOutput  *string               `json:"compiler_output,omitempty"`
	TestcaseResults []TestcaseResultEntry `json:"testcase_results"`
	CreatedAt       time.Time             `json:"created_at"`
}

type TestcaseResultEntry struct {
	TestcaseID   string          `json:"testcase_id"`
	Verdict      string          `json:"verdict"`
	TimeMs       *int            `json:"time_ms,omitempty"`
	MemoryKb     *int            `json:"memory_kb,omitempty"`
	ActualOutput *string         `json:"actual_output,omitempty"`
	StdoutOutput *string         `json:"stdout_output,omitempty"`
	InputData    json.RawMessage `json:"input_data,omitempty"`
	ExpectedData json.RawMessage `json:"expected_data,omitempty"`
}

type ProblemTestCase struct {
	ID           uuid.UUID       `json:"id"`
	ProblemID    uuid.UUID       `json:"problem_id"`
	ExternalID   string          `json:"external_id"`
	InputData    json.RawMessage `json:"input_data"`
	ExpectedData json.RawMessage `json:"expected_data"`
	OrderIndex   int             `json:"order_index"`
	IsActive     bool            `json:"is_active"`
	IsHidden     bool            `json:"is_hidden"`
}

type CreateSubmissionRequest struct {
	LanguageKey string `json:"language_key"`
	SourceText  string `json:"source_text"`
}

type ListFilters struct {
	UserID    uuid.UUID
	ProblemID *uuid.UUID
}
