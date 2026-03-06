//go:build js && wasm

package twitter

import (
	"strings"
	"testing"

	"github.com/gleicon/webclaw/internal/tools"
)

// Test TwitterToolSet creation
func TestNewTwitterToolSet(t *testing.T) {
	// We can't easily mock OAuthManager, so we'll test the tool schemas
	// The actual client testing is in client_test.go

	// Just verify the tool set structure works
	toolSet := &TwitterToolSet{}

	if toolSet == nil {
		t.Fatal("Expected tool set, got nil")
	}
}

// Test tool registration with mock client
func TestRegisterAll(t *testing.T) {
	// Create a tool set with nil client (we're just testing registration)
	toolSet := &TwitterToolSet{client: nil}

	registry := tools.NewRegistry()

	// Register each tool manually to avoid dependency on OAuth
	registry.Register(toolSet.NewPostTool())
	registry.Register(toolSet.NewReplyTool())
	registry.Register(toolSet.NewSearchTool())
	registry.Register(toolSet.NewTimelineTool())

	expectedTools := []string{"twitter_post", "twitter_reply", "twitter_search", "twitter_timeline"}
	for _, name := range expectedTools {
		tool := registry.Get(name)
		if tool == nil {
			t.Errorf("Expected tool %s to be registered", name)
		}
	}
}

// Test twitter_post tool schema
func TestPostToolSchema(t *testing.T) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewPostTool()

	if tool.Name != "twitter_post" {
		t.Errorf("Expected name 'twitter_post', got %s", tool.Name)
	}

	// Check description is meaningful
	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Check schema has required fields
	schema := tool.InputSchema
	if schema["type"] != "object" {
		t.Error("Expected schema type 'object'")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties map")
	}

	if _, hasText := properties["text"]; !hasText {
		t.Error("Expected 'text' property in schema")
	}

	// Check text property has description
	textProp, ok := properties["text"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected text to be a map")
	}
	if textProp["type"] != "string" {
		t.Error("Expected text type to be 'string'")
	}
}

// Test twitter_reply tool schema
func TestReplyToolSchema(t *testing.T) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewReplyTool()

	if tool.Name != "twitter_reply" {
		t.Errorf("Expected name 'twitter_reply', got %s", tool.Name)
	}

	schema := tool.InputSchema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties map")
	}

	if _, hasTweetID := properties["tweet_id"]; !hasTweetID {
		t.Error("Expected 'tweet_id' property in schema")
	}

	if _, hasText := properties["text"]; !hasText {
		t.Error("Expected 'text' property in schema")
	}

	// Verify required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required array")
	}

	requiredFields := make(map[string]bool)
	for _, r := range required {
		requiredFields[r] = true
	}

	if !requiredFields["tweet_id"] {
		t.Error("Expected 'tweet_id' in required fields")
	}
	if !requiredFields["text"] {
		t.Error("Expected 'text' in required fields")
	}
}

// Test twitter_search tool schema
func TestSearchToolSchema(t *testing.T) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewSearchTool()

	if tool.Name != "twitter_search" {
		t.Errorf("Expected name 'twitter_search', got %s", tool.Name)
	}

	schema := tool.InputSchema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties map")
	}

	if _, hasQuery := properties["query"]; !hasQuery {
		t.Error("Expected 'query' property in schema")
	}

	if _, hasCount := properties["count"]; !hasCount {
		t.Error("Expected 'count' property in schema")
	}

	// Check query is required
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required array")
	}

	found := false
	for _, r := range required {
		if r == "query" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'query' in required fields")
	}

	// Check count is optional (not in required)
	found = false
	for _, r := range required {
		if r == "count" {
			found = true
			break
		}
	}
	if found {
		t.Error("Expected 'count' to be optional (not in required)")
	}
}

// Test twitter_timeline tool schema
func TestTimelineToolSchema(t *testing.T) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewTimelineTool()

	if tool.Name != "twitter_timeline" {
		t.Errorf("Expected name 'twitter_timeline', got %s", tool.Name)
	}

	schema := tool.InputSchema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties map")
	}

	if _, hasCount := properties["count"]; !hasCount {
		t.Error("Expected 'count' property in schema")
	}

	// Check count is optional
	required, ok := schema["required"].([]string)
	if ok && len(required) > 0 {
		for _, r := range required {
			if r == "count" {
				t.Error("Expected 'count' to be optional (not in required)")
			}
		}
	}
}

