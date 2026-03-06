---
phase: 09-social-integrations
plan: 04
subsystem: integrations
completed: "2026-03-06T00:10:00Z"
duration: 6
tags: [github, oauth, api, graphql, rest]
tech-stack:
  added: [GitHub REST API v3, GraphQL v4, OAuth 2.0]
  patterns: [authenticated client, tool factory, pagination, rate limiting]
key-files:
  created:
    - internal/integrations/github/types.go
    - internal/integrations/github/types_test.go
    - internal/integrations/github/client.go
    - internal/integrations/github/client_test.go
    - internal/integrations/github/tools.go
    - internal/integrations/github/tools_test.go
    - internal/integrations/github/graphql.go
    - internal/integrations/github/graphql_test.go
  modified:
    - internal/integrations/registry.go
metrics:
  commits: 5
  test-files: 4
  tools-created: 5
  source-files: 4
---

# Phase 09 Plan 04: GitHub Integration Summary

## Overview

Full GitHub API integration with OAuth-authenticated tools for managing issues, pull requests, code search, and comments. Implements both REST API v3 for immediate use and GraphQL v4 foundation for future complex queries.

## What Was Built

### GitHub API Types (`internal/integrations/github/types.go`)
- **Issue** - Complete struct with title, body, state, labels, assignees, comments
- **PullRequest** - Extends Issue with Head/Base branch info, draft status
- **Repository** - Name, description, owner, visibility, issue counts
- **User** - Login, profile URL, type (User/Organization)
- **Label** - Name, color, description
- **Comment** - Body, author, timestamps
- **CodeSearchResult** - Search results with text matching snippets
- **GitHubError** - Structured error responses from GitHub API

### GitHub API Client (`internal/integrations/github/client.go`)
- **Authenticated requests** - Bearer token from OAuth manager
- **Core methods:**
  - `GetIssues()` - List issues for user or repository
  - `GetPullRequests()` - List PRs with branch information
  - `CreateIssue()` - Create issues with labels support
  - `GetRepository()` - Fetch repository metadata
  - `SearchCode()` - Search code across repositories
  - `CreateComment()` - Add comments to issues/PRs
  - `ListUserRepos()` - List user's repositories
- **Rate limit tracking** - X-RateLimit-* headers parsed and stored
- **Error handling** - GitHubError parsing with documentation URLs

### GitHub Tools (`internal/integrations/github/tools.go`)
| Tool | Description | Required Params |
|------|-------------|-----------------|
| `github_list_issues` | List issues (yours or repo-specific) | owner+repo (optional) |
| `github_list_prs` | List pull requests | owner, repo |
| `github_create_issue` | Create new issue | owner, repo, title |
| `github_search_code` | Search code | query |
| `github_comment` | Comment on issue/PR | owner, repo, number, body |

**Tool features:**
- OAuth connectivity checks with helpful "Please connect GitHub in Settings" messages
- Input validation (required params, enum values for state)
- Parameter extraction helpers (string, int, string slice)
- Formatted output with issue/PR numbers, URLs, labels, assignees
- Markdown code snippets in search results

### GraphQL Foundation (`internal/integrations/github/graphql.go`)
- **GraphQLClient** - Separate client sharing REST authentication
- `Query()` - Execute GraphQL queries with variables
- `QueryWithData()` - Execute and unmarshal into struct
- **Example queries** for future tools:
  - `GetPullRequestWithDetailsQuery` - PR + reviews + comments + files changed
  - `GetRepositoryWithIssuesQuery` - Repo with issues and PRs
  - `GetUserContributionsQuery` - User profile with contribution stats
  - `SearchCodeQuery` - Code search via GraphQL

### Registry Integration (`internal/integrations/registry.go`)
- `RegisterGitHubTools(registry, oauthMgr)` - Registers all 5 GitHub tools
- Clean separation from other integrations (Google, Twitter, Notion)

## GitHub REST API v3 Reference

### Endpoints Used
```
GET  /user/issues              - User's issues across repos
GET  /repos/{owner}/{repo}/issues
POST /repos/{owner}/{repo}/issues
GET  /repos/{owner}/{repo}/pulls
GET  /repos/{owner}/{repo}
GET  /search/code
POST /repos/{owner}/{repo}/issues/{number}/comments
GET  /user/repos
```

### Authentication Headers
```
Authorization: Bearer {oauth_token}
Accept: application/vnd.github.v3+json
User-Agent: webclaw/1.0
```

### Rate Limits
- **OAuth apps:** 5,000 requests/hour
- **Headers tracked:** X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
- **Error on limit:** Clear message with time until reset

## GitHub Search Syntax

GitHub code search supports powerful qualifiers:

| Qualifier | Example | Description |
|-----------|---------|-------------|
| `repo:` | `repo:owner/name` | Limit to repository |
| `language:` | `language:go` | Filter by language |
| `path:` | `path:internal/` | Search in path |
| `extension:` | `extension:go` | File extension |
| `"phrase"` | `"error handling"` | Exact phrase |
| `literal` | `TODO` | Search literal |

**Example queries:**
```
repo:gleicon/webclaw extension:go TODO
language:javascript path:src/ fetchAPI
"user agent" repo:facebook/react
```

