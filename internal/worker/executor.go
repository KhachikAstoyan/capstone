package worker

import (
	"context"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"go.uber.org/zap"
)

// ExecutionResult holds the outcome of executing a single job.
// This maps directly onto the fields of domain.ReportResultRequest.
type ExecutionResult struct {
	OverallVerdict  string
	TotalTimeMs     *int
	MaxMemoryKb     *int
	WallTimeMs      *int
	CompilerOutput  *string
	TestcaseResults []domain.TestcaseResultInput
}

// Executor is the interface that job execution backends must implement.
//
// The initial implementation is a no-op stub.  The real implementation will
// launch a Firecracker microVM, stream the source code into it, compile/run
// with the problem's time and memory limits, capture output, and compare
// against expected results.
type Executor interface {
	// Execute runs the job described by the assignment and returns a result.
	// The context is cancelled if the lease cannot be renewed or the worker
	// is shutting down.
	Execute(ctx context.Context, assignment *domain.Assignment) (*ExecutionResult, error)
}

// ---------------------------------------------------------------------------
// StubExecutor — used during development / integration testing.
// ---------------------------------------------------------------------------

// StubExecutor immediately returns an "Accepted" verdict for every job.
// It exists so the worker binary can be run end-to-end against a live
// control plane without needing Firecracker.
type StubExecutor struct {
	log *zap.Logger
}

func NewStubExecutor(log *zap.Logger) StubExecutor {
	if log == nil {
		log = zap.NewNop()
	}
	return StubExecutor{log: log}
}

func (s StubExecutor) Execute(_ context.Context, a *domain.Assignment) (*ExecutionResult, error) {
	s.log.Warn("stub executor executing job; no docker container will be created",
		zap.String("job_id", a.JobID.String()),
		zap.String("submission_id", a.SubmissionID.String()),
		zap.String("language", a.Language),
		zap.Int("test_cases", len(a.TestCases)),
	)
	zero := 0
	return &ExecutionResult{
		OverallVerdict: "Accepted",
		TotalTimeMs:    &zero,
		MaxMemoryKb:    &zero,
	}, nil
}
