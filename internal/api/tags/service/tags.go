package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/KhachikAstoyan/capstone/internal/api/tags/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/tags/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTag(ctx context.Context, req domain.CreateTagRequest) (*domain.Tag, error) {
	if req.Name == "" {
		return nil, errors.New("tag name is required")
	}

	// Check if tag already exists
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existing != nil {
		// Tag already exists, return it instead of error
		return existing, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Tag doesn't exist, create it
	return s.repo.Create(ctx, req.Name)
}

func (s *Service) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.List(ctx)
}

func (s *Service) UpdateProblemTags(ctx context.Context, problemID uuid.UUID, req domain.UpdateProblemTagsRequest) error {
	return s.repo.SetProblemTags(ctx, problemID, req.TagIDs)
}

func (s *Service) GetProblemTags(ctx context.Context, problemID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.GetProblemTags(ctx, problemID)
}
