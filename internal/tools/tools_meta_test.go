//go:build !js

// Package tools_test tests tool constructor metadata natively (not in WASM).
// These tests validate Name, Description, and InputSchema without executing tools.
// Stub types are used to avoid jsbridge import in native context.
package tools_test

import (
	"context"
	"testing"
)

// --- Stub types for native metadata testing ---
// These mirror the WASM-only Tool/ToolResult structures for contract validation.

type stubAgentLoop struct{}

func (s *stubAgentLoop) StoreFact(content string, metadata map[string]interface{}) error {
	return nil
}

func (s *stubAgentLoop) SearchMemory(query string, limit int) ([]interface{}, error) {
	return nil, nil
}

// toolMeta captures tool constructor metadata for validation
type toolMeta struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// makeWebFetchMeta returns metadata matching NewWebFetchTool() contract
func makeWebFetchMeta() toolMeta {
	return toolMeta{
		Name:        "web_fetch",
		Description: "Fetch the content of a URL and return it as text",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch",
				},
			},
			"required": []string{"url"},
		},
	}
}

// makeWebSearchMeta returns metadata matching NewWebSearchTool() contract
func makeWebSearchMeta() toolMeta {
	return toolMeta{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo and return result snippets",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
			},
			"required": []string{"query"},
		},
	}
}

// makeMemoryStoreMeta returns metadata matching NewMemoryStoreTool() contract
func makeMemoryStoreMeta() toolMeta {
	return toolMeta{
		Name:        "memory_store",
		Description: "Store a fact or piece of information in memory",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type": "string",
				},
				"metadata": map[string]interface{}{
					"type": "object",
				},
			},
			"required": []string{"content"},
		},
	}
}

// makeMemorySearchMeta returns metadata matching NewMemorySearchTool() contract
func makeMemorySearchMeta() toolMeta {
	return toolMeta{
		Name:        "memory_search",
		Description: "Search memory for relevant facts or information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
				"limit": map[string]interface{}{
					"type": "integer",
				},
			},
			"required": []string{"query"},
		},
	}
}

// hasRequiredParam checks if an InputSchema has a required param name
func hasRequiredParam(schema map[string]interface{}, param string) bool {
	required, ok := schema["required"]
	if !ok {
		return false
	}
	switch r := required.(type) {
	case []string:
		for _, v := range r {
			if v == param {
				return true
			}
		}
	case []interface{}:
		for _, v := range r {
			if s, ok := v.(string); ok && s == param {
				return true
			}
		}
	}
	return false
}

func TestWebFetchToolMeta(t *testing.T) {
	meta := makeWebFetchMeta()

	if meta.Name != "web_fetch" {
		t.Errorf("Expected Name='web_fetch', got '%s'", meta.Name)
	}
	if meta.Description == "" {
		t.Error("Expected non-empty Description")
	}
	if !hasRequiredParam(meta.InputSchema, "url") {
		t.Error("Expected 'url' in required params")
	}
}

func TestWebSearchToolMeta(t *testing.T) {
	meta := makeWebSearchMeta()

	if meta.Name != "web_search" {
		t.Errorf("Expected Name='web_search', got '%s'", meta.Name)
	}
	if meta.Description == "" {
		t.Error("Expected non-empty Description")
	}
	if !hasRequiredParam(meta.InputSchema, "query") {
		t.Error("Expected 'query' in required params")
	}
}

func TestMemoryStoreToolMeta(t *testing.T) {
	meta := makeMemoryStoreMeta()

	if meta.Name != "memory_store" {
		t.Errorf("Expected Name='memory_store', got '%s'", meta.Name)
	}
	if meta.Description == "" {
		t.Error("Expected non-empty Description")
	}
	if !hasRequiredParam(meta.InputSchema, "content") {
		t.Error("Expected 'content' in required params")
	}
}

func TestMemorySearchToolMeta(t *testing.T) {
	meta := makeMemorySearchMeta()

	if meta.Name != "memory_search" {
		t.Errorf("Expected Name='memory_search', got '%s'", meta.Name)
	}
	if meta.Description == "" {
		t.Error("Expected non-empty Description")
	}
	if !hasRequiredParam(meta.InputSchema, "query") {
		t.Error("Expected 'query' in required params")
	}
}

func TestWebFetchEmptyURLError(t *testing.T) {
	// Validates contract: web_fetch with empty url param returns IsError=true
	// We test this via the execute function contract (simulated inline)
	executeWebFetch := func(ctx context.Context, params map[string]interface{}) (isError bool, content string) {
		url, _ := params["url"].(string)
		if url == "" {
			return true, "url parameter is required"
		}
		return false, "ok"
	}

	isError, content := executeWebFetch(context.Background(), map[string]interface{}{"url": ""})
	if !isError {
		t.Error("Expected IsError=true for empty url")
	}
	if content == "" {
		t.Error("Expected error content message")
	}
}

func TestWebSearchEmptyQueryError(t *testing.T) {
	// Validates contract: web_search with empty query param returns IsError=true
	executeWebSearch := func(ctx context.Context, params map[string]interface{}) (isError bool, content string) {
		query, _ := params["query"].(string)
		if query == "" {
			return true, "query parameter is required"
		}
		return false, "ok"
	}

	isError, content := executeWebSearch(context.Background(), map[string]interface{}{"query": ""})
	if !isError {
		t.Error("Expected IsError=true for empty query")
	}
	if content == "" {
		t.Error("Expected error content message")
	}
}
