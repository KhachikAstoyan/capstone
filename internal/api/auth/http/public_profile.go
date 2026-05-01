package http

import (
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetPublicUserProfile(w http.ResponseWriter, r *http.Request) {
	userRef := chi.URLParam(r, "userRef")
	if userRef == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "user reference is required")
		return
	}

	profile, err := h.service.GetPublicProfile(r.Context(), userRef)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			common.RespondError(w, http.StatusNotFound, err, "user not found")
		case errors.Is(err, service.ErrInvalidUserRef):
			common.RespondSimpleError(w, http.StatusBadRequest, "invalid user reference")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to load profile")
		}
		return
	}

	common.RespondJSON(w, http.StatusOK, profile)
}
