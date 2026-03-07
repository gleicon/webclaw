---
status: resolved
phase: 09-social-integrations
source: 09-01-SUMMARY.md, 09-02-SUMMARY.md, 09-03-SUMMARY.md, 09-04-SUMMARY.md, 09-05-SUMMARY.md, 09-06-SUMMARY.md
started: 2026-03-07T00:00:00Z
updated: 2026-03-07T17:00:00Z
automated: true
test_runner: playwright + browser console analysis
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running webclaw server. Start fresh. Server boots, UI loads, chat and Settings accessible. No crash errors.
result: pass
note: Fixed in commit 3653b76 — db.Call('objectStoreNames') changed to db.Get('objectStoreNames') in idb_memory.go. Playwright test confirms no crash errors.
automated: test/phase06-browser-tests/phase09-smoke.spec.js (test 1), test/phase06-browser-tests/diagnose.spec.js

### 2. Connected Services Settings UI
expected: Settings shows Connected Services section with 4 provider cards (Twitter, Google, GitHub, Notion).
result: pass
automated: test/phase06-browser-tests/phase09-smoke.spec.js (test 2)
notes: #connected-services-section and #connected-services-list elements verified present in DOM. Provider cards rendered by JS after webclaw:host-ready event.

### 3. Graceful OAuth Failure Message
expected: Agent responds "Please connect X in Settings" when provider not connected, no crash.
result: pass
automated: code inspection
notes: All 4 integrations have hardcoded graceful failure messages confirmed via grep. Twitter: "Please connect Twitter in Settings first." GitHub: "Please connect GitHub in Settings first." Google/Gmail/Calendar: "Please connect Google in Settings first." Notion: "Please connect Notion in Settings first." No crash path exists.

### 4. Twitter: Post a Tweet
expected: Connect Twitter, ask agent to post a tweet, agent uses twitter_post tool, returns tweet URL.
result: skipped
reason: Requires live Twitter OAuth credentials. Cannot test without an OAuth app registered and a connected account.

### 5. Twitter: Search Tweets
expected: Agent uses twitter_search and returns recent tweets with metrics.
result: skipped
reason: Requires live Twitter OAuth credentials.

### 6. Twitter: View Home Timeline
expected: Agent uses twitter_timeline and returns timeline tweets.
result: skipped
reason: Requires live Twitter OAuth credentials.

### 7. Gmail: List Recent Emails
expected: Connect Google, agent uses gmail_list and returns inbox messages.
result: skipped
reason: Requires live Google OAuth credentials.

### 8. Gmail: Send an Email
expected: Agent uses gmail_send and confirms email sent.
result: skipped
reason: Requires live Google OAuth credentials.

### 9. Gmail: Search Inbox
expected: Agent uses gmail_search with Gmail query syntax.
result: skipped
reason: Requires live Google OAuth credentials.

### 10. Calendar: View Today's Events
expected: Agent uses calendar_today and returns today's events.
result: skipped
reason: Requires live Google OAuth credentials.

### 11. Calendar: Create an Event
expected: Agent uses calendar_create and confirms event created.
result: skipped
reason: Requires live Google OAuth credentials.

### 12. GitHub: List Assigned Issues
expected: Connect GitHub, agent uses github_list_issues and returns open issues.
result: skipped
reason: Requires live GitHub OAuth credentials.

### 13. GitHub: List Pull Requests
expected: Agent uses github_list_prs and returns open PRs with branch info.
result: skipped
reason: Requires live GitHub OAuth credentials.

### 14. GitHub: Create an Issue
expected: Agent uses github_create_issue and returns new issue URL and number.
result: skipped
reason: Requires live GitHub OAuth credentials.

### 15. GitHub: Search Code
expected: Agent uses github_search_code and returns files with code snippets.
result: skipped
reason: Requires live GitHub OAuth credentials.

### 16. Notion: List Databases
expected: Connect Notion, agent uses notion_list_databases and returns database list.
result: skipped
reason: Requires live Notion OAuth credentials.

### 17. Notion: Query a Database
expected: Agent uses notion_query and returns filtered pages.
result: skipped
reason: Requires live Notion OAuth credentials.

### 18. Notion: Read a Page
expected: Agent uses notion_read and returns page content blocks.
result: skipped
reason: Requires live Notion OAuth credentials.

### 19. Notion: Search Workspace
expected: Agent uses notion_search and returns matching pages/databases.
result: skipped
reason: Requires live Notion OAuth credentials.

### 20. OAuth JS API: window.webclaw.oauth exposed
expected: After WASM init, window.webclaw.oauth should have isConnected, getConnectionStatus, initiateConnection, and disconnect methods registered by oauthMgr.RegisterJSExports().
result: pass
note: Fixed in 09-06 gap closure — main.go OAuth goroutine wired on main thread; RegisterOAuthBridge() preserves existing JS oauth object using webclaw.Get('oauth') before adding Go functions. Playwright tests 4-6 confirm all 4 methods present and functional.
automated: test/phase06-browser-tests/phase09-smoke.spec.js (tests 4-6)

## Summary

total: 20
passed: 4
issues: 0
pending: 0
skipped: 16

## Gaps

- truth: "App starts cleanly with no WASM panics or console errors"
  status: resolved
  resolved_by: "commit 3653b76 — db.Call('objectStoreNames') changed to db.Get('objectStoreNames') in idb_memory.go"
  test: 1
  root_cause: "idb_memory.go:60 calls db.Call('objectStoreNames') but objectStoreNames is a DOMStringList property on IDBDatabase, not a callable method. Go WASM panics when trying to .Call() a non-function JS value."
  artifacts:
    - path: "internal/jsbridge/idb_memory.go"
      fix: "Lines 60, 71, 78 changed from db.Call('objectStoreNames') to db.Get('objectStoreNames')"

- truth: "window.webclaw.oauth exposes isConnected, getConnectionStatus, initiateConnection, disconnect after WASM init"
  status: resolved
  resolved_by: "09-06-PLAN.md gap closure — main.go OAuth goroutine wired on main thread; RegisterOAuthBridge() preserves existing JS oauth object"
  test: 20
  root_cause: "Two causes: (1) main thread WASM panic (idb_memory.go bug) prevents the OAuth goroutine from running on main thread. (2) Even if fixed, the OAuth goroutine runs in the worker WASM where js.Global() returns worker scope, making webclaw undefined — RegisterJSExports() early-returns silently. The OAuth manager initialization must happen on the main thread WASM."
  artifacts:
    - path: "cmd/webclaw/main.go"
      fix: "OAuth goroutine with InitOAuthBridge() → NewOAuthManager() → RegisterJSExports() added to main thread init sequence"
    - path: "internal/jsbridge/oauth_bridge.go"
      fix: "RegisterOAuthBridge() uses webclaw.Get('oauth') to preserve existing JS-side object (openPopup) before adding Go functions (handleCallback, exchangeCode)"
