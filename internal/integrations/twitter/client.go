//go:build js && wasm

package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/oauth"
)

const (
	baseURL        = "https://api.twitter.com/2"
	maxTweetLength = 280
)

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Fetch(url string, opts jsbridge.FetchOptions) (*jsbridge.FetchResponse, error)
}

// defaultHTTPClient wraps jsbridge.Fetch
type defaultHTTPClient struct{}

func (c *defaultHTTPClient) Fetch(url string, opts jsbridge.FetchOptions) (*jsbridge.FetchResponse, error) {
	return jsbridge.Fetch(url, opts)
}

// Client is the Twitter API v2 client
type Client struct {
	baseURL    string
	oauthMgr   *oauth.OAuthManager
	httpClient HTTPClient

	// Rate limit tracking
	rateLimitMu sync.RWMutex
	rateLimits  map[string]*RateLimit

	// Response cache for read operations
	cacheMu sync.RWMutex
	cache   map[string]*cacheEntry
}

// cacheEntry represents a cached response
type cacheEntry struct {
	data      []byte
	timestamp time.Time
	ttl       time.Duration
}

// NewClient creates a new Twitter API client
func NewClient(oauthMgr *oauth.OAuthManager) *Client {
	return &Client{
		baseURL:    baseURL,
		oauthMgr:   oauthMgr,
		httpClient: &defaultHTTPClient{},
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}
}

// NewClientWithHTTP creates a client with a custom HTTP client (for testing)
func NewClientWithHTTP(oauthMgr *oauth.OAuthManager, httpClient HTTPClient) *Client {
	return &Client{
		baseURL:    baseURL,
		oauthMgr:   oauthMgr,
		httpClient: httpClient,
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}
}

// getAuthHeader returns the Authorization header with Bearer token
func (c *Client) getAuthHeader() (string, error) {
	token, err := c.oauthMgr.GetToken("twitter")
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil
}

// isConnected checks if Twitter is connected
func (c *Client) isConnected() bool {
	return c.oauthMgr.IsConnected("twitter")
}

// makeRequest makes an authenticated API request with caching support
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, queryParams map[string]string, body string, useCache bool) (*jsbridge.FetchResponse, error) {
	if !c.isConnected() {
		return nil, fmt.Errorf("twitter not connected: please connect Twitter in Settings")
	}

	// Build URL with query parameters
	url := c.baseURL + endpoint
	if len(queryParams) > 0 {
		url = url + "?" + buildQueryString(queryParams)
	}

	// Check cache for GET requests
	if useCache && method == "GET" {
		if cached := c.getFromCache(url); cached != nil {
			// Return cached response
			return &jsbridge.FetchResponse{
				Status: 200,
				Body:   cached,
			}, nil
		}
	}

	// Check rate limit before making request
	if limit := c.GetRateLimit(endpoint); limit != nil && limit.Remaining <= 0 {
		resetTime := time.Unix(limit.ResetTime, 0)
		waitTime := time.Until(resetTime)
		if waitTime > 0 {
			return nil, fmt.Errorf("rate limited by Twitter. Try again in %d minutes", int(waitTime.Minutes())+1)
		}
	}

	// Get auth header
	authHeader, err := c.getAuthHeader()
	if err != nil {
		return nil, err
	}

	// Prepare headers
	headers := map[string]string{
		"Authorization": authHeader,
		"Content-Type":  "application/json",
	}

	// Make the request
	opts := jsbridge.FetchOptions{
		Method:  method,
		Headers: headers,
		Body:    body,
	}

	resp, err := c.httpClient.Fetch(url, opts)
	if err != nil {
		return nil, fmt.Errorf("twitter api request failed: %w", err)
	}

	// Parse and store rate limit headers
	c.parseRateLimitHeaders(endpoint, resp.Headers)

	// Handle error responses
	if resp.Status < 200 || resp.Status >= 300 {
		return c.handleErrorResponse(resp)
	}

	// Cache successful GET responses
	if useCache && method == "GET" && resp.Status == 200 {
		c.setCache(url, resp.Body, 2*time.Minute)
	}

	return resp, nil
}

