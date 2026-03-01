//go:build js && wasm

package agent

import (
	"context"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/provider"
)

// Summarizer handles conversation summarization via LLM calls
type Summarizer struct {
	provider Provider
}

// NewSummarizer creates a new summarizer with the given provider
func NewSummarizer(provider Provider) *Summarizer {
	return &Summarizer{
		provider: provider,
	}
}

// SummarizeRequest contains the input for summarization
type SummarizeRequest struct {
	// PreviousSummary is the existing summary (empty for first summarization)
	PreviousSummary string

	// MessagesToSummarize are the recent messages to include in the new summary
	MessagesToSummarize []ConversationMessage

	// MaxLength is the maximum desired length of the summary in tokens (approximate)
	MaxLength int
}

// SummarizeResult contains the summarization output
type SummarizeResult struct {
	Summary      string
	TokenCount   int
	MessageCount int
	Duration     time.Duration
}

// Summarize performs conversation summarization
// Takes previous summary + recent messages, returns condensed summary
func (s *Summarizer) Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResult, error) {
	start := time.Now()

	// Build the summarization prompt
	prompt := buildSummarizationPrompt(req)

	// Create messages for the LLM call
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a conversation summarizer. Create concise, accurate summaries that preserve key facts, decisions, and context. Focus on what was discussed, what was decided, and any action items.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call provider for summary
	var summary strings.Builder
	err := s.provider.Stream(ctx, messages, func(tok provider.Token) {
		summary.WriteString(tok.Text)
	})

	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	summaryText := strings.TrimSpace(summary.String())

	// Estimate token count
	tokenCount := len(summaryText) / 4

	return &SummarizeResult{
		Summary:      summaryText,
		TokenCount:   tokenCount,
		MessageCount: len(req.MessagesToSummarize),
		Duration:     time.Since(start),
	}, nil
}

