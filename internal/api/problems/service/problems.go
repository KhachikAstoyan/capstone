package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

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

const maxProblemSummaryRunes = 500

type Service interface {
	CreateProblem(ctx context.Context, req domain.CreateProblemRequest, createdByUserID uuid.UUID) (*domain.Problem, error)
	GetProblem(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Problem, error)
	GetProblemBySlug(ctx context.Context, slug string, userID *uuid.UUID) (*domain.Problem, error)
	ListProblems(ctx context.Context, filters repository.ListFilters, userID *uuid.UUID, limit, offset int) ([]*domain.Problem, int, error)
	UpdateProblem(ctx context.Context, id uuid.UUID, req domain.UpdateProblemRequest) (*domain.Problem, error)
	DeleteProblem(ctx context.Context, id uuid.UUID) error

	ListTestCases(ctx context.Context, problemID uuid.UUID) ([]*domain.TestCase, error)
	CreateTestCase(ctx context.Context, problemID uuid.UUID, req domain.CreateTestCaseRequest) (*domain.TestCase, error)
	UpdateTestCase(ctx context.Context, id uuid.UUID, req domain.UpdateTestCaseRequest) (*domain.TestCase, error)
	DeleteTestCase(ctx context.Context, id uuid.UUID) error
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

func (s *service) GetProblem(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Problem, error) {
	problem, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichSingleSolved(ctx, problem, userID)
	return problem, nil
}

func (s *service) GetProblemBySlug(ctx context.Context, slug string, userID *uuid.UUID) (*domain.Problem, error) {
	problem, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	s.enrichSingleSolved(ctx, problem, userID)
	return problem, nil
}

func (s *service) enrichSingleSolved(ctx context.Context, problem *domain.Problem, userID *uuid.UUID) {
	if userID == nil || problem == nil {
		return
	}
	ids, err := s.repo.GetSolvedProblemIDs(ctx, *userID, []uuid.UUID{problem.ID})
	if err != nil {
		return
	}
	for _, id := range ids {
		if id == problem.ID {
			problem.IsSolved = true
			return
		}
	}
}

func (s *service) ListProblems(ctx context.Context, filters repository.ListFilters, userID *uuid.UUID, limit, offset int) ([]*domain.Problem, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	problems, err := s.repo.List(ctx, filters, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	if len(problems) == 0 {
		return problems, count, nil
	}

	problemIDs := make([]uuid.UUID, len(problems))
	for i, p := range problems {
		problemIDs[i] = p.ID
	}

	tagsMap, err := s.repo.GetProblemTags(ctx, problemIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get problem tags: %w", err)
	}

	acceptanceRates, err := s.repo.GetAcceptanceRates(ctx, problemIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get acceptance rates: %w", err)
	}

	var solvedIDs []uuid.UUID
	if userID != nil {
		solvedIDs, err = s.repo.GetSolvedProblemIDs(ctx, *userID, problemIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get solved problem IDs: %w", err)
		}
	}

	solvedMap := make(map[uuid.UUID]bool)
	for _, id := range solvedIDs {
		solvedMap[id] = true
	}

	for _, problem := range problems {
		problem.Tags = tagsMap[problem.ID]
		if rate, ok := acceptanceRates[problem.ID]; ok {
			problem.AcceptanceRate = &rate
		}
		problem.IsSolved = solvedMap[problem.ID]
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

func (s *service) ListTestCases(ctx context.Context, problemID uuid.UUID) ([]*domain.TestCase, error) {
	return s.repo.ListTestCases(ctx, problemID)
}

func (s *service) CreateTestCase(ctx context.Context, problemID uuid.UUID, req domain.CreateTestCaseRequest) (*domain.TestCase, error) {
	if !json.Valid(req.InputData) || string(req.InputData) == "null" {
		return nil, fmt.Errorf("%w: input_data must be valid JSON", ErrInvalidInput)
	}
	if !json.Valid(req.ExpectedData) || string(req.ExpectedData) == "null" {
		return nil, fmt.Errorf("%w: expected_data must be valid JSON", ErrInvalidInput)
	}
	return s.repo.CreateTestCase(ctx, problemID, req)
}

func (s *service) UpdateTestCase(ctx context.Context, id uuid.UUID, req domain.UpdateTestCaseRequest) (*domain.TestCase, error) {
	if len(req.InputData) > 0 && (!json.Valid(req.InputData) || string(req.InputData) == "null") {
		return nil, fmt.Errorf("%w: input_data must be valid JSON", ErrInvalidInput)
	}
	if len(req.ExpectedData) > 0 && (!json.Valid(req.ExpectedData) || string(req.ExpectedData) == "null") {
		return nil, fmt.Errorf("%w: expected_data must be valid JSON", ErrInvalidInput)
	}
	return s.repo.UpdateTestCase(ctx, id, req)
}

func (s *service) DeleteTestCase(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTestCase(ctx, id)
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
	if req.Visibility != domain.VisibilityDraft &&
		req.Visibility != domain.VisibilityPublished &&
		req.Visibility != domain.VisibilityArchived {
		return errors.New("invalid visibility value")
	}
	if req.Difficulty != domain.DifficultyEasy &&
		req.Difficulty != domain.DifficultyMedium &&
		req.Difficulty != domain.DifficultyHard {
		return errors.New("invalid difficulty value")
	}
	if utf8.RuneCountInString(req.Summary) > maxProblemSummaryRunes {
		return errors.New("summary must be at most 500 characters")
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
	if req.Difficulty != nil {
		if *req.Difficulty != domain.DifficultyEasy &&
			*req.Difficulty != domain.DifficultyMedium &&
			*req.Difficulty != domain.DifficultyHard {
			return errors.New("invalid difficulty value")
		}
	}
	if req.Summary != nil && utf8.RuneCountInString(*req.Summary) > maxProblemSummaryRunes {
		return errors.New("summary must be at most 500 characters")
	}
	return nil
}
