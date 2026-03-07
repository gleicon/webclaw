//go:build js && wasm

package telemetry

import (
	"context"
	"fmt"
)

// global holds the package-level collector used by agent and provider packages.
// Initialised to a no-op collector so callers never need to nil-check.
var global *Collector

func init() {
	// Default: telemetry enabled with localStorage backend
	// main.go can call Init() to override with custom config.
	global = NewCollector(DefaultConfig(), NewLocalStorageBackend())
}

// Init replaces the global collector. Call this from main() to configure
// telemetry before any errors can be recorded.
func Init(cfg Config, storage Storage) {
	global = NewCollector(cfg, storage)
}

// Disable replaces the global collector with a disabled no-op collector.
func Disable() {
	global = NewCollector(Config{Enabled: false}, NewLocalStorageBackend())
}

// RecordError records an error against the global collector.
// Safe to call before Init() — uses default configuration.
func RecordError(level ErrorLevel, category string, err error, ctx map[string]interface{}) {
	if global != nil {
		global.RecordError(level, category, err, ctx)
	}
}

// RecordEvent records an event against the global collector.
func RecordEvent(category, action string, durationMs int64, ctx map[string]interface{}) {
	if global != nil {
		global.RecordEvent(category, action, durationMs, ctx)
	}
}

// GetGlobal returns the global collector (useful for generating reports from JS bridge).
func GetGlobal() *Collector {
	return global
}

// GetReport is a convenience wrapper that generates a report from the global collector.
func GetReport() (*TelemetryReport, error) {
	if global == nil {
		return nil, fmt.Errorf("telemetry not initialized")
	}
	return global.GetReport(context.Background())
}
