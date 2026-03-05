//go:build js && wasm

package oauth

import (
	"fmt"
	"strings"
	"syscall/js"
)

// OAuthProvider holds the configuration for an OAuth 2.0 provider
type OAuthProvider struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	AuthURL     string            `json:"auth_url"`
	TokenURL    string            `json:"token_url"`
	ClientID    string            `json:"client_id"` // Public client ID for SPA
	Scopes      []string          `json:"scopes"`
	ExtraParams map[string]string `json:"extra_params,omitempty"` // Provider-specific params
	Icon        string            `json:"icon"`                   // Icon identifier (used by UI)
	Description string            `json:"description"`            // Short description for UI
}

// AuthURLParams returns the URL parameters for the authorization request
func (p *OAuthProvider) AuthURLParams(redirectURI, state, codeChallenge string) map[string]string {
	params := map[string]string{
		"client_id":             p.ClientID,
		"redirect_uri":          redirectURI,
		"response_type":         "code",
		"state":                 state,
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
	}

	if len(p.Scopes) > 0 {
		params["scope"] = strings.Join(p.Scopes, " ")
	}

	// Add provider-specific params
	for k, v := range p.ExtraParams {
		params[k] = v
	}

	return params
}

// BuildAuthURL builds the full authorization URL with all parameters
func (p *OAuthProvider) BuildAuthURL(redirectURI, state, codeChallenge string) string {
	params := p.AuthURLParams(redirectURI, state, codeChallenge)

	var parts []string
	for k, v := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", k, js.Global().Get("encodeURIComponent").Invoke(v).String()))
	}

	return p.AuthURL + "?" + strings.Join(parts, "&")
}

// IsConfigured returns true if the provider has the minimum required configuration
func (p *OAuthProvider) IsConfigured() bool {
	return p.Name != "" && p.AuthURL != "" && p.TokenURL != "" && p.ClientID != ""
}

// ScopeString returns scopes as a space-separated string
func (p *OAuthProvider) ScopeString() string {
	return strings.Join(p.Scopes, " ")
}

// providerRegistry holds all registered OAuth providers
var providerRegistry = make(map[string]*OAuthProvider)

// RegisterProvider registers an OAuth provider configuration
func RegisterProvider(config OAuthProvider) error {
	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if config.AuthURL == "" {
		return fmt.Errorf("auth_url is required for provider %s", config.Name)
	}
	if config.TokenURL == "" {
		return fmt.Errorf("token_url is required for provider %s", config.Name)
	}

	// Set default display name if not provided
	if config.DisplayName == "" {
		config.DisplayName = strings.Title(config.Name)
	}

	providerRegistry[config.Name] = &config
	return nil
}

// GetProvider retrieves a provider configuration by name
func GetProvider(name string) (*OAuthProvider, error) {
	provider, ok := providerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return provider, nil
}

// ListProviders returns a list of all registered provider names
func ListProviders() []string {
	names := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		names = append(names, name)
	}
	return names
}

// ListConfiguredProviders returns only providers with valid ClientID
func ListConfiguredProviders() []string {
	var configured []string
	for name, provider := range providerRegistry {
		if provider.IsConfigured() && provider.ClientID != "" {
			configured = append(configured, name)
		}
	}
	return configured
}

// HasProvider checks if a provider is registered
func HasProvider(name string) bool {
	_, ok := providerRegistry[name]
	return ok
}

// UnregisterProvider removes a provider from the registry
func UnregisterProvider(name string) {
	delete(providerRegistry, name)
}

// ClearProviders removes all providers (mainly for testing)
func ClearProviders() {
	providerRegistry = make(map[string]*OAuthProvider)
}

// GetRedirectURI returns the redirect URI for OAuth flows
// For popup-based flows, this can be about:blank or a dedicated callback page
func GetRedirectURI() string {
	// Use about:blank for popup flows (code extracted via postMessage)
	// Alternatively, could use window.location.origin + "/oauth/callback"
	return "about:blank"
}

// DefaultProviderConfigs returns the default configurations for supported providers
// Client IDs should be loaded from environment or config, not hardcoded
func DefaultProviderConfigs() map[string]OAuthProvider {
	return map[string]OAuthProvider{
		"twitter": {
			Name:        "twitter",
			DisplayName: "Twitter / X",
			AuthURL:     "https://twitter.com/i/oauth2/authorize",
			TokenURL:    "https://api.twitter.com/2/oauth2/token",
			Scopes:      []string{"tweet.read", "tweet.write", "users.read", "offline.access"},
			ExtraParams: map[string]string{
				"access_type": "offline",
			},
			Icon:        "twitter",
			Description: "Post tweets and read timeline",
		},
		"google": {
			Name:        "google",
			DisplayName: "Google",
			AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:    "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"https://www.googleapis.com/auth/gmail.modify",
				"https://www.googleapis.com/auth/calendar.events",
			},
			ExtraParams: map[string]string{
				"access_type": "offline",
				"prompt":      "consent",
			},
			Icon:        "google",
			Description: "Gmail and Calendar access",
		},
		"github": {
			Name:        "github",
			DisplayName: "GitHub",
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			Scopes:      []string{"repo", "read:user", "user:email"},
			ExtraParams: map[string]string{},
			Icon:        "github",
			Description: "Repository and issue access",
		},
		"notion": {
			Name:        "notion",
			DisplayName: "Notion",
			AuthURL:     "https://api.notion.com/v1/oauth/authorize",
			TokenURL:    "https://api.notion.com/v1/oauth/token",
			Scopes:      []string{}, // Notion determines scope by integration
			ExtraParams: map[string]string{},
			Icon:        "notion",
			Description: "Database and page access",
		},
	}
}

// RegisterDefaultProviders registers all default providers
// Client IDs must be set separately (from env/config)
func RegisterDefaultProviders() {
	for _, config := range DefaultProviderConfigs() {
		// Skip registration if already registered
		if !HasProvider(config.Name) {
			RegisterProvider(config)
		}
	}
}

// SetProviderClientID sets the client ID for a provider
// This should be called with values from environment or config
func SetProviderClientID(providerName, clientID string) error {
	provider, ok := providerRegistry[providerName]
	if !ok {
		return fmt.Errorf("unknown provider: %s", providerName)
	}
	provider.ClientID = clientID
	return nil
}

// GetProviderDisplayInfo returns minimal info for UI display
func GetProviderDisplayInfo(name string) (displayName, icon, description string, err error) {
	provider, err := GetProvider(name)
	if err != nil {
		return "", "", "", err
	}
	return provider.DisplayName, provider.Icon, provider.Description, nil
}

// ProviderInfo holds display info for UI
type ProviderInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Configured  bool   `json:"configured"`
}

// GetAllProviderInfo returns info for all registered providers
func GetAllProviderInfo() []ProviderInfo {
	var info []ProviderInfo
	for name, provider := range providerRegistry {
		info = append(info, ProviderInfo{
			Name:        name,
			DisplayName: provider.DisplayName,
			Icon:        provider.Icon,
			Description: provider.Description,
			Configured:  provider.IsConfigured(),
		})
	}
	return info
}

// Init performs package initialization
// Called from main.go to set up default providers
func Init() {
	RegisterDefaultProviders()
}
