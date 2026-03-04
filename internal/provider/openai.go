//go:build js && wasm

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// OpenAIProvider implements the Provider interface for OpenAI's Chat Completions API
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	orgID   string // Optional organization ID
}

// NewOpenAIProvider creates a new OpenAI provider
// Note: OpenAI does not support direct browser CORS, users need a proxy
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
	}
}

// NewOpenAIProviderWithOrg creates an OpenAI provider with organization ID
func NewOpenAIProviderWithOrg(apiKey, orgID string) *OpenAIProvider {
	p := NewOpenAIProvider(apiKey)
	p.orgID = orgID
	return p
}

// Name returns the provider identifier
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// MaxContextWindow returns the maximum context window for the model
func (p *OpenAIProvider) MaxContextWindow(model string) int {
	// OpenAI model context windows
	if strings.Contains(model, "gpt-4o") {
		return 128000
	}
	if strings.Contains(model, "gpt-4-turbo") || strings.Contains(model, "gpt-4-0125") {
		return 128000
	}
	if strings.Contains(model, "gpt-4-32k") {
		return 32768
	}
	if strings.Contains(model, "gpt-4") {
		return 8192
	}
	if strings.Contains(model, "gpt-3.5-turbo-16k") {
		return 16384
	}
	if strings.Contains(model, "gpt-3.5-turbo") {
		return 4096
	}
	if strings.Contains(model, "text-embedding") {
		return 8191 // Embeddings have input limits, not context windows
	}
	return 4096 // Conservative default
}

// openAIRequest represents the request body for OpenAI Chat Completions API
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream"`
	User        string          `json:"user,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"` // Tool definitions for LLM
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Name      string           `json:"name,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"` // For streaming delta
}

// openAIToolCall represents a tool call in streaming delta
type openAIToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIToolFunction `json:"function,omitempty"`
}

type openAIToolFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// openAIResponse represents a non-streaming response
type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openAIStreamResponse represents a streaming chunk
type openAIStreamResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
}

type openAIStreamChoice struct {
	Index        int           `json:"index"`
	Delta        openAIMessage `json:"delta"`
	FinishReason string        `json:"finish_reason"`
}

// openAIEmbeddingRequest represents the request for embeddings
type openAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	User  string `json:"user,omitempty"`
}

// openAIEmbeddingResponse represents the response for embeddings
type openAIEmbeddingResponse struct {
	Object string               `json:"object"`
	Data   []openAIEmbedding    `json:"data"`
	Model  string               `json:"model"`
	Usage  openAIEmbeddingUsage `json:"usage"`
}

type openAIEmbedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type openAIEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// convertToOpenAITools converts generic tool definitions to OpenAI format
func convertToOpenAITools(tools []map[string]interface{}) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openAITool, len(tools))
	for i, t := range tools {
		result[i] = openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        getString(t, "name"),
				Description: getString(t, "description"),
				Parameters:  getMap(t, "input_schema"),
			},
		}
	}
	return result
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getMap safely extracts a map value from a map
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]interface{}); ok {
			return mm
		}
	}
	return nil
}

// Complete performs a non-streaming completion
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	if p.apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Log API call (without key)
	js.Global().Get("console").Call("log", "[OpenAI] API call: model=", req.Model)

	// Convert messages to OpenAI format
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// Build request
	openAIReq := openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       convertToOpenAITools(req.Tools), // Convert and pass tool definitions
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request via JS fetch bridge
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
	if p.orgID != "" {
		headers["OpenAI-Organization"] = p.orgID
	}

	opts := jsbridge.FetchOptions{
		Method:  "POST",
		Headers: headers,
		Body:    string(body),
	}

	resp, err := jsbridge.Fetch(p.baseURL+"/chat/completions", opts)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Handle errors
	if resp.Status >= 400 {
		js.Global().Get("console").Call("error", "[OpenAI] API error: status=", resp.Status)
		return nil, p.handleError(resp.Status, resp.Body)
	}

	// Log response metadata
	js.Global().Get("console").Call("log", "[OpenAI] API response: status=", resp.Status, "len=", len(resp.Body))

	// Parse response
	var openAIResp openAIResponse
	if err := json.Unmarshal(resp.Body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	return &Token{
		Text:         openAIResp.Choices[0].Message.Content,
		FinishReason: openAIResp.Choices[0].FinishReason,
	}, nil
}

