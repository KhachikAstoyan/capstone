package service

import (
	"context"
	"errors"
	"strings"

	"github.com/KhachikAstoyan/capstone/internal/api/languages/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/languages/repository"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListLanguages(ctx context.Context, search string) ([]domain.Language, error) {
	return s.repo.List(ctx, strings.TrimSpace(search))
}

func (s *Service) CreateLanguage(ctx context.Context, req domain.CreateLanguageRequest) (*domain.Language, error) {
	key := strings.ToLower(strings.TrimSpace(req.Key))
	displayName := strings.TrimSpace(req.DisplayName)
	if key == "" || displayName == "" {
		return nil, ErrInvalidInput
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	return s.repo.Upsert(ctx, key, displayName, enabled)
}

func (s *Service) ListProblemLanguages(ctx context.Context, problemID uuid.UUID) ([]domain.Language, error) {
	return s.repo.ListForProblem(ctx, problemID)
}

func (s *Service) UpdateProblemLanguages(ctx context.Context, problemID uuid.UUID, req domain.UpdateProblemLanguagesRequest) error {
	seen := make(map[uuid.UUID]struct{}, len(req.LanguageIDs))
	ids := make([]uuid.UUID, 0, len(req.LanguageIDs))
	for _, id := range req.LanguageIDs {
		if id == uuid.Nil {
			return ErrInvalidInput
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return s.repo.SetForProblem(ctx, problemID, ids)
}
