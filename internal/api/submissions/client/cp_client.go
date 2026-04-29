package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	cpdomain "github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
)

type Client interface {
	CreateJob(ctx context.Context, req cpdomain.CreateJobRequest) (*cpdomain.Job, error)
	GetJobBySubmission(ctx context.Context, submissionID uuid.UUID) (*cpdomain.Job, error)
	GetJobResult(ctx context.Context, jobID uuid.UUID) (*cpdomain.JobResult, error)
}

type CPClient struct {
	baseURL    string
	httpClient *http.Client
	key        string
}

func NewCPClient(baseURL, key string) *CPClient {
	return &CPClient{
		baseURL: baseURL,
		key:     key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *CPClient) CreateJob(ctx context.Context, req cpdomain.CreateJobRequest) (*cpdomain.Job, error) {
	var job cpdomain.Job
	if err := c.post(ctx, "/v1/jobs", req, &job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return &job, nil
}

func (c *CPClient) GetJobBySubmission(ctx context.Context, submissionID uuid.UUID) (*cpdomain.Job, error) {
	var job cpdomain.Job
	if err := c.get(ctx, fmt.Sprintf("/v1/jobs/by-submission/%s", submissionID), &job); err != nil {
		return nil, fmt.Errorf("get job by submission: %w", err)
	}
	return &job, nil
}

func (c *CPClient) GetJobResult(ctx context.Context, jobID uuid.UUID) (*cpdomain.JobResult, error) {
	var result cpdomain.JobResult
	if err := c.get(ctx, fmt.Sprintf("/v1/jobs/%s/result", jobID), &result); err != nil {
		return nil, fmt.Errorf("get job result: %w", err)
	}
	return &result, nil
}

func (c *CPClient) post(ctx context.Context, path string, body any, dst any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("X-Internal-Key", c.key)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("control-plane returned %d: %s", resp.StatusCode, string(respBody))
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *CPClient) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if c.key != "" {
		req.Header.Set("X-Internal-Key", c.key)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("control-plane returned %d: %s", resp.StatusCode, string(respBody))
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
