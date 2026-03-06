//go:build js && wasm

package github

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphQLClientCreation(t *testing.T) {
	// Test that GraphQL() method exists and returns a client
	// We can't fully test without a real Client, but we can verify the API

	client := &Client{}
	gqlClient := client.GraphQL()

	if gqlClient == nil {
		t.Fatal("GraphQL() should return a client, not nil")
	}

	if gqlClient.endpoint != "https://api.github.com/graphql" {
		t.Errorf("Expected endpoint to be https://api.github.com/graphql, got %s", gqlClient.endpoint)
	}

	if gqlClient.restClient != client {
		t.Error("GraphQL client should reference the REST client")
	}
}

func TestGraphQLError(t *testing.T) {
	err := &GraphQLError{
		Message: "Field 'xyz' doesn't exist on type 'Query'",
		Path:    []string{"repository", "xyz"},
		Type:    "INVALID_FIELD",
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("GraphQLError.Error() should return a non-empty string")
	}

	if !strings.Contains(errMsg, "Field 'xyz'") {
		t.Errorf("Error message should contain the error message, got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "repository") {
		t.Errorf("Error message should contain the path, got: %s", errMsg)
	}
}

func TestGraphQLErrorNoPath(t *testing.T) {
	err := &GraphQLError{
		Message: "Authentication required",
		Type:    "AUTH_REQUIRED",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Authentication required") {
		t.Errorf("Error message should contain the error message, got: %s", errMsg)
	}
}

func TestGraphQLResponse(t *testing.T) {
	// Test response struct fields
	resp := &GraphQLResponse{
		Data: json.RawMessage(`{"repository":{"name":"webclaw"}}`),
		Errors: []GraphQLError{
			{
				Message: "Something went wrong",
				Path:    []string{"user"},
			},
		},
	}

	if resp.Data == nil {
		t.Error("Response should have data")
	}

	if len(resp.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(resp.Errors))
	}

	if resp.Errors[0].Message != "Something went wrong" {
		t.Errorf("Expected error message 'Something went wrong', got %s", resp.Errors[0].Message)
	}
}

func TestExampleQueries(t *testing.T) {
	// Test that example queries are defined and non-empty
	queries := map[string]string{
		"GetPullRequestWithDetailsQuery": GetPullRequestWithDetailsQuery,
		"GetRepositoryWithIssuesQuery":   GetRepositoryWithIssuesQuery,
		"GetUserContributionsQuery":      GetUserContributionsQuery,
		"SearchCodeQuery":                SearchCodeQuery,
	}

	for name, query := range queries {
		if query == "" {
			t.Errorf("Query %s should not be empty", name)
		}
		if !strings.Contains(query, "query") {
			t.Errorf("Query %s should be a valid GraphQL query", name)
		}
	}
}

func TestQueryMethodExists(t *testing.T) {
	// Compile-time check that Query method exists
	client := &Client{}
	gqlClient := client.GraphQL()

	// This verifies the method signature exists
	var queryFunc func(string, map[string]interface{}) (*GraphQLResponse, error)
	queryFunc = gqlClient.Query
	_ = queryFunc
}

func TestQueryWithDataMethodExists(t *testing.T) {
	// Compile-time check that QueryWithData method exists
	client := &Client{}
	gqlClient := client.GraphQL()

	// This verifies the method signature exists
	var data interface{}
	var queryWithDataFunc func(string, map[string]interface{}, interface{}) error
	queryWithDataFunc = gqlClient.QueryWithData
	_ = queryWithDataFunc
	_ = data
}

// Test that GraphQL queries are valid GraphQL syntax (basic check)
func TestQuerySyntax(t *testing.T) {
	queries := []string{
		GetPullRequestWithDetailsQuery,
		GetRepositoryWithIssuesQuery,
		GetUserContributionsQuery,
		SearchCodeQuery,
	}

	for i, query := range queries {
		// Basic validation - should contain query or mutation keyword
		if !strings.Contains(query, "query") && !strings.Contains(query, "mutation") {
			t.Errorf("Query %d should contain 'query' or 'mutation' keyword", i)
		}

		// Should have opening and closing braces
		if !strings.Contains(query, "{") || !strings.Contains(query, "}") {
			t.Errorf("Query %d should contain braces", i)
		}

		// Should contain field selections (property: or just property)
		if !strings.Contains(query, ":") && !strings.Contains(query, "nodes") {
			t.Errorf("Query %d should contain field selections", i)
		}
	}
}
