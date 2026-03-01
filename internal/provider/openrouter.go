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

// OpenRouterProvider implements the Provider interface for OpenRouter API
// OpenRouter uses an OpenAI-compatible format with additional headers
type OpenRouterProvider struct {
	apiKey      string
	baseURL     string
	httpReferer string // Required by OpenRouter
	xTitle      string // Required by OpenRouter
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(apiKey, httpReferer, xTitle string) *OpenRouterProvider {
	return &OpenRouterProvider{
		apiKey:      apiKey,
		baseURL:     "https://openrouter.ai/api/v1",
		httpReferer: httpReferer,
		xTitle:      xTitle,
	}
}

// Name returns the provider identifier
func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

// MaxContextWindow returns the maximum context window for the model
func (p *OpenRouterProvider) MaxContextWindow(model string) int {
	// OpenRouter supports many models - use context sizes from underlying providers
	// These are common model patterns routed through OpenRouter

	// Anthropic models via OpenRouter
	if strings.Contains(model, "claude-3-opus") {
		return 200000
	}
	if strings.Contains(model, "claude-3-sonnet") || strings.Contains(model, "claude-sonnet") {
		return 200000
	}
	if strings.Contains(model, "claude-3-haiku") {
		return 200000
	}

	// OpenAI models via OpenRouter
	if strings.Contains(model, "gpt-4o") {
		return 128000
	}
	if strings.Contains(model, "gpt-4-turbo") {
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

	// Google models via OpenRouter
	if strings.Contains(model, "gemini-pro-vision") {
		return 16384
	}
	if strings.Contains(model, "gemini-pro") {
		return 32768
	}
	if strings.Contains(model, "gemini-ultra") {
		return 32768
	}

	// Meta models via OpenRouter
	if strings.Contains(model, "llama-2") && strings.Contains(model, "70b") {
		return 4096
	}
	if strings.Contains(model, "llama-2") {
		return 4096
	}
	if strings.Contains(model, "llama-3") && strings.Contains(model, "70b") {
		return 8192
	}
	if strings.Contains(model, "llama-3") {
		return 8192
	}

	// Mistral models via OpenRouter
	if strings.Contains(model, "mistral-large") || strings.Contains(model, "mistral-medium") {
		return 32768
	}
	if strings.Contains(model, "mixtral") {
		return 32768
	}

	// Default
	return 4096
}

// openRouterRequest represents the request body for OpenRouter API
// Uses OpenAI-compatible format
type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
	Stream      bool                `json:"stream"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// openRouterResponse represents a non-streaming response
type openRouterResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openRouterChoice `json:"choices"`
	Usage   openRouterUsage    `json:"usage"`
}

type openRouterChoice struct {
	Index        int               `json:"index"`
	Message      openRouterMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openRouterStreamResponse represents a streaming chunk
type openRouterStreamResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []openRouterStreamChoice `json:"choices"`
}

type openRouterStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openRouterMessage `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

// openRouterError represents an error response from OpenRouter
type openRouterError struct {
	Error struct {
		Message  string `json:"message"`
		Type     string `json:"type"`
		Code     int    `json:"code"`
		Metadata struct {
			ProviderName string `json:"provider_name"`
			RawError     string `json:"raw"`
		} `json:"metadata"`
	} `json:"error"`
}

// Complete performs a non-streaming completion
func (p *OpenRouterProvider) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	if p.apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Log API call (without key)
	js.Global().Get("console").Call("log", "[OpenRouter] API call: model=", req.Model)

	// Convert messages to OpenRouter format (same as OpenAI)
	messages := make([]openRouterMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openRouterMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// Build request
	openRouterReq := openRouterRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	body, err := json.Marshal(openRouterReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request via JS fetch bridge
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
		"HTTP-Referer":  p.httpReferer,
		"X-Title":       p.xTitle,
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
		js.Global().Get("console").Call("error", "[OpenRouter] API error: status=", resp.Status)
		return nil, p.handleError(resp.Status, resp.Body)
	}

	// Log response metadata
	js.Global().Get("console").Call("log", "[OpenRouter] API response: status=", resp.Status, "len=", len(resp.Body))

	// Parse response
	var openRouterResp openRouterResponse
	if err := json.Unmarshal(resp.Body, &openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	return &Token{
		Text:         openRouterResp.Choices[0].Message.Content,
		FinishReason: openRouterResp.Choices[0].FinishReason,
	}, nil
}

// Stream performs a streaming completion
func (p *OpenRouterProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	tokenChan := make(chan Token, 10)

	go func() {
		defer close(tokenChan)

		if p.apiKey == "" {
			tokenChan <- Token{FinishReason: "error", Text: ErrAPIKeyMissing.Error()}
			return
		}

		// Log streaming request (without key)
		js.Global().Get("console").Call("log", "[OpenRouter] Stream: model=", req.Model, "messages=", len(req.Messages))

		// Convert messages
		messages := make([]openRouterMessage, len(req.Messages))
		for i, m := range req.Messages {
			messages[i] = openRouterMessage{
				Role:    m.Role,
				Content: m.Content,
			}
		}

		// Build streaming request
		openRouterReq := openRouterRequest{
			Model:       req.Model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
			Temperature: req.Temperature,
			TopP:        req.TopP,
		}

		body, err := json.Marshal(openRouterReq)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("marshal error: %v", err)}
			return
		}

		// Initiate streaming fetch
		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + p.apiKey,
			"HTTP-Referer":  p.httpReferer,
			"X-Title":       p.xTitle,
			"Accept":        "text/event-stream",
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
			js.Global().Get("console").Call("error", "[OpenRouter] Stream error: status=", status)
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("HTTP %d", status)}
			return
		}

		// Log successful stream initiation
		js.Global().Get("console").Call("log", "[OpenRouter] Stream started: status=", status)

		// Create SSE reader
		sseReader := jsbridge.NewSSEStreamingReader(response)
		events := sseReader.Events()

		// Process SSE events
		for event := range events {
			if event.Event == "error" {
				tokenChan <- Token{FinishReason: "error", Text: event.Data}
				return
			}

			if event.Data == "" {
				continue
			}

			// Check for [DONE] terminator
			if event.Data == "[DONE]" {
				return
			}

			// Check for error JSON
			var errResp openRouterError
			if err := json.Unmarshal([]byte(event.Data), &errResp); err == nil && errResp.Error.Message != "" {
				tokenChan <- Token{
					FinishReason: "error",
					Text:         fmt.Sprintf("OpenRouter error: %s (provider: %s)", errResp.Error.Message, errResp.Error.Metadata.ProviderName),
				}
				return
			}

			// Parse the JSON chunk
			var streamResp openRouterStreamResponse
			if err := json.Unmarshal([]byte(event.Data), &streamResp); err != nil {
				// Skip unparsable chunks but continue
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]

			// Send content delta
			if choice.Delta.Content != "" {
				tokenChan <- Token{
					Text: choice.Delta.Content,
				}
			}

			// Check for finish reason
			if choice.FinishReason != "" {
				tokenChan <- Token{
					Text:         "",
					FinishReason: choice.FinishReason,
				}
				return
			}
		}
	}()

	return tokenChan
}

// Embed is not directly supported by OpenRouter's chat completions API
// OpenRouter focuses on routing LLM completions, not embeddings
func (p *OpenRouterProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("OpenRouter does not support embeddings - use OpenAI or another provider directly")
}

func (p *OpenRouterProvider) handleError(status int, body []byte) error {
	// Try to parse OpenRouter error response
	var errResp openRouterError
	message := string(body)

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		message = fmt.Sprintf("%s (provider: %s, code: %d)",
			errResp.Error.Message,
			errResp.Error.Metadata.ProviderName,
			errResp.Error.Code)
	}

	switch status {
	case 401:
		return ErrAPIKeyMissing
	case 402:
		return errors.New("payment required - check OpenRouter credits")
	case 429:
		return ErrRateLimit
	case 500, 502, 503:
		return ErrServerError
	default:
		return fmt.Errorf("API error %d: %s", status, message)
	}
}

// compile check
var _ Provider = (*OpenRouterProvider)(nil)
