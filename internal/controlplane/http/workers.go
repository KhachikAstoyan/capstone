package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/service"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// POST /v1/workers/heartbeat
// ---------------------------------------------------------------------------

// Heartbeat registers a new worker or refreshes the state of an existing one.
//
// Workers must call this endpoint at least once per HeartbeatTimeout
// (default 30 s) or they will be marked offline and stop receiving jobs.
//
// Request body: domain.HeartbeatRequest (JSON)
// Response:     200 OK with the updated domain.Worker
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req domain.HeartbeatRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	worker, err := h.svc.Heartbeat(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "heartbeat failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, worker)
}

// ---------------------------------------------------------------------------
// POST /v1/workers/poll
// ---------------------------------------------------------------------------

// PollJob attempts to claim the next available job for the requesting worker.
//
// The control plane uses a worker-pull model: workers ask for work rather than
// being pushed jobs.  This keeps the control plane stateless with respect to
// worker connections.
//
// The response includes a lease_expires_at timestamp.  The worker must either
// renew the lease (POST /v1/jobs/{jobID}/lease) or finish and report results
// before that time, or the job will be requeued by the background sweep.
//
// Request body: domain.PollRequest (JSON)
// Response:     200 OK with domain.Assignment
//               204 No Content when no jobs are available
func (h *Handler) PollJob(w http.ResponseWriter, r *http.Request) {
	var req domain.PollRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assignment, err := h.svc.PollJob(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoJobAvailable):
			// 204 is the normal "nothing to do" response; workers should back off
			// and retry after a short delay.
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "poll failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, assignment)
}

// ---------------------------------------------------------------------------
// POST /v1/jobs/{jobID}/running
// ---------------------------------------------------------------------------

// MarkRunning transitions a job from 'assigned' to 'running'.
//
// Workers call this once they have started executing the user's code.
// This is optional but recommended; it improves observability and allows the
// control plane to distinguish "waiting to start" from "actively executing".
//
// Request body: { "worker_id": "..." }
// Response:     204 No Content
func (h *Handler) MarkRunning(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r, "jobID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var body struct {
		WorkerID string `json:"worker_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.MarkRunning(r.Context(), jobID, body.WorkerID); err != nil {
		switch {
		case errors.Is(err, service.ErrLeaseMismatch):
			respondError(w, http.StatusConflict, "worker does not hold the lease for this job")
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to mark job as running")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /v1/jobs/{jobID}/lease
// ---------------------------------------------------------------------------

// RenewLease extends the worker's hold on a job by another LeaseDuration.
//
// Workers should call this periodically (e.g. every 30 s when LeaseDuration
// is 60 s) to avoid having the job requeued while still executing.
//
// Request body: domain.RenewLeaseRequest (JSON)
// Response:     204 No Content
func (h *Handler) RenewLease(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r, "jobID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req domain.RenewLeaseRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.RenewLease(r.Context(), jobID, req); err != nil {
		switch {
		case errors.Is(err, service.ErrLeaseMismatch):
			respondError(w, http.StatusConflict, "worker does not hold the lease for this job")
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to renew lease")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /v1/jobs/{jobID}/result
// ---------------------------------------------------------------------------

// ReportResult records the execution outcome and closes out the job.
//
// The job transitions to 'completed'.  The worker should call this endpoint
// exactly once, after all testcases have been evaluated.
//
// Request body: domain.ReportResultRequest (JSON)
// Response:     204 No Content
func (h *Handler) ReportResult(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r, "jobID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req domain.ReportResultRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ReportResult(r.Context(), jobID, req); err != nil {
		switch {
		case errors.Is(err, service.ErrLeaseMismatch):
			// The lease may have expired; the job could already have been requeued.
			respondError(w, http.StatusConflict, "worker does not hold the lease for this job")
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to report result")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GET /v1/workers/{workerID}  (diagnostic / admin)
// ---------------------------------------------------------------------------

// GetWorker returns the stored state for a single worker.  Intended for
// operational dashboards and debugging.
//
// Path param: workerID (string)
// Response:   200 OK with domain.Worker
func (h *Handler) GetWorker(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "workerID")
	if workerID == "" {
		respondError(w, http.StatusBadRequest, "worker id is required")
		return
	}
	respondError(w, http.StatusNotImplemented, "not yet implemented")
}
