// cmd/worker is the execution worker binary.
//
// It connects to the Execution Control Plane, polls for jobs, and executes
// them using the configured executor backend (Firecracker in production,
// stub in development).
//
// Minimum environment:
//
//	WORKER_CP_URL=http://localhost:9090    # control plane address
//	WORKER_CP_KEY=<shared-secret>         # omit to skip auth (dev only)
//	WORKER_LANGUAGES=python,javascript,go # comma-separated
//	WORKER_CAPACITY=1                     # max concurrent jobs
//	WORKER_ALLOW_STUB_EXECUTOR=false      # true only for local/dev fallback
package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/KhachikAstoyan/capstone/internal/worker"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := worker.LoadConfig()
	if err != nil {
		tmp := logger.Init("production")
		tmp.Fatal("failed to load config", zap.Error(err))
	}

	log := logger.Init(cfg.Environment)
	defer log.Sync() //nolint:errcheck

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Build the control-plane client.
	cpClient := worker.NewControlPlaneClient(cfg.ControlPlaneURL, cfg.ControlPlaneKey)

	// Build the Docker executor with the default language set.
	// Stub fallback is opt-in only; production should fail fast if Docker is
	// unavailable so jobs do not silently bypass sandbox execution.
	var executor worker.Executor
	dockerExec, err := worker.NewDockerExecutor(worker.DefaultLanguages, log)
	if err != nil {
		if !cfg.AllowStubExecutor {
			log.Fatal("docker executor unavailable and stub fallback is disabled",
				zap.Error(err),
				zap.Bool("allow_stub_executor", cfg.AllowStubExecutor),
			)
		}
		log.Warn("docker executor unavailable, using explicit stub executor fallback",
			zap.Error(err),
			zap.Bool("allow_stub_executor", cfg.AllowStubExecutor),
		)
		executor = worker.NewStubExecutor(log)
	} else {
		executor = dockerExec
		log.Info("docker executor ready")
	}

	w := worker.New(cfg, cpClient, executor, log)

	log.Info("starting worker",
		zap.String("worker_id", cfg.WorkerID),
		zap.String("languages", cfg.Languages),
		zap.Int("capacity", cfg.Capacity),
		zap.String("control_plane", cfg.ControlPlaneURL),
		zap.Bool("allow_stub_executor", cfg.AllowStubExecutor),
	)

	// Run blocks until ctx is cancelled (SIGINT/SIGTERM).
	w.Run(ctx)

	log.Info("worker stopped")
}
