package http

import (
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := getRefreshTokenFromCookie(r)
	if refreshToken == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to logout")
		return
	}

	h.clearRefreshTokenCookie(w)

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}
