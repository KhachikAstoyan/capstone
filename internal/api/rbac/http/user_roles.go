package http

import (
	"encoding/json"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AssignRoleRequest struct {
	RoleID string `json:"role_id"`
}

func (h *Handler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	// Get the current user (who is granting the role)
	grantedByID, ok := authhttp.GetUserIDFromContext(r.Context())
	var grantedBy *uuid.UUID
	if ok {
		grantedBy = &grantedByID
	}

	if err := h.service.AssignRoleToUser(r.Context(), userID, roleID, grantedBy); err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to assign role to user")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "role assigned successfully"})
}

func (h *Handler) RemoveRoleFromUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	if err := h.service.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to remove role from user")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]string{"message": "role removed successfully"})
}

func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	roles, err := h.service.GetUserRoles(r.Context(), userID)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get user roles")
		return
	}

	common.RespondJSON(w, http.StatusOK, roles)
}

func (h *Handler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	permissions, err := h.service.GetUserPermissions(r.Context(), userID)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get user permissions")
		return
	}

	common.RespondJSON(w, http.StatusOK, permissions)
}

func (h *Handler) GetMyRoles(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	roles, err := h.service.GetUserRoles(r.Context(), userID)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get user roles")
		return
	}

	common.RespondJSON(w, http.StatusOK, roles)
}

func (h *Handler) GetMyPermissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	permissions, err := h.service.GetUserPermissions(r.Context(), userID)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get user permissions")
		return
	}

	common.RespondJSON(w, http.StatusOK, permissions)
}
