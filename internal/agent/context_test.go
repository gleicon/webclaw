//go:build js && wasm

package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/identity"
	"github.com/gleicon/webclaw/internal/memory"
	"github.com/gleicon/webclaw/internal/provider"
)

// TestContextAssembler_CheckAndSummarize_Real tests real LLM-based summarization
func TestContextAssembler_CheckAndSummarize_Real(t *testing.T) {
	// Create mock summarizer
	mockProvider := &mockSummaryProvider{}
	summarizer := NewSummarizer(mockProvider)

	// Create assembler with summarizer
	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	assembler.SetSummarizer(summarizer)

	// Add 20 messages to trigger threshold (default MaxMessages is 20)
	for i := 0; i < 20; i++ {
		assembler.GetConversation().AddUserMessage("Test message " + string(rune('0'+i%10)))
	}

	// Call CheckAndSummarize
	ctx := context.Background()
	summary, triggered := assembler.CheckAndSummarize(ctx)

	if !triggered {
		t.Error("expected summarization to be triggered")
	}
	if summary == nil {
		t.Fatal("expected summary to be returned")
	}
	if summary.Content != "Mock summary of conversation" {
		t.Errorf("unexpected summary content: %s", summary.Content)
	}

	// Verify messages cleared but last 2 kept
	conv := assembler.GetConversation()
	if len(conv.Messages) != 2 {
		t.Errorf("expected 2 recent messages kept, got %d", len(conv.Messages))
	}

	t.Logf("Summarization successful: %d messages → %s", summary.MessageCount, summary.Content)
}

// TestContextAssembler_CheckAndSummarize_NoSummarizer tests placeholder behavior
func TestContextAssembler_CheckAndSummarize_NoSummarizer(t *testing.T) {
	// Create assembler without summarizer (placeholder mode)
	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	// No SetSummarizer call - should use placeholder

	// Add 20 messages to trigger threshold
	for i := 0; i < 20; i++ {
		assembler.GetConversation().AddUserMessage("Test message")
	}

	ctx := context.Background()
	summary, triggered := assembler.CheckAndSummarize(ctx)

	if !triggered {
		t.Error("expected summarization to be triggered")
	}
	if summary == nil {
		t.Fatal("expected summary to be returned")
	}

	// Should be placeholder content (contains "Conversation summary")
	if !strings.Contains(summary.Content, "Conversation summary") {
		t.Errorf("expected placeholder content, got: %s", summary.Content)
	}

	// With placeholder, all messages should be cleared (no continuity preservation)
	conv := assembler.GetConversation()
	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages after placeholder summarization, got %d", len(conv.Messages))
	}
}

// TestContextAssembler_CheckAndSummarize_NotTriggered tests no summarization below threshold
func TestContextAssembler_CheckAndSummarize_NotTriggered(t *testing.T) {
	mockProvider := &mockSummaryProvider{}
	summarizer := NewSummarizer(mockProvider)

	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	assembler.SetSummarizer(summarizer)

	// Add only 10 messages (below 20 threshold)
	for i := 0; i < 10; i++ {
		assembler.GetConversation().AddUserMessage("Test message")
	}

	ctx := context.Background()
	summary, triggered := assembler.CheckAndSummarize(ctx)

	if triggered {
		t.Error("expected summarization NOT to be triggered")
	}
	if summary != nil {
		t.Error("expected no summary when not triggered")
	}

	// All messages should still be there
	conv := assembler.GetConversation()
	if len(conv.Messages) != 10 {
		t.Errorf("expected 10 messages, got %d", len(conv.Messages))
	}
}

// TestContextAssembler_CheckAndSummarize_ErrorHandling tests error handling
func TestContextAssembler_CheckAndSummarize_ErrorHandling(t *testing.T) {
	// Create mock provider that returns error
	errorProvider := &mockErrorProvider{}
	summarizer := NewSummarizer(errorProvider)

	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	assembler.SetSummarizer(summarizer)

	// Add 20 messages to trigger threshold
	for i := 0; i < 20; i++ {
		assembler.GetConversation().AddUserMessage("Test message")
	}

	ctx := context.Background()
	summary, triggered := assembler.CheckAndSummarize(ctx)

	// Should NOT trigger when summarizer fails (graceful degradation)
	if triggered {
		t.Error("expected summarization NOT to trigger on error")
	}
	if summary != nil {
		t.Error("expected no summary when summarization fails")
	}

	// All messages should still be there
	conv := assembler.GetConversation()
	if len(conv.Messages) != 20 {
		t.Errorf("expected 20 messages preserved after error, got %d", len(conv.Messages))
	}
}

