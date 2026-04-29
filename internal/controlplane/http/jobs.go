package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// POST /v1/jobs
// ---------------------------------------------------------------------------

// CreateJob accepts a job creation request from the API service.
//
// Request body: domain.CreateJobRequest (JSON)
// Response:     201 Created with the new domain.Job
//
// The API service should call this endpoint after persisting the submission
// in its own database.  The control plane stores the job independently and
// returns a job_id that the API can use to poll for status.
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateJobRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	job, err := h.svc.CreateJob(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to create job")
		}
		return
	}

	respondJSON(w, http.StatusCreated, job)
}

// ---------------------------------------------------------------------------
// GET /v1/jobs/{jobID}
// ---------------------------------------------------------------------------

// GetJob returns the current lifecycle state of a job.
//
// Path param: jobID (UUID)
// Response:   200 OK with domain.Job
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r, "jobID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.svc.GetJob(r.Context(), jobID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobNotFound):
			respondError(w, http.StatusNotFound, "job not found")
		default:
			respondError(w, http.StatusInternalServerError, "failed to get job")
		}
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// ---------------------------------------------------------------------------
// GET /v1/jobs/by-submission/{submissionID}
// ---------------------------------------------------------------------------

// GetJobBySubmission looks up the most-recent job for a given submission id.
// The API service uses this to surface execution status to end users without
// having to store the job_id itself.
//
// Path param: submissionID (UUID)
// Response:   200 OK with domain.Job
func (h *Handler) GetJobBySubmission(w http.ResponseWriter, r *http.Request) {
	submissionID, err := parseUUID(r, "submissionID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	job, err := h.svc.GetJobBySubmission(r.Context(), submissionID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobNotFound):
			respondError(w, http.StatusNotFound, "job not found for submission")
		default:
			respondError(w, http.StatusInternalServerError, "failed to get job")
		}
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// ---------------------------------------------------------------------------
// GET /v1/jobs/{jobID}/result
// ---------------------------------------------------------------------------

// GetJobResult returns the stored execution result for a completed job.
//
// Path param: jobID (UUID)
// Response:   200 OK with domain.JobResult
//             404 if the job has not completed yet or does not exist
func (h *Handler) GetJobResult(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r, "jobID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	result, err := h.svc.GetJobResult(r.Context(), jobID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobNotFound):
			respondError(w, http.StatusNotFound, "result not found (job may not be complete)")
		default:
			respondError(w, http.StatusInternalServerError, "failed to get result")
		}
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}
