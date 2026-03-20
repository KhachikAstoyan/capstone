package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/problems"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/repository"
	"github.com/google/uuid"
)

var (
	ErrProblemNotFound     = repository.ErrProblemNotFound
	ErrProblemSlugConflict = repository.ErrProblemSlugConflict
	ErrInvalidInput        = errors.New("invalid input")
)

type Service interface {
	CreateProblem(ctx context.Context, req domain.CreateProblemRequest, createdByUserID uuid.UUID) (*domain.Problem, error)
	GetProblem(ctx context.Context, id uuid.UUID) (*domain.Problem, error)
	GetProblemBySlug(ctx context.Context, slug string) (*domain.Problem, error)
	ListProblems(ctx context.Context, visibility *domain.ProblemVisibility, limit, offset int) ([]*domain.Problem, int, error)
	UpdateProblem(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error)
	DeleteProblem(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateProblem(ctx context.Context, req domain.CreateProblemRequest, createdByUserID uuid.UUID) (*domain.Problem, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Generate slug from title
	baseSlug := problems.GenerateSlug(req.Title)
	slug := baseSlug

	// Try to create with generated slug, if conflict, append number
	var problem *domain.Problem
	var err error
	for i := 1; i <= 10; i++ {
		problem, err = s.repo.Create(ctx, req, slug, createdByUserID)
		if err == nil {
			return problem, nil
		}

		if !errors.Is(err, repository.ErrProblemSlugConflict) {
			return nil, err
		}

		// Slug conflict, try with number suffix
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	return nil, fmt.Errorf("failed to generate unique slug after 10 attempts")
}

func (s *service) GetProblem(ctx context.Context, id uuid.UUID) (*domain.Problem, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetProblemBySlug(ctx context.Context, slug string) (*domain.Problem, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *service) ListProblems(ctx context.Context, visibility *domain.ProblemVisibility, limit, offset int) ([]*domain.Problem, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	problems, err := s.repo.List(ctx, visibility, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, visibility)
	if err != nil {
		return nil, 0, err
	}

	return problems, count, nil
}

func (s *service) UpdateProblem(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error) {
	if err := validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	problem, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return problem, nil
}

func (s *service) DeleteProblem(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateCreateRequest(req domain.CreateProblemRequest) error {
	if req.Title == "" {
		return errors.New("title is required")
	}
	if req.StatementMarkdown == "" {
		return errors.New("statement is required")
	}
	if req.TimeLimitMs <= 0 {
		return errors.New("time_limit_ms must be positive")
	}
	if req.MemoryLimitMb <= 0 {
		return errors.New("memory_limit_mb must be positive")
	}
	if req.TestsRef == "" {
		return errors.New("tests_ref is required")
	}
	if req.Visibility != domain.VisibilityDraft &&
		req.Visibility != domain.VisibilityPublished &&
		req.Visibility != domain.VisibilityArchived {
		return errors.New("invalid visibility value")
	}
	return nil
}

func validateUpdateRequest(req domain.UpdateProblemRequest) error {
	if req.Slug != nil && *req.Slug == "" {
		return errors.New("slug cannot be empty")
	}
	if req.Title != nil && *req.Title == "" {
		return errors.New("title cannot be empty")
	}
	if req.StatementMarkdown != nil && *req.StatementMarkdown == "" {
		return errors.New("statement cannot be empty")
	}
	if req.TimeLimitMs != nil && *req.TimeLimitMs <= 0 {
		return errors.New("time_limit_ms must be positive")
	}
	if req.MemoryLimitMb != nil && *req.MemoryLimitMb <= 0 {
		return errors.New("memory_limit_mb must be positive")
	}
	if req.TestsRef != nil && *req.TestsRef == "" {
		return errors.New("tests_ref cannot be empty")
	}
	if req.Visibility != nil {
		if *req.Visibility != domain.VisibilityDraft &&
			*req.Visibility != domain.VisibilityPublished &&
			*req.Visibility != domain.VisibilityArchived {
			return errors.New("invalid visibility value")
		}
	}
	return nil
}
