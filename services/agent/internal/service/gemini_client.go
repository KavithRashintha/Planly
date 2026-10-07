package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GeminiClient implements LLMClient using Google Gemini API.
// It translates between our internal LLMRequest/LLMResponse and Gemini's generateContent format.
// Gemini's function-calling (tool use) is directly supported.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// ---- Gemini REST request/response types ----

type geminiPart struct {
	Type             string             `json:"type,omitempty"`
	Thought          bool               `json:"thought,omitempty"`
	ThoughtSignature string             `json:"thoughtSignature,omitempty"` // required for multi-turn with thinking models
	Text             string             `json:"text,omitempty"`
	FunctionCall     *geminiFuncCall    `json:"functionCall,omitempty"`
	FunctionResp     *geminiFuncResp    `json:"functionResponse,omitempty"`
	InlineData       *geminiInlineData  `json:"inlineData,omitempty"`
}

type geminiFuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type geminiFuncResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // user | model
	Parts []geminiPart `json:"parts"`
}

type geminiToolFuncDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiToolFuncDecl `json:"functionDeclarations"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     float64               `json:"temperature,omitempty"`
	MaxOutputTokens int                   `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
	Error         *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// Generate sends a request to the Gemini API and returns the structured LLMResponse.
func (c *GeminiClient) Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("%w: Gemini API key is not configured", ErrLLMUnavailable)
	}

	// Build Gemini contents from LLMMessages
	contents, err := convertMessagesToGemini(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages to Gemini format: %w", err)
	}

	gemReq := geminiRequest{
		Contents: contents,
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: 8192,
			Temperature:     0.7,
		},
	}

	// Add system instruction
	if req.System != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		decls := make([]geminiToolFuncDecl, 0, len(req.Tools))
		for _, t := range req.Tools {
			params, convErr := convertSchemaToGemini(t.InputSchema)
			if convErr != nil {
				params = t.InputSchema // fall back to raw schema
			}
			decls = append(decls, geminiToolFuncDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			})
		}
		gemReq.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	rawBody, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		c.model, c.apiKey,
	)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp geminiResponse
		_ = json.Unmarshal(respBytes, &errResp)
		errMsg := string(respBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errMsg = errResp.Error.Message
		}
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, errMsg)
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBytes, &gemResp); err != nil {
		return nil, fmt.Errorf("failed to decode Gemini response: %w", err)
	}

	if gemResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", gemResp.Error.Message)
	}

	if len(gemResp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	candidate := gemResp.Candidates[0]
	result := &LLMResponse{
		StopReason:   mapGeminiFinishReason(candidate.FinishReason),
		InputTokens:  gemResp.UsageMetadata.PromptTokenCount,
		OutputTokens: gemResp.UsageMetadata.CandidatesTokenCount,
	}

	// Parse candidate content parts
	// Note: gemini-3.8-flash (thinking model) emits "thought" typed parts for
	// internal reasoning — these must be skipped, only text and functionCall matter.
	// Collect candidate-level thought signature if available
	var defaultThoughtSig string
	for _, part := range candidate.Content.Parts {
		if part.ThoughtSignature != "" {
			defaultThoughtSig = part.ThoughtSignature
		}
	}

	var toolCallID int
	for _, part := range candidate.Content.Parts {
		// Skip internal thought/reasoning blocks emitted by thinking models
		if part.Thought || part.Type == "thought" {
			continue
		}
		if part.FunctionCall != nil {
			toolCallID++
			sig := part.ThoughtSignature
			if sig == "" {
				sig = defaultThoughtSig
			}
			result.ToolCalls = append(result.ToolCalls, LLMToolCall{
				ID:               fmt.Sprintf("gemini_tool_%d", toolCallID),
				Name:             part.FunctionCall.Name,
				Input:            part.FunctionCall.Args,
				ThoughtSignature: sig,
			})
		} else if part.Text != "" {
			if result.Text == "" {
				result.Text = part.Text
			} else {
				result.Text += "\n" + part.Text
			}
		}
	}

	return result, nil
}

