//go:build js && wasm

package oauth

import (
	"strings"
	"testing"
)

// TestRegisterProvider verifies provider registration
func TestRegisterProvider(t *testing.T) {
	// Clear registry before test
	ClearProviders()

	tests := []struct {
		name      string
		config    OAuthProvider
		wantError bool
	}{
		{
			name: "valid provider",
			config: OAuthProvider{
				Name:     "test",
				AuthURL:  "https://test.com/auth",
				TokenURL: "https://test.com/token",
				ClientID: "client123",
			},
			wantError: false,
		},
		{
			name: "missing name",
			config: OAuthProvider{
				AuthURL:  "https://test.com/auth",
				TokenURL: "https://test.com/token",
			},
			wantError: true,
		},
		{
			name: "missing auth_url",
			config: OAuthProvider{
				Name:     "test",
				TokenURL: "https://test.com/token",
			},
			wantError: true,
		},
		{
			name: "missing token_url",
			config: OAuthProvider{
				Name:    "test",
				AuthURL: "https://test.com/auth",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterProvider(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("RegisterProvider() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestGetProvider verifies provider retrieval
func TestGetProvider(t *testing.T) {
	// Clear and register test provider
	ClearProviders()
	RegisterProvider(OAuthProvider{
		Name:        "test",
		DisplayName: "Test Provider",
		AuthURL:     "https://test.com/auth",
		TokenURL:    "https://test.com/token",
		ClientID:    "client123",
		Scopes:      []string{"read", "write"},
	})

	// Get existing provider
	provider, err := GetProvider("test")
	if err != nil {
		t.Errorf("GetProvider() unexpected error = %v", err)
	}
	if provider.Name != "test" {
		t.Errorf("provider.Name = %v, want test", provider.Name)
	}
	if provider.DisplayName != "Test Provider" {
		t.Errorf("provider.DisplayName = %v, want 'Test Provider'", provider.DisplayName)
	}

	// Get non-existent provider
	_, err = GetProvider("nonexistent")
	if err == nil {
		t.Error("GetProvider() expected error for nonexistent provider")
	}
}

// TestHasProvider verifies provider existence check
func TestHasProvider(t *testing.T) {
	ClearProviders()

	if HasProvider("test") {
		t.Error("HasProvider() should return false for unregistered provider")
	}

	RegisterProvider(OAuthProvider{
		Name:     "test",
		AuthURL:  "https://test.com/auth",
		TokenURL: "https://test.com/token",
	})

	if !HasProvider("test") {
		t.Error("HasProvider() should return true for registered provider")
	}
}

// TestListProviders verifies provider listing
func TestListProviders(t *testing.T) {
	ClearProviders()

	// Empty registry
	providers := ListProviders()
	if len(providers) != 0 {
		t.Errorf("ListProviders() = %v, want empty", providers)
	}

	// Register some providers
	RegisterProvider(OAuthProvider{Name: "provider1", AuthURL: "https://p1.com/auth", TokenURL: "https://p1.com/token"})
	RegisterProvider(OAuthProvider{Name: "provider2", AuthURL: "https://p2.com/auth", TokenURL: "https://p2.com/token"})
	RegisterProvider(OAuthProvider{Name: "provider3", AuthURL: "https://p3.com/auth", TokenURL: "https://p3.com/token"})

	providers = ListProviders()
	if len(providers) != 3 {
		t.Errorf("ListProviders() returned %d providers, want 3", len(providers))
	}

	// Check that all providers are in the list
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}
	if !providerMap["provider1"] || !providerMap["provider2"] || !providerMap["provider3"] {
		t.Error("ListProviders() missing expected providers")
	}
}

// TestOAuthProviderAuthURLParams verifies auth URL parameter generation
func TestOAuthProviderAuthURLParams(t *testing.T) {
	provider := OAuthProvider{
		Name:     "test",
		ClientID: "my_client_id",
		Scopes:   []string{"read", "write", "delete"},
		ExtraParams: map[string]string{
			"access_type": "offline",
		},
	}

	params := provider.AuthURLParams("https://callback.com", "state123", "challenge456")

	// Required params
	if params["client_id"] != "my_client_id" {
		t.Errorf("client_id = %v, want my_client_id", params["client_id"])
	}
	if params["redirect_uri"] != "https://callback.com" {
		t.Errorf("redirect_uri = %v, want https://callback.com", params["redirect_uri"])
	}
	if params["response_type"] != "code" {
		t.Errorf("response_type = %v, want code", params["response_type"])
	}
	if params["state"] != "state123" {
		t.Errorf("state = %v, want state123", params["state"])
	}
	if params["code_challenge"] != "challenge456" {
		t.Errorf("code_challenge = %v, want challenge456", params["code_challenge"])
	}
	if params["code_challenge_method"] != "S256" {
		t.Errorf("code_challenge_method = %v, want S256", params["code_challenge_method"])
	}

	// Scopes
	if params["scope"] != "read write delete" {
		t.Errorf("scope = %v, want 'read write delete'", params["scope"])
	}

	// Extra params
	if params["access_type"] != "offline" {
		t.Errorf("access_type = %v, want offline", params["access_type"])
	}
}

// TestOAuthProviderIsConfigured verifies configuration check
func TestOAuthProviderIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		provider OAuthProvider
		want     bool
	}{
		{
			name:     "fully configured",
			provider: OAuthProvider{Name: "test", AuthURL: "https://test.com/auth", TokenURL: "https://test.com/token", ClientID: "123"},
			want:     true,
		},
		{
			name:     "missing name",
			provider: OAuthProvider{AuthURL: "https://test.com/auth", TokenURL: "https://test.com/token", ClientID: "123"},
			want:     false,
		},
		{
			name:     "missing auth_url",
			provider: OAuthProvider{Name: "test", TokenURL: "https://test.com/token", ClientID: "123"},
			want:     false,
		},
		{
			name:     "missing token_url",
			provider: OAuthProvider{Name: "test", AuthURL: "https://test.com/auth", ClientID: "123"},
			want:     false,
		},
		{
			name:     "missing client_id (but that's ok for IsConfigured)",
			provider: OAuthProvider{Name: "test", AuthURL: "https://test.com/auth", TokenURL: "https://test.com/token"},
			want:     true, // IsConfigured checks required fields, not optional ones
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDefaultProviderConfigs verifies default provider configurations
func TestDefaultProviderConfigs(t *testing.T) {
	configs := DefaultProviderConfigs()

	// Check that we have the expected providers
	expectedProviders := []string{"twitter", "google", "github", "notion"}
	for _, name := range expectedProviders {
		config, ok := configs[name]
		if !ok {
			t.Errorf("DefaultProviderConfigs() missing provider: %s", name)
			continue
		}

		if config.Name != name {
			t.Errorf("config.Name = %v, want %v", config.Name, name)
		}
		if config.AuthURL == "" {
			t.Errorf("%s: AuthURL is empty", name)
		}
		if config.TokenURL == "" {
			t.Errorf("%s: TokenURL is empty", name)
		}
		if config.DisplayName == "" {
			t.Errorf("%s: DisplayName is empty", name)
		}
		if config.Icon == "" {
			t.Errorf("%s: Icon is empty", name)
		}
		if config.Description == "" {
			t.Errorf("%s: Description is empty", name)
		}
	}

	// Verify specific configurations
	twitter := configs["twitter"]
	if !contains(twitter.Scopes, "tweet.read") || !contains(twitter.Scopes, "tweet.write") {
		t.Error("twitter: missing required scopes")
	}
	if twitter.ExtraParams["access_type"] != "offline" {
		t.Error("twitter: missing offline access_type")
	}

	google := configs["google"]
	if !contains(google.Scopes, "https://www.googleapis.com/auth/gmail.modify") {
		t.Error("google: missing gmail scope")
	}
	if google.ExtraParams["prompt"] != "consent" {
		t.Error("google: missing consent prompt")
	}

	github := configs["github"]
	if !contains(github.Scopes, "repo") {
		t.Error("github: missing repo scope")
	}
}

// TestSetProviderClientID verifies client ID setting
func TestSetProviderClientID(t *testing.T) {
	ClearProviders()

	// Register a provider without client ID
	RegisterProvider(OAuthProvider{
		Name:     "test",
		AuthURL:  "https://test.com/auth",
		TokenURL: "https://test.com/token",
		// ClientID intentionally empty
	})

	// Set client ID
	err := SetProviderClientID("test", "my_client_123")
	if err != nil {
		t.Errorf("SetProviderClientID() error = %v", err)
	}

	// Verify it was set
	provider, _ := GetProvider("test")
	if provider.ClientID != "my_client_123" {
		t.Errorf("ClientID = %v, want my_client_123", provider.ClientID)
	}

	// Try to set for non-existent provider
	err = SetProviderClientID("nonexistent", "client")
	if err == nil {
		t.Error("SetProviderClientID() should error for nonexistent provider")
	}
}

// TestUnregisterProvider verifies provider removal
func TestUnregisterProvider(t *testing.T) {
	ClearProviders()

	RegisterProvider(OAuthProvider{Name: "test1", AuthURL: "https://test.com/auth", TokenURL: "https://test.com/token"})
	RegisterProvider(OAuthProvider{Name: "test2", AuthURL: "https://test.com/auth", TokenURL: "https://test.com/token"})

	if !HasProvider("test1") || !HasProvider("test2") {
		t.Fatal("providers should be registered")
	}

	UnregisterProvider("test1")

	if HasProvider("test1") {
		t.Error("test1 should be unregistered")
	}
	if !HasProvider("test2") {
		t.Error("test2 should still exist")
	}
}

// TestGetProviderDisplayInfo verifies display info retrieval
func TestGetProviderDisplayInfo(t *testing.T) {
	ClearProviders()

	RegisterProvider(OAuthProvider{
		Name:        "test",
		DisplayName: "Test Provider",
		Icon:        "test-icon",
		Description: "Test description",
		AuthURL:     "https://test.com/auth",
		TokenURL:    "https://test.com/token",
	})

	displayName, icon, description, err := GetProviderDisplayInfo("test")
	if err != nil {
		t.Errorf("GetProviderDisplayInfo() error = %v", err)
	}
	if displayName != "Test Provider" {
		t.Errorf("displayName = %v, want 'Test Provider'", displayName)
	}
	if icon != "test-icon" {
		t.Errorf("icon = %v, want test-icon", icon)
	}
	if description != "Test description" {
		t.Errorf("description = %v, want 'Test description'", description)
	}

	// Non-existent provider
	_, _, _, err = GetProviderDisplayInfo("nonexistent")
	if err == nil {
		t.Error("GetProviderDisplayInfo() should error for nonexistent provider")
	}
}

// TestGetAllProviderInfo verifies batch info retrieval
func TestGetAllProviderInfo(t *testing.T) {
	ClearProviders()

	RegisterProvider(OAuthProvider{
		Name:        "provider1",
		DisplayName: "Provider 1",
		Icon:        "icon1",
		Description: "Desc 1",
		AuthURL:     "https://p1.com/auth",
		TokenURL:    "https://p1.com/token",
		ClientID:    "client1",
	})

	RegisterProvider(OAuthProvider{
		Name:        "provider2",
		DisplayName: "Provider 2",
		Icon:        "icon2",
		Description: "Desc 2",
		AuthURL:     "https://p2.com/auth",
		TokenURL:    "https://p2.com/token",
		// No client ID
	})

	info := GetAllProviderInfo()
	if len(info) != 2 {
		t.Errorf("GetAllProviderInfo() returned %d items, want 2", len(info))
	}

	for _, p := range info {
		if p.Name == "provider1" {
			if !p.Configured {
				t.Error("provider1 should be configured (has client_id)")
			}
		}
		if p.Name == "provider2" {
			if p.Configured {
				t.Error("provider2 should not be configured (no client_id)")
			}
		}
	}
}

// TestGetRedirectURI verifies redirect URI
func TestGetRedirectURI(t *testing.T) {
	uri := GetRedirectURI()
	if uri != "about:blank" {
		t.Errorf("GetRedirectURI() = %v, want about:blank", uri)
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(s, item) || s == item {
			return true
		}
	}
	return false
}
