//go:build js && wasm

package agent

import (
	"time"
)

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Summary represents a condensed conversation history
type Summary struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// Message is the basic message format for API calls (OpenAI/Claude compatible)
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant", "tool"
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // For tool messages
}

// ConversationMessage extends Message with conversation management metadata
// Note: Uses composition for API compatibility while adding tracking fields
type ConversationMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Name      string                 `json:"name,omitempty"`
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ToMessage converts a ConversationMessage to a basic Message for API calls
func (cm ConversationMessage) ToMessage() Message {
	return Message{
		Role:    cm.Role,
		Content: cm.Content,
		Name:    cm.Name,
	}
}

// Conversation manages the conversation state with automatic summarization
// support. It maintains full messages and optionally a summary of older history.
type Conversation struct {
	ID        string                 `json:"id"`
	Messages  []ConversationMessage  `json:"messages"`
	Summary   *Summary               `json:"summary,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`

	// Threshold configuration (mutable for testing)
	MaxMessages    int     `json:"max_messages"`     // Default: 20 messages
	MaxTokenPct    float64 `json:"max_token_pct"`    // Default: 0.75 (75%)
	ModelMaxTokens int     `json:"model_max_tokens"` // Default: 8192
}

// NewConversation creates a new conversation with default thresholds
func NewConversation(id string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:             id,
		Messages:       make([]ConversationMessage, 0),
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       make(map[string]interface{}),
		MaxMessages:    20,
		MaxTokenPct:    0.75,
		ModelMaxTokens: 8192,
	}
}

// AddMessage appends a new message to the conversation
// Returns the added message for convenience
func (c *Conversation) AddMessage(role Role, content string) ConversationMessage {
	msg := ConversationMessage{
		Role:      string(role),
		Content:   content,
		ID:        generateMessageID(),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
	return msg
}

// AddAssistantMessage adds an assistant response to the conversation
func (c *Conversation) AddAssistantMessage(content string) ConversationMessage {
	return c.AddMessage(RoleAssistant, content)
}

// AddUserMessage adds a user message to the conversation
func (c *Conversation) AddUserMessage(content string) ConversationMessage {
	return c.AddMessage(RoleUser, content)
}

// GetMessages returns all messages in the conversation
func (c *Conversation) GetMessages() []ConversationMessage {
	return c.Messages
}

// GetMessagesForAPI returns messages in basic Message format for API calls
func (c *Conversation) GetMessagesForAPI() []Message {
	result := make([]Message, len(c.Messages))
	for i, msg := range c.Messages {
		result[i] = msg.ToMessage()
	}
	return result
}

// GetRecentMessages returns the last n messages
func (c *Conversation) GetRecentMessages(n int) []ConversationMessage {
	if n <= 0 {
		return []ConversationMessage{}
	}
	if n >= len(c.Messages) {
		return c.Messages
	}
	return c.Messages[len(c.Messages)-n:]
}

// GetRecentMessagesForAPI returns the last n messages in basic format
func (c *Conversation) GetRecentMessagesForAPI(n int) []Message {
	recent := c.GetRecentMessages(n)
	result := make([]Message, len(recent))
	for i, msg := range recent {
		result[i] = msg.ToMessage()
	}
	return result
}

// GetMessageCount returns the total number of messages
func (c *Conversation) GetMessageCount() int {
	return len(c.Messages)
}

// GetMessageByID retrieves a specific message by ID
func (c *Conversation) GetMessageByID(id string) *ConversationMessage {
	for i := range c.Messages {
		if c.Messages[i].ID == id {
			return &c.Messages[i]
		}
	}
	return nil
}

// SetSummary sets the conversation summary (typically after summarization)
func (c *Conversation) SetSummary(summary *Summary) {
	c.Summary = summary
	c.UpdatedAt = time.Now()
}

// ClearMessages removes all messages (useful after summarization)
func (c *Conversation) ClearMessages() {
	c.Messages = make([]ConversationMessage, 0)
	c.UpdatedAt = time.Now()
}

// estimateTokens returns an estimate of token count using chars/4 heuristic
func estimateTokens(content string) int {
	return len(content) / 4
}

// GetTokenCount returns the estimated total token count for all messages
func (c *Conversation) GetTokenCount() int {
	total := 0
	for _, msg := range c.Messages {
		total += estimateTokens(msg.Content)
	}
	return total
}

// NeedsSummarization checks if summarization should be triggered
// Returns true if message count >= MaxMessages OR token count >= MaxTokenPct of model capacity
func (c *Conversation) NeedsSummarization() bool {
	// Check message count threshold
	if len(c.Messages) >= c.MaxMessages {
		return true
	}

	// Check token percentage threshold
	if c.ModelMaxTokens > 0 {
		tokenCount := c.GetTokenCount()
		maxAllowedTokens := int(float64(c.ModelMaxTokens) * c.MaxTokenPct)
		if tokenCount >= maxAllowedTokens {
			return true
		}
	}

	return false
}

// GetContextUsage returns current usage stats for monitoring
func (c *Conversation) GetContextUsage() (messages int, tokens int, tokenPct float64) {
	tokens = c.GetTokenCount()
	if c.ModelMaxTokens > 0 {
		tokenPct = float64(tokens) / float64(c.ModelMaxTokens)
	}
	return len(c.Messages), tokens, tokenPct
}

// SetThresholds allows dynamic threshold configuration
func (c *Conversation) SetThresholds(maxMessages int, maxTokenPct float64, modelMaxTokens int) {
	c.MaxMessages = maxMessages
	c.MaxTokenPct = maxTokenPct
	c.ModelMaxTokens = modelMaxTokens
}

// generateMessageID creates a unique message ID
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

// randomString generates a short random string for IDs
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		// Simple deterministic for now - replace with crypto/rand if needed
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
