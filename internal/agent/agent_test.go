//go:build js && wasm

package agent

import (
	"context"
	"syscall/js"
	"testing"
	"time"
)

// TestAgentLoopStreaming tests the end-to-end streaming flow
func TestAgentLoopStreaming(t *testing.T) {
	// Create agent loop
	loop := NewAgentLoopWithAssembler("test", "test-model", nil)

	// Create a worker bridge for callbacks
	bridge := &WorkerBridge{
		activeStreams: make(map[string]context.CancelFunc),
	}

	// Track tokens received
	tokensReceived := []string{}
	streamCompleted := false
	streamErrored := false

	// Set up callbacks
	bridge.onToken = func(token string) {
		tokensReceived = append(tokensReceived, token)
	}

	bridge.onComplete = func(result js.Value) {
		streamCompleted = true
	}

	bridge.onError = func(err error) {
		streamErrored = true
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test messages
	messages := []Message{
		{Role: "system", Content: "You are a test assistant."},
		{Role: "user", Content: "Hello"},
	}

	// Start the stream
	startTime := time.Now()
	err := loop.Run(ctx, messages, bridge)
	duration := time.Since(startTime)

	// Verify results
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if streamErrored {
		t.Error("Stream callback reported error")
	}

	if !streamCompleted {
		t.Error("Stream did not complete")
	}

	if len(tokensReceived) == 0 {
		t.Error("No tokens received")
	}

	// Verify streaming happened (not instantaneous)
	if duration < 50*time.Millisecond {
		t.Logf("Warning: stream completed very quickly (%v), may not have actually streamed", duration)
	}

	t.Logf("Stream completed in %v with %d tokens", duration, len(tokensReceived))
}

// TestAgentLoopAbort tests that abort stops the stream
func TestAgentLoopAbort(t *testing.T) {
	// Create agent loop with mock provider that streams slowly
	loop := NewAgentLoop("slow", "test-model")

	bridge := &WorkerBridge{
		activeStreams: make(map[string]context.CancelFunc),
	}

	tokensReceived := 0
	streamCompleted := false

	bridge.onToken = func(token string) {
		tokensReceived++
	}

	bridge.onComplete = func(result js.Value) {
		streamCompleted = true
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create test messages
	messages := []Message{
		{Role: "user", Content: "Count to 100"},
	}

	// Start stream in goroutine
	done := make(chan error, 1)
	go func() {
		done <- loop.Run(ctx, messages, bridge)
	}()

	// Abort after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stream did not complete after abort")
	}

	// Verify stream was aborted (not all tokens received)
	if tokensReceived >= 50 {
		t.Error("Stream received too many tokens - abort may not have worked")
	}

	t.Logf("Stream aborted after %d tokens", tokensReceived)
	if streamCompleted {
		t.Log("Stream completed (possibly with partial content)")
	}
}

// TestContextAssembly tests context assembly with system prompt
func TestContextAssembly(t *testing.T) {
	// This test would need config and identity store setup
	// For now, it's a placeholder that documents expected behavior
	t.Skip("Requires config and identity store - integration test")

	// Expected behavior:
	// 1. Create ContextAssembler with config and identity store
	// 2. Call AssembleContext("Hello")
	// 3. Verify first message is system prompt with identity
	// 4. Verify last message is user message with "Hello"
}

// TestWorkerBridgeCallbacks tests callback registration
func TestWorkerBridgeCallbacks(t *testing.T) {
	bridge := &WorkerBridge{
		activeStreams: make(map[string]context.CancelFunc),
	}

	tokenReceived := ""
	completeReceived := false

	// Register callbacks
	bridge.onToken = func(token string) {
		tokenReceived = token
	}

	bridge.onComplete = func(result js.Value) {
		completeReceived = true
	}

	// Test EmitToken
	bridge.EmitToken("test-token")
	if tokenReceived != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", tokenReceived)
	}

	// Test EmitComplete
	bridge.EmitComplete(true, "test-content")
	if !completeReceived {
		t.Error("Complete callback not invoked")
	}
}

// BenchmarkTokenThroughput benchmarks the token emission rate
func BenchmarkTokenThroughput(b *testing.B) {
	bridge := &WorkerBridge{
		activeStreams: make(map[string]context.CancelFunc),
	}

	tokenCount := 0
	bridge.onToken = func(token string) {
		tokenCount++
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bridge.EmitToken("test")
	}
	b.StopTimer()

	if tokenCount != b.N {
		b.Fatalf("Expected %d tokens, got %d", b.N, tokenCount)
	}
}
