package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := getRefreshTokenFromCookie(r)
	if refreshToken == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	response, err := h.service.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRefreshToken):
			common.RespondError(w, http.StatusUnauthorized, err, "invalid refresh token")
		case errors.Is(err, service.ErrRevokedRefreshToken):
			common.RespondError(w, http.StatusUnauthorized, err, "refresh token has been revoked")
		case errors.Is(err, service.ErrExpiredRefreshToken):
			common.RespondError(w, http.StatusUnauthorized, err, "refresh token has expired")
		case errors.Is(err, service.ErrUserBanned):
			common.RespondError(w, http.StatusForbidden, err, "user account is banned")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to refresh token")
		}
		return
	}

	h.setRefreshTokenCookie(w, response.RefreshToken)

	common.RespondJSON(w, http.StatusOK, response)
}
