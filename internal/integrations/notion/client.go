//go:build js && wasm

package notion

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/oauth"
)

// Client provides an authenticated interface to the Notion API.
type Client struct {
	baseURL  string
	oauthMgr *oauth.OAuthManager
	version  string
}

// NewClient creates a new Notion API client.
func NewClient(oauthMgr *oauth.OAuthManager) *Client {
	return &Client{
		baseURL:  BaseURL,
		oauthMgr: oauthMgr,
		version:  APIVersion,
	}
}

// getToken retrieves and validates the OAuth token for Notion.
func (c *Client) getToken() (string, error) {
	return c.oauthMgr.GetToken("notion")
}

// makeRequest performs an authenticated HTTP request to the Notion API.
func (c *Client) makeRequest(method, endpoint string, body interface{}) (*jsbridge.FetchResponse, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	url := c.baseURL + endpoint

	opts := jsbridge.FetchOptions{
		Method: method,
		Headers: map[string]string{
			"Authorization":  "Bearer " + token,
			"Notion-Version": c.version,
			"Content-Type":   "application/json",
		},
	}

	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		opts.Body = string(bodyJSON)
	}

	// Rate limit: ~3 requests per second average
	time.Sleep(350 * time.Millisecond)

	resp, err := jsbridge.Fetch(url, opts)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Handle rate limiting (429)
	if resp.Status == 429 {
		// Retry after 1 second
		time.Sleep(1 * time.Second)
		resp, err = jsbridge.Fetch(url, opts)
		if err != nil {
			return nil, fmt.Errorf("retry failed: %w", err)
		}
	}

	// Check for API errors
	if resp.Status >= 400 {
		var apiErr NotionError
		if err := json.Unmarshal(resp.Body, &apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.Status, string(resp.Body))
		}
		return nil, &apiErr
	}

	return resp, nil
}

