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
)

// Message represents a single conversation turn
type Message struct {
	ID        string                 `json:"id"`
	Role      Role                   `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Summary represents a condensed conversation history
type Summary struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// Conversation manages the conversation state with automatic summarization
// support. It maintains full messages and optionally a summary of older history.
type Conversation struct {
	ID        string                 `json:"id"`
	Messages  []Message              `json:"messages"`
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
		Messages:       make([]Message, 0),
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
func (c *Conversation) AddMessage(role Role, content string) Message {
	msg := Message{
		ID:        generateMessageID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
	return msg
}

// GetMessages returns all messages in the conversation
func (c *Conversation) GetMessages() []Message {
	return c.Messages
}

// GetRecentMessages returns the last n messages
func (c *Conversation) GetRecentMessages(n int) []Message {
	if n <= 0 {
		return []Message{}
	}
	if n >= len(c.Messages) {
		return c.Messages
	}
	return c.Messages[len(c.Messages)-n:]
}

// GetMessageCount returns the total number of messages
func (c *Conversation) GetMessageCount() int {
	return len(c.Messages)
}

// GetMessageByID retrieves a specific message by ID
func (c *Conversation) GetMessageByID(id string) *Message {
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
	c.Messages = make([]Message, 0)
	c.UpdatedAt = time.Now()
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
