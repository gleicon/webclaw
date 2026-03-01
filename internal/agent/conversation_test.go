//go:build js && wasm

package agent

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	// Test chars/4 heuristic
	if got := estimateTokens("abcd"); got != 1 {
		t.Errorf("estimateTokens(\"abcd\") = %d, want 1", got)
	}
	if got := estimateTokens("abcdefgh"); got != 2 {
		t.Errorf("estimateTokens(\"abcdefgh\") = %d, want 2", got)
	}
	if got := estimateTokens(""); got != 0 {
		t.Errorf("estimateTokens(\"\") = %d, want 0", got)
	}
}

func TestConversationNeedsSummarization(t *testing.T) {
	conv := NewConversation("test-1")

	// Test message count threshold: 19 messages should NOT trigger
	for i := 0; i < 19; i++ {
		conv.AddMessage(RoleUser, "test message")
	}
	if conv.NeedsSummarization() {
		t.Errorf("NeedsSummarization() = true for 19 messages, want false")
	}

	// Add 1 more (20th) - exactly at threshold, should trigger
	conv.AddMessage(RoleUser, "test message")
	if !conv.NeedsSummarization() {
		t.Errorf("NeedsSummarization() = false for 20 messages, want true")
	}
}

func TestConversationTokenThreshold(t *testing.T) {
	conv := NewConversation("test-2")
	conv.SetThresholds(100, 0.75, 1000) // 75% of 1000 = 750 tokens

	// Add messages that total ~740 tokens (just under 75% threshold)
	// Each "test" is 4 chars = 1 token
	// 740 tokens * 4 = 2960 chars
	content := make([]byte, 2960)
	for i := range content {
		content[i] = 'a'
	}
	conv.AddMessage(RoleUser, string(content))

	// 740 tokens is 74%, should NOT trigger
	if conv.NeedsSummarization() {
		t.Errorf("NeedsSummarization() = true for 74%% tokens, want false")
	}

	// Add more to push over 75%
	conv.AddMessage(RoleUser, string(make([]byte, 100))) // +25 tokens

	// Now at ~765 tokens = 76.5%, should trigger
	if !conv.NeedsSummarization() {
		t.Errorf("NeedsSummarization() = false for 76%%+ tokens, want true")
	}
}

func TestGetContextUsage(t *testing.T) {
	conv := NewConversation("test-3")
	conv.SetThresholds(20, 0.75, 1000)

	// Add 5 messages, each 40 chars = 10 tokens, total 50 tokens
	for i := 0; i < 5; i++ {
		conv.AddMessage(RoleUser, "0123456789012345678901234567890123456789") // 40 chars
	}

	messages, tokens, pct := conv.GetContextUsage()
	if messages != 5 {
		t.Errorf("GetContextUsage() messages = %d, want 5", messages)
	}
	if tokens != 50 {
		t.Errorf("GetContextUsage() tokens = %d, want 50", tokens)
	}
	if pct != 0.05 {
		t.Errorf("GetContextUsage() pct = %f, want 0.05", pct)
	}
}

func TestGetRecentMessages(t *testing.T) {
	conv := NewConversation("test-4")

	for i := 0; i < 10; i++ {
		conv.AddMessage(RoleUser, "message "+string(rune('0'+i)))
	}

	// Get last 3
	recent := conv.GetRecentMessages(3)
	if len(recent) != 3 {
		t.Errorf("GetRecentMessages(3) len = %d, want 3", len(recent))
	}

	// Get all (more than available)
	all := conv.GetRecentMessages(100)
	if len(all) != 10 {
		t.Errorf("GetRecentMessages(100) len = %d, want 10", len(all))
	}

	// Get 0
	none := conv.GetRecentMessages(0)
	if len(none) != 0 {
		t.Errorf("GetRecentMessages(0) len = %d, want 0", len(none))
	}
}
