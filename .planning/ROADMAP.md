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
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. WASM Pipeline | 2/2 | ✅ Complete | 2026-02-28 |
| 2. Configuration and Identity | 4/4 | ✅ Complete | 2026-03-01 |
| 3. Intelligence Core | 1/3 | 🔄 In Progress | 2026-03-01 |
| 4. Tools and Webchat UI | 0/TBD | Not started | - |
