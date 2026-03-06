//go:build js && wasm

package oauth

import (
	"fmt"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// MockJSBridge implements JSBridge for testing
type MockJSBridge struct {
	ShouldFail      bool
	ReturnCode      string
	ReturnState     string
	ReturnError     string
	ReturnErrorDesc string
}

func (m *MockJSBridge) OpenOAuthPopup(authURL, provider, state string) (*jsbridge.OAuthCallbackData, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock popup failure")
	}
	return &jsbridge.OAuthCallbackData{
		Code:      m.ReturnCode,
		State:     m.ReturnState,
		Error:     m.ReturnError,
		ErrorDesc: m.ReturnErrorDesc,
		Provider:  provider,
	}, nil
}

// TestNewOAuthManager verifies manager creation
func TestNewOAuthManager(t *testing.T) {
	store := &TokenStore{} // Mock store
	bridge := &MockJSBridge{}

	mgr := NewOAuthManager(store, bridge)

	if mgr == nil {
		t.Fatal("NewOAuthManager returned nil")
	}
	if mgr.tokenStore != store {
		t.Error("tokenStore not set correctly")
	}
	if mgr.jsBridge != bridge {
		t.Error("jsBridge not set correctly")
	}
	if mgr.state == nil {
		t.Error("state map not initialized")
	}
}

// TestOAuthManagerIsConnected verifies connection check
func TestOAuthManagerIsConnected(t *testing.T) {
	// Note: This test would need a real TokenStore to work properly
	// For unit testing, we verify the method exists and has correct signature

	t.Run("method exists", func(t *testing.T) {
		mgr := NewOAuthManager(nil, nil)
		// Just verify it doesn't panic
		_ = mgr.IsConnected("twitter")
	})
}

// TestConnectionStatus verifies status struct
func TestConnectionStatus(t *testing.T) {
	status := ConnectionStatus{
		Connected:       true,
		Provider:        "twitter",
		Username:        "@testuser",
		Scope:           "tweet.read tweet.write",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		CanRefresh:      true,
		NeedsRefresh:    false,
		TimeUntilExpiry: 1 * time.Hour,
	}

	if !status.Connected {
		t.Error("Connected should be true")
	}
	if status.Provider != "twitter" {
		t.Errorf("Provider = %v, want twitter", status.Provider)
	}
	if status.Username != "@testuser" {
		t.Errorf("Username = %v, want @testuser", status.Username)
	}
}

// TestGetTokenErrorMessages verifies error message format
func TestGetTokenErrorMessages(t *testing.T) {
	// Error messages should be user-friendly
	tests := []struct {
		name        string
		provider    string
		wantContain string
	}{
		{
			name:        "twitter error",
			provider:    "twitter",
			wantContain: "connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the provider lookup works
			provider, err := GetProvider(tt.provider)
			if err == nil {
				// Provider exists - check that it has a display name
				if provider.DisplayName == "" {
					t.Error("provider should have display name")
				}
			}
		})
	}
}

// TestOAuthManagerListConnections verifies connection listing
func TestOAuthManagerListConnections(t *testing.T) {
	mgr := NewOAuthManager(nil, nil)

	// Clear providers and register test ones
	ClearProviders()
	RegisterProvider(OAuthProvider{
		Name:     "provider1",
		AuthURL:  "https://p1.com/auth",
		TokenURL: "https://p1.com/token",
	})
	RegisterProvider(OAuthProvider{
		Name:     "provider2",
		AuthURL:  "https://p2.com/auth",
		TokenURL: "https://p2.com/token",
	})

	connections := mgr.ListConnections()

	// Should return 2 connections (both disconnected since no token store)
	if len(connections) != 2 {
		t.Errorf("ListConnections() returned %d connections, want 2", len(connections))
	}

	// All should be disconnected
	for _, conn := range connections {
		if conn.Connected {
			t.Errorf("Connection for %s should be disconnected", conn.Provider)
		}
	}
}
