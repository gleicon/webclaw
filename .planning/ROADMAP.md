# Roadmap: WebClaw

## Overview

WebClaw is built in four phases, each delivering a complete, verifiable capability. Phase 1 establishes the WASM build pipeline — nothing else can exist without it. Phase 2 layers in agent configuration, secure key storage, and identity files so the runtime knows who it is. Phase 3 builds the intelligence core: LLM provider routing, the agent loop, and the hybrid memory system. Phase 4 delivers the user-facing experience: browser tools and the webchat UI that makes the agent usable by the developer for dogfooding.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: WASM Pipeline** - Build pipeline that compiles, loads, and distributes the WASM binary in a browser tab
- [x] **Phase 2: Configuration and Identity** - Agent configuration with secure key storage and identity file system (COMPLETE)

### Phase 2: Configuration and Identity

**Goal**: The agent has a persistent identity, secure configuration, and encrypted API keys before any LLM call is made
**Depends on**: Phase 1
**Requirements**: CONF-01, CONF-02, CONF-03, CONF-04, SEC-01, SEC-02, SEC-03, IDNT-01, IDNT-02, IDNT-03, IDNT-04
**Success Criteria** (what must be TRUE):

1. User can load a JSON config (snake_case or camelCase) and it is persisted to IndexedDB under `webclaw:config`
2. User can export and reimport config as a JSON file from the browser and the agent state is fully restored
3. On first run, the agent prompts for a passphrase; subsequent runs require it to decrypt stored API keys
4. API keys are never readable as plaintext in the browser DevTools JavaScript console or memory inspector
5. Default identity files (IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md) are loaded from IndexedDB and user can edit them in the browser
   **Plans**: 02-01 (Config struct + IndexedDB persistence + first-run) — COMPLETE, 02-02 (Web Crypto bridge + Encrypted key storage) — COMPLETE, 02-03 (Identity file system) — COMPLETE, 02-04 (Import/Export config with browser file APIs) — COMPLETE

### Phase 3: Intelligence Core

**Goal**: The agent can hold a conversation with an LLM provider, manage its context window, and persist and recall memories
**Depends on**: Phase 2
**Requirements**: PROV-01, PROV-02, PROV-03, PROV-04, PROV-05, AGNT-01, AGNT-02, AGNT-03, AGNT-04, MEM-01, MEM-02, MEM-03, MEM-04, MEM-05
**Success Criteria** (what must be TRUE):

1. Agent routes a `vendor/model-id` string (e.g. `anthropic/claude-sonnet-4-5`) to the correct provider API via JS fetch() with no `net/http` imports
2. Agent streams an LLM response token-by-token to the caller without blocking the UI thread (runs in Web Worker)
3. When conversation history exceeds threshold, the agent automatically summarizes and replaces history — the user sees the conversation continue naturally
4. Agent stores a memory fact and later retrieves it via hybrid vector+BM25 search with ranked results
5. When IndexedDB usage exceeds 80% of quota, old memories are archived without user-visible data loss
   **Plans**: 03-01 (LLM Provider System with Anthropic, OpenAI, OpenRouter) — COMPLETE, 03-02 (Agent Loop) — Planned, 03-03 (Memory System) — Planned

### Phase 4: Tools and Webchat UI

**Goal**: The developer can interact with the agent through a browser chat interface, use browser tools, and dogfood the full system
**Depends on**: Phase 3
**Requirements**: TOOL-01, TOOL-02, TOOL-03, TOOL-04, TOOL-05, TOOL-06, UI-01, UI-02, UI-03, UI-04, UI-05
**Success Criteria** (what must be TRUE):

1. User can type a message and receive a streamed response in the browser chat UI with clear user/agent turn separation
2. Agent can invoke `web_fetch`, `web_search`, `memory_store`, and `memory_search` tools and the UI displays tool name, status, and result summary for each
3. User can view and edit identity files from a settings panel without leaving the browser tab
4. User can enter provider API keys in a settings panel and they are encrypted on save (key never visible in plaintext after entry)
5. Developer can dogfood a complete multi-turn conversation with tool use end-to-end from a browser tab with no server dependency
   **Plans**: 3 plans

