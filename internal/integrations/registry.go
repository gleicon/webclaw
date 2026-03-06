//go:build js && wasm

package integrations

import (
	"github.com/gleicon/webclaw/internal/integrations/github"
	"github.com/gleicon/webclaw/internal/integrations/google"
	"github.com/gleicon/webclaw/internal/integrations/google/calendar"
	"github.com/gleicon/webclaw/internal/integrations/google/gmail"
	"github.com/gleicon/webclaw/internal/oauth"
	"github.com/gleicon/webclaw/internal/tools"
)

// RegisterGitHubTools registers all GitHub integration tools with the registry
// This includes issues, pull requests, code search, and commenting tools
func RegisterGitHubTools(registry *tools.Registry, oauthMgr *oauth.OAuthManager) {
	// Create GitHub API client
	githubClient := github.NewClient(oauthMgr)

	// Register GitHub tools
	registry.Register(github.NewListIssuesTool(githubClient))
	registry.Register(github.NewListPRsTool(githubClient))
	registry.Register(github.NewCreateIssueTool(githubClient))
	registry.Register(github.NewSearchCodeTool(githubClient))
	registry.Register(github.NewCommentTool(githubClient))
}

// RegisterGoogleTools registers all Google integration tools with the registry
// This includes Gmail (send, list, read, search) and Calendar (list, create, delete, today) tools
func RegisterGoogleTools(registry *tools.Registry, oauthMgr *oauth.OAuthManager) {
	// Create the base Google API client
	baseClient := google.NewClient(oauthMgr)

	// Create Gmail client and register tools
	gmailClient := gmail.NewClient(baseClient)
	registry.Register(gmail.NewSendTool(gmailClient))
	registry.Register(gmail.NewListTool(gmailClient))
	registry.Register(gmail.NewReadTool(gmailClient))
	registry.Register(gmail.NewSearchTool(gmailClient))

	// Create Calendar client and register tools
	calendarClient := calendar.NewClient(baseClient)
	registry.Register(calendar.NewListTool(calendarClient))
	registry.Register(calendar.NewCreateTool(calendarClient))
	registry.Register(calendar.NewDeleteTool(calendarClient))
	registry.Register(calendar.NewTodayTool(calendarClient))
}
