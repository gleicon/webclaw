//go:build js && wasm

package telemetry

import (
	"strings"
	"time"
)

// ErrorLevel indicates severity
type ErrorLevel string

const (
	ErrorLevelWarning  ErrorLevel = "warning"
	ErrorLevelError    ErrorLevel = "error"
	ErrorLevelCritical ErrorLevel = "critical"
)

// ErrorRecord represents a captured error
type ErrorRecord struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     ErrorLevel             `json:"level"`
	Category  string                 `json:"category"` // "provider", "tool", "agent", "system"
	Message   string                 `json:"message"`
	Stack     string                 `json:"stack,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// EventRecord represents a significant event (not necessarily an error)
type EventRecord struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Category  string                 `json:"category"`
	Action    string                 `json:"action"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// SanitizeContext removes sensitive data from context maps.
// Keys containing "api_key", "token", "password", "secret", or "key"
// (case-insensitive) are replaced with "[REDACTED]".
func SanitizeContext(ctx map[string]interface{}) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	sensitiveKeys := []string{"api_key", "token", "password", "secret", "key"}
	cleaned := make(map[string]interface{}, len(ctx))
	for k, v := range ctx {
		lowerKey := strings.ToLower(k)
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			cleaned[k] = "[REDACTED]"
		} else {
			cleaned[k] = v
		}
	}
	return cleaned
}
