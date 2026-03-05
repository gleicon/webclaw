//go:build js && wasm

package e2e

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// TestPhase06_NonRetryableErrorFailFast - Test 7: Non-Retryable Error Fail-Fast
// Verifies that authentication errors (401, 403) and bad requests (400) fail immediately without retries.
func TestPhase06_NonRetryableErrorFailFast(t *testing.T) {
	t.Log("=== Test 7: Non-Retryable Error Fail-Fast ===")

	// Test configuration: 3 attempts with 1s initial backoff
	retryConfig := provider.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        8 * time.Second,
	}

	t.Logf("[Config] MaxAttempts=%d, InitialBackoff=%v, BackoffMultiplier=%.1f",
		retryConfig.MaxAttempts, retryConfig.InitialBackoff, retryConfig.BackoffMultiplier)
	t.Log("Expected backoff sequence for retryable errors: 1s → 2s → 4s")
	t.Log("")

	// Test Case 1: 401 Unauthorized - Should fail immediately (0 retries)
	t.Log("[Test Case 1] HTTP 401 Unauthorized - Non-retryable")
	callCount401 := 0
	mock401 := &mockErrorProvider{
		errorType:   "401",
		errorMsg:    "HTTP 401: Unauthorized - Invalid API key",
		callCounter: &callCount401,
	}

	chain401 := provider.NewProviderChain(mock401, "test-model")
	chain401.SetRetryConfig(retryConfig)

	start401 := time.Now()
	_, err401 := chain401.Complete(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
	})
	elapsed401 := time.Since(start401)

	// Verify immediate failure
	if err401 == nil {
		t.Fatal("FAIL: Expected error for 401, got nil")
	}
	if !strings.Contains(err401.Error(), "401") {
		t.Errorf("FAIL: Expected error to contain '401', got: %v", err401)
	}
	if callCount401 != 1 {
		t.Errorf("FAIL: 401 should not retry (expected 1 call, got %d)", callCount401)
	}
	if elapsed401 > 500*time.Millisecond {
		t.Errorf("FAIL: 401 should fail immediately (took %v, expected <500ms)", elapsed401)
	}

	health401 := chain401.GetHealth()
	t.Logf("  ✓ 401 Unauthorized: Failed in %v", elapsed401)
	t.Logf("  ✓ Call count: %d (no retries)", callCount401)
	t.Logf("  ✓ FailureCount: %d", health401.FailureCount)
	t.Logf("  ✓ IsHealthy: %v", health401.IsHealthy)
	t.Log("")

	// Test Case 2: 403 Forbidden - Should fail immediately (0 retries)
	t.Log("[Test Case 2] HTTP 403 Forbidden - Non-retryable")
	callCount403 := 0
	mock403 := &mockErrorProvider{
		errorType:   "403",
		errorMsg:    "HTTP 403: Forbidden - Access denied",
		callCounter: &callCount403,
	}

	chain403 := provider.NewProviderChain(mock403, "test-model")
	chain403.SetRetryConfig(retryConfig)

	start403 := time.Now()
	_, err403 := chain403.Complete(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
	})
	elapsed403 := time.Since(start403)

	// Verify immediate failure
	if err403 == nil {
		t.Fatal("FAIL: Expected error for 403, got nil")
	}
	if !strings.Contains(err403.Error(), "403") {
		t.Errorf("FAIL: Expected error to contain '403', got: %v", err403)
	}
	if callCount403 != 1 {
		t.Errorf("FAIL: 403 should not retry (expected 1 call, got %d)", callCount403)
	}
	if elapsed403 > 500*time.Millisecond {
		t.Errorf("FAIL: 403 should fail immediately (took %v, expected <500ms)", elapsed403)
	}

	health403 := chain403.GetHealth()
	t.Logf("  ✓ 403 Forbidden: Failed in %v", elapsed403)
	t.Logf("  ✓ Call count: %d (no retries)", callCount403)
	t.Logf("  ✓ FailureCount: %d", health403.FailureCount)
	t.Logf("  ✓ IsHealthy: %v", health403.IsHealthy)
	t.Log("")

	// Test Case 3: 400 Bad Request - Should fail immediately (0 retries)
	t.Log("[Test Case 3] HTTP 400 Bad Request - Non-retryable")
	callCount400 := 0
	mock400 := &mockErrorProvider{
		errorType:   "400",
		errorMsg:    "HTTP 400: Bad Request - Invalid request format",
		callCounter: &callCount400,
	}

	chain400 := provider.NewProviderChain(mock400, "test-model")
	chain400.SetRetryConfig(retryConfig)

	start400 := time.Now()
	_, err400 := chain400.Complete(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
	})
	elapsed400 := time.Since(start400)

	// Verify immediate failure
	if err400 == nil {
		t.Fatal("FAIL: Expected error for 400, got nil")
	}
	if !strings.Contains(err400.Error(), "400") {
		t.Errorf("FAIL: Expected error to contain '400', got: %v", err400)
	}
	if callCount400 != 1 {
		t.Errorf("FAIL: 400 should not retry (expected 1 call, got %d)", callCount400)
	}
	if elapsed400 > 500*time.Millisecond {
		t.Errorf("FAIL: 400 should fail immediately (took %v, expected <500ms)", elapsed400)
	}

	health400 := chain400.GetHealth()
	t.Logf("  ✓ 400 Bad Request: Failed in %v", elapsed400)
	t.Logf("  ✓ Call count: %d (no retries)", callCount400)
	t.Logf("  ✓ FailureCount: %d", health400.FailureCount)
	t.Logf("  ✓ IsHealthy: %v", health400.IsHealthy)
	t.Log("")

	// Test Case 4: 429 Too Many Requests - Should retry with backoff
	t.Log("[Test Case 4] HTTP 429 Too Many Requests - Retryable (contrast test)")
	callCount429 := 0
	mock429 := &mockErrorProvider{
		errorType:   "429",
		errorMsg:    "HTTP 429: Too Many Requests - Rate limit exceeded",
		callCounter: &callCount429,
	}

	// Use shorter backoff for test speed
	retryConfigShort := provider.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        500 * time.Millisecond,
	}

	chain429 := provider.NewProviderChain(mock429, "test-model")
	chain429.SetRetryConfig(retryConfigShort)

	start429 := time.Now()
	_, err429 := chain429.Complete(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
	})
	elapsed429 := time.Since(start429)

	// Verify retries occurred
	if err429 == nil {
		t.Fatal("FAIL: Expected error for 429 after retries, got nil")
	}
	if callCount429 != 3 {
		t.Errorf("FAIL: 429 should retry 3 times (expected 3 calls, got %d)", callCount429)
	}

	// Expected minimum time: 100ms + 200ms = 300ms (3 attempts with 100ms, 200ms backoff)
	expectedMinTime := 250 * time.Millisecond
	if elapsed429 < expectedMinTime {
		t.Errorf("FAIL: 429 should have backoff delays (took %v, expected >%v)", elapsed429, expectedMinTime)
	}

	health429 := chain429.GetHealth()
	t.Logf("  ✓ 429 Too Many Requests: Failed in %v", elapsed429)
	t.Logf("  ✓ Call count: %d (3 attempts with 100ms, 200ms backoff)", callCount429)
	t.Logf("  ✓ FailureCount: %d", health429.FailureCount)
	t.Logf("  ✓ IsHealthy: %v", health429.IsHealthy)
	t.Log("")

	// Test Case 5: 500 Internal Server Error - Should retry with backoff
	t.Log("[Test Case 5] HTTP 500 Internal Server Error - Retryable (contrast test)")
	callCount500 := 0
	mock500 := &mockErrorProvider{
		errorType:   "500",
		errorMsg:    "HTTP 500: Internal Server Error",
		callCounter: &callCount500,
	}

	chain500 := provider.NewProviderChain(mock500, "test-model")
	chain500.SetRetryConfig(retryConfigShort)

	start500 := time.Now()
	_, err500 := chain500.Complete(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
	})
	elapsed500 := time.Since(start500)

	// Verify retries occurred
	if err500 == nil {
		t.Fatal("FAIL: Expected error for 500 after retries, got nil")
	}
	if callCount500 != 3 {
		t.Errorf("FAIL: 500 should retry 3 times (expected 3 calls, got %d)", callCount500)
	}

	if elapsed500 < expectedMinTime {
		t.Errorf("FAIL: 500 should have backoff delays (took %v, expected >%v)", elapsed500, expectedMinTime)
	}

	health500 := chain500.GetHealth()
	t.Logf("  ✓ 500 Internal Server Error: Failed in %v", elapsed500)
	t.Logf("  ✓ Call count: %d (3 attempts with backoff)", callCount500)
	t.Logf("  ✓ FailureCount: %d", health500.FailureCount)
	t.Logf("  ✓ IsHealthy: %v", health500.IsHealthy)
	t.Log("")

	// Test Case 6: Test with streaming (401 should fail immediately)
	t.Log("[Test Case 6] Streaming with 401 - Non-retryable")
	callCount401Stream := 0
	mock401Stream := &mockStreamingErrorProvider{
		errorType:   "401",
		errorMsg:    "HTTP 401: Unauthorized",
		callCounter: &callCount401Stream,
	}

	chain401Stream := provider.NewProviderChain(mock401Stream, "test-model")
	chain401Stream.SetRetryConfig(retryConfigShort)

	start401Stream := time.Now()
	ch := chain401Stream.Stream(context.Background(), provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Test"},
		},
		Stream: true,
	})

	// Consume stream
	tokenCount := 0
	for range ch {
		tokenCount++
	}
	elapsed401Stream := time.Since(start401Stream)

	if callCount401Stream != 1 {
		t.Errorf("FAIL: Streaming 401 should not retry (expected 1 call, got %d)", callCount401Stream)
	}
	if elapsed401Stream > 200*time.Millisecond {
		t.Errorf("FAIL: Streaming 401 should fail immediately (took %v, expected <200ms)", elapsed401Stream)
	}

	t.Logf("  ✓ Streaming 401: Failed in %v", elapsed401Stream)
	t.Logf("  ✓ Call count: %d (no retries)", callCount401Stream)
	t.Log("")

	// Summary
	t.Log("=== Test 7 Results Summary ===")
	t.Logf("✓ 401 Unauthorized: %v, %d call(s), immediate fail", elapsed401, callCount401)
	t.Logf("✓ 403 Forbidden: %v, %d call(s), immediate fail", elapsed403, callCount403)
	t.Logf("✓ 400 Bad Request: %v, %d call(s), immediate fail", elapsed400, callCount400)
	t.Logf("✓ 429 Too Many Requests: %v, %d call(s), with retry/backoff", elapsed429, callCount429)
	t.Logf("✓ 500 Server Error: %v, %d call(s), with retry/backoff", elapsed500, callCount500)
	t.Logf("✓ Streaming 401: %v, %d call(s), immediate fail", elapsed401Stream, callCount401Stream)
	t.Log("")

	// Verify contrast between retryable and non-retryable
	if callCount401 != 1 || callCount403 != 1 || callCount400 != 1 {
		t.Fatal("\n=== Test 7: FAIL === Non-retryable errors were retried when they shouldn't be")
	}
	if callCount429 != 3 || callCount500 != 3 {
		t.Fatal("\n=== Test 7: FAIL === Retryable errors were not retried properly")
	}

	t.Log("=== Test 7: PASS ===")
	t.Log("All non-retryable errors (400, 401, 403) failed immediately without retries.")
	t.Log("All retryable errors (429, 500) were retried with proper exponential backoff.")
}

