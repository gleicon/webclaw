//go:build js && wasm

package oauth

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// OAuthManager orchestrates the complete OAuth 2.0 PKCE flow
type OAuthManager struct {
	tokenStore *TokenStore
	jsBridge   JSBridge
	state      map[string]string // state -> provider mapping for verification
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(tokenStore *TokenStore, jsBridge JSBridge) *OAuthManager {
	return &OAuthManager{
		tokenStore: tokenStore,
		jsBridge:   jsBridge,
		state:      make(map[string]string),
	}
}

// JSBridge interface for the OAuth flow
// Implemented by jsbridge.JSOAuthBridge
type JSBridge interface {
	OpenOAuthPopup(authURL, provider, state string) (*jsbridge.OAuthCallbackData, error)
}

// InitiateConnection initiates the full OAuth flow for a provider
// 1. Get provider config
// 2. Generate PKCE parameters
// 3. Generate state parameter
// 4. Build auth URL
// 5. Open popup
// 6. Exchange code for token
// 7. Store token
func (m *OAuthManager) InitiateConnection(providerName string) error {
	// Get provider configuration
	provider, err := GetProvider(providerName)
	if err != nil {
		return fmt.Errorf("unknown provider: %w", err)
	}

	if provider.ClientID == "" {
		return fmt.Errorf("%s is not configured (missing client ID)", provider.DisplayName)
	}

	// Generate PKCE parameters
	pkceParams := GeneratePKCEParams()

	// Generate state parameter (random string)
	state := GenerateCodeVerifier()[:32] // Use first 32 chars of verifier as state
	m.state[state] = providerName

	// Build authorization URL
	redirectURI := GetRedirectURI()
	authURL := provider.BuildAuthURL(redirectURI, state, pkceParams.CodeChallenge)

	// Open popup and wait for callback
	callbackData, err := m.jsBridge.OpenOAuthPopup(authURL, providerName, state)
	if err != nil {
		return fmt.Errorf("OAuth popup failed: %w", err)
	}

	// Verify state matches (CSRF protection)
	if callbackData.State != state {
		return fmt.Errorf("state mismatch - possible CSRF attack")
	}

	// Check for OAuth error response
	if callbackData.Error != "" {
		if callbackData.Error == "access_denied" {
			return fmt.Errorf("access denied: you rejected the authorization request")
		}
		return fmt.Errorf("OAuth error: %s - %s", callbackData.Error, callbackData.ErrorDesc)
	}

	// Exchange authorization code for access token
	token, err := m.exchangeCode(provider, callbackData.Code, pkceParams.CodeVerifier)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	// Store the token
	token.Provider = providerName
	if err := m.tokenStore.SaveToken(providerName, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// exchangeCode exchanges the authorization code for an access token
func (m *OAuthManager) exchangeCode(provider *OAuthProvider, code, codeVerifier string) (*Token, error) {
	// Use JS bridge to exchange code (needed for CORS)
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		return nil, fmt.Errorf("webclaw not available")
	}

	oauth := webclaw.Get("oauth")
	if oauth.IsUndefined() || oauth.IsNull() {
		return nil, fmt.Errorf("webclaw.oauth not available")
	}

	exchangeFn := oauth.Get("exchangeCode")
	if exchangeFn.IsUndefined() || exchangeFn.IsNull() {
		return nil, fmt.Errorf("exchangeCode not available")
	}

	// Build config JSON
	config := map[string]string{
		"token_url": provider.TokenURL,
		"client_id": provider.ClientID,
	}
	configJSON, _ := json.Marshal(config)

	// Call JS exchange function
	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	go func() {
		promise := exchangeFn.Invoke(
			js.ValueOf(provider.Name),
			js.ValueOf(code),
			js.ValueOf(codeVerifier),
			js.ValueOf(string(configJSON)),
		)

		if promise.IsUndefined() || promise.IsNull() {
			errorCh <- fmt.Errorf("exchangeCode returned undefined")
			return
		}

		promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultCh <- args[0]
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errMsg := "exchange failed"
			if len(args) > 0 && !args[0].IsUndefined() {
				errMsg = args[0].String()
			}
			errorCh <- fmt.Errorf("%s", errMsg)
			return nil
		}))
	}()

	select {
	case result := <-resultCh:
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()

		var tokenData struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			Scope        string `json:"scope"`
			TokenType    string `json:"token_type"`
			Error        string `json:"error"`
			ErrorDesc    string `json:"error_description"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &tokenData); err != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", err)
		}

		if tokenData.Error != "" {
			return nil, fmt.Errorf("token endpoint error: %s - %s", tokenData.Error, tokenData.ErrorDesc)
		}

		token := &Token{
			AccessToken:  tokenData.AccessToken,
			RefreshToken: tokenData.RefreshToken,
			Scope:        tokenData.Scope,
		}

		// Calculate expiration time
		if tokenData.ExpiresIn > 0 {
			token.ExpiresAt = time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)
		}

		return token, nil

	case err := <-errorCh:
		return nil, err

	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("token exchange timed out")
	}
}

// GetToken returns a valid access token for a provider
// Automatically refreshes if the token is expired or about to expire
func (m *OAuthManager) GetToken(providerName string) (string, error) {
	// Load existing token
	token, err := m.tokenStore.LoadToken(providerName)
	if err != nil {
		return "", fmt.Errorf("failed to load token: %w", err)
	}
	if token == nil {
		provider, _ := GetProvider(providerName)
		displayName := providerName
		if provider != nil {
			displayName = provider.DisplayName
		}
		return "", fmt.Errorf("please connect %s in Settings", displayName)
	}

	// Check if token needs refresh
	if token.NeedsRefresh() {
		if err := m.RefreshToken(providerName); err != nil {
			return "", fmt.Errorf("token expired and refresh failed: %w", err)
		}
		// Reload token after refresh
		token, err = m.tokenStore.LoadToken(providerName)
		if err != nil {
			return "", fmt.Errorf("failed to reload token after refresh: %w", err)
		}
	}

	if token.IsExpired() {
		return "", fmt.Errorf("token is expired and cannot be refreshed")
	}

	return token.AccessToken, nil
}

// RefreshToken refreshes an expired access token using the refresh token
func (m *OAuthManager) RefreshToken(providerName string) error {
	// Load existing token
	token, err := m.tokenStore.LoadToken(providerName)
	if err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("no token found for %s", providerName)
	}
	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available for %s", providerName)
	}

	// Get provider config
	provider, err := GetProvider(providerName)
	if err != nil {
		return err
	}

	// Perform refresh via JS bridge
	webclaw := js.Global().Get("webclaw")

	// Build token refresh request
	params := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": token.RefreshToken,
		"client_id":     provider.ClientID,
	}
	paramsJSON, _ := json.Marshal(params)

	// Call fetch via JS
	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	go func() {
		// Use jsFetch from bridge.go
		jsFetch := webclaw.Get("jsFetch")
		if jsFetch.IsUndefined() {
			errorCh <- fmt.Errorf("jsFetch not available")
			return
		}

		fetchPromise := jsFetch.Invoke(
			js.ValueOf(provider.TokenURL),
			js.ValueOf(string(paramsJSON)),
		)

		fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultCh <- args[0]
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errMsg := "refresh failed"
			if len(args) > 0 {
				errMsg = args[0].String()
			}
			errorCh <- fmt.Errorf("%s", errMsg)
			return nil
		}))
	}()

	select {
	case result := <-resultCh:
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()

		var refreshData struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"` // May be same or new
			ExpiresIn    int    `json:"expires_in"`
			Scope        string `json:"scope"`
			Error        string `json:"error"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &refreshData); err != nil {
			return fmt.Errorf("failed to parse refresh response: %w", err)
		}

		if refreshData.Error != "" {
			// Refresh failed - clear token and require re-auth
			m.tokenStore.DeleteToken(providerName)
			return fmt.Errorf("refresh failed (re-authentication required): %s", refreshData.Error)
		}

		// Update token
		token.AccessToken = refreshData.AccessToken
		if refreshData.RefreshToken != "" {
			token.RefreshToken = refreshData.RefreshToken
		}
		if refreshData.Scope != "" {
			token.Scope = refreshData.Scope
		}
		token.ExpiresAt = time.Now().Add(time.Duration(refreshData.ExpiresIn) * time.Second)

		// Save updated token
		if err := m.tokenStore.SaveToken(providerName, token); err != nil {
			return fmt.Errorf("failed to save refreshed token: %w", err)
		}

		return nil

	case err := <-errorCh:
		return err

	case <-time.After(30 * time.Second):
		return fmt.Errorf("token refresh timed out")
	}
}

// Disconnect removes the stored token for a provider
func (m *OAuthManager) Disconnect(providerName string) error {
	return m.tokenStore.DeleteToken(providerName)
}

// IsConnected checks if a valid token exists for a provider
func (m *OAuthManager) IsConnected(providerName string) bool {
	token, err := m.tokenStore.LoadToken(providerName)
	if err != nil {
		return false
	}
	if token == nil {
		return false
	}
	// Consider connected if token exists and is not expired (or can be refreshed)
	return !token.IsExpired() || token.RefreshToken != ""
}

// GetConnectionStatus returns detailed connection status for a provider
func (m *OAuthManager) GetConnectionStatus(providerName string) ConnectionStatus {
	token, err := m.tokenStore.LoadToken(providerName)
	if err != nil || token == nil {
		return ConnectionStatus{
			Connected: false,
			Provider:  providerName,
		}
	}

	return ConnectionStatus{
		Connected:       true,
		Provider:        providerName,
		Username:        token.Username,
		Scope:           token.Scope,
		ExpiresAt:       token.ExpiresAt,
		CanRefresh:      token.RefreshToken != "",
		NeedsRefresh:    token.NeedsRefresh(),
		TimeUntilExpiry: token.TimeUntilExpiry(),
	}
}

// ConnectionStatus holds detailed connection information
type ConnectionStatus struct {
	Connected       bool          `json:"connected"`
	Provider        string        `json:"provider"`
	Username        string        `json:"username,omitempty"`
	Scope           string        `json:"scope,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at,omitempty"`
	CanRefresh      bool          `json:"can_refresh"`
	NeedsRefresh    bool          `json:"needs_refresh"`
	TimeUntilExpiry time.Duration `json:"time_until_expiry,omitempty"`
}

// ListConnections returns status for all providers
func (m *OAuthManager) ListConnections() []ConnectionStatus {
	providers := ListProviders()
	statuses := make([]ConnectionStatus, 0, len(providers))

	for _, provider := range providers {
		statuses = append(statuses, m.GetConnectionStatus(provider))
	}

	return statuses
}
