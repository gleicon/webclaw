# WebClaw Integrations Guide

WebClaw now includes **23+ powerful integrations** across social media, email, calendar, developer tools, and knowledge management. This guide helps you get started with each integration.

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [Social Media (Twitter/X)](#social-media-twitterx)
- [Email & Calendar (Google)](#email--calendar-google)
- [Developer Tools (GitHub)](#developer-tools-github)
- [Knowledge Base (Notion)](#knowledge-base-notion)
- [Troubleshooting](#troubleshooting)

---

## Overview

WebClaw's integrations work entirely in your browser using OAuth 2.0 PKCE authentication. Your tokens are securely encrypted and stored locally - no data leaves your machine except API calls to the services themselves.

### Available Integrations

| Category | Tools | Description |
|----------|-------|-------------|
| 🐦 **Social Media** | 4 tools | Twitter/X posting, replies, search, timeline |
| 📧 **Email** | 4 tools | Gmail send, read, list, search |
| 📅 **Calendar** | 4 tools | Google Calendar create, list, delete, today view |
| 🔧 **Developer Tools** | 5 tools | GitHub issues, PRs, code search, comments |
| 📝 **Knowledge Base** | 5 tools | Notion databases, pages, search |

**Total: 23 tools** (+ existing web, file, and memory tools)

---

## Getting Started

### 1. Connect Your Accounts

Before using any integration, you need to connect your accounts:

1. Open WebClaw
2. Click on the **Settings** tab
3. Scroll down to **Connected Services**
4. Click **Connect** next to any service you want to use
5. A popup will open for OAuth authentication
6. Authorize WebClaw in the popup
7. Your token will be securely stored

### 2. Verify Connection

After connecting, you'll see a green "Connected" status next to the service.

### 3. Start Using Tools

You can now use natural language or direct tool commands:

```
"Post a tweet saying Hello world!"
"Check my Gmail inbox"
"What's on my calendar today?"
```

Or use direct tool syntax:

```
twitter_post text="Hello world!"
gmail_list count=5
calendar_today
```

---

## Social Media (Twitter/X)

**Tools:** `twitter_post`, `twitter_reply`, `twitter_search`, `twitter_timeline`

### Setup

1. Connect Twitter in Settings → Connected Services
2. Authorize WebClaw to post tweets and read your timeline
3. Required scopes: `tweet.read`, `tweet.write`, `users.read`

### Examples

**Post a tweet:**
```
twitter_post text="Just shipped a new feature with WebClaw! 🚀"
```

**Reply to a tweet:**
```
twitter_reply tweet_id="1234567890" text="Great insights! Thanks for sharing."
```

**Search tweets:**
```
twitter_search query="#AI news" count=10
```

**View your timeline:**
```
twitter_timeline count=20
```

### Tips

- Tweets are limited to 280 characters
- You can use Twitter search operators in `twitter_search`
- Rate limit: 300 requests per 15-minute window

---

## Email & Calendar (Google)

**Tools:** `gmail_send`, `gmail_list`, `gmail_read`, `gmail_search`, `calendar_list`, `calendar_create`, `calendar_delete`, `calendar_today`

### Setup

1. Connect Google in Settings → Connected Services
2. Authorize WebClaw for Gmail and Calendar access
3. Required scopes: `gmail.modify`, `calendar.events`

### Gmail Examples

**Send an email:**
```
gmail_send to="friend@example.com" subject="Hello" body="How are you doing?"
```

**List recent emails:**
```
gmail_list count=10 label="INBOX"
```

**Read a specific email:**
```
gmail_read message_id="abc123xyz"
```

**Search emails:**
```
gmail_search query="from:boss@company.com subject:urgent" count=5
```

#### Gmail Search Query Syntax

- `from:someone@example.com` - From specific sender
- `to:someone@example.com` - To specific recipient
- `subject:hello` - Subject contains
- `has:attachment` - Has attachments
- `is:unread` - Unread messages
- `after:2024/01/01` - After date
- `"exact phrase"` - Exact phrase match

### Calendar Examples

**List upcoming events:**
```
calendar_list days=7 count=10
```

**Create an event:**
```
calendar_create title="Team Meeting" start_time="2024-01-15T14:00:00Z" duration_minutes=60
```

**Today's events:**
```
calendar_today
```

**Delete an event:**
```
calendar_delete event_id="event_123456"
```

#### Time Format Tips

The `start_time` parameter accepts:
- RFC 3339 format: `2024-01-15T14:00:00Z`
- ISO 8601: `2024-01-15 14:00`
- Natural language: `tomorrow 2pm`, `next Monday 10am`

---

## Developer Tools (GitHub)

**Tools:** `github_list_issues`, `github_list_prs`, `github_create_issue`, `github_search_code`, `github_comment`

### Setup

1. Connect GitHub in Settings → Connected Services
2. Authorize WebClaw for repository access
3. Required scopes: `repo`, `issues`, `pull_requests`, `read:user`

### Examples

**List issues assigned to you:**
```
github_list_issues state="open" count=20
```

**List issues in a specific repo:**
```
github_list_issues owner="gleicon" repo="webclaw" state="open" count=10
```

**List pull requests:**
```
github_list_prs owner="gleicon" repo="webclaw" state="open"
```

**Create an issue:**
```
github_create_issue owner="gleicon" repo="webclaw" title="Bug: Login fails on mobile" body="Steps to reproduce:\n1. Open app\n2. Click login" labels=["bug", "mobile"]
```

**Search code:**
```
github_search_code query="repo:gleicon/webclaw TODO language:go"
```

**Add a comment:**
```
github_comment owner="gleicon" repo="webclaw" number=42 body="LGTM! Great work on this. 🚀"
```

### GitHub Search Query Syntax

- `repo:owner/name` - Limit to repository
- `language:go` - Filter by language
- `path:internal/` - Search in path
- `extension:go` - File extension
- `"search term"` - Exact phrase
- `TODO` - Search for literal

### Tips

- Works with both issues and PRs (PRs are a type of issue)
- Rate limit: 5,000 requests per hour for OAuth apps
- You can filter by state: `open`, `closed`, or `all`

---

## Knowledge Base (Notion)

**Tools:** `notion_list_databases`, `notion_query`, `notion_read`, `notion_update`, `notion_search`

### Setup

1. Connect Notion in Settings → Connected Services
2. Select the workspace(s) to grant access to
3. Required: Integration token with appropriate workspace permissions

### Examples

**List your databases:**
```
notion_list_databases
```

**Query a database:**
```
notion_query database_id="Tasks" filter_property="Status" filter_value="Not Started" limit=20
```

**Query by database name (fuzzy match):**
```
notion_query database_id="My Tasks" sort_by="Created" limit=10
```

**Read a page:**
```
notion_read page_id="abc123-def456" include_content=true
```

**Update page properties:**
```
notion_update page_id="abc123-def456" properties={"Status": "Done", "Priority": "High"}
```

**Search pages and databases:**
```
notion_search query="project roadmap" limit=10
```

### Database Query Tips

- Use database name instead of ID for convenience
- Common filter properties: `Status`, `Priority`, `Tags`, `Assigned To`
- Property values are automatically converted to appropriate types
- Supports sorting by any property

### Notion Rate Limits

- ~3 requests per second average
- Rate limit errors automatically retry after 1 second
- Database schemas are cached for 5 minutes

---

## Troubleshooting

### "Please connect [service] in Settings"

This means the OAuth token isn't available. Solution:
1. Go to Settings → Connected Services
2. Click Connect for the service
3. Complete the OAuth flow

### "Rate limited by [service]"

You've hit the API rate limit. Wait a few minutes and try again.

- Twitter: 300 requests per 15 minutes
- Gmail: 1 billion quota units per day (very high)
- GitHub: 5,000 requests per hour
- Notion: ~3 requests per second

### "Invalid credentials" or "Token expired"

Your OAuth token may have expired or been revoked. Solution:
1. Go to Settings → Connected Services
2. Click Disconnect for the service
3. Click Connect again to re-authorize

### Popup blocked

If the OAuth popup doesn't open:
1. Check your browser's popup blocker
2. Look for blocked popup icon in the address bar
3. Allow popups for this site
4. Try connecting again

### Tool not working as expected

1. Use `help tool="tool_name"` to see the correct syntax
2. Check that all required parameters are provided
3. Verify your account has the necessary permissions
4. Try the natural language version of your request

---

## Security & Privacy

- **OAuth tokens** are encrypted with AES-256-GCM using your browser's Web Crypto API
- **No server** - All authentication happens in your browser
- **Tokens are never sent** to any WebClaw server (there is no server!)
- **Local storage only** - Everything stays on your device
- **You control access** - Disconnect anytime in Settings

---

## Advanced Usage

### Combining Tools

You can chain multiple tool calls in a single conversation:

```
"Check my calendar for today, then post on Twitter that I'm available for meetings"
```

### Using Results

WebClaw remembers context across messages:

```
User: "Search Gmail for emails from boss"
Agent: [Returns email list]
User: "Read the first one"
Agent: [Reads that specific email]
```

### Natural Language vs Direct Commands

Both work! Use what feels natural:

- Natural: "Schedule a meeting with the team tomorrow at 2pm"
- Direct: `calendar_create title="Team Meeting" start_time="tomorrow 2pm"`

---

## Feedback & Support

Having issues with integrations? Check the help tool:

```
help
calendar_create help
github_create_issue help
```

Or look at the tool's documentation in Settings → Connected Services.

---

**Happy automating! 🤖✨**
