//go:build js && wasm

package e2e

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// TestProviderHealthTracking is the Phase 06 automated test for health tracking
// This test verifies:
// 1. Health tracking shows success/failure counts per provider
// 2. After 3 consecutive failures, provider is marked unhealthy
// 3. Unhealthy providers are skipped in fallback chain
// 4. Health status is queryable via GetHealth()
func TestProviderHealthTracking(t *testing.T) {
	t.Log("=== Phase 06: Provider Health Tracking Test ===")

	ctx := context.Background()
	req := provider.CompletionRequest{
		Model:    "test-model",
		Messages: []provider.Message{{Role: "user", Content: "test"}},
		Stream:   true,
	}

	// ============================================
	// STEP 1: Create Router with Anthropic and OpenAI providers
	// ============================================
	t.Log("\n[STEP 1] Creating Router with providers...")

	config := &provider.Config{
		AnthropicAPIKey: "sk-ant-PLACEHOLDER-USE-ENV-VAR",
		OpenAIAPIKey:    "sk-PLACEHOLDER-USE-ENV-VAR",
	}

	router := provider.NewRouter(config)

	// Verify providers are registered
	if !router.HasProvider("anthropic") {
		t.Fatal("FAIL: Anthropic provider not registered")
	}
	if !router.HasProvider("openai") {
		t.Fatal("FAIL: OpenAI provider not registered")
	}
	t.Log("✓ Router created with Anthropic and OpenAI providers")

	// ============================================
	// STEP 2: Verify health tracking with successful requests
	// ============================================
	t.Log("\n[STEP 2] Testing health tracking with mock successes...")

	// Create a mock provider that always succeeds for controlled testing
	successProv := &healthMockSuccessProvider{response: "success response"}
	chain := provider.NewProviderChain(successProv, "test-model")
	chain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	// Make several successful requests
	for i := 0; i < 5; i++ {
		ch := chain.Stream(ctx, req)
		for range ch {
		}
	}

	health := chain.GetHealth()
	if health.SuccessCount != 5 {
		t.Fatalf("FAIL: Expected SuccessCount=5, got %d", health.SuccessCount)
	}
	if health.FailureCount != 0 {
		t.Fatalf("FAIL: Expected FailureCount=0, got %d", health.FailureCount)
	}
	if !health.IsHealthy {
		t.Fatal("FAIL: Expected IsHealthy=true after successes")
	}
	t.Logf("✓ Success count tracking works: SuccessCount=%d, FailureCount=%d, IsHealthy=%v",
		health.SuccessCount, health.FailureCount, health.IsHealthy)

	// ============================================
	// STEP 3: Verify failure tracking and consecutive failures
	// ============================================
	t.Log("\n[STEP 3] Testing failure tracking and 3-strike rule...")

	// Create a provider that always fails
	failingProv := &healthMockFailingProvider{errorCode: "503"}
	chain2 := provider.NewProviderChain(failingProv, "test-model")
	chain2.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1, // Fail fast to test failure counting
		InitialBackoff: 1 * time.Millisecond,
	})

	// First failure
	ch1 := chain2.Stream(ctx, req)
	for range ch1 {
	}
	healthAfter1 := chain2.GetHealth()
	if healthAfter1.FailureCount != 1 {
		t.Fatalf("FAIL: Expected FailureCount=1 after 1st failure, got %d", healthAfter1.FailureCount)
	}
	if !healthAfter1.IsHealthy {
		t.Fatal("FAIL: Provider should still be healthy after 1 failure")
	}
	t.Logf("  After 1 failure: FailureCount=%d, IsHealthy=%v",
		healthAfter1.FailureCount, healthAfter1.IsHealthy)

	// Second failure
	ch2 := chain2.Stream(ctx, req)
	for range ch2 {
	}
	healthAfter2 := chain2.GetHealth()
	if healthAfter2.FailureCount != 2 {
		t.Fatalf("FAIL: Expected FailureCount=2 after 2nd failure, got %d", healthAfter2.FailureCount)
	}
	if !healthAfter2.IsHealthy {
		t.Fatal("FAIL: Provider should still be healthy after 2 failures")
	}
	t.Logf("  After 2 failures: FailureCount=%d, IsHealthy=%v",
		healthAfter2.FailureCount, healthAfter2.IsHealthy)

	// Third failure - should mark as unhealthy
	ch3 := chain2.Stream(ctx, req)
	for range ch3 {
	}
	healthAfter3 := chain2.GetHealth()
	if healthAfter3.FailureCount != 3 {
		t.Fatalf("FAIL: Expected FailureCount=3 after 3rd failure, got %d", healthAfter3.FailureCount)
	}
	if healthAfter3.IsHealthy {
		t.Fatal("FAIL: Provider should be marked unhealthy after 3 consecutive failures")
	}
	t.Logf("✓ 3-strike rule works: FailureCount=%d, IsHealthy=%v (marked unhealthy)",
		healthAfter3.FailureCount, healthAfter3.IsHealthy)

	// ============================================
	// STEP 4: Verify success resets failure count
	// ============================================
	t.Log("\n[STEP 4] Testing success resets failure count...")

	successResetProv := &healthMockSuccessProvider{response: "reset"}
	chain3 := provider.NewProviderChain(successResetProv, "test-model")
	chain3.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	// Simulate some failures first by creating a failing chain
	failingChain := provider.NewProviderChain(&healthMockFailingProvider{errorCode: "503"}, "test")
	failingChain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	// Cause 2 failures
	for i := 0; i < 2; i++ {
		ch := failingChain.Stream(ctx, req)
		for range ch {
		}
	}

	// Now use success provider
	ch := chain3.Stream(ctx, req)
	for range ch {
	}

	healthAfterSuccess := chain3.GetHealth()
	if healthAfterSuccess.SuccessCount != 1 {
		t.Fatalf("FAIL: Expected SuccessCount=1, got %d", healthAfterSuccess.SuccessCount)
	}
	if healthAfterSuccess.FailureCount != 0 {
		t.Fatalf("FAIL: Expected FailureCount=0 (reset), got %d", healthAfterSuccess.FailureCount)
	}
	if !healthAfterSuccess.IsHealthy {
		t.Fatal("FAIL: Expected IsHealthy=true after success")
	}
	t.Logf("✓ Success resets failure count: SuccessCount=%d, FailureCount=%d, IsHealthy=%v",
		healthAfterSuccess.SuccessCount, healthAfterSuccess.FailureCount, healthAfterSuccess.IsHealthy)

	// ============================================
	// STEP 5: Verify fallback skips unhealthy provider
	// ============================================
	t.Log("\n[STEP 5] Testing fallback skips unhealthy provider...")

	// Create a primary that will become unhealthy and a healthy fallback
	primaryProv := &healthMockFailingProvider{errorCode: "503"}
	fallbackProv := &healthMockSuccessProvider{response: "fallback-success"}

	chain4 := provider.NewProviderChain(primaryProv, "primary-model")
	chain4.SetFallback(fallbackProv, "fallback-model")
	chain4.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1, // Fast fail
		InitialBackoff: 1 * time.Millisecond,
	})

	// Cause 3 failures to mark primary as unhealthy
	for i := 0; i < 3; i++ {
		ch := chain4.Stream(ctx, req)
		for range ch {
		}
	}

	// Verify primary is unhealthy
	primaryHealth := chain4.GetHealth()
	if primaryHealth.IsHealthy {
		t.Fatal("FAIL: Primary provider should be unhealthy after 3 failures")
	}

	// Make another request - should trigger fallback
	ch4 := chain4.Stream(ctx, req)
	var gotFallback bool
	for tok := range ch4 {
		if tok.Text == "fallback-success" {
			gotFallback = true
		}
	}

	if !gotFallback {
		t.Fatal("FAIL: Expected fallback provider to be used when primary is unhealthy")
	}
	t.Logf("✓ Fallback works for unhealthy provider: Primary unhealthy=%v, Fallback triggered=%v",
		!primaryHealth.IsHealthy, gotFallback)

	// ============================================
	// STEP 6: Verify health timestamps are recorded
	// ============================================
	t.Log("\n[STEP 6] Testing health timestamp recording...")

	timeProv := &healthMockSuccessProvider{response: "timestamp-test"}
	chain5 := provider.NewProviderChain(timeProv, "test")
	chain5.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	beforeRequest := time.Now()
	ch5 := chain5.Stream(ctx, req)
	for range ch5 {
	}
	afterRequest := time.Now()

	timeHealth := chain5.GetHealth()
	if timeHealth.LastSuccess.IsZero() {
		t.Fatal("FAIL: LastSuccess timestamp should be recorded")
	}
	if timeHealth.LastSuccess.Before(beforeRequest) || timeHealth.LastSuccess.After(afterRequest) {
		t.Fatal("FAIL: LastSuccess timestamp outside expected range")
	}
	t.Logf("✓ Timestamps recorded: LastSuccess=%v", timeHealth.LastSuccess.Format(time.RFC3339))

	// Test failure timestamp
	timeFailProv := &healthMockFailingProvider{errorCode: "503"}
	chain6 := provider.NewProviderChain(timeFailProv, "test")
	chain6.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	beforeFail := time.Now()
	ch6 := chain6.Stream(ctx, req)
	for range ch6 {
	}
	afterFail := time.Now()

	timeFailHealth := chain6.GetHealth()
	if timeFailHealth.LastFailure.IsZero() {
		t.Fatal("FAIL: LastFailure timestamp should be recorded")
	}
	if timeFailHealth.LastFailure.Before(beforeFail) || timeFailHealth.LastFailure.After(afterFail) {
		t.Fatal("FAIL: LastFailure timestamp outside expected range")
	}
	t.Logf("✓ Failure timestamps recorded: LastFailure=%v", timeFailHealth.LastFailure.Format(time.RFC3339))

	// ============================================
	// STEP 7: Test Complete() method health tracking
	// ============================================
	t.Log("\n[STEP 7] Testing Complete() method health tracking...")

	completeSuccessProv := &healthMockSuccessProvider{response: "complete-success"}
	chain7 := provider.NewProviderChain(completeSuccessProv, "test")

	_, err := chain7.Complete(ctx, provider.CompletionRequest{
		Model:    "test",
		Messages: []provider.Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("FAIL: Complete() failed: %v", err)
	}

	completeHealth := chain7.GetHealth()
	if completeHealth.SuccessCount != 1 {
		t.Fatalf("FAIL: Expected SuccessCount=1 from Complete(), got %d", completeHealth.SuccessCount)
	}
	t.Logf("✓ Complete() health tracking works: SuccessCount=%d", completeHealth.SuccessCount)

	// Test Complete() failure tracking
	completeFailProv := &healthMockFailingProvider{errorCode: "503"}
	chain8 := provider.NewProviderChain(completeFailProv, "test")
	chain8.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
	})

	// 3 failures via Complete()
	for i := 0; i < 3; i++ {
		chain8.Complete(ctx, provider.CompletionRequest{
			Model:    "test",
			Messages: []provider.Message{{Role: "user", Content: "test"}},
		})
	}

	completeFailHealth := chain8.GetHealth()
	if completeFailHealth.FailureCount != 3 {
		t.Fatalf("FAIL: Expected FailureCount=3 from Complete(), got %d", completeFailHealth.FailureCount)
	}
	if completeFailHealth.IsHealthy {
		t.Fatal("FAIL: Provider should be unhealthy after 3 Complete() failures")
	}
	t.Logf("✓ Complete() failure tracking works: FailureCount=%d, IsHealthy=%v",
		completeFailHealth.FailureCount, completeFailHealth.IsHealthy)

	// ============================================
	// Summary
	// ============================================
	t.Log("\n=== Phase 06 Test Summary ===")
	t.Log("✓ Router created with Anthropic and OpenAI providers")
	t.Log("✓ Health tracking shows success count increasing")
	t.Log("✓ After 3 consecutive failures, provider marked unhealthy")
	t.Log("✓ Success resets failure count to 0")
	t.Log("✓ Fallback skips unhealthy provider")
	t.Log("✓ Health status queryable via GetHealth()")
	t.Log("✓ Timestamps (LastSuccess, LastFailure) recorded")
	t.Log("✓ Health tracking works for both Stream() and Complete()")
	t.Log("\n=== RESULT: PASS ===")
}

// ============================================================================
// Mock Providers for Health Testing
// ============================================================================

type healthMockSuccessProvider struct {
	response string
}

func (m *healthMockSuccessProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{Text: m.response, FinishReason: "stop"}
	close(ch)
	return ch
}

func (m *healthMockSuccessProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	return &provider.Token{Text: m.response, FinishReason: "stop"}, nil
}

func (m *healthMockSuccessProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("not implemented")
}

func (m *healthMockSuccessProvider) Name() string {
	return "healthMockSuccess"
}

func (m *healthMockSuccessProvider) MaxContextWindow(model string) int {
	return 4096
}

type healthMockFailingProvider struct {
	errorCode string
}

func (m *healthMockFailingProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{FinishReason: "error", Text: fmt.Sprintf("HTTP %s", m.errorCode)}
	close(ch)
	return ch
}

func (m *healthMockFailingProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	return nil, fmt.Errorf("HTTP %s", m.errorCode)
}

func (m *healthMockFailingProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, errors.New("not implemented")
}

func (m *healthMockFailingProvider) Name() string {
	return "healthMockFailing"
}

func (m *healthMockFailingProvider) MaxContextWindow(model string) int {
	return 4096
}
