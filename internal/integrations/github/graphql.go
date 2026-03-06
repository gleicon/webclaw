//go:build js && wasm

package github

import (
	"encoding/json"
	"fmt"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// GraphQLClient provides access to the GitHub GraphQL API v4
// This is an optional enhancement over the REST API for complex queries
type GraphQLClient struct {
	endpoint   string
	restClient *Client // Share auth from REST client
}

// GraphQLResponse represents a GraphQL API response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string   `json:"message"`
	Path    []string `json:"path"`
	Type    string   `json:"type"`
}

// Error implements the error interface for GraphQLError
func (e *GraphQLError) Error() string {
	if len(e.Path) > 0 {
		return fmt.Sprintf("GraphQL error at %v: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("GraphQL error: %s", e.Message)
}

// GraphQL creates a new GraphQL client using the same auth as the REST client
func (c *Client) GraphQL() *GraphQLClient {
	return &GraphQLClient{
		endpoint:   "https://api.github.com/graphql",
		restClient: c,
	}
}

// Query executes a GraphQL query with optional variables
func (g *GraphQLClient) Query(query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	// Get token from REST client
	token, err := g.restClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("GitHub not connected: %w", err)
	}

	// Build request body
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	// Make request
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github.v4+json",
		"Content-Type":  "application/json",
		"User-Agent":    "webclaw/1.0",
	}

	opts := jsbridge.FetchOptions{
		Method:  "POST",
		Headers: headers,
		Body:    string(jsonBody),
	}

	resp, err := jsbridge.Fetch(g.endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("GraphQL request failed: %w", err)
	}

	// Update rate limit from headers (GraphQL shares rate limit with REST)
	g.restClient.updateRateLimit(resp.Headers)

	if resp.Status != 200 {
		return nil, fmt.Errorf("GraphQL API returned status %d: %s", resp.Status, string(resp.Body))
	}

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.Unmarshal(resp.Body, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return &gqlResp, &gqlResp.Errors[0]
	}

	return &gqlResp, nil
}

// QueryWithData executes a GraphQL query and unmarshals the data into the provided struct
func (g *GraphQLClient) QueryWithData(query string, variables map[string]interface{}, data interface{}) error {
	resp, err := g.Query(query, variables)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resp.Data, data); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL data: %w", err)
	}

	return nil
}

// Example queries for future tool implementation:

// GetPullRequestWithDetailsQuery is an example complex query for PR details
const GetPullRequestWithDetailsQuery = `
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
      number
      title
      body
      state
      author {
        login
      }
      headRefName
      baseRefName
      isDraft
      createdAt
      updatedAt
      url
      comments(first: 100) {
        nodes {
          id
          body
          author {
            login
          }
          createdAt
        }
        totalCount
      }
      reviews(first: 100) {
        nodes {
          id
          state
          body
          author {
            login
          }
        }
        totalCount
      }
      files(first: 100) {
        nodes {
          path
          additions
          deletions
          changeType
        }
        totalCount
      }
    }
  }
}
`

// GetRepositoryWithIssuesQuery is an example query for repository with issues
const GetRepositoryWithIssuesQuery = `
query($owner: String!, $repo: String!, $states: [IssueState!]) {
  repository(owner: $owner, name: $repo) {
    id
    name
    description
    url
    issues(first: 50, states: $states, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        id
        number
        title
        body
        state
        author {
          login
        }
        createdAt
        updatedAt
        url
        labels(first: 10) {
          nodes {
            name
            color
          }
        }
        assignees(first: 10) {
          nodes {
            login
          }
        }
      }
      totalCount
    }
    pullRequests(first: 50, states: [OPEN], orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        id
        number
        title
        state
        author {
          login
        }
        url
      }
      totalCount
    }
  }
}
`

// GetUserContributionsQuery is an example query for user contributions
const GetUserContributionsQuery = `
query($login: String!) {
  user(login: $login) {
    id
    login
    name
    bio
    url
    repositories(first: 100, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        id
        name
        description
        url
        stargazerCount
        forkCount
        primaryLanguage {
          name
        }
      }
      totalCount
    }
    contributionsCollection {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionCalendar {
        totalContributions
        weeks {
          contributionDays {
            contributionCount
            date
          }
        }
      }
    }
  }
}
`

// SearchCodeQuery is an example query for code search
const SearchCodeQuery = `
query($query: String!, $first: Int!) {
  search(query: $query, type: CODE, first: $first) {
    codeCount
    edges {
      node {
        ... on Blob {
          text
          repository {
            nameWithOwner
            url
          }
          path
        }
      }
      textMatches {
        fragment
        property
      }
    }
  }
}
`

// Note: GraphQL tools are not yet exposed as WebClaw tools.
// They provide the foundation for complex queries that can be
// added as tools in the future. Benefits of GraphQL over REST:
//
// 1. Single request for complex data (PR + reviews + comments + files)
// 2. Precise field selection (less data transfer)
// 3. Better for analytics queries (user contributions, repo stats)
// 4. Strong typing and introspection
//
// To add a GraphQL-based tool:
// 1. Define the query (like examples above)
// 2. Create result structs matching the GraphQL schema
// 3. Implement tool Execute function using QueryWithData()
// 4. Add to tools.go with NewXTool(client) constructor
