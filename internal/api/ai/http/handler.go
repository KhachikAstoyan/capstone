package http

import (
	"encoding/json"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/ai/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/ai/service"
	"github.com/google/uuid"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

type ValidateCodeRequest struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	ProblemID    string `json:"problem_id"`
	Code         string `json:"code"`
	LanguageID   string `json:"language_id"`
	LanguageKey  string `json:"language_key"`
}

type ValidationResponse struct {
	ID        uuid.UUID                 `json:"id"`
	IsAllowed bool                      `json:"is_allowed"`
	Severity  domain.ValidationSeverity `json:"severity"`
	Reason    *string                   `json:"reason"`
	CreatedAt string                    `json:"created_at"`
}

func (h *Handler) ValidateCode(w http.ResponseWriter, r *http.Request) {
	var req ValidateCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	submissionID, err := uuid.Parse(req.SubmissionID)
	if err != nil {
		http.Error(w, "Invalid submission_id", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	problemID, err := uuid.Parse(req.ProblemID)
	if err != nil {
		http.Error(w, "Invalid problem_id", http.StatusBadRequest)
		return
	}

	languageID, err := uuid.Parse(req.LanguageID)
	if err != nil {
		http.Error(w, "Invalid language_id", http.StatusBadRequest)
		return
	}

	validation, err := h.svc.ValidateCodeSubmission(r.Context(), domain.ValidateCodeRequest{
		SubmissionID: submissionID,
		UserID:       userID,
		ProblemID:    problemID,
		Code:         req.Code,
		LanguageID:   languageID,
		LanguageKey:  req.LanguageKey,
	})
	if err != nil {
		http.Error(w, "Failed to validate code", http.StatusInternalServerError)
		return
	}

	resp := ValidationResponse{
		ID:        validation.ID,
		IsAllowed: validation.IsAllowed,
		CreatedAt: validation.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if validation.Severity != nil {
		resp.Severity = *validation.Severity
	}
	if validation.Reason != nil {
		resp.Reason = validation.Reason
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetValidation(w http.ResponseWriter, r *http.Request) {
	submissionID := r.URL.Query().Get("submission_id")
	if submissionID == "" {
		http.Error(w, "submission_id query parameter required", http.StatusBadRequest)
		return
	}

	validation, err := h.svc.GetValidationBySubmission(r.Context(), submissionID)
	if err != nil {
		http.Error(w, "Failed to get validation", http.StatusInternalServerError)
		return
	}

	if validation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp := ValidationResponse{
		ID:        validation.ID,
		IsAllowed: validation.IsAllowed,
		CreatedAt: validation.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if validation.Severity != nil {
		resp.Severity = *validation.Severity
	}
	if validation.Reason != nil {
		resp.Reason = validation.Reason
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
