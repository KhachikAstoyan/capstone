//go:build !linux

package main

import (
	"context"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/worker"
	"go.uber.org/zap"
)

func buildExecutor(_ context.Context, cfg *worker.Config, log *zap.Logger) (worker.Executor, error) {
	dockerExec, err := worker.NewDockerExecutor(worker.DefaultLanguages, cfg.DockerRuntime, log)
	if err != nil {
		if !cfg.AllowStubExecutor {
			return nil, fmt.Errorf("docker executor unavailable: %w", err)
		}
		log.Warn("docker executor unavailable, falling back to stub", zap.Error(err))
		return worker.NewStubExecutor(log), nil
	}
	log.Info("docker executor ready")
	return dockerExec, nil
}
