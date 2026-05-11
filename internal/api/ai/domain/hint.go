package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
)

type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	Role      ChatRole  `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type HintRequest struct {
	UserID           uuid.UUID
	ProblemID        uuid.UUID
	ProblemStatement string
	LanguageKey      string
	Code             string
	Message          string
}

type HintResponse struct {
	UserMessage      ChatMessage
	AssistantMessage ChatMessage
}