// mockErrorProvider simulates a provider that returns specific HTTP errors
type mockErrorProvider struct {
	errorType   string
	errorMsg    string
	callCounter *int
}

func (m *mockErrorProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	*m.callCounter++
	return nil, errors.New(m.errorMsg)
}

func (m *mockErrorProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	*m.callCounter++
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{
		FinishReason: "error",
		Text:         m.errorMsg,
	}
	close(ch)
	return ch
}

func (m *mockErrorProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	*m.callCounter++
	return nil, errors.New(m.errorMsg)
}

func (m *mockErrorProvider) Name() string {
	return fmt.Sprintf("mock%s", m.errorType)
}

func (m *mockErrorProvider) MaxContextWindow(model string) int {
	return 4096
}

// mockStreamingErrorProvider simulates a provider that returns specific HTTP errors in streaming mode
type mockStreamingErrorProvider struct {
	errorType   string
	errorMsg    string
	callCounter *int
}

func (m *mockStreamingErrorProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	*m.callCounter++
	return nil, errors.New(m.errorMsg)
}

func (m *mockStreamingErrorProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	*m.callCounter++
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{
		FinishReason: "error",
		Text:         m.errorMsg,
	}
	close(ch)
	return ch
}

func (m *mockStreamingErrorProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	*m.callCounter++
	return nil, errors.New(m.errorMsg)
}

func (m *mockStreamingErrorProvider) Name() string {
	return fmt.Sprintf("mockStreaming%s", m.errorType)
}

func (m *mockStreamingErrorProvider) MaxContextWindow(model string) int {
	return 4096
}
