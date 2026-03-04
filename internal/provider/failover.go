//go:build js && wasm

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	BackoffMultiplier float64
	MaxBackoff        time.Duration
}

// DefaultRetryConfig returns the default retry configuration
// 3 attempts with exponential backoff: 1s, 2s, 4s
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        8 * time.Second,
	}
}

// ProviderChain implements provider failover with exponential backoff
type ProviderChain struct {
	primary       Provider
	fallback      Provider // Optional fallback provider
	modelID       string   // Primary model ID
	fallbackModel string   // Fallback model ID (if different)
	retry         RetryConfig
}

// NewProviderChain creates a new provider chain with retry
func NewProviderChain(primary Provider, modelID string) *ProviderChain {
	return &ProviderChain{
		primary: primary,
		modelID: modelID,
		retry:   DefaultRetryConfig(),
	}
}

// WithFallback sets a fallback provider and model
func (pc *ProviderChain) WithFallback(fallback Provider, modelID string) *ProviderChain {
	pc.fallback = fallback
	pc.fallbackModel = modelID
	return pc
}

// WithRetryConfig sets custom retry configuration (fluent API)
func (pc *ProviderChain) WithRetryConfig(config RetryConfig) *ProviderChain {
	pc.retry = config
	return pc
}

// SetRetryConfig sets custom retry configuration (direct API)
func (pc *ProviderChain) SetRetryConfig(config RetryConfig) {
	pc.retry = config
}

// SetFallback sets a fallback provider and model (direct API)
func (pc *ProviderChain) SetFallback(fallback Provider, modelID string) {
	pc.fallback = fallback
	pc.fallbackModel = modelID
}

// Name returns the provider name
func (pc *ProviderChain) Name() string {
	return "chain:" + pc.primary.Name()
}

// MaxContextWindow returns the context window (uses primary provider)
func (pc *ProviderChain) MaxContextWindow(model string) int {
	return pc.primary.MaxContextWindow(model)
}

// Complete performs a completion with retry and fallback
func (pc *ProviderChain) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	// Try primary with retries
	token, err := pc.completeWithRetry(ctx, pc.primary, pc.modelID, req)
	if err == nil {
		return token, nil
	}

	// If retryable error and fallback available, try fallback
	if pc.shouldFallback(err) && pc.fallback != nil {
		fallbackReq := req
		if pc.fallbackModel != "" {
			fallbackReq.Model = pc.fallbackModel
		}

		// Try fallback without additional retries (assume fallback is reliable)
		token, err = pc.fallback.Complete(ctx, fallbackReq)
		if err == nil {
			return token, nil
		}

		return nil, fmt.Errorf("primary failed: %w; fallback also failed: %v", err, err)
	}

	return nil, err
}

// completeWithRetry attempts completion with exponential backoff retries
func (pc *ProviderChain) completeWithRetry(ctx context.Context, provider Provider, modelID string, req CompletionRequest) (*Token, error) {
	req.Model = modelID

	backoff := pc.retry.InitialBackoff

	for attempt := 0; attempt < pc.retry.MaxAttempts; attempt++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		token, err := provider.Complete(ctx, req)
		if err == nil {
			return token, nil
		}

		// Don't retry if it's not a server/rate limit error
		if !pc.isRetryableError(err) {
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < pc.retry.MaxAttempts-1 {
			// Exponential backoff with jitter could be added here
			time.Sleep(backoff)

			// Increase backoff for next attempt
			backoff = time.Duration(float64(backoff) * pc.retry.BackoffMultiplier)
			if backoff > pc.retry.MaxBackoff {
				backoff = pc.retry.MaxBackoff
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts", pc.retry.MaxAttempts)
}

// isRetryableError checks if an error should trigger a retry
func (pc *ProviderChain) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if errors.Is(err, ErrRateLimit) {
		return true
	}
	if errors.Is(err, ErrServerError) {
		return true
	}

	// Check error message for HTTP status codes
	errStr := err.Error()
	if strings.Contains(errStr, "429") {
		return true // Rate limit
	}
	if strings.Contains(errStr, "503") || strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "504") || strings.Contains(errStr, "529") {
		return true // Server errors
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return true // Timeout errors
	}
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") {
		return true // Network errors
	}

	return false
}

