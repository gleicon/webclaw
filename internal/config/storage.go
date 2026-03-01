//go:build js && wasm

package config

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

const (
	DBName      = "webclaw"
	DBVersion   = 5 // Bumped to match identity/keystore
	ConfigStore = "config"
)

// Storage handles persistence of config to IndexedDB
type Storage struct {
	db js.Value
}

// NewStorage creates a new storage instance, opening the IndexedDB database
func NewStorage() (*Storage, error) {
	s := &Storage{}
	if err := s.openDB(); err != nil {
		return nil, err
	}
	return s, nil
}

// openDB opens the IndexedDB database and creates ALL object stores
func (s *Storage) openDB() error {
	req := jsbridge.IDBOpen(DBName, DBVersion)

	// Handle upgrade needed - create ALL stores for all packages
	req.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db := event.Get("target").Get("result")

		// Create config object store
		if !db.Get("objectStoreNames").Call("contains", ConfigStore).Bool() {
			db.Call("createObjectStore", ConfigStore, map[string]interface{}{"keyPath": "key"})
		}

		// Create identity object store (needed by identity package)
		if !db.Get("objectStoreNames").Call("contains", "identity").Bool() {
			db.Call("createObjectStore", "identity", map[string]interface{}{
				"keyPath": "filename",
			})
		}

		// Create keystore object store (needed by keystore package)
		if !db.Get("objectStoreNames").Call("contains", "keystore").Bool() {
			db.Call("createObjectStore", "keystore", map[string]interface{}{
				"keyPath": "provider",
			})
		}

		return nil
	}))

	// Wait for success/error
	successCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	req.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		successCh <- req.Get("result")
		return nil
	}))

	req.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("failed to open IndexedDB: %v", req.Get("error"))
		return nil
	}))

	req.Set("onblocked", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("IndexedDB blocked - close other tabs")
		return nil
	}))

	select {
	case db := <-successCh:
		s.db = db
		return nil
	case err := <-errorCh:
		return err
	}
}

// GetConfig retrieves the config from IndexedDB
// Returns nil if no config exists
func (s *Storage) GetConfig() (*Config, error) {
	if s.db.IsUndefined() || s.db.IsNull() {
		return nil, nil // Database not ready, treat as no config
	}

	// Check if store exists
	if !s.db.Get("objectStoreNames").Call("contains", ConfigStore).Bool() {
		return nil, nil // Store doesn't exist yet, treat as no config
	}

	// Wrap IndexedDB operation in promise
	promise := jsbridge.IDBGet(s.db, ConfigStore, ConfigKey)

	// Wait for promise resolution
	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- args[0]
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("IDB get failed: %v", args[0])
		return nil
	}))

	select {
	case result := <-resultCh:
		if result.IsNull() || result.IsUndefined() {
			return nil, nil // No config exists
		}

		// Extract the value from the result object
		// IndexedDB stores {key: "webclaw:config", value: {...}}
		value := result.Get("value")
		if value.IsUndefined() || value.IsNull() {
			return nil, nil
		}

		// Convert JS object to JSON string, then parse to Config
		jsonStr := js.Global().Get("JSON").Call("stringify", value).String()
		return FromJSON([]byte(jsonStr))

	case err := <-errorCh:
		return nil, err
	}
}

// SetConfig persists the config to IndexedDB
func (s *Storage) SetConfig(cfg *Config) error {
	if s.db.IsUndefined() || s.db.IsNull() {
		return fmt.Errorf("database not open")
	}

	// Check if store exists
	if !s.db.Get("objectStoreNames").Call("contains", ConfigStore).Bool() {
		return fmt.Errorf("config store not available")
	}

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		return err
	}

	cfg.UpdateTimestamps()

	// Convert config to JS object
	jsonBytes, err := cfg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Parse JSON to get JS object
	var configMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Add the key field for IndexedDB
	configMap["key"] = ConfigKey

	// Wrap in promise and wait
	promise := jsbridge.IDBPut(s.db, ConfigStore, configMap)

	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- args[0]
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("IDB put failed: %v", args[0])
		return nil
	}))

	select {
	case <-resultCh:
		return nil
	case err := <-errorCh:
		return err
	}
}

// ConfigExists checks if a config exists in IndexedDB
func (s *Storage) ConfigExists() (bool, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return false, err
	}
	return cfg != nil, nil
}

// Close closes the IndexedDB connection
func (s *Storage) Close() {
	if !s.db.IsUndefined() && !s.db.IsNull() {
		s.db.Call("close")
	}
}
