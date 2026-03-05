//go:build js && wasm

// Package tools provides file operation tools using just-bash
// These tools enable browser-only file operations without requiring a local bridge binary
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// NewFileReadTool creates a tool for reading file contents
func NewFileReadTool() *Tool {
	return &Tool{
		Name:        "file_read",
		Description: "Read the contents of a file from the workspace. Use this to view code, configuration files, or any text files. Can read multiple files at once by providing an array of paths.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read (relative to workspace root)",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (0-indexed, optional)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read (optional, default: all)",
				},
			},
			"required": []string{"path"},
		},
		Execute: executeFileRead,
	}
}

func executeFileRead(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: path parameter is required",
			IsError:        true,
			ToolName:       "file_read",
			Status:         "error",
		}, nil
	}

	// Check if just-bash is available
	if !jsbridge.IsJustBashReady() {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: File system not initialized. Please wait for just-bash to load.",
			IsError:        true,
			ToolName:       "file_read",
			Status:         "error",
		}, nil
	}

	// Read the file
	content, err := jsbridge.JustBashReadFile(path)
	if err != nil {
		return &ToolResult{
			Content:        fmt.Sprintf("Error reading file: %v", err),
			DisplayContent: fmt.Sprintf("❌ Failed to read %s", path),
			IsError:        true,
			ToolName:       "file_read",
			Status:         "error",
		}, nil
	}

	// Handle offset and limit if provided
	offset := 0
	if offsetParam, ok := params["offset"].(float64); ok {
		offset = int(offsetParam)
	}

	limit := 0
	if limitParam, ok := params["limit"].(float64); ok {
		limit = int(limitParam)
	}

	// Apply offset and limit
	lines := strings.Split(content, "\n")
	if offset > 0 && offset < len(lines) {
		lines = lines[offset:]
	}
	if limit > 0 && limit < len(lines) {
		lines = lines[:limit]
	}

	resultContent := strings.Join(lines, "\n")

	// Format for LLM
	llmContent := fmt.Sprintf("File: %s\n```\n%s\n```", path, resultContent)

	return &ToolResult{
		Content:        llmContent,
		DisplayContent: fmt.Sprintf("📄 Read %s (%d bytes)", path, len(content)),
		IsError:        false,
		ToolName:       "file_read",
		Status:         "done",
	}, nil
}

// NewFileWriteTool creates a tool for writing files
func NewFileWriteTool() *Tool {
	return &Tool{
		Name:        "file_write",
		Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Use this to create new files or update existing ones. Parent directories are created automatically.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to write (relative to workspace root)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to the file",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, append to existing content instead of overwriting",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: executeFileWrite,
	}
}

func executeFileWrite(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: path parameter is required",
			IsError:        true,
			ToolName:       "file_write",
			Status:         "error",
		}, nil
	}

	content, ok := params["content"].(string)
	if !ok {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: content parameter is required",
			IsError:        true,
			ToolName:       "file_write",
			Status:         "error",
		}, nil
	}

	// Check if just-bash is available
	if !jsbridge.IsJustBashReady() {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: File system not initialized",
			IsError:        true,
			ToolName:       "file_write",
			Status:         "error",
		}, nil
	}

	// Handle append mode
	append, _ := params["append"].(bool)
	if append {
		existingContent, err := jsbridge.JustBashReadFile(path)
		if err == nil {
			content = existingContent + content
		}
	}

	// Write the file
	err := jsbridge.JustBashWriteFile(path, content)
	if err != nil {
		return &ToolResult{
			Content:        fmt.Sprintf("Error writing file: %v", err),
			DisplayContent: fmt.Sprintf("❌ Failed to write %s", path),
			IsError:        true,
			ToolName:       "file_write",
			Status:         "error",
		}, nil
	}

	return &ToolResult{
		Content:        fmt.Sprintf("File written successfully: %s (%d bytes)", path, len(content)),
		DisplayContent: fmt.Sprintf("✍️ Wrote %s (%d bytes)", path, len(content)),
		IsError:        false,
		ToolName:       "file_write",
		Status:         "done",
	}, nil
}

// NewDirListTool creates a tool for listing directory contents
func NewDirListTool() *Tool {
	return &Tool{
		Name:        "dir_list",
		Description: "List files and directories in a given path. Shows file sizes, permissions, and modification dates. Use this to explore the workspace structure or find specific files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list (default: current directory)",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "List recursively (include subdirectories)",
				},
			},
		},
		Execute: executeDirList,
	}
}

