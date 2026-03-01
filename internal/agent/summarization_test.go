//go:build js && wasm

package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestSummarizationWorkflow tests the complete summarization pipeline
func TestSummarizationWorkflow(t *testing.T) {
	// Create mock summarizer
	summarizer := CreateMockSummarizer()

	// Create summarization manager
	manager := NewSummarizationManager(summarizer)

	// Create conversation
	conv := NewConversation("test-conv-1")
	conv.SetThresholds(10, 0.75, 1000) // Lower threshold for testing

	// Add messages until we hit the threshold
	for i := 0; i < 9; i++ {
		conv.AddUserMessage("Test user message " + string(rune('0'+i)))
		conv.AddAssistantMessage("Test assistant response " + string(rune('0'+i)))
	}

	// Should not need summarization yet (18 messages > 10 threshold but needs check)
	if !manager.ShouldSummarize(conv) {
		t.Error("Should need summarization at 18 messages with threshold of 10")
	}

	// Perform summarization
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := manager.PerformSummarization(ctx, conv)
	if err != nil {
		t.Fatalf("Summarization failed: %v", err)
	}

	// Verify result
	if result.Summary == "" {
		t.Error("Summary is empty")
	}

	if result.MessageCount == 0 {
		t.Error("Message count is zero")
	}

	if result.Duration == 0 {
		t.Error("Duration is zero")
	}

	// Verify conversation was updated
	if conv.Summary == nil {
		t.Error("Conversation summary is nil after summarization")
	}

	if conv.Summary.Content == "" {
		t.Error("Conversation summary content is empty")
	}

	// Verify sliding window state
	sw := manager.GetSlidingWindow()
	if sw.Summary == "" {
		t.Error("Sliding window summary is empty")
	}

	if len(sw.RecentMessages) != 0 {
		t.Errorf("Recent messages should be cleared after compact, got %d", len(sw.RecentMessages))
	}

	t.Logf("Summarization complete: %d messages, %d tokens, %v duration",
		result.MessageCount, result.TokenCount, result.Duration)
}

