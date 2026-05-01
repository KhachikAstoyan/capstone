package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/ai/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/ai/repository"
	"github.com/google/uuid"
	"go.jetify.com/ai"
	aiapi "go.jetify.com/ai/api"
	"go.uber.org/zap"
)

const systemPrompt = `You are a code security validator for an online programming judge. Your sole responsibility is to block any code that is NOT a pure algorithmic solution.

FORBIDDEN - block immediately if code contains ANY of:
- Process spawning: subprocess.run, subprocess.Popen, os.fork, os.system, os.exec, os.spawn, Runtime.exec, ProcessBuilder, fork(), exec(), system(), popen()
- Filesystem operations: open, read, write, delete files; os.path, pathlib, File I/O beyond reading stdin
- Network operations: socket, requests, urllib, http.client, socket module, fetch, XMLHttpRequest, WebSocket, DNS lookups
- Dangerous imports: subprocess, os, sys, socket, requests, urllib, tempfile, shutil, webbrowser, ctypes, threading (except basic use), multiprocessing, asyncio
- Environment access: os.environ, getenv, System.getenv, process.env
- Privilege escalation: setuid, setgid, sudo calls
- Reflection/Dynamic code execution: eval, exec, __import__, importlib, compile (except string compilation for parsing), pickle, marshal
- Infinite loops without guaranteed termination (resource exhaustion)
- Anything that escapes the execution sandbox

ALLOWED - only code that:
- Performs algorithmic computation (sorting, searching, math, string manipulation, etc.)
- Writes to stdout/return values
- Uses standard safe library functions (math operations, string operations, basic data structures)

You must respond ONLY with valid JSON in this exact format:
{
  "is_allowed": true/false,
  "severity": "info|warn|high|block",
  "reason": "brief explanation of the decision",
  "details": {
    "violations": ["list", "of", "violations", "or", "empty"],
    "safe_features_detected": ["list", "of", "safe", "features", "or", "empty"]
  }
}

Rules:
- is_allowed: true ONLY if code contains NO forbidden patterns and performs only algorithmic problem-solving
- severity: "block" if is_allowed=false, "warn" if minor concerns but allowed, "info" if allowed with notes
- reason: concise explanation. If violations found, state them clearly. NO suggestions to user.
- If violations are detected, is_allowed MUST be false with severity "block"
- No markdown, no extra text, ONLY valid JSON

Example BLOCKED response:
{
  "is_allowed": false,
  "severity": "block",
  "reason": "Code uses subprocess.run to spawn child processes, which is not allowed in sandboxed execution",
  "details": {
    "violations": ["subprocess.run() import and call detected", "process spawning"],
    "safe_features_detected": []
  }
}

Example ALLOWED response:
{
  "is_allowed": true,
  "severity": "info",
  "reason": "Code is a pure algorithmic solution using only standard math and string operations",
  "details": {
    "violations": [],
    "safe_features_detected": ["uses sorting", "uses loops", "string manipulation"]
  }
}`

type Service struct {
	repo  *repository.Repository
	model aiapi.LanguageModel
	log   *zap.Logger
}

func New(repo *repository.Repository, model aiapi.LanguageModel, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:  repo,
		model: model,
		log:   log,
	}
}

