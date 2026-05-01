// Package controlplane holds shared configuration and bootstrapping helpers
// for the Execution Control Plane service.
package controlplane

import (
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/config"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the control-plane binary.
// Values are read from environment variables at startup via envconfig.
type Config struct {
	config.CommonConfig

	// ── HTTP server ───────────────────────────────────────────────────────────
	ServerHost string `envconfig:"CP_HOST" default:"0.0.0.0"`
	ServerPort int    `envconfig:"CP_PORT" default:"9090"`

	// ── Database ─────────────────────────────────────────────────────────────
	// This is the control plane's own PostgreSQL database — separate from the
	// main API database.
	DatabaseURL    string `envconfig:"CP_DATABASE_URL"    required:"true"`
	MigrationsPath string `envconfig:"CP_MIGRATIONS_PATH" default:"./internal/controlplane/migrations"`

	// ── Security ─────────────────────────────────────────────────────────────
	// Shared secret that all callers (API service, workers) must present in the
	// X-Internal-Key request header.  If empty, authentication is skipped
	// (development mode only — never leave this empty in production).
	InternalKey string `envconfig:"CP_INTERNAL_KEY"`

	// ── Lease / scheduling ───────────────────────────────────────────────────
	// How long a worker has to complete (or renew) a job before it is requeued.
	LeaseDurationSec int `envconfig:"CP_LEASE_DURATION_SEC" default:"60"`

	// How often the background goroutine checks for expired leases.
	LeaseCheckIntervalSec int `envconfig:"CP_LEASE_CHECK_INTERVAL_SEC" default:"10"`

	// How long a worker can go without sending a heartbeat before it is marked
	// offline.
	HeartbeatTimeoutSec int `envconfig:"CP_HEARTBEAT_TIMEOUT_SEC" default:"30"`

	// How often the background goroutine sweeps for stale workers.
	WorkerSweepIntervalSec int `envconfig:"CP_WORKER_SWEEP_INTERVAL_SEC" default:"15"`
}

// LoadConfig reads the control-plane configuration from environment variables.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load control-plane config: %w", err)
	}
	return &cfg, nil
}

// LeaseDuration converts LeaseDurationSec to a time.Duration.
func (c *Config) LeaseDuration() time.Duration {
	return time.Duration(c.LeaseDurationSec) * time.Second
}

// LeaseCheckInterval converts LeaseCheckIntervalSec to a time.Duration.
func (c *Config) LeaseCheckInterval() time.Duration {
	return time.Duration(c.LeaseCheckIntervalSec) * time.Second
}

// HeartbeatTimeout converts HeartbeatTimeoutSec to a time.Duration.
func (c *Config) HeartbeatTimeout() time.Duration {
	return time.Duration(c.HeartbeatTimeoutSec) * time.Second
}

// WorkerSweepInterval converts WorkerSweepIntervalSec to a time.Duration.
func (c *Config) WorkerSweepInterval() time.Duration {
	return time.Duration(c.WorkerSweepIntervalSec) * time.Second
}