// convertMessagesToGemini converts our internal LLMMessages to Gemini contents.
// Gemini uses role "user" / "model" and has its own content block format.
func convertMessagesToGemini(messages []LLMMessage) ([]geminiContent, error) {
	var contents []geminiContent

	for _, msg := range messages {
		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}

		var parts []geminiPart
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				parts = append(parts, geminiPart{Text: block.Text})

			case "tool_use":
				// Assistant is calling a tool → functionCall part
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFuncCall{
						Name: block.Name,
						Args: block.Input,
					},
					ThoughtSignature: block.ThoughtSignature,
				})

			case "tool_result":
				// Tool result → functionResponse part (role must be "user")
				// Gemini expects the response to be a JSON object
				var responseRaw json.RawMessage
				if block.Content != "" {
					responseRaw = json.RawMessage(block.Content)
				} else {
					responseRaw = json.RawMessage(`{}`)
				}
				// Wrap in {"output": ...} if not already an object
				if !isJSONObject(responseRaw) {
					wrapped, _ := json.Marshal(map[string]any{"output": string(responseRaw)})
					responseRaw = wrapped
				}
				funcName := block.Name
				if funcName == "" {
					funcName = block.ToolUseID
				}
				parts = append(parts, geminiPart{
					FunctionResp: &geminiFuncResp{
						Name:     funcName,
						Response: responseRaw,
					},
				})
			}
		}

		if len(parts) == 0 {
			continue
		}

		// Gemini requires alternating user/model turns.
		// If same role back-to-back, merge into previous content.
		if len(contents) > 0 && contents[len(contents)-1].Role == role {
			contents[len(contents)-1].Parts = append(contents[len(contents)-1].Parts, parts...)
		} else {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	// Fix up functionResponse names: tool_use_id doesn't map to function name in Gemini.
	// We need to resolve the name from the preceding functionCall parts.
	fixFunctionResponseNames(contents)

	return contents, nil
}

// fixFunctionResponseNames resolves tool_use_id references in functionResponse blocks
// by matching them against functionCall IDs from assistant turns.
func fixFunctionResponseNames(contents []geminiContent) {
	// Build a map of our synthetic IDs to function names
	// Our IDs are like "gemini_tool_1" and we stored the tool_use_id from Anthropic format
	// In practice, we need to look at the preceding model turn's functionCall parts.
	// Since we generated IDs like "gemini_tool_N", they won't match tool_use_ids from DB.
	// Instead, match by order: the Nth functionResponse corresponds to the Nth functionCall.

	// Collect all function call names in order from model turns
	var funcNames []string
	for _, content := range contents {
		if content.Role == "model" {
			for _, part := range content.Parts {
				if part.FunctionCall != nil {
					funcNames = append(funcNames, part.FunctionCall.Name)
				}
			}
		}
	}

	// Assign names to functionResponse parts in order
	nameIdx := 0
	for i := range contents {
		if contents[i].Role == "user" {
			for j := range contents[i].Parts {
				if contents[i].Parts[j].FunctionResp != nil {
					name := contents[i].Parts[j].FunctionResp.Name
					if (name == "" || strings.HasPrefix(name, "gemini_tool_") || strings.HasPrefix(name, "toolu_")) && nameIdx < len(funcNames) {
						contents[i].Parts[j].FunctionResp.Name = funcNames[nameIdx]
					}
					nameIdx++
				}
			}
		}
	}
}

// convertSchemaToGemini converts Anthropic-style JSON schema to Gemini-compatible parameters.
// Gemini uses uppercase TYPE values (OBJECT, STRING, INTEGER, etc.) in its schema.
func convertSchemaToGemini(schema json.RawMessage) (json.RawMessage, error) {
	if len(schema) == 0 {
		return json.RawMessage(`{"type": "OBJECT", "properties": {}}`), nil
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return nil, err
	}

	converted := convertSchemaNode(schemaMap)
	result, err := json.Marshal(converted)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// convertSchemaNode recursively converts JSON schema nodes to Gemini format.
// Main difference: type values must be uppercased.
func convertSchemaNode(node map[string]any) map[string]any {
	result := make(map[string]any)

	for k, v := range node {
		switch k {
		case "type":
			// Gemini requires uppercase type values
			if typeStr, ok := v.(string); ok {
				result["type"] = strings.ToUpper(typeStr)
			} else {
				result[k] = v
			}
		case "properties":
			if props, ok := v.(map[string]any); ok {
				convertedProps := make(map[string]any)
				for propName, propVal := range props {
					if propMap, ok := propVal.(map[string]any); ok {
						convertedProps[propName] = convertSchemaNode(propMap)
					} else {
						convertedProps[propName] = propVal
					}
				}
				result["properties"] = convertedProps
			} else {
				result[k] = v
			}
		case "items":
			if itemMap, ok := v.(map[string]any); ok {
				result["items"] = convertSchemaNode(itemMap)
			} else {
				result[k] = v
			}
		case "enum":
			// Gemini uses "format" for enum sometimes but also supports enum directly
			result[k] = v
		default:
			result[k] = v
		}
	}

	return result
}

// mapGeminiFinishReason maps Gemini finish reasons to our internal stop reason strings.
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "FUNCTION_CALL":
		return "tool_use"
	default:
		return reason
	}
}

// isJSONObject checks if a raw JSON value is an object (starts with {).
func isJSONObject(raw json.RawMessage) bool {
	for _, b := range raw {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		return b == '{'
	}
	return false
}
