//go:build js && wasm

package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

const (
	IdentityStoreName = "identity"
)

// IdentityFile represents a single identity file
type IdentityFile struct {
	Filename   string    `json:"filename"`
	Content    string    `json:"content"`
	Size       int       `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Checksum   string    `json:"checksum"`
}

// Store provides persistence for identity files
type Store struct {
	db js.Value
}

// NewStore creates a new identity store
func NewStore() (*Store, error) {
	s := &Store{}
	if err := s.openDB(); err != nil {
		return nil, err
	}
	return s, nil
}

// openDB opens the IndexedDB connection and creates all required object stores
func (s *Store) openDB() error {
	// Use version 5 to force schema upgrade
	req := jsbridge.IDBOpen("webclaw", 5)

	dbCh := make(chan js.Value, 1)
	errCh := make(chan error, 1)

	// Handle upgrade - create ALL stores (identity, keystore, config)
	req.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db := event.Get("target").Get("result")

		// Create identity object store
		if !db.Get("objectStoreNames").Call("contains", "identity").Bool() {
			db.Call("createObjectStore", "identity", map[string]interface{}{
				"keyPath": "filename",
			})
		}

		// Create keystore object store (needed by other packages)
		if !db.Get("objectStoreNames").Call("contains", "keystore").Bool() {
			db.Call("createObjectStore", "keystore", map[string]interface{}{
				"keyPath": "provider",
			})
		}

		// Create config object store
		if !db.Get("objectStoreNames").Call("contains", "config").Bool() {
			db.Call("createObjectStore", "config", map[string]interface{}{
				"keyPath": "key",
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
		s.db = db
		return nil
	case err := <-errCh:
		return err
	}
}

// Get retrieves an identity file by filename
func (s *Store) Get(filename string) (*IdentityFile, error) {
	if s.db.IsUndefined() || s.db.IsNull() {
		return nil, fmt.Errorf("database not open")
	}

	promise := jsbridge.IDBGet(s.db, IdentityStoreName, filename)

	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- args[0]
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("get failed: %v", args[0])
		return nil
	}))

	select {
	case result := <-resultCh:
		if result.IsNull() || result.IsUndefined() {
			return nil, nil // Not found
		}

		// Parse JSON
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()
		var file IdentityFile
		if err := json.Unmarshal([]byte(jsonStr), &file); err != nil {
			return nil, fmt.Errorf("failed to parse file: %w", err)
		}
		return &file, nil

	case err := <-errorCh:
		return nil, err
	}
}

// Put stores an identity file
func (s *Store) Put(file *IdentityFile) error {
	if s.db.IsUndefined() || s.db.IsNull() {
		return fmt.Errorf("database not open")
	}

	// Update metadata
	file.Size = len(file.Content)
	file.ModifiedAt = time.Now()
	file.Checksum = calculateChecksum(file.Content)

	// Convert to JS object
	data, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to marshal file: %w", err)
	}

	var fileObj map[string]interface{}
	if err := json.Unmarshal(data, &fileObj); err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	promise := jsbridge.IDBPut(s.db, IdentityStoreName, fileObj)

	resultCh := make(chan struct{}, 1)
	errorCh := make(chan error, 1)

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("put failed: %v", args[0])
		return nil
	}))

	select {
	case <-resultCh:
		return nil
	case err := <-errorCh:
		return err
	}
}

// Delete removes an identity file
func (s *Store) Delete(filename string) error {
	if s.db.IsUndefined() || s.db.IsNull() {
		return fmt.Errorf("database not open")
	}

	promise := jsbridge.IDBDelete(s.db, IdentityStoreName, filename)

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

// List returns all identity filenames
func (s *Store) List() ([]string, error) {
	if s.db.IsUndefined() || s.db.IsNull() {
		return nil, fmt.Errorf("database not open")
	}

	// Check if store exists first
	storeNames := s.db.Get("objectStoreNames")
	if !storeNames.Call("contains", IdentityStoreName).Bool() {
		// Store doesn't exist yet - return empty list
		return []string{}, nil
	}

	transaction := s.db.Call("transaction", IdentityStoreName, "readonly")
	store := transaction.Call("objectStore", IdentityStoreName)

	// Try to use getAllKeys if available
	keysPromise := store.Call("getAllKeys")

	// Check if getAllKeys returned a valid promise with 'then' method
	if keysPromise.IsUndefined() || keysPromise.IsNull() {
		return []string{}, nil
	}
	// Verify it has a 'then' method (is actually a Promise)
	if keysPromise.Get("then").IsUndefined() || keysPromise.Get("then").IsNull() {
		return []string{}, nil
	}

	resultCh := make(chan []string, 1)
	errorCh := make(chan error, 1)

	keysPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		keys := args[0]
		length := keys.Length()
		filenames := make([]string, length)
		for i := 0; i < length; i++ {
			filenames[i] = keys.Index(i).String()
		}
		resultCh <- filenames
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorCh <- fmt.Errorf("list failed: %v", args[0])
		return nil
	}))

	select {
	case filenames := <-resultCh:
		return filenames, nil
	case err := <-errorCh:
		return nil, err
	}
}

// Exists checks if a file exists
func (s *Store) Exists(filename string) (bool, error) {
	file, err := s.Get(filename)
	if err != nil {
		return false, err
	}
	return file != nil, nil
}

// LoadDefaults creates default identity files if they don't exist
func (s *Store) LoadDefaults() error {
	files := []struct {
		name    string
		content string
	}{
		{"IDENTITY.md", DefaultIdentityContent()},
		{"SOUL.md", DefaultSoulContent()},
		{"USER.md", DefaultUserContent()},
		{"AGENTS.md", DefaultAgentsContent()},
		{"TOOLS.md", DefaultToolsContent()},
		{"HEARTBEAT.md", DefaultHeartbeatContent()},
	}

	for _, f := range files {
		exists, err := s.Exists(f.name)
		if err != nil {
			return fmt.Errorf("failed to check %s: %w", f.name, err)
		}

		if !exists {
			file := &IdentityFile{
				Filename: f.name,
				Content:  f.content,
			}
			if err := s.Put(file); err != nil {
				return fmt.Errorf("failed to create %s: %w", f.name, err)
			}
		}
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() {
	if !s.db.IsUndefined() && !s.db.IsNull() {
		s.db.Call("close")
	}
}

// AppendToMemoryFile appends content to the MEMORY.md identity file
// Creates the file if it doesn't exist
func (s *Store) AppendToMemoryFile(content string) error {
	// Get current MEMORY.md content
	current, err := s.Get("MEMORY.md")
	if err != nil {
		return fmt.Errorf("failed to get MEMORY.md: %w", err)
	}

	if current == nil {
		// MEMORY.md doesn't exist yet, create it with header
		current = &IdentityFile{
			Filename: "MEMORY.md",
			Content:  "# Memory\n\nPersistent facts extracted from conversations.\n\n---\n",
		}
	}

	// Append new content
	current.Content += content
	current.ModifiedAt = time.Now()
	current.Checksum = calculateChecksum(current.Content)

	// Save back
	return s.Put(current)
}

// calculateChecksum computes SHA256 hash of content
func calculateChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
