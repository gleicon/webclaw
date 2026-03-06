//go:build js && wasm

package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/oauth"
)

// Client provides access to the GitHub API v3 (REST)
type Client struct {
	baseURL   string
	oauthMgr  *oauth.OAuthManager
	rateLimit *RateLimit
}

// RateLimit tracks GitHub API rate limit status
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// NewClient creates a new GitHub API client
func NewClient(oauthMgr *oauth.OAuthManager) *Client {
	return &Client{
		baseURL:   "https://api.github.com",
		oauthMgr:  oauthMgr,
		rateLimit: &RateLimit{},
	}
}

// GetRateLimit returns the current rate limit status
func (c *Client) GetRateLimit() *RateLimit {
	return c.rateLimit
}

// GetToken retrieves the GitHub OAuth token
func (c *Client) GetToken() (string, error) {
	return c.oauthMgr.GetToken("github")
}

// IsConnected checks if GitHub is connected
func (c *Client) IsConnected() bool {
	return c.oauthMgr.IsConnected("github")
}

// doRequest makes an authenticated request to the GitHub API
func (c *Client) doRequest(method, path string, body string, params map[string]string) (*jsbridge.FetchResponse, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, fmt.Errorf("GitHub not connected: %w", err)
	}

	// Build URL with query parameters
	requestURL := c.baseURL + path
	if len(params) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github.v3+json",
		"User-Agent":    "webclaw/1.0",
	}

	opts := jsbridge.FetchOptions{
		Method:  method,
		Headers: headers,
		Body:    body,
	}

	resp, err := jsbridge.Fetch(requestURL, opts)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Update rate limit from headers
	c.updateRateLimit(resp.Headers)

	return resp, nil
}

// updateRateLimit extracts rate limit info from response headers
func (c *Client) updateRateLimit(headers map[string]string) {
	if limit := headers["x-ratelimit-limit"]; limit != "" {
		c.rateLimit.Limit, _ = strconv.Atoi(limit)
	}
	if remaining := headers["x-ratelimit-remaining"]; remaining != "" {
		c.rateLimit.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := headers["x-ratelimit-reset"]; reset != "" {
		if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimit.Reset = time.Unix(epoch, 0)
		}
	}
}

// checkRateLimit returns an error if rate limited
func (c *Client) checkRateLimit() error {
	if c.rateLimit.Remaining == 0 && time.Now().Before(c.rateLimit.Reset) {
		waitTime := time.Until(c.rateLimit.Reset)
		return fmt.Errorf("GitHub API rate limit exceeded. Reset in %v", waitTime)
	}
	return nil
}

// parseError parses a GitHub API error response
func parseError(body []byte) error {
	var ghErr GitHubError
	if err := json.Unmarshal(body, &ghErr); err != nil {
		return fmt.Errorf("GitHub API error: %s", string(body))
	}
	return &ghErr
}

// GetIssues returns issues for the authenticated user or a specific repository
func (c *Client) GetIssues(state, assignee string, labels []string, owner, repo string, perPage int) ([]*Issue, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"state":    state,
		"per_page": strconv.Itoa(perPage),
	}

	if assignee != "" {
		params["assignee"] = assignee
	}
	if len(labels) > 0 {
		params["labels"] = strings.Join(labels, ",")
	}

	var path string
	if owner != "" && repo != "" {
		// Repository-specific issues
		path = fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	} else {
		// User's issues across all repos
		path = "/user/issues"
		params["filter"] = "assigned"
	}

	resp, err := c.doRequest("GET", path, "", params)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, parseError(resp.Body)
	}

	var issues []*Issue
	if err := json.Unmarshal(resp.Body, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// GetPullRequests returns pull requests for a repository
func (c *Client) GetPullRequests(owner, repo, state string, perPage int) ([]*PullRequest, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"state":     state,
		"per_page":  strconv.Itoa(perPage),
		"sort":      "updated",
		"direction": "desc",
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	resp, err := c.doRequest("GET", path, "", params)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, parseError(resp.Body)
	}

	var prs []*PullRequest
	if err := json.Unmarshal(resp.Body, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull requests: %w", err)
	}

	return prs, nil
}

// CreateIssue creates a new issue in a repository
func (c *Client) CreateIssue(owner, repo, title, body string, labels []string) (*Issue, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"title": title,
	}
	if body != "" {
		payload["body"] = body
	}
	if len(labels) > 0 {
		payload["labels"] = labels
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	resp, err := c.doRequest("POST", path, string(jsonBody), nil)
	if err != nil {
		return nil, err
	}

	if resp.Status != 201 {
		return nil, parseError(resp.Body)
	}

	var issue Issue
	if err := json.Unmarshal(resp.Body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse created issue: %w", err)
	}

	return &issue, nil
}

// GetRepository returns information about a repository
func (c *Client) GetRepository(owner, repo string) (*Repository, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	resp, err := c.doRequest("GET", path, "", nil)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, parseError(resp.Body)
	}

	var repository Repository
	if err := json.Unmarshal(resp.Body, &repository); err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	return &repository, nil
}

// SearchCode searches code across GitHub repositories
func (c *Client) SearchCode(query string, perPage int) (*CodeSearchResult, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"q":        query,
		"per_page": strconv.Itoa(perPage),
	}

	resp, err := c.doRequest("GET", "/search/code", "", params)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, parseError(resp.Body)
	}

	var result CodeSearchResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return &result, nil
}

// CreateComment adds a comment to an issue or pull request
func (c *Client) CreateComment(owner, repo string, issueNumber int, body string) (*Comment, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"body": body,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, issueNumber)
	resp, err := c.doRequest("POST", path, string(jsonBody), nil)
	if err != nil {
		return nil, err
	}

	if resp.Status != 201 {
		return nil, parseError(resp.Body)
	}

	var comment Comment
	if err := json.Unmarshal(resp.Body, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse comment: %w", err)
	}

	return &comment, nil
}

// ListUserRepos returns repositories for the authenticated user
func (c *Client) ListUserRepos(perPage int) ([]*Repository, error) {
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"per_page":  strconv.Itoa(perPage),
		"sort":      "updated",
		"direction": "desc",
	}

	resp, err := c.doRequest("GET", "/user/repos", "", params)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, parseError(resp.Body)
	}

	var repos []*Repository
	if err := json.Unmarshal(resp.Body, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repositories: %w", err)
	}

	return repos, nil
}
