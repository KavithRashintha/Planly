package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	ErrLLMUnavailable = errors.New("llm provider is unavailable")
	ErrLLMTokenLimit   = errors.New("llm token limit reached")
)

type LLMMessageRole string

const (
	RoleUser      LLMMessageRole = "user"
	RoleAssistant LLMMessageRole = "assistant"
)

type LLMContentBlock struct {
	Type      string          `json:"type"` // text | tool_use | tool_result
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`           // for tool_use
	Name      string          `json:"name,omitempty"`         // for tool_use
	Input     json.RawMessage `json:"input,omitempty"`        // for tool_use
	ToolUseID string          `json:"tool_use_id,omitempty"` // for tool_result
	Content   string          `json:"content,omitempty"`     // for tool_result
	IsError   bool            `json:"is_error,omitempty"`    // for tool_result
}

type LLMMessage struct {
	Role    LLMMessageRole    `json:"role"`
	Content []LLMContentBlock `json:"content"`
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type LLMRequest struct {
	System    string           `json:"system"`
	Messages  []LLMMessage     `json:"messages"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
	MaxTokens int              `json:"max_tokens,omitempty"`
}

type LLMToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type LLMResponse struct {
	Text         string        `json:"text"`
	ToolCalls    []LLMToolCall `json:"tool_calls,omitempty"`
	StopReason   string        `json:"stop_reason"` // end_turn | tool_use | max_tokens
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
}

// LLMClient is the interface for interacting with language models.
type LLMClient interface {
	Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}

// ----------------- Anthropic Client -----------------

type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: "https://api.anthropic.com/v1/messages",
	}
}

type anthropicRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []LLMMessage     `json:"messages"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Role       string            `json:"role"`
	Content    []LLMContentBlock `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      anthropicUsage    `json:"usage"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *AnthropicClient) Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("%w: API key is not configured", ErrLLMUnavailable)
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	bodyPayload := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages:  req.Messages,
		Tools:     req.Tools,
	}

	rawBody, err := json.Marshal(bodyPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		_ = json.Unmarshal(respBytes, &errResp)
		errMsg := string(respBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errMsg = errResp.Error.Message
		}
		return nil, fmt.Errorf("anthropic api error (status %d): %s", resp.StatusCode, errMsg)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBytes, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode anthropic response: %w", err)
	}

	result := &LLMResponse{
		StopReason:   anthropicResp.StopReason,
		InputTokens:  anthropicResp.Usage.InputTokens,
		OutputTokens: anthropicResp.Usage.OutputTokens,
	}

	for _, block := range anthropicResp.Content {
		switch block.Type {
		case "text":
			if result.Text == "" {
				result.Text = block.Text
			} else {
				result.Text += "\n" + block.Text
			}
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, LLMToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return result, nil
}

// ----------------- Fake LLM for Testing -----------------

type FakeLLMHandler func(ctx context.Context, req LLMRequest) (*LLMResponse, error)

type FakeLLM struct {
	mu            sync.Mutex
	Handler       FakeLLMHandler
	Responses     []*LLMResponse
	ReceivedReqs  []LLMRequest
	DefaultReply  string
}

func NewFakeLLM() *FakeLLM {
	return &FakeLLM{
		DefaultReply: "I have processed your request.",
	}
}

func (f *FakeLLM) QueueResponse(resp *LLMResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Responses = append(f.Responses, resp)
}

func (f *FakeLLM) Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ReceivedReqs = append(f.ReceivedReqs, req)

	if f.Handler != nil {
		return f.Handler(ctx, req)
	}

	if len(f.Responses) > 0 {
		resp := f.Responses[0]
		f.Responses = f.Responses[1:]
		return resp, nil
	}

	return &LLMResponse{
		Text:         f.DefaultReply,
		StopReason:   "end_turn",
		InputTokens:  50,
		OutputTokens: 20,
	}, nil
}

func (f *FakeLLM) LastRequest() *LLMRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ReceivedReqs) == 0 {
		return nil
	}
	return &f.ReceivedReqs[len(f.ReceivedReqs)-1]
}
