//go:build js && wasm

package e2e

import (
	"testing"

	"github.com/gleicon/webclaw/internal/agent"
)

// Test summarization triggers correctly at message threshold
func TestFullAgentLoop_SummarizationTrigger(t *testing.T) {
	conv := agent.NewConversation("test")
	conv.SetThresholds(5, 0.75, 1000) // 5 message threshold for testing

	// Add 4 messages (should not trigger)
	for i := 0; i < 4; i++ {
		conv.AddUserMessage("Test message")
	}

	if conv.NeedsSummarization() {
		t.Error("should not need summarization at 4 messages")
	}

	// Add 5th message (should trigger)
	conv.AddUserMessage("Fifth message")

	if !conv.NeedsSummarization() {
		t.Error("should need summarization at 5 messages")
	}
}

// Test token counting accuracy with new tokenizer
func TestFullAgentLoop_TokenCounting(t *testing.T) {
	conv := agent.NewConversation("test")

	// Add messages and verify token count increases
	initialTokens := conv.GetTokenCount()

	conv.AddUserMessage("Hello world this is a test message")

	newTokens := conv.GetTokenCount()
	if newTokens <= initialTokens {
		t.Error("token count should increase after adding message")
	}

	// Verify count is reasonable (not 0, not crazy high)
	// With hybrid tokenizer, a short message should be 10-30 tokens
	if newTokens < 5 || newTokens > 50 {
		t.Errorf("token count %d seems unreasonable for short message (expected 5-50)", newTokens)
	}
}

// Test threshold configuration
func TestFullAgentLoop_ThresholdConfiguration(t *testing.T) {
	conv := agent.NewConversation("test")

	// Test default thresholds
	if conv.MaxMessages != 20 {
		t.Errorf("default MaxMessages = %d, want 20", conv.MaxMessages)
	}
	if conv.MaxTokenPct != 0.75 {
		t.Errorf("default MaxTokenPct = %f, want 0.75", conv.MaxTokenPct)
	}

	// Test custom thresholds
	conv.SetThresholds(10, 0.5, 4000)
	if conv.MaxMessages != 10 {
		t.Errorf("custom MaxMessages = %d, want 10", conv.MaxMessages)
	}
	if conv.MaxTokenPct != 0.5 {
		t.Errorf("custom MaxTokenPct = %f, want 0.5", conv.MaxTokenPct)
	}
}

// Test conversation message management
func TestFullAgentLoop_ConversationManagement(t *testing.T) {
	conv := agent.NewConversation("test")

	// Add messages
	conv.AddUserMessage("First message")
	conv.AddAssistantMessage("Response")
	conv.AddUserMessage("Second message")

	if conv.GetMessageCount() != 3 {
		t.Errorf("message count = %d, want 3", conv.GetMessageCount())
	}

	// Get recent messages
	recent := conv.GetRecentMessages(2)
	if len(recent) != 2 {
		t.Errorf("recent messages count = %d, want 2", len(recent))
	}

	// Verify message roles
	if recent[0].Role != "assistant" {
		t.Errorf("first recent role = %s, want assistant", recent[0].Role)
	}
	if recent[1].Role != "user" {
		t.Errorf("second recent role = %s, want user", recent[1].Role)
	}
}

// Test token estimate accuracy for various text types
func TestFullAgentLoop_TokenEstimateAccuracy(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty",
			text:      "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "short sentence",
			text:      "Hello world",
			minTokens: 3,
			maxTokens: 10,
		},
		{
			name:      "medium paragraph",
			text:      "The quick brown fox jumps over the lazy dog. This is a test sentence.",
			minTokens: 10,
			maxTokens: 25,
		},
		{
			name:      "with newlines",
			text:      "Line 1\nLine 2\nLine 3",
			minTokens: 6,
			maxTokens: 15,
		},
		{
			name:      "code block",
			text:      "\`\`\`go\nfmt.Println(\"hello\")\n\`\`\`",
			minTokens: 8,
			maxTokens: 20,
		},
		{
			name:      "long word",
			text:      "supercalifragilisticexpialidocious",
			minTokens: 10,
			maxTokens: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.EstimateTokens(tt.text)
			if got < tt.minTokens || got > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
					tt.text, got, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

// Test token threshold based on model capacity
func TestFullAgentLoop_TokenThreshold(t *testing.T) {
	conv := agent.NewConversation("test")
	conv.SetThresholds(100, 0.75, 1000) // 75% of 1000 = 750 tokens

	// Add messages that total ~740 tokens (just under 75% threshold)
	// Each word roughly 2 tokens with new hybrid estimator
	// 370 words * 2 tokens = ~740 tokens
	content := "word "
	for i := 0; i < 369; i++ {
		content += "word "
	}
	conv.AddMessage(agent.RoleUser, content)

	// 740 tokens is 74%, should NOT trigger
	if conv.NeedsSummarization() {
		t.Errorf("NeedsSummarization() = true for 74%% tokens, want false")
	}

	// Add more to push over 75%
	conv.AddMessage(agent.RoleUser, "more words here to exceed threshold")

	// Now should trigger
	if !conv.NeedsSummarization() {
		t.Error("should need summarization after exceeding 75% token threshold")
	}
}

// Test context usage tracking
func TestFullAgentLoop_GetContextUsage(t *testing.T) {
	conv := agent.NewConversation("test")
	conv.SetThresholds(20, 0.75, 1000)

	// Add 5 messages
	for i := 0; i < 5; i++ {
		conv.AddMessage(agent.RoleUser, "test message content")
	}

	messages, tokens, pct := conv.GetContextUsage()
	if messages != 5 {
		t.Errorf("messages = %d, want 5", messages)
	}
	if tokens <= 0 {
		t.Error("tokens should be > 0")
	}
	if pct <= 0 || pct > 1 {
		t.Errorf("percentage %f should be between 0 and 1", pct)
	}
}

// Test validate estimate function
func TestFullAgentLoop_ValidateEstimate(t *testing.T) {
	// Perfect match
	ratio := agent.ValidateEstimate(100, 100)
	if ratio != 1.0 {
		t.Errorf("perfect match ratio = %f, want 1.0", ratio)
	}

	// Overestimate (120 actual = 100)
	ratio = agent.ValidateEstimate(120, 100)
	if ratio != 1.2 {
		t.Errorf("overestimate ratio = %f, want 1.2", ratio)
	}

	// Underestimate (80 actual = 100)
	ratio = agent.ValidateEstimate(80, 100)
	if ratio != 0.8 {
		t.Errorf("underestimate ratio = %f, want 0.8", ratio)
	}

	// Zero actual (should return 1.0 to avoid division by zero)
	ratio = agent.ValidateEstimate(100, 0)
	if ratio != 1.0 {
		t.Errorf("zero actual ratio = %f, want 1.0", ratio)
	}
}
