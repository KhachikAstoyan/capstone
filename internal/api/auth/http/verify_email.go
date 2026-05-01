package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

type VerifyEmailRequestDTO struct {
	Token string `json:"token"`
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "token is required")
		return
	}

	outcome, err := h.service.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVerificationToken):
			common.RespondError(w, http.StatusBadRequest, err, "invalid or expired verification token")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to verify email")
		}
		return
	}

	switch outcome {
	case service.VerifyEmailOutcomeResent:
		common.RespondJSON(w, http.StatusOK, map[string]string{
			"status":  string(service.VerifyEmailOutcomeResent),
			"message": "this verification link expired; a new verification email will be sent",
		})
	default:
		common.RespondJSON(w, http.StatusOK, map[string]string{
			"status":  string(service.VerifyEmailOutcomeVerified),
			"message": "email verified successfully",
		})
	}
}
