//go:build js && wasm

package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry holds all registered tools and dispatches calls to them.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry by its name.
func (r *Registry) Register(t *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name] = t
}

// Dispatch looks up the tool by name and calls its Execute function.
// Returns an error if the tool is not registered.
func (r *Registry) Dispatch(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, params)
}

// ToAPISchema returns Anthropic-compatible tool definitions for injection into LLM requests.
// Each entry has "name", "description", and "input_schema" keys.
func (r *Registry) ToAPISchema() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemas := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		schemas = append(schemas, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.InputSchema,
		})
	}
	return schemas
}

// List returns the names of all registered tools.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
