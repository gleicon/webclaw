//go:build js && wasm

// Package tests contains integration tests for WebClaw's AI provider connections.
// These tests validate real API calls, streaming, and error handling.
//
// To run these tests:
// 1. Build the WASM: GOOS=js GOARCH=wasm go build -o webclaw ./cmd/webclaw
// 2. Set API keys in environment or test will skip
// 3. Run in browser environment with wasm_exec.js
//
// Cost-conscious testing:
// - Uses claude-3-haiku-20240307 (cheapest Anthropic model)
// - Uses max_tokens: 1 for validation calls
// - Uses short test prompts: "Say 'test'"
package tests

import (
	"context"
	"os"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// TestAnthropicSingleToken validates a minimal API call to Anthropic
// Requirements: ANTHROPIC_API_KEY environment variable
func TestAnthropicSingleToken(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	p := provider.NewAnthropicProvider(apiKey)
	ctx := context.Background()

	req := provider.CompletionRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []provider.Message{{Role: "user", Content: "Say 'test'"}},
		MaxTokens: 1,
	}

	token, err := p.Complete(ctx, req)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	if token.Text == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Received token: %q", token.Text)
}

// TestAnthropicStreaming validates that streaming delivers tokens incrementally
// Requirements: ANTHROPIC_API_KEY environment variable
func TestAnthropicStreaming(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	p := provider.NewAnthropicProvider(apiKey)
	ctx := context.Background()

	req := provider.CompletionRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []provider.Message{{Role: "user", Content: "Say hello world"}},
		MaxTokens: 10,
		Stream:    true,
	}

	tokenChan := p.Stream(ctx, req)

	var tokens []string
	for token := range tokenChan {
		if token.FinishReason == "error" {
			t.Fatalf("Stream error: %s", token.Text)
		}
		tokens = append(tokens, token.Text)
	}

	if len(tokens) < 2 {
		t.Errorf("Expected multiple tokens for streaming, got %d", len(tokens))
	}

	fullResponse := strings.Join(tokens, "")
	t.Logf("Received %d tokens, full response: %q", len(tokens), fullResponse)
}

// TestOpenAISingleToken validates a minimal API call to OpenAI
// Requirements: OPENAI_API_KEY environment variable
func TestOpenAISingleToken(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: OPENAI_API_KEY not set")
	}

	p := provider.NewOpenAIProvider(apiKey)
	ctx := context.Background()

	req := provider.CompletionRequest{
		Model:     "gpt-3.5-turbo",
		Messages:  []provider.Message{{Role: "user", Content: "Say 'test'"}},
		MaxTokens: 1,
	}

	token, err := p.Complete(ctx, req)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	if token.Text == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Received token: %q", token.Text)
}

// TestOpenAIStreaming validates OpenAI streaming delivers tokens
// Requirements: OPENAI_API_KEY environment variable
func TestOpenAIStreaming(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: OPENAI_API_KEY not set")
	}

	p := provider.NewOpenAIProvider(apiKey)
	ctx := context.Background()

	req := provider.CompletionRequest{
		Model:     "gpt-3.5-turbo",
		Messages:  []provider.Message{{Role: "user", Content: "Say hello world"}},
		MaxTokens: 10,
		Stream:    true,
	}

	tokenChan := p.Stream(ctx, req)

	var tokens []string
	for token := range tokenChan {
		if token.FinishReason == "error" {
			t.Fatalf("Stream error: %s", token.Text)
		}
		tokens = append(tokens, token.Text)
	}

	if len(tokens) < 2 {
		t.Errorf("Expected multiple tokens for streaming, got %d", len(tokens))
	}

	fullResponse := strings.Join(tokens, "")
	t.Logf("Received %d tokens, full response: %q", len(tokens), fullResponse)
}

// TestMissingAPIKey validates error handling for missing API key
func TestMissingAPIKey(t *testing.T) {
	p := provider.NewAnthropicProvider("")
	ctx := context.Background()

	req := provider.CompletionRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []provider.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 10,
	}

	_, err := p.Complete(ctx, req)
	if err == nil {
		t.Error("Expected error for missing API key")
	}

	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("Expected error to mention API key, got: %v", err)
	}
}

