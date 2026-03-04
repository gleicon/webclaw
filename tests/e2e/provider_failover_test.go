//go:build js && wasm

package e2e

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// TestProviderChain_RetryExponentialBackoff tests that retry attempts use exponential backoff
func TestProviderChain_RetryExponentialBackoff(t *testing.T) {
	// Create mock provider that fails twice then succeeds
	mockProv := &mockRetryProvider{
		failCount: 2,
		maxFails:  2,
	}

	chain := provider.NewProviderChain(mockProv, "test-model")
	chain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        500 * time.Millisecond,
	})

	ctx := context.Background()
	req := provider.CompletionRequest{
		Model:    "test",
		Messages: []provider.Message{{Role: "user", Content: "hello"}},
		Stream:   true,
	}

	start := time.Now()
	ch := chain.Stream(ctx, req)

	// Consume all tokens
	var tokens []provider.Token
	for tok := range ch {
		tokens = append(tokens, tok)
	}

	elapsed := time.Since(start)

	// Should have succeeded after 2 retries
	if len(tokens) == 0 {
		t.Error("expected tokens, got none")
	}

	// Should have taken at least 100ms + 200ms = 300ms for backoff
	if elapsed < 250*time.Millisecond {
		t.Errorf("expected backoff delay, elapsed: %v", elapsed)
	}

	// Should have retried twice
	if mockProv.actualRetries != 2 {
		t.Errorf("expected 2 retries, got %d", mockProv.actualRetries)
	}
}

// TestProviderChain_FallbackOnFailure tests fallback to secondary provider
func TestProviderChain_FallbackOnFailure(t *testing.T) {
	// Create failing primary and succeeding fallback
	primary := &mockFailingProvider{errorCode: "503"}
	fallback := &mockSuccessProvider{response: "fallback response"}

	chain := provider.NewProviderChain(primary, "primary-model")
	chain.SetFallback(fallback, "fallback-model")
	chain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1, // Fast fail to fallback
		InitialBackoff: 10 * time.Millisecond,
	})

	ctx := context.Background()
	req := provider.CompletionRequest{Stream: true}

	ch := chain.Stream(ctx, req)

	var gotFallback bool
	for tok := range ch {
		if tok.Text == "fallback response" {
			gotFallback = true
		}
	}

	if !gotFallback {
		t.Error("expected fallback provider to be used")
	}
}

// TestProviderChain_NonRetryableError tests that 401 errors fail fast without retry
func TestProviderChain_NonRetryableError(t *testing.T) {
	// 401 errors should not retry
	mockProv := &mockFailingProvider{errorCode: "401"}
	chain := provider.NewProviderChain(mockProv, "test")
	chain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
	})

	ctx := context.Background()
	req := provider.CompletionRequest{Stream: true}

	ch := chain.Stream(ctx, req)

	// Consume the error token
	for range ch {
	}

	// Should not have retried
	if mockProv.callCount > 1 {
		t.Errorf("expected no retries for 401, got %d calls", mockProv.callCount)
	}
}

// TestProviderChain_RetryableErrors tests various retryable error codes
func TestProviderChain_RetryableErrors(t *testing.T) {
	testCases := []struct {
		name        string
		errorCode   string
		shouldRetry bool
	}{
		{"429 Rate Limit", "429", true},
		{"502 Bad Gateway", "502", true},
		{"503 Service Unavailable", "503", true},
		{"504 Gateway Timeout", "504", true},
		{"529 Overloaded", "529", true},
		{"401 Unauthorized", "401", false},
		{"403 Forbidden", "403", false},
		{"400 Bad Request", "400", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProv := &mockFailingProvider{errorCode: tc.errorCode}
			chain := provider.NewProviderChain(mockProv, "test")
			chain.SetRetryConfig(provider.RetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 10 * time.Millisecond,
			})

			ctx := context.Background()
			req := provider.CompletionRequest{Stream: true}

			ch := chain.Stream(ctx, req)
			for range ch {
			}

			if tc.shouldRetry && mockProv.callCount < 2 {
				t.Errorf("expected retries for %s, got %d calls", tc.errorCode, mockProv.callCount)
			}
			if !tc.shouldRetry && mockProv.callCount > 1 {
				t.Errorf("expected no retries for %s, got %d calls", tc.errorCode, mockProv.callCount)
			}
		})
	}
}

// TestProviderChain_HealthTracking tests health status recording
func TestProviderChain_HealthTracking(t *testing.T) {
	// Test success tracking
	successProv := &mockSuccessProvider{response: "success"}
	chain := provider.NewProviderChain(successProv, "test")

	ctx := context.Background()
	req := provider.CompletionRequest{Stream: true}

	ch := chain.Stream(ctx, req)
	for range ch {
	}

	health := chain.GetHealth()
	if health.SuccessCount != 1 {
		t.Errorf("expected SuccessCount=1, got %d", health.SuccessCount)
	}
	if !health.IsHealthy {
		t.Error("expected IsHealthy=true after success")
	}

	// Test failure tracking
	failingProv := &mockFailingProvider{errorCode: "503"}
	chain2 := provider.NewProviderChain(failingProv, "test")
	chain2.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1, // Fail fast
		InitialBackoff: 1 * time.Millisecond,
	})

	ch2 := chain2.Stream(ctx, req)
	for range ch2 {
	}

	health2 := chain2.GetHealth()
	if health2.FailureCount != 1 {
		t.Errorf("expected FailureCount=1, got %d", health2.FailureCount)
	}
	if health2.IsHealthy {
		t.Error("expected IsHealthy=false after failure")
	}
}

// Mock types for testing

type mockRetryProvider struct {
	failCount     int
	maxFails      int
	actualRetries int
}

func (m *mockRetryProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	m.actualRetries++
	ch := make(chan provider.Token)

	go func() {
		defer close(ch)
		if m.actualRetries <= m.maxFails {
			// Simulate retryable error - return error token
			ch <- provider.Token{FinishReason: "error", Text: "HTTP 503"}
			return
		}
		ch <- provider.Token{Text: "success", FinishReason: "stop"}
	}()

	return ch
}

func (m *mockRetryProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRetryProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRetryProvider) Name() string {
	return "mockRetry"
}

func (m *mockRetryProvider) MaxContextWindow(model string) int {
	return 4096
}

type mockFailingProvider struct {
	errorCode string
	callCount int
}

func (m *mockFailingProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	m.callCount++
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{FinishReason: "error", Text: "HTTP " + m.errorCode}
	close(ch)
	return ch
}

func (m *mockFailingProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	m.callCount++
	return nil, errors.New("HTTP " + m.errorCode)
}

func (m *mockFailingProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFailingProvider) Name() string {
	return "mockFailing"
}

func (m *mockFailingProvider) MaxContextWindow(model string) int {
	return 4096
}

type mockSuccessProvider struct {
	response string
}

func (m *mockSuccessProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{Text: m.response, FinishReason: "stop"}
	close(ch)
	return ch
}

func (m *mockSuccessProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	return &provider.Token{Text: m.response, FinishReason: "stop"}, nil
}

func (m *mockSuccessProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSuccessProvider) Name() string {
	return "mockSuccess"
}

func (m *mockSuccessProvider) MaxContextWindow(model string) int {
	return 4096
}
