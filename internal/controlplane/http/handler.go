// Package http contains the HTTP layer for the Execution Control Plane.
//
// Endpoints are split into two groups:
//
//   API-facing  — called by the main API service to submit jobs and query status
//   Worker-facing — called by execution workers to register, poll, and report
//
// Authentication is handled by a simple shared-secret middleware (see
// middleware.go).  Both caller types use the same key; a future iteration
// could issue separate keys per caller type.
package http

import (
	"encoding/json"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/service"
)

// Handler holds all HTTP handlers for the control plane.
// It is constructed once and mounted into the router.
type Handler struct {
	svc service.Service
}

// NewHandler creates a Handler.
func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// ---------------------------------------------------------------------------
// Shared response helpers (local copies so this package has no dependency on
// the main API's common package)
// ---------------------------------------------------------------------------

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, errorResponse{Error: msg})
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