// buildSummarizationPrompt creates the prompt for the LLM summarizer
func buildSummarizationPrompt(req SummarizeRequest) string {
	var parts []string

	parts = append(parts, "Please summarize the following conversation.")

	if req.MaxLength > 0 {
		parts = append(parts, fmt.Sprintf("Keep the summary under approximately %d tokens.", req.MaxLength))
	}

	parts = append(parts, "")

	// Include previous summary if it exists
	if req.PreviousSummary != "" {
		parts = append(parts, "=== PREVIOUS CONTEXT ===")
		parts = append(parts, req.PreviousSummary)
		parts = append(parts, "")
		parts = append(parts, "=== RECENT MESSAGES TO INCORPORATE ===")
	} else {
		parts = append(parts, "=== CONVERSATION TO SUMMARIZE ===")
	}

	// Add messages to summarize
	for _, msg := range req.MessagesToSummarize {
		role := msg.Role
		if role == "" {
			role = "unknown"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", role, msg.Content))
	}

	parts = append(parts, "")
	parts = append(parts, "=== SUMMARY ===")
	parts = append(parts, "Provide a concise summary that captures:")
	parts = append(parts, "- Main topics discussed")
	parts = append(parts, "- Key facts and information shared")
	parts = append(parts, "- Decisions made or conclusions reached")
	parts = append(parts, "- Any open questions or next steps")

	return strings.Join(parts, "\n")
}

// SummarizeConversation is a convenience function that summarizes from a Conversation
// It uses the sliding window pattern: summary + recent messages
func (s *Summarizer) SummarizeConversation(ctx context.Context, conv *Conversation) (*SummarizeResult, error) {
	// Get existing summary if any
	previousSummary := ""
	if conv.Summary != nil {
		previousSummary = conv.Summary.Content
	}

	// Get all messages to summarize (all of them since we'll replace with summary)
	messages := conv.GetMessages()

	// If no messages, nothing to summarize
	if len(messages) == 0 {
		return &SummarizeResult{
			Summary:      previousSummary,
			TokenCount:   len(previousSummary) / 4,
			MessageCount: 0,
			Duration:     0,
		}, nil
	}

	req := SummarizeRequest{
		PreviousSummary:     previousSummary,
		MessagesToSummarize: messages,
		MaxLength:           500, // Default max summary length
	}

	return s.Summarize(ctx, req)
}

// SummarizeWithProgressiveWindow uses progressive summarization with a sliding window
// It merges previous summary with new messages incrementally
func (s *Summarizer) SummarizeWithProgressiveWindow(ctx context.Context, sw *SlidingWindow) (*SummarizeResult, error) {
	start := time.Now()

	// Build prompt from progressive merge
	prompt := "Please create or update a conversation summary based on the following context:\n\n"
	prompt += sw.ProgressiveMerge()
	prompt += "\n\n=== INSTRUCTIONS ===\n"
	prompt += "If there's a previous context above, merge it with the recent conversation into an updated summary. "
	prompt += "If this is the first summarization, create a concise summary of the conversation. "
	prompt += "Keep important facts, decisions, and context. Remove repetitive or unimportant details."

	// Create messages for the LLM call
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a conversation summarizer. You excel at progressive summarization - merging previous summaries with new conversation to maintain continuity.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call provider for summary
	var summary strings.Builder
	err := s.provider.Stream(ctx, messages, func(tok provider.Token) {
		summary.WriteString(tok.Text)
	})

	if err != nil {
		return nil, fmt.Errorf("progressive summarization failed: %w", err)
	}

	summaryText := strings.TrimSpace(summary.String())
	tokenCount := len(summaryText) / 4

	return &SummarizeResult{
		Summary:      summaryText,
		TokenCount:   tokenCount,
		MessageCount: sw.GetRecentMessageCount(),
		Duration:     time.Since(start),
	}, nil
}

// CreateMockSummarizer creates a mock summarizer for testing
// Returns deterministic summaries without calling a real LLM
func CreateMockSummarizer() *Summarizer {
	return &Summarizer{
		provider: &mockSummarizerProvider{},
	}
}

// mockSummarizerProvider is a mock provider that returns deterministic summaries
type mockSummarizerProvider struct{}

func (mp *mockSummarizerProvider) Stream(ctx context.Context, messages []Message, callback func(tok provider.Token)) error {
	// Extract the conversation content from the prompt
	var conversationContent string
	for _, msg := range messages {
		if msg.Role == "user" {
			conversationContent = msg.Content
			break
		}
	}

	// Generate a simple mock summary based on content
	var summary string
	if strings.Contains(conversationContent, "user:") {
		summary = "The user and assistant discussed various topics. Key points were exchanged and questions were answered."
	} else {
		summary = "Conversation summary covering the main topics and outcomes discussed."
	}

	// Stream the summary word by word
	words := strings.Fields(summary)
	for i, word := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			tok := provider.Token{Text: word + " "}
			if i == len(words)-1 {
				tok.FinishReason = "stop"
			}
			callback(tok)
			// Small artificial delay to simulate streaming
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

func (mp *mockSummarizerProvider) GetName() string {
	return "mock-summarizer"
}

func (mp *mockSummarizerProvider) GetModel() string {
	return "mock"
}

// ExtractKeyFacts extracts key facts from a conversation using the LLM
// This is useful for knowledge extraction before summarization
func (s *Summarizer) ExtractKeyFacts(ctx context.Context, messages []ConversationMessage) ([]string, error) {
	if len(messages) == 0 {
		return []string{}, nil
	}

	// Build extraction prompt
	var conversation strings.Builder
	for _, msg := range messages {
		conversation.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	prompt := fmt.Sprintf(`Extract key facts, decisions, and important information from the following conversation.
List each fact as a separate line starting with "- ".
Only include substantive facts, not pleasantries or filler.

Conversation:
%s

Key facts:`, conversation.String())

	// Call LLM for extraction
	extractionMessages := []Message{
		{
			Role:    "system",
			Content: "You are an information extraction specialist. Extract only concrete, verifiable facts from conversations.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	var result strings.Builder
	err := s.provider.Stream(ctx, extractionMessages, func(tok provider.Token) {
		result.WriteString(tok.Text)
	})

	if err != nil {
		return nil, fmt.Errorf("fact extraction failed: %w", err)
	}

	// Parse the result into individual facts
	rawFacts := strings.Split(result.String(), "\n")
	var facts []string
	for _, fact := range rawFacts {
		fact = strings.TrimSpace(fact)
		// Only keep lines that look like list items
		if strings.HasPrefix(fact, "-") || strings.HasPrefix(fact, "*") {
			facts = append(facts, strings.TrimPrefix(strings.TrimPrefix(fact, "-"), "*"))
		}
	}

	return facts, nil
}

// jsLog is a helper for logging from WASM
func jsLog(args ...interface{}) {
	js.Global().Get("console").Call("log", args...)
}
