//go:build !js

// Package tools_test tests the tool registry natively (not in WASM).
// This file re-implements registry logic inline since registry.go is WASM-only.
package tools_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// --- Inline registry implementation for native testing ---
// (mirrors internal/tools/registry.go contract exactly)

type testToolResult struct {
	Content        string
	DisplayContent string
	IsError        bool
	ToolName       string
	Status         string
}

type testTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Execute     func(ctx context.Context, params map[string]interface{}) (*testToolResult, error)
}

type testRegistry struct {
	mu    sync.RWMutex
	tools map[string]*testTool
}

func newTestRegistry() *testRegistry {
	return &testRegistry{tools: make(map[string]*testTool)}
}

func (r *testRegistry) Register(t *testTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name] = t
}

func (r *testRegistry) Dispatch(ctx context.Context, name string, params map[string]interface{}) (*testToolResult, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, params)
}

func (r *testRegistry) ToAPISchema() []map[string]interface{} {
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

func (r *testRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// --- Tests ---

func TestRegistryDispatch(t *testing.T) {
	reg := newTestRegistry()

	// Register a mock tool
	mockTool := &testTool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{"type": "string"},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*testToolResult, error) {
			return &testToolResult{
				Content:        "test result",
				DisplayContent: "test display",
				IsError:        false,
				ToolName:       "test_tool",
				Status:         "done",
			}, nil
		},
	}
	reg.Register(mockTool)

	// Test: Dispatch returns tool result for registered name
	result, err := reg.Dispatch(context.Background(), "test_tool", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("Dispatch failed for registered tool: %v", err)
	}
	if result.Content != "test result" {
		t.Errorf("Expected 'test result', got '%s'", result.Content)
	}
	if result.IsError {
		t.Error("Expected IsError=false")
	}
	if result.Status != "done" {
		t.Errorf("Expected Status='done', got '%s'", result.Status)
	}
	if result.ToolName != "test_tool" {
		t.Errorf("Expected ToolName='test_tool', got '%s'", result.ToolName)
	}
}

func TestRegistryDispatchUnknownTool(t *testing.T) {
	reg := newTestRegistry()

	// Test: Dispatch returns error for unknown tool name
	_, err := reg.Dispatch(context.Background(), "nonexistent_tool", nil)
	if err == nil {
		t.Fatal("Expected error for unknown tool, got nil")
	}
}

func TestRegistryToAPISchema(t *testing.T) {
	reg := newTestRegistry()

	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{"type": "string", "description": "The URL to fetch"},
		},
		"required": []string{"url"},
	}

	reg.Register(&testTool{
		Name:        "web_fetch",
		Description: "Fetch a URL",
		InputSchema: inputSchema,
		Execute: func(ctx context.Context, params map[string]interface{}) (*testToolResult, error) {
			return &testToolResult{}, nil
		},
	})

	// Test: ToAPISchema returns slice with Name, Description, InputSchema per tool
	schemas := reg.ToAPISchema()
	if len(schemas) != 1 {
		t.Fatalf("Expected 1 schema, got %d", len(schemas))
	}

	schema := schemas[0]
	if schema["name"] != "web_fetch" {
		t.Errorf("Expected name='web_fetch', got %v", schema["name"])
	}
	if schema["description"] != "Fetch a URL" {
		t.Errorf("Expected description='Fetch a URL', got %v", schema["description"])
	}
	if schema["input_schema"] == nil {
		t.Error("Expected input_schema to be present")
	}
}

func TestRegistryList(t *testing.T) {
	reg := newTestRegistry()

	reg.Register(&testTool{Name: "tool_a", Execute: func(ctx context.Context, params map[string]interface{}) (*testToolResult, error) {
		return &testToolResult{}, nil
	}})
	reg.Register(&testTool{Name: "tool_b", Execute: func(ctx context.Context, params map[string]interface{}) (*testToolResult, error) {
		return &testToolResult{}, nil
	}})

	// Test: List returns all registered tool names
	names := reg.List()
	if len(names) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["tool_a"] || !nameSet["tool_b"] {
		t.Errorf("Expected tool_a and tool_b in list, got %v", names)
	}
}

func TestToolResultFields(t *testing.T) {
	// Test: ToolResult has Content, DisplayContent, IsError, ToolName, Status fields
	result := &testToolResult{
		Content:        "full content",
		DisplayContent: "display",
		IsError:        true,
		ToolName:       "my_tool",
		Status:         "error",
	}

	if result.Content != "full content" {
		t.Errorf("Content field mismatch")
	}
	if result.DisplayContent != "display" {
		t.Errorf("DisplayContent field mismatch")
	}
	if !result.IsError {
		t.Errorf("IsError field mismatch")
	}
	if result.ToolName != "my_tool" {
		t.Errorf("ToolName field mismatch")
	}
	if result.Status != "error" {
		t.Errorf("Status field mismatch")
	}
}
