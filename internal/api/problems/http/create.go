package http

import (
	"encoding/json"
	"errors"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
)

func (h *Handler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.CreateProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	problem, err := h.service.CreateProblem(r.Context(), req, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			common.RespondError(w, http.StatusBadRequest, err, "invalid input")
		case errors.Is(err, service.ErrProblemSlugConflict):
			common.RespondError(w, http.StatusConflict, err, "problem with this slug already exists")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to create problem")
		}
		return
	}

	common.RespondJSON(w, http.StatusCreated, problem)
}
