//go:build js && wasm

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/memory"
)

// MemoryAgent is the interface that AgentLoop must satisfy for memory tool operations.
// Using an interface rather than *AgentLoop avoids a circular import between
// internal/tools and internal/agent.
type MemoryAgent interface {
	StoreFact(content string, metadata map[string]interface{}) error
	SearchMemory(query string, limit int) ([]*memory.MemorySearchResult, error)
}

// NewMemoryStoreTool creates a tool that stores a fact in memory.
// loop is the AgentLoop instance used for memory operations (injected at registration time).
func NewMemoryStoreTool(loop MemoryAgent) *Tool {
	return &Tool{
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
		Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			content, _ := params["content"].(string)
			if content == "" {
				return &ToolResult{
					Content:        "content parameter is required",
					DisplayContent: "memory store failed: content parameter is required",
					IsError:        true,
					ToolName:       "memory_store",
					Status:         "error",
				}, nil
			}

			// Extract optional metadata
			var metadata map[string]interface{}
			if m, ok := params["metadata"].(map[string]interface{}); ok {
				metadata = m
			}

			if err := loop.StoreFact(content, metadata); err != nil {
				return &ToolResult{
					Content:        "failed to store memory: " + err.Error(),
					DisplayContent: "memory store failed: " + err.Error(),
					IsError:        true,
					ToolName:       "memory_store",
					Status:         "error",
				}, nil
			}

			// Display first 100 chars of content
			preview := content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}

			return &ToolResult{
				Content:        "Stored: " + content,
				DisplayContent: "Stored: " + preview,
				IsError:        false,
				ToolName:       "memory_store",
				Status:         "done",
			}, nil
		},
	}
}

// NewMemorySearchTool creates a tool that searches memory for relevant facts.
// loop is the AgentLoop instance used for memory operations (injected at registration time).
func NewMemorySearchTool(loop MemoryAgent) *Tool {
	return &Tool{
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
		Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			query, _ := params["query"].(string)
			if query == "" {
				return &ToolResult{
					Content:        "query parameter is required",
					DisplayContent: "memory search failed: query parameter is required",
					IsError:        true,
					ToolName:       "memory_search",
					Status:         "error",
				}, nil
			}

			// Default limit is 5
			limit := 5
			if l, ok := params["limit"].(float64); ok && l > 0 {
				limit = int(l)
			}

			results, err := loop.SearchMemory(query, limit)
			if err != nil {
				return &ToolResult{
					Content:        "memory search failed: " + err.Error(),
					DisplayContent: "memory search failed: " + err.Error(),
					IsError:        true,
					ToolName:       "memory_search",
					Status:         "error",
				}, nil
			}

			if len(results) == 0 {
				return &ToolResult{
					Content:        fmt.Sprintf("No memories found for: %s", query),
					DisplayContent: fmt.Sprintf("Found 0 memories for: %s", query),
					IsError:        false,
					ToolName:       "memory_search",
					Status:         "done",
				}, nil
			}

			// Format results
			var sb strings.Builder
			for i, r := range results {
				sb.WriteString(fmt.Sprintf("Score: %.3f — %s\n", r.Score, r.Document.Content))
				if i < len(results)-1 {
					sb.WriteString("\n")
				}
			}

			return &ToolResult{
				Content:        sb.String(),
				DisplayContent: fmt.Sprintf("Found %d memories for: %s", len(results), query),
				IsError:        false,
				ToolName:       "memory_search",
				Status:         "done",
			}, nil
		},
	}
}
