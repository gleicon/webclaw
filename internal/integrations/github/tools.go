//go:build js && wasm

package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gleicon/webclaw/internal/tools"
)

// NewListIssuesTool creates the github_list_issues tool
func NewListIssuesTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "github_list_issues",
		Description: "List GitHub issues (assigned to you or in a specific repository). Shows issue number, title, state, and URL.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (optional - if omitted, shows your assigned issues across all repos)",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (required if owner specified)",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"open", "closed", "all"},
					"description": "Filter by state (default: open)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Max issues to return (default 20, max 100)",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Check if GitHub is connected
			if !client.IsConnected() {
				return &tools.ToolResult{
					Content:        "Please connect GitHub in Settings first. Click the settings icon → Connected Services → Connect GitHub.",
					DisplayContent: "❌ GitHub not connected",
					IsError:        true,
					ToolName:       "github_list_issues",
					Status:         "error",
				}, nil
			}

			// Parse parameters
			owner := getStringParam(params, "owner")
			repo := getStringParam(params, "repo")
			state := getStringParam(params, "state")
			if state == "" {
				state = "open"
			}
			count := getIntParam(params, "count", 20)
			if count > 100 {
				count = 100
			}

			// Validate parameters
			if owner != "" && repo == "" {
				return &tools.ToolResult{
					Content:        "Error: If owner is specified, repo is required",
					DisplayContent: "❌ Missing repo parameter",
					IsError:        true,
					ToolName:       "github_list_issues",
					Status:         "error",
				}, nil
			}

			// Fetch issues
			var issues []*Issue
			var err error

			if owner != "" && repo != "" {
				// Repository-specific issues
				issues, err = client.GetIssues(state, "", nil, owner, repo, count)
			} else {
				// User's issues across all repos
				issues, err = client.GetIssues(state, "", nil, "", "", count)
			}

			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to fetch issues: %v", err),
					DisplayContent: "❌ Failed to fetch issues",
					IsError:        true,
					ToolName:       "github_list_issues",
					Status:         "error",
				}, nil
			}

			if len(issues) == 0 {
				return &tools.ToolResult{
					Content:        "No issues found matching your criteria.",
					DisplayContent: "✅ No issues found",
					IsError:        false,
					ToolName:       "github_list_issues",
					Status:         "done",
				}, nil
			}

			content := formatIssueList(issues)
			display := fmt.Sprintf("✅ Found %d issue(s)", len(issues))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "github_list_issues",
				Status:         "done",
			}, nil
		},
	}
}

// NewListPRsTool creates the github_list_prs tool
func NewListPRsTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "github_list_prs",
		Description: "List open pull requests in a repository. Shows PR number, title, author, and branch information.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (required)",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (required)",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"open", "closed", "all"},
					"description": "Filter by state (default: open)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Max PRs to return (default 20, max 100)",
				},
			},
			"required": []string{"owner", "repo"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			if !client.IsConnected() {
				return &tools.ToolResult{
					Content:        "Please connect GitHub in Settings first. Click the settings icon → Connected Services → Connect GitHub.",
					DisplayContent: "❌ GitHub not connected",
					IsError:        true,
					ToolName:       "github_list_prs",
					Status:         "error",
				}, nil
			}

			owner := getStringParam(params, "owner")
			repo := getStringParam(params, "repo")
			state := getStringParam(params, "state")
			if state == "" {
				state = "open"
			}
			count := getIntParam(params, "count", 20)
			if count > 100 {
				count = 100
			}

			if owner == "" || repo == "" {
				return &tools.ToolResult{
					Content:        "Error: Both owner and repo are required",
					DisplayContent: "❌ Missing owner or repo",
					IsError:        true,
					ToolName:       "github_list_prs",
					Status:         "error",
				}, nil
			}

			prs, err := client.GetPullRequests(owner, repo, state, count)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to fetch pull requests: %v", err),
					DisplayContent: "❌ Failed to fetch PRs",
					IsError:        true,
					ToolName:       "github_list_prs",
					Status:         "error",
				}, nil
			}

			if len(prs) == 0 {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("No %s pull requests found in %s/%s.", state, owner, repo),
					DisplayContent: fmt.Sprintf("✅ No %s PRs found", state),
					IsError:        false,
					ToolName:       "github_list_prs",
					Status:         "done",
				}, nil
			}

			content := formatPRList(prs)
			display := fmt.Sprintf("✅ Found %d PR(s) in %s/%s", len(prs), owner, repo)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "github_list_prs",
				Status:         "done",
			}, nil
		},
	}
}