// TestContextAssembler_CheckAndSummarize_ContextCancellation tests context cancellation
func TestContextAssembler_CheckAndSummarize_ContextCancellation(t *testing.T) {
	// Create a slow mock provider
	slowProvider := &mockSlowProvider{}
	summarizer := NewSummarizer(slowProvider)

	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	assembler.SetSummarizer(summarizer)

	// Add 20 messages to trigger threshold
	for i := 0; i < 20; i++ {
		assembler.GetConversation().AddUserMessage("Test message")
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	summary, triggered := assembler.CheckAndSummarize(ctx)

	// Should NOT trigger when context is cancelled
	if triggered {
		t.Error("expected summarization NOT to trigger on cancelled context")
	}
	if summary != nil {
		t.Error("expected no summary when context is cancelled")
	}
}

// TestContextAssembler_SetSummarizer tests the setter method
func TestContextAssembler_SetSummarizer(t *testing.T) {
	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)

	// Initially nil
	if assembler.summarizer != nil {
		t.Error("expected summarizer to be nil initially")
	}

	// Set summarizer
	mockProvider := &mockSummaryProvider{}
	summarizer := NewSummarizer(mockProvider)
	assembler.SetSummarizer(summarizer)

	// Now should be set
	if assembler.summarizer == nil {
		t.Error("expected summarizer to be set after SetSummarizer")
	}
}

// mockSummaryProvider is a mock provider that returns a deterministic summary
type mockSummaryProvider struct{}

func (mp *mockSummaryProvider) Stream(ctx context.Context, messages []Message, tools []map[string]interface{}, callback func(provider.Token)) error {
	words := []string{"Mock", "summary", "of", "conversation"}
	for i, word := range words {
		tok := provider.Token{Text: word + " "}
		if i == len(words)-1 {
			tok.FinishReason = "stop"
		}
		callback(tok)
	}
	return nil
}

func (mp *mockSummaryProvider) GetName() string  { return "mock" }
func (mp *mockSummaryProvider) GetModel() string { return "test-model" }

// mockErrorProvider is a mock provider that returns an error
type mockErrorProvider struct{}

func (mep *mockErrorProvider) Stream(ctx context.Context, messages []Message, tools []map[string]interface{}, callback func(provider.Token)) error {
	return context.Canceled
}

func (mep *mockErrorProvider) GetName() string  { return "error-mock" }
func (mep *mockErrorProvider) GetModel() string { return "test-model" }

// mockSlowProvider is a mock provider that simulates slow streaming
type mockSlowProvider struct{}

func (msp *mockSlowProvider) Stream(ctx context.Context, messages []Message, tools []map[string]interface{}, callback func(provider.Token)) error {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate slow operation
	time.Sleep(100 * time.Millisecond)

	// Check again
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tok := provider.Token{Text: "slow summary", FinishReason: "stop"}
	callback(tok)
	return nil
}

func (msp *mockSlowProvider) GetName() string  { return "slow-mock" }
func (msp *mockSlowProvider) GetModel() string { return "test-model" }

// TestContextAssembler_CheckAndSummarize_Continuity tests that context continuity is maintained
func TestContextAssembler_CheckAndSummarize_Continuity(t *testing.T) {
	mockProvider := &mockSummaryProvider{}
	summarizer := NewSummarizer(mockProvider)

	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := NewContextAssembler(cfg, store)
	assembler.SetSummarizer(summarizer)

	// Add messages with specific content to verify continuity
	for i := 0; i < 18; i++ {
		assembler.GetConversation().AddUserMessage("Old message " + string(rune('0'+i%10)))
		assembler.GetConversation().AddAssistantMessage("Old response " + string(rune('0'+i%10)))
	}
	// Last 2 messages that should be preserved
	assembler.GetConversation().AddUserMessage("Recent user question")
	assembler.GetConversation().AddAssistantMessage("Recent assistant answer")

	ctx := context.Background()
	summary, triggered := assembler.CheckAndSummarize(ctx)

	if !triggered {
		t.Error("expected summarization to be triggered")
	}
	if summary == nil {
		t.Fatal("expected summary")
	}

	// Verify last 2 messages are preserved with correct content
	conv := assembler.GetConversation()
	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
	}

	// Check that the last 2 messages are preserved (user question, assistant answer)
	lastUser := conv.Messages[0]
	lastAssistant := conv.Messages[1]

	if lastUser.Content != "Recent user question" || lastUser.Role != "user" {
		t.Errorf("expected recent user question, got: %s (role: %s)", lastUser.Content, lastUser.Role)
	}
	if lastAssistant.Content != "Recent assistant answer" || lastAssistant.Role != "assistant" {
		t.Errorf("expected recent assistant answer, got: %s (role: %s)", lastAssistant.Content, lastAssistant.Role)
	}

	t.Logf("Context continuity preserved: summary + %d recent messages", len(conv.Messages))
}

