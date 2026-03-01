//go:build js && wasm

package tools

import (
	"context"
	"fmt"
	"strings"
)

// MemoryLoopInterface defines the memory operations the tools need from the agent loop.
// Using an interface instead of *agent.AgentLoop avoids an import cycle
// (agent imports tools, tools cannot import agent).
type MemoryLoopInterface interface {
	StoreFact(content string, metadata map[string]interface{}) error
	SearchMemory(query string, limit int) ([]MemorySearchResultIface, error)
}

// MemorySearchResultIface is the minimal interface for memory search results.
// This avoids importing the memory package directly.
type MemorySearchResultIface interface {
	GetScore() float64
	GetContent() string
}

// NewMemoryStoreTool creates a tool that stores facts in the agent's memory.
func NewMemoryStoreTool(loop MemoryLoopInterface) *Tool {
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
					DisplayContent: "store failed: content parameter is required",
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
					Content:        "failed to store fact: " + err.Error(),
					DisplayContent: "store failed: " + err.Error(),
					IsError:        true,
					ToolName:       "memory_store",
					Status:         "error",
				}, nil
			}

			// Display first 100 chars of content
			displayContent := content
			if len(displayContent) > 100 {
				displayContent = displayContent[:100]
			}

			return &ToolResult{
				Content:        "Successfully stored: " + content,
				DisplayContent: "Stored: " + displayContent,
				IsError:        false,
				ToolName:       "memory_store",
				Status:         "done",
			}, nil
		},
	}
}

// NewMemorySearchTool creates a tool that searches the agent's memory.
func NewMemorySearchTool(loop MemoryLoopInterface) *Tool {
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
					DisplayContent: "search failed: query parameter is required",
					IsError:        true,
					ToolName:       "memory_search",
					Status:         "error",
				}, nil
			}

			// Extract optional limit (default 5)
			limit := 5
			if l, ok := params["limit"].(float64); ok && l > 0 {
				limit = int(l)
			} else if l, ok := params["limit"].(int); ok && l > 0 {
				limit = l
			}

			results, err := loop.SearchMemory(query, limit)
			if err != nil {
				return &ToolResult{
					Content:        "search failed: " + err.Error(),
					DisplayContent: "search failed: " + err.Error(),
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

			var sb strings.Builder
			for _, r := range results {
				sb.WriteString(fmt.Sprintf("Score: %.3f — %s\n", r.GetScore(), r.GetContent()))
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