// NewCreateIssueTool creates the github_create_issue tool
func NewCreateIssueTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "github_create_issue",
		Description: "Create a new GitHub issue in a repository. Returns the created issue URL.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Issue title",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Issue body/description (supports Markdown)",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Labels to apply (optional)",
				},
			},
			"required": []string{"owner", "repo", "title"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			if !client.IsConnected() {
				return &tools.ToolResult{
					Content:        "Please connect GitHub in Settings first. Click the settings icon → Connected Services → Connect GitHub.",
					DisplayContent: "❌ GitHub not connected",
					IsError:        true,
					ToolName:       "github_create_issue",
					Status:         "error",
				}, nil
			}

			owner := getStringParam(params, "owner")
			repo := getStringParam(params, "repo")
			title := getStringParam(params, "title")
			body := getStringParam(params, "body")
			labels := getStringSliceParam(params, "labels")

			if owner == "" || repo == "" || title == "" {
				return &tools.ToolResult{
					Content:        "Error: owner, repo, and title are required",
					DisplayContent: "❌ Missing required parameters",
					IsError:        true,
					ToolName:       "github_create_issue",
					Status:         "error",
				}, nil
			}

			issue, err := client.CreateIssue(owner, repo, title, body, labels)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to create issue: %v", err),
					DisplayContent: "❌ Failed to create issue",
					IsError:        true,
					ToolName:       "github_create_issue",
					Status:         "error",
				}, nil
			}

			content := fmt.Sprintf("Created issue #%d: %s\n\n%s", issue.Number, issue.Title, issue.HTMLURL)
			display := fmt.Sprintf("✅ Created issue #%d in %s/%s", issue.Number, owner, repo)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "github_create_issue",
				Status:         "done",
			}, nil
		},
	}
}

// NewSearchCodeTool creates the github_search_code tool
func NewSearchCodeTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "github_search_code",
		Description: "Search code across GitHub repositories. Supports advanced search syntax: repo:owner/name, language:go, extension:go, path:src/, and quoted phrases.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (GitHub code search syntax: repo:owner/name, language:go, extension:go, etc.)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Max results (default 10, max 100)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			if !client.IsConnected() {
				return &tools.ToolResult{
					Content:        "Please connect GitHub in Settings first. Click the settings icon → Connected Services → Connect GitHub.",
					DisplayContent: "❌ GitHub not connected",
					IsError:        true,
					ToolName:       "github_search_code",
					Status:         "error",
				}, nil
			}

			query := getStringParam(params, "query")
			count := getIntParam(params, "count", 10)
			if count > 100 {
				count = 100
			}

			if query == "" {
				return &tools.ToolResult{
					Content:        "Error: query is required",
					DisplayContent: "❌ Missing query parameter",
					IsError:        true,
					ToolName:       "github_search_code",
					Status:         "error",
				}, nil
			}

			result, err := client.SearchCode(query, count)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to search code: %v", err),
					DisplayContent: "❌ Search failed",
					IsError:        true,
					ToolName:       "github_search_code",
					Status:         "error",
				}, nil
			}

			if len(result.Items) == 0 {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("No results found for query: %s", query),
					DisplayContent: "✅ No results found",
					IsError:        false,
					ToolName:       "github_search_code",
					Status:         "done",
				}, nil
			}

			content := formatCodeSearchResult(result)
			display := fmt.Sprintf("✅ Found %d result(s) for '%s'", len(result.Items), query)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "github_search_code",
				Status:         "done",
			}, nil
		},
	}
}

