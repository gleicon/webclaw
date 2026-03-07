package agent

import (
	"encoding/json"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	// Test hybrid word-length algorithm (not chars/4)
	// "abcd" = 4 chars = 1 word (4 chars) = 2 tokens (medium word)
	if got := estimateTokens("abcd"); got != 2 {
		t.Errorf("estimateTokens(\"abcd\") = %d, want 2 (medium word)", got)
	}
	// "abcdefgh" = 8 chars = 1 word (8 chars) = 2 tokens (long word)
	if got := estimateTokens("abcdefgh"); got != 2 {
		t.Errorf("estimateTokens(\"abcdefgh\") = %d, want 2 (long word)", got)
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
	// Set high message threshold so we only test token threshold
	conv.SetThresholds(200, 0.75, 1000) // 75% of 1000 = 750 tokens, max 200 messages

	// Add messages that total ~700 tokens (under 75% threshold)
	// With hybrid algorithm: "a" (1 char) = 1 short word = 1 token
	// "a b c" = 3 short words = 3 tokens + 4 overhead = 7 per message
	// 100 messages × 7 tokens = 700 tokens (~70%)
	for i := 0; i < 100; i++ {
		conv.AddMessage(RoleUser, "a b c")
	}

	// 100 × 7 = 700 tokens is 70%, should NOT trigger
	if conv.NeedsSummarization() {
		_, tokens, pct := conv.GetContextUsage()
		t.Errorf("NeedsSummarization() = true for ~70%% tokens (actual: %d tokens, %.1f%%), want false", tokens, pct*100)
	}

	// Add 10 more messages: 10 × 7 = 70, total 770 (77%)
	for i := 0; i < 10; i++ {
		conv.AddMessage(RoleUser, "a b c")
	}

	// Now at 770 tokens = 77%, should trigger
	if !conv.NeedsSummarization() {
		_, tokens, pct := conv.GetContextUsage()
		t.Errorf("NeedsSummarization() = false for 77%%+ tokens (actual: %d tokens, %.1f%%), want true", tokens, pct*100)
	}
}

func TestGetContextUsage(t *testing.T) {
	conv := NewConversation("test-3")
	conv.SetThresholds(20, 0.75, 1000)

	// Add 5 messages, each with 10 words of 4 chars each
	// Each word "0123" = 4 chars (medium word) = 2 tokens
	// 10 words × 2 tokens = 20 tokens content + 4 overhead = 24 per message
	// 5 messages = 120 tokens total (12%)
	for i := 0; i < 5; i++ {
		conv.AddMessage(RoleUser, "0123 0123 0123 0123 0123 0123 0123 0123 0123 0123")
	}

	messages, tokens, pct := conv.GetContextUsage()
	if messages != 5 {
		t.Errorf("GetContextUsage() messages = %d, want 5", messages)
	}
	if tokens != 120 {
		t.Errorf("GetContextUsage() tokens = %d, want 120", tokens)
	}
	if pct != 0.12 {
		t.Errorf("GetContextUsage() pct = %f, want 0.12", pct)
	}
}

func TestConversationExportImport(t *testing.T) {
	conv := NewConversation("test-export")
	conv.AddUserMessage("Hello")
	conv.AddAssistantMessage("Hi there!")

	// Export
	data, err := conv.ExportToJSON()
	if err != nil {
		t.Fatalf("ExportToJSON() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("ExportToJSON() returned empty data")
	}

	// Verify JSON structure
	var export map[string]interface{}
	if err = json.Unmarshal(data, &export); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, want nil", err)
	}
	if export["version"] != "1.0" {
		t.Errorf("export version = %v, want 1.0", export["version"])
	}
	if export["conversation"] == nil {
		t.Error("export missing conversation field")
	}
	if export["exported_at"] == nil {
		t.Error("export missing exported_at field")
	}

	// Import
	imported, err := ImportFromJSON(data)
	if err != nil {
		t.Fatalf("ImportFromJSON() error = %v, want nil", err)
	}
	if imported.ID != conv.ID {
		t.Errorf("imported.ID = %s, want %s", imported.ID, conv.ID)
	}
	if len(imported.Messages) != len(conv.Messages) {
		t.Errorf("imported messages count = %d, want %d", len(imported.Messages), len(conv.Messages))
	}
	if len(imported.Messages) > 0 && imported.Messages[0].Content != conv.Messages[0].Content {
		t.Errorf("imported.Messages[0].Content = %s, want %s", imported.Messages[0].Content, conv.Messages[0].Content)
	}
}

func TestImportFromJSONEdgeCases(t *testing.T) {
	// Empty data
	_, err := ImportFromJSON([]byte{})
	if err == nil {
		t.Error("ImportFromJSON(empty) expected error, got nil")
	}

	// Invalid JSON
	_, err = ImportFromJSON([]byte("not-json"))
	if err == nil {
		t.Error("ImportFromJSON(invalid JSON) expected error, got nil")
	}

	// Wrong version
	wrongVersion := `{"version":"2.0","exported_at":"2026-01-01T00:00:00Z","conversation":{"id":"test"}}`
	_, err = ImportFromJSON([]byte(wrongVersion))
	if err == nil {
		t.Error("ImportFromJSON(wrong version) expected error, got nil")
	}

	// Missing conversation ID
	missingID := `{"version":"1.0","exported_at":"2026-01-01T00:00:00Z","conversation":{"id":""}}`
	_, err = ImportFromJSON([]byte(missingID))
	if err == nil {
		t.Error("ImportFromJSON(missing ID) expected error, got nil")
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
