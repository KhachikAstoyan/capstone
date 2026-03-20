package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CreatePermissionRequest struct {
	Key         string  `json:"key"`
	Description *string `json:"description,omitempty"`
}

type UpdatePermissionRequest struct {
	Key         string  `json:"key"`
	Description *string `json:"description,omitempty"`
}

func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "key is required")
		return
	}

	permission, err := h.service.CreatePermission(r.Context(), req.Key, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrPermissionAlreadyExists) {
			common.RespondError(w, http.StatusConflict, err, "permission already exists")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to create permission")
		return
	}

	common.RespondJSON(w, http.StatusCreated, permission)
}

func (h *Handler) GetPermission(w http.ResponseWriter, r *http.Request) {
	permissionIDStr := chi.URLParam(r, "permissionID")
	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid permission ID")
		return
	}

	permission, err := h.service.GetPermission(r.Context(), permissionID)
	if err != nil {
		if errors.Is(err, repository.ErrPermissionNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "permission not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get permission")
		return
	}

	common.RespondJSON(w, http.StatusOK, permission)
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	permissions, err := h.service.ListPermissions(r.Context())
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list permissions")
		return
	}

	common.RespondJSON(w, http.StatusOK, permissions)
}

func (h *Handler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	permissionIDStr := chi.URLParam(r, "permissionID")
	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid permission ID")
		return
	}

	var req UpdatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "key is required")
		return
	}

	permission, err := h.service.UpdatePermission(r.Context(), permissionID, req.Key, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrPermissionNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "permission not found")
			return
		}
		if errors.Is(err, repository.ErrPermissionAlreadyExists) {
			common.RespondError(w, http.StatusConflict, err, "permission key already exists")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to update permission")
		return
	}

	common.RespondJSON(w, http.StatusOK, permission)
}

func (h *Handler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	permissionIDStr := chi.URLParam(r, "permissionID")
	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid permission ID")
		return
	}

	if err := h.service.DeletePermission(r.Context(), permissionID); err != nil {
		if errors.Is(err, repository.ErrPermissionNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "permission not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to delete permission")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "permission deleted successfully"})
}
