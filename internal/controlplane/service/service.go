// Package service implements the business logic of the Execution Control Plane.
//
// The service layer sits between the HTTP handlers and the repository layer.
// It owns:
//   - Input validation
//   - The scheduling decision (which worker gets the next job)
//   - Lease lifecycle management
//   - The background loops for lease expiry and worker health
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Default durations — callers can override via Config.
const (
	DefaultLeaseDuration      = 60 * time.Second
	DefaultLeaseCheckInterval = 10 * time.Second
	DefaultHeartbeatTimeout   = 30 * time.Second // workers offline after this
)

// Sentinel errors surfaced to HTTP handlers.
var (
	ErrJobNotFound    = repository.ErrJobNotFound
	ErrNoJobAvailable = repository.ErrNoJobAvailable
	ErrLeaseMismatch  = repository.ErrLeaseMismatch
	ErrInvalidInput   = errors.New("invalid input")
)

// Service is the primary interface for the control plane.
// HTTP handlers depend on this interface, not on the concrete struct, so they
// are easy to test with a mock.
type Service interface {
	// ── API-facing ──────────────────────────────────────────────────────────

	// CreateJob enqueues a new execution job.  Called by the main API service
	// when a user submits code.
	CreateJob(ctx context.Context, req domain.CreateJobRequest) (*domain.Job, error)

	// GetJob returns the current state of a job by its id.
	GetJob(ctx context.Context, jobID uuid.UUID) (*domain.Job, error)

	// GetJobBySubmission returns the most-recent job for a given submission id.
	GetJobBySubmission(ctx context.Context, submissionID uuid.UUID) (*domain.Job, error)

	// GetJobResult returns the stored result for a completed job.
	GetJobResult(ctx context.Context, jobID uuid.UUID) (*domain.JobResult, error)

	// ── Worker-facing ────────────────────────────────────────────────────────

	// Heartbeat registers a worker (first call) or refreshes its state
	// (subsequent calls).  Workers must call this at least once per
	// HeartbeatTimeout or they will be marked offline.
	Heartbeat(ctx context.Context, req domain.HeartbeatRequest) (*domain.Worker, error)

	// PollJob attempts to assign the next available queued job to the calling
	// worker.  Returns ErrNoJobAvailable when the queue is empty for the
	// worker's supported languages.
	PollJob(ctx context.Context, req domain.PollRequest) (*domain.Assignment, error)

	// MarkRunning transitions a job from assigned → running.  Workers call
	// this when execution has actually started.
	MarkRunning(ctx context.Context, jobID uuid.UUID, workerID string) error

	// RenewLease extends the job's lease by LeaseDuration from now.
	// Workers must call this before the current lease_expires_at.
	RenewLease(ctx context.Context, jobID uuid.UUID, req domain.RenewLeaseRequest) error

	// ReportResult records the execution outcome and transitions the job to
	// completed (or failed for non-retryable errors).
	ReportResult(ctx context.Context, jobID uuid.UUID, req domain.ReportResultRequest) error

	// ── Background loops (called by the main process) ────────────────────────

	// RequeueExpiredLeases sweeps for jobs with lapsed leases and requeues
	// them.  Intended to be called on a ticker by the main goroutine.
	RequeueExpiredLeases(ctx context.Context) (int, error)

	// MarkStaleWorkers sweeps for workers that have not sent a heartbeat within
	// HeartbeatTimeout and marks them offline.  Also intended for a ticker.
	MarkStaleWorkers(ctx context.Context) (int, error)
}

// Config holds tunables for the service.
type Config struct {
	LeaseDuration    time.Duration
	HeartbeatTimeout time.Duration
}

type service struct {
	jobs    repository.JobRepository
	workers repository.WorkerRepository
	cfg     Config
	log     *zap.Logger
}

// New creates a Service.  If cfg fields are zero, defaults are applied.
func New(jobs repository.JobRepository, workers repository.WorkerRepository, cfg Config, log *zap.Logger) Service {
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = DefaultLeaseDuration
	}
	if cfg.HeartbeatTimeout == 0 {
		cfg.HeartbeatTimeout = DefaultHeartbeatTimeout
	}
	return &service{jobs: jobs, workers: workers, cfg: cfg, log: log}
}

