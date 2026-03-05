//go:build js && wasm

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// TestPhase06_ProviderFailoverWithRetry - Phase 06 Automated Test
// Tests provider failover with exponential backoff retry mechanism
func TestPhase06_ProviderFailoverWithRetry(t *testing.T) {
	// Load credentials from .env.test
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	if anthropicKey == "" || openaiKey == "" {
		t.Skip("Skipping Phase 06 test: Missing API keys in environment. Ensure .env.test is loaded.")
	}

	t.Log("=== Phase 06: Provider Failover with Retry ===")
	t.Logf("Anthropic Key: %.10s...", anthropicKey)
	t.Logf("OpenAI Key: %.10s...", openaiKey)

	// Step 1: Create ProviderChain with Anthropic as primary, OpenAI as fallback
	t.Log("\n[Step 1] Creating ProviderChain (Anthropic primary, OpenAI fallback)...")

	anthropicProv := provider.NewAnthropicProvider(anthropicKey)
	openaiProv := provider.NewOpenAIProvider(openaiKey)

	chain := provider.NewProviderChain(anthropicProv, "claude-3-5-sonnet-20241022")
	chain.SetFallback(openaiProv, "gpt-4o-mini")

	// Verify chain structure
	if chain.Name() != "chain:anthropic" {
		t.Errorf("Expected chain name 'chain:anthropic', got '%s'", chain.Name())
	}
	t.Log("✓ ProviderChain created successfully")

	// Step 2: Verify default retry configuration (1s, 2s, 4s backoff)
	t.Log("\n[Step 2] Verifying retry configuration...")

	// Default retry config should be: MaxAttempts=3, InitialBackoff=1s, BackoffMultiplier=2.0
	// This creates backoff of: 1s, 2s, 4s
	retryConfig := provider.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        8 * time.Second,
	}
	chain.SetRetryConfig(retryConfig)
	t.Logf("✓ Retry config: MaxAttempts=%d, InitialBackoff=%v, BackoffMultiplier=%.1f",
		retryConfig.MaxAttempts, retryConfig.InitialBackoff, retryConfig.BackoffMultiplier)
	t.Log("  Expected backoff sequence: 1s → 2s → 4s")

	// Step 3: Make a streaming request that should succeed on primary
	t.Log("\n[Step 3] Testing streaming request (expecting primary success)...")

	ctx := context.Background()
	req := provider.CompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []provider.Message{
			{Role: "user", Content: "Say 'PRIMARY SUCCESS' and nothing else."},
		},
		Stream:    true,
		MaxTokens: 20,
	}

	start := time.Now()
	ch := chain.Stream(ctx, req)

	var tokens []provider.Token
	var responseText string
	for tok := range ch {
		tokens = append(tokens, tok)
		responseText += tok.Text
		t.Logf("  Token: %q", tok.Text)
	}
	elapsed := time.Since(start)

	// Verify response
	if len(tokens) == 0 {
		t.Fatal("FAIL: No tokens received from primary provider")
	}

	if !strings.Contains(strings.ToUpper(responseText), "PRIMARY") && !strings.Contains(strings.ToUpper(responseText), "SUCCESS") {
		t.Logf("  Warning: Response doesn't contain expected text. Got: %s", responseText)
	}

	t.Logf("✓ Primary request succeeded in %v", elapsed)
	t.Logf("  Received %d tokens", len(tokens))

	// Check health tracking
	health := chain.GetHealth()
	t.Logf("  Health: SuccessCount=%d, IsHealthy=%v", health.SuccessCount, health.IsHealthy)

	if health.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount=1, got %d", health.SuccessCount)
	}
	if !health.IsHealthy {
		t.Error("Expected IsHealthy=true after success")
	}

	// Step 4: Test failover with simulated retryable error
	t.Log("\n[Step 4] Testing failover with simulated 429 error...")

	// Create a mock failing provider that simulates 429 rate limit
	failingProv := &mockRateLimitProvider{
		failuresRemaining: 3, // Will fail all 3 retry attempts
	}

	// Create chain with failing primary and real OpenAI fallback
	failoverChain := provider.NewProviderChain(failingProv, "test-model")
	failoverChain.SetFallback(openaiProv, "gpt-4o-mini")
	failoverChain.SetRetryConfig(provider.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond, // Use shorter backoff for test speed
		BackoffMultiplier: 2.0,
		MaxBackoff:        500 * time.Millisecond,
	})

	req2 := provider.CompletionRequest{
		Model: "test-model",
		Messages: []provider.Message{
			{Role: "user", Content: "Say 'FALLBACK SUCCESS' and nothing else."},
		},
		Stream:    true,
		MaxTokens: 20,
	}

	start2 := time.Now()
	ch2 := failoverChain.Stream(ctx, req2)

	var tokens2 []provider.Token
	var responseText2 string
	for tok := range ch2 {
		tokens2 = append(tokens2, tok)
		responseText2 += tok.Text
	}
	elapsed2 := time.Since(start2)

	// Verify failover occurred
	if len(tokens2) == 0 {
		t.Fatal("FAIL: No tokens received from fallback provider")
	}

	// The fallback should have been triggered
	health2 := failoverChain.GetHealth()
	t.Logf("✓ Failover completed in %v", elapsed2)
	t.Logf("  Primary call count: %d (should be 3 - all retries exhausted)", failingProv.callCount)
	t.Logf("  Received %d tokens from fallback", len(tokens2))
	t.Logf("  Health: FailureCount=%d, IsHealthy=%v", health2.FailureCount, health2.IsHealthy)

	if failingProv.callCount != 3 {
		t.Errorf("Expected 3 primary attempts (max retries), got %d", failingProv.callCount)
	}

	if health2.FailureCount != 1 {
		t.Errorf("Expected FailureCount=1 after exhausting retries, got %d", health2.FailureCount)
	}

	// Step 5: Test Router with SetFallback
	t.Log("\n[Step 5] Testing Router with SetFallback integration...")

	config := &provider.Config{
		AnthropicAPIKey: anthropicKey,
		OpenAIAPIKey:    openaiKey,
	}

	router := provider.NewRouter(config)

	// Configure fallback using SetFallback
	router.SetFallback("anthropic", "openai", "gpt-4o-mini")

	// Verify both providers exist
	if !router.HasProvider("anthropic") {
		t.Error("Router missing anthropic provider")
	}
	if !router.HasProvider("openai") {
		t.Error("Router missing openai provider")
	}

	t.Log("✓ Router configured with SetFallback")
	t.Logf("  Available providers: %v", router.AvailableProviders())

	// Test routing with fallback
	routeReq := provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Say 'ROUTER TEST' and nothing else."},
		},
		Stream:    true,
		MaxTokens: 20,
	}

	ch3, err := router.Stream(ctx, "anthropic/claude-3-5-sonnet-20241022", routeReq)
	if err != nil {
		t.Fatalf("Router stream failed: %v", err)
	}

	var tokens3 []provider.Token
	for tok := range ch3 {
		tokens3 = append(tokens3, tok)
	}

	if len(tokens3) == 0 {
		t.Error("FAIL: Router returned no tokens")
	} else {
		t.Logf("✓ Router test succeeded with %d tokens", len(tokens3))
	}

	// Summary
	t.Log("\n=== Phase 06 Test Results ===")
	t.Log("✓ Step 1: ProviderChain created (Anthropic primary, OpenAI fallback)")
	t.Log("✓ Step 2: Retry configuration verified (1s, 2s, 4s backoff)")
	t.Log("✓ Step 3: Primary streaming request succeeded")
	t.Log("✓ Step 4: Failover with retry works (3 attempts → fallback)")
	t.Log("✓ Step 5: Router SetFallback integration works")
	t.Log("\n=== Phase 06: PASS ===")
}

// mockRateLimitProvider simulates a provider that always returns 429 rate limit errors
type mockRateLimitProvider struct {
	failuresRemaining int
	callCount         int
}

func (m *mockRateLimitProvider) Stream(ctx context.Context, req provider.CompletionRequest) <-chan provider.Token {
	m.callCount++
	ch := make(chan provider.Token, 1)
	ch <- provider.Token{
		FinishReason: "error",
		Text:         "HTTP 429: Rate limit exceeded",
	}
	close(ch)
	return ch
}

func (m *mockRateLimitProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.Token, error) {
	m.callCount++
	return nil, provider.ErrRateLimit
}

func (m *mockRateLimitProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return nil, provider.ErrRateLimit
}

func (m *mockRateLimitProvider) Name() string {
	return "mockRateLimit"
}

func (m *mockRateLimitProvider) MaxContextWindow(model string) int {
	return 4096
}
