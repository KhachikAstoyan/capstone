package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	err = h.service.DeleteProblem(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to delete problem")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "problem deleted successfully"})
}