// QueryDatabase queries a Notion database with optional filters and sorting.
func (c *Client) QueryDatabase(databaseID string, query *Query) (*QueryResponse, error) {
	// Clean up database ID (remove dashes if present)
	databaseID = cleanID(databaseID)

	endpoint := fmt.Sprintf("/v1/databases/%s/query", databaseID)

	var body interface{}
	if query != nil {
		body = query
	} else {
		body = map[string]interface{}{}
	}

	resp, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var result QueryResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetDatabase retrieves database metadata including property schemas.
func (c *Client) GetDatabase(databaseID string) (*Database, error) {
	databaseID = cleanID(databaseID)

	endpoint := fmt.Sprintf("/v1/databases/%s", databaseID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result Database
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetPage retrieves a page's metadata (properties).
func (c *Client) GetPage(pageID string) (*Page, error) {
	pageID = cleanID(pageID)

	endpoint := fmt.Sprintf("/v1/pages/%s", pageID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result Page
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetPageContent retrieves the blocks (content) of a page.
func (c *Client) GetPageContent(pageID string, cursor string) (*BlockChildrenResponse, error) {
	pageID = cleanID(pageID)

	endpoint := fmt.Sprintf("/v1/blocks/%s/children", pageID)
	if cursor != "" {
		endpoint += "?start_cursor=" + cursor
	}

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result BlockChildrenResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetPageContentAll retrieves all blocks from a page (handles pagination).
func (c *Client) GetPageContentAll(pageID string) ([]Block, error) {
	var allBlocks []Block
	var cursor string

	for {
		resp, err := c.GetPageContent(pageID, cursor)
		if err != nil {
			return nil, err
		}

		allBlocks = append(allBlocks, resp.Results...)

		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}

	return allBlocks, nil
}

// UpdatePage updates a page's properties (not block content).
func (c *Client) UpdatePage(pageID string, properties map[string]PropertyValue) (*Page, error) {
	pageID = cleanID(pageID)

	endpoint := fmt.Sprintf("/v1/pages/%s", pageID)

	body := map[string]interface{}{
		"properties": properties,
	}

	resp, err := c.makeRequest("PATCH", endpoint, body)
	if err != nil {
		return nil, err
	}

	var result Page
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Search searches pages and databases in Notion.
func (c *Client) Search(query string, filterType string, limit int) (*SearchResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	body := map[string]interface{}{
		"query":     query,
		"page_size": limit,
	}

	// Add filter if specified
	if filterType != "" {
		body["filter"] = map[string]interface{}{
			"value":    filterType, // "page" or "database"
			"property": "object",
		}
	}

	resp, err := c.makeRequest("POST", "/v1/search", body)
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ListDatabases returns all databases the user has access to.
func (c *Client) ListDatabases() ([]*Database, error) {
	resp, err := c.Search("", "database", 100)
	if err != nil {
		return nil, err
	}

	var databases []*Database
	for _, result := range resp.Results {
		if result.IsDatabase() {
			databases = append(databases, result.Database)
		}
	}

	return databases, nil
}

// QueryDatabaseAll queries a database and returns all results (handles pagination).
func (c *Client) QueryDatabaseAll(databaseID string, query *Query) ([]Page, error) {
	var allPages []Page
	var cursor string
	pageSize := 100

	// Build initial query
	if query == nil {
		query = &Query{}
	}

	for {
		query.StartCursor = cursor
		query.PageSize = pageSize

		resp, err := c.QueryDatabase(databaseID, query)
		if err != nil {
			return nil, err
		}

		allPages = append(allPages, resp.Results...)

		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}

	return allPages, nil
}

// cleanID removes dashes from UUIDs and extracts IDs from URLs.
func cleanID(id string) string {
	// Check if it's a URL
	if strings.Contains(id, "notion.so/") {
		// Extract ID from URL
		id = extractIDFromURL(id)
	}

	// Remove dashes from UUID
	return strings.ReplaceAll(id, "-", "")
}

// extractIDFromURL extracts a Notion page/database ID from a URL.
func extractIDFromURL(url string) string {
	// Handle notion.so URLs
	// Format: https://www.notion.so/workspace/Page-Title-1234567890abcdef1234567890abcdef
	// Or: https://www.notion.so/page/1234567890abcdef1234567890abcdef
	// Or: https://www.notion.so/1234567890abcdef1234567890abcdef

	// Try to extract the last part after the last hyphen
	parts := strings.Split(url, "-")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove any query parameters or fragments
		lastPart = strings.Split(lastPart, "?")[0]
		lastPart = strings.Split(lastPart, "#")[0]
		// Remove trailing slash
		lastPart = strings.TrimSuffix(lastPart, "/")

		// Check if it's a 32-character hex string (Notion ID)
		if matched, _ := regexp.MatchString("^[a-f0-9]{32}$", lastPart); matched {
			return lastPart
		}
	}

	// Try direct ID extraction from URL path
	parts = strings.Split(url, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		part = strings.Split(part, "?")[0]
		part = strings.Split(part, "#")[0]

		// Check for 32-char hex ID
		if matched, _ := regexp.MatchString("^[a-f0-9]{32}$", part); matched {
			return part
		}

		// Check for UUID format with dashes
		cleanPart := strings.ReplaceAll(part, "-", "")
		if matched, _ := regexp.MatchString("^[a-f0-9]{32}$", cleanPart); matched {
			return cleanPart
		}
	}

	return url // Return as-is if we can't extract
}

// IsNotConnectedError checks if an error indicates the user needs to connect Notion.
func IsNotConnectedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "please connect") ||
		strings.Contains(errStr, "not found") && strings.Contains(errStr, "token")
}

// IsNotFoundError checks if an error is a "not found" API error.
func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*NotionError); ok {
		return apiErr.Code == "object_not_found"
	}
	return false
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	if apiErr, ok := err.(*NotionError); ok {
		return apiErr.Code == "validation_error"
	}
	return false
}
