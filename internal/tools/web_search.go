//go:build js && wasm

package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// NewWebSearchTool creates a tool that searches the web using DuckDuckGo's HTML endpoint.
// Uses jsbridge.Fetch() — never net/http.
func NewWebSearchTool() *Tool {
	return &Tool{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo and return result snippets",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
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
					ToolName:       "web_search",
					Status:         "error",
				}, nil
			}

			encodedQuery := url.QueryEscape(query)
			searchURL := "https://html.duckduckgo.com/html/?q=" + encodedQuery

			resp, err := jsbridge.Fetch(searchURL, jsbridge.FetchOptions{
				Method: "GET",
				Headers: map[string]string{
					"User-Agent": "Mozilla/5.0 (compatible; webclaw/1.0)",
				},
			})
			if err != nil {
				return &ToolResult{
					Content:        "search failed: " + err.Error(),
					DisplayContent: "search failed: " + err.Error(),
					IsError:        true,
					ToolName:       "web_search",
					Status:         "error",
				}, nil
			}

			html := string(resp.Body)

			// Check for CAPTCHA or error responses
			if resp.Status != 200 {
				return &ToolResult{
					Content:        fmt.Sprintf("DuckDuckGo returned HTTP %d", resp.Status),
					DisplayContent: fmt.Sprintf("search error: HTTP %d", resp.Status),
					IsError:        true,
					ToolName:       "web_search",
					Status:         "error",
				}, nil
			}

			// Parse results conservatively using substring extraction
			// No html.Parse — not available in WASM stdlib without CGo
			results := extractDDGResults(html, 5)

			if len(results) == 0 {
				// Graceful degradation: CAPTCHA or parse failure
				return &ToolResult{
					Content:        "No results found (possible CAPTCHA or parse failure)",
					DisplayContent: fmt.Sprintf("Found 0 results for: %s", query),
					IsError:        false,
					ToolName:       "web_search",
					Status:         "done",
				}, nil
			}

			content := fmt.Sprintf("Search results for: %s\n\n", query)
			for i, r := range results {
				content += fmt.Sprintf("%d. %s\n%s\n\n", i+1, r.title, r.snippet)
			}

			return &ToolResult{
				Content:        content,
				DisplayContent: fmt.Sprintf("Found %d results for: %s", len(results), query),
				IsError:        false,
				ToolName:       "web_search",
				Status:         "done",
			}, nil
		},
	}
}

type searchResult struct {
	title   string
	snippet string
}

// extractDDGResults extracts result titles and snippets from DuckDuckGo HTML.
// Uses simple substring matching — no html.Parse.
func extractDDGResults(html string, maxResults int) []searchResult {
	var results []searchResult

	// Find result links with class="result__a"
	// DuckDuckGo HTML format: <a class="result__a" ...>TITLE</a>
	pos := 0
	for len(results) < maxResults {
		// Find next result__a anchor
		linkStart := strings.Index(html[pos:], `class="result__a"`)
		if linkStart < 0 {
			// Also try result__url or result-link variants
			linkStart = strings.Index(html[pos:], `class="result__url"`)
			if linkStart < 0 {
				break
			}
		}
		linkStart += pos

		// Find closing > for the opening tag
		tagEnd := strings.Index(html[linkStart:], ">")
		if tagEnd < 0 {
			break
		}
		tagEnd += linkStart + 1

		// Find closing </a>
		closeTag := strings.Index(html[tagEnd:], "</a>")
		if closeTag < 0 {
			break
		}

		title := strings.TrimSpace(html[tagEnd : tagEnd+closeTag])
		title = stripHTMLTags(title)

		// Find snippet near this location: look for result__snippet
		snippetSearch := html[tagEnd:]
		snippetStart := strings.Index(snippetSearch, `class="result__snippet"`)
		snippet := ""
		if snippetStart >= 0 {
			snippetTagEnd := strings.Index(snippetSearch[snippetStart:], ">")
			if snippetTagEnd >= 0 {
				snippetContentStart := snippetStart + snippetTagEnd + 1
				snippetClose := strings.Index(snippetSearch[snippetContentStart:], "</")
				if snippetClose >= 0 {
					snippet = strings.TrimSpace(snippetSearch[snippetContentStart : snippetContentStart+snippetClose])
					snippet = stripHTMLTags(snippet)
				}
			}
		}

		if title != "" {
			results = append(results, searchResult{title: title, snippet: snippet})
		}

		pos = tagEnd + closeTag + 4
		if pos >= len(html) {
			break
		}
	}

	return results
}

// stripHTMLTags removes simple HTML tags from a string.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			inTag = true
		} else if s[i] == '>' {
			inTag = false
		} else if !inTag {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}
