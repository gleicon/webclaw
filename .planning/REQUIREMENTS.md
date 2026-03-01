# Requirements: WebClaw

**Defined:** 2026-02-28
**Core Value:** A fully functional OpenClaw-compatible AI assistant that boots from a URL in a browser tab — zero install, instant distribution, no Node.js or server required.

## v1 Requirements

Requirements for initial release (core WASM runtime). Goal: a working browser AI agent, dogfoodable by the developer.

### Build

- [x] **BUILD-01**: Project compiles with `GOOS=js GOARCH=wasm go build` producing a `.wasm` binary
- [x] **BUILD-02**: Host page (`index.html` + `webclaw-host.js`) loads and instantiates the WASM module
- [x] **BUILD-03**: WASM imports expose `jsFetch` and `jsIndexedDB` bridges from JS to Go via `syscall/js`
- [x] **BUILD-04**: Build produces a brotli-compressed WASM artifact for distribution

### Config

- [x] **CONF-01**: User can define a config in JSON (snake_case and camelCase both accepted)
- [x] **CONF-02**: Config is persisted in IndexedDB under `webclaw:config`
- [x] **CONF-03**: Config covers: identity, agents defaults (model, maxToolIterations, temperature, bootstrap limits), model providers (apiKey, baseUrl), memory settings
- [ ] **CONF-04**: User can import/export config as a JSON file from the browser UI

### Identity

- [x] **IDNT-01**: Agent loads identity files from IndexedDB virtual filesystem (IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md)
- [x] **IDNT-02**: Identity files are injected into agent system prompt each turn, capped at bootstrapMaxChars per file (default 20K) and bootstrapTotalMaxChars total (default 150K)
- [ ] **IDNT-03**: User can edit identity files through the browser UI
- [x] **IDNT-04**: Default identity files are pre-loaded on first run

### Providers

- [ ] **PROV-01**: Agent routes LLM calls using `vendor/model-id` format (e.g. `anthropic/claude-sonnet-4-5`)
- [ ] **PROV-02**: All provider HTTP calls go through `syscall/js` fetch() — no `net/http`
- [ ] **PROV-03**: Provider supports streaming completions (streamed to UI incrementally)
- [ ] **PROV-04**: Provider failover: primary → retries with exponential backoff → fallback model chain
- [ ] **PROV-05**: Supported providers in v1: Anthropic, OpenAI, OpenRouter

### Security

- [x] **SEC-01**: API keys are encrypted with AES-256-GCM via Web Crypto API before storage in IndexedDB
- [x] **SEC-02**: Key decryption happens inside WASM linear memory — keys never exist as plaintext in JavaScript
- [x] **SEC-03**: Passphrase-derived encryption key (PBKDF2) — user sets a passphrase on first run

### Agent Loop

- [ ] **AGNT-01**: Agent executes a turn: assemble context (system prompt + identity + history) → call provider → handle response → execute tools if any → loop until no tool calls or maxToolIterations reached
- [ ] **AGNT-02**: Context history is capped — when history exceeds threshold (20 messages or 75% of context window), agent triggers summarization
- [ ] **AGNT-03**: Summarization is performed by calling the LLM provider with a summarize prompt; summary replaces history
- [ ] **AGNT-04**: Agent loop runs in a Web Worker to avoid blocking the UI thread

### Memory