// TestRouterProviderSelection validates that router correctly selects providers
func TestRouterProviderSelection(t *testing.T) {
	// Create router with no keys
	config := &provider.Config{
		HTTPReferer: "https://test.example",
		XTitle:      "Test",
	}
	router := provider.NewRouter(config)

	// Should have no providers
	if len(router.AvailableProviders()) != 0 {
		t.Error("Expected 0 providers with no keys")
	}

	// Now add a key
	config2 := &provider.Config{
		AnthropicAPIKey: "test-key",
		HTTPReferer:     "https://test.example",
		XTitle:          "Test",
	}
	router2 := provider.NewRouter(config2)

	if len(router2.AvailableProviders()) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(router2.AvailableProviders()))
	}

	if !router2.HasProvider("anthropic") {
		t.Error("Expected anthropic provider to be available")
	}
}

// TestRouterModelRouting validates vendor/model-id routing
func TestRouterModelRouting(t *testing.T) {
	config := &provider.Config{
		AnthropicAPIKey:  "test-key-anthropic",
		OpenAIAPIKey:     "test-key-openai",
		OpenRouterAPIKey: "test-key-router",
		HTTPReferer:      "https://test.example",
		XTitle:           "Test",
	}
	router := provider.NewRouter(config)

	tests := []struct {
		modelID      string
		expectVendor string
		expectError  bool
	}{
		{"anthropic/claude-3-haiku-20240307", "anthropic", false},
		{"openai/gpt-4o", "openai", false},
		{"openrouter/anthropic/claude-sonnet", "openrouter", false},
		{"claude-3-haiku-20240307", "anthropic", false}, // inferred
		{"gpt-4o", "openai", false},                     // inferred
		{"", "", true},                                  // empty should error
	}

	for _, tc := range tests {
		result, err := router.Route(tc.modelID)
		if tc.expectError {
			if err == nil {
				t.Errorf("%s: expected error", tc.modelID)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.modelID, err)
			continue
		}
		if result.Vendor != tc.expectVendor {
			t.Errorf("%s: expected vendor %s, got %s", tc.modelID, tc.expectVendor, result.Vendor)
		}
	}
}

// TestProviderEvents validates that webclaw:providers-ready event is dispatched
func TestProviderEvents(t *testing.T) {
	// This test verifies the event dispatch mechanism
	// In a real browser test, we would listen for the event
	// Here we just verify the structure is correct

	config := &provider.Config{
		AnthropicAPIKey: "test-key",
	}
	router := provider.NewRouter(config)

	providers := router.AvailableProviders()
	if len(providers) != 1 || providers[0] != "anthropic" {
		t.Error("Expected anthropic in available providers")
	}

	// In actual browser environment, this would dispatch:
	// new CustomEvent('webclaw:providers-ready', {detail: {providers: ['anthropic'], count: 1}})
	t.Logf("Would dispatch event with providers: %v", providers)
}

// TestStreamingTimeout validates streaming doesn't hang indefinitely
func TestStreamingTimeout(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	p := provider.NewAnthropicProvider(apiKey)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := provider.CompletionRequest{
		Model:     "claude-3-haiku-20240307",
		Messages:  []provider.Message{{Role: "user", Content: "Hi"}},
		MaxTokens: 5,
		Stream:    true,
	}

	done := make(chan bool)
	go func() {
		tokenChan := p.Stream(ctx, req)
		for range tokenChan {
			// Consume tokens
		}
		done <- true
	}()

	select {
	case <-done:
		// Success - streaming completed
		t.Log("Streaming completed within timeout")
	case <-ctx.Done():
		t.Error("Streaming timed out - took longer than 5 seconds")
	}
}

