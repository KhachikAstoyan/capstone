package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	var req domain.UpdateProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	problem, err := h.service.UpdateProblem(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProblemNotFound):
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
		case errors.Is(err, service.ErrInvalidInput):
			common.RespondError(w, http.StatusBadRequest, err, "invalid input")
		case errors.Is(err, service.ErrProblemSlugConflict):
			common.RespondError(w, http.StatusConflict, err, "problem with this slug already exists")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to update problem")
		}
		return
	}

	common.RespondJSON(w, http.StatusOK, problem)
}