// shouldFallback checks if we should try the fallback provider
func (pc *ProviderChain) shouldFallback(err error) bool {
	// Always fallback on rate limit or server errors after retries exhausted
	return pc.isRetryableError(err)
}

// Stream performs a streaming completion with retry and fallback
func (pc *ProviderChain) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	resultChan := make(chan Token, 10)

	go func() {
		defer close(resultChan)

		// Try primary stream with fallback on error
		var streamErr error

		// Attempt primary stream
		for attempt := 0; attempt < pc.retry.MaxAttempts; attempt++ {
			req.Model = pc.modelID
			ch := pc.primary.Stream(ctx, req)

			// Consume the stream and check for errors
			tokens, err := pc.consumeStream(ctx, ch)
			if err == nil {
				// Stream succeeded - forward all tokens
				for _, token := range tokens {
					resultChan <- token
				}
				return
			}

			streamErr = err

			// Check if error is retryable
			if !pc.isRetryableError(err) {
				break
			}

			// Wait before retry (except last attempt)
			if attempt < pc.retry.MaxAttempts-1 {
				backoff := time.Duration(float64(pc.retry.InitialBackoff) *
					pow(pc.retry.BackoffMultiplier, float64(attempt)))
				if backoff > pc.retry.MaxBackoff {
					backoff = pc.retry.MaxBackoff
				}
				time.Sleep(backoff)
			}
		}

		// Primary failed after retries - try fallback if available
		if pc.fallback != nil && pc.shouldFallback(streamErr) {
			fallbackReq := req
			if pc.fallbackModel != "" {
				fallbackReq.Model = pc.fallbackModel
			}

			ch := pc.fallback.Stream(ctx, fallbackReq)
			tokens, err := pc.consumeStream(ctx, ch)
			if err == nil {
				// Fallback succeeded
				for _, token := range tokens {
					resultChan <- token
				}
				return
			}

			// Both failed
			resultChan <- Token{
				FinishReason: "error",
				Text: fmt.Sprintf("primary failed after %d attempts: %v; fallback failed: %v",
					pc.retry.MaxAttempts, streamErr, err),
			}
			return
		}

		// Primary failed, no fallback or not retryable error
		resultChan <- Token{
			FinishReason: "error",
			Text:         fmt.Sprintf("stream failed after %d attempts: %v", pc.retry.MaxAttempts, streamErr),
		}
	}()

	return resultChan
}

// consumeStream reads all tokens from a channel and returns them
// Returns an error if any token has FinishReason="error"
func (pc *ProviderChain) consumeStream(ctx context.Context, ch <-chan Token) ([]Token, error) {
	var tokens []Token

	for {
		select {
		case token, ok := <-ch:
			if !ok {
				return tokens, nil
			}

			tokens = append(tokens, token)

			if token.FinishReason == "error" {
				return tokens, errors.New(token.Text)
			}

			if token.FinishReason != "" {
				// Stream finished (stop, length, etc.)
				return tokens, nil
			}

		case <-ctx.Done():
			return tokens, ctx.Err()
		}
	}
}

// pow calculates x^y for float64
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

// Embed performs an embedding with retry and fallback
func (pc *ProviderChain) Embed(ctx context.Context, input string) ([]float32, error) {
	// Try primary with retries
	embeddings, err := pc.embedWithRetry(ctx, pc.primary, input)
	if err == nil {
		return embeddings, nil
	}

	// Try fallback if available
	if pc.shouldFallback(err) && pc.fallback != nil {
		embeddings, err = pc.fallback.Embed(ctx, input)
		if err == nil {
			return embeddings, nil
		}
		return nil, fmt.Errorf("primary failed: %w; fallback also failed: %v", err, err)
	}

	return nil, err
}

