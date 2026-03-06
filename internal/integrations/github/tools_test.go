//go:build js && wasm

package github

import (
	"strings"
	"testing"

	"github.com/gleicon/webclaw/internal/tools"
)

// MockClient is a mock GitHub client for testing
type MockClient struct {
	connected    bool
	issues       []*Issue
	prs          []*PullRequest
	issue        *Issue
	repo         *Repository
	searchResult *CodeSearchResult
	comment      *Comment
	err          error
}

func (m *MockClient) IsConnected() bool {
	return m.connected
}

func (m *MockClient) GetIssues(state, assignee string, labels []string, owner, repo string, perPage int) ([]*Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

func (m *MockClient) GetPullRequests(owner, repo, state string, perPage int) ([]*PullRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.prs, nil
}

func (m *MockClient) CreateIssue(owner, repo, title, body string, labels []string) (*Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issue, nil
}

func (m *MockClient) SearchCode(query string, perPage int) (*CodeSearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.searchResult, nil
}

func (m *MockClient) CreateComment(owner, repo string, issueNumber int, body string) (*Comment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.comment, nil
}

func TestNewListIssuesTool_NotConnected(t *testing.T) {
	// Create a mock client that's not connected
	mockClient := &Client{}
	tool := NewListIssuesTool(mockClient)

	// Verify tool properties
	if tool.Name != "github_list_issues" {
		t.Errorf("Expected tool name 'github_list_issues', got %s", tool.Name)
	}

	// The tool should have an execute function
	if tool.Execute == nil {
		t.Error("Expected Execute function to be defined")
	}
}

func TestNewListPRsTool_NotConnected(t *testing.T) {
	mockClient := &Client{}
	tool := NewListPRsTool(mockClient)

	if tool.Name != "github_list_prs" {
		t.Errorf("Expected tool name 'github_list_prs', got %s", tool.Name)
	}

	// Verify required fields are in schema
	schema, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected required field in schema")
	}

	requiredFound := map[string]bool{
		"owner": false,
		"repo":  false,
	}
	for _, field := range schema {
		if _, exists := requiredFound[field]; exists {
			requiredFound[field] = true
		}
	}

	for field, found := range requiredFound {
		if !found {
			t.Errorf("Expected %s to be in required fields", field)
		}
	}
}

func TestNewCreateIssueTool_NotConnected(t *testing.T) {
	mockClient := &Client{}
	tool := NewCreateIssueTool(mockClient)

	if tool.Name != "github_create_issue" {
		t.Errorf("Expected tool name 'github_create_issue', got %s", tool.Name)
	}

	// Verify required fields
	schema, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected required field in schema")
	}

	if len(schema) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(schema))
	}
}

func TestNewSearchCodeTool_NotConnected(t *testing.T) {
	mockClient := &Client{}
	tool := NewSearchCodeTool(mockClient)

	if tool.Name != "github_search_code" {
		t.Errorf("Expected tool name 'github_search_code', got %s", tool.Name)
	}

	// Verify query is required
	schema, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected required field in schema")
	}

	found := false
	for _, field := range schema {
		if field == "query" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'query' to be a required field")
	}
}

func TestNewCommentTool_NotConnected(t *testing.T) {
	mockClient := &Client{}
	tool := NewCommentTool(mockClient)

	if tool.Name != "github_comment" {
		t.Errorf("Expected tool name 'github_comment', got %s", tool.Name)
	}

	// Verify number is required
	schema, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected required field in schema")
	}

	if len(schema) != 4 {
		t.Errorf("Expected 4 required fields, got %d", len(schema))
	}
}

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"owner": "gleicon",
		"repo":  "webclaw",
		"count": 10,
	}

	if got := getStringParam(params, "owner"); got != "gleicon" {
		t.Errorf("getStringParam(owner) = %s, want gleicon", got)
	}
	if got := getStringParam(params, "repo"); got != "webclaw" {
		t.Errorf("getStringParam(repo) = %s, want webclaw", got)
	}
	if got := getStringParam(params, "missing"); got != "" {
		t.Errorf("getStringParam(missing) = %s, want empty", got)
	}
}

func TestGetIntParam(t *testing.T) {
	params := map[string]interface{}{
		"count_int":     20,
		"count_float":   30.0,
		"count_string":  "40",
		"count_missing": nil,
	}

	if got := getIntParam(params, "count_int", 10); got != 20 {
		t.Errorf("getIntParam(count_int) = %d, want 20", got)
	}
	if got := getIntParam(params, "count_float", 10); got != 30 {
		t.Errorf("getIntParam(count_float) = %d, want 30", got)
	}
	if got := getIntParam(params, "count_string", 10); got != 40 {
		t.Errorf("getIntParam(count_string) = %d, want 40", got)
	}
	if got := getIntParam(params, "missing", 10); got != 10 {
		t.Errorf("getIntParam(missing) = %d, want 10 (default)", got)
	}
}

func TestGetStringSliceParam(t *testing.T) {
	params := map[string]interface{}{
		"labels": []interface{}{"bug", "feature", "help wanted"},
		"empty":  []interface{}{},
		"single": "not an array",
	}

	labels := getStringSliceParam(params, "labels")
	if len(labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(labels))
	}
	if labels[0] != "bug" {
		t.Errorf("Expected first label to be 'bug', got %s", labels[0])
	}

	empty := getStringSliceParam(params, "empty")
	if len(empty) != 0 {
		t.Errorf("Expected 0 labels for empty, got %d", len(empty))
	}

	single := getStringSliceParam(params, "single")
	if len(single) != 0 {
		t.Errorf("Expected 0 labels for non-array, got %d", len(single))
	}
}

