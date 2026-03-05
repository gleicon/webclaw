//go:build js && wasm

package tools

import (
	"context"
	"fmt"
	"strings"
)

// NewHelpTool creates a tool that provides documentation and help for other tools
func NewHelpTool(registry *Registry) *Tool {
	return &Tool{
		Name:        "help",
		Description: "Get help and documentation for available tools. Use this to learn what tools are available and how to use them. Can list all tools or get detailed help for a specific tool.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tool to get help for. If not provided, lists all available tools.",
				},
				"verbose": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, show detailed information including full input schema (default: false)",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			return executeHelp(ctx, params, registry)
		},
	}
}

func executeHelp(ctx context.Context, params map[string]interface{}, registry *Registry) (*ToolResult, error) {
	toolName, hasSpecificTool := params["tool"].(string)
	verbose, _ := params["verbose"].(bool)

	if hasSpecificTool && toolName != "" {
		// Get help for specific tool
		tool := registry.Get(toolName)
		if tool == nil {
			availableTools := registry.List()
			return &ToolResult{
				Content:        fmt.Sprintf("Tool '%s' not found.\n\nAvailable tools: %s", toolName, strings.Join(availableTools, ", ")),
				DisplayContent: fmt.Sprintf("❓ Unknown tool: %s", toolName),
				IsError:        true,
				ToolName:       "help",
				Status:         "error",
			}, nil
		}

		// Format detailed help
		var helpText strings.Builder
		helpText.WriteString(fmt.Sprintf("# %s\n\n", tool.Name))
		helpText.WriteString(fmt.Sprintf("**Description:** %s\n\n", tool.Description))

		if verbose && tool.InputSchema != nil {
			helpText.WriteString("## Input Schema\n\n")
			helpText.WriteString(formatSchema(tool.InputSchema, 0))
			helpText.WriteString("\n")
		}

		// Add usage examples
		helpText.WriteString("## Usage Examples\n\n")
		helpText.WriteString(getUsageExample(tool.Name))

		content := helpText.String()
		return &ToolResult{
			Content:        content,
			DisplayContent: fmt.Sprintf("📖 Help: %s", tool.Name),
			IsError:        false,
			ToolName:       "help",
			Status:         "done",
		}, nil
	}

	// List all available tools
	tools := registry.GetAll()

	var helpText strings.Builder
	helpText.WriteString("# Available Tools\n\n")
	helpText.WriteString(fmt.Sprintf("WebClaw has %d tools available:\n\n", len(tools)))

	// Group tools by category
	categories := map[string][]string{
		"Web & Search":    {},
		"File Operations": {},
		"Memory":          {},
		"System":          {},
	}

	for name, tool := range tools {
		category := categorizeTool(name)
		categories[category] = append(categories[category], fmt.Sprintf("- **%s**: %s", name, tool.Description))
	}

	for category, items := range categories {
		if len(items) > 0 {
			helpText.WriteString(fmt.Sprintf("## %s\n\n", category))
			for _, item := range items {
				helpText.WriteString(item + "\n")
			}
			helpText.WriteString("\n")
		}
	}

	helpText.WriteString("## Getting Help\n\n")
	helpText.WriteString("To get detailed help for a specific tool, use:\n")
	helpText.WriteString("```\nhelp tool=\"<tool_name>\"\n```\n\n")
	helpText.WriteString("For verbose output with full schema:\n")
	helpText.WriteString("```\nhelp tool=\"<tool_name>\" verbose=true\n```\n")

	content := helpText.String()
	return &ToolResult{
		Content:        content,
		DisplayContent: fmt.Sprintf("📚 %d tools available", len(tools)),
		IsError:        false,
		ToolName:       "help",
		Status:         "done",
	}, nil
}

func formatSchema(schema map[string]interface{}, indent int) string {
	var result strings.Builder
	prefix := strings.Repeat("  ", indent)

	if schema["type"] == "object" && schema["properties"] != nil {
		result.WriteString(fmt.Sprintf("%sObject with properties:\n", prefix))
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			for propName, propSchema := range props {
				if propMap, ok := propSchema.(map[string]interface{}); ok {
					propType, _ := propMap["type"].(string)
					propDesc, _ := propMap["description"].(string)
					required := ""
					if req, ok := schema["required"].([]interface{}); ok {
						for _, r := range req {
							if r == propName {
								required = " (required)"
								break
							}
						}
					}
					result.WriteString(fmt.Sprintf("%s  - **%s** (%s)%s: %s\n", prefix, propName, propType, required, propDesc))
				}
			}
		}
	}

	return result.String()
}

func categorizeTool(name string) string {
	switch name {
	case "web_fetch", "web_search":
		return "Web & Search"
	case "file_read", "file_write", "dir_list", "file_search":
		return "File Operations"
	case "memory_store", "memory_search":
		return "Memory"
	case "help":
		return "System"
	default:
		return "System"
	}
}

func getUsageExample(toolName string) string {
	examples := map[string]string{
		"web_fetch":     `web_fetch url="https://example.com"`,
		"web_search":    `web_search query="latest AI developments"`,
		"file_read":     `file_read path="/workspace/main.go"`,
		"file_write":    `file_write path="/workspace/README.md" content="# My Project"`,
		"dir_list":      `dir_list path="/workspace" recursive=true`,
		"file_search":   `file_search pattern="TODO" path="/workspace" recursive=true`,
		"memory_store":  `memory_store content="Important fact to remember" tags=["important"]`,
		"memory_search": `memory_search query="important fact" limit=5`,
		"help":          `help tool="file_read" verbose=true`,
	}

	if example, ok := examples[toolName]; ok {
		return fmt.Sprintf("```\n%s\n```\n", example)
	}
	return "See tool description for usage.\n"
}