// embedWithRetry attempts embedding with exponential backoff
func (pc *ProviderChain) embedWithRetry(ctx context.Context, provider Provider, input string) ([]float32, error) {
	backoff := pc.retry.InitialBackoff

	for attempt := 0; attempt < pc.retry.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		embeddings, err := provider.Embed(ctx, input)
		if err == nil {
			return embeddings, nil
		}

		if !pc.isRetryableError(err) {
			return nil, err
		}

		if attempt < pc.retry.MaxAttempts-1 {
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * pc.retry.BackoffMultiplier)
			if backoff > pc.retry.MaxBackoff {
				backoff = pc.retry.MaxBackoff
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts", pc.retry.MaxAttempts)
}

// ProviderChainBuilder helps build provider chains with multiple fallbacks
type ProviderChainBuilder struct {
	providers []chainProvider
	config    RetryConfig
}

type chainProvider struct {
	provider Provider
	modelID  string
}

// NewProviderChainBuilder creates a new chain builder
func NewProviderChainBuilder() *ProviderChainBuilder {
	return &ProviderChainBuilder{
		providers: make([]chainProvider, 0),
		config:    DefaultRetryConfig(),
	}
}

// AddProvider adds a provider to the chain
func (b *ProviderChainBuilder) AddProvider(p Provider, modelID string) *ProviderChainBuilder {
	b.providers = append(b.providers, chainProvider{provider: p, modelID: modelID})
	return b
}

// WithConfig sets the retry configuration
func (b *ProviderChainBuilder) WithConfig(config RetryConfig) *ProviderChainBuilder {
	b.config = config
	return b
}

// Build creates the provider chain
// Currently only supports primary + single fallback
// Future: could support arbitrary chain depth
func (b *ProviderChainBuilder) Build() (*ProviderChain, error) {
	if len(b.providers) == 0 {
		return nil, errors.New("no providers in chain")
	}

	chain := NewProviderChain(b.providers[0].provider, b.providers[0].modelID)
	chain.WithRetryConfig(b.config)

	if len(b.providers) > 1 {
		chain.WithFallback(b.providers[1].provider, b.providers[1].modelID)
	}

	return chain, nil
}

// RouterWithFailover wraps a Router with automatic failover
// This provides failover at the routing level
type RouterWithFailover struct {
	router   *Router
	fallback map[string]string // modelID -> fallbackModelID
	retry    RetryConfig
}

// NewRouterWithFailover creates a router with built-in failover support
func NewRouterWithFailover(router *Router) *RouterWithFailover {
	return &RouterWithFailover{
		router:   router,
		fallback: make(map[string]string),
		retry:    DefaultRetryConfig(),
	}
}

// SetFallback sets a fallback model for a given model
func (r *RouterWithFailover) SetFallback(primaryModel, fallbackModel string) {
	r.fallback[primaryModel] = fallbackModel
}

// Complete routes and completes with failover
func (r *RouterWithFailover) Complete(ctx context.Context, modelID string, req CompletionRequest) (*Token, error) {
	route, err := r.router.Route(modelID)
	if err != nil {
		return nil, err
	}

	chain := NewProviderChain(route.Provider, route.ModelID).
		WithRetryConfig(r.retry)

	// Check if we have a fallback configured
	if fallbackModelID, ok := r.fallback[modelID]; ok {
		fallbackRoute, err := r.router.Route(fallbackModelID)
		if err == nil {
			chain.WithFallback(fallbackRoute.Provider, fallbackRoute.ModelID)
		}
	}

	return chain.Complete(ctx, req)
}

// Stream routes and streams with failover
func (r *RouterWithFailover) Stream(ctx context.Context, modelID string, req CompletionRequest) (<-chan Token, error) {
	route, err := r.router.Route(modelID)
	if err != nil {
		return nil, err
	}

	chain := NewProviderChain(route.Provider, route.ModelID).
		WithRetryConfig(r.retry)

	// Check if we have a fallback configured
	if fallbackModelID, ok := r.fallback[modelID]; ok {
		fallbackRoute, err := r.router.Route(fallbackModelID)
		if err == nil {
			chain.WithFallback(fallbackRoute.Provider, fallbackRoute.ModelID)
		}
	}

	return chain.Stream(ctx, req), nil
}

// compile check
var _ Provider = (*ProviderChain)(nil)

// compile check
var _ = jsbridge.FetchResponse{}
var _ = js.Value{}
