//go:build js && wasm

package github

import (
	"encoding/json"
	"testing"
)

func TestTypesJSONMarshaling(t *testing.T) {
	// Test Issue marshaling/unmarshaling
	issue := &Issue{
		ID:        123,
		Number:    42,
		Title:     "Test Issue",
		Body:      "This is a test issue",
		State:     "open",
		User:      &User{ID: 1, Login: "testuser", Type: "User"},
		Assignees: []*User{{ID: 2, Login: "assignee", Type: "User"}},
		Labels:    []*Label{{Name: "bug", Color: "ff0000", Description: "Bug report"}},
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
		HTMLURL:   "https://github.com/test/repo/issues/42",
		Comments:  5,
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal Issue: %v", err)
	}

	var unmarshaled Issue
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Issue: %v", err)
	}

	if unmarshaled.ID != issue.ID {
		t.Errorf("ID mismatch: got %d, want %d", unmarshaled.ID, issue.ID)
	}
	if unmarshaled.Number != issue.Number {
		t.Errorf("Number mismatch: got %d, want %d", unmarshaled.Number, issue.Number)
	}
	if unmarshaled.Title != issue.Title {
		t.Errorf("Title mismatch: got %s, want %s", unmarshaled.Title, issue.Title)
	}
}

func TestPullRequestTypes(t *testing.T) {
	pr := &PullRequest{
		ID:     456,
		Number: 10,
		Title:  "Test PR",
		State:  "open",
		Head: &PRBranch{
			Ref: "feature-branch",
			Sha: "abc123",
			Repo: &Repository{
				Name:     "repo",
				FullName: "owner/repo",
			},
		},
		Base: &PRBranch{
			Ref: "main",
			Sha: "def456",
		},
		Draft: true,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Failed to marshal PullRequest: %v", err)
	}

	var unmarshaled PullRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal PullRequest: %v", err)
	}

	if unmarshaled.Draft != pr.Draft {
		t.Errorf("Draft mismatch: got %v, want %v", unmarshaled.Draft, pr.Draft)
	}
	if unmarshaled.Head.Ref != pr.Head.Ref {
		t.Errorf("Head.Ref mismatch: got %s, want %s", unmarshaled.Head.Ref, pr.Head.Ref)
	}
}

func TestCodeSearchTypes(t *testing.T) {
	result := &CodeSearchResult{
		TotalCount: 100,
		Items: []*CodeSearchItem{
			{
				Name:       "test.go",
				Path:       "src/test.go",
				Sha:        "abc123",
				HTMLURL:    "https://github.com/owner/repo/blob/main/src/test.go",
				Repository: &Repository{FullName: "owner/repo"},
				TextMatches: []TextMatch{
					{
						Fragment: "func Test()",
						Matches: []Match{
							{Text: "Test", Indices: []int{5, 9}},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CodeSearchResult: %v", err)
	}

	var unmarshaled CodeSearchResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CodeSearchResult: %v", err)
	}

	if unmarshaled.TotalCount != result.TotalCount {
		t.Errorf("TotalCount mismatch: got %d, want %d", unmarshaled.TotalCount, result.TotalCount)
	}
	if len(unmarshaled.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(unmarshaled.Items))
	}
	if unmarshaled.Items[0].Name != "test.go" {
		t.Errorf("Item name mismatch: got %s, want test.go", unmarshaled.Items[0].Name)
	}
}

func TestGitHubError(t *testing.T) {
	ghErr := &GitHubError{
		Message:          "Validation Failed",
		DocumentationURL: "https://docs.github.com",
		Errors: []ErrorDetail{
			{
				Resource: "Issue",
				Field:    "title",
				Code:     "missing_field",
				Message:  "Title is required",
			},
		},
	}

	errMsg := ghErr.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
	if !contains(errMsg, "Validation Failed") {
		t.Errorf("Error message should contain 'Validation Failed', got: %s", errMsg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRepositoryType(t *testing.T) {
	repo := &Repository{
		ID:              12345,
		Name:            "webclaw",
		FullName:        "gleicon/webclaw",
		Description:     "A browser-native AI assistant",
		Private:         false,
		HTMLURL:         "https://github.com/gleicon/webclaw",
		Owner:           &User{ID: 1, Login: "gleicon", Type: "User"},
		OpenIssuesCount: 10,
	}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal Repository: %v", err)
	}

	var unmarshaled Repository
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Repository: %v", err)
	}

	if unmarshaled.FullName != repo.FullName {
		t.Errorf("FullName mismatch: got %s, want %s", unmarshaled.FullName, repo.FullName)
	}
	if unmarshaled.OpenIssuesCount != repo.OpenIssuesCount {
		t.Errorf("OpenIssuesCount mismatch: got %d, want %d", unmarshaled.OpenIssuesCount, repo.OpenIssuesCount)
	}
}

func TestCommentType(t *testing.T) {
	comment := &Comment{
		ID:        987,
		Body:      "This is a comment",
		User:      &User{ID: 1, Login: "commenter"},
		CreatedAt: "2024-01-01T00:00:00Z",
		HTMLURL:   "https://github.com/test/repo/issues/42#issuecomment-987",
	}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("Failed to marshal Comment: %v", err)
	}

	var unmarshaled Comment
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Comment: %v", err)
	}

	if unmarshaled.Body != comment.Body {
		t.Errorf("Body mismatch: got %s, want %s", unmarshaled.Body, comment.Body)
	}
}