func executeDirList(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	path := "."
	if pathParam, ok := params["path"].(string); ok && pathParam != "" {
		path = pathParam
	}

	recursive, _ := params["recursive"].(bool)

	// Check if just-bash is available
	if !jsbridge.IsJustBashReady() {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: File system not initialized",
			IsError:        true,
			ToolName:       "dir_list",
			Status:         "error",
		}, nil
	}

	// List directory contents
	entries, err := jsbridge.JustBashListDir(path, false, true)
	if err != nil {
		return &ToolResult{
			Content:        fmt.Sprintf("Error listing directory: %v", err),
			DisplayContent: fmt.Sprintf("❌ Failed to list %s", path),
			IsError:        true,
			ToolName:       "dir_list",
			Status:         "error",
		}, nil
	}

	// Format entries for LLM
	var resultLines []string
	resultLines = append(resultLines, fmt.Sprintf("Directory: %s", path))
	resultLines = append(resultLines, "")

	for _, entry := range entries {
		fileType := "📄"
		if entry.IsDirectory {
			fileType = "📁"
		}
		resultLines = append(resultLines, fmt.Sprintf("%s %s %10d %s %s",
			fileType, entry.Permissions, entry.Size, entry.Date, entry.Name))
	}

	llmContent := strings.Join(resultLines, "\n")

	// Handle recursive listing
	if recursive {
		for _, entry := range entries {
			if entry.IsDirectory && entry.Name != "." && entry.Name != ".." {
				subResult, _ := executeDirList(ctx, map[string]interface{}{
					"path":      path + "/" + entry.Name,
					"recursive": true,
				})
				if subResult != nil && !subResult.IsError {
					llmContent += "\n\n" + subResult.Content
				}
			}
		}
	}

	return &ToolResult{
		Content:        llmContent,
		DisplayContent: fmt.Sprintf("📂 Listed %s (%d items)", path, len(entries)),
		IsError:        false,
		ToolName:       "dir_list",
		Status:         "done",
	}, nil
}

// NewFileSearchTool creates a tool for searching file contents
func NewFileSearchTool() *Tool {
	return &Tool{
		Name:        "file_search",
		Description: "Search for text patterns in files using grep. Supports regular expressions. Returns matching lines with file names and line numbers. Use this to find code, TODOs, or specific patterns across the workspace.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Search pattern (regex supported)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to search (file or directory, default: current directory)",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Search recursively in subdirectories",
				},
				"ignore_case": map[string]interface{}{
					"type":        "boolean",
					"description": "Case-insensitive search",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: executeFileSearch,
	}
}

func executeFileSearch(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: pattern parameter is required",
			IsError:        true,
			ToolName:       "file_search",
			Status:         "error",
		}, nil
	}

	path := "."
	if pathParam, ok := params["path"].(string); ok && pathParam != "" {
		path = pathParam
	}

	recursive, _ := params["recursive"].(bool)
	ignoreCase, _ := params["ignore_case"].(bool)

	// Check if just-bash is available
	if !jsbridge.IsJustBashReady() {
		return &ToolResult{
			Content:        "",
			DisplayContent: "Error: File system not initialized",
			IsError:        true,
			ToolName:       "file_search",
			Status:         "error",
		}, nil
	}

	// Search files
	matches, err := jsbridge.JustBashSearchFiles(pattern, path, recursive, ignoreCase)
	if err != nil {
		return &ToolResult{
			Content:        fmt.Sprintf("Error searching files: %v", err),
			DisplayContent: fmt.Sprintf("❌ Search failed: %s", pattern),
			IsError:        true,
			ToolName:       "file_search",
			Status:         "error",
		}, nil
	}

	if len(matches) == 0 {
		return &ToolResult{
			Content:        fmt.Sprintf("No matches found for pattern: %s", pattern),
			DisplayContent: fmt.Sprintf("🔍 No matches for '%s'", pattern),
			IsError:        false,
			ToolName:       "file_search",
			Status:         "done",
		}, nil
	}

	// Format matches for LLM
	var resultLines []string
	resultLines = append(resultLines, fmt.Sprintf("Search results for: %s", pattern))
	resultLines = append(resultLines, fmt.Sprintf("Found %d matches:", len(matches)))
	resultLines = append(resultLines, "")

	for _, match := range matches {
		resultLines = append(resultLines, fmt.Sprintf("%s:%d: %s", match.File, match.Line, match.Text))
	}

	llmContent := strings.Join(resultLines, "\n")

	return &ToolResult{
		Content:        llmContent,
		DisplayContent: fmt.Sprintf("🔍 Found %d matches for '%s'", len(matches), pattern),
		IsError:        false,
		ToolName:       "file_search",
		Status:         "done",
	}, nil
}

// RegisterJustBashFileTools registers all just-bash file tools with the registry
func RegisterJustBashFileTools(registry *Registry) {
	registry.Register(NewFileReadTool())
	registry.Register(NewFileWriteTool())
	registry.Register(NewDirListTool())
	registry.Register(NewFileSearchTool())
}
