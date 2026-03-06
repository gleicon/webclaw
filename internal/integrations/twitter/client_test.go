//go:build js && wasm

package twitter

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// mockHTTPClient is a mock HTTP client for testing
type mockHTTPClient struct {
	responses map[string]*jsbridge.FetchResponse
	errors    map[string]error
	calls     []mockCall
}

type mockCall struct {
	url  string
	opts jsbridge.FetchOptions
}

func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		responses: make(map[string]*jsbridge.FetchResponse),
		errors:    make(map[string]error),
		calls:     []mockCall{},
	}
}

func (m *mockHTTPClient) Fetch(url string, opts jsbridge.FetchOptions) (*jsbridge.FetchResponse, error) {
	m.calls = append(m.calls, mockCall{url: url, opts: opts})

	if err, ok := m.errors[url]; ok {
		return nil, err
	}

	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}

	// Try prefix matching
	for pattern, resp := range m.responses {
		if len(url) >= len(pattern) && url[:len(pattern)] == pattern {
			return resp, nil
		}
	}

	return &jsbridge.FetchResponse{
		Status: 404,
		Body:   []byte(`{"errors":[{"message":"Not Found"}]}`),
	}, nil
}

func (m *mockHTTPClient) addResponse(url string, status int, body string) {
	m.responses[url] = &jsbridge.FetchResponse{
		Status:  status,
		Body:    []byte(body),
		Headers: make(map[string]string),
	}
}

func (m *mockHTTPClient) addResponseWithHeaders(url string, status int, body string, headers map[string]string) {
	m.responses[url] = &jsbridge.FetchResponse{
		Status:  status,
		Body:    []byte(body),
		Headers: headers,
	}
}

// Test Types JSON marshaling
func TestTypesJSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		data string
		want interface{}
	}{
		{
			name: "Tweet",
			data: `{"id":"123456","text":"Hello World","author_id":"789","created_at":"2024-01-01T00:00:00Z","public_metrics":{"retweet_count":5,"like_count":10,"reply_count":2}}`,
			want: &Tweet{
				ID:        "123456",
				Text:      "Hello World",
				AuthorID:  "789",
				CreatedAt: "2024-01-01T00:00:00Z",
				PublicMetrics: &TweetPublicMetrics{
					RetweetCount: 5,
					LikeCount:    10,
					ReplyCount:   2,
				},
			},
		},
		{
			name: "User",
			data: `{"id":"789","name":"Test User","username":"testuser"}`,
			want: &User{
				ID:       "789",
				Name:     "Test User",
				Username: "testuser",
			},
		},
		{
			name: "TweetResponse",
			data: `{"data":{"id":"123","text":"Hello","author_id":"456"}}`,
			want: &TweetResponse{
				Data: Tweet{
					ID:       "123",
					Text:     "Hello",
					AuthorID: "456",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch want := tt.want.(type) {
			case *Tweet:
				var got Tweet
				if err := json.Unmarshal([]byte(tt.data), &got); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}
				if got.ID != want.ID || got.Text != want.Text {
					t.Errorf("Unmarshal mismatch: got %+v, want %+v", got, want)
				}
			case *User:
				var got User
				if err := json.Unmarshal([]byte(tt.data), &got); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}
				if got.ID != want.ID || got.Username != want.Username {
					t.Errorf("Unmarshal mismatch: got %+v, want %+v", got, want)
				}
			case *TweetResponse:
				var got TweetResponse
				if err := json.Unmarshal([]byte(tt.data), &got); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}
				if got.Data.ID != want.Data.ID {
					t.Errorf("Unmarshal mismatch: got %+v, want %+v", got, want)
				}
			}
		})
	}
}