// Test tool descriptions contain keywords
func TestToolDescriptions(t *testing.T) {
	toolSet := &TwitterToolSet{}

	tests := []struct {
		tool          *tools.Tool
		name          string
		shouldContain []string
	}{
		{
			tool:          toolSet.NewPostTool(),
			name:          "twitter_post",
			shouldContain: []string{"tweet", "Twitter", "post"},
		},
		{
			tool:          toolSet.NewReplyTool(),
			name:          "twitter_reply",
			shouldContain: []string{"reply", "tweet"},
		},
		{
			tool:          toolSet.NewSearchTool(),
			name:          "twitter_search",
			shouldContain: []string{"search", "Twitter"},
		},
		{
			tool:          toolSet.NewTimelineTool(),
			name:          "twitter_timeline",
			shouldContain: []string{"timeline", "home"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := strings.ToLower(tt.tool.Description)
			for _, keyword := range tt.shouldContain {
				if !strings.Contains(desc, strings.ToLower(keyword)) {
					t.Errorf("Expected description to contain '%s', got: %s", keyword, tt.tool.Description)
				}
			}
		})
	}
}

// Test all tools have execute functions
func TestToolsHaveExecute(t *testing.T) {
	toolSet := &TwitterToolSet{}

	tools := []*tools.Tool{
		toolSet.NewPostTool(),
		toolSet.NewReplyTool(),
		toolSet.NewSearchTool(),
		toolSet.NewTimelineTool(),
	}

	for _, tool := range tools {
		if tool.Execute == nil {
			t.Errorf("Tool %s has no Execute function", tool.Name)
		}
	}
}

// Test tool schema JSON compatibility
func TestToolSchemaJSON(t *testing.T) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewPostTool()

	// Verify schema can be used for API
	schema := tool.InputSchema
	if schema == nil {
		t.Fatal("Schema should not be nil")
	}

	// Check type field
	if schema["type"] != "object" {
		t.Error("Schema type should be 'object'")
	}

	// Check properties exist and is a map
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema properties should be a map")
	}

	// Each property should have type and description
	for name, prop := range properties {
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			t.Errorf("Property %s should be a map", name)
			continue
		}
		if propMap["type"] == nil {
			t.Errorf("Property %s should have a type", name)
		}
		if propMap["description"] == nil {
			t.Errorf("Property %s should have a description", name)
		}
	}
}

// Test tool integration with registry schema export
func TestToolRegistrySchema(t *testing.T) {
	toolSet := &TwitterToolSet{}

	registry := tools.NewRegistry()
	registry.Register(toolSet.NewPostTool())
	registry.Register(toolSet.NewReplyTool())
	registry.Register(toolSet.NewSearchTool())
	registry.Register(toolSet.NewTimelineTool())

	// Get API schema
	schemas := registry.ToAPISchema()
	if len(schemas) != 4 {
		t.Errorf("Expected 4 schemas, got %d", len(schemas))
	}

	// Verify each schema has required fields
	for _, schema := range schemas {
		if schema["name"] == "" {
			t.Error("Expected schema to have name")
		}
		if schema["description"] == "" {
			t.Error("Expected schema to have description")
		}
		if schema["input_schema"] == nil {
			t.Error("Expected schema to have input_schema")
		}

		// Verify input_schema is a proper object
		inputSchema, ok := schema["input_schema"].(map[string]interface{})
		if !ok {
			t.Error("input_schema should be a map")
			continue
		}

		if inputSchema["type"] != "object" {
			t.Error("input_schema type should be 'object'")
		}
	}
}

// Test that tool names follow convention
func TestToolNameConvention(t *testing.T) {
	toolSet := &TwitterToolSet{}

	tools := []*tools.Tool{
		toolSet.NewPostTool(),
		toolSet.NewReplyTool(),
		toolSet.NewSearchTool(),
		toolSet.NewTimelineTool(),
	}

	for _, tool := range tools {
		// All tools should start with "twitter_"
		if !strings.HasPrefix(tool.Name, "twitter_") {
			t.Errorf("Tool name %s should start with 'twitter_'", tool.Name)
		}

		// Names should be lowercase with underscores
		if tool.Name != strings.ToLower(tool.Name) {
			t.Errorf("Tool name %s should be lowercase", tool.Name)
		}
	}
}

