package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/KhachikAstoyan/capstone/internal/api/submissions/client"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/driver"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/repository"
	problemsdomain "github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	cpdomain "github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
)

var (
	ErrForbidden          = errors.New("access denied")
	ErrNoTestCases        = errors.New("problem has no active test cases")
	ErrInvalidInput       = errors.New("invalid input")
	ErrProblemNotFound    = errors.New("problem not found")
	ErrSubmissionNotFound = repository.ErrSubmissionNotFound
	ErrLanguageNotAllowed = repository.ErrLanguageNotAllowed
	ErrLanguageNotFound   = repository.ErrLanguageNotFound
)

// ProblemsReader is a minimal interface satisfied by the problems repository.
type ProblemsReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*problemsdomain.Problem, error)
}

type Service interface {
	Submit(ctx context.Context, userID, problemID uuid.UUID, req domain.CreateSubmissionRequest) (*domain.Submission, error)
	Run(ctx context.Context, userID, problemID uuid.UUID, req domain.CreateSubmissionRequest) (*domain.Submission, error)
	GetSubmission(ctx context.Context, id, callerUserID uuid.UUID, callerIsAdmin bool) (*domain.Submission, error)
	ListSubmissions(ctx context.Context, filters domain.ListFilters, limit, offset int) ([]*domain.Submission, int, error)
}

type service struct {
	repo         repository.Repository
	cpClient     client.Client
	problemsRepo ProblemsReader
}

func NewService(repo repository.Repository, cp client.Client, pr ProblemsReader) Service {
	return &service{repo: repo, cpClient: cp, problemsRepo: pr}
}

func (s *service) Submit(ctx context.Context, userID, problemID uuid.UUID, req domain.CreateSubmissionRequest) (*domain.Submission, error) {
	return s.create(ctx, userID, problemID, req, domain.KindSubmit)
}

func (s *service) Run(ctx context.Context, userID, problemID uuid.UUID, req domain.CreateSubmissionRequest) (*domain.Submission, error) {
	return s.create(ctx, userID, problemID, req, domain.KindRun)
}

func (s *service) create(ctx context.Context, userID, problemID uuid.UUID, req domain.CreateSubmissionRequest, kind domain.SubmissionKind) (*domain.Submission, error) {
	if strings.TrimSpace(req.SourceText) == "" || strings.TrimSpace(req.LanguageKey) == "" {
		return nil, ErrInvalidInput
	}

	langID, langKey, err := s.repo.ResolveLanguage(ctx, req.LanguageKey)
	if err != nil {
		return nil, err
	}

	allowed, err := s.repo.IsLanguageAllowed(ctx, problemID, langID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrLanguageNotAllowed
	}

	problem, err := s.problemsRepo.GetByID(ctx, problemID)
	if err != nil {
		return nil, ErrProblemNotFound
	}

	testCases, err := s.repo.GetTestCasesForProblem(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if kind == domain.KindRun {
		visible := testCases[:0]
		for _, tc := range testCases {
			if !tc.IsHidden {
				visible = append(visible, tc)
			}
		}
		testCases = visible
	}
	if len(testCases) == 0 {
		return nil, ErrNoTestCases
	}

	if problem.FunctionSpec == nil {
		return nil, fmt.Errorf("%w: problem missing function spec", ErrInvalidInput)
	}
	sourceToSend, err := driver.Wrap(*problem.FunctionSpec, langKey, req.SourceText)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	sub, err := s.repo.Create(ctx, userID, problemID, langID, req.SourceText, kind)
	if err != nil {
		return nil, err
	}

	cpTestCases := make([]cpdomain.TestCase, len(testCases))
	for i, tc := range testCases {
		cpTestCases[i] = cpdomain.TestCase{
			ID:       tc.ExternalID,
			Input:    string(tc.InputData),
			Expected: string(tc.ExpectedData),
		}
	}

	job, err := s.cpClient.CreateJob(ctx, cpdomain.CreateJobRequest{
		SubmissionID:  sub.ID,
		Language:      langKey,
		SourceText:    &sourceToSend,
		TimeLimitMs:   problem.TimeLimitMs,
		MemoryLimitMb: problem.MemoryLimitMb,
		TestCases:     cpTestCases,
	})
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, sub.ID, domain.StatusInternalError)
		return nil, err
	}

	_ = s.repo.UpdateCPJobID(ctx, sub.ID, job.ID)
	_ = s.repo.UpdateStatus(ctx, sub.ID, domain.StatusQueued)

	sub.CPJobID = &job.ID
	sub.Status = domain.StatusQueued
	return sub, nil
}

