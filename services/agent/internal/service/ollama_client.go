package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient implements LLMClient using Ollama's OpenAI-compatible REST API.
// No API key required — Ollama runs fully locally.
// Supports tool/function calling with models like qwen2.5, llama3.1, mistral-nemo.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://ollama:11434"
	}
	if model == "" {
		model = "qwen2.5:7b"
	}
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			// Local LLM inference can be slow — allow generous timeout
			Timeout: 120 * time.Second,
		},
	}
}

// ---- OpenAI-compatible request/response types (used by Ollama) ----

type ollamaMessage struct {
	Role       string           `json:"role"`                  // system | user | assistant | tool
	Content    string           `json:"content,omitempty"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // for role=tool
	Name       string           `json:"name,omitempty"`         // for role=tool
}

type ollamaToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"` // always "function"
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ollamaTool struct {
	Type     string             `json:"type"` // "function"
	Function ollamaFunctionDef  `json:"function"`
}

type ollamaFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaChoice struct {
	Index        int           `json:"index"`
	Message      ollamaMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type ollamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ollamaResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Choices []ollamaChoice `json:"choices"`
	Usage   ollamaUsage    `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Generate sends a request to Ollama's OpenAI-compatible /v1/chat/completions endpoint.
func (c *OllamaClient) Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	// Build messages: system goes first as a system-role message
	var messages []ollamaMessage
	if req.System != "" {
		messages = append(messages, ollamaMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	// Convert LLMMessages → Ollama messages
	for _, msg := range req.Messages {
		converted, err := convertLLMMessageToOllama(msg)
		if err != nil {
			return nil, fmt.Errorf("message conversion error: %w", err)
		}
		messages = append(messages, converted...)
	}

	// Convert tool definitions
	var tools []ollamaTool
	for _, t := range req.Tools {
		tools = append(tools, ollamaTool{
			Type: "function",
			Function: ollamaFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	ollamaReq := ollamaRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		Options: map[string]any{
			"temperature": 0.7,
			"num_predict": 4096,
		},
	}

	rawBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Ollama's OpenAI endpoint accepts but doesn't require an API key
	httpReq.Header.Set("Authorization", "Bearer ollama")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: Ollama may not be running — %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ollamaResponse
		_ = json.Unmarshal(respBytes, &errResp)
		errMsg := string(respBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errMsg = errResp.Error.Message
		}
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, errMsg)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	if ollamaResp.Error != nil {
		return nil, fmt.Errorf("ollama error: %s", ollamaResp.Error.Message)
	}

	if len(ollamaResp.Choices) == 0 {
		return nil, fmt.Errorf("ollama returned no choices")
	}

	choice := ollamaResp.Choices[0]
	result := &LLMResponse{
		StopReason:   choice.FinishReason,
		InputTokens:  ollamaResp.Usage.PromptTokens,
		OutputTokens: ollamaResp.Usage.CompletionTokens,
	}

	// Map finish_reason to our internal values
	switch result.StopReason {
	case "stop":
		result.StopReason = "end_turn"
	case "tool_calls":
		result.StopReason = "tool_use"
	case "length":
		result.StopReason = "max_tokens"
	}

	// Extract text and tool calls from the response message
	result.Text = choice.Message.Content
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, LLMToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: tc.Function.Arguments,
		})
	}

	return result, nil
}

// convertLLMMessageToOllama converts one LLMMessage (which may have multiple content blocks)
// into one or more Ollama messages, since Ollama uses a flatter message format.
func convertLLMMessageToOllama(msg LLMMessage) ([]ollamaMessage, error) {
	var result []ollamaMessage

	role := string(msg.Role)

	// Collect text content and tool-related blocks separately
	var textParts []string
	var toolCalls []ollamaToolCall
	var toolResults []ollamaMessage

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}

		case "tool_use":
			// Assistant calling a tool
			toolCalls = append(toolCalls, ollamaToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ollamaFunctionCall{
					Name:      block.Name,
					Arguments: block.Input,
				},
			})

		case "tool_result":
			// Tool result — becomes a "tool" role message
			content := block.Content
			if content == "" {
				content = "{}"
			}
			toolResults = append(toolResults, ollamaMessage{
				Role:       "tool",
				ToolCallID: block.ToolUseID,
				Content:    content,
			})
		}
	}

	// Build the primary message
	if len(toolCalls) > 0 {
		// Assistant message with tool calls
		primary := ollamaMessage{
			Role:      role,
			ToolCalls: toolCalls,
		}
		if len(textParts) > 0 {
			primary.Content = joinStrings(textParts, "\n")
		}
		result = append(result, primary)
	} else if len(toolResults) > 0 {
		// Tool results (user turn containing tool_result blocks)
		result = append(result, toolResults...)
	} else {
		// Normal text message
		content := joinStrings(textParts, "\n")
		result = append(result, ollamaMessage{
			Role:    role,
			Content: content,
		})
	}

	return result, nil
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}
