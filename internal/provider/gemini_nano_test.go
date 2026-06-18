//go:build js && wasm

package provider

import "testing"

func TestParseMessagesForGemini_SingleTurn(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "hello"},
	}
	sys, user, hist := parseMessagesForGemini(msgs)
	if sys != "" {
		t.Errorf("want empty system prompt, got %q", sys)
	}
	if user != "hello" {
		t.Errorf("want userMsg %q, got %q", "hello", user)
	}
	if len(hist) != 0 {
		t.Errorf("want empty history, got %d entries", len(hist))
	}
}

func TestParseMessagesForGemini_WithSystemPrompt(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "hi"},
	}
	sys, user, hist := parseMessagesForGemini(msgs)
	if sys != "You are helpful." {
		t.Errorf("want system %q, got %q", "You are helpful.", sys)
	}
	if user != "hi" {
		t.Errorf("want userMsg %q, got %q", "hi", user)
	}
	if len(hist) != 0 {
		t.Errorf("want empty history, got %d entries", len(hist))
	}
}

func TestParseMessagesForGemini_MultiTurn(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "turn1"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "turn2"},
	}
	sys, user, hist := parseMessagesForGemini(msgs)
	if sys != "sys" {
		t.Errorf("system: want %q got %q", "sys", sys)
	}
	if user != "turn2" {
		t.Errorf("userMsg: want %q got %q", "turn2", user)
	}
	if len(hist) != 2 {
		t.Fatalf("history: want 2 entries, got %d", len(hist))
	}
	if hist[0].Role != "user" || hist[0].Content != "turn1" {
		t.Errorf("hist[0]: want user/turn1, got %s/%s", hist[0].Role, hist[0].Content)
	}
	if hist[1].Role != "assistant" || hist[1].Content != "resp1" {
		t.Errorf("hist[1]: want assistant/resp1, got %s/%s", hist[1].Role, hist[1].Content)
	}
}

func TestParseMessagesForGemini_Empty(t *testing.T) {
	sys, user, hist := parseMessagesForGemini(nil)
	if sys != "" || user != "" || len(hist) != 0 {
		t.Errorf("want all empty, got sys=%q user=%q hist=%v", sys, user, hist)
	}
}

func TestParseMessagesForGemini_TrailingAssistant(t *testing.T) {
	// Completed exchange with no new user message: both prior turns go to history.
	// userMsg is empty because the last message is not a user message.
	msgs := []Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", Content: "a"},
	}
	_, user, hist := parseMessagesForGemini(msgs)
	if user != "" {
		t.Errorf("want empty userMsg for trailing assistant, got %q", user)
	}
	if len(hist) != 2 {
		t.Fatalf("want 2 history entries, got %d: %v", len(hist), hist)
	}
	if hist[0].Role != "user" || hist[0].Content != "q" {
		t.Errorf("hist[0]: want user/q, got %s/%s", hist[0].Role, hist[0].Content)
	}
	if hist[1].Role != "assistant" || hist[1].Content != "a" {
		t.Errorf("hist[1]: want assistant/a, got %s/%s", hist[1].Role, hist[1].Content)
	}
}
