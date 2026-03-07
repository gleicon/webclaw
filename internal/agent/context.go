//go:build js && wasm

package agent

import (
	"context"
	"fmt"
	"strings" // NEW
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/identity"
	"github.com/gleicon/webclaw/internal/memory" // NEW
)

// ContextAssembler builds the complete context for LLM requests
// Includes system prompt + identity files + conversation history
type ContextAssembler struct {
	config        *config.Config
	identityStore *identity.Store
	conversation  *Conversation
	summarizer    *Summarizer  // NEW: for real LLM-based summarization
	memoryStore   memory.Store // NEW: for fact storage
}

// NewContextAssembler creates a new context assembler
func NewContextAssembler(cfg *config.Config, store *identity.Store) *ContextAssembler {
	return &ContextAssembler{
		config:        cfg,
		identityStore: store,
		conversation:  NewConversation(generateConversationID()),
		summarizer:    nil, // Must be set via SetSummarizer
	}
}

// NewContextAssemblerWithConversation creates an assembler with existing conversation
func NewContextAssemblerWithConversation(cfg *config.Config, store *identity.Store, conv *Conversation) *ContextAssembler {
	return &ContextAssembler{
		config:        cfg,
		identityStore: store,
		conversation:  conv,
		summarizer:    nil,
	}
}

// AssembleContext builds the complete message array for a provider request
// Returns messages in format compatible with LLM APIs: [system, ...history, user]
func (ca *ContextAssembler) AssembleContext(userMessage string) ([]Message, error) {
	// Build system prompt from identity files
	systemPrompt, err := ca.buildSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	// Create messages array
	// Start with capacity for system message + existing messages + new user message
	allMessages := ca.conversation.GetMessages()
	messages := make([]Message, 0, len(allMessages)+2)

	// Add system message first if available
	if systemPrompt != "" {
		systemMsg := Message{
			Role:    string(RoleSystem),
			Content: systemPrompt,
		}
		messages = append(messages, systemMsg)
	}

	// Add conversation history (convert ConversationMessage to Message)
	for _, cm := range allMessages {
		messages = append(messages, cm.ToMessage())
	}

	// Add current user message
	userMsg := Message{
		Role:    string(RoleUser),
		Content: userMessage,
	}
	messages = append(messages, userMsg)

	return messages, nil
}

// AssembleAndAdd builds context and adds the user message to conversation history
// This is the typical flow: build context → send to LLM → later add response
func (ca *ContextAssembler) AssembleAndAdd(userMessage string) ([]Message, error) {
	// First build the context for the request
	messages, err := ca.AssembleContext(userMessage)
	if err != nil {
		return nil, err
	}

	// Add user message to conversation history for future turns
	ca.conversation.AddUserMessage(userMessage)

	return messages, nil
}

// AddAssistantResponse adds an assistant response to the conversation history
func (ca *ContextAssembler) AddAssistantResponse(content string) ConversationMessage {
	return ca.conversation.AddAssistantMessage(content)
}

// buildSystemPrompt assembles the system prompt from identity files
func (ca *ContextAssembler) buildSystemPrompt() (string, error) {
	if ca.identityStore == nil {
		// Fallback: return basic system prompt
		return ca.buildFallbackSystemPrompt(), nil
	}

	result, err := identity.AssembleSystemPrompt(ca.identityStore, ca.config)
	if err != nil {
		return "", err
	}

	return result.SystemPrompt, nil
}

// buildFallbackSystemPrompt creates a minimal system prompt when identity is unavailable
func (ca *ContextAssembler) buildFallbackSystemPrompt() string {
	if ca.config == nil {
		return "You are an AI assistant."
	}
	return fmt.Sprintf("You are %s, an AI assistant.", ca.config.Identity.Name)
}

// GetConversation returns the underlying conversation
func (ca *ContextAssembler) GetConversation() *Conversation {
	return ca.conversation
}

// SetConversation replaces the underlying conversation (used for import)
func (ca *ContextAssembler) SetConversation(conv *Conversation) {
	ca.conversation = conv
}

// SetSummarizer wires the summarizer for conversation management
func (ca *ContextAssembler) SetSummarizer(s *Summarizer) {
	ca.summarizer = s
}

// SetMemoryStore wires the memory store for fact persistence
func (ca *ContextAssembler) SetMemoryStore(store memory.Store) {
	ca.memoryStore = store
}

// GetHistoryCount returns the number of messages in conversation history
func (ca *ContextAssembler) GetHistoryCount() int {
	return ca.conversation.GetMessageCount()
}

// ClearHistory clears the conversation history
func (ca *ContextAssembler) ClearHistory() {
	ca.conversation.ClearMessages()
}

