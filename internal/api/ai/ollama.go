package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	aiapi "go.jetify.com/ai/api"
)

type OllamaModel struct {
	baseURL   string
	modelID   string
	httpClient *http.Client
}

func NewOllamaModel(baseURL, modelID string) *OllamaModel {
	return &OllamaModel{
		baseURL:    baseURL,
		modelID:    modelID,
		httpClient: &http.Client{},
	}
}

func (m *OllamaModel) ProviderName() string {
	return "ollama"
}

func (m *OllamaModel) ModelID() string {
	return m.modelID
}

func (m *OllamaModel) SupportedUrls() []aiapi.SupportedURL {
	return nil
}

func (m *OllamaModel) Generate(ctx context.Context, messages []aiapi.Message, opts aiapi.CallOptions) (*aiapi.Response, error) {
	req := m.buildRequest(messages, opts)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(bodyBytes))
	}

	if len(ollamaResp.Choices) == 0 || ollamaResp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("ollama returned empty content (full response: %s)", string(bodyBytes))
	}

	return &aiapi.Response{
		Content: []aiapi.ContentBlock{
			&aiapi.TextBlock{Text: ollamaResp.Choices[0].Message.Content},
		},
		Usage: aiapi.Usage{
			InputTokens:  ollamaResp.Usage.PromptTokens,
			OutputTokens: ollamaResp.Usage.CompletionTokens,
		},
	}, nil
}

func (m *OllamaModel) Stream(ctx context.Context, messages []aiapi.Message, opts aiapi.CallOptions) (*aiapi.StreamResponse, error) {
	return nil, fmt.Errorf("streaming not implemented for ollama")
}

func (m *OllamaModel) buildRequest(messages []aiapi.Message, opts aiapi.CallOptions) ollamaChatRequest {
	msgs := make([]ollamaMessage, 0)

	for _, msg := range messages {
		switch m := msg.(type) {
		case *aiapi.SystemMessage:
			msgs = append(msgs, ollamaMessage{
				Role:    "system",
				Content: m.Content,
			})
		case *aiapi.UserMessage:
			content := ""
			for _, block := range m.Content {
				if tb, ok := block.(*aiapi.TextBlock); ok {
					content += tb.Text
				}
			}
			msgs = append(msgs, ollamaMessage{
				Role:    "user",
				Content: content,
			})
		case *aiapi.AssistantMessage:
			content := ""
			for _, block := range m.Content {
				if tb, ok := block.(*aiapi.TextBlock); ok {
					content += tb.Text
				}
			}
			msgs = append(msgs, ollamaMessage{
				Role:    "assistant",
				Content: content,
			})
		}
	}

	maxTokens := 1024
	if opts.MaxOutputTokens > 0 {
		maxTokens = opts.MaxOutputTokens
	}

	return ollamaChatRequest{
		Model:     m.modelID,
		Messages:  msgs,
		Stream:    false,
		NumPredict: maxTokens,
	}
}

type ollamaChatRequest struct {
	Model      string         `json:"model"`
	Messages   []ollamaMessage `json:"messages"`
	Stream     bool           `json:"stream"`
	NumPredict int            `json:"num_predict"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Choices []ollamaChoice `json:"choices"`
	Usage   ollamaUsage    `json:"usage"`
}

type ollamaChoice struct {
	Message ollamaMessage `json:"message"`
}

type ollamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
