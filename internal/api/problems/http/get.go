package http

import (
	"errors"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/access"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) GetProblem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	var userID *uuid.UUID
	if uid, ok := authhttp.GetUserIDFromContext(r.Context()); ok {
		userID = &uid
	}

	problem, err := h.service.GetProblem(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get problem")
		return
	}

	if !access.IsPublished(problem) && !access.CanViewUnpublishedProblems(r.Context()) {
		common.RespondError(w, http.StatusNotFound, service.ErrProblemNotFound, "problem not found")
		return
	}

	common.RespondJSON(w, http.StatusOK, problem)
}

func (h *Handler) GetProblemBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "slug is required")
		return
	}

	var userID *uuid.UUID
	if uid, ok := authhttp.GetUserIDFromContext(r.Context()); ok {
		userID = &uid
	}

	problem, err := h.service.GetProblemBySlug(r.Context(), slug, userID)
	if err != nil {
		if errors.Is(err, service.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get problem")
		return
	}

	if !access.IsPublished(problem) && !access.CanViewUnpublishedProblems(r.Context()) {
		common.RespondError(w, http.StatusNotFound, service.ErrProblemNotFound, "problem not found")
		return
	}

	common.RespondJSON(w, http.StatusOK, problem)
}
