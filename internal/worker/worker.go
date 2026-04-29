package worker

import (
	"context"
	"sync"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Worker is the top-level struct that drives the worker process.
//
// Lifecycle
// ─────────
//  1. Call New() with config, client, executor, logger.
//  2. Call Run(ctx).  This blocks until ctx is cancelled.
//     Internally it starts three loops:
//     • heartbeat — keeps the worker registered and healthy
//     • poll      — claims jobs when there is free capacity
//     • per-job   — executes the job and renews the lease
//  3. On shutdown (ctx cancelled), the worker waits for in-flight jobs to
//     finish (or their context to be cancelled) before returning.
type Worker struct {
	id       string
	langs    []string
	capacity int

	client   *ControlPlaneClient
	executor Executor
	log      *zap.Logger

	heartbeatInterval    time.Duration
	pollInterval         time.Duration
	leaseRenewalInterval time.Duration

	// mu protects activeJobs.
	mu         sync.Mutex
	activeJobs map[uuid.UUID]context.CancelFunc
}

// New creates a Worker.  workerID may be empty — a random UUID is generated.
func New(cfg *Config, client *ControlPlaneClient, executor Executor, log *zap.Logger) *Worker {
	id := cfg.WorkerID
	if id == "" {
		id = uuid.New().String()
		log.Info("generated worker id", zap.String("worker_id", id))
	}

	log.Info("worker configured",
		zap.String("worker_id", id),
		zap.Strings("languages", cfg.LanguageList()),
		zap.Int("capacity", cfg.Capacity),
		zap.Duration("heartbeat_interval", cfg.HeartbeatInterval()),
		zap.Duration("poll_interval", cfg.PollInterval()),
		zap.Duration("lease_renewal_interval", cfg.LeaseRenewalInterval()),
	)

	return &Worker{
		id:                   id,
		langs:                cfg.LanguageList(),
		capacity:             cfg.Capacity,
		client:               client,
		executor:             executor,
		log:                  log,
		heartbeatInterval:    cfg.HeartbeatInterval(),
		pollInterval:         cfg.PollInterval(),
		leaseRenewalInterval: cfg.LeaseRenewalInterval(),
		activeJobs:           make(map[uuid.UUID]context.CancelFunc),
	}
}

// Run starts all worker loops and blocks until ctx is cancelled.
// After cancellation it waits for in-flight jobs to drain.
func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup

	w.log.Info("worker run loop starting")

	// Send an immediate heartbeat so the control plane knows about us.
	w.log.Info("sending initial heartbeat")
	w.sendHeartbeat(ctx)

	// Heartbeat loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.log.Info("heartbeat loop started")
		w.heartbeatLoop(ctx)
		w.log.Info("heartbeat loop stopped")
	}()

	// Poll loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.log.Info("poll loop started")
		w.pollLoop(ctx)
		w.log.Info("poll loop stopped")
	}()

	// Wait for shutdown signal, then wait for in-flight work.
	<-ctx.Done()
	w.log.Info("shutting down, waiting for in-flight jobs to finish...")
	wg.Wait()
	w.log.Info("all loops stopped")
}

// ---------------------------------------------------------------------------
// Heartbeat loop
// ---------------------------------------------------------------------------

func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sendHeartbeat(ctx)
		}
	}
}

func (w *Worker) sendHeartbeat(ctx context.Context) {
	w.mu.Lock()
	activeCount := len(w.activeJobs)
	w.mu.Unlock()

	req := domain.HeartbeatRequest{
		WorkerID:     w.id,
		Languages:    w.langs,
		Capacity:     w.capacity,
		ActiveJobs:   activeCount,
		HealthStatus: domain.WorkerHealthy,
	}

	w.log.Info("sending heartbeat",
		zap.String("worker_id", req.WorkerID),
		zap.Strings("languages", req.Languages),
		zap.Int("capacity", req.Capacity),
		zap.Int("active_jobs", req.ActiveJobs),
		zap.String("health_status", string(req.HealthStatus)),
	)
	if _, err := w.client.Heartbeat(ctx, req); err != nil {
		w.log.Warn("heartbeat failed", zap.Error(err))
		return
	}
	w.log.Info("heartbeat accepted")
}

// ---------------------------------------------------------------------------
// Poll loop
// ---------------------------------------------------------------------------

func (w *Worker) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tryPoll(ctx)
		}
	}
}

func (w *Worker) tryPoll(ctx context.Context) {
	// Check capacity.
	w.mu.Lock()
	activeCount := len(w.activeJobs)
	if activeCount >= w.capacity {
		w.mu.Unlock()
		w.log.Info("skipping poll because worker is at capacity",
			zap.Int("active_jobs", activeCount),
			zap.Int("capacity", w.capacity),
		)
		return
	}
	w.mu.Unlock()

	req := domain.PollRequest{
		WorkerID:  w.id,
		Languages: w.langs,
	}
	w.log.Info("polling for job",
		zap.String("worker_id", req.WorkerID),
		zap.Strings("languages", req.Languages),
		zap.Int("active_jobs", activeCount),
		zap.Int("capacity", w.capacity),
	)
	assignment, err := w.client.Poll(ctx, req)
	if err != nil {
		w.log.Warn("poll failed", zap.Error(err))
		return
	}
	if assignment == nil {
		// No work available — the normal idle case.
		w.log.Info("poll returned no job")
		return
	}

	w.log.Info("received job assignment",
		zap.String("job_id", assignment.JobID.String()),
		zap.String("submission_id", assignment.SubmissionID.String()),
		zap.String("language", assignment.Language),
		zap.Int("test_cases", len(assignment.TestCases)),
		zap.Int("time_limit_ms", assignment.TimeLimitMs),
		zap.Int("memory_limit_mb", assignment.MemoryLimitMb),
		zap.Time("lease_expires_at", assignment.LeaseExpiresAt),
	)

	// Start the job in its own goroutine.
	jobCtx, cancel := context.WithCancel(ctx)

	w.mu.Lock()
	w.activeJobs[assignment.JobID] = cancel
	activeCount = len(w.activeJobs)
	w.mu.Unlock()
	w.log.Info("job registered as active",
		zap.String("job_id", assignment.JobID.String()),
		zap.Int("active_jobs", activeCount),
		zap.Int("capacity", w.capacity),
	)

	go w.runJob(jobCtx, assignment, cancel)
}

