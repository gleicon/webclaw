//go:build js && wasm

package telemetry

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// Config holds telemetry configuration
type Config struct {
	Enabled         bool // Enable/disable telemetry collection
	MaxErrorsStored int  // Maximum errors to keep in storage (default: 100)
	MaxEventsStored int  // Maximum events to keep in storage (default: 50)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		MaxErrorsStored: 100,
		MaxEventsStored: 50,
	}
}

// Storage is the persistence interface for telemetry data
type Storage interface {
	StoreError(ctx context.Context, err ErrorRecord) error
	StoreEvent(ctx context.Context, evt EventRecord) error
	GetErrors(ctx context.Context, limit int) ([]ErrorRecord, error)
	GetEvents(ctx context.Context, limit int) ([]EventRecord, error)
}

// Collector is the main telemetry object — create one per process via NewCollector
type Collector struct {
	config  Config
	storage Storage
}

// NewCollector creates a telemetry collector backed by the provided storage
func NewCollector(config Config, storage Storage) *Collector {
	return &Collector{
		config:  config,
		storage: storage,
	}
}

// TelemetryReport is a snapshot suitable for export / debugging
type TelemetryReport struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Errors      []ErrorRecord `json:"errors"`
	Events      []EventRecord `json:"events"`
}

// RecordError captures an error with full context.
// Storage write is performed asynchronously so it never blocks the caller.
func (c *Collector) RecordError(level ErrorLevel, category string, err error, ctx map[string]interface{}) {
	if !c.config.Enabled || err == nil {
		return
	}

	record := ErrorRecord{
		ID:        generateID("err"),
		Timestamp: time.Now(),
		Level:     level,
		Category:  category,
		Message:   err.Error(),
		Stack:     string(debug.Stack()),
		Context:   SanitizeContext(ctx),
	}

	go func() {
		storeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if storeErr := c.storage.StoreError(storeCtx, record); storeErr != nil {
			// Telemetry failure must not cascade — log to console only
			fmt.Printf("[Telemetry] Failed to store error: %v\n", storeErr)
		}
	}()
}

// RecordEvent captures a significant event (not necessarily an error).
// Duration should be in milliseconds; pass 0 if not measured.
func (c *Collector) RecordEvent(category, action string, durationMs int64, ctx map[string]interface{}) {
	if !c.config.Enabled {
		return
	}

	record := EventRecord{
		ID:        generateID("evt"),
		Timestamp: time.Now(),
		Category:  category,
		Action:    action,
		Duration:  durationMs,
		Context:   SanitizeContext(ctx),
	}

	go func() {
		storeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if storeErr := c.storage.StoreEvent(storeCtx, record); storeErr != nil {
			fmt.Printf("[Telemetry] Failed to store event: %v\n", storeErr)
		}
	}()
}

// GetReport generates a telemetry report for debugging / export.
func (c *Collector) GetReport(ctx context.Context) (*TelemetryReport, error) {
	errors, err := c.storage.GetErrors(ctx, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get errors: %w", err)
	}

	events, err := c.storage.GetEvents(ctx, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return &TelemetryReport{
		GeneratedAt: time.Now(),
		Errors:      errors,
		Events:      events,
	}, nil
}

// generateID creates a unique telemetry ID with the given prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
