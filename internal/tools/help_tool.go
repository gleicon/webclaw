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
	helpText.WriteString("# 🛠️ Available Tools\n\n")
	helpText.WriteString(fmt.Sprintf("WebClaw has **%d tools** available to help you:\n\n", len(tools)))

	helpText.WriteString("## Quick Start\n\n")
	helpText.WriteString("🆕 **New to WebClaw?** Here are some things you can try:\n\n")
	helpText.WriteString("- **Send a tweet:** `twitter_post text=\"Hello world!\"`\n")
	helpText.WriteString("- **Check your inbox:** `gmail_list count=5`\n")
	helpText.WriteString("- **Search the web:** `web_search query=\"latest AI news\"`\n")
	helpText.WriteString("- **Manage files:** `dir_list path=\"/workspace\"`\n")
	helpText.WriteString("- **Create a calendar event:** `calendar_create title=\"Meeting\" start_time=\"tomorrow 2pm\"`\n\n")
	helpText.WriteString("📋 **Note:** Some tools require connecting accounts in Settings first.\n\n")

	// Group tools by category
	categories := map[string][]string{
		"🌐 Web & Search":     {},
		"📁 File Operations":  {},
		"💭 Memory":           {},
		"🐦 Social Media":     {},
		"📧 Email & Calendar": {},
		"🔧 Developer Tools":  {},
		"📝 Knowledge Base":   {},
		"⚙️ System":          {},
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
	helpText.WriteString("```\nhelp tool=\"<tool_name>\" verbose=true\n```\n\n")
	helpText.WriteString("## Full Documentation\n\n")
	helpText.WriteString("For complete documentation with examples, open the help page:\n\n")
	helpText.WriteString("**Click the 'View Full Docs' button in the welcome message, or open `static/help.html` in your browser.**\n\n")
	helpText.WriteString("The documentation includes:\n")
	helpText.WriteString("- Complete tool reference\n")
	helpText.WriteString("- OAuth setup instructions\n")
	helpText.WriteString("- Usage examples for all 23+ tools\n")
	helpText.WriteString("- Search syntax guides\n\n")

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
		return "🌐 Web & Search"
	case "file_read", "file_write", "dir_list", "file_search":
		return "📁 File Operations"
	case "memory_store", "memory_search":
		return "💭 Memory"
	// Social Media
	case "twitter_post", "twitter_reply", "twitter_search", "twitter_timeline":
		return "🐦 Social Media"
	// Email & Calendar
	case "gmail_send", "gmail_list", "gmail_read", "gmail_search":
		return "📧 Email & Calendar"
	case "calendar_list", "calendar_create", "calendar_delete", "calendar_today":
		return "📧 Email & Calendar"
	// Developer Tools
	case "github_list_issues", "github_list_prs", "github_create_issue", "github_search_code", "github_comment":
		return "🔧 Developer Tools"
	// Knowledge Base
	case "notion_list_databases", "notion_query", "notion_read", "notion_update", "notion_search":
		return "📝 Knowledge Base"
	case "help":
		return "⚙️ System"
	default:
		return "⚙️ System"
	}
}

func getUsageExample(toolName string) string {
	examples := map[string]string{
		// Web & Search
		"web_fetch":  `web_fetch url="https://example.com"`,
		"web_search": `web_search query="latest AI developments"`,
		// File Operations
		"file_read":   `file_read path="/workspace/main.go"`,
		"file_write":  `file_write path="/workspace/README.md" content="# My Project"`,
		"dir_list":    `dir_list path="/workspace" recursive=true`,
		"file_search": `file_search pattern="TODO" path="/workspace" recursive=true`,
		// Memory
		"memory_store":  `memory_store content="Important fact to remember" tags=["important"]`,
		"memory_search": `memory_search query="important fact" limit=5`,
		// Social Media
		"twitter_post":     `twitter_post text="Hello from WebClaw! 🤖"`,
		"twitter_reply":    `twitter_reply tweet_id="123456789" text="Great point!"`,
		"twitter_search":   `twitter_search query="#AI news" count=10`,
		"twitter_timeline": `twitter_timeline count=20`,
		// Email
		"gmail_send":   `gmail_send to="friend@example.com" subject="Hello" body="How are you?"`,
		"gmail_list":   `gmail_list count=10 label="INBOX"`,
		"gmail_read":   `gmail_read message_id="abc123"`,
		"gmail_search": `gmail_search query="from:boss@company.com subject:urgent" count=5`,
		// Calendar
		"calendar_list":   `calendar_list days=7 count=10`,
		"calendar_create": `calendar_create title="Team Meeting" start_time="2024-01-15T14:00:00Z" duration_minutes=60`,
		"calendar_delete": `calendar_delete event_id="event_123"`,
		"calendar_today":  `calendar_today`,
		// Developer Tools
		"github_list_issues":  `github_list_issues state="open" count=20`,
		"github_list_prs":     `github_list_prs owner="gleicon" repo="webclaw" state="open"`,
		"github_create_issue": `github_create_issue owner="gleicon" repo="webclaw" title="Bug: Login fails" body="Steps to reproduce..."`,
		"github_search_code":  `github_search_code query="repo:gleicon/webclaw TODO"`,
		"github_comment":      `github_comment owner="gleicon" repo="webclaw" number=42 body="LGTM! 🚀"`,
		// Knowledge Base
		"notion_list_databases": `notion_list_databases`,
		"notion_query":          `notion_query database_id="Tasks" filter_property="Status" filter_value="Not Started"`,
		"notion_read":           `notion_read page_id="abc123"`,
		"notion_update":         `notion_update page_id="abc123" properties={"Status": "Done"}`,
		"notion_search":         `notion_search query="project roadmap"`,
		// System
		"help": `help tool="twitter_post" verbose=true`,
	}

	if example, ok := examples[toolName]; ok {
		return fmt.Sprintf("```\n%s\n```\n", example)
	}
	return "See tool description for usage.\n"
}
