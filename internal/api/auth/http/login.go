package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

type LoginRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "email is required")
		return
	}

	if req.Password == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "password is required")
		return
	}

	serviceReq := service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	response, err := h.service.Login(r.Context(), serviceReq)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			common.RespondError(w, http.StatusUnauthorized, err, "invalid email or password")
		case errors.Is(err, service.ErrUserBanned):
			common.RespondError(w, http.StatusForbidden, err, "user account is banned")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to login")
		}
		return
	}

	h.setRefreshTokenCookie(w, response.RefreshToken)

	common.RespondJSON(w, http.StatusOK, response)
}
