//go:build linux

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/KhachikAstoyan/capstone/internal/worker"
	"go.uber.org/zap"
)

func buildExecutor(ctx context.Context, cfg *worker.Config, log *zap.Logger) (worker.Executor, error) {
	switch strings.ToLower(cfg.Executor) {
	case "firecracker":
		return buildFirecrackerExecutor(ctx, cfg, log)
	default:
		return buildDockerExecutor(cfg, log)
	}
}

func buildFirecrackerExecutor(_ context.Context, cfg *worker.Config, log *zap.Logger) (worker.Executor, error) {
	fcCfg := worker.FCConfig{
		FirecrackerBin: cfg.FCBin,
		JailerBin:      cfg.JailerBin,
		KernelPath:     cfg.FCKernel,
		ChrootBase:     cfg.FCChrootBase,
		SnapshotsDir:   cfg.FCSnapshotsDir,
		JailerUID:      cfg.FCJailerUID,
		JailerGID:      cfg.FCJailerGID,
		VCPU:           cfg.FCVCPU,
		MemMB:          cfg.FCMemMB,
	}

	// Build rootfs map: language → /path/to/{language}.ext4
	rootfsByLang := make(map[string]string)
	for _, lang := range cfg.LanguageList() {
		rootfsByLang[lang] = filepath.Join(cfg.FCRootfsDir, lang+".ext4")
	}

	exec, err := worker.NewFirecrackerExecutor(fcCfg, worker.DefaultLanguages, rootfsByLang, log)
	if err != nil {
		return nil, fmt.Errorf("firecracker executor: %w", err)
	}
	log.Info("firecracker executor ready")
	return exec, nil
}

func buildDockerExecutor(cfg *worker.Config, log *zap.Logger) (worker.Executor, error) {
	dockerExec, err := worker.NewDockerExecutor(worker.DefaultLanguages, cfg.DockerRuntime, log)
	if err != nil {
		if !cfg.AllowStubExecutor {
			return nil, fmt.Errorf("docker executor unavailable (set WORKER_ALLOW_STUB_EXECUTOR=true for stub fallback): %w", err)
		}
		log.Warn("docker executor unavailable, falling back to stub", zap.Error(err))
		return worker.NewStubExecutor(log), nil
	}
	log.Info("docker executor ready")
	return dockerExec, nil
}
