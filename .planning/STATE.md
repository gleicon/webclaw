---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: completed
stopped_at: Completed 09.1-03-PLAN.md (pending checkpoint)
last_updated: "2026-03-08T18:35:25.795Z"
last_activity: 2026-03-07 — Single-file distribution with inline JS/CSS/WASM, standard and ultimate modes
progress:
  total_phases: 12
  completed_phases: 9
  total_plans: 47
  completed_plans: 43
  percent: 87
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-28)

**Core value:** A fully functional OpenClaw-compatible AI assistant that boots from a URL in a browser tab — zero install, instant distribution, no Node.js or server required.
**Current focus:** Phase 7a - just-bash filesystem integration (partially complete)

## Current Position

Phase: 08 of 10 (Polish & Release - Static Bundle)
Status: **08-02 Complete - Single-File Bundle Mode**
Last activity: 2026-03-07 — Single-file distribution with inline JS/CSS/WASM, standard and ultimate modes

Progress: [████████████████████░░░░] 87%

## Phase 7a Status

**Completed:**

- ✅ 07a-01: just-bash integration foundation (just-bash bridge, Go bindings, 4 file tools)

**Not Implemented:**

- ❌ 07a-02: Filesystem UI (tree view, editor panel - deferred)
- ❌ 07a-03: OverlayFs mounts (File System Access API - deferred)
- ⚠️ 07a-04: Advanced tools (file_edit, file_stat not built - partial)
- ⚠️ 07a-05: Tests and docs (Phase 6 tests cover - partial)

**What Works:**

- File operations via agent: file_read, file_write, dir_list, file_search
- 79+ bash commands available in browser
- Virtual filesystem (InMemoryFs)
- No bridge binary required

**What's Missing:**

- Visual filesystem UI (file tree, editor panel)
- Mount real directories (OverlayFs)
- Advanced editing (sed/awk operations)
- Phase-specific tests and README

## Phase 08 Status

**Completed:**

- ✅ 08-01: Vite Static Bundle (bundler setup, WASM copy, Tailwind compilation, GitHub Actions workflow)
- ✅ 08-02: Single-file bundle mode (inline JS/CSS/WASM, Blob Worker, standard + ultimate modes)

**In Progress:**

- 🔄 08-03: Distribution (npm package, CLI, Docker)

**Not Started:**

_None - all planned Phase 08 tasks have been started or completed_

**What Works:**

- Vite bundler configured with vite-plugin-static-copy
- WASM files copied to dist-bundle/ (webclaw.wasm + webclaw.wasm.br)
- Static JS files copied (worker.js, wasm_exec.js, justbash-bridge.js, webclaw-host.js)
- Tailwind CSS compiled to ~15KB (no CDN dependency)
- Relative paths for file:// protocol compatibility
- Dual compression (Brotli + Gzip) for all assets
- GitHub Actions workflow for automated builds and releases
- Build outputs: dist-bundle/ with index.html, assets/, static/, vendor/
- npm package configuration with bin entry for CLI
- Zero-dependency CLI tool (`webclaw-static serve --port=8080 --open`)
- GitHub Actions release workflow (build → release → publish → docker)
- Multi-stage Dockerfile with nginx:alpine (~25MB image)
- Distribution documentation (4 methods: npx, npm, download, Docker)

**What's New (08-03):**

- `npx webclaw-static serve` works without installation
- Automated npm publishing on git tag push
- Docker image serves static files on port 80
- Brotli compression support in CLI and Docker
- SPA routing support for all bundle formats

## Phase 09 Status

**Completed:**

- ✅ 09-01: OAuth Infrastructure (PKCE flow, encrypted storage, popup bridge, Connected Services UI)
- ✅ 09-02: Twitter/X Integration (post, timeline, search, reply tools with rate limiting)
- ✅ 09-04: GitHub Integration (REST API v3 tools, GraphQL v4 foundation)
- ✅ 09-05: Notion Integration (database queries, page read/update, search)

**In Progress:**

- 🔄 09-03: Google integration (Gmail, Calendar)

**What Works:**

- Automatic token refresh (5-minute proactive window)
- Popup-based OAuth flow with postMessage callbacks
- Settings UI for managing connections
- Provider configs: Twitter, Google, GitHub, Notion
- Twitter/X API v2 integration:
  - Post tweets with text validation (280 char limit)
  - Reply to existing tweets
  - Search recent tweets with query operators
  - Get home timeline from followed users
  - Rate limit tracking and response caching