- [ ] **MEM-01**: Agent can store memory documents to IndexedDB (key-value + vector embedding)
- [ ] **MEM-02**: Agent can recall memories using hybrid search: cosine vector similarity (0.7 weight) + BM25 keyword (0.3 weight), results merged and ranked
- [ ] **MEM-03**: Embeddings are computed via the active LLM provider's embedding endpoint (stored as Float32Arrays in IndexedDB)
- [ ] **MEM-04**: Before compaction/summarization, durable knowledge is flushed from conversation into MEMORY.md (matching OpenClaw's memory flush behavior)
- [ ] **MEM-05**: Storage hygiene: when IndexedDB usage exceeds 80% of quota, old memories are archived/purged

### Tools

- [x] **TOOL-01**: Agent can invoke `web_fetch` — fetches a URL via JS fetch(), returns content to agent
- [x] **TOOL-02**: Agent can invoke `web_search` — queries a search provider (DuckDuckGo as default) and returns results
- [x] **TOOL-03**: Agent can invoke `memory_store` — stores a fact or document to memory
- [x] **TOOL-04**: Agent can invoke `memory_search` — recalls relevant memories for a query
- [x] **TOOL-05**: Tool registry allows registering tools with name, description, JSON schema parameters, and execute function
- [x] **TOOL-06**: Tool execution results have dual output: content fed back to LLM for next iteration, and display content for the UI

### Webchat UI

- [ ] **UI-01**: User can type a message and receive a streamed response in the browser
- [ ] **UI-02**: UI displays tool execution events (tool name, status, result summary)
- [x] **UI-03**: User can view and edit identity files from a settings panel
- [x] **UI-04**: User can configure provider API keys from a settings panel (keys encrypted on save)
- [ ] **UI-05**: Conversation history is displayed with clear user/agent turn separation

## v2 Requirements

Deferred to next milestone after v1 is dogfooded.

### Bridge

- **BRDG-01**: Local bridge binary (`webclaw-bridge`) compiles and runs as a standalone Go binary on macOS/Linux/Windows
- **BRDG-02**: Bridge exposes REST API for simple ops: read_file, write_file, list_dir, edit_file (scoped to workspace)
- **BRDG-03**: Bridge exposes WebSocket endpoint for streaming ops: exec (shell command with live stdout/stderr)
- **BRDG-04**: Bridge pairing: 6-digit OTP displayed in terminal, entered in browser UI, generates bearer token
- **BRDG-05**: Bridge binds 127.0.0.1 only; bearer token stored encrypted in IndexedDB
- **BRDG-06**: Browser agent detects bridge availability and upgrades tool profile to include file/exec/git tools
- **BRDG-07**: Bridge implements dangerous command blocking (rm -rf, sudo, dd, fork bombs) and workspace path scoping

### Plugin SDK

- **PLUG-01**: JS/TS plugin API: registerTool, registerHook, registerChannel, registerService
- **PLUG-02**: Plugins are ES modules loaded at runtime from configured URLs
- **PLUG-03**: Plugin manifest declares required permissions; user approves on install
- **PLUG-04**: Lifecycle hooks: message:received, message:sent, tool:before, tool:after, compaction:before, memory:flush

### Migration

- **MIGR-01**: User can import an OpenClaw workspace zip (maps IDENTITY.md, SOUL.md etc. into IndexedDB)
- **MIGR-02**: User can import a PicoClaw `config.json` (direct field mapping)
- **MIGR-03**: User can import a NullClaw SQLite memory export (JSON snapshot → IndexedDB vectors)

### Service Worker

- **SW-01**: Agent registers a Service Worker for background heartbeat execution
- **SW-02**: Agent survives tab closure — Service Worker continues heartbeat, resumes on tab reopen

## Out of Scope

| Feature | Reason |
|---------|--------|
| Native messaging channels (Telegram, Discord, Slack, WhatsApp) | WebClaw IS the channel — webchat + bridge handles all interaction |
| Mobile app | Browser-first for v1; mobile is a separate project |
| DM pairing flow | Single-user browser context; not applicable |
| WASM SIMD128 for vector math | Future optimization; standard float32 math is sufficient for v1 corpus sizes |
| Full OpenClaw Node.js plugin API | Node.js APIs unavailable in browser; JS/TS SDK is the replacement |
| TinyGo compilation target | Full Go chosen for reflect/encoding/json support; size tradeoff accepted |
| PicoClaw fork/dependency | Rebuilding cleanly to avoid channel SDK baggage and design assumption inheritance |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUILD-01 | Phase 1 | ✅ Complete |
| BUILD-02 | Phase 1 | ✅ Complete |
| BUILD-03 | Phase 1 | ✅ Complete |
| BUILD-04 | Phase 1 | ✅ Complete |
| CONF-01 | Phase 2 | 🚧 In Progress |
| CONF-02 | Phase 2 | 🚧 In Progress |
| CONF-03 | Phase 2 | 🚧 In Progress |
| CONF-04 | Phase 2 | 🚧 In Progress |
| IDNT-01 | Phase 2 | 🚧 In Progress |
| IDNT-02 | Phase 2 | 🚧 In Progress |
| IDNT-03 | Phase 2 | 🚧 In Progress |
| IDNT-04 | Phase 2 | 🚧 In Progress |
| SEC-01 | Phase 2 | 🚧 In Progress |
| SEC-02 | Phase 2 | 🚧 In Progress |
| SEC-03 | Phase 2 | 🚧 In Progress |
| PROV-01 | Phase 3 | Pending |
| PROV-02 | Phase 3 | Pending |
| PROV-03 | Phase 3 | Pending |
| PROV-04 | Phase 3 | Pending |
| PROV-05 | Phase 3 | Pending |
| AGNT-01 | Phase 3 | Pending |
| AGNT-02 | Phase 3 | Pending |
| AGNT-03 | Phase 3 | Pending |
| AGNT-04 | Phase 3 | Pending |
| MEM-01 | Phase 3 | Pending |
| MEM-02 | Phase 3 | Pending |
| MEM-03 | Phase 3 | Pending |
| MEM-04 | Phase 3 | Pending |
| MEM-05 | Phase 3 | Pending |
| TOOL-01 | Phase 4 | Complete |
| TOOL-02 | Phase 4 | Complete |
| TOOL-03 | Phase 4 | Complete |
| TOOL-04 | Phase 4 | Complete |
| TOOL-05 | Phase 4 | Complete |
| TOOL-06 | Phase 4 | Complete |
| UI-01 | Phase 4 | Pending |
| UI-02 | Phase 4 | Pending |
| UI-03 | Phase 4 | Complete |
| UI-04 | Phase 4 | Complete |
| UI-05 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 40 total
- Mapped to phases: 40/40 (100%) ✓
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-28*
*Last updated: 2026-02-28 after Phase 1 completion (BUILD-01, BUILD-02, BUILD-03, BUILD-04 complete)*
