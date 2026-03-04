//go:build js && wasm

package e2e

import (
	"context"
	"testing"

	"github.com/gleicon/webclaw/internal/tools"
)

// TestToolFlow_RegistryToProvider verifies that tools flow from registry to provider
func TestToolFlow_RegistryToProvider(t *testing.T) {
	// Create tool registry with test tool
	registry := tools.NewRegistry()
	registry.Register(&tools.Tool{
		Name:        "test_echo",
		Description: "Echoes back the input",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string"},
			},
			"required": []string{"message"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			msg := params["message"].(string)
			return &tools.ToolResult{Content: msg}, nil
		},
	})

	// Verify ToAPISchema returns tools
	schemas := registry.ToAPISchema()
	if len(schemas) != 1 {
		t.Errorf("expected 1 tool schema, got %d", len(schemas))
	}

	// Verify schema has required fields
	schema := schemas[0]
	if schema["name"] != "test_echo" {
		t.Errorf("expected name 'test_echo', got %v", schema["name"])
	}

	// Verify description
	if schema["description"] != "Echoes back the input" {
		t.Errorf("expected description 'Echoes back the input', got %v", schema["description"])
	}

	// Verify input_schema exists
	inputSchema, ok := schema["input_schema"].(map[string]interface{})
	if !ok {
		t.Errorf("expected input_schema to be map[string]interface{}, got %T", schema["input_schema"])
	}

	// Verify schema type
	if inputSchema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", inputSchema["type"])
	}

	// Verify List() returns tool names
	toolNames := registry.List()
	if len(toolNames) != 1 {
		t.Errorf("expected 1 tool name, got %d", len(toolNames))
	}
	if toolNames[0] != "test_echo" {
		t.Errorf("expected tool name 'test_echo', got %s", toolNames[0])
	}
}

// TestToolFlow_Dispatch verifies tool dispatch through registry
func TestToolFlow_Dispatch(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&tools.Tool{
		Name:        "calculator",
		Description: "Adds two numbers",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number"},
				"b": map[string]interface{}{"type": "number"},
			},
			"required": []string{"a", "b"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			a := params["a"].(float64)
			b := params["b"].(float64)
			sum := a + b
			return &tools.ToolResult{
				Content:        string(rune(int(sum))), // Simple conversion for test
				DisplayContent: "Result: " + string(rune(int(sum))),
			}, nil
		},
	})

	// Test dispatch
	ctx := context.Background()
	result, err := registry.Dispatch(ctx, "calculator", map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	})
	if err != nil {
		t.Errorf("dispatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

// TestToolFlow_MultipleTools verifies multiple tools in registry
func TestToolFlow_MultipleTools(t *testing.T) {
	registry := tools.NewRegistry()

	// Register multiple tools
	toolNames := []string{"tool_a", "tool_b", "tool_c"}
	for _, name := range toolNames {
		registry.Register(&tools.Tool{
			Name:        name,
			Description: "Test tool " + name,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
				return &tools.ToolResult{Content: "ok"}, nil
			},
		})
	}

	// Verify all tools are listed
	schemas := registry.ToAPISchema()
	if len(schemas) != 3 {
		t.Errorf("expected 3 tool schemas, got %d", len(schemas))
	}

	// Verify all names present
	schemaNames := make(map[string]bool)
	for _, schema := range schemas {
		name, ok := schema["name"].(string)
		if !ok {
			t.Errorf("schema name not found or not a string")
			continue
		}
		schemaNames[name] = true
	}

	for _, expected := range toolNames {
		if !schemaNames[expected] {
			t.Errorf("expected tool %s not found in schemas", expected)
		}
	}
}

// TestToolFlow_EmptyRegistry verifies empty registry handling
func TestToolFlow_EmptyRegistry(t *testing.T) {
	registry := tools.NewRegistry()

	// Empty registry should return empty slice
	schemas := registry.ToAPISchema()
	if schemas == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}

	// List should also be empty
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected 0 tool names, got %d", len(names))
	}
}

// TestToolFlow_UnknownToolDispatch verifies dispatch with unknown tool
func TestToolFlow_UnknownToolDispatch(t *testing.T) {
	registry := tools.NewRegistry()

	ctx := context.Background()
	_, err := registry.Dispatch(ctx, "unknown_tool", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for unknown tool, got nil")
	}
}