// ---------------------------------------------------------------------------
// Job execution
// ---------------------------------------------------------------------------

// runJob executes a single job: marks it running, starts lease renewal,
// invokes the executor, and reports the result.
func (w *Worker) runJob(ctx context.Context, assignment *domain.Assignment, cancel context.CancelFunc) {
	jobID := assignment.JobID
	log := w.log.With(zap.String("job_id", jobID.String()))
	log = log.With(
		zap.String("submission_id", assignment.SubmissionID.String()),
		zap.String("language", assignment.Language),
	)

	defer func() {
		log.Info("job cleanup starting")
		cancel()
		w.mu.Lock()
		delete(w.activeJobs, assignment.JobID)
		activeCount := len(w.activeJobs)
		w.mu.Unlock()
		log.Info("job cleanup finished", zap.Int("active_jobs", activeCount))
	}()

	log.Info("job lifecycle started",
		zap.Int("test_cases", len(assignment.TestCases)),
		zap.Int("time_limit_ms", assignment.TimeLimitMs),
		zap.Int("memory_limit_mb", assignment.MemoryLimitMb),
		zap.Time("lease_expires_at", assignment.LeaseExpiresAt),
	)

	// 1. Tell the control plane we are now running.
	log.Info("marking job as running")
	if err := w.client.MarkRunning(ctx, jobID, w.id); err != nil {
		log.Error("failed to mark job as running", zap.Error(err))
		// If we can't transition to running, the lease will eventually expire
		// and the job will be requeued.
		return
	}
	log.Info("job marked as running")

	// 2. Start lease renewal in the background.
	leaseCtx, stopLease := context.WithCancel(ctx)
	defer stopLease()

	log.Info("starting lease renewal loop", zap.Duration("lease_renewal_interval", w.leaseRenewalInterval))
	go w.leaseRenewalLoop(leaseCtx, jobID, log)

	// 3. Execute the job.
	log.Info("calling executor")
	result, execErr := w.executor.Execute(ctx, assignment)
	if execErr != nil {
		log.Error("executor returned error", zap.Error(execErr))
	} else if result == nil {
		log.Error("executor returned nil result without error")
	} else {
		log.Info("executor returned result",
			zap.String("overall_verdict", result.OverallVerdict),
			zap.Int("testcase_results", len(result.TestcaseResults)),
			zap.Bool("has_compiler_output", result.CompilerOutput != nil),
		)
	}

	// Stop renewing the lease before we report — we are done.
	log.Info("stopping lease renewal loop before report")
	stopLease()

	// 4. Report the result.
	reportReq := buildReportRequest(w.id, result, execErr)
	log.Info("reporting job result",
		zap.String("overall_verdict", reportReq.OverallVerdict),
		zap.Int("testcase_results", len(reportReq.TestcaseResults)),
	)
	if err := w.client.ReportResult(ctx, jobID, reportReq); err != nil {
		log.Error("failed to report result", zap.Error(err))
		return
	}

	log.Info("job result reported")
	log.Info("job completed", zap.String("verdict", reportReq.OverallVerdict))
}

// buildReportRequest converts the executor's output (or error) into the
// domain.ReportResultRequest expected by the control plane.
func buildReportRequest(workerID string, result *ExecutionResult, execErr error) domain.ReportResultRequest {
	if execErr != nil {
		// Worker-side failure — report as InternalError so the control plane
		// can count it as a completed job with an error verdict.
		return domain.ReportResultRequest{
			WorkerID:       workerID,
			OverallVerdict: "InternalError",
		}
	}
	if result == nil {
		return domain.ReportResultRequest{
			WorkerID:       workerID,
			OverallVerdict: "InternalError",
		}
	}

	return domain.ReportResultRequest{
		WorkerID:        workerID,
		OverallVerdict:  result.OverallVerdict,
		TotalTimeMs:     result.TotalTimeMs,
		MaxMemoryKb:     result.MaxMemoryKb,
		WallTimeMs:      result.WallTimeMs,
		CompilerOutput:  result.CompilerOutput,
		TestcaseResults: result.TestcaseResults,
	}
}

// ---------------------------------------------------------------------------
// Lease renewal loop
// ---------------------------------------------------------------------------

func (w *Worker) leaseRenewalLoop(ctx context.Context, jobID uuid.UUID, log *zap.Logger) {
	ticker := time.NewTicker(w.leaseRenewalInterval)
	defer ticker.Stop()
	log.Info("lease renewal loop running")

	for {
		select {
		case <-ctx.Done():
			log.Info("lease renewal loop stopping", zap.Error(ctx.Err()))
			return
		case <-ticker.C:
			log.Info("renewing job lease")
			if err := w.client.RenewLease(ctx, jobID, w.id); err != nil {
				log.Warn("lease renewal failed", zap.Error(err))
				continue
			}
			log.Info("job lease renewed")
		}
	}
}