Plans:

- [ ] 04-01-PLAN.md — Tool registry + four browser tool implementations + real provider wiring in AgentLoop
- [ ] 04-02-PLAN.md — JS bridge exports for identity/keystore + tool event emission across worker boundary
- [ ] 04-03-PLAN.md — Full Tailwind dark-mode chat UI (3 tabs, chat bubbles, tool panel, settings, identity editor)

### Phase 5: Live AI Provider Connection

**Goal**: WebClaw connects to real AI providers (Anthropic, OpenAI, OpenRouter) using stored API keys, enabling actual conversations beyond mock responses
**Depends on**: Phase 4
**Requirements**: PROV-01, PROV-02, SEC-02
**Success Criteria** (what must be TRUE):

1. ✅ API keys are retrieved from encrypted keystore and passed to provider router at initialization
2. ✅ Real API calls succeed with valid keys (tested against Anthropic, OpenAI, or OpenRouter)
3. ✅ Missing or invalid keys return clear error messages to the UI
4. ✅ Provider selection dropdown actually routes to correct provider with live API calls
5. ✅ End-to-end: User message → LLM API call → streamed response → UI display (no mocks)
   **Plans**: 05-01 (Async keystore init) — COMPLETE, 05-02 (Router config) — COMPLETE, 05-03 (E2E testing) — COMPLETE

### Phase 6: Real Agent Loop

**Goal:** Make WebClaw a real OpenClaw implementation with working tool_use loop, real LLM-based summarization, memory system, and provider failover
**Depends on:** Phase 5
**Requirements:** AGNT-01, AGNT-02, AGNT-03, AGNT-04, MEM-01, MEM-02, MEM-03, MEM-04, MEM-05, PROV-03, PROV-04
**Success Criteria** (what must be TRUE):

1. Providers send tool definitions to LLM and parse tool_use/tool_calls from responses
2. Agent loop passes tools to provider on every iteration, enabling real tool execution
3. When conversation exceeds 20 messages or 75% of context window, automatic LLM-based summarization occurs
4. Before summarization, key facts are extracted and flushed to memory store + MEMORY.md
5. Memory documents stored in IndexedDB with Float32 embeddings retrieved via hybrid search (0.7 vector + 0.3 BM25)
6. Storage hygiene triggers at 80% quota with LRU eviction of old memories
7. Provider failover with exponential backoff: primary → retries → fallback chain (1s, 2s, 4s delays)
8. Token counting uses accurate estimation (not crude chars/4)
9. Full E2E flow works: user message → LLM with tools → tool_use → execute → tool_result → final response
   **Plans**: 06-01 (Provider tool support) — COMPLETE, 06-02 (Agent loop wiring) — COMPLETE, 06-03 (Real summarization) — COMPLETE, 06-04 (Memory flush) — COMPLETE, 06-05 (Integration) — COMPLETE, 06-06 (Memory system) — COMPLETE, 06-07 (Provider streaming & failover) — COMPLETE

### Phase 7a: just-bash Filesystem (INSERTED)

**Goal**: Enable file operations in WebClaw WITHOUT requiring the local bridge binary to be installed. Use just-bash as a fallback when bridge is not connected.
**Depends on**: Phase 6
**Requirements**: BRIDGE-01, BRIDGE-02, BRIDGE-03, BRIDGE-04, TOOL-01, TOOL-02, TOOL-03, TOOL-04, TOOL-05, TOOL-06, UI-01, UI-02, UI-03, UI-04, UI-05
**Success Criteria** (what must be TRUE):

1. ✅ just-bash npm dependency added and loaded in browser
2. ✅ Go→JS bridge enables WASM to call just-bash commands
3. ⚠️ File tools work via just-bash: file_read, file_write, file_search, dir_list (file_edit, file_stat not built)
4. ❌ Filesystem UI provides VS Code-like file explorer with toolbar, tree, and editor (NOT IMPLEMENTED)
5. ❌ OverlayFs allows mounting local directories (Chrome/Edge) for safe preview mode (NOT IMPLEMENTED)
6. ✅ Smart routing uses just-bash when bridge unavailable
7. ✅ All file operations work immediately without bridge binary
   **Plans**:

- ✅ 07a-01 (just-bash integration: npm dependency, JS bridge, Go bindings, file tools)
- ❌ 07a-02 (Filesystem UI: tab, tree view, editor, file operations - NOT IMPLEMENTED)
- ❌ 07a-03 (OverlayFs mounts: File System Access API, mount dialog - NOT IMPLEMENTED)
- ⚠️ 07a-04 (Advanced tools: sed/awk edit - file_edit/file_stat not built)
- ⚠️ 07a-05 (Tests and docs: Partial - Phase 6 tests cover, no phase-specific README)

### Phase 7: Local Bridge Binary

**Goal**: Unlock capabilities browsers can't do (file I/O, shell commands, git operations) via a local companion binary
**Depends on**: Phase 6 (or Phase 7a after completion)
**Requirements**: BRIDGE-01, BRIDGE-02, BRIDGE-03, BRIDGE-04
**Success Criteria** (what must be TRUE):

1. `webclaw-bridge` binary runs on macOS/Linux and binds to 127.0.0.1:18800
2. Browser connects via WebSocket with 6-digit OTP + bearer token pairing
3. File read/write operations work through bridge (file_picker, file_read, file_write tools)
4. Shell execution tool runs commands and returns stdout/stderr
5. Git operations tool clones, commits, pushes via bridge
6. Connection is 127.0.0.1-only (no remote access)

### Phase 8: Polish & Release

**Goal**: Production-ready release with documentation, distribution, and stability improvements
**Depends on**: Phase 6 (can run parallel to Phase 7a)
**Requirements**: DOCS-01, DIST-01, STABLE-01, PERF-01
**Success Criteria** (what must be TRUE):

1. README with installation and usage instructions
2. Static site bundle with zero external dependencies (npm, GitHub releases, Docker)
3. Multi-file bundle (~920KB) for web server hosting
4. Single-file bundle (~1MB) for folder distribution
5. Ultimate standalone HTML (~1.3MB) for email/sharing
6. CLI command `npx webclaw-static serve` works without installation
7. Binary releases for bridge (macOS/Linux ARM64/x86_64)
8. Conversation export/import (save/load chat history)
9. Performance: WASM <2s load time, streaming <1s first token
10. Error telemetry and graceful degradation
    **Plans**:

- [x] 08-01 (Vite bundler setup: multi-file bundle with WASM, Tailwind compilation)
- [x] 08-02 (Single-file mode: inline JS/CSS/WASM, Blob Worker, ultimate standalone)
- [ ] 08-03 (Distribution: npm package, CLI, GitHub Actions, Docker)

### Phase 9: Social & Productivity Integrations

**Goal**: Enable WebClaw to interact with popular APIs (Twitter/X, GitHub, Google, Notion) for real-world productivity use cases
**Depends on**: Phase 8 (or can run parallel to Phase 7/7a)
**Requirements**: INTEG-01, INTEG-02, INTEG-03, INTEG-04, INTEG-05, AUTH-01, AUTH-02
**Success Criteria** (what must be TRUE):

1. User can authenticate with OAuth 2.0 for Twitter/X, Google (Gmail/Calendar), GitHub
2. User can post tweets, manage Twitter timeline via WebClaw
3. User can check email, send messages via Gmail
4. User can view calendar, create events via Google Calendar
5. User can list GitHub issues, review PRs, create comments
6. User can query Notion databases, update pages
7. All OAuth tokens encrypted and stored securely in IndexedDB
8. Integration tools follow same pattern as existing tools (web_fetch, web_search)
   **Plans**:

- [x] 09-01-PLAN.md — OAuth infrastructure (PKCE flow, token storage, refresh handling, JS bridge)
- [x] 09-02-PLAN.md — Twitter/X integration (tweet, reply, search, timeline tools)
- [x] 09-03-PLAN.md — Google integration (Gmail send/read, Calendar events)
- [x] 09-04-PLAN.md — GitHub integration (issues, PRs, repos, comments)
- [x] 09-05-PLAN.md — Notion integration (databases, pages, queries)

