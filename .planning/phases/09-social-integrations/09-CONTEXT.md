# Phase 9: Social & Productivity Integrations - Context

**Gathered:** 2026-03-05  
**Status:** Ready for planning  
**Source:** OpenClaw parity analysis + User requirements

---

## Domain

### Phase Boundary

**What this phase delivers:**
OAuth-based integrations with popular social media and productivity APIs that work entirely in the browser. This phase enables WebClaw to perform real-world actions like posting to Twitter, managing Gmail inbox, checking Google Calendar, reviewing GitHub PRs, and querying Notion databases.

**What is NOT in scope:**
- Chat provider integrations (WhatsApp, Telegram, Discord) - require persistent server connections
- Native app integrations (Apple Notes, Things 3) - require OS-level access
- Smart home controls (Philips Hue, Sonos) - require local network access
- Background automation (cron jobs) - requires persistent process

**Key constraint:** All integrations must work in a browser-only environment using public APIs with OAuth 2.0 authentication.

---

## Implementation Decisions

### Locked Decisions (Must implement this way)

**OAuth Architecture:**
- **Flow:** PKCE (Proof Key for Code Exchange) for all OAuth providers
- **Reason:** Required for browser-based apps where client secret can't be protected
- **Implementation:** Popup window flow (not redirect) for better UX
- **Storage:** Encrypted IndexedDB using Web Crypto API (same as API keys)
- **Refresh:** Automatic token refresh before expiration

**Integration Pattern:**
- **Architecture:** Same as existing WebClaw tools (web_fetch, web_search)
- **Pattern:** Tool registered in ToolRegistry, called by AgentLoop via tool_use
- **Authentication:** OAuth token retrieved at tool execution time
- **Error handling:** Graceful degradation - if not connected, prompt user to connect

**Supported Integrations (Priority Order):**
1. **Twitter/X** - POST tweets, GET timeline, search tweets
2. **Google** - Gmail (send/read), Calendar (events/scheduling)
3. **GitHub** - Issues, PRs, repos, comments
4. **Notion** - Databases, pages, queries
5. **Spotify** - Playback control, playlists (nice-to-have)

**UI/UX:**
- **Settings panel:** List integrations with "Connect" buttons
- **Connection flow:** OAuth popup → authenticate → store token → show connected status
- **Tool usage:** Agent automatically uses tools when LLM detects intent (e.g., "post to Twitter")
- **Feedback:** Show tool execution in Tool Activity panel (same as web_fetch, etc.)

### Claude's Discretion

**Technical Choices:**
- OAuth library: Can use standard Go oauth2 package or implement manually (recommend: implement manually for better WASM compatibility)
- HTTP client: Use existing JS fetch bridge or pure Go net/http via WASM (recommend: JS fetch bridge for CORS handling)
- Token refresh: Proactive (before expiration) or reactive (on 401) - recommend proactive
- UI style: Follow existing WebClaw dark theme with Tailwind CSS

**API Implementation Order:**
- Can implement in any order within Priority 1, but recommend: Twitter first (simplest), then Google (more complex scopes), then GitHub, then Notion

**Scope Creep Guards:**
- Focus on read/write operations only, not complex workflows
- Skip features requiring webhooks (need server)
- Skip features requiring background processing
- Build for "single user personal use" not "enterprise team features"

### Implementation Approach

**Wave 1 (Foundation):**
- OAuth infrastructure (PKCE, token storage, refresh)
- JavaScript bridge for popup flow
- Generic OAuth provider interface

**Wave 2 (High Value):**
- Twitter integration (tweet, timeline, search)
- Google integration (Gmail + Calendar)

**Wave 3 (Developer Tools):**
- GitHub integration (issues, PRs, repos)
- Notion integration (databases, pages)

**Wave 4 (Polish):**
- Spotify (if time permits)
- Comprehensive testing
- Documentation

---

## Specific Ideas

### Twitter/X Use Cases

1. **Post tweet:** "Tweet: Just shipped a new feature!"
2. **Reply to mention:** "Reply to @user: Thanks for the feedback!"
3. **Check timeline:** "What's happening on Twitter?"
4. **Search:** "Search Twitter for #AI news"

**API endpoints:**
- POST /2/tweets - Create tweet
- GET /2/users/:id/timelines/reverse_chronological - Get timeline
- GET /2/tweets/search/recent - Search tweets

