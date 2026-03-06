//go:build js && wasm

package gmail

import (
	"context"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/tools"
)

// NewSendTool creates the gmail_send tool
func NewSendTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "gmail_send",
		Description: "Send an email via Gmail",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Recipient email address",
				},
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Email subject",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Email body (plain text)",
				},
			},
			"required": []string{"to", "subject", "body"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			to, _ := params["to"].(string)
			subject, _ := params["subject"].(string)
			body, _ := params["body"].(string)

			if to == "" {
				return &tools.ToolResult{
					Content:        "Error: 'to' parameter is required",
					DisplayContent: "Failed to send: recipient email is required",
					IsError:        true,
					ToolName:       "gmail_send",
					Status:         "error",
				}, nil
			}

			if subject == "" {
				return &tools.ToolResult{
					Content:        "Error: 'subject' parameter is required",
					DisplayContent: "Failed to send: subject is required",
					IsError:        true,
					ToolName:       "gmail_send",
					Status:         "error",
				}, nil
			}

			msg, err := client.SendMessage(to, subject, body)
			if err != nil {
				// Check for OAuth not connected error
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to send emails",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "gmail_send",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to send email: %s", errStr),
					DisplayContent: fmt.Sprintf("Send failed: %s", errStr),
					IsError:        true,
					ToolName:       "gmail_send",
					Status:         "error",
				}, nil
			}

			return &tools.ToolResult{
				Content:        fmt.Sprintf("Email sent successfully. Message ID: %s", msg.ID),
				DisplayContent: fmt.Sprintf("✓ Email sent to %s", to),
				IsError:        false,
				ToolName:       "gmail_send",
				Status:         "done",
			}, nil
		},
	}
}

// NewListTool creates the gmail_list tool
func NewListTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "gmail_list",
		Description: "List recent emails from Gmail inbox",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of emails to list (max 50, default 10)",
				},
				"label": map[string]interface{}{
					"type":        "string",
					"description": "Filter by label (INBOX, UNREAD, SENT, etc.)",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			countFloat, _ := params["count"].(float64)
			count := int(countFloat)
			if count <= 0 || count > 50 {
				count = 10
			}

			label, _ := params["label"].(string)

			// Build label filter
			var labelIDs []string
			query := ""
			if label != "" {
				label = strings.ToUpper(label)
				query = "label:" + label
			}

			resp, err := client.ListMessages(count, query, labelIDs, "")
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to read emails",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "gmail_list",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to list emails: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to list emails: %s", errStr),
					IsError:        true,
					ToolName:       "gmail_list",
					Status:         "error",
				}, nil
			}

			if resp.Messages == nil || len(resp.Messages) == 0 {
				return &tools.ToolResult{
					Content:        "No emails found in your inbox.",
					DisplayContent: "No emails found.",
					IsError:        false,
					ToolName:       "gmail_list",
					Status:         "done",
				}, nil
			}

			content := formatMessageList(resp.Messages, label)
			display := fmt.Sprintf("📧 Found %d emails", len(resp.Messages))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "gmail_list",
				Status:         "done",
			}, nil
		},
	}
}

// NewReadTool creates the gmail_read tool
func NewReadTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "gmail_read",
		Description: "Read full content of a specific email",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message_id": map[string]interface{}{
					"type":        "string",
					"description": "Gmail message ID",
				},
			},
			"required": []string{"message_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			messageID, _ := params["message_id"].(string)

			if messageID == "" {
				return &tools.ToolResult{
					Content:        "Error: 'message_id' parameter is required",
					DisplayContent: "Failed: message ID is required",
					IsError:        true,
					ToolName:       "gmail_read",
					Status:         "error",
				}, nil
			}

			msg, err := client.GetMessage(messageID, "full")
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to read emails",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "gmail_read",
						Status:         "error",
					}, nil
				}

				if strings.Contains(errStr, "404") {
					return &tools.ToolResult{
						Content:        "Email not found. It may have been deleted.",
						DisplayContent: "Email not found",
						IsError:        true,
						ToolName:       "gmail_read",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to read email: %s", errStr),
					DisplayContent: fmt.Sprintf("Failed to read email: %s", errStr),
					IsError:        true,
					ToolName:       "gmail_read",
					Status:         "error",
				}, nil
			}

			content := formatMessage(msg)
			display := fmt.Sprintf("📧 %s", truncateString(GetHeader(msg, "Subject"), 50))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "gmail_read",
				Status:         "done",
			}, nil
		},
	}
}

