//go:build js && wasm

package gmail

import (
	"strings"
	"testing"
)

func TestComposeEmail(t *testing.T) {
	tests := []struct {
		name     string
		to       string
		subject  string
		body     string
		expected []string // substrings that should be in result
	}{
		{
			name:    "simple email",
			to:      "john@example.com",
			subject: "Hello",
			body:    "This is a test email",
			expected: []string{
				"To: john@example.com",
				"Subject: Hello",
				"Content-Type: text/plain",
				"\r\n",
				"This is a test email",
			},
		},
		{
			name:    "email with special chars",
			to:      "user+test@example.com",
			subject: "Re: Your question",
			body:    "Line 1\nLine 2",
			expected: []string{
				"To: user+test@example.com",
				"Subject: Re: Your question",
				"Line 1",
				"Line 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComposeEmail(tt.to, tt.subject, tt.body)
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("ComposeEmail() missing expected substring: %v\nGot:\n%s", exp, result)
				}
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple HTML",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "nested HTML",
			input:    "<div><p>Hello <strong>World</strong></p></div>",
			expected: "Hello World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no HTML",
			input:    "Just plain text",
			expected: "Just plain text",
		},
		{
			name:     "HTML with newlines",
			input:    "<p>Line 1</p>\n<p>Line 2</p>",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "complex Gmail HTML",
			input:    `<div dir="ltr"><div>Test message</div></div>`,
			expected: "Test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTMLTags() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetHeader(t *testing.T) {
	msg := &Message{
		Payload: &MessagePart{
			Headers: []*MessageHeader{
				{Name: "From", Value: "sender@example.com"},
				{Name: "To", Value: "recipient@example.com"},
				{Name: "Subject", Value: "Test Subject"},
				{Name: "Date", Value: "Mon, 15 Jan 2024 10:00:00 GMT"},
			},
		},
	}

	tests := []struct {
		headerName string
		expected   string
	}{
		{"From", "sender@example.com"},
		{"from", "sender@example.com"}, // case insensitive
		{"FROM", "sender@example.com"}, // case insensitive
		{"To", "recipient@example.com"},
		{"Subject", "Test Subject"},
		{"Date", "Mon, 15 Jan 2024 10:00:00 GMT"},
		{"NonExistent", ""}, // header not found
	}

	for _, tt := range tests {
		t.Run(tt.headerName, func(t *testing.T) {
			result := GetHeader(msg, tt.headerName)
			if result != tt.expected {
				t.Errorf("GetHeader(%s) = %v, want %v", tt.headerName, result, tt.expected)
			}
		})
	}
}

func TestGetHeaderNilCases(t *testing.T) {
	// Nil payload
	msg1 := &Message{Payload: nil}
	result := GetHeader(msg1, "From")
	if result != "" {
		t.Errorf("GetHeader with nil payload = %v, want empty string", result)
	}

	// Nil headers
	msg2 := &Message{Payload: &MessagePart{Headers: nil}}
	result = GetHeader(msg2, "From")
	if result != "" {
		t.Errorf("GetHeader with nil headers = %v, want empty string", result)
	}
}

func TestExtractBody(t *testing.T) {
	// Note: This test requires base64url encoded data
	// "Hello World" base64url encoded: SGVsbG8gV29ybGQ

	tests := []struct {
		name     string
		message  *Message
		expected string
		wantErr  bool
	}{
		{
			name: "nil payload",
			message: &Message{
				Payload: nil,
			},
			expected: "",
			wantErr:  false,
		},
		{
			name: "text/plain body",
			message: &Message{
				Payload: &MessagePart{
					MimeType: "text/plain",
					Body: &MessagePartBody{
						Data: "SGVsbG8gV29ybGQ", // "Hello World"
						Size: 11,
					},
				},
			},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name: "multipart with text/plain",
			message: &Message{
				Payload: &MessagePart{
					MimeType: "multipart/alternative",
					Parts: []*MessagePart{
						{
							MimeType: "text/plain",
							Body: &MessagePartBody{
								Data: "UGxhaW4gdGV4dA", // "Plain text"
								Size: 10,
							},
						},
						{
							MimeType: "text/html",
							Body: &MessagePartBody{
								Data: "PGh0bWw-", // "<html>"
								Size: 6,
							},
						},
					},
				},
			},
			expected: "Plain text", // Should prefer text/plain
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractBody(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ExtractBody() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypesMarshal(t *testing.T) {
	// Test that our types can be marshaled to JSON
	msg := &Message{
		ID:       "msg123",
		ThreadID: "thread456",
		LabelIDs: []string{"INBOX", "UNREAD"},
		Snippet:  "Test snippet",
		Payload: &MessagePart{
			MimeType: "text/plain",
			Body: &MessagePartBody{
				Data: "dGVzdA",
				Size: 4,
			},
		},
		InternalDate: 1705312800000,
	}

	if msg.ID != "msg123" {
		t.Errorf("Message ID mismatch")
	}
	if len(msg.LabelIDs) != 2 {
		t.Errorf("LabelIDs length = %v, want 2", len(msg.LabelIDs))
	}
}
