package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTagRequest struct {
	Name string `json:"name"`
}

type UpdateProblemTagsRequest struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}
