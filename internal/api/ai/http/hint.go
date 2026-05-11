package http

import (
	"encoding/json"
	"errors"
	"net/http"

	aidomain "github.com/KhachikAstoyan/capstone/internal/api/ai/domain"
	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/common"
	problemsservice "github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type hintHTTPRequest struct {
	LanguageKey string `json:"language_key"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

type hintHTTPResponse struct {
	UserMessage      aidomain.ChatMessage `json:"user_message"`
	AssistantMessage aidomain.ChatMessage `json:"assistant_message"`
}

type hintHistoryResponse struct {
	Messages []aidomain.ChatMessage `json:"messages"`
}

func (h *Handler) GetHint(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	problemID, err := uuid.Parse(chi.URLParam(r, "problemID"))
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	var req hintHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.LanguageKey == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "language_key is required")
		return
	}
	if req.Message == "" {
		common.RespondSimpleError(w, http.StatusBadRequest, "message is required")
		return
	}

	problem, err := h.problemsSvc.GetProblem(r.Context(), problemID, nil)
	if err != nil {
		if errors.Is(err, problemsservice.ErrProblemNotFound) {
			common.RespondSimpleError(w, http.StatusNotFound, "problem not found")
			return
		}
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to fetch problem")
		return
	}

	hint, err := h.svc.GetHint(r.Context(), aidomain.HintRequest{
		UserID:           userID,
		ProblemID:        problemID,
		ProblemStatement: problem.StatementMarkdown,
		LanguageKey:      req.LanguageKey,
		Code:             req.Code,
		Message:          req.Message,
	})
	if err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to generate hint")
		return
	}

	common.RespondJSON(w, http.StatusOK, hintHTTPResponse{
		UserMessage:      hint.UserMessage,
		AssistantMessage: hint.AssistantMessage,
	})
}

func (h *Handler) GetHintHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := authhttp.GetUserIDFromContext(r.Context())
	if !ok {
		common.RespondSimpleError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	problemID, err := uuid.Parse(chi.URLParam(r, "problemID"))
	if err != nil {
		common.RespondSimpleError(w, http.StatusBadRequest, "invalid problem id")
		return
	}

	msgs, err := h.svc.GetHintHistory(r.Context(), userID, problemID)
	if err != nil {
		common.RespondSimpleError(w, http.StatusInternalServerError, "failed to get hint history")
		return
	}

	if msgs == nil {
		msgs = []aidomain.ChatMessage{}
	}

	common.RespondJSON(w, http.StatusOK, hintHistoryResponse{Messages: msgs})
}