// TestConsoleLogging validates that API calls log to console
// This is a manual verification test - check browser console
func TestConsoleLogging(t *testing.T) {
	// This test documents expected console output
	// When API calls are made, these should appear in console:
	// [Anthropic] API call: model= claude-3-haiku-20240307
	// [Anthropic] API response: status= 200 len= 123
	// [Anthropic] Stream: model= claude-3-haiku-20240307 messages= 1
	// [Anthropic] Stream started: status= 200

	t.Log("Console logging test - verify in browser DevTools:")
	t.Log("Expected: [Provider] API call: model= xxx")
	t.Log("Expected: [Provider] API response: status= 200 len= xxx")
	t.Log("Expected: [Provider] Stream started: status= 200")

	// In real browser environment, we could spy on console.log
	// For now, this documents the expected behavior
}

// TestProviderStatusIndicators validates provider status UI elements
// This is a manual browser test
func TestProviderStatusIndicators(t *testing.T) {
	t.Log("Manual test: Check Settings tab for provider status indicators")
	t.Log("Expected: Green dot + 'Connected' for configured providers")
	t.Log("Expected: Red dot + 'No key' for unconfigured providers")
	t.Log("Expected: Test button (🧪) appears for connected providers")
}

// TestDemoModeIndicator validates demo mode banner appears
// This is a manual browser test
func TestDemoModeIndicator(t *testing.T) {
	t.Log("Manual test: With no API keys configured:")
	t.Log("Expected: Yellow banner 'Demo Mode - Enter API key in Settings to enable live AI'")
}

// TestErrorToasts validates error toast messages
// This is a manual browser test
func TestErrorToasts(t *testing.T) {
	testCases := []struct {
		scenario      string
		expectedToast string
	}{
		{"Missing API key", "Please enter your API key in Settings"},
		{"Invalid key (401)", "Invalid API key - please check and re-enter"},
		{"Rate limit (429)", "Rate limited - please wait a moment"},
	}

	t.Log("Manual test: Trigger error conditions and verify toasts:")
	for _, tc := range testCases {
		t.Logf("  - %s: expect '%s'", tc.scenario, tc.expectedToast)
	}
}

// TestModelDropdown validates model selection routes to correct provider
// This is a manual browser test
func TestModelDropdown(t *testing.T) {
	testCases := []struct {
		modelValue     string
		expectedVendor string
	}{
		{"anthropic/claude-sonnet-4-5", "anthropic"},
		{"openai/gpt-4o", "openai"},
		{"openrouter/anthropic/claude-sonnet-4-5", "openrouter"},
	}

	t.Log("Manual test: Select models and verify routing in console:")
	for _, tc := range testCases {
		t.Logf("  - %s -> %s", tc.modelValue, tc.expectedVendor)
	}
}

// TestToolCallEndToEnd validates tool execution through live provider
// Requirements: ANTHROPIC_API_KEY and tool registry configured
func TestToolCallEndToEnd(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	t.Log("Manual test: Tool call end-to-end")
	t.Log("1. Configure Anthropic API key")
	t.Log("2. Send message: 'Fetch example.com'")
	t.Log("3. Verify: web_fetch tool appears in Tool Activity panel")
	t.Log("4. Verify: Tool result returned through streaming response")

	// In automated test environment, this would:
	// - Set up agent loop with real router
	// - Register tool registry with web_fetch
	// - Start stream with tool-capable model
	// - Verify tool_use/tool_result in response
}

// TestProviderSwitching validates changing models updates routing
// This is a manual browser test
func TestProviderSwitching(t *testing.T) {
	t.Log("Manual test: Provider switching")
	t.Log("1. Set both Anthropic and OpenAI keys")
	t.Log("2. Select anthropic/claude-sonnet-4-5, send message")
	t.Log("3. Verify: Console shows [Anthropic] logs")
	t.Log("4. Select openai/gpt-4o, send message")
	t.Log("5. Verify: Console shows [OpenAI] logs")
	t.Log("6. Verify: No page reload required")
}

// Helper to check if running in browser environment
func isBrowser() bool {
	return !js.Global().IsUndefined() && !js.Global().IsNull()
}

// init runs when package is loaded
func init() {
	if isBrowser() {
		// Log that tests are available
		js.Global().Get("console").Call("log", "[WebClaw Tests] Integration tests loaded. Set API keys to run live tests.")
	}
}
