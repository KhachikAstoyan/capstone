// Package repository contains the database access layer for the control plane.
// All SQL lives here; the service layer never constructs queries directly.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Sentinel errors returned by the job repository.
var (
	ErrJobNotFound      = errors.New("job not found")
	ErrNoJobAvailable   = errors.New("no job available for the requested languages")
	ErrLeaseMismatch    = errors.New("lease renewal rejected: worker_id does not match current holder")
)

// JobRepository defines all database operations on jobs.
type JobRepository interface {
	// Create inserts a new job in the 'queued' state and returns it.
	Create(ctx context.Context, req domain.CreateJobRequest) (*domain.Job, error)

	// GetByID fetches a single job by its primary key.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)

	// GetBySubmissionID returns the most-recent job for a given submission.
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Job, error)

	// AssignNext atomically claims the oldest queued job whose language is in
	// the provided set, marks it 'assigned', sets the worker_id, and writes
	// lease_expires_at = NOW() + leaseDuration.  Returns ErrNoJobAvailable
	// when the queue is empty for those languages.
	//
	// Uses SELECT … FOR UPDATE SKIP LOCKED so concurrent workers never race.
	AssignNext(ctx context.Context, workerID string, languages []string, leaseDuration time.Duration) (*domain.Job, error)

	// MarkRunning transitions a job from 'assigned' to 'running' and records
	// started_at.  Called by the worker once execution has actually begun.
	MarkRunning(ctx context.Context, jobID uuid.UUID, workerID string) error

	// RenewLease extends lease_expires_at by leaseDuration from now.
	// Returns ErrLeaseMismatch if workerID does not hold the lease.
	RenewLease(ctx context.Context, jobID uuid.UUID, workerID string, leaseDuration time.Duration) error

	// Complete records the result rows and transitions the job to 'completed'.
	Complete(ctx context.Context, jobID uuid.UUID, workerID string, req domain.ReportResultRequest) error

	// RequeueExpiredLeases finds all assigned/running jobs whose
	// lease_expires_at is in the past.  Jobs that still have retries left are
	// moved to 'retry_pending' with retry_count incremented; jobs that have
	// exhausted retries are moved to 'failed'.
	// Returns the number of jobs that were requeued.
	RequeueExpiredLeases(ctx context.Context) (int, error)

	// GetResult returns the stored result for a completed job.
	GetResult(ctx context.Context, jobID uuid.UUID) (*domain.JobResult, error)
}

type jobRepository struct {
	db *sql.DB
}

