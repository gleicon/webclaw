//go:build js && wasm

package twitter

import (
	"context"
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/oauth"
	"github.com/gleicon/webclaw/internal/tools"
)

// TwitterToolSet holds all Twitter tools and their shared client
type TwitterToolSet struct {
	client *Client
}

// NewTwitterToolSet creates a new set of Twitter tools
func NewTwitterToolSet(oauthMgr *oauth.OAuthManager) *TwitterToolSet {
	return &TwitterToolSet{
		client: NewClient(oauthMgr),
	}
}

// NewTwitterToolSetWithClient creates a tool set with a custom client (for testing)
func NewTwitterToolSetWithClient(client *Client) *TwitterToolSet {
	return &TwitterToolSet{
		client: client,
	}
}

// RegisterAll registers all Twitter tools with the provided registry
func (t *TwitterToolSet) RegisterAll(registry *tools.Registry) {
	registry.Register(t.NewPostTool())
	registry.Register(t.NewReplyTool())
	registry.Register(t.NewSearchTool())
	registry.Register(t.NewTimelineTool())
}

// NewPostTool creates the twitter_post tool
func (t *TwitterToolSet) NewPostTool() *tools.Tool {
	return &tools.Tool{
		Name:        "twitter_post",
		Description: "Post a new tweet to Twitter/X. Use for sharing updates, announcements, or any public message.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": fmt.Sprintf("Tweet content (max %d characters)", maxTweetLength),
				},
			},
			"required": []string{"text"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Check Twitter connection
			if !t.client.isConnected() {
				return &tools.ToolResult{
					Content:        "Please connect Twitter in Settings first.",
					DisplayContent: "Twitter not connected. Go to Settings → Connected Services to connect your account.",
					IsError:        true,
					ToolName:       "twitter_post",
					Status:         "error",
				}, nil
			}

			// Extract and validate text
			text, ok := params["text"].(string)
			if !ok || text == "" {
				return &tools.ToolResult{
					Content:        "text parameter is required",
					DisplayContent: "Failed: text parameter is required",
					IsError:        true,
					ToolName:       "twitter_post",
					Status:         "error",
				}, nil
			}

			if len(text) > maxTweetLength {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Tweet exceeds %d character limit (%d characters)", maxTweetLength, len(text)),
					DisplayContent: fmt.Sprintf("Failed: Tweet too long (%d/%d characters)", len(text), maxTweetLength),
					IsError:        true,
					ToolName:       "twitter_post",
					Status:         "error",
				}, nil
			}

			// Post the tweet
			tweet, err := t.client.PostTweet(ctx, text, "")
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to post tweet: %v", err),
					DisplayContent: fmt.Sprintf("Failed to post: %v", err),
					IsError:        true,
					ToolName:       "twitter_post",
					Status:         "error",
				}, nil
			}

			// Format success response
			tweetURL := fmt.Sprintf("https://twitter.com/i/web/status/%s", tweet.ID)
			content := fmt.Sprintf("Successfully posted tweet!\n\nTweet ID: %s\nText: %s\nURL: %s", tweet.ID, tweet.Text, tweetURL)
			display := fmt.Sprintf("Posted tweet: %s", truncateString(tweet.Text, 50))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "twitter_post",
				Status:         "done",
			}, nil
		},
	}
}

