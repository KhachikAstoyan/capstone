// Package domain contains the core types for the Execution Control Plane.
//
// The control plane is responsible for:
//   - Accepting job requests from the API service
//   - Tracking live workers and their health
//   - Assigning jobs to workers via a pull (worker-poll) model
//   - Managing per-job leases so stale jobs are automatically requeued
//   - Storing execution results reported by workers
//
// None of these types reference the main API database; the control plane is
// fully self-contained.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Job
// ---------------------------------------------------------------------------

// JobState is the lifecycle state of an execution job.
type JobState string

const (
	// JobStateQueued means the job is waiting to be picked up by a worker.
	JobStateQueued JobState = "queued"

	// JobStateAssigned means a worker has claimed the job and holds a lease,
	// but has not yet reported that execution has begun.
	JobStateAssigned JobState = "assigned"

	// JobStateRunning means the worker has confirmed execution is in progress.
	JobStateRunning JobState = "running"

	// JobStateCompleted means the worker reported a final result.
	JobStateCompleted JobState = "completed"

	// JobStateFailed is a terminal failure state (retries exhausted or
	// non-retryable error).
	JobStateFailed JobState = "failed"

	// JobStateRetryPending means the job's lease expired or a transient error
	// occurred; the background requeue loop will move it back to queued.
	JobStateRetryPending JobState = "retry_pending"
)