- GitHub REST API v3 integration:
  - List issues (assigned to user or in repo)
  - List pull requests with branch info
  - Create issues with labels
  - Search code with GitHub syntax
  - Comment on issues/PRs
- GitHub GraphQL v4 foundation (queries, variables, error handling)
- Rate limit tracking from GitHub API headers
- OAuth-authenticated API calls with automatic token refresh
- Notion API integration:
  - List databases with property schemas
  - Query databases with filters (status, date, checkbox, etc.)
  - Read page content with block formatting
  - Update page properties with type inference
  - Search pages and databases
  - Database discovery with name→ID resolution
  - Query builder with fluent API
  - Rate limiting (3 req/sec with 429 retry)

**What's Missing:**

- Google integration tools (Gmail send/read, Calendar events)
- OAuth client IDs (must be configured per deployment)
- Privacy policy page (required for OAuth apps)

## Performance Metrics

**Velocity:**

- Total plans completed: 33
- Average duration: 26 min
- Total execution time: ~14 hours

**By Phase:**

| Phase                   | Plans | Total   | Avg/Plan |
| ----------------------- | ----- | ------- | -------- |
| 01-wasm-pipeline        | 2     | 24 min  | 12 min   |
| 02-config-identity      | 4     | 155 min | 39 min   |
| 03-intelligence-core    | 3     | 14 min  | 5 min    |
| 04-tools-and-webchat-ui | 3     | 31 min  | 10 min   |
| 05-live-ai-connection   | 3     | 4 min   | 1 min    |
| 06-real-agent-loop      | 1     | 3 min   | 3 min    |
| 08-static-bundle        | 1     | 12 min  | 12 min   |
| 09-social-integrations  | 4     | ~50 min | 12 min   |

**Recent Trend:**

- Last 5 plans: 08-03 (2 min), 08-01 (12 min), 09-05 (17 min), 09-01 (7 min), 06-01 (3 min)
- Trend: Phase 8 (Static Bundle) - 2 of 3 plans complete, distribution channels ready

| Phase 08 P03 | 2 min | 5 tasks | 6 files |
| Phase 08 P01 | 12 min | 5 tasks | 4 files |

| Phase 06 P01 | 3 min | 5 tasks | 6 files |
| Phase 05 P03 | 3 min | 13 tasks | 6 files |
| Phase 05 P02 | 1 min | 10 tasks | 2 files |
| Phase 05 P01 | 1 min | 6 tasks | 1 files |
| Phase 04 P03 | 15 min | 2 tasks | 1 files |
| Phase 04 P02 | 10 min | 2 tasks | 4 files |
| Phase 04 P01 | 6 min | 4 tasks | 11 files |
| Phase 03 P04 | 6 min | 7 tasks | 7 files |
| Phase 03 P03 | 4 min | 7 tasks | 9 files |
| Phase 03 P02 | 4 min | 7 tasks | 9 files |
| Phase 03 P01 | 4 min | 7 tasks | 8 files |
| Phase 02 P04 | 35 min | 4 tasks | 6 files |
| Phase 02 P03 | 8 min | 4 tasks | 4 files |
| Phase 02 P02 | 12 min | 4 tasks | 6 files |
| Phase 02 P01 | 116s | 3 tasks | 4 files |
| Phase 06-real-agent-loop P07 | 3 min | 6 tasks | 4 files |
| Phase 06-real-agent-loop P06 | 18 min | 6 tasks | 5 files |
| Phase 06-real-agent-loop P03 | 3min | 5 tasks | 4 files |
| Phase 06-real-agent-loop P04 | 2min | 5 tasks | 5 files |
| Phase 09-social-integrations P09-05 | 17 | 6 tasks | 7 files |
| Phase 09-social-integrations P09-04 | 6 | 5 tasks | 9 files |
| Phase 09-social-integrations P09-03 | 5 | 6 tasks | 10 files |
| Phase 09-social-integrations P09-02 | 374 | 6 tasks | 6 files |
| Phase 08 P03 | 2 | 5 tasks | 6 files |
| Phase 09-social-integrations P09-06 | 15 | 3 tasks | 5 files |
| Phase 09.1-oauth-ux-api-token-rework P01 | 8 | 2 tasks | 3 files |
| Phase 09.1-oauth-ux-api-token-rework P02 | 2 | 2 tasks | 5 files |

## Accumulated Context

### Roadmap Evolution