// ---------------------------------------------------------------------------
// CreateJob
// ---------------------------------------------------------------------------

func (s *service) CreateJob(ctx context.Context, req domain.CreateJobRequest) (*domain.Job, error) {
	if err := validateCreateJobRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	job, err := s.jobs.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	s.log.Info("job enqueued",
		zap.String("job_id", job.ID.String()),
		zap.String("submission_id", job.SubmissionID.String()),
		zap.String("language", job.Language),
	)
	return job, nil
}

// ---------------------------------------------------------------------------
// GetJob / GetJobBySubmission / GetJobResult
// ---------------------------------------------------------------------------

func (s *service) GetJob(ctx context.Context, jobID uuid.UUID) (*domain.Job, error) {
	return s.jobs.GetByID(ctx, jobID)
}

func (s *service) GetJobBySubmission(ctx context.Context, submissionID uuid.UUID) (*domain.Job, error) {
	return s.jobs.GetBySubmissionID(ctx, submissionID)
}

func (s *service) GetJobResult(ctx context.Context, jobID uuid.UUID) (*domain.JobResult, error) {
	return s.jobs.GetResult(ctx, jobID)
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

func (s *service) Heartbeat(ctx context.Context, req domain.HeartbeatRequest) (*domain.Worker, error) {
	if req.WorkerID == "" {
		return nil, fmt.Errorf("%w: worker_id is required", ErrInvalidInput)
	}
	if req.Capacity <= 0 {
		return nil, fmt.Errorf("%w: capacity must be positive", ErrInvalidInput)
	}

	// Validate health_status value.
	switch req.HealthStatus {
	case domain.WorkerHealthy, domain.WorkerDraining, domain.WorkerOffline:
	default:
		return nil, fmt.Errorf("%w: unknown health_status %q", ErrInvalidInput, req.HealthStatus)
	}

	w, err := s.workers.Upsert(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat upsert: %w", err)
	}
	return w, nil
}

// ---------------------------------------------------------------------------
// PollJob
// ---------------------------------------------------------------------------

// PollJob implements the worker-pull scheduling model:
//  1. Validate the worker is known and healthy.
//  2. Check the worker has spare capacity.
//  3. Atomically claim the next queued job via AssignNext (SKIP LOCKED).
//  4. Increment the worker's active_jobs counter.
func (s *service) PollJob(ctx context.Context, req domain.PollRequest) (*domain.Assignment, error) {
	if req.WorkerID == "" {
		return nil, fmt.Errorf("%w: worker_id is required", ErrInvalidInput)
	}
	if len(req.Languages) == 0 {
		return nil, fmt.Errorf("%w: languages must not be empty", ErrInvalidInput)
	}

	// Verify worker is healthy and has capacity.
	w, err := s.workers.GetByID(ctx, req.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("unknown worker %q: %w", req.WorkerID, err)
	}
	if w.HealthStatus != domain.WorkerHealthy {
		return nil, fmt.Errorf("%w: worker %q is %s", ErrInvalidInput, req.WorkerID, w.HealthStatus)
	}
	if w.ActiveJobs >= w.Capacity {
		return nil, ErrNoJobAvailable // caller should back off and retry later
	}

	job, err := s.jobs.AssignNext(ctx, req.WorkerID, req.Languages, s.cfg.LeaseDuration)
	if err != nil {
		return nil, err // ErrNoJobAvailable propagates as-is
	}

	// Bump the worker's active_jobs counter (best-effort; the heartbeat will
	// reconcile the real count on the next tick).
	if incErr := s.workers.IncrementActiveJobs(ctx, req.WorkerID, +1); incErr != nil {
		s.log.Warn("failed to increment active_jobs after poll",
			zap.String("worker_id", req.WorkerID),
			zap.Error(incErr),
		)
	}

	s.log.Info("job assigned",
		zap.String("job_id", job.ID.String()),
		zap.String("worker_id", req.WorkerID),
		zap.Timep("lease_expires_at", job.LeaseExpiresAt),
	)

	assignment := &domain.Assignment{
		JobID:          job.ID,
		SubmissionID:   job.SubmissionID,
		Language:       job.Language,
		SourceText:     job.SourceText,
		SourceRef:      job.SourceRef,
		SourceSHA256:   job.SourceSHA256,
		TimeLimitMs:    job.TimeLimitMs,
		MemoryLimitMb:  job.MemoryLimitMb,
		LeaseExpiresAt: *job.LeaseExpiresAt,
		TestCases:      job.TestCases,
	}
	return assignment, nil
}

// ---------------------------------------------------------------------------
// MarkRunning
// ---------------------------------------------------------------------------

func (s *service) MarkRunning(ctx context.Context, jobID uuid.UUID, workerID string) error {
	if workerID == "" {
		return fmt.Errorf("%w: worker_id is required", ErrInvalidInput)
	}
	return s.jobs.MarkRunning(ctx, jobID, workerID)
}

// ---------------------------------------------------------------------------
// RenewLease
// ---------------------------------------------------------------------------

func (s *service) RenewLease(ctx context.Context, jobID uuid.UUID, req domain.RenewLeaseRequest) error {
	if req.WorkerID == "" {
		return fmt.Errorf("%w: worker_id is required", ErrInvalidInput)
	}
	return s.jobs.RenewLease(ctx, jobID, req.WorkerID, s.cfg.LeaseDuration)
}

// ---------------------------------------------------------------------------
// ReportResult
// ---------------------------------------------------------------------------

func (s *service) ReportResult(ctx context.Context, jobID uuid.UUID, req domain.ReportResultRequest) error {
	if req.WorkerID == "" {
		return fmt.Errorf("%w: worker_id is required", ErrInvalidInput)
	}
	if req.OverallVerdict == "" {
		return fmt.Errorf("%w: overall_verdict is required", ErrInvalidInput)
	}

	if err := s.jobs.Complete(ctx, jobID, req.WorkerID, req); err != nil {
		return err
	}

	// Decrement active_jobs on the worker (best-effort).
	if decErr := s.workers.IncrementActiveJobs(ctx, req.WorkerID, -1); decErr != nil {
		s.log.Warn("failed to decrement active_jobs after result report",
			zap.String("worker_id", req.WorkerID),
			zap.Error(decErr),
		)
	}

	s.log.Info("job completed",
		zap.String("job_id", jobID.String()),
		zap.String("worker_id", req.WorkerID),
		zap.String("verdict", req.OverallVerdict),
	)
	return nil
}

// ---------------------------------------------------------------------------
// Background loops
// ---------------------------------------------------------------------------

// RequeueExpiredLeases is called on a ticker (see cmd/control-plane/main.go).
// It requeues jobs whose workers vanished or timed out.
func (s *service) RequeueExpiredLeases(ctx context.Context) (int, error) {
	n, err := s.jobs.RequeueExpiredLeases(ctx)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		s.log.Info("requeued expired leases", zap.Int("count", n))
	}
	return n, nil
}

// MarkStaleWorkers marks workers offline when they have not sent a heartbeat
// within HeartbeatTimeout.
func (s *service) MarkStaleWorkers(ctx context.Context) (int, error) {
	threshold := time.Now().Add(-s.cfg.HeartbeatTimeout)
	n, err := s.workers.MarkStaleOffline(ctx, threshold)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		s.log.Warn("marked workers offline due to missed heartbeat", zap.Int("count", n))
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

func validateCreateJobRequest(req domain.CreateJobRequest) error {
	if req.SubmissionID == uuid.Nil {
		return errors.New("submission_id is required")
	}
	if req.Language == "" {
		return errors.New("language is required")
	}
	if req.SourceText == nil && req.SourceRef == nil {
		return errors.New("exactly one of source_text or source_ref must be provided")
	}
	if req.SourceText != nil && req.SourceRef != nil {
		return errors.New("exactly one of source_text or source_ref must be provided")
	}
	if req.TimeLimitMs <= 0 {
		return errors.New("time_limit_ms must be positive")
	}
	if req.MemoryLimitMb <= 0 {
		return errors.New("memory_limit_mb must be positive")
	}
	return nil
}
