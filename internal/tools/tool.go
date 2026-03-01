//go:build js && wasm

package tools

import "context"

// ToolResult is the dual-output result from tool execution.
// Content is fed back to the LLM as the tool_result message.
// DisplayContent is shown in the UI tool activity panel (may be shorter or formatted differently).
type ToolResult struct {
	Content        string
	DisplayContent string
	IsError        bool
	ToolName       string
	Status         string // "running" | "done" | "error"
}

// Tool defines a single callable tool with its metadata and execute function.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{} // JSON Schema "object" type
	Execute     func(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}