- Phase 09.1 inserted after Phase 9: OAuth UX & API Token Rework (URGENT) — GitHub/Notion OAuth broken (require client_secret, no server), replace with PAT/token inputs; Twitter/Google keep PKCE but need prominent redirect URI UX
- Phase 11 added: Cron Scheduler — browser-based crontab-style scheduler for automation tasks while browser is open; future extensibility to external scheduling services

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Full Go WASM over TinyGo: reflect and encoding/json required; ~5MB compressed size is acceptable
- No net/http in WASM: all HTTP via syscall/js fetch() bridge — must be reflected in every provider implementation
- Rebuild core cleanly (not fork PicoClaw): design around browser constraints from day one
- [01-01] syscall/js allowed in cmd/webclaw/main.go (boundary layer), restricted to internal/jsbridge elsewhere
- [01-01] static/wasm_exec.js excluded from git — generated from GOROOT at build time to avoid Go version lock
- [01-01] Phase 1 indexedDBOpen is smoke-test stub; full IndexedDB ops deferred to Phases 2-3
- [02-01] Config events (webclaw:first-run, webclaw:config-ready) pass primitive values only — Go structs can't cross JS boundary
- [02-01] IndexedDB operations use goroutine-spawn pattern to avoid blocking main thread
- [02-02] Web Crypto API for all crypto operations (PBKDF2, AES-256-GCM) — FIPS-compliant via browser
- [02-02] Keys never exist as plaintext in JavaScript — only in WASM linear memory
- [02-03] Six identity files in IndexedDB with SHA256 checksums
- [02-03] Bootstrap limits: 20K per file, 150K total for system prompt assembly
- [02-04] Used interfaces (IdentityFileProvider, IdentityFileImporter) to avoid import cycle between config and identity packages
- [02-04] Placed export/import bridge registration in main.go to prevent circular dependencies
- [03-01] All provider HTTP calls use syscall/js fetch bridge — no net/http imports allowed in provider package
- [03-01] SSE parsing implemented in Go for flexibility with Anthropic, OpenAI, OpenRouter event formats
- [03-01] Provider chain pattern (primary→retry→fallback) for resilience without external dependencies
- [03-01] Router infers vendor from model names for convenience: claude-*→anthropic, gpt-*→openai
- [03-02] Two message types: ConversationMessage (internal with metadata) and Message (API format)
- [03-02] Web Worker runs separate WASM instance for non-blocking streaming
- [03-02] Context cancellation for stream abort (simpler than channels in WASM)
- [03-04] Hybrid search with 70% cosine similarity + 30% BM25 for semantic + keyword balance
- [03-04] Memory embeddings stored as Float32Array in IndexedDB for compact storage
- [03-04] LRU eviction with multi-factor scoring: age, access count, and importance
- [03-04] Created duplicate types in jsbridge to avoid import cycles with memory package
- [03-04] Gzip compression for memory archival before deletion
- [Phase 04-01]: MemoryAgent interface in tools package avoids circular import between tools and agent
- [Phase 04-01]: Provider interface callback changed from func(string) to func(provider.Token) to carry tool_use metadata through the dispatch loop
- [Phase 04-01]: providerAdapter bridges provider.Router channel-based stream to agent.Provider callback-based interface
- [Phase 04-02]: v1 keystore passphrase is fixed string webclaw-v1-key; keys encrypted at rest but not user-derived; v2 will prompt user
- [Phase 04-02]: onToolEvent uses callback pattern (not direct postMessage) so WASM in worker context posts via worker.js
- [Phase 04-02]: globalAgentLoop singleton in worker_bridge.go: pre-configured loop reused per stream so SetRouter/SetToolRegistry/SetWorkerBridge wiring is preserved
- [Phase 04-03]: Container-based DOM manipulation (innerHTML) simpler than while-loop child removal for dynamic UI sections
- [Phase 04-03]: Disabled button states during async operations prevent race conditions and double-submits
- [Phase 05-01]: Goroutine-based async keystore initialization pattern for non-blocking startup with continue-on-error tolerance
- [Phase 05-live-ai-connection]: Dispatch event once at end of async loader with full provider list, rather than per-provider events (reduces event noise)
- [Phase 05-live-ai-connection]: Add HasProvider() check at start of providerAdapter.Stream for fail-fast validation with clearer error messages
- [Phase 05-03]: Console logging via syscall/js for browser DevTools visibility (not Go log package)
- [Phase 05-03]: Error toast notifications with auto-dismiss and specific error type handling
- [Phase 05-03]: Demo mode messaging '[Demo Mode] Enter API key in Settings to enable live AI'
- **[Phase 06-01]:** FinishReason normalization to "tool_use" across all providers (Anthropic uses natively, OpenAI/OpenRouter use "tool_calls" but we normalize)
- **[Phase 06-01]:** Accumulate partial JSON from streaming deltas, parse at message_stop/finish
- **[Phase 06-01]:** Convert generic []map[string]interface{} tools to provider-specific formats at request time
- **[Phase 06-01]:** Handle one tool at a time per Token (simplifies agent loop integration)
- **[Phase 06-07]:** Router wraps all providers in ProviderChain for automatic retry and fallback
- **[Phase 06-07]:** Exponential backoff: 1s → 2s → 4s delays with multiplier 2.0
- **[Phase 06-07]:** Fallback chain: Anthropic → OpenAI → OpenRouter configured in async goroutine
- **[Phase 06-07]:** Provider health tracking with consecutive failure detection (unhealthy after 3 failures)
- **[Phase 06-07]:** Non-retryable errors (401, 403, 400) fail fast without wasting retry attempts
- [Phase 06-real-agent-loop]: CheckAndSummarize called before AddAssistantResponse to prevent losing new responses in summary
- [Phase 06-real-agent-loop]: Last 2 messages preserved after summarization for context continuity
- [Phase 06-real-agent-loop]: Created summarizerProviderAdapter to wrap router for agent.Provider interface
- **[Phase 06-06]:** Memory store initializes BM25-only, async goroutine loads OpenAI key to enable hybrid search
- **[Phase 06-06]:** Storage hygiene via CheckQuota before every Store(), LRU eviction at 80% quota threshold
- **[Phase 06-06]:** QuotaInfo.ShouldEvict flag unifies eviction decision logic across memory system
- **[Phase 06-04]:** Async flush pattern: fact extraction runs in goroutine to avoid blocking summarization
- **[Phase 06-04]:** Dual storage strategy: facts stored to both memory store (searchable) and MEMORY.md (human-readable)
- **[Phase 06-04]:** Metadata tagging for facts includes conversation_id for traceability
- [Phase 06-real-agent-loop]: Hybrid token estimation better than chars/4 heuristic - uses word length, punctuation, and formatting overhead
- [Phase 09-social-integrations]: Tools use gmail* and calendar* prefixes for clarity and namespacing
- [Phase 09-social-integrations]: Both Gmail and Calendar use the same 'google' OAuth provider token
- [Phase 09-social-integrations]: Email composition uses RFC 2822 format with base64url encoding for Gmail API
- [Phase 09-social-integrations]: OAuth goroutine runs on main thread WASM (not worker) so js.Global() returns window scope for RegisterJSExports()
- [Phase 09-social-integrations]: RegisterOAuthBridge() uses webclaw.Get('oauth') get-or-create pattern to preserve JS-side openPopup function
- [Phase 09.1-oauth-ux-api-token-rework]: testDir changed to '.' with testMatch glob — zero risk to existing phase06 tests, single playwright command runs all suites
- [Phase 09.1-oauth-ux-api-token-rework]: Smoke tests skip window.webclaw wait — DOM-only assertions reduce WASM startup flakiness
- [Phase 09.1-02]: AuthType omitempty for backward-compat PAT tokens — existing OAuth tokens keep empty AuthType, no DB migration needed
- [Phase 09.1-02]: savePATToken async (IndexedDB I/O), markInvalid sync (in-memory flag) — matches operation type
- [Phase 09.1-02]: invalidTokens map is transient (resets on page reload) — correct for 401/403 recovery flow
- [Phase 09.1-oauth-ux-api-token-rework]: setStatusDot() helper replaces all status span updates with safe DOM methods — eliminates XSS risk in Connected Services code path
- [Phase 09.1-oauth-ux-api-token-rework]: WASM export tests wait for savePATToken/markInvalid specifically — exports registered ~300ms after webclaw.oauth initializes (async goroutine)

