//go:build js && wasm

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// AnthropicProvider implements the Provider interface for Anthropic's Messages API
type AnthropicProvider struct {
	apiKey  string
	baseURL string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
	}
}

// Name returns the provider identifier
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// MaxContextWindow returns the maximum context window for the model
func (p *AnthropicProvider) MaxContextWindow(model string) int {
	// Anthropic model context windows
	switch model {
	case "claude-3-opus-20240229", "claude-opus-4-5":
		return 200000
	case "claude-3-sonnet-20240229", "claude-sonnet-4-5":
		return 200000
	case "claude-3-haiku-20240307":
		return 200000
	case "claude-3-5-sonnet-20241022":
		return 200000
	default:
		return 100000 // Conservative default
	}
}

// anthropicRequest represents the request body for Anthropic Messages API
type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Stream      bool               `json:"stream"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse represents a non-streaming response
type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicStreamEvent represents SSE events from Anthropic streaming
type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta *anthropicDelta `json:"delta,omitempty"`
	Usage *anthropicUsage `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Complete performs a non-streaming completion
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	if p.apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Convert messages to Anthropic format
	var systemMsg string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	// Build request
	anthropicReq := anthropicRequest{
		Model:       req.Model,
		Messages:    messages,
		System:      systemMsg,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request via JS fetch bridge
	opts := jsbridge.FetchOptions{
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         p.apiKey,
			"anthropic-version": "2023-06-01",
		},
		Body: string(body),
	}

	resp, err := jsbridge.Fetch(p.baseURL+"/messages", opts)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Handle errors
	if resp.Status >= 400 {
		return nil, p.handleError(resp.Status, resp.Body)
	}

	// Parse response
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(resp.Body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from content blocks
	var text string
	for _, content := range anthropicResp.Content {
		if content.Type == "text" {
			text += content.Text
		}
	}

	return &Token{
		Text:         text,
		FinishReason: anthropicResp.StopReason,
	}, nil
}

// Stream performs a streaming completion
func (p *AnthropicProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	tokenChan := make(chan Token, 10)

	go func() {
		defer close(tokenChan)

		if p.apiKey == "" {
			tokenChan <- Token{FinishReason: "error", Text: ErrAPIKeyMissing.Error()}
			return
		}

		// Convert messages
		var systemMsg string
		var messages []anthropicMessage

		for _, m := range req.Messages {
			if m.Role == "system" {
				systemMsg = m.Content
			} else {
				messages = append(messages, anthropicMessage{
					Role:    m.Role,
					Content: m.Content,
				})
			}
		}

		// Build streaming request
		anthropicReq := anthropicRequest{
			Model:       req.Model,
			Messages:    messages,
			System:      systemMsg,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
			Temperature: req.Temperature,
			TopP:        req.TopP,
		}

		if anthropicReq.MaxTokens == 0 {
			anthropicReq.MaxTokens = 4096
		}

		body, err := json.Marshal(anthropicReq)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("marshal error: %v", err)}
			return
		}

		// Initiate streaming fetch
		opts := jsbridge.FetchOptions{
			Method: "POST",
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"x-api-key":         p.apiKey,
				"anthropic-version": "2023-06-01",
				"Accept":            "text/event-stream",
			},
			Body: string(body),
		}

		response, err := jsbridge.FetchStream(p.baseURL+"/messages", opts)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("stream error: %v", err)}
			return
		}

		// Check for immediate error status
		status := response.Get("status").Int()
		if status >= 400 {
			tokenChan <- Token{FinishReason: "error", Text: fmt.Sprintf("HTTP %d", status)}
			return
		}

		// Create SSE reader
		sseReader := jsbridge.NewSSEStreamingReader(response)
		events := sseReader.Events()

		// Process SSE events
		for event := range events {
			if event.Event == "error" {
				tokenChan <- Token{FinishReason: "error", Text: event.Data}
				return
			}

			// Anthropic SSE events have data field containing JSON
			if event.Data == "" {
				continue
			}

			// Handle SSE comment lines (start with ":")
			if len(event.Data) > 0 && event.Data[0] == ':' {
				continue // Ignore comments
			}

			var streamEvent anthropicStreamEvent
			if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
				// Some events might not be JSON (like [DONE])
				if event.Data == "[DONE]" {
					return
				}
				continue // Skip unparsable events
			}

			switch streamEvent.Type {
			case "content_block_delta":
				if streamEvent.Delta != nil && streamEvent.Delta.Type == "text_delta" {
					tokenChan <- Token{
						Text: streamEvent.Delta.Text,
					}
				}

			case "message_stop":
				tokenChan <- Token{
					Text:         "",
					FinishReason: "stop",
				}
				return

			case "error":
				tokenChan <- Token{
					FinishReason: "error",
					Text:         "stream error",
				}
				return
			}
		}
	}()

	return tokenChan
}

// Embed is not supported by Anthropic's Messages API
func (p *AnthropicProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("Anthropic does not support embeddings via Messages API")
}

func (p *AnthropicProvider) handleError(status int, body []byte) error {
	switch status {
	case 401:
		return ErrAPIKeyMissing
	case 429:
		return ErrRateLimit
	case 529:
		return ErrServerError
	case 500, 502, 503:
		return ErrServerError
	default:
		return fmt.Errorf("API error %d: %s", status, string(body))
	}
}

// compile check
var _ Provider = (*AnthropicProvider)(nil)
