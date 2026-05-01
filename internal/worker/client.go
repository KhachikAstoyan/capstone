package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/google/uuid"
)

// ControlPlaneClient is a thin HTTP wrapper around the control-plane API.
// It is the worker's only way to talk to the control plane.
type ControlPlaneClient struct {
	baseURL    string       // e.g. "http://localhost:9090"
	httpClient *http.Client // shared client with sane timeouts
	key        string       // value for X-Internal-Key header
}

// NewControlPlaneClient creates a client.  If key is empty, the
// X-Internal-Key header is omitted (dev mode).
func NewControlPlaneClient(baseURL, key string) *ControlPlaneClient {
	return &ControlPlaneClient{
		baseURL: baseURL,
		key:     key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

// Heartbeat registers or refreshes the worker's state.
func (c *ControlPlaneClient) Heartbeat(ctx context.Context, req domain.HeartbeatRequest) (*domain.Worker, error) {
	var w domain.Worker
	if err := c.post(ctx, "/v1/workers/heartbeat", req, &w); err != nil {
		return nil, fmt.Errorf("heartbeat: %w", err)
	}
	return &w, nil
}

// ---------------------------------------------------------------------------
// Poll
// ---------------------------------------------------------------------------

// Poll asks the control plane for the next available job.
// Returns (nil, nil) when there is no work (HTTP 204).
func (c *ControlPlaneClient) Poll(ctx context.Context, req domain.PollRequest) (*domain.Assignment, error) {
	var a domain.Assignment
	err := c.post(ctx, "/v1/workers/poll", req, &a)
	if err != nil {
		if err == errNoContent {
			return nil, nil // no work available
		}
		return nil, fmt.Errorf("poll: %w", err)
	}
	return &a, nil
}

// ---------------------------------------------------------------------------
// MarkRunning
// ---------------------------------------------------------------------------

// MarkRunning tells the control plane that execution has started.
func (c *ControlPlaneClient) MarkRunning(ctx context.Context, jobID uuid.UUID, workerID string) error {
	body := struct {
		WorkerID string `json:"worker_id"`
	}{WorkerID: workerID}
	return c.post(ctx, fmt.Sprintf("/v1/jobs/%s/running", jobID), body, nil)
}

// ---------------------------------------------------------------------------
// RenewLease
// ---------------------------------------------------------------------------

// RenewLease extends the lease on a job.
func (c *ControlPlaneClient) RenewLease(ctx context.Context, jobID uuid.UUID, workerID string) error {
	req := domain.RenewLeaseRequest{WorkerID: workerID}
	return c.post(ctx, fmt.Sprintf("/v1/jobs/%s/lease", jobID), req, nil)
}

// ---------------------------------------------------------------------------
// ReportResult
// ---------------------------------------------------------------------------

// ReportResult sends the final execution outcome.
func (c *ControlPlaneClient) ReportResult(ctx context.Context, jobID uuid.UUID, req domain.ReportResultRequest) error {
	return c.post(ctx, fmt.Sprintf("/v1/jobs/%s/result", jobID), req, nil)
}

// ---------------------------------------------------------------------------
// Internal HTTP helpers
// ---------------------------------------------------------------------------

// errNoContent is a sentinel for HTTP 204 responses.
var errNoContent = fmt.Errorf("no content")

// post sends a JSON POST request and decodes the response into dst.
// If dst is nil, the response body is discarded.
func (c *ControlPlaneClient) post(ctx context.Context, path string, body any, dst any) error {
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

	// 204 No Content is a successful empty response for fire-and-forget
	// endpoints such as mark-running, renew-lease, and report-result. Poll is
	// the only caller that passes a destination and treats 204 as "no work".
	if resp.StatusCode == http.StatusNoContent {
		if dst == nil {
			return nil
		}
		return errNoContent
	}

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
