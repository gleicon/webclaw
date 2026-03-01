//go:build js && wasm

package agent

import (
	"time"
)

// SlidingWindow implements progressive summarization by maintaining
// a condensed summary of older conversation plus recent full messages.
// This allows the conversation to stay within context limits while
// preserving both semantic understanding (summary) and recent context.
type SlidingWindow struct {
	// Summary is the condensed representation of older conversation turns
	Summary string `json:"summary"`

	// RecentMessages contains the last N full messages (typically 3 turns = 6 messages)
	RecentMessages []ConversationMessage `json:"recent_messages"`

	// MessageCount is the total number of messages represented by this window
	// (includes both summarized messages and recent messages)
	MessageCount int `json:"message_count"`

	// LastUpdated tracks when the window was last modified
	LastUpdated time.Time `json:"last_updated"`

	// KeepLastN configures how many recent messages to preserve (default: 6 for 3 turns)
	KeepLastN int `json:"keep_last_n"`
}

// NewSlidingWindow creates a new sliding window with default settings
func NewSlidingWindow() *SlidingWindow {
	return &SlidingWindow{
		RecentMessages: make([]ConversationMessage, 0),
		LastUpdated:    time.Now(),
		KeepLastN:      6, // 3 turns (user + assistant) * 2 = 6 messages
	}
}

// NewSlidingWindowWithSummary creates a sliding window initialized with a summary
func NewSlidingWindowWithSummary(summary string, messageCount int) *SlidingWindow {
	return &SlidingWindow{
		Summary:        summary,
		RecentMessages: make([]ConversationMessage, 0),
		MessageCount:   messageCount,
		LastUpdated:    time.Now(),
		KeepLastN:      6,
	}
}

// SetKeepLastN configures how many recent messages to preserve
func (sw *SlidingWindow) SetKeepLastN(n int) {
	sw.KeepLastN = n
}

// AddMessage adds a new message to the recent messages buffer
// Returns true if the buffer is now full and summarization should be triggered
func (sw *SlidingWindow) AddMessage(msg ConversationMessage) bool {
	sw.RecentMessages = append(sw.RecentMessages, msg)
	sw.LastUpdated = time.Now()
	sw.MessageCount++

	// Check if we need to compact (recent messages exceeded threshold)
	return len(sw.RecentMessages) >= sw.KeepLastN
}

// Compact merges recent messages into the summary using the provided summary text
// This is called after LLM summarization completes
func (sw *SlidingWindow) Compact(newSummary string) {
	sw.Summary = newSummary
	sw.RecentMessages = make([]ConversationMessage, 0) // Clear recent messages after compaction
	sw.LastUpdated = time.Now()
}

// ProgressiveMerge merges previous summary with recent messages for the summarization prompt
// Returns a formatted string suitable for LLM summarization
func (sw *SlidingWindow) ProgressiveMerge() string {
	result := ""

	// Include previous summary if exists
	if sw.Summary != "" {
		result += "Previous context:\n" + sw.Summary + "\n\n"
	}

	// Include recent messages to be incorporated
	if len(sw.RecentMessages) > 0 {
		result += "Recent conversation:\n"
		for _, msg := range sw.RecentMessages {
			result += msg.Role + ": " + msg.Content + "\n"
		}
	}

	return result
}

// GetAllMessages reconstructs the full conversation context
// Returns messages in order: summary-as-system (if any) + recent messages
func (sw *SlidingWindow) GetAllMessages() []ConversationMessage {
	result := make([]ConversationMessage, 0)

	// If we have a summary, add it as a system message at the start
	if sw.Summary != "" {
		summaryMsg := ConversationMessage{
			Role:      string(RoleSystem),
			Content:   "Previous conversation summary: " + sw.Summary,
			ID:        "summary_" + time.Now().Format("20060102150405"),
			Timestamp: sw.LastUpdated,
			Metadata:  map[string]interface{}{"type": "summary"},
		}
		result = append(result, summaryMsg)
	}

	// Add all recent messages
	result = append(result, sw.RecentMessages...)

	return result
}

// GetRecentMessageCount returns the number of messages in the recent buffer
func (sw *SlidingWindow) GetRecentMessageCount() int {
	return len(sw.RecentMessages)
}

// GetTotalMessageCount returns the total count of all messages (summarized + recent)
func (sw *SlidingWindow) GetTotalMessageCount() int {
	return sw.MessageCount
}

// IsEmpty returns true if the sliding window has no content
func (sw *SlidingWindow) IsEmpty() bool {
	return sw.Summary == "" && len(sw.RecentMessages) == 0
}

// EstimateTokenCount returns an estimate of tokens in the current window
func (sw *SlidingWindow) EstimateTokenCount() int {
	total := 0

	// Estimate tokens in summary
	if sw.Summary != "" {
		total += len(sw.Summary) / 4
	}

	// Estimate tokens in recent messages
	for _, msg := range sw.RecentMessages {
		total += len(msg.Content) / 4
	}

	return total
}

// Reset clears the sliding window to empty state
func (sw *SlidingWindow) Reset() {
	sw.Summary = ""
	sw.RecentMessages = make([]ConversationMessage, 0)
	sw.MessageCount = 0
	sw.LastUpdated = time.Now()
}

// ToConversation reconstructs a Conversation from this sliding window
// Useful for creating a conversation object after summarization
func (sw *SlidingWindow) ToConversation(id string) *Conversation {
	conv := NewConversation(id)
	conv.Summary = &Summary{
		ID:           "sum_" + id,
		Content:      sw.Summary,
		MessageCount: sw.MessageCount - len(sw.RecentMessages),
		CreatedAt:    sw.LastUpdated,
	}
	conv.Messages = sw.RecentMessages
	conv.UpdatedAt = sw.LastUpdated
	return conv
}