### Pending Todos

- 08-03: Distribution (npm package, CLI, Docker)

### Blockers/Concerns

None. Phase 08-02 single-file bundle mode complete. Ready for distribution packaging.

## Session Continuity

Last session: 2026-03-08T18:35:17.327Z
Stopped at: Completed 09.1-03-PLAN.md (pending checkpoint)
Resume file: None

## Phase 9 Summary

Plans completed in Phase 9:

- 09-01: OAuth Infrastructure (PKCE flow, encrypted storage, popup bridge, Connected Services UI)
- 09-02: Twitter/X Integration (API v2, 4 tools: post, reply, search, timeline)
- 09-04: GitHub Integration (REST API v3, GraphQL v4 foundation, 5 tools)
- 09-05: Notion Integration (databases, pages, queries, 5 tools, discovery helpers)

**Phase 9 IN PROGRESS** - Social & Productivity Integrations

- PKCE parameter generation (RFC 7636 compliant)
- Encrypted OAuth token storage (AES-256-GCM via Web Crypto API)
- Automatic token refresh (5-minute proactive window)
- Popup-based OAuth flow with postMessage callbacks
- Settings UI for managing connections with real-time status
- Provider configs: Twitter, Google, GitHub, Notion
- JavaScript exports: initiateConnection(), disconnect(), getConnectionStatus()
- Token store with IndexedDB persistence
- OAuth manager with full flow orchestration
- Connected Services section in Settings view

