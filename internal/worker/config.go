// Package worker implements the execution worker process.
//
// A worker registers with the Execution Control Plane, polls for jobs, runs
// user code in a sandbox, and reports results back.
package worker

import (
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/config"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the worker binary.
type Config struct {
	config.CommonConfig

	// ── Identity ────────────────────────────────────────────────────────────
	// Stable identifier for this worker.  If empty, a random UUID is
	// generated at startup.  Re-using the same ID across restarts lets the
	// control plane reconcile the row instead of creating a new one.
	WorkerID string `envconfig:"WORKER_ID"`

	// Languages this worker can execute (e.g. "python,go,cpp").
	// Comma-separated; parsed into a []string at startup.
	Languages string `envconfig:"WORKER_LANGUAGES" default:"python,javascript,go"`

	// Maximum number of jobs this worker will run concurrently.
	Capacity int `envconfig:"WORKER_CAPACITY" default:"1"`

	// ── Control Plane ────────────────────────────────────────────────────────
	// Base URL of the Execution Control Plane (no trailing slash).
	ControlPlaneURL string `envconfig:"WORKER_CP_URL" required:"true"`

	// Shared secret sent in X-Internal-Key header.
	ControlPlaneKey string `envconfig:"WORKER_CP_KEY"`

	// ── Intervals ────────────────────────────────────────────────────────────
	// How often the worker sends heartbeats to the control plane.
	HeartbeatIntervalSec int `envconfig:"WORKER_HEARTBEAT_INTERVAL_SEC" default:"10"`

	// How often the worker polls for new jobs when it has free capacity.
	PollIntervalSec int `envconfig:"WORKER_POLL_INTERVAL_SEC" default:"2"`

	// How often the worker renews a lease while executing a job.
	LeaseRenewalIntervalSec int `envconfig:"WORKER_LEASE_RENEWAL_INTERVAL_SEC" default:"20"`

	// AllowStubExecutor permits falling back to the in-memory stub executor
	// when Docker is unavailable. This should stay false outside local/dev runs.
	AllowStubExecutor bool `envconfig:"WORKER_ALLOW_STUB_EXECUTOR" default:"false"`
}

// LoadConfig loads the worker configuration from environment variables.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load worker config: %w", err)
	}
	return &cfg, nil
}

// LanguageList splits the comma-separated Languages string into a slice.
func (c *Config) LanguageList() []string {
	if c.Languages == "" {
		return nil
	}
	var out []string
	for _, s := range splitAndTrim(c.Languages) {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (c *Config) HeartbeatInterval() time.Duration {
	return time.Duration(c.HeartbeatIntervalSec) * time.Second
}

func (c *Config) PollInterval() time.Duration {
	return time.Duration(c.PollIntervalSec) * time.Second
}

func (c *Config) LeaseRenewalInterval() time.Duration {
	return time.Duration(c.LeaseRenewalIntervalSec) * time.Second
}

// splitAndTrim splits s on commas and trims whitespace from each element.
func splitAndTrim(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			// trim spaces
			lo, hi := 0, len(part)
			for lo < hi && part[lo] == ' ' {
				lo++
			}
			for hi > lo && part[hi-1] == ' ' {
				hi--
			}
			out = append(out, part[lo:hi])
			start = i + 1
		}
	}
	return out
}
