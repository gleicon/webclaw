//go:build js && wasm

package gmail

// Message represents a Gmail message
type Message struct {
	ID           string       `json:"id"`
	ThreadID     string       `json:"threadId"`
	LabelIDs     []string     `json:"labelIds"`
	Snippet      string       `json:"snippet"`      // Preview text
	Payload      *MessagePart `json:"payload"`      // Full message structure
	InternalDate int64        `json:"internalDate"` // Unix ms timestamp
}

// MessagePart represents a part of a message (headers, body, or multipart)
type MessagePart struct {
	PartID   string           `json:"partId"`
	MimeType string           `json:"mimeType"`
	Filename string           `json:"filename"`
	Headers  []*MessageHeader `json:"headers"`
	Body     *MessagePartBody `json:"body"`
	Parts    []*MessagePart   `json:"parts"` // For multipart messages
}

// MessageHeader represents a single email header
type MessageHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MessagePartBody represents the body of a message part
type MessagePartBody struct {
	AttachmentID string `json:"attachmentId"`
	Data         string `json:"data"` // base64url encoded
	Size         int    `json:"size"`
}

// ListMessagesResponse is the response from listing messages
type ListMessagesResponse struct {
	Messages           []*Message `json:"messages"`
	NextPageToken      string     `json:"nextPageToken"`
	ResultSizeEstimate int        `json:"resultSizeEstimate"`
}

// Thread represents a Gmail conversation thread
type Thread struct {
	ID        string     `json:"id"`
	Snippet   string     `json:"snippet"`
	HistoryID string     `json:"historyId"`
	Messages  []*Message `json:"messages"`
}

// Label represents a Gmail label
type Label struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"` // "system" or "user"
	MessagesTotal  int    `json:"messagesTotal"`
	MessagesUnread int    `json:"messagesUnread"`
}

// ListLabelsResponse is the response from listing labels
type ListLabelsResponse struct {
	Labels []*Label `json:"labels"`
}

// SendMessageRequest is the request body for sending a message
type SendMessageRequest struct {
	Raw string `json:"raw"` // base64url encoded RFC 2822 message
}

// ModifyMessageRequest is the request body for modifying labels
type ModifyMessageRequest struct {
	AddLabelIDs    []string `json:"addLabelIds,omitempty"`
	RemoveLabelIDs []string `json:"removeLabelIds,omitempty"`
}

// BatchModifyRequest is the request body for batch modifying messages
type BatchModifyRequest struct {
	IDs            []string `json:"ids"`
	AddLabelIDs    []string `json:"addLabelIds,omitempty"`
	RemoveLabelIDs []string `json:"removeLabelIds,omitempty"`
}
