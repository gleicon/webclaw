//go:build js && wasm

package gmail

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/integrations/google"
)

// Client provides Gmail API operations
type Client struct {
	base *google.Client
}

// NewClient creates a new Gmail client
func NewClient(baseClient *google.Client) *Client {
	return &Client{
		base: baseClient,
	}
}

// ListMessages lists messages in the user's mailbox
// Parameters:
//   - maxResults: Maximum number of messages to return (max 500, default 100)
//   - query: Gmail search query (optional, uses Gmail query syntax)
//   - labelIDs: Filter by label IDs (optional)
//   - pageToken: Token for pagination (optional)
func (c *Client) ListMessages(maxResults int, query string, labelIDs []string, pageToken string) (*ListMessagesResponse, error) {
	if maxResults <= 0 || maxResults > 500 {
		maxResults = 100
	}

	// Build URL with query parameters
	url := c.base.BuildGmailURL("/users/me/messages")
	params := []string{fmt.Sprintf("maxResults=%d", maxResults)}

	if query != "" {
		// URL-encode the query
		query = strings.ReplaceAll(query, " ", "%20")
		query = strings.ReplaceAll(query, "+", "%2B")
		params = append(params, "q="+query)
	}

	if len(labelIDs) > 0 {
		params = append(params, "labelIds="+strings.Join(labelIDs, ","))
	}

	if pageToken != "" {
		params = append(params, "pageToken="+pageToken)
	}

	url = url + "?" + strings.Join(params, "&")

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result ListMessagesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse message list: %w", err)
	}

	return &result, nil
}

// GetMessage retrieves a specific message by ID
// format can be: "minimal", "full", "raw", or "metadata"
func (c *Client) GetMessage(id string, format string) (*Message, error) {
	if format == "" {
		format = "full"
	}

	url := c.base.BuildGmailURL(fmt.Sprintf("/users/me/messages/%s?format=%s", id, format))

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(resp.Body, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, nil
}

// SendMessage sends a new email message
// The message is sent in base64url-encoded RFC 2822 format
func (c *Client) SendMessage(to, subject, body string) (*Message, error) {
	// Compose the email in RFC 2822 format
	rfcMessage := ComposeEmail(to, subject, body)

	// Encode to base64url
	raw := google.EncodeBase64URL([]byte(rfcMessage))

	// Build request body
	req := SendMessageRequest{Raw: raw}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal send request: %w", err)
	}

	url := c.base.BuildGmailURL("/users/me/messages/send")

	resp, err := c.base.DoRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(resp.Body, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse sent message: %w", err)
	}

	return &msg, nil
}

// TrashMessage moves a message to trash
func (c *Client) TrashMessage(id string) error {
	url := c.base.BuildGmailURL(fmt.Sprintf("/users/me/messages/%s/trash", id))
	_, err := c.base.DoRequest("POST", url, nil)
	return err
}

// DeleteMessage permanently deletes a message
func (c *Client) DeleteMessage(id string) error {
	url := c.base.BuildGmailURL(fmt.Sprintf("/users/me/messages/%s", id))
	_, err := c.base.DoRequest("DELETE", url, nil)
	return err
}

// SearchMessages searches messages using Gmail query syntax
// This is a convenience wrapper around ListMessages
func (c *Client) SearchMessages(query string, maxResults int) (*ListMessagesResponse, error) {
	return c.ListMessages(maxResults, query, nil, "")
}

// ListLabels lists all labels in the user's mailbox
func (c *Client) ListLabels() (*ListLabelsResponse, error) {
	url := c.base.BuildGmailURL("/users/me/labels")

	resp, err := c.base.DoRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result ListLabelsResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse labels: %w", err)
	}

	return &result, nil
}

// ModifyMessage modifies labels on a message
func (c *Client) ModifyMessage(id string, addLabelIDs, removeLabelIDs []string) (*Message, error) {
	req := ModifyMessageRequest{
		AddLabelIDs:    addLabelIDs,
		RemoveLabelIDs: removeLabelIDs,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modify request: %w", err)
	}

	url := c.base.BuildGmailURL(fmt.Sprintf("/users/me/messages/%s/modify", id))

	resp, err := c.base.DoRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(resp.Body, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse modified message: %w", err)
	}

	return &msg, nil
}

// ComposeEmail creates an RFC 2822 formatted email message
func ComposeEmail(to, subject, body string) string {
	var b strings.Builder

	// Headers
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("\r\n") // Empty line separates headers from body

	// Body
	b.WriteString(body)

	return b.String()
}

// ExtractBody extracts the text body from a message
// Handles multipart messages and decodes base64url content
func ExtractBody(msg *Message) (string, error) {
	if msg.Payload == nil {
		return "", nil
	}

	return extractPartBody(msg.Payload)
}

// extractPartBody recursively extracts text from a message part
func extractPartBody(part *MessagePart) (string, error) {
	// Check if this part has body data
	if part.Body != nil && part.Body.Data != "" {
		// Decode base64url
		data, err := google.DecodeBase64URL(part.Body.Data)
		if err != nil {
			return "", fmt.Errorf("failed to decode body: %w", err)
		}

		// If it's text/plain, return it
		if part.MimeType == "text/plain" {
			return string(data), nil
		}

		// If it's text/html, return it (caller can strip HTML if needed)
		if part.MimeType == "text/html" {
			return string(data), nil
		}
	}

	// For multipart messages, look for text/plain or text/html
	if part.Parts != nil && len(part.Parts) > 0 {
		// First pass: look for text/plain
		for _, subPart := range part.Parts {
			if subPart.MimeType == "text/plain" {
				return extractPartBody(subPart)
			}
		}

		// Second pass: accept text/html
		for _, subPart := range part.Parts {
			if subPart.MimeType == "text/html" {
				return extractPartBody(subPart)
			}
		}

		// Third pass: accept any text/*
		for _, subPart := range part.Parts {
			if strings.HasPrefix(subPart.MimeType, "text/") {
				return extractPartBody(subPart)
			}
		}

		// Last resort: take the first part
		return extractPartBody(part.Parts[0])
	}

	return "", nil
}

// GetHeader extracts a specific header value from a message
func GetHeader(msg *Message, name string) string {
	if msg.Payload == nil || msg.Payload.Headers == nil {
		return ""
	}

	name = strings.ToLower(name)
	for _, h := range msg.Payload.Headers {
		if strings.ToLower(h.Name) == name {
			return h.Value
		}
	}
	return ""
}

// StripHTMLTags removes HTML tags from text
// Simple implementation - for more complex needs, consider a proper HTML parser
func StripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false

	for i := 0; i < len(html); i++ {
		if html[i] == '<' {
			inTag = true
			continue
		}
		if html[i] == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteByte(html[i])
		}
	}

	return strings.TrimSpace(result.String())
}