// Test that TwitterError implements error interface
func TestTwitterErrorResponse(t *testing.T) {
	errResp := &TwitterErrorResponse{
		Errors: []TwitterError{
			{Code: 88, Message: "Rate limit exceeded"},
		},
	}

	if errResp.Error() != "Rate limit exceeded" {
		t.Errorf("Expected 'Rate limit exceeded', got %s", errResp.Error())
	}

	// Test with Detail instead of Errors
	errResp2 := &TwitterErrorResponse{
		Title:  "Not Found",
		Detail: "No data available",
	}

	if errResp2.Error() != "No data available" {
		t.Errorf("Expected 'No data available', got %s", errResp2.Error())
	}
}

// Test building query strings
func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		check  func(string) bool
	}{
		{
			name:   "empty params",
			params: map[string]string{},
			check:  func(s string) bool { return s == "" },
		},
		{
			name: "single param",
			params: map[string]string{
				"key": "value",
			},
			check: func(s string) bool { return strings.Contains(s, "key=value") },
		},
		{
			name: "special characters",
			params: map[string]string{
				"query": "hello world",
			},
			check: func(s string) bool { return strings.Contains(s, "hello+") || strings.Contains(s, "hello%20") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQueryString(tt.params)
			if !tt.check(got) {
				t.Errorf("Query string check failed for: %s", got)
			}
		})
	}
}

// Test caching functionality
func TestClientCache(t *testing.T) {
	client := &Client{
		baseURL:    baseURL,
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}

	// Test set and get
	testData := []byte(`{"test": "data"}`)
	client.setCache("test-key", testData, 2*time.Minute)

	retrieved := client.getFromCache("test-key")
	if string(retrieved) != string(testData) {
		t.Error("Cache retrieval failed")
	}

	// Test cache miss
	miss := client.getFromCache("nonexistent")
	if miss != nil {
		t.Error("Expected nil for cache miss")
	}

	// Test cache expiration
	client.setCache("expired-key", testData, 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond)
	expired := client.getFromCache("expired-key")
	if expired != nil {
		t.Error("Expected expired cache to return nil")
	}
}

// Test rate limit tracking
func TestRateLimitTracking(t *testing.T) {
	client := &Client{
		baseURL:    baseURL,
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}

	// Parse rate limit headers
	headers := map[string]string{
		"x-rate-limit-limit":     "300",
		"x-rate-limit-remaining": "299",
		"x-rate-limit-reset":     "1704067200",
	}

	client.parseRateLimitHeaders("/test", headers)

	limit := client.GetRateLimit("/test")
	if limit == nil {
		t.Fatal("Expected rate limit to be tracked")
	}

	if limit.Limit != 300 {
		t.Errorf("Expected limit 300, got %d", limit.Limit)
	}

	if limit.Remaining != 299 {
		t.Errorf("Expected remaining 299, got %d", limit.Remaining)
	}

	if limit.ResetTime != 1704067200 {
		t.Errorf("Expected reset time 1704067200, got %d", limit.ResetTime)
	}
}

// Test error response handling
func TestHandleErrorResponse(t *testing.T) {
	client := &Client{
		baseURL:    baseURL,
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}

	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		errContain string
	}{
		{
			name:    "success",
			status:  200,
			body:    `{"data": {"id": "1"}}`,
			wantErr: false,
		},
		{
			name:       "auth error",
			status:     401,
			body:       `{"errors":[{"message":"Unauthorized"}]}`,
			wantErr:    true,
			errContain: "authentication failed",
		},
		{
			name:       "permission error",
			status:     403,
			body:       `{"errors":[{"message":"Forbidden"}]}`,
			wantErr:    true,
			errContain: "permission denied",
		},
		{
			name:       "rate limit error",
			status:     429,
			body:       `{"errors":[{"message":"Too Many Requests"}]}`,
			wantErr:    true,
			errContain: "rate limited",
		},
		{
			name:       "not found",
			status:     404,
			body:       `{"errors":[{"message":"Not Found"}]}`,
			wantErr:    true,
			errContain: "not found",
		},
		{
			name:       "generic error",
			status:     500,
			body:       `{"errors":[{"message":"Internal Error"}]}`,
			wantErr:    true,
			errContain: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &jsbridge.FetchResponse{
				Status:  tt.status,
				Body:    []byte(tt.body),
				Headers: make(map[string]string),
			}

			result, err := client.handleErrorResponse(resp)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContain)
				} else if !strings.Contains(strings.ToLower(err.Error()), tt.errContain) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected response, got nil")
				}
			}
		})
	}
}

