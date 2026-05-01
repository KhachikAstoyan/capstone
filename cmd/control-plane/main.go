// cmd/control-plane is the Execution Control Plane service.
//
// # Responsibilities
//
//   - Accept job creation requests from the main API service.
//   - Register workers and track their health via heartbeat calls.
//   - Assign queued jobs to workers using a worker-pull model with leases.
//   - Store execution results reported by workers.
//   - Run background sweeps to requeue expired leases and mark stale workers.
//
// # Architecture
//
// The control plane has its own PostgreSQL database (CP_DATABASE_URL).
// It does NOT share a database with the API service.  Communication between
// the two services is entirely over HTTP.
//
//	API service ──POST /v1/jobs──────────────► Control Plane
//	                                                │
//	Workers ──POST /v1/workers/heartbeat──────────► │
//	Workers ──POST /v1/workers/poll───────────────► │
//	Workers ──POST /v1/jobs/{id}/running──────────► │
//	Workers ──POST /v1/jobs/{id}/lease────────────► │
//	Workers ──POST /v1/jobs/{id}/result───────────► │
//
// # Authentication
//
// Every request must carry X-Internal-Key: <CP_INTERNAL_KEY>.
// If CP_INTERNAL_KEY is unset the check is skipped (development only).
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	controlplane "github.com/KhachikAstoyan/capstone/internal/controlplane"
	cphttp "github.com/KhachikAstoyan/capstone/internal/controlplane/http"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/repository"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/service"
	"github.com/KhachikAstoyan/capstone/pkg/database"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"github.com/KhachikAstoyan/capstone/pkg/migrations"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	cfg, err := controlplane.LoadConfig()
	if err != nil {
		// Logger not yet initialised — use a temporary one.
		tmp := logger.Init("production")
		tmp.Fatal("failed to load config", zap.Error(err))
	}

	log := logger.Init(cfg.Environment)
	defer log.Sync() //nolint:errcheck

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx = logger.WithLogger(ctx, log)

	log.Info("starting Execution Control Plane",
		zap.String("environment", cfg.Environment),
		zap.String("address", fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)),
	)

	// ── Database ──────────────────────────────────────────────────────────────
	db := database.MustConnect(ctx, database.Config{URL: cfg.DatabaseURL})
	defer db.Close()

	absPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		log.Fatal("failed to resolve migrations path", zap.Error(err))
	}
	log.Info("running control-plane migrations", zap.String("path", absPath))
	if err := migrations.RunMigrations(db, absPath); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}
	log.Info("migrations complete")

	// ── Repositories ──────────────────────────────────────────────────────────
	jobRepo := repository.NewJobRepository(db)
	workerRepo := repository.NewWorkerRepository(db)

	// ── Service ───────────────────────────────────────────────────────────────
	svc := service.New(jobRepo, workerRepo, service.Config{
		LeaseDuration:    cfg.LeaseDuration(),
		HeartbeatTimeout: cfg.HeartbeatTimeout(),
	}, log)

	// ── HTTP handler + router ─────────────────────────────────────────────────
	handler := cphttp.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Shared-secret authentication for all routes.
	if cfg.InternalKey != "" {
		r.Use(internalKeyMiddleware(cfg.InternalKey))
	} else {
		log.Warn("CP_INTERNAL_KEY is not set — authentication is DISABLED (development mode)")
	}

	r.Mount("/", setupRoutes(handler))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler: r,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// ── Background sweeps ─────────────────────────────────────────────────────
	go runLeaseExpirySweep(ctx, svc, cfg.LeaseCheckInterval(), log)
	go runWorkerHealthSweep(ctx, svc, cfg.WorkerSweepInterval(), log)

	// ── Start ─────────────────────────────────────────────────────────────────
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	log.Info("control plane ready",
		zap.String("address", fmt.Sprintf("http://%s:%d", cfg.ServerHost, cfg.ServerPort)),
	)

	<-ctx.Done()
	log.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("stopped")
}

// ---------------------------------------------------------------------------
// Background goroutines
// ---------------------------------------------------------------------------

// runLeaseExpirySweep periodically requeues jobs whose leases have expired.
// This is the primary recovery mechanism for crashed or stalled workers.
func runLeaseExpirySweep(ctx context.Context, svc service.Service, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := svc.RequeueExpiredLeases(ctx); err != nil {
				log.Error("lease expiry sweep failed", zap.Error(err))
			}
		}
	}
}

// runWorkerHealthSweep periodically marks workers offline when they have
// missed their heartbeat deadline.
func runWorkerHealthSweep(ctx context.Context, svc service.Service, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := svc.MarkStaleWorkers(ctx); err != nil {
				log.Error("worker health sweep failed", zap.Error(err))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

// internalKeyMiddleware rejects requests that do not present the correct
// X-Internal-Key header.  This is a lightweight shared-secret guard suitable
// for internal service-to-service communication on a private network.
func internalKeyMiddleware(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Key") != expectedKey {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