// NewSearchTool creates the gmail_search tool
func NewSearchTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "gmail_search",
		Description: "Search emails using Gmail query syntax. Examples: 'from:boss@company.com', 'subject:urgent', 'has:attachment', 'is:unread', 'after:2024/01/01'",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Gmail search query (e.g., 'from:boss@company.com subject:urgent', 'has:attachment', 'is:unread')",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Max results (default 10, max 50)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			query, _ := params["query"].(string)
			countFloat, _ := params["count"].(float64)
			count := int(countFloat)

			if query == "" {
				return &tools.ToolResult{
					Content:        "Error: 'query' parameter is required",
					DisplayContent: "Failed: search query is required",
					IsError:        true,
					ToolName:       "gmail_search",
					Status:         "error",
				}, nil
			}

			if count <= 0 || count > 50 {
				count = 10
			}

			resp, err := client.SearchMessages(query, count)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "not connected") || strings.Contains(errStr, "google not connected") {
					return &tools.ToolResult{
						Content:        "Please connect Google in Settings to search emails",
						DisplayContent: "Google not connected. Please connect Google in Settings first.",
						IsError:        true,
						ToolName:       "gmail_search",
						Status:         "error",
					}, nil
				}

				return &tools.ToolResult{
					Content:        fmt.Sprintf("Search failed: %s", errStr),
					DisplayContent: fmt.Sprintf("Search failed: %s", errStr),
					IsError:        true,
					ToolName:       "gmail_search",
					Status:         "error",
				}, nil
			}

			if resp.Messages == nil || len(resp.Messages) == 0 {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("No emails found for query: %s", query),
					DisplayContent: fmt.Sprintf("No results for: %s", query),
					IsError:        false,
					ToolName:       "gmail_search",
					Status:         "done",
				}, nil
			}

			content := formatSearchResults(resp.Messages, query)
			display := fmt.Sprintf("🔍 Found %d emails for: %s", len(resp.Messages), query)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "gmail_search",
				Status:         "done",
			}, nil
		},
	}
}

// formatMessageList formats a list of messages for display
func formatMessageList(messages []*Message, label string) string {
	var b strings.Builder

	if label != "" {
		b.WriteString(fmt.Sprintf("Recent %d emails in %s:\n\n", len(messages), label))
	} else {
		b.WriteString(fmt.Sprintf("Recent %d emails:\n\n", len(messages)))
	}

	for i, msg := range messages {
		// Get headers from message (if available)
		subject := msg.Snippet
		if subject == "" {
			subject = "(No subject)"
		}

		// Format each message on one line with ID
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncateString(subject, 60)))
		b.WriteString(fmt.Sprintf("   ID: %s\n", msg.ID))
		if i < len(messages)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatMessage formats a full message with body
func formatMessage(msg *Message) string {
	var b strings.Builder

	// Headers
	subject := GetHeader(msg, "Subject")
	from := GetHeader(msg, "From")
	to := GetHeader(msg, "To")
	date := GetHeader(msg, "Date")

	if subject == "" {
		subject = "(No subject)"
	}

	b.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	b.WriteString(fmt.Sprintf("From: %s\n", from))
	b.WriteString(fmt.Sprintf("To: %s\n", to))
	b.WriteString(fmt.Sprintf("Date: %s\n", date))
	b.WriteString(fmt.Sprintf("ID: %s\n", msg.ID))
	b.WriteString("\n")

	// Body
	body, err := ExtractBody(msg)
	if err != nil {
		body = fmt.Sprintf("(Error extracting body: %s)", err.Error())
	}

	if body == "" {
		// Fall back to snippet
		body = msg.Snippet
		if body == "" {
			body = "(No message body)"
		}
	}

	// Strip HTML tags if present
	if strings.Contains(body, "<html") || strings.Contains(body, "<body") {
		body = StripHTMLTags(body)
	}

	// Truncate very long bodies
	if len(body) > 5000 {
		body = body[:5000] + "\n\n... [message truncated, use ID to read full content]"
	}

	b.WriteString(body)

	return b.String()
}

// formatSearchResults formats search results
func formatSearchResults(messages []*Message, query string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	for i, msg := range messages {
		subject := msg.Snippet
		if subject == "" {
			subject = "(No subject)"
		}

		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncateString(subject, 60)))
		b.WriteString(fmt.Sprintf("   ID: %s\n", msg.ID))
		if i < len(messages)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// truncateString truncates a string to maxLen with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
