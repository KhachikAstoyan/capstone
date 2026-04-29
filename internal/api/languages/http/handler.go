package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/languages/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/languages/service"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/access"
	problemsservice "github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service     *service.Service
	problemsSvc problemsservice.Service
}

func NewHandler(service *service.Service, problemsSvc problemsservice.Service) *Handler {
	return &Handler{service: service, problemsSvc: problemsSvc}
}

func (h *Handler) ListLanguages(w http.ResponseWriter, r *http.Request) {
	languages, err := h.service.ListLanguages(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to list languages")
		return
	}
	if languages == nil {
		languages = []domain.Language{}
	}
	common.RespondJSON(w, http.StatusOK, map[string]any{
		"languages": languages,
	})
}

func (h *Handler) CreateLanguage(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lang, err := h.service.CreateLanguage(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			common.RespondSimpleError(w, http.StatusBadRequest, "language key and display name are required")
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to save language")
		return
	}

	common.RespondJSON(w, http.StatusCreated, lang)
}

func (h *Handler) GetProblemLanguages(w http.ResponseWriter, r *http.Request) {
	problemID, ok := parseProblemID(w, r)
	if !ok {
		return
	}

	problem, err := h.problemsSvc.GetProblem(r.Context(), problemID, nil)
	if err != nil {
		if errors.Is(err, problemsservice.ErrProblemNotFound) {
			common.RespondSimpleError(w, http.StatusNotFound, "problem not found")
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to get problem")
		return
	}

	if !access.IsPublished(problem) && !access.CanViewUnpublishedProblems(r.Context()) {
		common.RespondSimpleError(w, http.StatusNotFound, "problem not found")
		return
	}

	h.respondProblemLanguages(w, r, problemID)
}

func (h *Handler) GetInternalProblemLanguages(w http.ResponseWriter, r *http.Request) {
	problemID, ok := parseProblemID(w, r)
	if !ok {
		return
	}
	h.respondProblemLanguages(w, r, problemID)
}

func (h *Handler) UpdateProblemLanguages(w http.ResponseWriter, r *http.Request) {
	problemID, ok := parseProblemID(w, r)
	if !ok {
		return
	}

	var req domain.UpdateProblemLanguagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateProblemLanguages(r.Context(), problemID, req); err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			common.RespondSimpleError(w, http.StatusBadRequest, "invalid language id")
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to update problem languages")
		return
	}

	h.respondProblemLanguages(w, r, problemID)
}

func (h *Handler) respondProblemLanguages(w http.ResponseWriter, r *http.Request, problemID uuid.UUID) {
	languages, err := h.service.ListProblemLanguages(r.Context(), problemID)
	if err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to list problem languages")
		return
	}
	if languages == nil {
		languages = []domain.Language{}
	}
	common.RespondJSON(w, http.StatusOK, map[string]any{
		"languages": languages,
	})
}

func parseProblemID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return uuid.Nil, false
	}
	return id, true
}