func (s *Service) ValidateCodeSubmission(ctx context.Context, req domain.ValidateCodeRequest) (*domain.CodeValidation, error) {
	startTime := time.Now()
	s.log.Info("starting code validation", zap.String("submission_id", req.SubmissionID.String()), zap.String("language", req.LanguageKey))

	userMsg := fmt.Sprintf("Language: %s\n\nCode to validate:\n```\n%s\n```", req.LanguageKey, req.Code)

	messages := []aiapi.Message{
		&aiapi.SystemMessage{
			Content: systemPrompt,
		},
		&aiapi.UserMessage{
			Content: []aiapi.ContentBlock{
				&aiapi.TextBlock{Text: userMsg},
			},
		},
	}

	responseTime := int(time.Since(startTime).Milliseconds())
	var validationResp domain.ValidateCodeResponse
	var apiError *string
	var tokensUsed *int

	s.log.Debug("calling AI API for code validation")
	resp, err := ai.GenerateText(
		ctx,
		messages,
		ai.WithModel(s.model),
		ai.WithMaxOutputTokens(1024),
	)

	if err != nil {
		s.log.Error("AI API call failed", zap.String("submission_id", req.SubmissionID.String()), zap.Error(err))
		errMsg := fmt.Sprintf("failed to call AI API: %v", err)
		apiError = &errMsg
		validationResp = domain.ValidateCodeResponse{
			IsAllowed: false,
			Severity:  domain.SeverityBlock,
			Reason:    "Failed to validate code",
			Details:   map[string]interface{}{"error": err.Error()},
		}
	} else {
		tokensUsed = &resp.Usage.OutputTokens
		s.log.Debug("AI API response received", zap.String("submission_id", req.SubmissionID.String()), zap.Int("output_tokens", resp.Usage.OutputTokens))

		respText := ""
		if len(resp.Content) > 0 {
			if textBlock, ok := resp.Content[0].(*aiapi.TextBlock); ok {
				respText = textBlock.Text
			}
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(respText), &parsed); err != nil {
			s.log.Warn("failed to parse AI response as JSON", zap.String("submission_id", req.SubmissionID.String()), zap.Error(err), zap.String("raw_response", respText))
			errMsg := fmt.Sprintf("failed to parse API response: %v", err)
			apiError = &errMsg
			validationResp = domain.ValidateCodeResponse{
				IsAllowed: false,
				Severity:  domain.SeverityBlock,
				Reason:    "Failed to parse validation response",
				Details:   map[string]interface{}{"raw_response": respText},
			}
		} else {
			s.log.Debug("parsed validation response", zap.String("submission_id", req.SubmissionID.String()), zap.Any("response", parsed))
			validationResp = s.parseValidationResponse(parsed)
			s.log.Info("code validation completed", zap.String("submission_id", req.SubmissionID.String()), zap.Bool("is_allowed", validationResp.IsAllowed), zap.String("severity", string(validationResp.Severity)))
		}
	}

	s.log.Debug("storing validation in database", zap.String("submission_id", req.SubmissionID.String()))
	validation, err := s.repo.CreateValidation(ctx, req, validationResp)
	if err != nil {
		s.log.Error("failed to create validation record", zap.String("submission_id", req.SubmissionID.String()), zap.Error(err))
		return nil, fmt.Errorf("create validation in db: %w", err)
	}

	_, _ = s.repo.LogValidationRequest(
		ctx,
		validation.ID,
		map[string]interface{}{"language": req.LanguageKey},
		map[string]interface{}{"is_allowed": validationResp.IsAllowed, "severity": validationResp.Severity},
		apiError,
		tokensUsed,
		&responseTime,
	)

	s.log.Debug("validation request logged", zap.String("submission_id", req.SubmissionID.String()), zap.Int("response_time_ms", responseTime))
	return validation, nil
}

func (s *Service) parseValidationResponse(data map[string]interface{}) domain.ValidateCodeResponse {
	resp := domain.ValidateCodeResponse{
		IsAllowed: false,
		Severity:  domain.SeverityWarn,
		Reason:    "Unable to determine",
		Details:   data,
	}

	if allowed, ok := data["is_allowed"].(bool); ok {
		resp.IsAllowed = allowed
	}

	if severity, ok := data["severity"].(string); ok {
		resp.Severity = domain.ValidationSeverity(severity)
	}

	if reason, ok := data["reason"].(string); ok {
		resp.Reason = reason
	}

	return resp
}

func (s *Service) GetValidationBySubmission(ctx context.Context, submissionID uuid.UUID) (*domain.CodeValidation, error) {
	s.log.Debug("retrieving validation for submission", zap.String("submission_id", submissionID.String()))
	validation, err := s.repo.GetValidationBySubmission(ctx, submissionID)
	if err != nil {
		s.log.Warn("failed to retrieve validation", zap.String("submission_id", submissionID.String()), zap.Error(err))
		return nil, err
	}
	return validation, nil
}
