//go:build js && wasm

package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestOpenAIRequestWithTools verifies that Tools are properly serialized in requests
func TestOpenAIRequestWithTools(t *testing.T) {
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

	req := openAIRequest{
		Model:     "gpt-4",
		Messages:  []openAIMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 4096,
		Tools:     convertToOpenAITools(tools),
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

	// Verify tool structure
	tool := toolsList[0].(map[string]interface{})
	if tool["type"] != "function" {
		t.Errorf("expected tool type='function', got %v", tool["type"])
	}

	fn, ok := tool["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function object")
	}
	if fn["name"] != "web_search" {
		t.Errorf("expected function.name='web_search', got %v", fn["name"])
	}
}

// TestConvertToOpenAITools verifies tool conversion
func TestConvertToOpenAITools(t *testing.T) {
	tests := []struct {
		name  string
		tools []map[string]interface{}
		want  int
	}{
		{
			name:  "empty tools",
			tools: nil,
			want:  0,
		},
		{
			name:  "empty slice",
			tools: []map[string]interface{}{},
			want:  0,
		},
		{
			name: "single tool",
			tools: []map[string]interface{}{
				{
					"name":        "test",
					"description": "Test tool",
					"input_schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
			want: 1,
		},
		{
			name: "multiple tools",
			tools: []map[string]interface{}{
				{
					"name":        "tool1",
					"description": "First tool",
					"input_schema": map[string]interface{}{
						"type": "object",
					},
				},
				{
					"name":        "tool2",
					"description": "Second tool",
					"input_schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToOpenAITools(tt.tools)
			if len(result) != tt.want {
				t.Errorf("got %d tools, want %d", len(result), tt.want)
			}
		})
	}
}

// TestGetStringAndGetMap verifies helper functions
func TestGetStringAndGetMap(t *testing.T) {
	m := map[string]interface{}{
		"name":         "test",
		"description":  "A test tool",
		"input_schema": map[string]interface{}{"type": "object"},
		"extra":        123, // not a string
	}

	if got := getString(m, "name"); got != "test" {
		t.Errorf("getString(name) = %v, want test", got)
	}
	if got := getString(m, "description"); got != "A test tool" {
		t.Errorf("getString(description) = %v, want 'A test tool'", got)
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString(missing) = %v, want empty", got)
	}
	if got := getString(m, "extra"); got != "" {
		t.Errorf("getString(extra) = %v, want empty (not a string)", got)
	}

	schema := getMap(m, "input_schema")
	if schema == nil {
		t.Error("getMap(input_schema) is nil")
	} else if schema["type"] != "object" {
		t.Errorf("schema[type] = %v, want object", schema["type"])
	}

	if got := getMap(m, "name"); got != nil {
		t.Errorf("getMap(name) should be nil (not a map), got %v", got)
	}
}

// TestOpenAIStreamChoiceParsing verifies streaming choice parsing
func TestOpenAIStreamChoiceParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected openAIStreamChoice
	}{
		{
			name: "content delta",
			json: `{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}`,
			expected: openAIStreamChoice{
				Index: 0,
				Delta: openAIMessage{
					Role:    "assistant",
					Content: "Hello",
				},
			},
		},
		{
			name: "tool_calls delta with index 0",
			json: `{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"web_search","arguments":""}}]}}`,
			expected: openAIStreamChoice{
				Index: 0,
				Delta: openAIMessage{
					ToolCalls: []openAIToolCall{
						{
							Index: 0,
							ID:    "call_123",
							Type:  "function",
							Function: openAIToolFunction{
								Name: "web_search",
							},
						},
					},
				},
			},
		},
		{
			name: "tool_calls delta with arguments",
			json: `{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\":\"test\"}"}}]}}`,
			expected: openAIStreamChoice{
				Index: 0,
				Delta: openAIMessage{
					ToolCalls: []openAIToolCall{
						{
							Index: 0,
							Function: openAIToolFunction{
								Arguments: `{"query":"test"}`,
							},
						},
					},
				},
			},
		},
		{
			name: "finish with tool_calls",
			json: `{"index":0,"delta":{},"finish_reason":"tool_calls"}`,
			expected: openAIStreamChoice{
				Index:        0,
				Delta:        openAIMessage{},
				FinishReason: "tool_calls",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var choice openAIStreamChoice
			if err := json.Unmarshal([]byte(tt.json), &choice); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if choice.Index != tt.expected.Index {
				t.Errorf("Index = %v, want %v", choice.Index, tt.expected.Index)
			}
			if choice.FinishReason != tt.expected.FinishReason {
				t.Errorf("FinishReason = %v, want %v", choice.FinishReason, tt.expected.FinishReason)
			}
			if choice.Delta.Content != tt.expected.Delta.Content {
				t.Errorf("Delta.Content = %v, want %v", choice.Delta.Content, tt.expected.Delta.Content)
			}

			if len(tt.expected.Delta.ToolCalls) > 0 {
				if len(choice.Delta.ToolCalls) != len(tt.expected.Delta.ToolCalls) {
					t.Fatalf("ToolCalls length = %v, want %v", len(choice.Delta.ToolCalls), len(tt.expected.Delta.ToolCalls))
				}
				for i, tc := range tt.expected.Delta.ToolCalls {
					if choice.Delta.ToolCalls[i].Index != tc.Index {
						t.Errorf("ToolCalls[%d].Index = %v, want %v", i, choice.Delta.ToolCalls[i].Index, tc.Index)
					}
					if choice.Delta.ToolCalls[i].ID != tc.ID {
						t.Errorf("ToolCalls[%d].ID = %v, want %v", i, choice.Delta.ToolCalls[i].ID, tc.ID)
					}
					if choice.Delta.ToolCalls[i].Function.Name != tc.Function.Name {
						t.Errorf("ToolCalls[%d].Function.Name = %v, want %v", i, choice.Delta.ToolCalls[i].Function.Name, tc.Function.Name)
					}
					if choice.Delta.ToolCalls[i].Function.Arguments != tc.Function.Arguments {
						t.Errorf("ToolCalls[%d].Function.Arguments = %v, want %v", i, choice.Delta.ToolCalls[i].Function.Arguments, tc.Function.Arguments)
					}
				}
			}
		})
	}
}

// TestOpenAIToolArgumentsAccumulation verifies JSON accumulation
func TestOpenAIToolArgumentsAccumulation(t *testing.T) {
	// Simulate accumulating JSON from partial fragments
	fragments := []string{
		`{`,
		`"query"`,
		`:`,
		`"golang testing"`,
		`}`,
	}

	var builder strings.Builder
	for _, frag := range fragments {
		builder.WriteString(frag)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(builder.String()), &result); err != nil {
		t.Fatalf("failed to parse accumulated JSON: %v", err)
	}

	if query, ok := result["query"].(string); !ok || query != "golang testing" {
		t.Errorf("expected query='golang testing', got %v", result["query"])
	}
}

// TestOpenAIToolInNonStreamingResponse verifies tool_calls in non-streaming response
func TestOpenAIToolInNonStreamingResponse(t *testing.T) {
	respJSON := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [
						{
							"id": "call_123",
							"type": "function",
							"function": {
								"name": "web_search",
								"arguments": "{\"query\":\"test\"}"
							}
						}
					]
				},
				"finish_reason": "tool_calls"
			}
		]
	}`

	var resp openAIResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}

	choice := resp.Choices[0]
	if choice.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %v, want tool_calls", choice.FinishReason)
	}

	if len(choice.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(choice.Message.ToolCalls))
	}

	tc := choice.Message.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("ToolCall.ID = %v, want call_123", tc.ID)
	}
	if tc.Function.Name != "web_search" {
		t.Errorf("ToolCall.Function.Name = %v, want web_search", tc.Function.Name)
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}
	if args["query"] != "test" {
		t.Errorf("arguments.query = %v, want test", args["query"])
	}
}
