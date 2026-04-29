package http

import (
	"net/http"
	"strconv"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/domain"
	"github.com/google/uuid"
)

type listSubmissionsResponse struct {
	Submissions []*domain.Submission `json:"submissions"`
	Total       int                  `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

func (h *Handler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	filters := domain.ListFilters{UserID: userID}
	if v := r.URL.Query().Get("problem_id"); v != "" {
		if pid, err := uuid.Parse(v); err == nil {
			filters.ProblemID = &pid
		}
	}

	subs, total, err := h.service.ListSubmissions(r.Context(), filters, limit, offset)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list submissions")
		return
	}

	if subs == nil {
		subs = []*domain.Submission{}
	}

	common.RespondJSON(w, http.StatusOK, listSubmissionsResponse{
		Submissions: subs,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	})
}