// TestProgressiveSummarization tests the progressive window approach
func TestProgressiveSummarization(t *testing.T) {
	summarizer := CreateMockSummarizer()
	manager := NewSummarizationManager(summarizer)
	conv := NewConversation("test-conv-2")

	// Add initial messages
	for i := 0; i < 5; i++ {
		conv.AddUserMessage("Initial message " + string(rune('0'+i)))
		conv.AddAssistantMessage("Initial response " + string(rune('0'+i)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First summarization
	_, err := manager.PerformSummarization(ctx, conv)
	if err != nil {
		t.Fatalf("First summarization failed: %v", err)
	}

	firstSummary := manager.GetSlidingWindow().Summary

	// Add more messages
	for i := 0; i < 3; i++ {
		conv.AddUserMessage("Follow-up message " + string(rune('0'+i)))
		conv.AddAssistantMessage("Follow-up response " + string(rune('0'+i)))
	}

	// Second summarization - should merge with previous summary
	_, err = manager.PerformSummarization(ctx, conv)
	if err != nil {
		t.Fatalf("Second summarization failed: %v", err)
	}

	secondSummary := manager.GetSlidingWindow().Summary

	// Verify progressive summarization worked
	if secondSummary == firstSummary {
		t.Error("Summary should be different after second summarization")
	}

	// Both should contain some content
	if len(secondSummary) < len(firstSummary) {
		t.Logf("Second summary is shorter: first=%d, second=%d chars", len(firstSummary), len(secondSummary))
	}
}

// TestSlidingWindowBehavior tests sliding window operations
func TestSlidingWindowBehavior(t *testing.T) {
	sw := NewSlidingWindow()

	// Add messages
	for i := 0; i < 5; i++ {
		msg := ConversationMessage{
			Role:      "user",
			Content:   "Message " + string(rune('0'+i)),
			ID:        "msg_" + string(rune('0'+i)),
			Timestamp: time.Now(),
		}
		sw.AddMessage(msg)
	}

	if sw.GetRecentMessageCount() != 5 {
		t.Errorf("Expected 5 recent messages, got %d", sw.GetRecentMessageCount())
	}

	if sw.GetTotalMessageCount() != 5 {
		t.Errorf("Expected 5 total messages, got %d", sw.GetTotalMessageCount())
	}

	// Test progressive merge
	merge := sw.ProgressiveMerge()
	if !strings.Contains(merge, "Recent conversation") {
		t.Error("Progressive merge should contain 'Recent conversation' header")
	}

	// Test compact
	sw.Compact("Test summary content")

	if sw.Summary != "Test summary content" {
		t.Errorf("Summary mismatch: expected 'Test summary content', got '%s'", sw.Summary)
	}

	if sw.GetRecentMessageCount() != 0 {
		t.Errorf("Recent messages should be cleared after compact, got %d", sw.GetRecentMessageCount())
	}

	if sw.GetTotalMessageCount() != 5 {
		t.Errorf("Total message count should remain 5, got %d", sw.GetTotalMessageCount())
	}

	// Test progressive merge with summary
	merge = sw.ProgressiveMerge()
	if !strings.Contains(merge, "Previous context") {
		t.Error("Progressive merge should contain 'Previous context' header with summary")
	}
}

// TestConversationThresholds tests threshold detection
func TestConversationThresholds(t *testing.T) {
	conv := NewConversation("threshold-test")
	conv.SetThresholds(5, 0.5, 100) // 5 messages or 50 tokens

	// Should not trigger initially
	if conv.NeedsSummarization() {
		t.Error("Should not need summarization with 0 messages")
	}

	// Add 4 messages (below threshold)
	for i := 0; i < 4; i++ {
		conv.AddUserMessage("Test")
	}

	if conv.NeedsSummarization() {
		t.Error("Should not need summarization with 4 messages (threshold 5)")
	}

	// Add 1 more to hit threshold
	conv.AddUserMessage("Test")

	if !conv.NeedsSummarization() {
		t.Error("Should need summarization with 5 messages (threshold 5)")
	}

	// Test token threshold (50% of 100 = 50 tokens)
	conv2 := NewConversation("token-test")
	conv2.SetThresholds(100, 0.5, 100) // High message threshold, low token threshold

	// Add messages with ~48 tokens total (below 50% threshold)
	for i := 0; i < 3; i++ {
		conv2.AddUserMessage("This is a message with about sixteen tokens in it.")
	}

	// 3 * 16 = 48 tokens, 48% of 100, should NOT trigger
	if conv2.NeedsSummarization() {
		t.Error("Should not need summarization at 48% token usage")
	}

	// Add one more to push over 50%
	conv2.AddUserMessage("This is a message with about sixteen tokens in it.")

	// 4 * 16 = 64 tokens, 64% of 100, should trigger
	if !conv2.NeedsSummarization() {
		t.Error("Should need summarization at 64% token usage (threshold 50%)")
	}
}

// TestSummarizerMock tests the mock summarizer
func TestSummarizerMock(t *testing.T) {
	summarizer := CreateMockSummarizer()
	conv := NewConversation("mock-test")

	// Add some messages
	conv.AddUserMessage("Hello, how are you?")
	conv.AddAssistantMessage("I'm doing well, thank you for asking!")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := summarizer.SummarizeConversation(ctx, conv)
	if err != nil {
		t.Fatalf("Mock summarization failed: %v", err)
	}

	if result.Summary == "" {
		t.Error("Mock summary is empty")
	}

	if result.TokenCount == 0 {
		t.Error("Mock token count is zero")
	}

	t.Logf("Mock summary: %s (%d tokens)", result.Summary, result.TokenCount)
}

// TestSummarizationCallback tests the onSummarized callback
func TestSummarizationCallback(t *testing.T) {
	summarizer := CreateMockSummarizer()
	manager := NewSummarizationManager(summarizer)
	conv := NewConversation("callback-test")

	// Set up callback
	callbackCalled := false
	var callbackResult *SummarizeResult

	manager.SetOnSummarizedCallback(func(result *SummarizeResult) {
		callbackCalled = true
		callbackResult = result
	})

	// Add enough messages to trigger summarization
	for i := 0; i < 10; i++ {
		conv.AddUserMessage("Test message")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := manager.PerformSummarization(ctx, conv)
	if err != nil {
		t.Fatalf("Summarization failed: %v", err)
	}

	if !callbackCalled {
		t.Error("OnSummarized callback was not called")
	}

	if callbackResult == nil {
		t.Error("Callback result is nil")
	}
}

// BenchmarkSummarization benchmarks the summarization workflow
func BenchmarkSummarization(b *testing.B) {
	summarizer := CreateMockSummarizer()
	manager := NewSummarizationManager(summarizer)

	for i := 0; i < b.N; i++ {
		conv := NewConversation("bench-conv")
		for j := 0; j < 10; j++ {
			conv.AddUserMessage("Benchmark test message")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		manager.PerformSummarization(ctx, conv)
		cancel()
	}
}
