package http

import (
	"net/http"
	"strconv"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/access"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/repository"
	"github.com/google/uuid"
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
	difficultyStr := r.URL.Query().Get("difficulty")
	searchStr := r.URL.Query().Get("search")
	tags := r.URL.Query()["tags[]"]

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

	filters := repository.ListFilters{
		Search: searchStr,
		Tags:   tags,
	}

	canUnpublished := access.CanViewUnpublishedProblems(r.Context())
	if !canUnpublished {
		switch visibilityStr {
		case "":
			v := domain.VisibilityPublished
			filters.Visibility = &v
		case string(domain.VisibilityPublished):
			v := domain.VisibilityPublished
			filters.Visibility = &v
		default:
			common.RespondSimpleError(w, http.StatusForbidden, "insufficient permissions to list non-published problems")
			return
		}
	} else if visibilityStr != "" {
		v := domain.ProblemVisibility(visibilityStr)
		filters.Visibility = &v
	}

	if difficultyStr != "" {
		d := domain.ProblemDifficulty(difficultyStr)
		filters.Difficulty = &d
	}

	var userID *uuid.UUID
	if uid, ok := authhttp.GetUserIDFromContext(r.Context()); ok {
		userID = &uid
	}

	problems, total, err := h.service.ListProblems(r.Context(), filters, userID, limit, offset)
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
