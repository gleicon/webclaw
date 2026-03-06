//go:build js && wasm

package google

import (
	"encoding/base64"
	"testing"
)

func TestEncodeBase64URL(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "simple text",
			input:    []byte("Hello, World!"),
			expected: "SGVsbG8sIFdvcmxkIQ",
		},
		{
			name:     "empty string",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "RFC 2822 message",
			input:    []byte("To: test@example.com\r\nSubject: Test\r\n\r\nBody"),
			expected: "VG86IHRlc3RAZXhhbXBsZS5jb20NClN1YmplY3Q6IFRlc3QNClxuQm9keQ",
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			expected: "AAECA_8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeBase64URL(tt.input)
			if result != tt.expected {
				t.Errorf("EncodeBase64URL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDecodeBase64URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
		wantErr  bool
	}{
		{
			name:     "simple text",
			input:    "SGVsbG8sIFdvcmxkIQ",
			expected: []byte("Hello, World!"),
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: []byte(""),
			wantErr:  false,
		},
		{
			name:     "standard base64 (should fail)",
			input:    "SGVsbG8sIFdvcmxkIQ==",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64URL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeBase64URL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(result) != string(tt.expected) {
				t.Errorf("DecodeBase64URL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRoundTripBase64(t *testing.T) {
	// Test that encode/decode round-trips correctly
	tests := [][]byte{
		[]byte("Test message"),
		[]byte("Special chars: <>&\"'"),
		[]byte("Unicode: 日本語"),
		[]byte{},
		[]byte{0x00, 0x01, 0xFE, 0xFF},
	}

	for _, input := range tests {
		encoded := EncodeBase64URL(input)
		decoded, err := DecodeBase64URL(encoded)
		if err != nil {
			t.Errorf("Round-trip decode failed: %v", err)
			continue
		}
		if string(decoded) != string(input) {
			t.Errorf("Round-trip failed: got %v, want %v", decoded, input)
		}
	}
}

func TestClientBuildURLs(t *testing.T) {
	// Create a mock client (won't have OAuth manager)
	c := &Client{}
	c.baseURLs.Gmail = "https://gmail.googleapis.com/gmail/v1"
	c.baseURLs.Calendar = "https://www.googleapis.com/calendar/v3"

	// Test Gmail URL building
	gmailURL := c.BuildGmailURL("/users/me/messages")
	expected := "https://gmail.googleapis.com/gmail/v1/users/me/messages"
	if gmailURL != expected {
		t.Errorf("BuildGmailURL() = %v, want %v", gmailURL, expected)
	}

	// Test Calendar URL building
	calURL := c.BuildCalendarURL("/calendars/primary/events")
	expected = "https://www.googleapis.com/calendar/v3/calendars/primary/events"
	if calURL != expected {
		t.Errorf("BuildCalendarURL() = %v, want %v", calURL, expected)
	}
}

func TestBase64URLEncodingStandard(t *testing.T) {
	// Verify our encoding matches the standard URL encoding without padding
	input := []byte("Hello, World!")

	ours := EncodeBase64URL(input)
	standard := base64.RawURLEncoding.EncodeToString(input)

	if ours != standard {
		t.Errorf("Encoding mismatch: ours=%v, standard=%v", ours, standard)
	}
}