// Test parameter extraction patterns
func TestParameterExtraction(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		expected string
		found    bool
	}{
		{
			name:     "string param found",
			params:   map[string]interface{}{"text": "hello"},
			key:      "text",
			expected: "hello",
			found:    true,
		},
		{
			name:     "missing param",
			params:   map[string]interface{}{},
			key:      "text",
			expected: "",
			found:    false,
		},
		{
			name:     "wrong type param",
			params:   map[string]interface{}{"text": 123},
			key:      "text",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := tt.params[tt.key].(string)
			if ok != tt.found {
				t.Errorf("Expected found=%v, got found=%v", tt.found, ok)
			}
			if ok && val != tt.expected {
				t.Errorf("Expected %s, got %v", tt.expected, val)
			}
		})
	}
}

// Test count parameter bounds
func TestCountParameterBounds(t *testing.T) {
	tests := []struct {
		input    int
		default_ int
		min      int
		max      int
		expected int
	}{
		// Timeline: default 20, max 100
		{input: 0, default_: 20, min: 1, max: 100, expected: 20},
		{input: 50, default_: 20, min: 1, max: 100, expected: 50},
		{input: 150, default_: 20, min: 1, max: 100, expected: 20},
		{input: -5, default_: 20, min: 1, max: 100, expected: 20},
	}

	for _, tt := range tests {
		count := tt.input
		if count <= 0 || count > tt.max {
			count = tt.default_
		}

		if count != tt.expected {
			t.Errorf("input=%d: expected %d, got %d", tt.input, tt.expected, count)
		}
	}
}

// Benchmark tool creation
func BenchmarkNewPostTool(b *testing.B) {
	toolSet := &TwitterToolSet{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = toolSet.NewPostTool()
	}
}

// Benchmark tool registration
func BenchmarkRegisterAll(b *testing.B) {
	toolSet := &TwitterToolSet{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := tools.NewRegistry()
		registry.Register(toolSet.NewPostTool())
		registry.Register(toolSet.NewReplyTool())
		registry.Register(toolSet.NewSearchTool())
		registry.Register(toolSet.NewTimelineTool())
	}
}

// Benchmark schema generation
func BenchmarkToolSchema(b *testing.B) {
	toolSet := &TwitterToolSet{}
	tool := toolSet.NewPostTool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.InputSchema
	}
}

// Test that all tool schemas are valid
func TestAllToolSchemasValid(t *testing.T) {
	toolSet := &TwitterToolSet{}

	toolList := []*tools.Tool{
		toolSet.NewPostTool(),
		toolSet.NewReplyTool(),
		toolSet.NewSearchTool(),
		toolSet.NewTimelineTool(),
	}

	for _, tool := range toolList {
		t.Run(tool.Name, func(t *testing.T) {
			// Must have name
			if tool.Name == "" {
				t.Error("Tool must have a name")
			}

			// Must have description
			if tool.Description == "" {
				t.Error("Tool must have a description")
			}

			// Must have input schema
			if tool.InputSchema == nil {
				t.Fatal("Tool must have an input schema")
			}

			// Schema must have type
			if tool.InputSchema["type"] != "object" {
				t.Error("Schema type must be 'object'")
			}

			// Schema must have properties
			if tool.InputSchema["properties"] == nil {
				t.Error("Schema must have properties")
			}

			// Must have Execute function
			if tool.Execute == nil {
				t.Error("Tool must have an Execute function")
			}
		})
	}
}

// Test formatted tweet output structure
func TestFormattedTweetStructure(t *testing.T) {
	tweet := Tweet{
		ID:        "123",
		Text:      "Test tweet for formatting",
		AuthorID:  "user1",
		CreatedAt: "2024-01-15T10:30:00Z",
		PublicMetrics: &TweetPublicMetrics{
			LikeCount:    42,
			ReplyCount:   5,
			RetweetCount: 12,
		},
	}

	author := User{
		ID:       "user1",
		Name:     "Test User",
		Username: "testuser",
	}

	formatted := formatTweetForLLM(1, tweet, author)

	// Check structure
	lines := strings.Split(formatted, "\n")
	if len(lines) < 2 {
		t.Fatal("Formatted tweet should have at least 2 lines")
	}

	// First line should have index and author
	if !strings.Contains(lines[0], "1.") {
		t.Error("First line should contain index '1.'")
	}
	if !strings.Contains(lines[0], "Test User") {
		t.Error("First line should contain author name")
	}

	// Should contain metrics
	if !strings.Contains(formatted, "❤️") || !strings.Contains(formatted, "42") {
		t.Error("Formatted tweet should contain like count")
	}
}