func (s *service) GetSubmission(ctx context.Context, id, callerUserID uuid.UUID, callerIsAdmin bool) (*domain.Submission, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if sub.UserID != callerUserID && !callerIsAdmin {
		return nil, ErrForbidden
	}

	if sub.Status.IsTerminal() {
		result, err := s.repo.GetResult(ctx, sub.ID)
		if err != nil && !errors.Is(err, repository.ErrSubmissionNotFound) {
			return nil, err
		}
		if result != nil {
			s.enrichResult(ctx, sub.ProblemID, result)
		}
		sub.Result = result
		return sub, nil
	}

	if sub.CPJobID == nil {
		return sub, nil
	}

	job, err := s.cpClient.GetJobBySubmission(ctx, sub.ID)
	if err != nil {
		return sub, nil
	}

	newStatus := jobStateToStatus(job.State)
	if newStatus != sub.Status {
		_ = s.repo.UpdateStatus(ctx, sub.ID, newStatus)
		sub.Status = newStatus
	}

	if job.State == cpdomain.JobStateCompleted || job.State == cpdomain.JobStateFailed {
		jobResult, err := s.cpClient.GetJobResult(ctx, job.ID)
		if err != nil {
			return sub, nil
		}

		result := cpResultToSubmissionResult(sub.ID, jobResult)
		finalStatus := verdictToStatus(jobResult.OverallVerdict)

		_ = s.repo.SaveResult(ctx, result, finalStatus)
		sub.Status = finalStatus
		s.enrichResult(ctx, sub.ProblemID, &result)
		sub.Result = &result
	}

	return sub, nil
}

func (s *service) ListSubmissions(ctx context.Context, filters domain.ListFilters, limit, offset int) ([]*domain.Submission, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, filters, limit, offset)
}

func jobStateToStatus(state cpdomain.JobState) domain.SubmissionStatus {
	switch state {
	case cpdomain.JobStateQueued:
		return domain.StatusQueued
	case cpdomain.JobStateAssigned, cpdomain.JobStateRunning:
		return domain.StatusRunning
	case cpdomain.JobStateCompleted:
		return domain.StatusRunning // will be resolved by verdict shortly
	case cpdomain.JobStateFailed, cpdomain.JobStateRetryPending:
		return domain.StatusInternalError
	default:
		return domain.StatusQueued
	}
}

func verdictToStatus(verdict string) domain.SubmissionStatus {
	switch verdict {
	case "Accepted":
		return domain.StatusAccepted
	case "WrongAnswer":
		return domain.StatusWrongAnswer
	case "TimeLimitExceeded":
		return domain.StatusTimeLimitExceeded
	case "MemoryLimitExceeded":
		return domain.StatusMemoryLimitExceeded
	case "RuntimeError":
		return domain.StatusRuntimeError
	case "CompilationError":
		return domain.StatusCompilationError
	default:
		return domain.StatusInternalError
	}
}

func cpResultToSubmissionResult(submissionID uuid.UUID, r *cpdomain.JobResult) domain.SubmissionResult {
	entries := make([]domain.TestcaseResultEntry, len(r.TestcaseResults))
	for i, tc := range r.TestcaseResults {
		entries[i] = domain.TestcaseResultEntry{
			TestcaseID:   tc.TestcaseID,
			Verdict:      tc.Verdict,
			TimeMs:       tc.TimeMs,
			MemoryKb:     tc.MemoryKb,
			ActualOutput: tc.ActualOutput,
			StdoutOutput: tc.StdoutRef,
		}
	}
	return domain.SubmissionResult{
		SubmissionID:    submissionID,
		OverallVerdict:  r.OverallVerdict,
		TotalTimeMs:     r.TotalTimeMs,
		MaxMemoryKb:     r.MaxMemoryKb,
		WallTimeMs:      r.WallTimeMs,
		CompilerOutput:  r.CompilerOutput,
		TestcaseResults: entries,
	}
}

// enrichResult attaches input/expected data to non-hidden test case entries
// and strips actual output from hidden ones. Called before returning to the caller.
func (s *service) enrichResult(ctx context.Context, problemID uuid.UUID, result *domain.SubmissionResult) {
	if result == nil || len(result.TestcaseResults) == 0 {
		return
	}
	tcs, err := s.repo.GetTestCasesForProblem(ctx, problemID)
	if err != nil {
		return
	}
	tcMap := make(map[string]*domain.ProblemTestCase, len(tcs))
	for _, tc := range tcs {
		tcMap[tc.ExternalID] = tc
	}
	for i := range result.TestcaseResults {
		entry := &result.TestcaseResults[i]
		tc, ok := tcMap[entry.TestcaseID]
		if !ok || tc.IsHidden {
			entry.ActualOutput = nil
			entry.StdoutOutput = nil
			continue
		}
		entry.InputData = tc.InputData
		entry.ExpectedData = tc.ExpectedData
	}
}
