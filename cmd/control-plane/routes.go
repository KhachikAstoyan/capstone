package main

import (
	"net/http"

	cphttp "github.com/KhachikAstoyan/capstone/internal/controlplane/http"
	"github.com/go-chi/chi/v5"
)

// setupRoutes mounts all control-plane endpoints onto a new Chi router and
// returns it as an http.Handler.
//
// Route groups
// ─────────────
//   /v1/jobs         — API-service-facing: submit and query jobs
//   /v1/workers      — Worker-facing: heartbeat and poll
//   /v1/jobs/:id/…   — Worker-facing: lease management and result reporting
//
// All routes sit behind the internalKeyMiddleware (see main.go) so only
// callers that present the correct X-Internal-Key header are accepted.
func setupRoutes(h *cphttp.Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/v1", func(r chi.Router) {

		// ── Job endpoints (called by the main API service) ──────────────────
		r.Route("/jobs", func(r chi.Router) {
			// Submit a new execution job.
			r.Post("/", h.CreateJob)

			// Look up a job by its control-plane id.
			r.Get("/{jobID}", h.GetJob)

			// Look up the latest job for a given submission id.
			// The API service uses this to avoid storing job ids itself.
			r.Get("/by-submission/{submissionID}", h.GetJobBySubmission)

			// Fetch the stored execution result (only available once completed).
			r.Get("/{jobID}/result", h.GetJobResult)

			// ── Job lifecycle endpoints (called by workers) ────────────────
			// Transition assigned → running.
			r.Post("/{jobID}/running", h.MarkRunning)

			// Extend the lease.
			r.Post("/{jobID}/lease", h.RenewLease)

			// Report the final execution result.
			r.Post("/{jobID}/result", h.ReportResult)
		})

		// ── Worker endpoints ─────────────────────────────────────────────────
		r.Route("/workers", func(r chi.Router) {
			// Register or refresh worker state (must be called periodically).
			r.Post("/heartbeat", h.Heartbeat)

			// Claim the next available job.
			r.Post("/poll", h.PollJob)

			// Diagnostic: inspect a single worker's stored state.
			r.Get("/{workerID}", h.GetWorker)
		})
	})

	return r
}
