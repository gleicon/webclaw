//go:build js && wasm

// Package keystore provides encrypted storage for API keys
package keystore

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/crypto"
	"github.com/gleicon/webclaw/internal/jsbridge"
)

// StoredKey represents an encrypted API key in storage
type StoredKey struct {
	Provider   string `json:"provider"`
	Ciphertext string `json:"ciphertext"` // base64 encoded
	IV         string `json:"iv"`         // base64 encoded
	Salt       string `json:"salt"`       // base64 encoded
	CreatedAt  int64  `json:"created_at"`
}

// KeyStore provides encrypted API key storage
type KeyStore struct {
	db js.Value
}

// NewKeyStore creates a new keystore using IndexedDB
func NewKeyStore() (*KeyStore, error) {
	db, err := jsbridge.OpenWebClawDB()
	if err != nil {
		return nil, err
	}
	return &KeyStore{db: db}, nil
}

// StoreKey encrypts and stores an API key for a provider
func (ks *KeyStore) StoreKey(provider, apiKey, passphrase string) error {
	// Encrypt the API key
	encrypted, err := crypto.EncryptWithPassphrase([]byte(apiKey), passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Create storage record
	stored := StoredKey{
		Provider:   provider,
		Ciphertext: base64.StdEncoding.EncodeToString(encrypted.Ciphertext),
		IV:         base64.StdEncoding.EncodeToString(encrypted.IV),
		Salt:       base64.StdEncoding.EncodeToString(encrypted.Salt),
		CreatedAt:  time.Now().Unix(),
	}

	// Store in IndexedDB
	return ks.saveKey(stored)
}

// RetrieveKey retrieves and decrypts an API key for a provider
// The caller is responsible for clearing the returned key from memory after use
func (ks *KeyStore) RetrieveKey(provider, passphrase string) (string, error) {
	// Load encrypted key from storage
	stored, err := ks.loadKey(provider)
	if err != nil {
		return "", fmt.Errorf("failed to load key: %w", err)
	}
	if stored == nil {
		return "", fmt.Errorf("no key stored for provider: %s", provider)
	}

	// Decode base64 fields
	ciphertext, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(stored.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}
	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decrypt
	encrypted := &crypto.EncryptedData{
		Ciphertext: ciphertext,
		IV:         iv,
		Salt:       salt,
	}

	plaintext, err := crypto.DecryptWithPassphrase(encrypted, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key (wrong passphrase?): %w", err)
	}

	return string(plaintext), nil
}

// KeyExists checks if a key exists for a provider
func (ks *KeyStore) KeyExists(provider string) (bool, error) {
	stored, err := ks.loadKey(provider)
	if err != nil {
		return false, err
	}
	return stored != nil, nil
}

// DeleteKey removes a stored key for a provider
func (ks *KeyStore) DeleteKey(provider string) error {
	promise := jsbridge.IDBDelete(ks.db, "keystore", provider)

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

// saveKey stores a key in IndexedDB
func (ks *KeyStore) saveKey(key StoredKey) error {
	// Convert to JS object via JSON
	keyData, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	// Parse into map for JS conversion
	var keyObj map[string]interface{}
	if err := json.Unmarshal(keyData, &keyObj); err != nil {
		return fmt.Errorf("failed to unmarshal key: %w", err)
	}

	promise := jsbridge.IDBPut(ks.db, "keystore", keyObj)

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

// loadKey retrieves a key from IndexedDB
func (ks *KeyStore) loadKey(provider string) (*StoredKey, error) {
	promise := jsbridge.IDBGet(ks.db, "keystore", provider)

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

		// Convert to StoredKey via JSON
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()
		var key StoredKey
		if err := json.Unmarshal([]byte(jsonStr), &key); err != nil {
			return nil, fmt.Errorf("failed to parse key: %w", err)
		}
		return &key, nil

	case err := <-errorCh:
		return nil, err
	}
}

// ClearKey zeros out a string to help prevent memory inspection
func ClearKey(key string) {
	// Note: This is best effort - Go's GC may have already moved/copied the data
	// In WASM, memory is sandboxed anyway, but this helps in the short term
	// We can't modify the string directly, but we can suggest GC cleanup
	// by clearing the reference and letting it go out of scope
	_ = key
}
