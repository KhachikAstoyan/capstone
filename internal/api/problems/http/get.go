package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
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

	problem, err := h.service.GetProblem(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get problem")
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

	problem, err := h.service.GetProblemBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get problem")
		return
	}

	common.RespondJSON(w, http.StatusOK, problem)
}
