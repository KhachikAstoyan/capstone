package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	stats, err := h.service.GetUserStats(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "user not found")
		} else {
			common.RespondError(w, http.StatusInternalServerError, err, "failed to load user stats")
		}
		return
	}

	common.RespondJSON(w, http.StatusOK, stats)
}
