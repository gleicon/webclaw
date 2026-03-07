//go:build js && wasm

package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
)

const (
	// localStorage keys for telemetry — keep under 5MB browser limit
	errorsKey = "webclaw:telemetry:errors"
	eventsKey = "webclaw:telemetry:events"

	// Retention limits
	maxStoredErrors = 100
	maxStoredEvents = 50
)

// LocalStorageBackend implements Storage using browser localStorage.
// localStorage is synchronous, zero-dependency, and its 5 MB limit comfortably
// holds 100 error records (typical record < 4 KB).
type LocalStorageBackend struct{}

// NewLocalStorageBackend creates a new localStorage-backed telemetry storage
func NewLocalStorageBackend() *LocalStorageBackend {
	return &LocalStorageBackend{}
}

// StoreError appends an error record and trims the list to maxStoredErrors
func (s *LocalStorageBackend) StoreError(_ context.Context, rec ErrorRecord) error {
	records, err := s.readErrors()
	if err != nil {
		records = []ErrorRecord{}
	}

	records = append(records, rec)

	// Trim oldest entries to stay within retention limit
	if len(records) > maxStoredErrors {
		records = records[len(records)-maxStoredErrors:]
	}

	return s.writeErrors(records)
}

// StoreEvent appends an event record and trims the list to maxStoredEvents
func (s *LocalStorageBackend) StoreEvent(_ context.Context, evt EventRecord) error {
	events, err := s.readEvents()
	if err != nil {
		events = []EventRecord{}
	}

	events = append(events, evt)

	if len(events) > maxStoredEvents {
		events = events[len(events)-maxStoredEvents:]
	}

	return s.writeEvents(events)
}

// GetErrors returns the most recent errors up to limit
func (s *LocalStorageBackend) GetErrors(_ context.Context, limit int) ([]ErrorRecord, error) {
	records, err := s.readErrors()
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}

	return records, nil
}

// GetEvents returns the most recent events up to limit
func (s *LocalStorageBackend) GetEvents(_ context.Context, limit int) ([]EventRecord, error) {
	events, err := s.readEvents()
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	return events, nil
}

// readErrors loads error records from localStorage
func (s *LocalStorageBackend) readErrors() ([]ErrorRecord, error) {
	return readJSONSlice[ErrorRecord](errorsKey)
}

// writeErrors saves error records to localStorage
func (s *LocalStorageBackend) writeErrors(records []ErrorRecord) error {
	return writeJSONSlice(errorsKey, records)
}

// readEvents loads event records from localStorage
func (s *LocalStorageBackend) readEvents() ([]EventRecord, error) {
	return readJSONSlice[EventRecord](eventsKey)
}

// writeEvents saves event records to localStorage
func (s *LocalStorageBackend) writeEvents(events []EventRecord) error {
	return writeJSONSlice(eventsKey, events)
}

// readJSONSlice reads and parses a JSON array from localStorage
func readJSONSlice[T any](key string) ([]T, error) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil, fmt.Errorf("localStorage not available")
	}

	raw := ls.Call("getItem", key)
	if raw.IsNull() || raw.IsUndefined() {
		return []T{}, nil // Not yet written — treat as empty
	}

	var result []T
	if err := json.Unmarshal([]byte(raw.String()), &result); err != nil {
		// Corrupted data — reset and start fresh
		ls.Call("removeItem", key)
		return []T{}, nil
	}

	return result, nil
}

// writeJSONSlice serialises a slice and writes it to localStorage
func writeJSONSlice[T any](key string, data []T) error {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return fmt.Errorf("localStorage not available")
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry: %w", err)
	}

	ls.Call("setItem", key, string(jsonBytes))
	return nil
}
