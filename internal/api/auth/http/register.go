package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
)

type RegisterRequestDTO struct {
	Handle      string  `json:"handle"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"display_name,omitempty"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Handle == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "handle is required")
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

	serviceReq := service.RegisterRequest{
		Handle:      req.Handle,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	}

	response, err := h.service.Register(r.Context(), serviceReq)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserAlreadyExists):
			common.RespondError(w, http.StatusConflict, err, "user with this email or handle already exists")
		case errors.Is(err, auth.ErrPasswordTooShort):
			common.RespondError(w, http.StatusBadRequest, err, "password is too short")
		case errors.Is(err, auth.ErrPasswordTooLong):
			common.RespondError(w, http.StatusBadRequest, err, "password is too long")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to register user")
		}
		return
	}

	common.RespondJSON(w, http.StatusCreated, response)
}
