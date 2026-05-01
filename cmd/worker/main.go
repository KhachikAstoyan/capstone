// cmd/worker is the execution worker binary.
//
// It connects to the Execution Control Plane, polls for jobs, and executes
// them using the configured executor backend (Firecracker in production,
// Docker in container environments, stub in development).
//
// Minimum environment:
//
//	WORKER_CP_URL=http://localhost:9090    # control plane address
//	WORKER_CP_KEY=<shared-secret>         # omit to skip auth (dev only)
//	WORKER_LANGUAGES=python,javascript,go # comma-separated
//	WORKER_CAPACITY=1                     # max concurrent jobs
//	WORKER_EXECUTOR=docker                # "firecracker" | "docker" | (stub fallback)
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

	cpClient := worker.NewControlPlaneClient(cfg.ControlPlaneURL, cfg.ControlPlaneKey)

	executor, err := buildExecutor(ctx, cfg, log)
	if err != nil {
		log.Fatal("failed to build executor", zap.Error(err))
	}

	w := worker.New(cfg, cpClient, executor, log)

	log.Info("starting worker",
		zap.String("worker_id", cfg.WorkerID),
		zap.String("languages", cfg.Languages),
		zap.Int("capacity", cfg.Capacity),
		zap.String("control_plane", cfg.ControlPlaneURL),
		zap.String("executor", cfg.Executor),
	)

	w.Run(ctx)
	log.Info("worker stopped")
}
