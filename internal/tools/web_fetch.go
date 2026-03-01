//go:build js && wasm

package tools

import (
	"context"
	"fmt"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// NewWebFetchTool creates a tool that fetches the content of a URL via jsbridge.
// Never uses net/http — all HTTP calls go through the jsbridge.Fetch() bridge.
func NewWebFetchTool() *Tool {
	return &Tool{
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
		Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			url, _ := params["url"].(string)
			if url == "" {
				return &ToolResult{
					Content:        "url parameter is required",
					DisplayContent: "fetch failed: url parameter is required",
					IsError:        true,
					ToolName:       "web_fetch",
					Status:         "error",
				}, nil
			}

			resp, err := jsbridge.Fetch(url, jsbridge.FetchOptions{Method: "GET"})
			if err != nil {
				return &ToolResult{
					Content:        err.Error(),
					DisplayContent: "fetch failed: " + err.Error(),
					IsError:        true,
					ToolName:       "web_fetch",
					Status:         "error",
				}, nil
			}

			body := string(resp.Body)

			// Build display summary: truncate to 200 chars
			summary := body
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
			displayContent := fmt.Sprintf("HTTP %d — %s", resp.Status, summary)

			return &ToolResult{
				Content:        body,
				DisplayContent: displayContent,
				IsError:        false,
				ToolName:       "web_fetch",
				Status:         "done",
			}, nil
		},
	}
}
