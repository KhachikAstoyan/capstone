package http

import (
	"encoding/json"
	"errors"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	problemIDStr := chi.URLParam(r, "problemID")
	problemID, err := uuid.Parse(problemIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	var req domain.CreateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.service.Run(r.Context(), userID, problemID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			common.RespondError(w, http.StatusBadRequest, err, "invalid input")
		case errors.Is(err, service.ErrProblemNotFound):
			common.RespondSimpleError(w, http.StatusNotFound, "problem not found")
		case errors.Is(err, service.ErrLanguageNotFound),
			errors.Is(err, service.ErrLanguageNotAllowed),
			errors.Is(err, service.ErrNoTestCases):
			common.RespondError(w, http.StatusUnprocessableEntity, err, err.Error())
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to create run")
		}
		return
	}

	common.RespondJSON(w, http.StatusCreated, sub)
}
