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
**Plans**: 06-01 (Provider tool support), 06-02 (Agent loop wiring), 06-03 (Real summarization), 06-04 (Memory flush), 06-05 (Integration), 06-06 (Memory system), 06-07 (Provider streaming & failover)

### Phase 7: Local Bridge Binary
**Goal**: Unlock capabilities browsers can't do (file I/O, shell commands, git operations) via a local companion binary
**Depends on**: Phase 6
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
**Depends on**: Phase 7 (or can skip to after Phase 6)
**Requirements**: DOCS-01, DIST-01, STABLE-01, PERF-01
**Success Criteria** (what must be TRUE):
  1. README with installation and usage instructions
  2. Static site deployed (GitHub Pages/Netlify) for immediate use
  3. Binary releases for bridge (macOS/Linux ARM64/x86_64)
  4. Conversation export/import (save/load chat history)
  5. Performance: WASM <2s load time, streaming <1s first token
  6. Error telemetry and graceful degradation

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. WASM Pipeline | 2/2 | Complete | 2026-02-28 |
| 2. Configuration and Identity | 4/4 | Complete | 2026-03-01 |
| 3. Intelligence Core | 4/4 | Complete | 2026-03-01 |
| 4. Tools and Webchat UI | 3/3 | Complete | 2026-03-01 |
| 5. Live AI Provider Connection | 3/3 | Complete | 2026-03-02 |
| 6. Real Agent Loop | 0/7 | Planned |  |
| 7. Local Bridge Binary | 0/0 | Planned |  |
| 8. Polish & Release | 0/0 | Planned |  |