// buildQueryString builds a URL query string from parameters
func buildQueryString(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

// handleErrorResponse handles API error responses
func (c *Client) handleErrorResponse(resp *jsbridge.FetchResponse) (*jsbridge.FetchResponse, error) {
	body := string(resp.Body)

	// Try to parse as Twitter error
	var twitterErr TwitterErrorResponse
	if err := json.Unmarshal(resp.Body, &twitterErr); err == nil && (len(twitterErr.Errors) > 0 || twitterErr.Detail != "") {
		return nil, fmt.Errorf("twitter error: %s", twitterErr.Error())
	}

	// Map HTTP status codes to user-friendly errors
	switch resp.Status {
	case 401:
		return nil, fmt.Errorf("twitter authentication failed: please reconnect your account in Settings")
	case 403:
		return nil, fmt.Errorf("twitter permission denied: please reconnect with more permissions in Settings")
	case 429:
		// Rate limit - get retry time from headers
		if limit := c.GetRateLimit("/tweets"); limit != nil && limit.ResetTime > 0 {
			resetTime := time.Unix(limit.ResetTime, 0)
			waitTime := time.Until(resetTime)
			return nil, fmt.Errorf("rate limited by Twitter. Try again in %d minutes", int(waitTime.Minutes())+1)
		}
		return nil, fmt.Errorf("rate limited by Twitter. Please try again later")
	case 404:
		return nil, fmt.Errorf("tweet not found")
	default:
		return nil, fmt.Errorf("twitter api error (HTTP %d): %s", resp.Status, body)
	}
}

// parseRateLimitHeaders extracts rate limit info from response headers
func (c *Client) parseRateLimitHeaders(endpoint string, headers map[string]string) {
	limit := &RateLimit{}

	if v, ok := headers["x-rate-limit-limit"]; ok {
		limit.Limit, _ = strconv.Atoi(v)
	}
	if v, ok := headers["x-rate-limit-remaining"]; ok {
		limit.Remaining, _ = strconv.Atoi(v)
	}
	if v, ok := headers["x-rate-limit-reset"]; ok {
		limit.ResetTime, _ = strconv.ParseInt(v, 10, 64)
	}

	if limit.Limit > 0 {
		c.rateLimitMu.Lock()
		c.rateLimits[endpoint] = limit
		c.rateLimitMu.Unlock()
	}
}

// GetRateLimit returns the current rate limit for an endpoint
func (c *Client) GetRateLimit(endpoint string) *RateLimit {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()
	return c.rateLimits[endpoint]
}

// getFromCache retrieves cached data if valid
func (c *Client) getFromCache(key string) []byte {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	entry, ok := c.cache[key]
	if !ok {
		return nil
	}

	if time.Since(entry.timestamp) > entry.ttl {
		return nil
	}

	return entry.data
}

// setCache stores data in cache
func (c *Client) setCache(key string, data []byte, ttl time.Duration) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.cache[key] = &cacheEntry{
		data:      data,
		timestamp: time.Now(),
		ttl:       ttl,
	}
}

// clearCache removes cached data for a key
func (c *Client) clearCache(key string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	delete(c.cache, key)
}

// PostTweet creates a new tweet or reply
func (c *Client) PostTweet(ctx context.Context, text string, replyToID string) (*Tweet, error) {
	// Validate tweet length
	if len(text) > maxTweetLength {
		return nil, fmt.Errorf("tweet exceeds %d character limit (%d characters)", maxTweetLength, len(text))
	}
	if len(text) == 0 {
		return nil, fmt.Errorf("tweet text cannot be empty")
	}

	// Build request body
	req := PostTweetRequest{
		Text: text,
	}
	if replyToID != "" {
		req.Reply = &ReplyInfo{
			InReplyToTweetID: replyToID,
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tweet request: %w", err)
	}

	// Make request
	resp, err := c.makeRequest(ctx, "POST", "/tweets", nil, string(body), false)
	if err != nil {
		return nil, err
	}

	// Parse response
	var tweetResp TweetResponse
	if err := json.Unmarshal(resp.Body, &tweetResp); err != nil {
		return nil, fmt.Errorf("failed to parse tweet response: %w", err)
	}

	return &tweetResp.Data, nil
}

// GetTimeline fetches the user's home timeline
func (c *Client) GetTimeline(ctx context.Context, maxResults int, nextToken string) (*TimelineResponse, error) {
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 20
	}

	params := map[string]string{
		"max_results":  strconv.Itoa(maxResults),
		"tweet.fields": "created_at,public_metrics,author_id",
		"expansions":   "author_id",
		"user.fields":  "name,username",
	}
	if nextToken != "" {
		params["pagination_token"] = nextToken
	}

	resp, err := c.makeRequest(ctx, "GET", "/users/me/timelines/reverse_chronological", params, "", true)
	if err != nil {
		return nil, err
	}

	var timelineResp TimelineResponse
	if err := json.Unmarshal(resp.Body, &timelineResp); err != nil {
		return nil, fmt.Errorf("failed to parse timeline response: %w", err)
	}

	return &timelineResp, nil
}

// SearchTweets searches for recent tweets
func (c *Client) SearchTweets(ctx context.Context, query string, maxResults int, nextToken string) (*SearchResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 10
	}

	params := map[string]string{
		"query":        query,
		"max_results":  strconv.Itoa(maxResults),
		"tweet.fields": "created_at,public_metrics,author_id",
		"expansions":   "author_id",
		"user.fields":  "name,username",
	}
	if nextToken != "" {
		params["next_token"] = nextToken
	}

	resp, err := c.makeRequest(ctx, "GET", "/tweets/search/recent", params, "", true)
	if err != nil {
		return nil, err
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(resp.Body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return &searchResp, nil
}

// GetTweet fetches a specific tweet by ID
func (c *Client) GetTweet(ctx context.Context, id string) (*Tweet, error) {
	if id == "" {
		return nil, fmt.Errorf("tweet ID cannot be empty")
	}

	params := map[string]string{
		"tweet.fields": "created_at,public_metrics,author_id",
		"expansions":   "author_id",
		"user.fields":  "name,username",
	}

	endpoint := "/tweets/" + id
	resp, err := c.makeRequest(ctx, "GET", endpoint, params, "", true)
	if err != nil {
		return nil, err
	}

	var tweetResp TweetResponse
	if err := json.Unmarshal(resp.Body, &tweetResp); err != nil {
		return nil, fmt.Errorf("failed to parse tweet response: %w", err)
	}

	return &tweetResp.Data, nil
}
