//go:build js && wasm

package agent

import (
	"fmt"
	"time"

	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/identity"
)

// ContextAssembler builds the complete context for LLM requests
// Includes system prompt + identity files + conversation history
type ContextAssembler struct {
	config        *config.Config
	identityStore *identity.Store
	conversation  *Conversation
}

// NewContextAssembler creates a new context assembler
func NewContextAssembler(cfg *config.Config, store *identity.Store) *ContextAssembler {
	return &ContextAssembler{
		config:        cfg,
		identityStore: store,
		conversation:  NewConversation(generateConversationID()),
	}
}

// NewContextAssemblerWithConversation creates an assembler with existing conversation
func NewContextAssemblerWithConversation(cfg *config.Config, store *identity.Store, conv *Conversation) *ContextAssembler {
	return &ContextAssembler{
		config:        cfg,
		identityStore: store,
		conversation:  conv,
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

// GetHistoryCount returns the number of messages in conversation history
func (ca *ContextAssembler) GetHistoryCount() int {
	return ca.conversation.GetMessageCount()
}

// ClearHistory clears the conversation history
func (ca *ContextAssembler) ClearHistory() {
	ca.conversation.ClearMessages()
}

// CheckAndSummarize checks if conversation needs summarization and returns the summary if triggered
func (ca *ContextAssembler) CheckAndSummarize() (*Summary, bool) {
	if ca.conversation.NeedsSummarization() {
		// For now, return a placeholder - summarization would be implemented in a future phase
		// This would call an LLM to summarize the conversation
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
	return nil, false
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
