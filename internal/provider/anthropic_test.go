//go:build js && wasm

package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAnthropicRequestWithTools verifies that Tools are properly serialized in requests
func TestAnthropicRequestWithTools(t *testing.T) {
	tools := []map[string]interface{}{
		{
			"name":        "web_search",
			"description": "Search the web for information",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	req := anthropicRequest{
		Model:     "claude-sonnet-4-5",
		Messages:  []anthropicMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 4096,
		Tools:     tools,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Verify tools are in the JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if _, ok := result["tools"]; !ok {
		t.Error("expected 'tools' field in request")
	}

	toolsList, ok := result["tools"].([]interface{})
	if !ok || len(toolsList) != 1 {
		t.Errorf("expected 1 tool, got %d", len(toolsList))
	}
}

// TestAnthropicStreamEventParsing verifies SSE event parsing
func TestAnthropicStreamEventParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected anthropicStreamEvent
	}{
		{
			name: "content_block_start with tool_use",
			json: `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool_123","name":"web_search"}}`,
			expected: anthropicStreamEvent{
				Type:  "content_block_start",
				Index: 1,
				ContentBlock: &anthropicContentBlock{
					Type: "tool_use",
					ID:   "tool_123",
					Name: "web_search",
				},
			},
		},
		{
			name: "content_block_delta with text_delta",
			json: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			expected: anthropicStreamEvent{
				Type:  "content_block_delta",
				Index: 0,
				Delta: &anthropicDelta{
					Type: "text_delta",
					Text: "Hello",
				},
			},
		},
		{
			name: "content_block_delta with input_json_delta",
			json: `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"test\"}"}}`,
			expected: anthropicStreamEvent{
				Type:  "content_block_delta",
				Index: 1,
				Delta: &anthropicDelta{
					Type:        "input_json_delta",
					PartialJSON: `{"query":"test"}`,
				},
			},
		},
		{
			name: "message_stop",
			json: `{"type":"message_stop"}`,
			expected: anthropicStreamEvent{
				Type: "message_stop",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(tt.json), &event); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if event.Type != tt.expected.Type {
				t.Errorf("Type = %v, want %v", event.Type, tt.expected.Type)
			}
			if event.Index != tt.expected.Index {
				t.Errorf("Index = %v, want %v", event.Index, tt.expected.Index)
			}

			if tt.expected.ContentBlock != nil {
				if event.ContentBlock == nil {
					t.Fatal("ContentBlock is nil")
				}
				if event.ContentBlock.Type != tt.expected.ContentBlock.Type {
					t.Errorf("ContentBlock.Type = %v, want %v", event.ContentBlock.Type, tt.expected.ContentBlock.Type)
				}
				if event.ContentBlock.ID != tt.expected.ContentBlock.ID {
					t.Errorf("ContentBlock.ID = %v, want %v", event.ContentBlock.ID, tt.expected.ContentBlock.ID)
				}
				if event.ContentBlock.Name != tt.expected.ContentBlock.Name {
					t.Errorf("ContentBlock.Name = %v, want %v", event.ContentBlock.Name, tt.expected.ContentBlock.Name)
				}
			}

			if tt.expected.Delta != nil {
				if event.Delta == nil {
					t.Fatal("Delta is nil")
				}
				if event.Delta.Type != tt.expected.Delta.Type {
					t.Errorf("Delta.Type = %v, want %v", event.Delta.Type, tt.expected.Delta.Type)
				}
				if event.Delta.Text != tt.expected.Delta.Text {
					t.Errorf("Delta.Text = %v, want %v", event.Delta.Text, tt.expected.Delta.Text)
				}
				if event.Delta.PartialJSON != tt.expected.Delta.PartialJSON {
					t.Errorf("Delta.PartialJSON = %v, want %v", event.Delta.PartialJSON, tt.expected.Delta.PartialJSON)
				}
			}
		})
	}
}

// TestAnthropicToolInputAccumulation verifies JSON accumulation from partial_json
func TestAnthropicToolInputAccumulation(t *testing.T) {
	// Simulate accumulating JSON from partial fragments
	fragments := []string{
		`{"`,
		`query`,
		`":"`,
		`test`,
		`"}`,
	}

	var builder strings.Builder
	for _, frag := range fragments {
		builder.WriteString(frag)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(builder.String()), &result); err != nil {
		t.Fatalf("failed to parse accumulated JSON: %v", err)
	}

	if query, ok := result["query"].(string); !ok || query != "test" {
		t.Errorf("expected query='test', got %v", result["query"])
	}
}

// TestTokenToolFields verifies Token struct has correct tool fields
func TestTokenToolFields(t *testing.T) {
	token := Token{
		FinishReason: "tool_use",
		ToolName:     "web_search",
		ToolInput: map[string]interface{}{
			"query": "golang testing",
		},
		ToolUseID: "tool_abc123",
	}

	if token.FinishReason != "tool_use" {
		t.Errorf("FinishReason = %v, want tool_use", token.FinishReason)
	}
	if token.ToolName != "web_search" {
		t.Errorf("ToolName = %v, want web_search", token.ToolName)
	}
	if token.ToolUseID != "tool_abc123" {
		t.Errorf("ToolUseID = %v, want tool_abc123", token.ToolUseID)
	}
	if token.ToolInput == nil {
		t.Error("ToolInput is nil")
	}
}