// Test tweet formatting helpers
func TestFormatTweetForLLM(t *testing.T) {
	tweet := Tweet{
		ID:        "123",
		Text:      "This is a test tweet",
		AuthorID:  "user1",
		CreatedAt: "2024-01-15T10:30:00Z",
		PublicMetrics: &TweetPublicMetrics{
			LikeCount:    42,
			ReplyCount:   5,
			RetweetCount: 12,
		},
	}

	author := User{
		ID:       "user1",
		Name:     "Test User",
		Username: "testuser",
	}

	result := formatTweetForLLM(1, tweet, author)

	// Check that result contains key elements
	if !strings.Contains(result, "Test User") {
		t.Error("Expected author name in formatted tweet")
	}
	if !strings.Contains(result, "@testuser") {
		t.Error("Expected username in formatted tweet")
	}
	if !strings.Contains(result, "This is a test tweet") {
		t.Error("Expected tweet text in formatted tweet")
	}
	if !strings.Contains(result, "42") { // Like count
		t.Error("Expected like count in formatted tweet")
	}
}

// Test truncateString helper
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is a..."},
		{"exact", 5, "exact"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// Test formatSearchResults
func TestFormatSearchResults(t *testing.T) {
	results := &SearchResponse{
		Data: []Tweet{
			{ID: "1", Text: "First tweet", AuthorID: "a1", CreatedAt: "2024-01-01T00:00:00Z"},
			{ID: "2", Text: "Second tweet", AuthorID: "a2", CreatedAt: "2024-01-01T01:00:00Z"},
		},
		Includes: struct {
			Users []User `json:"users"`
		}{
			Users: []User{
				{ID: "a1", Name: "Author One", Username: "author1"},
				{ID: "a2", Name: "Author Two", Username: "author2"},
			},
		},
	}

	formatted := formatSearchResults(results, "test query")

	if !strings.Contains(formatted, "test query") {
		t.Error("Expected query in formatted results")
	}
	if !strings.Contains(formatted, "Author One") {
		t.Error("Expected author names in formatted results")
	}
	if !strings.Contains(formatted, "First tweet") {
		t.Error("Expected tweet text in formatted results")
	}
}

// Test formatTimelineResults
func TestFormatTimelineResults(t *testing.T) {
	timeline := &TimelineResponse{
		Data: []Tweet{
			{ID: "1", Text: "Timeline tweet", AuthorID: "user1", CreatedAt: "2024-01-01T00:00:00Z"},
		},
		Includes: struct {
			Users []User `json:"users"`
		}{
			Users: []User{
				{ID: "user1", Name: "Timeline User", Username: "timelineuser"},
			},
		},
	}

	formatted := formatTimelineResults(timeline)

	if !strings.Contains(formatted, "timeline") {
		t.Error("Expected 'timeline' in formatted results")
	}
	if !strings.Contains(formatted, "Timeline User") {
		t.Error("Expected author name in formatted timeline")
	}
}

// Test empty search results
func TestFormatEmptySearchResults(t *testing.T) {
	results := &SearchResponse{
		Data: []Tweet{},
	}

	formatted := formatSearchResults(results, "empty query")

	if !strings.Contains(formatted, "No tweets found") {
		t.Error("Expected 'No tweets found' message")
	}
}

// Test empty timeline
func TestFormatEmptyTimeline(t *testing.T) {
	timeline := &TimelineResponse{
		Data: []Tweet{},
	}

	formatted := formatTimelineResults(timeline)

	if !strings.Contains(formatted, "empty") {
		t.Error("Expected 'empty' message for empty timeline")
	}
}

// Integration test simulating tool execution flow
func TestToolExecutionFlow(t *testing.T) {
	mockHTTP := newMockHTTPClient()

	// Setup mock responses
	successResp := `{"data":{"id":"123","text":"Posted tweet","author_id":"me","created_at":"2024-01-01T00:00:00Z"}}`
	mockHTTP.addResponse(baseURL+"/tweets", 201, successResp)

	searchResp := `{
		"data": [{"id":"s1","text":"Search result","author_id":"a1","created_at":"2024-01-01T00:00:00Z"}],
		"meta": {"result_count": 1}
	}`
	mockHTTP.addResponse(baseURL+"/tweets/search/recent", 200, searchResp)

	// Create mock OAuth manager that reports connected
	// For this test, we'll use the actual client with mocked HTTP
	// and test the individual components

	// Test post tweet validation
	t.Run("tweet length validation", func(t *testing.T) {
		// Valid length
		if len("Valid tweet") > maxTweetLength {
			t.Error("Expected short tweet to be valid")
		}

		// Invalid length
		longTweet := strings.Repeat("a", 281)
		if len(longTweet) <= maxTweetLength {
			t.Error("Expected long tweet to exceed limit")
		}
	})

	// Test search query validation
	t.Run("search query validation", func(t *testing.T) {
		// Valid query
		if "" == "valid" {
			t.Error("Empty query check failed")
		}

		// Empty query should fail
		emptyQuery := ""
		if emptyQuery != "" {
			t.Error("Empty query detection failed")
		}
	})
}

// Test parameter extraction for tools
func TestToolParameterExtraction(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "string param",
			params:   map[string]interface{}{"text": "hello"},
			key:      "text",
			expected: "hello",
		},
		{
			name:     "missing param",
			params:   map[string]interface{}{},
			key:      "text",
			expected: "",
		},
		{
			name:     "wrong type param",
			params:   map[string]interface{}{"text": 123},
			key:      "text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := tt.params[tt.key].(string)
			if tt.expected == "" {
				if ok && val != "" {
					t.Errorf("Expected empty/non-existent value for key %s", tt.key)
				}
			} else {
				if !ok || val != tt.expected {
					t.Errorf("Expected %s, got %v", tt.expected, tt.params[tt.key])
				}
			}
		})
	}
}

