package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/access"
	problemsservice "github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/KhachikAstoyan/capstone/internal/api/tags/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/tags/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service     *service.Service
	problemsSvc problemsservice.Service
}

func NewHandler(tagSvc *service.Service, problemsSvc problemsservice.Service) *Handler {
	return &Handler{service: tagSvc, problemsSvc: problemsSvc}
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.service.CreateTag(r.Context(), req)
	if err != nil {
		if err.Error() == "tag name is required" || err.Error() == "tag with this name already exists" {
			common.RespondSimpleError(w, http.StatusBadRequest, err.Error())
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to create tag")
		return
	}

	common.RespondJSON(w, http.StatusCreated, tag)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.service.ListTags(r.Context())
	if err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to list tags")
		return
	}

	// Ensure we return an empty array instead of null
	if tags == nil {
		tags = []domain.Tag{}
	}

	common.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"tags": tags,
	})
}

func (h *Handler) UpdateProblemTags(w http.ResponseWriter, r *http.Request) {
	problemIDStr := chi.URLParam(r, "id")
	problemID, err := uuid.Parse(problemIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem ID")
		return
	}

	var req domain.UpdateProblemTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateProblemTags(r.Context(), problemID, req); err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to update problem tags")
		return
	}

	common.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "tags updated successfully",
	})
}

func (h *Handler) GetProblemTags(w http.ResponseWriter, r *http.Request) {
	problemIDStr := chi.URLParam(r, "id")
	problemID, err := uuid.Parse(problemIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem ID")
		return
	}

	problem, err := h.problemsSvc.GetProblem(r.Context(), problemID, nil)
	if err != nil {
		if errors.Is(err, problemsservice.ErrProblemNotFound) {
			common.RespondError(w, http.StatusNotFound, err, "problem not found")
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to get problem")
		return
	}

	if !access.IsPublished(problem) && !access.CanViewUnpublishedProblems(r.Context()) {
		common.RespondError(w, http.StatusNotFound, problemsservice.ErrProblemNotFound, "problem not found")
		return
	}

	tags, err := h.service.GetProblemTags(r.Context(), problemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			common.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"tags": []domain.Tag{},
			})
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to get problem tags")
		return
	}

	// Ensure we return an empty array instead of null
	if tags == nil {
		tags = []domain.Tag{}
	}

	common.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"tags": tags,
	})
}