func TestFormatIssue(t *testing.T) {
	issue := &Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "open",
		User:   &User{Login: "author"},
		Assignees: []*User{
			{Login: "assignee1"},
			{Login: "assignee2"},
		},
		Labels: []*Label{
			{Name: "bug"},
			{Name: "help wanted"},
		},
		HTMLURL: "https://github.com/test/repo/issues/42",
	}

	formatted := formatIssue(issue)

	if !strings.Contains(formatted, "#42") {
		t.Error("Expected formatted issue to contain issue number")
	}
	if !strings.Contains(formatted, "Test Issue") {
		t.Error("Expected formatted issue to contain title")
	}
	if !strings.Contains(formatted, "@assignee1") {
		t.Error("Expected formatted issue to contain assignee")
	}
	if !strings.Contains(formatted, "[bug") {
		t.Error("Expected formatted issue to contain labels")
	}
}

func TestFormatIssueList(t *testing.T) {
	issues := []*Issue{
		{Number: 1, Title: "Issue 1", State: "open", HTMLURL: "http://example.com/1"},
		{Number: 2, Title: "Issue 2", State: "closed", HTMLURL: "http://example.com/2"},
	}

	formatted := formatIssueList(issues)

	if !strings.Contains(formatted, "Found 2 issue(s)") {
		t.Error("Expected formatted list to contain count")
	}
	if !strings.Contains(formatted, "#1") || !strings.Contains(formatted, "#2") {
		t.Error("Expected formatted list to contain issue numbers")
	}
}

func TestFormatPR(t *testing.T) {
	pr := &PullRequest{
		Number:  10,
		Title:   "Test PR",
		State:   "open",
		Draft:   true,
		User:    &User{Login: "author"},
		Head:    &PRBranch{Ref: "feature"},
		Base:    &PRBranch{Ref: "main"},
		HTMLURL: "https://github.com/test/repo/pull/10",
	}

	formatted := formatPR(pr)

	if !strings.Contains(formatted, "#10") {
		t.Error("Expected formatted PR to contain PR number")
	}
	if !strings.Contains(formatted, "[DRAFT]") {
		t.Error("Expected formatted PR to contain draft label")
	}
	if !strings.Contains(formatted, "feature → main") {
		t.Error("Expected formatted PR to contain branch info")
	}
}

func TestFormatPRList(t *testing.T) {
	prs := []*PullRequest{
		{Number: 1, Title: "PR 1", State: "open", HTMLURL: "http://example.com/1"},
		{Number: 2, Title: "PR 2", State: "closed", HTMLURL: "http://example.com/2"},
	}

	formatted := formatPRList(prs)

	if !strings.Contains(formatted, "Found 2 pull request(s)") {
		t.Error("Expected formatted list to contain count")
	}
}

func TestFormatCodeSearchResult(t *testing.T) {
	result := &CodeSearchResult{
		TotalCount: 100,
		Items: []*CodeSearchItem{
			{
				Name:       "main.go",
				Path:       "cmd/main.go",
				Repository: &Repository{FullName: "owner/repo"},
				HTMLURL:    "https://github.com/owner/repo/blob/main/cmd/main.go",
				TextMatches: []TextMatch{
					{Fragment: "func main() {\n    fmt.Println(\"Hello\")\n}"},
				},
			},
		},
	}

	formatted := formatCodeSearchResult(result)

	if !strings.Contains(formatted, "Found 1 result(s)") {
		t.Error("Expected formatted result to contain count")
	}
	if !strings.Contains(formatted, "main.go") {
		t.Error("Expected formatted result to contain filename")
	}
	if !strings.Contains(formatted, "owner/repo") {
		t.Error("Expected formatted result to contain repository")
	}
	if !strings.Contains(formatted, "func main()") {
		t.Error("Expected formatted result to contain code snippet")
	}
}

func TestFormatEmptyResults(t *testing.T) {
	// Empty issues
	if got := formatIssueList([]*Issue{}); !strings.Contains(got, "No issues") {
		t.Errorf("formatIssueList empty: expected 'No issues', got %s", got)
	}

	// Empty PRs
	if got := formatPRList([]*PullRequest{}); !strings.Contains(got, "No pull") {
		t.Errorf("formatPRList empty: expected 'No pull', got %s", got)
	}

	// Empty search
	emptySearch := &CodeSearchResult{TotalCount: 0, Items: []*CodeSearchItem{}}
	if got := formatCodeSearchResult(emptySearch); !strings.Contains(got, "No results") {
		t.Errorf("formatCodeSearchResult empty: expected 'No results', got %s", got)
	}
}

// Test that all tools can be created and have proper schemas
func TestAllTools(t *testing.T) {
	client := &Client{}

	tools := []*tools.Tool{
		NewListIssuesTool(client),
		NewListPRsTool(client),
		NewCreateIssueTool(client),
		NewSearchCodeTool(client),
		NewCommentTool(client),
	}

	expectedNames := []string{
		"github_list_issues",
		"github_list_prs",
		"github_create_issue",
		"github_search_code",
		"github_comment",
	}

	for i, tool := range tools {
		if tool.Name != expectedNames[i] {
			t.Errorf("Tool %d: expected name %s, got %s", i, expectedNames[i], tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("Tool %s: missing description", tool.Name)
		}
		if tool.Execute == nil {
			t.Errorf("Tool %s: missing Execute function", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("Tool %s: missing InputSchema", tool.Name)
		}
	}
}
