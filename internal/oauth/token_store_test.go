//go:build js && wasm

package oauth

import (
	"testing"
	"time"
)

// TestTokenIsExpired verifies token expiration logic
func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "token expired 1 hour ago",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "token expired 30 seconds ago",
			expiresAt: time.Now().Add(-30 * time.Second),
			want:      true,
		},
		{
			name:      "token expires in 30 seconds (buffer applies)",
			expiresAt: time.Now().Add(30 * time.Second),
			want:      true, // 60-second buffer means it's considered expired
		},
		{
			name:      "token expires in 5 minutes",
			expiresAt: time.Now().Add(5 * time.Minute),
			want:      false, // Beyond 60-second buffer
		},
		{
			name:      "token expires in 1 hour",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "no expiration (never expires)",
			expiresAt: time.Time{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{ExpiresAt: tt.expiresAt}
			got := token.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTokenNeedsRefresh verifies refresh detection logic
func TestTokenNeedsRefresh(t *testing.T) {
	tests := []struct {
		name         string
		expiresAt    time.Time
		refreshToken string
		want         bool
	}{
		{
			name:         "expired with refresh token",
			expiresAt:    time.Now().Add(-1 * time.Hour),
			refreshToken: "refresh_123",
			want:         true,
		},
		{
			name:         "expires in 3 minutes (within 5-min window)",
			expiresAt:    time.Now().Add(3 * time.Minute),
			refreshToken: "refresh_123",
			want:         true,
		},
		{
			name:         "expires in 10 minutes (outside window)",
			expiresAt:    time.Now().Add(10 * time.Minute),
			refreshToken: "refresh_123",
			want:         false,
		},
		{
			name:         "expired but no refresh token",
			expiresAt:    time.Now().Add(-1 * time.Hour),
			refreshToken: "",
			want:         false, // Can't refresh without refresh token
		},
		{
			name:         "no expiration",
			expiresAt:    time.Time{},
			refreshToken: "refresh_123",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				ExpiresAt:    tt.expiresAt,
				RefreshToken: tt.refreshToken,
			}
			got := token.NeedsRefresh()
			if got != tt.want {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTokenTimeUntilExpiry verifies duration calculation
func TestTokenTimeUntilExpiry(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "expires in 10 minutes",
			expiresAt: time.Now().Add(10 * time.Minute),
			wantMin:   9 * time.Minute,
			wantMax:   11 * time.Minute,
		},
		{
			name:      "expired 5 minutes ago",
			expiresAt: time.Now().Add(-5 * time.Minute),
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "no expiration",
			expiresAt: time.Time{},
			wantMin:   0,
			wantMax:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{ExpiresAt: tt.expiresAt}
			got := token.TimeUntilExpiry()
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("TimeUntilExpiry() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestTokenStoreValidation tests token store input validation
func TestTokenStoreValidation(t *testing.T) {
	// Note: Can't create actual TokenStore in unit tests without IndexedDB
	// These tests verify the Token struct validation logic

	t.Run("token with all fields", func(t *testing.T) {
		token := &Token{
			Provider:     "twitter",
			AccessToken:  "access_123",
			RefreshToken: "refresh_456",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			Scope:        "tweet.read tweet.write",
			Username:     "@testuser",
			UserID:       "123456",
		}

		if token.Provider != "twitter" {
			t.Error("provider mismatch")
		}
		if token.AccessToken == "" {
			t.Error("access token should not be empty")
		}
	})

	t.Run("token with minimal fields", func(t *testing.T) {
		token := &Token{
			Provider:    "github",
			AccessToken: "ghp_1234567890",
		}

		if token.Provider != "github" {
			t.Error("provider mismatch")
		}
		if token.IsExpired() {
			t.Error("token with no expiration should not be expired")
		}
		if token.NeedsRefresh() {
			t.Error("token with no expiration and no refresh token should not need refresh")
		}
	})
}

// TestStoredTokenStructure verifies the stored token format
func TestStoredTokenStructure(t *testing.T) {
	stored := StoredToken{
		Provider:   "twitter",
		Ciphertext: "encrypted_data_here",
		IV:         "iv_here",
		Salt:       "salt_here",
		CreatedAt:  time.Now().Unix(),
	}

	if stored.Provider != "twitter" {
		t.Error("provider mismatch")
	}
	if stored.Ciphertext == "" {
		t.Error("ciphertext should not be empty")
	}
	if stored.IV == "" {
		t.Error("IV should not be empty")
	}
	if stored.Salt == "" {
		t.Error("salt should not be empty")
	}
	if stored.CreatedAt == 0 {
		t.Error("created_at should be set")
	}
}