// Stream performs a streaming completion
func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	tokenChan := make(chan Token, 10)

	go func() {
		defer close(tokenChan)

		if p.apiKey == "" {
			tokenChan <- Token{FinishReason: "error", Text: ErrAPIKeyMissing.Error()}
			return
		}

		// Log streaming request (without key)
		js.Global().Get("console").Call("log", "[OpenAI] Stream: model=", req.Model, "messages=", len(req.Messages))

		// Convert messages
		messages := make([]openAIMessage, len(req.Messages))
		for i, m := range req.Messages {
			messages[i] = openAIMessage{
				Role:    m.Role,
				Content: m.Content,
			}
		}

		// Build streaming request
		openAIReq := openAIRequest{
			Model:       req.Model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Tools:       convertToOpenAITools(req.Tools), // Convert and pass tool definitions
		}

		body, err := json.Marshal(openAIReq)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("marshal error: %v", err)}
			return
		}

		// Initiate streaming fetch
		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + p.apiKey,
			"Accept":        "text/event-stream",
		}
		if p.orgID != "" {
			headers["OpenAI-Organization"] = p.orgID
		}

		opts := jsbridge.FetchOptions{
			Method:  "POST",
			Headers: headers,
			Body:    string(body),
		}

		response, err := jsbridge.FetchStream(p.baseURL+"/chat/completions", opts)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("stream error: %v", err)}
			return
		}

		// Check for immediate error status
		status := response.Get("status").Int()
		if status >= 400 {
			// Try to get error body - text() returns a Promise
			textPromise := response.Call("text")
			if !textPromise.IsUndefined() && !textPromise.IsNull() {
				textPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					if len(args) > 0 {
						js.Global().Get("console").Call("error", "[OpenAI] Error response body:", args[0].String())
					}
					return nil
				}))
			}
			js.Global().Get("console").Call("error", "[OpenAI] Stream error: status=", status)
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("HTTP %d", status)}
			return
		}

		// Log successful stream initiation
		js.Global().Get("console").Call("log", "[OpenAI] Stream started: status=", status)

		// Create SSE reader
		sseReader := jsbridge.NewSSEStreamingReader(response)
		events := sseReader.Events()

		// Track tool calls during streaming
		toolCalls := make(map[int]*struct {
			ID        string
			Name      string
			Arguments strings.Builder
		})

		// Process SSE events
		for event := range events {
			if event.Event == "error" {
				tokenChan <- Token{FinishReason: "error", Text: event.Data}
				return
			}

			if event.Data == "" {
				continue
			}

			// OpenAI SSE events start with "data: " prefix
			// The event.Data field from SSEParser already strips the "data: " prefix

			// Check for [DONE] terminator
			if event.Data == "[DONE]" {
				return
			}

			// Parse the JSON chunk
			var streamResp openAIStreamResponse
			if err := json.Unmarshal([]byte(event.Data), &streamResp); err != nil {
				// Skip unparsable chunks but continue
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]

			// Handle tool_calls in delta
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if _, exists := toolCalls[tc.Index]; !exists {
						toolCalls[tc.Index] = &struct {
							ID        string
							Name      string
							Arguments strings.Builder
						}{
							ID:   tc.ID,
							Name: tc.Function.Name,
						}
					}
					if tc.Function.Arguments != "" {
						toolCalls[tc.Index].Arguments.WriteString(tc.Function.Arguments)
					}
				}
			}

			// Send content delta
			if choice.Delta.Content != "" {
				tokenChan <- Token{
					Text: choice.Delta.Content,
				}
			}

			// Check for finish reason
			if choice.FinishReason != "" {
				if choice.FinishReason == "tool_calls" {
					// Process first tool call (OpenAI may return multiple)
					for _, tc := range toolCalls {
						var args map[string]interface{}
						if err := json.Unmarshal([]byte(tc.Arguments.String()), &args); err != nil {
							js.Global().Get("console").Call("error", "[OpenAI] Failed to parse tool arguments:", err.Error())
							args = make(map[string]interface{}) // Use empty map on error
						}

						js.Global().Get("console").Call("log", "[OpenAI] Tool call:", tc.Name)
						tokenChan <- Token{
							FinishReason: "tool_use",
							ToolName:     tc.Name,
							ToolInput:    args,
							ToolUseID:    tc.ID,
						}
						break // Handle one at a time for now
					}
				} else {
					tokenChan <- Token{
						Text:         "",
						FinishReason: choice.FinishReason,
					}
				}
				return
			}
		}
	}()

	return tokenChan
}

// Embed creates embeddings for the given input
func (p *OpenAIProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Default to text-embedding-3-small for embeddings
	model := "text-embedding-3-small"

	req := openAIEmbeddingRequest{
		Model: model,
		Input: input,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
	if p.orgID != "" {
		headers["OpenAI-Organization"] = p.orgID
	}

	opts := jsbridge.FetchOptions{
		Method:  "POST",
		Headers: headers,
		Body:    string(body),
	}

	resp, err := jsbridge.Fetch(p.baseURL+"/embeddings", opts)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	if resp.Status >= 400 {
		return nil, p.handleError(resp.Status, resp.Body)
	}

	var embeddingResp openAIEmbeddingResponse
	if err := json.Unmarshal(resp.Body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, errors.New("no embeddings in response")
	}

	return embeddingResp.Data[0].Embedding, nil
}

func (p *OpenAIProvider) handleError(status int, body []byte) error {
	// Try to parse error response
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	message := string(body)
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		message = errResp.Error.Message
	}

	switch status {
	case 401:
		return ErrAPIKeyMissing
	case 429:
		return ErrRateLimit
	case 500, 502, 503, 529:
		return ErrServerError
	default:
		return fmt.Errorf("API error %d: %s", status, message)
	}
}

// compile check
var _ Provider = (*OpenAIProvider)(nil)
