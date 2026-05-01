package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const adminPageSize = 20

type adminUsersResponse struct {
	Users  interface{} `json:"users"`
	Total  int         `json:"total"`
	Page   int         `json:"page"`
	Limit  int         `json:"limit"`
}

func (h *Handler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = adminPageSize
	}
	offset := (page - 1) * limit

	users, total, err := h.service.ListAdminUsers(r.Context(), q, sortBy, limit, offset)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list users")
		return
	}
	if users == nil {
		users = []interface{}{}
	}

	common.RespondJSON(w, http.StatusOK, adminUsersResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (h *Handler) GetAdminUserSecurityEvents(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = adminPageSize
	}
	offset := (page - 1) * limit

	events, total, err := h.service.GetUserSecurityEvents(r.Context(), userID, limit, offset)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			common.RespondSimpleError(w, http.StatusNotFound, "user not found")
			return
		}
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get security events")
		return
	}
	if events == nil {
		events = []interface{}{}
	}

	common.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}