**Scopes:** tweet.read, tweet.write, users.read, offline.access

### Gmail Use Cases

1. **Check inbox:** "Do I have any important emails?"
2. **Send email:** "Send email to boss: Running late, be there in 10"
3. **Summarize:** "Summarize my unread emails"
4. **Find:** "Find the email from John about the project"

**API endpoints:**
- GET /gmail/v1/users/me/messages - List messages
- GET /gmail/v1/users/me/messages/:id - Get message
- POST /gmail/v1/users/me/messages/send - Send message

**Scopes:** https://www.googleapis.com/auth/gmail.modify

### Google Calendar Use Cases

1. **Check schedule:** "What's on my calendar today?"
2. **Create event:** "Schedule meeting with team tomorrow at 2pm"
3. **Find time:** "When am I free this week?"

**API endpoints:**
- GET /calendar/v3/calendars/primary/events - List events
- POST /calendar/v3/calendars/primary/events - Create event

**Scopes:** https://www.googleapis.com/auth/calendar.events

### GitHub Use Cases

1. **Check issues:** "What issues are assigned to me?"
2. **Review PR:** "Show me open PRs in repo webclaw"
3. **Create issue:** "Create issue: Fix bug in OAuth flow"
4. **Code search:** "Search for TODO comments in repo"

**API endpoints:**
- GET /repos/:owner/:repo/issues - List issues
- GET /repos/:owner/:repo/pulls - List PRs
- POST /repos/:owner/:repo/issues - Create issue
- GraphQL v4 for complex queries (optional)

**Scopes:** repo, issues, pull_requests, read:user

### Notion Use Cases

1. **Query database:** "Show me tasks in Notion"
2. **Update page:** "Add to my notes: Meeting scheduled for Friday"
3. **Search:** "Find documents about OAuth"

**API endpoints:**
- POST /v1/databases/:id/query - Query database
- PATCH /v1/pages/:id - Update page
- POST /v1/search - Search

**Scopes:** Determined by integration capabilities

---

## Deferred Ideas

**Not in Phase 9 (Future Phases):**

1. **Spotify** - Nice-to-have, lower priority than productivity tools
2. **Weather** - Simple API key integration (no OAuth needed, can be separate tool)
3. **Image Generation** - Can use existing AI providers, just needs tool wrapper
4. **Trello** - Similar to Notion but less popular
5. **Discord (basic)** - Read-only via REST API (webhooks need server)
6. **Slack (basic)** - Same limitation as Discord

**Reason for deferral:** Lower impact or blocked by technical constraints (need server for real-time features)

---

## Success Criteria

Phase 9 will be considered complete when:

1. ✅ User can connect OAuth account for at least 3 integrations (Twitter, Google, GitHub)
2. ✅ User can disconnect and revoke tokens
3. ✅ Agent can successfully post tweet, send email, create calendar event, list GitHub issues
4. ✅ Tokens are encrypted and survive page reload
5. ✅ Token refresh works automatically (no user re-auth needed within refresh window)
6. ✅ Clear error messages when not connected ("Please connect Twitter in Settings")
7. ✅ UI shows connection status and allows management
8. ✅ All tools follow WebClaw tool patterns (registry, execution, error handling)

---

## Open Questions

**Q: Do we need a privacy policy/terms of service for OAuth apps?**
A: For Twitter, GitHub, Google - yes. Recommend creating simple privacy policy page stating "WebClaw stores tokens locally in your browser only."

**Q: Should we cache API responses?**
A: Yes, brief cache (5 minutes) for read operations to reduce API calls and improve speed.

**Q: What about rate limiting?**
A: Implement exponential backoff. Show user-friendly error: "Rate limited by Twitter. Please try again in X minutes."

---

## Notes

**Key Differentiator:**
WebClaw's advantage is instant accessibility. While OpenClaw has more integrations overall, WebClaw requires zero setup - just open URL, connect accounts, start using. This is the core value prop to emphasize.

**User Onboarding:**
- Default welcome message should mention available integrations
- First-time user should see "Connect your accounts" in Settings
- Example prompts: "Try saying 'Post to Twitter: Hello world!'" or "Check my email"

---

*Phase: 09-social-integrations*  
*Context gathered: 2026-03-05 via OpenClaw parity analysis*  
*Next: Create PLAN.md files for 09-01 through 09-05*
