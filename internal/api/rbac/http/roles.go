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

type CreateRoleRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateRoleRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "name is required")
		return
	}

	role, err := h.service.CreateRole(r.Context(), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrRoleAlreadyExists) {
			common.RespondError(w, http.StatusConflict, err, "role already exists")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to create role")
		return
	}

	common.RespondJSON(w, http.StatusCreated, role)
}

func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	role, err := h.service.GetRole(r.Context(), roleID)
	if err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "role not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get role")
		return
	}

	common.RespondJSON(w, http.StatusOK, role)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list roles")
		return
	}

	common.RespondJSON(w, http.StatusOK, roles)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "name is required")
		return
	}

	role, err := h.service.UpdateRole(r.Context(), roleID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "role not found")
			return
		}
		if errors.Is(err, repository.ErrRoleAlreadyExists) {
			common.RespondError(w, http.StatusConflict, err, "role name already exists")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to update role")
		return
	}

	common.RespondJSON(w, http.StatusOK, role)
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	if err := h.service.DeleteRole(r.Context(), roleID); err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "role not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to delete role")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "role deleted successfully"})
}

func (h *Handler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	roleWithPerms, err := h.service.GetRoleWithPermissions(r.Context(), roleID)
	if err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "role not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get role permissions")
		return
	}

	common.RespondJSON(w, http.StatusOK, roleWithPerms)
}

type AssignPermissionToRoleRequest struct {
	PermissionID string `json:"permission_id"`
}

func (h *Handler) AssignPermissionToRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	var req AssignPermissionToRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	permissionID, err := uuid.Parse(req.PermissionID)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid permission ID")
		return
	}

	if err := h.service.AssignPermissionToRole(r.Context(), roleID, permissionID); err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to assign permission to role")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "permission assigned successfully"})
}

func (h *Handler) RemovePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	permissionIDStr := chi.URLParam(r, "permissionID")
	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid permission ID")
		return
	}

	if err := h.service.RemovePermissionFromRole(r.Context(), roleID, permissionID); err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to remove permission from role")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "permission removed successfully"})
}
