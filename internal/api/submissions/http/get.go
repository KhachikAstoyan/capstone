package http

import (
	"errors"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/service"
	"github.com/KhachikAstoyan/capstone/pkg/permissions"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	isAdmin := rbac.CheckPermissionInContext(r.Context(), permissions.SubmissionsViewAll)

	sub, err := h.service.GetSubmission(r.Context(), id, userID, isAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSubmissionNotFound):
			common.RespondSimpleError(w, http.StatusNotFound, "submission not found")
		case errors.Is(err, service.ErrForbidden):
			common.RespondSimpleError(w, http.StatusForbidden, "access denied")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to get submission")
		}
		return
	}

	common.RespondJSON(w, http.StatusOK, sub)
}