// NewCommentTool creates the github_comment tool
func NewCommentTool(client *Client) *tools.Tool {
	return &tools.Tool{
		Name:        "github_comment",
		Description: "Add a comment to a GitHub issue or pull request. Works for both issues and PRs (since PRs are issues under the hood).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
				},
				"number": map[string]interface{}{
					"type":        "integer",
					"description": "Issue or PR number",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Comment body (supports Markdown)",
				},
			},
			"required": []string{"owner", "repo", "number", "body"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			if !client.IsConnected() {
				return &tools.ToolResult{
					Content:        "Please connect GitHub in Settings first. Click the settings icon → Connected Services → Connect GitHub.",
					DisplayContent: "❌ GitHub not connected",
					IsError:        true,
					ToolName:       "github_comment",
					Status:         "error",
				}, nil
			}

			owner := getStringParam(params, "owner")
			repo := getStringParam(params, "repo")
			number := getIntParam(params, "number", 0)
			body := getStringParam(params, "body")

			if owner == "" || repo == "" || number == 0 || body == "" {
				return &tools.ToolResult{
					Content:        "Error: owner, repo, number, and body are required",
					DisplayContent: "❌ Missing required parameters",
					IsError:        true,
					ToolName:       "github_comment",
					Status:         "error",
				}, nil
			}

			comment, err := client.CreateComment(owner, repo, number, body)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to add comment: %v", err),
					DisplayContent: "❌ Failed to add comment",
					IsError:        true,
					ToolName:       "github_comment",
					Status:         "error",
				}, nil
			}

			content := fmt.Sprintf("Added comment to #%d\n\n%s", number, comment.HTMLURL)
			display := fmt.Sprintf("✅ Comment added to #%d in %s/%s", number, owner, repo)

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "github_comment",
				Status:         "done",
			}, nil
		},
	}
}

// Helper functions for parameter extraction

func getStringParam(params map[string]interface{}, key string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultVal
}

func getStringSliceParam(params map[string]interface{}, key string) []string {
	if val, ok := params[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}

// Formatting helpers

func formatIssue(issue *Issue) string {
	assignees := ""
	if len(issue.Assignees) > 0 {
		names := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			names[i] = "@" + a.Login
		}
		assignees = " (assigned to " + strings.Join(names, ", ") + ")"
	}

	labels := ""
	if len(issue.Labels) > 0 {
		labelNames := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labelNames[i] = l.Name
		}
		labels = " [" + strings.Join(labelNames, ", ") + "]"
	}

	return fmt.Sprintf("#%d: %s (%s)%s%s\n   %s",
		issue.Number,
		issue.Title,
		issue.State,
		labels,
		assignees,
		issue.HTMLURL,
	)
}

func formatIssueList(issues []*Issue) string {
	if len(issues) == 0 {
		return "No issues found."
	}

	parts := make([]string, 0, len(issues)+1)
	parts = append(parts, fmt.Sprintf("Found %d issue(s):\n", len(issues)))

	for _, issue := range issues {
		parts = append(parts, formatIssue(issue))
	}

	return strings.Join(parts, "\n\n")
}

func formatPR(pr *PullRequest) string {
	draft := ""
	if pr.Draft {
		draft = " [DRAFT]"
	}

	branchInfo := ""
	if pr.Head != nil && pr.Base != nil {
		branchInfo = fmt.Sprintf("\n   %s → %s", pr.Head.Ref, pr.Base.Ref)
	}

	author := ""
	if pr.User != nil {
		author = " by @" + pr.User.Login
	}

	return fmt.Sprintf("#%d: %s (%s)%s%s%s\n   %s",
		pr.Number,
		pr.Title,
		pr.State,
		draft,
		author,
		branchInfo,
		pr.HTMLURL,
	)
}

func formatPRList(prs []*PullRequest) string {
	if len(prs) == 0 {
		return "No pull requests found."
	}

	parts := make([]string, 0, len(prs)+1)
	parts = append(parts, fmt.Sprintf("Found %d pull request(s):\n", len(prs)))

	for _, pr := range prs {
		parts = append(parts, formatPR(pr))
	}

	return strings.Join(parts, "\n\n")
}

func formatCodeSearchResult(result *CodeSearchResult) string {
	if len(result.Items) == 0 {
		return "No results found."
	}

	parts := make([]string, 0, len(result.Items)+2)
	parts = append(parts, fmt.Sprintf("Found %d result(s):\n", len(result.Items)))

	for i, item := range result.Items {
		repoName := "unknown"
		if item.Repository != nil {
			repoName = item.Repository.FullName
		}

		snippet := ""
		if len(item.TextMatches) > 0 && len(item.TextMatches[0].Fragment) > 0 {
			frag := item.TextMatches[0].Fragment
			if len(frag) > 200 {
				frag = frag[:200] + "..."
			}
			snippet = fmt.Sprintf("\n   ```\n   %s\n   ```", frag)
		}

		parts = append(parts, fmt.Sprintf("%d. **%s** in `%s`\n   Path: `%s`%s\n   %s",
			i+1,
			item.Name,
			repoName,
			item.Path,
			snippet,
			item.HTMLURL,
		))
	}

	return strings.Join(parts, "\n\n")
}
