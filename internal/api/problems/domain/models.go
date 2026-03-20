package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProblemVisibility string

const (
	VisibilityDraft     ProblemVisibility = "draft"
	VisibilityPublished ProblemVisibility = "published"
	VisibilityArchived  ProblemVisibility = "archived"
)

type Problem struct {
	ID                uuid.UUID         `json:"id"`
	Slug              string            `json:"slug"`
	Title             string            `json:"title"`
	StatementMarkdown string            `json:"statement_markdown"`
	TimeLimitMs       int               `json:"time_limit_ms"`
	MemoryLimitMb     int               `json:"memory_limit_mb"`
	TestsRef          string            `json:"tests_ref"`
	TestsHash         *string           `json:"tests_hash,omitempty"`
	Visibility        ProblemVisibility `json:"visibility"`
	CreatedByUserID   *uuid.UUID        `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type CreateProblemRequest struct {
	Title             string            `json:"title"`
	StatementMarkdown string            `json:"statement_markdown"`
	TimeLimitMs       int               `json:"time_limit_ms"`
	MemoryLimitMb     int               `json:"memory_limit_mb"`
	TestsRef          string            `json:"tests_ref"`
	Visibility        ProblemVisibility `json:"visibility"`
}

type UpdateProblemRequest struct {
	Slug              *string            `json:"slug,omitempty"`
	Title             *string            `json:"title,omitempty"`
	StatementMarkdown *string            `json:"statement_markdown,omitempty"`
	TimeLimitMs       *int               `json:"time_limit_ms,omitempty"`
	MemoryLimitMb     *int               `json:"memory_limit_mb,omitempty"`
	TestsRef          *string            `json:"tests_ref,omitempty"`
	Visibility        *ProblemVisibility `json:"visibility,omitempty"`
}
