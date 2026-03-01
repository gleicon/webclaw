//go:build js && wasm

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExtractedFact represents a single fact extracted from conversation
type ExtractedFact struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Category    string                 `json:"category"`   // e.g., "user_preference", "decision", "fact"
	Confidence  float64                `json:"confidence"` // 0.0 to 1.0
	Source      string                 `json:"source"`     // e.g., "conversation_20240301_120000"
	ExtractedAt time.Time              `json:"extracted_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractionResult contains all facts extracted from a conversation segment
type ExtractionResult struct {
	Facts       []ExtractedFact `json:"facts"`
	ExtractedAt time.Time       `json:"extracted_at"`
	SourceID    string          `json:"source_id"` // Conversation ID
}

// FactCategory constants for categorizing extracted information
const (
	FactCategoryPreference = "user_preference"
	FactCategoryDecision   = "decision"
	FactCategoryFact       = "fact"
	FactCategoryActionItem = "action_item"
	FactCategoryTopic      = "topic"
)

// MemoryExtractor extracts durable knowledge from conversation using LLM
type MemoryExtractor struct {
	provider LLMProvider
}

// LLMProvider interface for extraction (subset of full provider interface)
type LLMProvider interface {
	Complete(ctx context.Context, messages []Message, temperature float64) (string, error)
}

// Message is a simple message format for extraction calls
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewMemoryExtractor creates a new extractor with the given LLM provider
func NewMemoryExtractor(provider LLMProvider) *MemoryExtractor {
	return &MemoryExtractor{
		provider: provider,
	}
}

// ExtractFromConversation extracts key facts from conversation messages
// Returns structured facts ready for storage in MEMORY.md
func (me *MemoryExtractor) ExtractFromConversation(ctx context.Context, conversationID string, messages []Message) (*ExtractionResult, error) {
	if len(messages) == 0 {
		return &ExtractionResult{
			Facts:       []ExtractedFact{},
			ExtractedAt: time.Now(),
			SourceID:    conversationID,
		}, nil
	}

	// Build extraction prompt
	prompt := buildExtractionPrompt(messages)

	// Call LLM for extraction
	extractionMessages := []Message{
		{
			Role:    "system",
			Content: "You are a knowledge extraction specialist. Extract concrete facts, user preferences, and decisions from conversations. Output valid JSON only.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := me.provider.Complete(ctx, extractionMessages, 0.3) // Low temperature for consistency
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Parse the extracted facts
	facts, err := parseExtractionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extraction: %w", err)
	}

	// Add metadata to each fact
	result := &ExtractionResult{
		Facts:       make([]ExtractedFact, 0, len(facts)),
		ExtractedAt: time.Now(),
		SourceID:    conversationID,
	}

	for _, fact := range facts {
		fact.ID = generateFactID()
		fact.Source = conversationID
		fact.ExtractedAt = result.ExtractedAt
		if fact.Metadata == nil {
			fact.Metadata = make(map[string]interface{})
		}
		result.Facts = append(result.Facts, fact)
	}

	return result, nil
}

// buildExtractionPrompt creates the LLM prompt for knowledge extraction
func buildExtractionPrompt(messages []Message) string {
	var parts []string

	parts = append(parts, "Extract key facts, user preferences, decisions, and important information from the following conversation.")
	parts = append(parts, "")
	parts = append(parts, "Return your response as a JSON array of objects with this structure:")
	parts = append(parts, `[`)
	parts = append(parts, `  {`)
	parts = append(parts, `    "content": "the fact or preference in clear, standalone form",`)
	parts = append(parts, `    "category": "user_preference|decision|fact|action_item|topic",`)
	parts = append(parts, `    "confidence": 0.95`)
	parts = append(parts, `  }`)
	parts = append(parts, `]`)
	parts = append(parts, "")
	parts = append(parts, "Categories:")
	parts = append(parts, "- user_preference: User's likes, dislikes, preferred formats, styles, or settings")
	parts = append(parts, "- decision: Agreements, choices made, conclusions reached")
	parts = append(parts, "- fact: Objective facts or information shared")
	parts = append(parts, "- action_item: Tasks, todos, or follow-ups mentioned")
	parts = append(parts, "- topic: Main subjects discussed (higher-level categorization)")
	parts = append(parts, "")
	parts = append(parts, "Only extract information that would be useful to remember for future conversations.")
	parts = append(parts, "Skip pleasantries, greetings, and temporary information.")
	parts = append(parts, "")
	parts = append(parts, "Conversation:")

	// Add the conversation
	for _, msg := range messages {
		parts = append(parts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
	}

	return strings.Join(parts, "\n")
}

// parseExtractionResponse parses the LLM response into structured facts
func parseExtractionResponse(response string) ([]ExtractedFact, error) {
	// Clean up the response - sometimes LLMs wrap in markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var facts []ExtractedFact
	if err := json.Unmarshal([]byte(response), &facts); err != nil {
		// If parsing fails, try to extract facts manually
		return parseFallback(response), nil
	}

	return facts, nil
}

// parseFallback attempts to extract facts when JSON parsing fails
func parseFallback(response string) []ExtractedFact {
	var facts []ExtractedFact
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "[" || line == "]" || line == "{" || line == "}" {
			continue
		}

		// Try to extract content from rough JSON-like format
		if strings.Contains(line, `"content"`) {
			// Extract between quotes after content
			start := strings.Index(line, `":`)
			if start == -1 {
				continue
			}
			start = strings.Index(line[start:], `"`)
			if start == -1 {
				continue
			}
			start = start + 1 + strings.Index(line, `":`)
			end := strings.Index(line[start:], `"`)
			if end == -1 {
				continue
			}
			content := line[start : start+end]
			if content != "" {
				facts = append(facts, ExtractedFact{
					Content:    content,
					Category:   FactCategoryFact,
					Confidence: 0.7, // Lower confidence for fallback parsing
				})
			}
		}
	}

	return facts
}

// generateFactID creates a unique ID for an extracted fact
func generateFactID() string {
	return fmt.Sprintf("fact_%d_%s", time.Now().UnixNano(), randomString(6))
}

// randomString generates a short random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// ExtractHighConfidenceFacts filters facts by minimum confidence threshold
func (er *ExtractionResult) ExtractHighConfidenceFacts(minConfidence float64) []ExtractedFact {
	var result []ExtractedFact
	for _, fact := range er.Facts {
		if fact.Confidence >= minConfidence {
			result = append(result, fact)
		}
	}
	return result
}

// FilterByCategory returns facts of a specific category
func (er *ExtractionResult) FilterByCategory(category string) []ExtractedFact {
	var result []ExtractedFact
	for _, fact := range er.Facts {
		if fact.Category == category {
			result = append(result, fact)
		}
	}
	return result
}

// MockExtractor creates a mock extractor that returns deterministic test data
func MockExtractor() *MemoryExtractor {
	return &MemoryExtractor{
		provider: &mockLLMProvider{},
	}
}

// mockLLMProvider is a mock provider for testing
type mockLLMProvider struct{}

func (mp *mockLLMProvider) Complete(ctx context.Context, messages []Message, temperature float64) (string, error) {
	// Return mock JSON response
	mockFacts := []ExtractedFact{
		{
			Content:    "User prefers concise responses",
			Category:   FactCategoryPreference,
			Confidence: 0.9,
		},
		{
			Content:    "Project deadline is next Friday",
			Category:   FactCategoryFact,
			Confidence: 0.95,
		},
	}

	data, _ := json.Marshal(mockFacts)
	return string(data), nil
}
