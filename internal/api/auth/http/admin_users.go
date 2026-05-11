package http

import (
	"net/http"
	"strconv"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const adminPageSize = 20

type adminUsersResponse struct {
	Users []domain.AdminUserSummary `json:"users"`
	Total int                       `json:"total"`
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
}

type adminSecurityEventsResponse struct {
	Events []domain.SecurityEvent `json:"events"`
	Total  int                    `json:"total"`
	Page   int                    `json:"page"`
	Limit  int                    `json:"limit"`
}

func parsePage(r *http.Request) (page, limit, offset int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = adminPageSize
	}
	offset = (page - 1) * limit
	return
}

func (h *Handler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")
	page, limit, offset := parsePage(r)

	users, total, err := h.service.ListAdminUsers(r.Context(), q, sortBy, limit, offset)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list users")
		return
	}
	if users == nil {
		users = []domain.AdminUserSummary{}
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

	page, limit, offset := parsePage(r)

	events, total, err := h.service.GetUserSecurityEvents(r.Context(), userID, limit, offset)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to get security events")
		return
	}
	if events == nil {
		events = []domain.SecurityEvent{}
	}

	common.RespondJSON(w, http.StatusOK, adminSecurityEventsResponse{
		Events: events,
		Total:  total,
		Page:   page,
		Limit:  limit,
	})
}
