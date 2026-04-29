package domain

import (
	"encoding/json"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/submissions/driver"
	"github.com/google/uuid"
)

type ProblemVisibility string

const (
	VisibilityDraft     ProblemVisibility = "draft"
	VisibilityPublished ProblemVisibility = "published"
	VisibilityArchived  ProblemVisibility = "archived"
)

type ProblemDifficulty string

const (
	DifficultyEasy   ProblemDifficulty = "easy"
	DifficultyMedium ProblemDifficulty = "medium"
	DifficultyHard   ProblemDifficulty = "hard"
)

type Problem struct {
	ID                uuid.UUID         `json:"id"`
	Slug              string            `json:"slug"`
	Title             string            `json:"title"`
	Summary           string            `json:"summary"`
	StatementMarkdown string            `json:"statement_markdown"`
	TimeLimitMs       int               `json:"time_limit_ms"`
	MemoryLimitMb     int               `json:"memory_limit_mb"`
	TestsRef          string            `json:"tests_ref"`
	TestsHash         *string           `json:"tests_hash,omitempty"`
	Visibility        ProblemVisibility `json:"visibility"`
	Difficulty        ProblemDifficulty `json:"difficulty"`
	CreatedByUserID   *uuid.UUID        `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	Tags              []string                `json:"tags,omitempty"`
	AcceptanceRate    *float64                `json:"acceptance_rate,omitempty"`
	IsSolved          bool                    `json:"is_solved"`
	FunctionSpec      *driver.FunctionSpec    `json:"function_spec,omitempty"`
}

type CreateProblemRequest struct {
	Title             string                `json:"title"`
	Summary           string                `json:"summary"`
	StatementMarkdown string                `json:"statement_markdown"`
	TimeLimitMs       int                   `json:"time_limit_ms"`
	MemoryLimitMb     int                   `json:"memory_limit_mb"`
	TestsRef          string                `json:"tests_ref"`
	Visibility        ProblemVisibility     `json:"visibility"`
	Difficulty        ProblemDifficulty     `json:"difficulty"`
	FunctionSpec      *driver.FunctionSpec  `json:"function_spec,omitempty"`
}

type UpdateProblemRequest struct {
	Slug              *string               `json:"slug,omitempty"`
	Title             *string               `json:"title,omitempty"`
	Summary           *string               `json:"summary,omitempty"`
	StatementMarkdown *string               `json:"statement_markdown,omitempty"`
	TimeLimitMs       *int                  `json:"time_limit_ms,omitempty"`
	MemoryLimitMb     *int                  `json:"memory_limit_mb,omitempty"`
	TestsRef          *string               `json:"tests_ref,omitempty"`
	Visibility        *ProblemVisibility    `json:"visibility,omitempty"`
	Difficulty        *ProblemDifficulty    `json:"difficulty,omitempty"`
	FunctionSpec      *driver.FunctionSpec  `json:"function_spec,omitempty"`
}

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

type TestCase struct {
	ID           uuid.UUID       `json:"id"`
	ProblemID    uuid.UUID       `json:"problem_id"`
	ExternalID   string          `json:"external_id"`
	InputData    json.RawMessage `json:"input_data"`
	ExpectedData json.RawMessage `json:"expected_data"`
	OrderIndex   int             `json:"order_index"`
	IsActive     bool            `json:"is_active"`
	IsHidden     bool            `json:"is_hidden"`
	CreatedAt    time.Time       `json:"created_at"`
}

type CreateTestCaseRequest struct {
	InputData    json.RawMessage `json:"input_data"`
	ExpectedData json.RawMessage `json:"expected_data"`
	OrderIndex   int             `json:"order_index"`
	IsHidden     bool            `json:"is_hidden"`
}

type UpdateTestCaseRequest struct {
	InputData    json.RawMessage `json:"input_data,omitempty"`
	ExpectedData json.RawMessage `json:"expected_data,omitempty"`
	OrderIndex   *int            `json:"order_index,omitempty"`
	IsHidden     *bool           `json:"is_hidden,omitempty"`
}