// TestContextAssembler_MemoryFlush tests memory flush before summarization
func TestContextAssembler_MemoryFlush(t *testing.T) {
	// Create mock summarizer that extracts facts
	mockProvider := &mockFactProvider{
		facts: []string{
			"User's name is Alice",
			"User works at Acme Corp",
			"User prefers Go programming",
		},
	}
	summarizer := NewSummarizer(mockProvider)

	// Create mock memory store
	mockMemStore := &mockMemoryStore{
		storedDocs: make([]*mockMemoryDoc, 0),
	}

	// Create assembler with dependencies
	cfg := &config.Config{}
	assembler := NewContextAssembler(cfg, &identity.Store{})
	assembler.SetSummarizer(summarizer)
	assembler.SetMemoryStore(mockMemStore)

	// Add 20 messages to trigger summarization threshold
	for i := 0; i < 20; i++ {
		assembler.GetConversation().AddUserMessage("Test message " + string(rune('A'+i%26)))
	}

	// Call CheckAndSummarize
	ctx := context.Background()
	_, triggered := assembler.CheckAndSummarize(ctx)

	if !triggered {
		t.Error("expected summarization to be triggered")
	}

	// Wait for async flush to complete
	time.Sleep(150 * time.Millisecond)

	// Verify facts were stored in memory
	if len(mockMemStore.storedDocs) != 3 {
		t.Errorf("expected 3 facts stored, got %d", len(mockMemStore.storedDocs))
	}

	// Verify facts content
	expectedFacts := map[string]bool{
		"User's name is Alice":        false,
		"User works at Acme Corp":     false,
		"User prefers Go programming": false,
	}
	for _, doc := range mockMemStore.storedDocs {
		if _, ok := expectedFacts[doc.content]; ok {
			expectedFacts[doc.content] = true
		}
		// Check metadata
		if doc.metadata["type"] != "conversation_fact" {
			t.Errorf("expected type=conversation_fact, got %v", doc.metadata["type"])
		}
		if doc.metadata["source"] != "pre_summarization_flush" {
			t.Errorf("expected source=pre_summarization_flush, got %v", doc.metadata["source"])
		}
	}

	for fact, found := range expectedFacts {
		if !found {
			t.Errorf("expected fact not found: %s", fact)
		}
	}

	t.Logf("Memory flush verified: %d facts stored", len(mockMemStore.storedDocs))
}

// mockFactProvider is a mock provider that returns facts for extraction
type mockFactProvider struct {
	facts []string
}

func (mfp *mockFactProvider) Stream(ctx context.Context, messages []Message, tools []map[string]interface{}, callback func(provider.Token)) error {
	// Check if this is a fact extraction call (contains "Extract key facts")
	isExtraction := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "Extract key facts") {
			isExtraction = true
			break
		}
	}

	if isExtraction {
		// Return facts as bullet points
		var response strings.Builder
		for _, fact := range mfp.facts {
			response.WriteString("- " + fact + "\n")
		}
		tok := provider.Token{Text: response.String(), FinishReason: "stop"}
		callback(tok)
	} else {
		// Return summary
		tok := provider.Token{Text: "Mock summary of conversation", FinishReason: "stop"}
		callback(tok)
	}
	return nil
}

func (mfp *mockFactProvider) GetName() string  { return "mock-fact" }
func (mfp *mockFactProvider) GetModel() string { return "test-model" }

// mockMemoryStore is a mock memory store for testing
type mockMemoryStore struct {
	storedDocs []*mockMemoryDoc
}

type mockMemoryDoc struct {
	id       string
	content  string
	metadata map[string]interface{}
}

func (mms *mockMemoryStore) Store(doc *memory.MemoryDocument) error {
	mms.storedDocs = append(mms.storedDocs, &mockMemoryDoc{
		id:       doc.ID,
		content:  doc.Content,
		metadata: doc.Metadata,
	})
	return nil
}

func (mms *mockMemoryStore) Get(id string) (*memory.MemoryDocument, error) { return nil, nil }
func (mms *mockMemoryStore) Delete(id string) error                        { return nil }
func (mms *mockMemoryStore) Search(query string, opts memory.SearchOptions) ([]*memory.MemorySearchResult, error) {
	return nil, nil
}
func (mms *mockMemoryStore) GetAll() ([]*memory.MemoryDocument, error) { return nil, nil }
func (mms *mockMemoryStore) CheckQuota() (*memory.QuotaInfo, error) {
	return &memory.QuotaInfo{Usage: 0, Quota: 1000000, Percent: 0}, nil
}
func (mms *mockMemoryStore) EvictIfNeeded() error { return nil }
