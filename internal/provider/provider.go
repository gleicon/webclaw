//go:build js && wasm

// Package provider implements LLM provider routing for Anthropic, OpenAI, and OpenRouter.
// All HTTP calls go through syscall/js fetch bridge - no net/http imports allowed.
package provider

import (
	"context"
	"errors"
	"io"
	"syscall/js"
)

// Token represents a streaming token from the LLM
type Token struct {
	Text         string
	FinishReason string // "stop", "length", "content_filter", ""
}

// Message represents a chat message
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// CompletionRequest contains all parameters for a chat completion
type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	TopP        float64
	Stream      bool
}

// Provider is the interface for LLM providers
// Implementations must use jsbridge fetch, not net/http
type Provider interface {
	// Complete returns a full completion (non-streaming)
	Complete(ctx context.Context, req CompletionRequest) (*Token, error)

	// Stream returns a channel of tokens for streaming responses
	// The channel will be closed when the stream is complete
	// Errors are sent as the last token with FinishReason="error" and Text containing the error message
	Stream(ctx context.Context, req CompletionRequest) <-chan Token

	// Embed creates embeddings for the given input
	Embed(ctx context.Context, input string) ([]float32, error)

	// Name returns the provider identifier (e.g., "anthropic", "openai", "openrouter")
	Name() string

	// MaxContextWindow returns the maximum context window for the model
	MaxContextWindow(model string) int
}

// ParseModelID parses a vendor/model-id string and returns vendor and model parts
// Supported formats:
//   - "anthropic/claude-sonnet-4-5" → vendor="anthropic", model="claude-sonnet-4-5"
//   - "openai/gpt-4" → vendor="openai", model="gpt-4"
//   - "openrouter/anthropic/claude-sonnet-4-5" → vendor="openrouter", model="anthropic/claude-sonnet-4-5"
func ParseModelID(modelID string) (vendor, model string, err error) {
	if modelID == "" {
		return "", "", errors.New("model ID cannot be empty")
	}

	// Find first slash
	for i := 0; i < len(modelID); i++ {
		if modelID[i] == '/' {
			return modelID[:i], modelID[i+1:], nil
		}
	}

	return "", "", errors.New("model ID must be in vendor/model format")
}

// Common errors
var (
	ErrInvalidModel     = errors.New("invalid model ID format")
	ErrProviderNotFound = errors.New("provider not found")
	ErrAPIKeyMissing    = errors.New("API key not configured")
	ErrContextTooLong   = errors.New("context exceeds model's maximum window")
	ErrRateLimit        = errors.New("rate limit exceeded")
	ErrServerError      = errors.New("server error from provider")
)

// JSFetchFunc is the type signature for the JS fetch bridge function
// This matches the signature used in jsbridge package
type JSFetchFunc func(url string, options js.Value) js.Value

// Config holds provider-specific configuration
type Config struct {
	AnthropicAPIKey  string
	OpenAIAPIKey     string
	OpenRouterAPIKey string

	// HTTPReferer and XTitle are required by OpenRouter
	HTTPReferer string
	XTitle      string
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
}

// SSEParser parses Server-Sent Events from a reader
type SSEParser struct {
	buffer []byte
	done   bool
}

// NewSSEParser creates a new SSE parser
func NewSSEParser() *SSEParser {
	return &SSEParser{
		buffer: make([]byte, 0, 4096),
	}
}

// Write adds data to the parser buffer
func (p *SSEParser) Write(data []byte) {
	p.buffer = append(p.buffer, data...)
}

// MarkDone signals the parser that the stream is complete
func (p *SSEParser) MarkDone() {
	p.done = true
}

// NextEvent returns the next SSE event from the buffer
// Returns nil if no complete event is available yet
func (p *SSEParser) NextEvent() *SSEEvent {
	// Look for double newline (event terminator)
	data := string(p.buffer)

	var endIdx int
	if idx := findDoubleNewline(data); idx >= 0 {
		endIdx = idx
	} else if p.done && len(data) > 0 {
		// If done, treat remaining data as last event
		endIdx = len(data)
	} else {
		return nil
	}

	// Parse the event block
	block := data[:endIdx]
	p.buffer = p.buffer[endIdx+2:]
	if len(p.buffer) > 0 && p.buffer[0] == '\n' {
		p.buffer = p.buffer[1:] // Handle \r\n\r\n case
	}

	return parseSSEBlock(block)
}

func findDoubleNewline(s string) int {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\n' && s[i+1] == '\n' {
			return i
		}
		if i < len(s)-3 && s[i] == '\r' && s[i+1] == '\n' && s[i+2] == '\r' && s[i+3] == '\n' {
			return i + 2
		}
	}
	return -1
}

func parseSSEBlock(block string) *SSEEvent {
	event := &SSEEvent{}
	lines := splitLines(block)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// SSE format: "field: value"
		if idx := findColon(line); idx >= 0 {
			field := line[:idx]
			value := ""
			if idx+1 < len(line) && line[idx+1] == ' ' {
				value = line[idx+2:]
			} else if idx+1 < len(line) {
				value = line[idx+1:]
			}

			switch field {
			case "event":
				event.Event = value
			case "data":
				event.Data = value
			}
		}
	}

	return event
}

func splitLines(s string) []string {
	var lines []string
	var start int

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			// Remove trailing \r for Windows line endings
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

func findColon(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// StreamReader provides io.Reader-like interface for SSE streams
// using syscall/js fetch responses
type StreamReader struct {
	reader js.Value
	buffer []byte
}

// NewStreamReader creates a stream reader from a JS ReadableStream
func NewStreamReader(reader js.Value) *StreamReader {
	return &StreamReader{
		reader: reader,
		buffer: make([]byte, 0, 4096),
	}
}

// Read reads data from the stream into the provided buffer
func (r *StreamReader) Read(p []byte) (n int, err error) {
	// If we have buffered data, return it first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Check if reader is done
	done := r.reader.Get("done").Bool()
	if done {
		return 0, io.EOF
	}

	// Read next chunk using JS API
	// This is a simplified implementation - real one uses Promise-based reading
	// through the jsbridge fetch mechanism
	return 0, io.EOF
}