// NewReplyTool creates the twitter_reply tool
func (t *TwitterToolSet) NewReplyTool() *tools.Tool {
	return &tools.Tool{
		Name:        "twitter_reply",
		Description: "Reply to an existing tweet on Twitter/X. Provide the tweet ID and your reply text.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tweet_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the tweet to reply to",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": fmt.Sprintf("Reply text (max %d characters)", maxTweetLength),
				},
			},
			"required": []string{"tweet_id", "text"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Check Twitter connection
			if !t.client.isConnected() {
				return &tools.ToolResult{
					Content:        "Please connect Twitter in Settings first.",
					DisplayContent: "Twitter not connected. Go to Settings → Connected Services to connect your account.",
					IsError:        true,
					ToolName:       "twitter_reply",
					Status:         "error",
				}, nil
			}

			// Extract parameters
			tweetID, ok := params["tweet_id"].(string)
			if !ok || tweetID == "" {
				return &tools.ToolResult{
					Content:        "tweet_id parameter is required",
					DisplayContent: "Failed: tweet_id parameter is required",
					IsError:        true,
					ToolName:       "twitter_reply",
					Status:         "error",
				}, nil
			}

			text, ok := params["text"].(string)
			if !ok || text == "" {
				return &tools.ToolResult{
					Content:        "text parameter is required",
					DisplayContent: "Failed: text parameter is required",
					IsError:        true,
					ToolName:       "twitter_reply",
					Status:         "error",
				}, nil
			}

			if len(text) > maxTweetLength {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Reply exceeds %d character limit (%d characters)", maxTweetLength, len(text)),
					DisplayContent: fmt.Sprintf("Failed: Reply too long (%d/%d characters)", len(text), maxTweetLength),
					IsError:        true,
					ToolName:       "twitter_reply",
					Status:         "error",
				}, nil
			}

			// Post the reply
			tweet, err := t.client.PostTweet(ctx, text, tweetID)
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to post reply: %v", err),
					DisplayContent: fmt.Sprintf("Failed to reply: %v", err),
					IsError:        true,
					ToolName:       "twitter_reply",
					Status:         "error",
				}, nil
			}

			// Format success response
			tweetURL := fmt.Sprintf("https://twitter.com/i/web/status/%s", tweet.ID)
			content := fmt.Sprintf("Successfully posted reply!\n\nReply ID: %s\nText: %s\nURL: %s", tweet.ID, tweet.Text, tweetURL)
			display := fmt.Sprintf("Replied to tweet %s: %s", truncateString(tweetID, 15), truncateString(tweet.Text, 40))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "twitter_reply",
				Status:         "done",
			}, nil
		},
	}
}

// NewSearchTool creates the twitter_search tool
func (t *TwitterToolSet) NewSearchTool() *tools.Tool {
	return &tools.Tool{
		Name: "twitter_search",
		Description: "Search recent tweets (last 7 days) on Twitter. Supports Twitter search operators like #hashtag, from:username, to:username, " +
			"\"exact phrase\", min_retweets:N, etc.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (supports Twitter search operators: #hashtag, from:username, \"exact phrase\")",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return (max 100, default 10)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Check Twitter connection
			if !t.client.isConnected() {
				return &tools.ToolResult{
					Content:        "Please connect Twitter in Settings first.",
					DisplayContent: "Twitter not connected. Go to Settings → Connected Services to connect your account.",
					IsError:        true,
					ToolName:       "twitter_search",
					Status:         "error",
				}, nil
			}

			// Extract query
			query, ok := params["query"].(string)
			if !ok || query == "" {
				return &tools.ToolResult{
					Content:        "query parameter is required",
					DisplayContent: "Failed: query parameter is required",
					IsError:        true,
					ToolName:       "twitter_search",
					Status:         "error",
				}, nil
			}

			// Extract count with default
			count := 10
			if c, ok := params["count"].(float64); ok {
				count = int(c)
			}
			if c, ok := params["count"].(int); ok {
				count = c
			}
			if count <= 0 || count > 100 {
				count = 10
			}

			// Search tweets
			results, err := t.client.SearchTweets(ctx, query, count, "")
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Search failed: %v", err),
					DisplayContent: fmt.Sprintf("Search failed: %v", err),
					IsError:        true,
					ToolName:       "twitter_search",
					Status:         "error",
				}, nil
			}

			// Format results
			content := formatSearchResults(results, query)
			display := fmt.Sprintf("Found %d tweets for \"%s\"", len(results.Data), truncateString(query, 30))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "twitter_search",
				Status:         "done",
			}, nil
		},
	}
}

