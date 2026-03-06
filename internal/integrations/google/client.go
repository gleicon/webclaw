//go:build js && wasm

package google

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/oauth"
)

// Client provides a shared foundation for Google API clients (Gmail, Calendar, etc.)
type Client struct {
	oauthMgr *oauth.OAuthManager
	baseURLs struct {
		Gmail    string // https://gmail.googleapis.com/gmail/v1
		Calendar string // https://www.googleapis.com/calendar/v3
	}
}

// NewClient creates a new Google API client foundation
func NewClient(oauthMgr *oauth.OAuthManager) *Client {
	c := &Client{
		oauthMgr: oauthMgr,
	}
	c.baseURLs.Gmail = "https://gmail.googleapis.com/gmail/v1"
	c.baseURLs.Calendar = "https://www.googleapis.com/calendar/v3"
	return c
}

// DoRequest makes an authenticated request to a Google API
// Automatically adds the Authorization header with the OAuth token
func (c *Client) DoRequest(method, url string, body []byte) (*jsbridge.FetchResponse, error) {
	// Get valid access token (with automatic refresh if needed)
	token, err := c.oauthMgr.GetToken("google")
	if err != nil {
		return nil, fmt.Errorf("google not connected: %w", err)
	}

	// Build request options
	opts := jsbridge.FetchOptions{
		Method: method,
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "application/json",
		},
	}

	// Add body for POST/PUT/PATCH requests
	if body != nil && len(body) > 0 {
		opts.Headers["Content-Type"] = "application/json"
		opts.Body = string(body)
	}

	// Make the request
	resp, err := jsbridge.Fetch(url, opts)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Parse Google API error format
	if resp.Status >= 400 {
		return nil, c.parseAPIError(resp)
	}

	return resp, nil
}

// GoogleAPIError represents an error response from Google APIs
type GoogleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Errors  []struct {
		Domain  string `json:"domain"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"errors"`
}

// ErrorResponse wraps the Google API error format
type ErrorResponse struct {
	Error GoogleAPIError `json:"error"`
}

// parseAPIError extracts a user-friendly error from Google API response
func (c *Client) parseAPIError(resp *jsbridge.FetchResponse) error {
	// Try to parse structured error
	var errResp ErrorResponse
	if err := json.Unmarshal(resp.Body, &errResp); err == nil && errResp.Error.Code != 0 {
		// Structured error response
		msg := errResp.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("Google API error (code %d)", errResp.Error.Code)
		}

		// Add specific handling for common errors
		switch resp.Status {
		case 401:
			return fmt.Errorf("authentication failed: %s (try reconnecting Google in Settings)", msg)
		case 403:
			return fmt.Errorf("permission denied: %s (check Google account permissions)", msg)
		case 404:
			return fmt.Errorf("not found: %s", msg)
		case 429:
			return fmt.Errorf("rate limited: %s (please try again later)", msg)
		default:
			return fmt.Errorf("%s (HTTP %d)", msg, resp.Status)
		}
	}

	// Fallback to generic error
	return fmt.Errorf("Google API error: HTTP %d - %s", resp.Status, resp.StatusText)
}

// PaginatedRequest follows page tokens to collect all results
// pageTokenExtractor should extract the nextPageToken from the response
func (c *Client) PaginatedRequest(
	method, baseURL string,
	body []byte,
	pageTokenParam string,
	pageTokenExtractor func([]byte) string,
	resultCollector func([]byte),
) error {
	pageToken := ""

	for {
		// Build URL with page token if present
		url := baseURL
		if pageToken != "" {
			if strings.Contains(url, "?") {
				url = url + "&" + pageTokenParam + "=" + pageToken
			} else {
				url = url + "?" + pageTokenParam + "=" + pageToken
			}
		}

		// Make request
		resp, err := c.DoRequest(method, url, body)
		if err != nil {
			return err
		}

		// Collect results
		resultCollector(resp.Body)

		// Extract next page token
		pageToken = pageTokenExtractor(resp.Body)
		if pageToken == "" {
			break // No more pages
		}
	}

	return nil
}

// EncodeBase64URL encodes data using base64url encoding (URL-safe, no padding)
// This is required for Gmail message encoding (RFC 4648)
func EncodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// DecodeBase64URL decodes base64url encoded data
func DecodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// BuildGmailURL constructs a Gmail API URL
func (c *Client) BuildGmailURL(path string) string {
	return c.baseURLs.Gmail + path
}

// BuildCalendarURL constructs a Calendar API URL
func (c *Client) BuildCalendarURL(path string) string {
	return c.baseURLs.Calendar + path
}