// CheckAndSummarize checks if conversation needs summarization and triggers it
// Returns the summary and true if summarization was triggered
func (ca *ContextAssembler) CheckAndSummarize(ctx context.Context) (*Summary, bool) {
	// Check if summarization is needed
	if !ca.conversation.NeedsSummarization() {
		return nil, false
	}

	js.Global().Get("console").Call("log",
		"webclaw: summarization triggered -",
		ca.conversation.GetMessageCount(), "messages")

	// PHASE 6-4: Memory flush before summarization
	// Extract and store key facts so they aren't lost
	if ca.summarizer != nil && ca.memoryStore != nil {
		go func() {
			// Extract key facts (async so we don't block)
			facts, err := ca.summarizer.ExtractKeyFacts(ctx, ca.conversation.GetMessages())
			if err != nil {
				js.Global().Get("console").Call("error",
					"webclaw: fact extraction failed:", err.Error())
				return
			}

			if len(facts) == 0 {
				js.Global().Get("console").Call("log",
					"webclaw: no key facts to extract")
				return
			}

			js.Global().Get("console").Call("log",
				"webclaw: extracted", len(facts), "key facts for memory")

			// Store each fact to memory store
			for i, fact := range facts {
				doc := memory.NewMemoryDocument(
					generateMemoryID(),
					fact,
					nil, // No embedding for now (can be added later)
				)
				doc.Metadata = map[string]interface{}{
					"type":            "conversation_fact",
					"source":          "pre_summarization_flush",
					"extracted_at":    time.Now().Format(time.RFC3339),
					"fact_index":      i,
					"conversation_id": ca.conversation.ID,
				}

				if err := ca.memoryStore.Store(doc); err != nil {
					js.Global().Get("console").Call("error",
						"webclaw: failed to store fact:", err.Error())
					// Continue with other facts
				}
			}

			js.Global().Get("console").Call("log",
				"webclaw: stored", len(facts), "facts to memory")

			// Also append to MEMORY.md identity file
			if ca.identityStore != nil {
				// Format facts for MEMORY.md
				var memoryContent strings.Builder
				memoryContent.WriteString(fmt.Sprintf("\n\n## Facts from %s\n\n",
					time.Now().Format("2006-01-02 15:04")))
				for _, fact := range facts {
					memoryContent.WriteString(fmt.Sprintf("- %s\n", fact))
				}

				if err := ca.identityStore.AppendToMemoryFile(memoryContent.String()); err != nil {
					js.Global().Get("console").Call("error",
						"webclaw: failed to append to MEMORY.md:", err.Error())
				} else {
					js.Global().Get("console").Call("log",
						"webclaw: appended facts to MEMORY.md")
				}
			}
		}()
	}

	// If no summarizer configured, use placeholder (backwards compatibility)
	if ca.summarizer == nil {
		js.Global().Get("console").Call("warn",
			"webclaw: no summarizer configured, using placeholder")

		msgCount, tokenCount, tokenPct := ca.conversation.GetContextUsage()
		summary := &Summary{
			ID:           generateMessageID(),
			Content:      fmt.Sprintf("Conversation summary: %d messages, ~%d tokens (%.1f%% of context)", msgCount, tokenCount, tokenPct*100),
			MessageCount: msgCount,
			CreatedAt:    time.Now(),
		}
		ca.conversation.SetSummary(summary)
		ca.conversation.ClearMessages()
		return summary, true
	}

	// Real LLM-based summarization
	start := time.Now()
	result, err := ca.summarizer.SummarizeConversation(ctx, ca.conversation)
	if err != nil {
		js.Global().Get("console").Call("error",
			"webclaw: summarization failed:", err.Error())
		// Don't block on summarization failure
		return nil, false
	}

	duration := time.Since(start)
	js.Global().Get("console").Call("log",
		"webclaw: summarization complete -",
		result.TokenCount, "tokens in", duration.Seconds(), "s")

	// Create summary object
	summary := &Summary{
		ID:           generateMessageID(),
		Content:      result.Summary,
		MessageCount: result.MessageCount,
		CreatedAt:    time.Now(),
	}

	// PHASE 6-3: Keep last 2 messages for continuity
	// Get last 2 messages before clearing
	recentMessages := ca.conversation.GetRecentMessages(2)

	// Set summary and clear old messages
	ca.conversation.SetSummary(summary)
	ca.conversation.ClearMessages()

	// Restore last 2 messages for context continuity
	for _, msg := range recentMessages {
		ca.conversation.AddMessage(Role(msg.Role), msg.Content)
	}

	js.Global().Get("console").Call("log",
		"webclaw: conversation compacted - summary +", len(recentMessages), "recent messages")

	return summary, true
}

// EstimateTokens returns approximate token count for the complete context
func (ca *ContextAssembler) EstimateTokens(userMessage string) int {
	// System prompt tokens
	systemPrompt, _ := ca.buildSystemPrompt()
	total := estimateTokens(systemPrompt)

	// History tokens
	for _, msg := range ca.conversation.GetMessages() {
		total += estimateTokens(msg.Content)
		// Add overhead for role labels
		total += 4
	}

	// New user message tokens
	total += estimateTokens(userMessage)
	total += 4 // Overhead

	return total
}

// WillFitInContext checks if the message would fit within model context limits
func (ca *ContextAssembler) WillFitInContext(userMessage string, maxTokens int) bool {
	estimated := ca.EstimateTokens(userMessage)
	return estimated < int(float64(maxTokens)*0.9) // 90% threshold
}

// generateConversationID creates a unique conversation ID
func generateConversationID() string {
	return "conv_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}
