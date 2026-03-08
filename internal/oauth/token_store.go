//go:build js && wasm

package oauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/crypto"
	"github.com/gleicon/webclaw/internal/jsbridge"
)

// Token represents an OAuth access token with refresh capability
type Token struct {
	Provider     string    `json:"provider"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	AuthType     string    `json:"auth_type,omitempty"` // "pat" for Personal Access Tokens; empty means OAuth
	// Optional metadata
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

// IsExpired returns true if the token has expired (with 60-second buffer)
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false // No expiration = never expires
	}
	// Check if expired with 60-second buffer for clock skew
	return time.Now().Add(60 * time.Second).After(t.ExpiresAt)
}

// NeedsRefresh returns true if the token needs refreshing
// (expires within 5 minutes or already expired)
func (t *Token) NeedsRefresh() bool {
	if t.RefreshToken == "" {
		return false // No refresh token available
	}
	if t.ExpiresAt.IsZero() {
		return false // Never expires
	}
	// Refresh if expires within 5 minutes
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// TimeUntilExpiry returns duration until token expires
// Returns 0 if already expired or no expiration
func (t *Token) TimeUntilExpiry() time.Duration {
	if t.ExpiresAt.IsZero() || t.IsExpired() {
		return 0
	}
	return time.Until(t.ExpiresAt)
}

// StoredToken is the encrypted on-disk format
type StoredToken struct {
	Provider   string `json:"provider"`
	Ciphertext string `json:"ciphertext"` // base64 encoded
	IV         string `json:"iv"`         // base64 encoded
	Salt       string `json:"salt"`       // base64 encoded
	CreatedAt  int64  `json:"created_at"`
}

// TokenStore provides encrypted OAuth token storage in IndexedDB
type TokenStore struct {
	db         js.Value
	passphrase string // Same passphrase used for API key encryption
}

// NewTokenStore creates a new token store with the given passphrase
// The passphrase should be the same one used for API key encryption
func NewTokenStore(passphrase string) (*TokenStore, error) {
	// Use version 5 to match keystore schema
	req := jsbridge.IDBOpen("webclaw", 5)

	dbCh := make(chan js.Value, 1)
	errCh := make(chan error, 1)

	// Handle upgrade - ensure oauth store exists
	req.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db := event.Get("target").Get("result")

		// Create oauth object store if it doesn't exist
		if !db.Get("objectStoreNames").Call("contains", "oauth").Bool() {
			db.Call("createObjectStore", "oauth", map[string]interface{}{
				"keyPath": "provider",
			})
		}

		return nil
	}))

	req.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		dbCh <- req.Get("result")
		return nil
	}))

	req.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- fmt.Errorf("failed to open database: %v", req.Get("error"))
		return nil
	}))

	req.Set("onblocked", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- fmt.Errorf("database blocked - close other tabs")
		return nil
	}))

	select {
	case db := <-dbCh:
		return &TokenStore{db: db, passphrase: passphrase}, nil
	case err := <-errCh:
		return nil, err
	}
}

// SetPassphrase updates the passphrase used for encryption/decryption
func (ts *TokenStore) SetPassphrase(passphrase string) {
	ts.passphrase = passphrase
}

// SaveToken encrypts and stores an OAuth token
func (ts *TokenStore) SaveToken(provider string, token *Token) error {
	if token == nil {
		return fmt.Errorf("cannot save nil token")
	}
	if provider == "" {
		return fmt.Errorf("provider name required")
	}
	if ts.passphrase == "" {
		return fmt.Errorf("passphrase not set")
	}

	// Set provider on token
	token.Provider = provider

	// Marshal token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encrypt the token data
	encrypted, err := crypto.EncryptWithPassphrase(data, ts.passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Create storage record
	stored := StoredToken{
		Provider:   provider,
		Ciphertext: base64.StdEncoding.EncodeToString(encrypted.Ciphertext),
		IV:         base64.StdEncoding.EncodeToString(encrypted.IV),
		Salt:       base64.StdEncoding.EncodeToString(encrypted.Salt),
		CreatedAt:  time.Now().Unix(),
	}

	return ts.saveToken(stored)
}

// LoadToken retrieves and decrypts an OAuth token
func (ts *TokenStore) LoadToken(provider string) (*Token, error) {
	if provider == "" {
		return nil, fmt.Errorf("provider name required")
	}
	if ts.passphrase == "" {
		return nil, fmt.Errorf("passphrase not set")
	}

	// Load encrypted token from storage
	stored, err := ts.loadStoredToken(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}
	if stored == nil {
		return nil, nil // No token found (not an error)
	}

	// Decode base64 fields
	ciphertext, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(stored.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}
	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decrypt
	encrypted := &crypto.EncryptedData{
		Ciphertext: ciphertext,
		IV:         iv,
		Salt:       salt,
	}

	plaintext, err := crypto.DecryptWithPassphrase(encrypted, ts.passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token (wrong passphrase?): %w", err)
	}

	// Unmarshal token
	var token Token
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteToken removes a stored token for a provider
func (ts *TokenStore) DeleteToken(provider string) error {
	if provider == "" {
		return fmt.Errorf("provider name required")
	}

	// Check if store exists
	if ts.db.IsUndefined() || ts.db.IsNull() {
		return fmt.Errorf("database not open")
	}
	storeNames := ts.db.Get("objectStoreNames")
	if !storeNames.Call("contains", "oauth").Bool() {
		return nil // Store doesn't exist, nothing to delete
	}

	promise := jsbridge.IDBDelete(ts.db, "oauth", provider)

	resultCh := make(chan struct{}, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("delete failed: %v", args[0])
		return nil
	}))

	select {
	case <-resultCh:
		return nil
	case err := <-errorCh:
		return err
	}
}

// HasToken checks if a token exists for a provider (without decrypting)
func (ts *TokenStore) HasToken(provider string) bool {
	if provider == "" {
		return false
	}

	stored, err := ts.loadStoredToken(provider)
	if err != nil {
		return false
	}
	return stored != nil
}

// ListProviders returns a list of all providers with stored tokens
func (ts *TokenStore) ListProviders() ([]string, error) {
	// Check if store exists
	if ts.db.IsUndefined() || ts.db.IsNull() {
		return nil, fmt.Errorf("database not open")
	}
	storeNames := ts.db.Get("objectStoreNames")
	if !storeNames.Call("contains", "oauth").Bool() {
		return []string{}, nil // Store doesn't exist yet
	}

	transaction := ts.db.Call("transaction", "oauth", "readonly")
	store := transaction.Call("objectStore", "oauth")
	request := store.Call("getAllKeys")

	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- request.Get("result")
		return nil
	}))

	request.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("failed to list providers: %v", request.Get("error"))
		return nil
	}))

	select {
	case result := <-resultCh:
		if result.IsNull() || result.IsUndefined() {
			return []string{}, nil
		}

		// Convert JS array to Go slice
		length := result.Get("length").Int()
		providers := make([]string, 0, length)
		for i := 0; i < length; i++ {
			provider := result.Index(i).String()
			providers = append(providers, provider)
		}
		return providers, nil

	case err := <-errorCh:
		return nil, err
	}
}

// ClearAllTokens removes all stored OAuth tokens (use with caution)
func (ts *TokenStore) ClearAllTokens() error {
	providers, err := ts.ListProviders()
	if err != nil {
		return err
	}

	for _, provider := range providers {
		if err := ts.DeleteToken(provider); err != nil {
			return fmt.Errorf("failed to delete token for %s: %w", provider, err)
		}
	}

	return nil
}

// saveToken stores an encrypted token in IndexedDB
func (ts *TokenStore) saveToken(stored StoredToken) error {
	// Check if store exists
	if ts.db.IsUndefined() || ts.db.IsNull() {
		return fmt.Errorf("database not open")
	}
	storeNames := ts.db.Get("objectStoreNames")
	if !storeNames.Call("contains", "oauth").Bool() {
		return fmt.Errorf("oauth store not available")
	}

	// Convert to JS object via JSON
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal stored token: %w", err)
	}

	var tokenObj map[string]interface{}
	if err := json.Unmarshal(data, &tokenObj); err != nil {
		return fmt.Errorf("failed to unmarshal token: %w", err)
	}

	promise := jsbridge.IDBPut(ts.db, "oauth", tokenObj)

	resultCh := make(chan struct{}, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("save failed: %v", args[0])
		return nil
	}))

	select {
	case <-resultCh:
		return nil
	case err := <-errorCh:
		return err
	}
}

// loadStoredToken retrieves the encrypted token record from IndexedDB
func (ts *TokenStore) loadStoredToken(provider string) (*StoredToken, error) {
	// Check if store exists
	if ts.db.IsUndefined() || ts.db.IsNull() {
		return nil, nil // Store not ready, treat as not found
	}
	storeNames := ts.db.Get("objectStoreNames")
	if !storeNames.Call("contains", "oauth").Bool() {
		return nil, nil // Store doesn't exist yet, treat as not found
	}

	promise := jsbridge.IDBGet(ts.db, "oauth", provider)

	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- args[0]
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("load failed: %v", args[0])
		return nil
	}))

	select {
	case result := <-resultCh:
		if result.IsNull() || result.IsUndefined() {
			return nil, nil
		}

		// Convert to StoredToken via JSON
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()
		var stored StoredToken
		if err := json.Unmarshal([]byte(jsonStr), &stored); err != nil {
			return nil, fmt.Errorf("failed to parse stored token: %w", err)
		}
		return &stored, nil

	case err := <-errorCh:
		return nil, err
	}
}