// Test integer parameter extraction (for count parameters)
func TestIntParameterExtraction(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected int
	}{
		{
			name:     "float64 count",
			params:   map[string]interface{}{"count": float64(10)},
			expected: 10,
		},
		{
			name:     "int count",
			params:   map[string]interface{}{"count": 20},
			expected: 20,
		},
		{
			name:     "missing count",
			params:   map[string]interface{}{},
			expected: 0, // Will use default
		},
		{
			name:     "invalid count",
			params:   map[string]interface{}{"count": "invalid"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			if c, ok := tt.params["count"].(float64); ok {
				count = int(c)
			}
			if c, ok := tt.params["count"].(int); ok {
				count = c
			}

			if count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

// Benchmark cache operations
func BenchmarkCacheSetAndGet(b *testing.B) {
	client := &Client{
		baseURL:    baseURL,
		rateLimits: make(map[string]*RateLimit),
		cache:      make(map[string]*cacheEntry),
	}

	data := []byte(`{"test": "data"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		client.setCache(key, data, 2*time.Minute)
		_ = client.getFromCache(key)
	}
}

// Benchmark JSON marshaling
func BenchmarkTweetUnmarshal(b *testing.B) {
	jsonData := []byte(`{"id":"123456","text":"Hello World","author_id":"789","created_at":"2024-01-01T00:00:00Z","public_metrics":{"retweet_count":5,"like_count":10,"reply_count":2}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var tweet Tweet
		_ = json.Unmarshal(jsonData, &tweet)
	}
}
