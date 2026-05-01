package domain

import (
	"time"

	"github.com/google/uuid"
)

type Language struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	DisplayName string    `json:"display_name"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateLanguageRequest struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	IsEnabled   *bool  `json:"is_enabled,omitempty"`
}

type UpdateProblemLanguagesRequest struct {
	LanguageIDs []uuid.UUID `json:"language_ids"`
}