**NEW: Twitter/X Integration (09-02)**

- Twitter API v2 client with OAuth authentication
- 4 WebClaw tools: twitter_post, twitter_reply, twitter_search, twitter_timeline
- Rate limit tracking with preemptive limiting (300 req/15min)
- Response caching for read operations (2-minute TTL)
- Tweet length validation (280 character limit)
- Formatted output with engagement metrics (likes, retweets, replies)
- Comprehensive test coverage (2 test files, ~1,200 lines of tests)

**NEW: GitHub Integration (09-04)**

- GitHub REST API v3 client with OAuth authentication
- 5 WebClaw tools: list_issues, list_prs, create_issue, search_code, comment
- Rate limit tracking from GitHub API headers (5,000 req/hour)
- Input schemas with validation and helpful error messages
- OAuth connectivity checks with "Please connect GitHub in Settings" prompts
- Formatted output with issue/PR numbers, URLs, labels, assignees
- GraphQL v4 foundation for future complex queries
- Comprehensive test coverage (4 test files, ~1,000 lines of tests)

Ready for:

- Google integration tools (Gmail, Calendar) - 09-03
- Notion integration tools (databases, pages) - 09-05
- Google integration (Gmail send/read, Calendar events)
- Notion integration (databases, pages, queries)
- Additional GitHub tools via GraphQL (complex queries, analytics)

Plans completed in Phase 6:

- 06-01: Provider-Side Tool Support with tool_use/tool_calls parsing
- 06-02: Tool Registry Wired to Provider (tools flow from registry → agent loop → provider → LLM)
- 06-03: Real LLM-Based Summarization (20-message threshold, 75% token limit, last 2 messages preserved)
- 06-04: Memory Flush Before Summarization (extract key facts, store to memory and MEMORY.md)
- 06-06: Memory System Integration (async OpenAI embedder, storage hygiene, LRU eviction at 80% quota)
- 06-07: Provider Streaming Failover with exponential backoff and fallback chains

**Phase 6 IN PROGRESS** - Real Agent Loop

- Tool definitions in CompletionRequest (all providers)
- Anthropic content_block_start/content_block_delta tool_use parsing
- OpenAI/OpenRouter tool_calls parsing with FinishReason normalization
- Token struct has ToolName, ToolInput, ToolUseID for agent loop integration
- Comprehensive test coverage for all providers
- Tool registry integration with agent loop
- Provider interface accepts tools parameter
- Console logging for debugging tool flow
- Provider failover with exponential backoff (1s, 2s, 4s)
- Automatic fallback chains: Anthropic → OpenAI → OpenRouter
- Provider health tracking with failure/success monitoring
- Retryable error classification (429, 502, 503, 504, 529)
- Non-retryable errors fail fast (401, 403, 400)
- Real LLM-based summarization with 20-message threshold
- Context window management (75% token threshold)
- Context continuity via last 2 message preservation after summarization
- Summarizer wired to agent loop and main.go initialization
- **NEW: Memory flush before summarization extracts and stores key facts**
- **NEW: Facts stored to memory store and MEMORY.md for dual durability**
- **NEW: Async non-blocking flush pattern for conversation continuity**
- **NEW: Memory store with IndexedDB backing and Float32Array embeddings**
- **NEW: Hybrid search (70% cosine + 30% BM25) for semantic + keyword relevance**
- **NEW: Storage hygiene with navigator.storage.estimate quota checking**
- **NEW: LRU eviction triggered at 80% quota threshold**
- **NEW: Async OpenAI embedder loading for non-blocking startup**

Ready for:

- End-to-end testing with live LLM and real tool execution including memory_store and memory_search
- Additional tool implementations
- Tool result formatting and UI display refinements