// NewTimelineTool creates the twitter_timeline tool
func (t *TwitterToolSet) NewTimelineTool() *tools.Tool {
	return &tools.Tool{
		Name:        "twitter_timeline",
		Description: "Get recent tweets from your home timeline (tweets from people you follow).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of tweets to fetch (max 100, default 20)",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			// Check Twitter connection
			if !t.client.isConnected() {
				return &tools.ToolResult{
					Content:        "Please connect Twitter in Settings first.",
					DisplayContent: "Twitter not connected. Go to Settings → Connected Services to connect your account.",
					IsError:        true,
					ToolName:       "twitter_timeline",
					Status:         "error",
				}, nil
			}

			// Extract count with default
			count := 20
			if c, ok := params["count"].(float64); ok {
				count = int(c)
			}
			if c, ok := params["count"].(int); ok {
				count = c
			}
			if count <= 0 || count > 100 {
				count = 20
			}

			// Get timeline
			timeline, err := t.client.GetTimeline(ctx, count, "")
			if err != nil {
				return &tools.ToolResult{
					Content:        fmt.Sprintf("Failed to fetch timeline: %v", err),
					DisplayContent: fmt.Sprintf("Failed to fetch timeline: %v", err),
					IsError:        true,
					ToolName:       "twitter_timeline",
					Status:         "error",
				}, nil
			}

			// Format results
			content := formatTimelineResults(timeline)
			display := fmt.Sprintf("Fetched %d tweets from timeline", len(timeline.Data))

			return &tools.ToolResult{
				Content:        content,
				DisplayContent: display,
				IsError:        false,
				ToolName:       "twitter_timeline",
				Status:         "done",
			}, nil
		},
	}
}

// formatSearchResults formats search results for LLM consumption
func formatSearchResults(results *SearchResponse, query string) string {
	if len(results.Data) == 0 {
		return fmt.Sprintf("No tweets found for query: %s", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\" (%d tweets found):\n\n", query, len(results.Data)))

	// Build user lookup map
	userMap := make(map[string]User)
	for _, user := range results.Includes.Users {
		userMap[user.ID] = user
	}

	for i, tweet := range results.Data {
		author := userMap[tweet.AuthorID]
		sb.WriteString(formatTweetForLLM(i+1, tweet, author))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTimelineResults formats timeline results for LLM consumption
func formatTimelineResults(timeline *TimelineResponse) string {
	if len(timeline.Data) == 0 {
		return "Your timeline is empty (no recent tweets from people you follow)."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Your Twitter timeline (%d recent tweets):\n\n", len(timeline.Data)))

	// Build user lookup map
	userMap := make(map[string]User)
	for _, user := range timeline.Includes.Users {
		userMap[user.ID] = user
	}

	for i, tweet := range timeline.Data {
		author := userMap[tweet.AuthorID]
		sb.WriteString(formatTweetForLLM(i+1, tweet, author))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTweetForLLM formats a single tweet for LLM consumption
func formatTweetForLLM(index int, tweet Tweet, author User) string {
	var sb strings.Builder

	// Header: index, author, date
	authorName := "Unknown"
	authorHandle := ""
	if author.Username != "" {
		authorName = author.Name
		authorHandle = fmt.Sprintf(" (@%s)", author.Username)
	}

	sb.WriteString(fmt.Sprintf("%d. %s%s\n", index, authorName, authorHandle))

	// Tweet text
	sb.WriteString(fmt.Sprintf("   %s\n", tweet.Text))

	// Metadata line
	metaParts := []string{}
	if tweet.CreatedAt != "" {
		metaParts = append(metaParts, fmt.Sprintf("📅 %s", tweet.CreatedAt))
	}
	if tweet.PublicMetrics != nil {
		metrics := tweet.PublicMetrics
		if metrics.LikeCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("❤️ %d", metrics.LikeCount))
		}
		if metrics.ReplyCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("💬 %d", metrics.ReplyCount))
		}
		if metrics.RetweetCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("🔄 %d", metrics.RetweetCount))
		}
	}
	metaParts = append(metaParts, fmt.Sprintf("🆔 %s", tweet.ID))

	if len(metaParts) > 0 {
		sb.WriteString(fmt.Sprintf("   %s\n", strings.Join(metaParts, " | ")))
	}

	return sb.String()
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
