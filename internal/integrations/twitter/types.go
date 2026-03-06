//go:build js && wasm

package twitter

// Tweet represents a Twitter API v2 tweet
type Tweet struct {
	ID            string              `json:"id"`
	Text          string              `json:"text"`
	AuthorID      string              `json:"author_id"`
	CreatedAt     string              `json:"created_at"`
	PublicMetrics *TweetPublicMetrics `json:"public_metrics,omitempty"`
}

// TweetPublicMetrics contains engagement metrics for a tweet
type TweetPublicMetrics struct {
	RetweetCount int `json:"retweet_count"`
	LikeCount    int `json:"like_count"`
	ReplyCount   int `json:"reply_count"`
	QuoteCount   int `json:"quote_count"`
}

// User represents a Twitter API v2 user
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// TweetResponse is the response from creating or getting a single tweet
type TweetResponse struct {
	Data Tweet `json:"data"`
}

// TimelineResponse is the response from the home timeline endpoint
type TimelineResponse struct {
	Data     []Tweet `json:"data"`
	Includes struct {
		Users []User `json:"users"`
	} `json:"includes,omitempty"`
	Meta struct {
		NextToken   string `json:"next_token"`
		ResultCount int    `json:"result_count"`
		NewestID    string `json:"newest_id"`
		OldestID    string `json:"oldest_id"`
	} `json:"meta"`
}

// SearchResponse is the response from the search endpoint
type SearchResponse struct {
	Data     []Tweet `json:"data"`
	Includes struct {
		Users []User `json:"users"`
	} `json:"includes,omitempty"`
	Meta struct {
		NextToken   string `json:"next_token"`
		ResultCount int    `json:"result_count"`
		NewestID    string `json:"newest_id"`
		OldestID    string `json:"oldest_id"`
	} `json:"meta"`
}

// TwitterError represents a single error from the Twitter API
type TwitterError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TwitterErrorResponse is the error response from Twitter API
type TwitterErrorResponse struct {
	Errors []TwitterError `json:"errors"`
	Title  string         `json:"title,omitempty"`
	Detail string         `json:"detail,omitempty"`
	Type   string         `json:"type,omitempty"`
}

// Error implements the error interface for TwitterErrorResponse
func (e *TwitterErrorResponse) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0].Message
	}
	if e.Detail != "" {
		return e.Detail
	}
	return e.Title
}

// PostTweetRequest is the request body for creating a tweet
type PostTweetRequest struct {
	Text  string     `json:"text"`
	Reply *ReplyInfo `json:"reply,omitempty"`
}

// ReplyInfo contains information for a reply tweet
type ReplyInfo struct {
	InReplyToTweetID string `json:"in_reply_to_tweet_id"`
}

// RateLimit contains rate limit information from API response headers
type RateLimit struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	ResetTime int64 `json:"reset_time"` // Unix timestamp
}