## Tool Usage Examples

### List My Issues
```
User: "What issues are assigned to me?"
→ github_list_issues (no owner/repo)
→ Shows open issues assigned to you across all repos
```

### List Repo Issues
```
User: "Show issues in gleicon/webclaw"
→ github_list_issues owner="gleicon" repo="webclaw"
→ Shows open issues in that repo
```

### List Pull Requests
```
User: "Show open PRs in webclaw"
→ github_list_prs owner="gleicon" repo="webclaw"
→ Shows open PRs with branch info (feature → main)
```

### Create Issue
```
User: "Create issue in webclaw: Fix login bug"
→ github_create_issue owner="gleicon" repo="webclaw" title="Fix login bug"
→ Returns issue URL
```

### Search Code
```
User: "Search for TODO in gleicon/webclaw"
→ github_search_code query="repo:gleicon/webclaw TODO"
→ Shows files with TODO comments
```

### Add Comment
```
User: "Comment on issue 42 in webclaw: LGTM!"
→ github_comment owner="gleicon" repo="webclaw" number=42 body="LGTM!"
→ Adds comment, returns comment URL
```

## OAuth Scope Requirements

Required OAuth scopes for GitHub integration:

| Scope | Purpose |
|-------|---------|
| `repo` | Access to private repositories |
| `public_repo` | Access to public repositories |
| `read:user` | Read user profile |
| `read:org` | Read organization membership |

**Note:** Scopes are configured in the GitHub OAuth app settings, requested during the OAuth flow in 09-01.

## Testing

All components have comprehensive tests:
- **types_test.go** - JSON marshaling/unmarshaling, field tags
- **client_test.go** - Rate limit tracking, error parsing, method signatures
- **tools_test.go** - Tool creation, parameter extraction, formatting helpers
- **graphql_test.go** - Client creation, error handling, query syntax

Run tests:
```bash
go test -v ./internal/integrations/github/...
```

## Deviation from Plan

**None** - Plan executed exactly as written.

All 5 tasks completed as specified:
1. ✅ Types defined with correct JSON tags
2. ✅ Client built with all core methods
3. ✅ Tools created with proper schemas
4. ✅ Tools registered and tested
5. ✅ GraphQL foundation implemented

## Success Criteria Verification

| Criterion | Status | Verification |
|-----------|--------|------------|
| List assigned issues | ✅ | `github_list_issues` without owner/repo |
| List repo issues | ✅ | `github_list_issues` with owner+repo |
| List open PRs | ✅ | `github_list_prs` with owner+repo |
| Create issue | ✅ | `github_create_issue` with labels support |
| Search code | ✅ | `github_search_code` with GitHub syntax |
| Add comment | ✅ | `github_comment` on issues/PRs |
| Graceful auth failure | ✅ | "Please connect GitHub in Settings" message |
| OAuth token usage | ✅ | Bearer token from `oauthManager.GetToken()` |
| Rate limit tracking | ✅ | Headers parsed, error on limit exceeded |
| Test coverage | ✅ | 4 test files covering all components |
| Tools registered | ✅ | `RegisterGitHubTools()` adds all 5 tools |

## Architecture Notes

### Why Both REST and GraphQL?

**REST API (Primary):**
- Simpler, well-documented
- All current tools use REST
- Stable and predictable
- Better error messages for common operations

**GraphQL (Foundation):**
- Single request for complex data (PR + reviews + comments + files)
- Precise field selection (less data transfer)
- Better for analytics queries
- Future tool expansion

### Design Decisions

1. **Shared OAuth** - Both REST and GraphQL use same token from OAuthManager
2. **Separate clients** - REST and GraphQL clients are separate for clarity
3. **Rate limit sharing** - GitHub's rate limit is shared across both APIs
4. **No pagination tools yet** - Pagination handled internally, could expose if needed
5. **No PR creation** - Complex (needs branch, commits, push) - deferred

## Next Steps (Future Plans)

Potential enhancements for 09-06 or later:
- GraphQL-based tools for complex queries
- PR creation (complex workflow)
- PR merge functionality
- Issue/PR update operations
- Repository creation
- Webhook management
- Actions/workflow triggers

## Files Summary

```
internal/integrations/github/
├── types.go           # 210 lines - All GitHub API types
├── types_test.go      # 180 lines - Type marshaling tests
├── client.go          # 314 lines - REST API client
├── client_test.go     # 200 lines - Client tests
├── tools.go           # 624 lines - 5 WebClaw tools
├── tools_test.go      # 458 lines - Tool tests
├── graphql.go         # 245 lines - GraphQL foundation
└── graphql_test.go    # 165 lines - GraphQL tests

internal/integrations/
└── registry.go        # +21 lines - RegisterGitHubTools()

Total: ~2,400 lines of code + tests
```

## References

- GitHub REST API v3: https://docs.github.com/en/rest
- GitHub GraphQL API v4: https://docs.github.com/en/graphql
- GitHub Search Syntax: https://docs.github.com/en/search-github/searching-on-github
- OAuth Scopes: https://docs.github.com/en/developers/apps/scopes-for-oauth-apps