### Phase 09.1: OAuth UX & API Token Rework (INSERTED)

**Goal:** Fix broken OAuth integrations and improve auth UX for all four social providers. GitHub and Notion OAuth flows are replaced with PAT/token inputs. Twitter and Google PKCE stays intact but UX improved with prominent redirect URI guidance and Client ID field on the card.
**Requirements**: AUTH-UX-01, AUTH-UX-02, AUTH-UX-03, AUTH-UX-04, AUTH-UX-05, AUTH-UX-06, AUTH-UX-07, AUTH-UX-08, AUTH-UX-09, AUTH-UX-10, AUTH-UX-11
**Depends on:** Phase 9
**Plans:** 1/3 plans executed

Plans:
- [ ] 09.1-01-PLAN.md — Playwright test scaffold (DOM smoke tests for card structure, PAT save flow)
- [ ] 09.1-02-PLAN.md — Go/WASM backend (Token.AuthType, SavePAT, MarkInvalid, savePATToken/markInvalid JS exports, WASM rebuild)
- [ ] 09.1-03-PLAN.md — index.html UI rework (PAT card variant, OAuth card callout, auth-type badges, invalid token state)

### Phase 10: Browser-Based Local Model (INSERTED)

**Goal**: Enable WebClaw to run a lightweight local AI model directly in the browser for offline/basic chat and automation capabilities
**Depends on**: Phase 6 (Real Agent Loop)
**Requirements**: LOCAL-01, LOCAL-02, LOCAL-03, LOCAL-04, LOCAL-05
**Success Criteria** (what must be TRUE):

1. User can load and run a compact AI model (e.g., ~100MB) in the browser via WebAssembly or WebGPU
2. Local model supports basic chat conversations without requiring API keys or internet connection
3. Tool execution works with local model for simple automation tasks (file operations, memory search)
4. Seamless fallback: local model when offline, cloud providers when online
5. Model can be downloaded and cached locally for subsequent sessions
   **Plans**: 4 plans

Plans:

- [ ] 10-01-PLAN.md — npm deps + test scaffolds + Go stubs (LocalProvider, jsbridge, local-llm-worker)
- [ ] 10-02-PLAN.md — WebLLM/wllama worker implementation + webclaw-host.js WebGPU detection and worker lifecycle
- [ ] 10-03-PLAN.md — Go LocalProvider channel bridge + router registration + wllama WASM vendor serving
- [ ] 10-04-PLAN.md — Settings UI (model selector, progress bar, status) + human verify checkpoint

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6

| Phase                          | Plans Complete | Status      | Completed  |
| ------------------------------ | -------------- | ----------- | ---------- |
| 1. WASM Pipeline               | 2/2            | Complete    | 2026-02-28 |
| 2. Configuration and Identity  | 4/4            | Complete    | 2026-03-01 |
| 3. Intelligence Core           | 4/4            | Complete    | 2026-03-01 |
| 4. Tools and Webchat UI        | 3/3            | Complete    | 2026-03-01 |
| 5. Live AI Provider Connection | 3/3            | Complete    | 2026-03-02 |
| 6. Real Agent Loop             | 7/7            | Complete    | 2026-03-04 |
| 7a. just-bash Filesystem       | 1/5            | Partial     | 2026-03-05 |
| 7. Local Bridge Binary         | 0/0            | Planned     |            |
| 8. Polish & Release            | 2/3            | In Progress | 2026-03-07 |
| 9. Social & Productivity       | 5/5            | Complete    | 2026-03-05 |
| 9.1 OAuth UX & Token Rework    | 0/3            | Planned     |            |
| 10. Browser Local Model        | 0/4            | Planned     |            |

### Phase 11: Cron Scheduler

**Goal:** Browser-based crontab-style scheduler that runs automation tasks on a user-defined schedule while the browser is open, with future extensibility to external scheduling services (e.g. push-based triggers or a server-side cron proxy).
**Requirements**: TBD
**Depends on:** Phase 10
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd:plan-phase 11 to break down)
