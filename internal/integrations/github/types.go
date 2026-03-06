//go:build js && wasm

package github

import "fmt"

// Issue represents a GitHub issue (or PR, since PRs are issues under the hood)
type Issue struct {
	ID        int64    `json:"id"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	State     string   `json:"state"` // open, closed
	User      *User    `json:"user"`
	Assignees []*User  `json:"assignees"`
	Labels    []*Label `json:"labels"`
	CreatedAt string   `json:"created_at"` // ISO 8601
	UpdatedAt string   `json:"updated_at"`
	HTMLURL   string   `json:"html_url"` // Web URL
	Comments  int      `json:"comments"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID        int64     `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	User      *User     `json:"user"`
	Head      *PRBranch `json:"head"` // Source branch
	Base      *PRBranch `json:"base"` // Target branch
	Draft     bool      `json:"draft"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Comments  int       `json:"comments"`
}

// PRBranch represents a branch in a pull request
type PRBranch struct {
	Ref  string      `json:"ref"` // Branch name
	Sha  string      `json:"sha"`
	Repo *Repository `json:"repo"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"` // owner/repo
	Description     string `json:"description"`
	Private         bool   `json:"private"`
	HTMLURL         string `json:"html_url"`
	Owner           *User  `json:"owner"`
	OpenIssuesCount int    `json:"open_issues_count"`
}

// User represents a GitHub user
type User struct {
	ID      int64  `json:"id"`
	Login   string `json:"login"` // Username
	HTMLURL string `json:"html_url"`
	Type    string `json:"type"` // User, Organization
}

// Label represents a GitHub label
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// Comment represents a GitHub issue/PR comment
type Comment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	User      *User  `json:"user"`
	CreatedAt string `json:"created_at"`
	HTMLURL   string `json:"html_url"`
}

// CodeSearchResult represents the response from GitHub code search
type CodeSearchResult struct {
	TotalCount int               `json:"total_count"`
	Items      []*CodeSearchItem `json:"items"`
}

// CodeSearchItem represents a single code search result
type CodeSearchItem struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Sha         string      `json:"sha"`
	URL         string      `json:"url"`
	HTMLURL     string      `json:"html_url"`
	Repository  *Repository `json:"repository"`
	TextMatches []TextMatch `json:"text_matches"`
}

// TextMatch represents a text match in a code search result
type TextMatch struct {
	Fragment string  `json:"fragment"`
	Matches  []Match `json:"matches"`
}

// Match represents a single match within a text fragment
type Match struct {
	Text    string `json:"text"`
	Indices []int  `json:"indices"`
}

// GitHubError represents an error response from the GitHub API
type GitHubError struct {
	Message          string        `json:"message"`
	DocumentationURL string        `json:"documentation_url"`
	Errors           []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents a detailed error from the GitHub API
type ErrorDetail struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message,omitempty"`
}

// Error implements the error interface for GitHubError
func (e *GitHubError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("GitHub API error: %s (%s: %s)", e.Message, e.Errors[0].Field, e.Errors[0].Code)
	}
	return fmt.Sprintf("GitHub API error: %s", e.Message)
}