// Job is the central entity managed by the control plane.
type Job struct {
	ID            uuid.UUID `json:"id"`
	SubmissionID  uuid.UUID `json:"submission_id"` // opaque reference to main API DB

	// Source code — exactly one of SourceText / SourceRef is set.
	Language      string  `json:"language"`
	SourceText    *string `json:"source_text,omitempty"`
	SourceRef     *string `json:"source_ref,omitempty"`
	SourceSHA256  *string `json:"source_sha256,omitempty"`

	TimeLimitMs   int `json:"time_limit_ms"`
	MemoryLimitMb int `json:"memory_limit_mb"`

	TestCases []TestCase `json:"test_cases,omitempty"`

	State    JobState `json:"state"`
	WorkerID *string  `json:"worker_id,omitempty"`

	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`

	RetryCount  int `json:"retry_count"`
	MaxRetries  int `json:"max_retries"`

	FailureReason *string `json:"failure_reason,omitempty"`

	QueuedAt    time.Time  `json:"queued_at"`
	AssignedAt  *time.Time `json:"assigned_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Worker
// ---------------------------------------------------------------------------

// WorkerHealth is the operational status of a worker.
type WorkerHealth string

const (
	WorkerHealthy  WorkerHealth = "healthy"  // accepting new work
	WorkerDraining WorkerHealth = "draining" // finishing current work; no new assignments
	WorkerOffline  WorkerHealth = "offline"  // missed heartbeat deadline
)

// Worker represents a registered execution worker process.
type Worker struct {
	ID            string       `json:"id"`
	Languages     []string     `json:"languages"`
	Capacity      int          `json:"capacity"`
	ActiveJobs    int          `json:"active_jobs"`
	HealthStatus  WorkerHealth `json:"health_status"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	RegisteredAt  time.Time    `json:"registered_at"`
}

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

// TestCase is a single input/expected-output pair for a problem.
// Workers use these to drive execution and determine verdicts.
// The model is stdin/stdout: Input is piped to the program's stdin and
// ActualOutput is compared (trimmed) against Expected.
type TestCase struct {
	ID       string `json:"id"`       // matches testcases.external_id in the main DB
	Input    string `json:"input"`    // piped to stdin
	Expected string `json:"expected"` // expected stdout (compared after trim)
}

// Assignment is the payload returned to a worker when it successfully polls
// for a job.  It contains everything the worker needs to execute the job
// without making any additional calls to the control plane.
type Assignment struct {
	JobID          uuid.UUID  `json:"job_id"`
	SubmissionID   uuid.UUID  `json:"submission_id"`
	Language       string     `json:"language"`
	SourceText     *string    `json:"source_text,omitempty"`
	SourceRef      *string    `json:"source_ref,omitempty"`
	SourceSHA256   *string    `json:"source_sha256,omitempty"`
	TimeLimitMs    int        `json:"time_limit_ms"`
	MemoryLimitMb  int        `json:"memory_limit_mb"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at"`
	TestCases      []TestCase `json:"test_cases,omitempty"`
}

// ---------------------------------------------------------------------------
// Results
// ---------------------------------------------------------------------------

// JobResult is the overall outcome of a completed job.
type JobResult struct {
	JobID           uuid.UUID          `json:"job_id"`
	OverallVerdict  string             `json:"overall_verdict"`
	TotalTimeMs     *int               `json:"total_time_ms,omitempty"`
	MaxMemoryKb     *int               `json:"max_memory_kb,omitempty"`
	WallTimeMs      *int               `json:"wall_time_ms,omitempty"`
	CompilerOutput  *string            `json:"compiler_output,omitempty"`
	TestcaseResults []TestcaseResult   `json:"testcase_results,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

// TestcaseResult is the outcome for a single testcase within a job.
type TestcaseResult struct {
	TestcaseID   string  `json:"testcase_id"`
	Verdict      string  `json:"verdict"`
	TimeMs       *int    `json:"time_ms,omitempty"`
	MemoryKb     *int    `json:"memory_kb,omitempty"`
	ActualOutput *string `json:"actual_output,omitempty"`
	StdoutRef    *string `json:"stdout_ref,omitempty"`
	StderrRef    *string `json:"stderr_ref,omitempty"`
}

// ---------------------------------------------------------------------------
// Request types (used by the service layer)
// ---------------------------------------------------------------------------

// CreateJobRequest is sent by the API service when a user submits code.
// All fields needed for execution must be included; the control plane will
// not call back to the API database to resolve them.
type CreateJobRequest struct {
	SubmissionID   uuid.UUID  `json:"submission_id"`
	Language       string     `json:"language"`
	SourceText     *string    `json:"source_text,omitempty"`
	SourceRef      *string    `json:"source_ref,omitempty"`
	SourceSHA256   *string    `json:"source_sha256,omitempty"`
	TimeLimitMs    int        `json:"time_limit_ms"`
	MemoryLimitMb  int        `json:"memory_limit_mb"`
	TestCases      []TestCase `json:"test_cases,omitempty"`

	// MaxRetries overrides the default (3) if set to a positive value.
	MaxRetries int `json:"max_retries,omitempty"`
}

// HeartbeatRequest is sent by a worker on each heartbeat tick.
type HeartbeatRequest struct {
	WorkerID     string       `json:"worker_id"`
	Languages    []string     `json:"languages"`
	Capacity     int          `json:"capacity"`
	ActiveJobs   int          `json:"active_jobs"`
	HealthStatus WorkerHealth `json:"health_status"`
}

// PollRequest is sent by a worker when it is ready to accept a new job.
type PollRequest struct {
	WorkerID  string   `json:"worker_id"`
	Languages []string `json:"languages"`
}

// ReportResultRequest is sent by a worker when a job finishes (success or
// failure).
type ReportResultRequest struct {
	// WorkerID must match the worker that currently holds the lease.
	WorkerID       string `json:"worker_id"`

	OverallVerdict string  `json:"overall_verdict"`
	TotalTimeMs    *int    `json:"total_time_ms,omitempty"`
	MaxMemoryKb    *int    `json:"max_memory_kb,omitempty"`
	WallTimeMs     *int    `json:"wall_time_ms,omitempty"`
	CompilerOutput *string `json:"compiler_output,omitempty"`

	TestcaseResults []TestcaseResultInput `json:"testcase_results,omitempty"`
}

// TestcaseResultInput is a single testcase entry inside ReportResultRequest.
type TestcaseResultInput struct {
	TestcaseID   string  `json:"testcase_id"`
	Verdict      string  `json:"verdict"`
	TimeMs       *int    `json:"time_ms,omitempty"`
	MemoryKb     *int    `json:"memory_kb,omitempty"`
	ActualOutput *string `json:"actual_output,omitempty"`
	StdoutRef    *string `json:"stdout_ref,omitempty"`
	StderrRef    *string `json:"stderr_ref,omitempty"`
}

// RenewLeaseRequest is sent by a worker to extend its hold on a job.
type RenewLeaseRequest struct {
	WorkerID string `json:"worker_id"`
}
