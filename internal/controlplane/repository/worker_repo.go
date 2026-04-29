package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/lib/pq"
)

// WorkerRepository defines all database operations on workers.
type WorkerRepository interface {
	// Upsert inserts a new worker row or updates an existing one.
	// Called on every heartbeat so the row always reflects current state.
	Upsert(ctx context.Context, req domain.HeartbeatRequest) (*domain.Worker, error)

	// GetByID returns a single worker by its id.
	GetByID(ctx context.Context, id string) (*domain.Worker, error)

	// ListHealthy returns all workers in 'healthy' status.
	// Optionally filtered to those that support at least one of the given languages.
	ListHealthy(ctx context.Context, languages []string) ([]*domain.Worker, error)

	// IncrementActiveJobs atomically adds delta (positive or negative) to
	// active_jobs.  Called by the job assignment path.
	IncrementActiveJobs(ctx context.Context, workerID string, delta int) error

	// MarkStaleOffline sets health_status = 'offline' for workers whose
	// last_heartbeat is older than the given threshold.
	// Returns the number of workers affected.
	MarkStaleOffline(ctx context.Context, threshold time.Time) (int, error)
}

type workerRepository struct {
	db *sql.DB
}

// NewWorkerRepository creates a WorkerRepository backed by a *sql.DB.
func NewWorkerRepository(db *sql.DB) WorkerRepository {
	return &workerRepository{db: db}
}

// ---------------------------------------------------------------------------
// Upsert
// ---------------------------------------------------------------------------

func (r *workerRepository) Upsert(ctx context.Context, req domain.HeartbeatRequest) (*domain.Worker, error) {
	// ON CONFLICT updates every mutable field so the row is always current.
	const q = `
		INSERT INTO workers (id, languages, capacity, active_jobs, health_status, last_heartbeat)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE SET
			languages      = EXCLUDED.languages,
			capacity       = EXCLUDED.capacity,
			active_jobs    = EXCLUDED.active_jobs,
			health_status  = EXCLUDED.health_status,
			last_heartbeat = NOW()
		RETURNING id, languages, capacity, active_jobs, health_status, last_heartbeat, registered_at
	`
	row := r.db.QueryRowContext(ctx, q,
		req.WorkerID,
		pq.Array(req.Languages),
		req.Capacity,
		req.ActiveJobs,
		req.HealthStatus,
	)
	return scanWorker(row)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func (r *workerRepository) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	const q = `
		SELECT id, languages, capacity, active_jobs, health_status, last_heartbeat, registered_at
		FROM workers WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	w, err := scanWorker(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("worker %q not found", id)
	}
	return w, err
}

// ---------------------------------------------------------------------------
// ListHealthy
// ---------------------------------------------------------------------------

func (r *workerRepository) ListHealthy(ctx context.Context, languages []string) ([]*domain.Worker, error) {
	// When languages is non-empty, restrict to workers that support at least
	// one of the requested languages using the && (overlap) array operator.
	var (
		rows *sql.Rows
		err  error
	)

	if len(languages) == 0 {
		const q = `
			SELECT id, languages, capacity, active_jobs, health_status, last_heartbeat, registered_at
			FROM workers
			WHERE health_status = 'healthy'
			ORDER BY active_jobs ASC
		`
		rows, err = r.db.QueryContext(ctx, q)
	} else {
		const q = `
			SELECT id, languages, capacity, active_jobs, health_status, last_heartbeat, registered_at
			FROM workers
			WHERE health_status = 'healthy'
			  AND languages && $1
			ORDER BY active_jobs ASC
		`
		rows, err = r.db.QueryContext(ctx, q, pq.Array(languages))
	}

	if err != nil {
		return nil, fmt.Errorf("list healthy workers: %w", err)
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		w, err := scanWorkerRow(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter workers: %w", err)
	}
	return workers, nil
}

// ---------------------------------------------------------------------------
// IncrementActiveJobs
// ---------------------------------------------------------------------------

func (r *workerRepository) IncrementActiveJobs(ctx context.Context, workerID string, delta int) error {
	const q = `
		UPDATE workers
		SET active_jobs = GREATEST(0, active_jobs + $2)
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, q, workerID, delta)
	if err != nil {
		return fmt.Errorf("increment active_jobs for worker %q: %w", workerID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// MarkStaleOffline
// ---------------------------------------------------------------------------

func (r *workerRepository) MarkStaleOffline(ctx context.Context, threshold time.Time) (int, error) {
	const q = `
		UPDATE workers
		SET health_status = 'offline'
		WHERE last_heartbeat < $1
		  AND health_status != 'offline'
	`
	res, err := r.db.ExecContext(ctx, q, threshold)
	if err != nil {
		return 0, fmt.Errorf("mark stale workers offline: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// ---------------------------------------------------------------------------
// scanWorker / scanWorkerRow — helpers
// ---------------------------------------------------------------------------

func scanWorker(row *sql.Row) (*domain.Worker, error) {
	w := &domain.Worker{}
	err := row.Scan(
		&w.ID,
		pq.Array(&w.Languages),
		&w.Capacity,
		&w.ActiveJobs,
		&w.HealthStatus,
		&w.LastHeartbeat,
		&w.RegisteredAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan worker: %w", err)
	}
	return w, nil
}

func scanWorkerRow(rows *sql.Rows) (*domain.Worker, error) {
	w := &domain.Worker{}
	err := rows.Scan(
		&w.ID,
		pq.Array(&w.Languages),
		&w.Capacity,
		&w.ActiveJobs,
		&w.HealthStatus,
		&w.LastHeartbeat,
		&w.RegisteredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan worker row: %w", err)
	}
	return w, nil
}
