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
	DBVersion   = 1
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

// openDB opens the IndexedDB database and creates object stores if needed
func (s *Storage) openDB() error {
	// For simplicity in Phase 2, we use a synchronous-like pattern
	// In production, you'd want to handle upgrades properly
	req := jsbridge.IDBOpen(DBName, DBVersion)

	// Handle upgrade needed event
	req.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db := event.Get("target").Get("result")

		// Create config object store if it doesn't exist
		if !db.Call("objectStoreNames", "contains").Invoke(ConfigStore).Bool() {
			db.Call("createObjectStore", ConfigStore, map[string]interface{}{"keyPath": "key"})
		}

		return nil
	}))

	// Wait for success/error (simplified - in production use proper async)
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
		return nil, fmt.Errorf("database not open")
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