// NewJobRepository creates a JobRepository backed by a *sql.DB.
func NewJobRepository(db *sql.DB) JobRepository {
	return &jobRepository{db: db}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func (r *jobRepository) Create(ctx context.Context, req domain.CreateJobRequest) (*domain.Job, error) {
	maxRetries := 3
	if req.MaxRetries > 0 {
		maxRetries = req.MaxRetries
	}

	tcJSON, err := marshalTestCases(req.TestCases)
	if err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO jobs (
			submission_id, language,
			source_text, source_ref, source_sha256,
			time_limit_ms, memory_limit_mb,
			test_cases, max_retries
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING
			id, submission_id, language,
			source_text, source_ref, source_sha256,
			time_limit_ms, memory_limit_mb,
			test_cases,
			state, worker_id, lease_expires_at,
			retry_count, max_retries, failure_reason,
			queued_at, assigned_at, started_at, finished_at
	`

	row := r.db.QueryRowContext(ctx, q,
		req.SubmissionID, req.Language,
		req.SourceText, req.SourceRef, req.SourceSHA256,
		req.TimeLimitMs, req.MemoryLimitMb,
		tcJSON, maxRetries,
	)

	return scanJob(row)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func (r *jobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	const q = `
		SELECT
			id, submission_id, language,
			source_text, source_ref, source_sha256,
			time_limit_ms, memory_limit_mb,
			test_cases,
			state, worker_id, lease_expires_at,
			retry_count, max_retries, failure_reason,
			queued_at, assigned_at, started_at, finished_at
		FROM jobs WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	job, err := scanJob(row)
	if errors.Is(err, ErrJobNotFound) {
		return nil, ErrJobNotFound
	}
	return job, err
}

// ---------------------------------------------------------------------------
// GetBySubmissionID
// ---------------------------------------------------------------------------

func (r *jobRepository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Job, error) {
	const q = `
		SELECT
			id, submission_id, language,
			source_text, source_ref, source_sha256,
			time_limit_ms, memory_limit_mb,
			test_cases,
			state, worker_id, lease_expires_at,
			retry_count, max_retries, failure_reason,
			queued_at, assigned_at, started_at, finished_at
		FROM jobs
		WHERE submission_id = $1
		ORDER BY queued_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, q, submissionID)
	job, err := scanJob(row)
	if errors.Is(err, ErrJobNotFound) {
		return nil, ErrJobNotFound
	}
	return job, err
}

// ---------------------------------------------------------------------------
// AssignNext
// ---------------------------------------------------------------------------

func (r *jobRepository) AssignNext(ctx context.Context, workerID string, languages []string, leaseDuration time.Duration) (*domain.Job, error) {
	// The CTE selects the oldest queued job whose language is supported by this
	// worker, locking that row exclusively.  SKIP LOCKED means concurrent
	// workers each get a different row instead of blocking each other.
	const q = `
		WITH next AS (
			SELECT id
			FROM   jobs
			WHERE  state    = 'queued'
			  AND  language = ANY($1)
			ORDER  BY queued_at ASC
			LIMIT  1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE jobs
		SET
			state            = 'assigned',
			worker_id        = $2,
			lease_expires_at = NOW() + $3::interval,
			assigned_at      = NOW()
		FROM next
		WHERE jobs.id = next.id
		RETURNING
			jobs.id, jobs.submission_id, jobs.language,
			jobs.source_text, jobs.source_ref, jobs.source_sha256,
			jobs.time_limit_ms, jobs.memory_limit_mb,
			jobs.test_cases,
			jobs.state, jobs.worker_id, jobs.lease_expires_at,
			jobs.retry_count, jobs.max_retries, jobs.failure_reason,
			jobs.queued_at, jobs.assigned_at, jobs.started_at, jobs.finished_at
	`

	interval := fmt.Sprintf("%d seconds", int(leaseDuration.Seconds()))
	row := r.db.QueryRowContext(ctx, q, pq.Array(languages), workerID, interval)
	job, err := scanJob(row)
	if errors.Is(err, ErrJobNotFound) {
		return nil, ErrNoJobAvailable
	}
	return job, err
}

// ---------------------------------------------------------------------------
// MarkRunning
// ---------------------------------------------------------------------------

func (r *jobRepository) MarkRunning(ctx context.Context, jobID uuid.UUID, workerID string) error {
	const q = `
		UPDATE jobs
		SET state = 'running', started_at = NOW()
		WHERE id = $1 AND worker_id = $2 AND state = 'assigned'
	`
	res, err := r.db.ExecContext(ctx, q, jobID, workerID)
	if err != nil {
		return fmt.Errorf("mark running: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrLeaseMismatch
	}
	return nil
}

// ---------------------------------------------------------------------------
// RenewLease
// ---------------------------------------------------------------------------

func (r *jobRepository) RenewLease(ctx context.Context, jobID uuid.UUID, workerID string, leaseDuration time.Duration) error {
	interval := fmt.Sprintf("%d seconds", int(leaseDuration.Seconds()))
	const q = `
		UPDATE jobs
		SET lease_expires_at = NOW() + $3::interval
		WHERE id = $1 AND worker_id = $2 AND state IN ('assigned','running')
	`
	res, err := r.db.ExecContext(ctx, q, jobID, workerID, interval)
	if err != nil {
		return fmt.Errorf("renew lease: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrLeaseMismatch
	}
	return nil
}

// ---------------------------------------------------------------------------
// Complete
// ---------------------------------------------------------------------------

func (r *jobRepository) Complete(ctx context.Context, jobID uuid.UUID, workerID string, req domain.ReportResultRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Transition job state.
	const updateJob = `
		UPDATE jobs
		SET state = 'completed', finished_at = NOW(), lease_expires_at = NULL
		WHERE id = $1 AND worker_id = $2 AND state IN ('assigned','running')
	`
	res, err := tx.ExecContext(ctx, updateJob, jobID, workerID)
	if err != nil {
		return fmt.Errorf("complete job update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrLeaseMismatch
	}

	// Insert overall result.
	const insertResult = `
		INSERT INTO job_results (job_id, overall_verdict, total_time_ms, max_memory_kb, wall_time_ms, compiler_output)
		VALUES ($1,$2,$3,$4,$5,$6)
	`
	if _, err = tx.ExecContext(ctx, insertResult,
		jobID, req.OverallVerdict,
		req.TotalTimeMs, req.MaxMemoryKb, req.WallTimeMs, req.CompilerOutput,
	); err != nil {
		return fmt.Errorf("insert job_result: %w", err)
	}

	// Insert per-testcase rows.
	for _, tc := range req.TestcaseResults {
		const insertTC = `
			INSERT INTO job_tc_results (job_id, testcase_id, verdict, time_ms, memory_kb, actual_output, stdout_ref, stderr_ref)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`
		if _, err = tx.ExecContext(ctx, insertTC,
			jobID, tc.TestcaseID, tc.Verdict,
			tc.TimeMs, tc.MemoryKb, tc.ActualOutput, tc.StdoutRef, tc.StderrRef,
		); err != nil {
			return fmt.Errorf("insert job_tc_result (testcase %s): %w", tc.TestcaseID, err)
		}
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// RequeueExpiredLeases
// ---------------------------------------------------------------------------

// RequeueExpiredLeases is called by the background lease-expiry goroutine.
// It moves expired leases back to 'queued' (retries remaining) or 'failed'
// (retries exhausted) in a single UPDATE, returning how many rows changed.
func (r *jobRepository) RequeueExpiredLeases(ctx context.Context) (int, error) {
	const q = `
		UPDATE jobs
		SET
			state            = CASE
				WHEN retry_count + 1 < max_retries THEN 'queued'::JOB_STATE
				ELSE                                     'failed'::JOB_STATE
			END,
			retry_count      = retry_count + 1,
			worker_id        = NULL,
			lease_expires_at = NULL,
			failure_reason   = CASE
				WHEN retry_count + 1 >= max_retries THEN 'lease expired – retries exhausted'
				ELSE NULL
			END
		WHERE
			state IN ('assigned','running')
			AND lease_expires_at < NOW()
	`
	res, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("requeue expired leases: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// ---------------------------------------------------------------------------
// GetResult
// ---------------------------------------------------------------------------

func (r *jobRepository) GetResult(ctx context.Context, jobID uuid.UUID) (*domain.JobResult, error) {
	const q = `
		SELECT job_id, overall_verdict, total_time_ms, max_memory_kb, wall_time_ms, compiler_output, created_at
		FROM job_results
		WHERE job_id = $1
	`
	row := r.db.QueryRowContext(ctx, q, jobID)

	result := &domain.JobResult{}
	err := row.Scan(
		&result.JobID, &result.OverallVerdict,
		&result.TotalTimeMs, &result.MaxMemoryKb, &result.WallTimeMs,
		&result.CompilerOutput, &result.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("get job result: %w", err)
	}

	// Fetch per-testcase rows.
	const tcQ = `
		SELECT testcase_id, verdict, time_ms, memory_kb, actual_output, stdout_ref, stderr_ref
		FROM   job_tc_results
		WHERE  job_id = $1
		ORDER  BY testcase_id
	`
	rows, err := r.db.QueryContext(ctx, tcQ, jobID)
	if err != nil {
		return nil, fmt.Errorf("get tc results: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tc domain.TestcaseResult
		if err := rows.Scan(&tc.TestcaseID, &tc.Verdict, &tc.TimeMs, &tc.MemoryKb, &tc.ActualOutput, &tc.StdoutRef, &tc.StderrRef); err != nil {
			return nil, fmt.Errorf("scan tc result: %w", err)
		}
		result.TestcaseResults = append(result.TestcaseResults, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter tc results: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// scanJob — shared helper
// ---------------------------------------------------------------------------

// scanJob scans a single jobs row into a domain.Job, mapping sql.ErrNoRows to
// ErrJobNotFound.  Callers must include test_cases in the SELECT column list
// in the same position as scanned here.
func scanJob(row *sql.Row) (*domain.Job, error) {
	j := &domain.Job{}
	var tcJSON []byte
	err := row.Scan(
		&j.ID, &j.SubmissionID, &j.Language,
		&j.SourceText, &j.SourceRef, &j.SourceSHA256,
		&j.TimeLimitMs, &j.MemoryLimitMb,
		&tcJSON,
		&j.State, &j.WorkerID, &j.LeaseExpiresAt,
		&j.RetryCount, &j.MaxRetries, &j.FailureReason,
		&j.QueuedAt, &j.AssignedAt, &j.StartedAt, &j.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("scan job: %w", err)
	}
	if len(tcJSON) > 0 {
		if err := json.Unmarshal(tcJSON, &j.TestCases); err != nil {
			return nil, fmt.Errorf("unmarshal test_cases: %w", err)
		}
	}
	return j, nil
}

// ---------------------------------------------------------------------------
// JSONB helpers
// ---------------------------------------------------------------------------

// marshalTestCases serialises test cases to a JSONB-compatible []byte.
// Returns nil (SQL NULL) when the slice is empty.
func marshalTestCases(tcs []domain.TestCase) ([]byte, error) {
	if len(tcs) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(tcs)
	if err != nil {
		return nil, fmt.Errorf("marshal test_cases: %w", err)
	}
	return b, nil
}
