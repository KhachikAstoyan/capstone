package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/common"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) ListTestCases(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	tcs, err := h.service.ListTestCases(r.Context(), id)
	if err != nil {
		common.RespondError(w, http.StatusInternalServerError, err, "failed to list test cases")
		return
	}

	if tcs == nil {
		tcs = []*domain.TestCase{}
	}
	common.RespondJSON(w, http.StatusOK, map[string]any{"test_cases": tcs})
}

func (h *Handler) CreateTestCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	var req domain.CreateTestCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := h.service.CreateTestCase(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			common.RespondError(w, http.StatusBadRequest, err, "invalid input")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to create test case")
		}
		return
	}

	common.RespondJSON(w, http.StatusCreated, tc)
}

func (h *Handler) UpdateTestCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	tcIDStr := chi.URLParam(r, "tcId")
	tcID, err := uuid.Parse(tcIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid test case id")
		return
	}

	var req domain.UpdateTestCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tc, err := h.service.UpdateTestCase(r.Context(), tcID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			common.RespondError(w, http.StatusBadRequest, err, "invalid input")
		case errors.Is(err, repository.ErrTestCaseNotFound):
			common.RespondSimpleError(w, http.StatusNotFound, "test case not found")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to update test case")
		}
		return
	}

	common.RespondJSON(w, http.StatusOK, tc)
}

func (h *Handler) DeleteTestCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	tcIDStr := chi.URLParam(r, "tcId")
	tcID, err := uuid.Parse(tcIDStr)
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid test case id")
		return
	}

	if err := h.service.DeleteTestCase(r.Context(), tcID); err != nil {
		switch {
		case errors.Is(err, repository.ErrTestCaseNotFound):
			common.RespondSimpleError(w, http.StatusNotFound, "test case not found")
		default:
			common.RespondError(w, http.StatusInternalServerError, err, "failed to delete test case")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
