//go:build js && wasm

package github

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/oauth"
)

// MockOAuthManager is a mock for testing
type MockOAuthManager struct {
	token     string
	connected bool
}

func (m *MockOAuthManager) GetToken(provider string) (string, error) {
	if !m.connected {
		return "", fmt.Errorf("not connected")
	}
	return m.token, nil
}

func (m *MockOAuthManager) IsConnected(provider string) bool {
	return m.connected
}

func TestClientIsConnected(t *testing.T) {
	mock := &MockOAuthManager{
		token:     "test-token",
		connected: true,
	}

	// We can't directly use the mock since OAuthManager is a concrete type
	// In real tests, we'd need to use interface or test via integration
	// For now, we verify the client struct compiles correctly

	// Verify mock was created (avoid unused variable)
	if !mock.connected {
		t.Error("Expected mock to be connected")
	}

	// This is a compile-time check
	var client *Client
	if client != nil {
		_ = client.IsConnected()
	}
}

func TestRateLimitTracking(t *testing.T) {
	client := &Client{
		baseURL:   "https://api.github.com",
		rateLimit: &RateLimit{},
	}

	headers := map[string]string{
		"x-ratelimit-limit":     "5000",
		"x-ratelimit-remaining": "4999",
		"x-ratelimit-reset":     "1704067200",
	}

	client.updateRateLimit(headers)

	if client.rateLimit.Limit != 5000 {
		t.Errorf("Expected limit 5000, got %d", client.rateLimit.Limit)
	}
	if client.rateLimit.Remaining != 4999 {
		t.Errorf("Expected remaining 4999, got %d", client.rateLimit.Remaining)
	}
	if client.rateLimit.Reset.IsZero() {
		t.Error("Expected reset time to be set")
	}
}

func TestRateLimitCheck(t *testing.T) {
	// Test when rate limited
	client := &Client{
		rateLimit: &RateLimit{
			Remaining: 0,
			Reset:     time.Now().Add(time.Hour),
		},
	}

	err := client.checkRateLimit()
	if err == nil {
		t.Error("Expected error when rate limited")
	}

	// Test when not rate limited
	client2 := &Client{
		rateLimit: &RateLimit{
			Remaining: 100,
			Reset:     time.Now().Add(time.Hour),
		},
	}

	err2 := client2.checkRateLimit()
	if err2 != nil {
		t.Errorf("Expected no error when not rate limited, got: %v", err2)
	}
}

func TestParseError(t *testing.T) {
	body := []byte(`{
		"message": "Validation Failed",
		"documentation_url": "https://docs.github.com",
		"errors": [
			{
				"resource": "Issue",
				"field": "title",
				"code": "missing_field"
			}
		]
	}`)

	err := parseError(body)
	if err == nil {
		t.Fatal("Expected error")
	}

	ghErr, ok := err.(*GitHubError)
	if !ok {
		t.Fatal("Expected GitHubError type")
	}

	if ghErr.Message != "Validation Failed" {
		t.Errorf("Expected message 'Validation Failed', got %s", ghErr.Message)
	}
}

func TestParseErrorPlainText(t *testing.T) {
	body := []byte("Some random error")

	err := parseError(body)
	if err == nil {
		t.Fatal("Expected error")
	}

	// Should return generic error message
	if !strings.Contains(err.Error(), "Some random error") {
		t.Errorf("Expected error to contain original text, got: %s", err.Error())
	}
}

// Test struct field tags
func TestClientStructs(t *testing.T) {
	// Test that all expected methods exist
	client := &Client{}

	// These should compile if the methods exist
	_ = client.GetRateLimit()

	// Test that structs have proper JSON tags by marshaling
	issue := Issue{
		Number: 42,
		Title:  "Test",
		State:  "open",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify JSON field names
	if _, ok := result["number"]; !ok {
		t.Error("Expected 'number' field in JSON")
	}
	if _, ok := result["title"]; !ok {
		t.Error("Expected 'title' field in JSON")
	}
	if _, ok := result["state"]; !ok {
		t.Error("Expected 'state' field in JSON")
	}
}

// Test that Client struct can be created with OAuthManager
func TestNewClient(t *testing.T) {
	// Can't fully test without real OAuthManager, but we can verify the function exists
	// and takes the right parameters
	var mockMgr *oauth.OAuthManager // This will be nil, but that's okay for compile check

	// Compile-time check that NewClient takes *oauth.OAuthManager
	_ = NewClient(mockMgr)
}
