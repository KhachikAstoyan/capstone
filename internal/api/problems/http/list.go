package http

import (
	"net/http"
	"strconv"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
)

type ListProblemsResponse struct {
	Problems []*domain.Problem `json:"problems"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func (h *Handler) ListProblems(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	visibilityStr := r.URL.Query().Get("visibility")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	var visibility *domain.ProblemVisibility
	if visibilityStr != "" {
		v := domain.ProblemVisibility(visibilityStr)
		visibility = &v
	}

	problems, total, err := h.service.ListProblems(r.Context(), visibility, limit, offset)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list problems")
		return
	}

	response := ListProblemsResponse{
		Problems: problems,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}

	common.RespondJSON(w, http.StatusOK, response)
}
