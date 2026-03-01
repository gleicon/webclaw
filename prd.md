# WebClaw: browser-native OpenClaw for the ephemeral edge

**The OpenClaw ecosystem has fractured into a constellation of lightweight reimplementations—PicoClaw (Go, <10MB), NullClaw (Zig, 678KB), ZeroClaw (Rust, 3.4MB)—all targeting hardware where the 240K-star TypeScript original cannot run.** None of them run in a browser. WebClaw fills this gap: a TinyGo-compiled WASM agent runtime that implements the OpenClaw spec natively in the browser, with a JS/TS plugin system and a local bridge for machine-level commands. This report documents every component of the three reference implementations, their compatibility surface, and synthesizes a complete WebClaw product design.

---

## The OpenClaw family tree and what the spec actually defines

OpenClaw (formerly Clawdbot, then Moltbot) is Peter Steinberger's TypeScript personal AI assistant, now at **240K GitHub stars** with **46K forks**. It runs as a single Gateway process on `ws://127.0.0.1:18789`, routing messages between messaging channels (WhatsApp, Telegram, Slack, Discord, Signal, iMessage, Teams, Matrix, and more) and an embedded LLM agent runtime built on `@mariozechner/pi-coding-agent`. The "spec" is not a formal RFC—it is the de facto standard set by OpenClaw's config schema, workspace layout, tool/skill interfaces, and identity files. Every derivative project implements some subset of this surface.

**The OpenClaw spec surface, concretely:**

The config lives at `~/.openclaw/openclaw.json` (JSON5, Zod-validated). Top-level keys are `identity`, `agents`, `models`, `channels`, `tools`, `skills`, `plugins`, `session`, `security`, `gateway`, `browser`, `cron`, `hooks`, `secrets`, `logging`, `bindings`, `messages`, `ui`, `auth`, `env`, and `meta`. Provider configuration sits under `models.providers` with per-provider `apiKey`/`baseUrl`/`models[]` blocks. Channel configuration uses `channels.<platform>` with `enabled`, `botToken`/`token`, `dmPolicy` (`open`|`allowlist`|`pairing`), and `allowFrom[]`. The agent defaults (`agents.defaults`) specify `workspace`, `model.primary`, `model.fallbacks`, `bootstrapMaxChars` (**20K per file**), `bootstrapTotalMaxChars` (**150K total**), heartbeat schedule, sandbox mode, and tool profiles.

The **workspace bootstrap files**—the identity layer—are the most replicated part of the spec. Every derivative loads these markdown files into the system prompt each turn: `IDENTITY.md` (persona), `SOUL.md` (behavioral traits), `USER.md` (user knowledge), `AGENTS.md` (agent instructions), `TOOLS.md` (tool guidance), `HEARTBEAT.md` (proactive checklist), and `memory/MEMORY.md` (persistent memory). This file-based identity system is OpenClaw's most portable interface.

The **tool surface** includes `read`, `write`, `edit`, `exec`, `web_search`, `web_fetch`, `browser`, `canvas`, `cron`, `message`, `memory_search`, `memory_get`, `sessions_spawn`, `tts`, and `lobster` (workflow shell). Tools have profiles (`minimal`, `coding`, `messaging`, `full`) with cascading allow/deny policies. **Skills** are SKILL.md files (YAML frontmatter + natural language instructions) distributed via ClawHub (**13,729+ community skills**). The Plugin SDK exports TypeScript interfaces for `registerService`, `registerHook`, `registerTool`, with `configSchema` for UI-driven settings.

OpenClaw's **agent loop** is a six-layer stack: turn orchestration → fallback loop → model fallback → auth + retry → single attempt → streaming handler. Context assembly loads workspace files (capped at 150K tokens total), appends history, injects relevant skills (not all skills—selective injection per turn), and dispatches to the provider. Compaction triggers a memory flush (promoting durable info into `memory/*.md` files) before summarization. The **subagent system** allows spawning background agents (`maxSpawnDepth: 2`, `maxChildrenPerAgent: 5`) with progressively restricted tool access.

---

## PicoClaw dissected: Go minimalism on $10 hardware

PicoClaw launched **February 9, 2026** and hit 21K stars in three weeks. Built by Sipeed—a Chinese maker of RISC-V boards—it targets their **$9.90 LicheeRV-Nano** (256MB RAM, WiFi6) as the minimum deployment platform. The tagline: **<10MB RAM, <1s boot, single binary across RISC-V/ARM/x86**.

**Source architecture** follows a clean Go layout: `cmd/picoclaw/main.go` dispatches CLI commands, with internal packages under `cmd/picoclaw/internal/` (agent, auth, cron, gateway, migrate, onboard, skills, status, version) and shared libraries under `pkg/` (agent, config, cron, heartbeat, logger, providers, skills, state, tools). The `go.mod` pins Go 1.25.7 with direct dependencies on `openai/openai-go`, `anthropics/anthropic-sdk-go`, `telego` (Telegram), `discordgo`, `slack-go`, `botgo` (QQ), `dingtalk-stream-sdk-go`, `larksuite/oapi-sdk-go` (Feishu), `gorilla/websocket`, and `cobra` for CLI.

**Config format** (`~/.picoclaw/config.json`) uses flat JSON with four top-level sections: `agents` (workspace, model, max_tokens, temperature, max_tool_iterations, restrict_to_workspace, model_list), `providers` (openrouter, anthropic, openai, zhipu, gemini, groq, vllm—each with api_key/api_base), `channels` (telegram, discord, whatsapp, feishu, qq, dingtalk, line, slack, onebot—each with enabled/token/allow_from), and `tools` (web search config, cron timeout). Three-tier loading: `DefaultConfig()` → file → `PICOCLAW_` environment variables. The `vendor/model-id` format (e.g., `anthropic/claude-sonnet-4-5`) in `model_list` drives automatic provider routing via `CreateProviderFromConfig()`.

**The agent loop** (`pkg/agent/loop.go`) is simpler than OpenClaw's: build context (system prompt from workspace files + history + optional summary) → call LLM → if tool_calls in response, execute via ToolRegistry with security validation → append results → loop (up to `max_tool_iterations`, default 20) → summarize when history exceeds 20 messages or 75% of context window. Summarization is multi-part parallel. **No memory flush step** (unlike OpenClaw). Sessions persist as JSON files in `~/.picoclaw/workspace/sessions/`.

**Memory is file-based only.** `MEMORY.md` for long-term knowledge, session JSON for history, automatic summarization for context management. **No vector search, no embeddings, no SQLite.** This is the biggest architectural gap versus NullClaw and OpenClaw.

**Tools** (~12): ReadFile, WriteFile, ListDir, Exec, WebSearch (Brave/DuckDuckGo with auto-fallback), WebFetch, Cron, Spawn, Message, Memory. Tool results have dual output: `ForLLM` (feeds next iteration) and `ForUser` (display). Security: workspace sandboxing (`restrict_to_workspace: true`), dangerous command blocking (`rm -rf`, `sudo`, `dd`, fork bombs—always enforced), path validation, channel allowlists.

**No WASM support.** No TinyGo. No browser target. No Cloudflare Workers. No WASI. PicoClaw compiles only to native binaries via standard Go: linux/{amd64,arm64,riscv64,loong64,arm}, darwin/arm64, windows/amd64. The Makefile has no WASM build target. Edge deployment means literally running on Sipeed's boards, Raspberry Pis, Android phones via Termux, or Docker/Kubernetes (health endpoints `/health` and `/ready` added in v0.1.2). **PicoLM**, a companion ~2,500-line C inference engine, enables fully offline operation with TinyLlama 1.1B.

**Skills system** uses SKILL.md marker files. Four built-in skills: weather, github, summarize, skill-creator (meta-skill). CLI: `picoclaw skill install/enable/disable/list/update/remove`. Skills can be installed from GitHub repos and packaged as `.skill` zip files. Compatible with OpenClaw's skill ecosystem in concept but not in plugin API.

**License: MIT.** OpenClaw workspace migration exists via `cmd/picoclaw/internal/migrate/`.

---

## NullClaw dissected: Zig at the absolute performance frontier

NullClaw pushes the "smallest possible agent" concept to its logical extreme: **678KB static binary, ~1MB peak RSS, <2ms boot on Apple Silicon, <8ms on 0.8GHz edge hardware.** Written in **~45,000 lines of Zig** across **~110 source files** with **3,230+ tests**. Latest release: v2026.2.26 (CalVer). **2,600 stars**, 312 forks, 17 contributors. MIT licensed. Requires exactly Zig 0.15.2.

**Architecture uses vtable interfaces** for every major subsystem—Zig's zero-cost runtime polymorphism pattern:

```zig
pub const Interface = struct {
    ptr: *anyopaque,
    vtable: *const VTable,
    pub const VTable = struct {
        method: *const fn (ptr: *anyopaque, ...) anyerror!ReturnType,
    };
};
```

**Nine vtable subsystems**: Provider (Anthropic, OpenAI, OpenRouter, Ollama, Gemini, Compatible/41 services, Claude CLI, Codex CLI, Reliable, Router), Channel (CLI, Telegram, Discord, Slack, WhatsApp, Matrix, Signal, IRC, iMessage, Email, Mattermost, LINE, Lark/Feishu, DingTalk, QQ, OneBot, MaixCam—**17 total**), Memory (SQLite/FTS5+vector, Markdown, Lucid, None), Tool (30+ including Shell, FileRead/Write/Edit/Append, Git, HTTP, WebFetch, Browser, Screenshot, MemoryStore/Recall/Forget, Schedule, Cron, Delegate, Spawn, Message, I2C, SPI, HardwareBoardInfo, Composio, MCP), Sandbox (Landlock, Firejail, Bubblewrap, Docker, auto-detect), Runtime (Native, Docker, **WASM**), Tunnel (None, Cloudflare, Tailscale, ngrok, Custom), Observer (Noop, Log, File, Multi), Peripheral (Serial, Arduino, RPi GPIO, STM32/Nucleo).

**The hybrid memory system** is NullClaw's standout feature. It runs **both vector cosine similarity and BM25 keyword search simultaneously** on SQLite, then merges results with configurable weights (default **0.7 vector / 0.3 keyword**). Embeddings stored as compressed BLOBs in SQLite. FTS5 virtual tables for keyword indexing. No external vector database. Hygiene system auto-archives stale memories. Snapshot export/import for portability. The `EmbeddingProvider` vtable supports OpenAI, custom URL, or noop.

**Security is multi-layered**: gateway binds 127.0.0.1 (refuses 0.0.0.0 without tunnel or explicit override), 6-digit OTP pairing flow, filesystem scoping with symlink escape detection, auto-detected sandbox (Landlock → Firejail → Bubblewrap → Docker), **ChaCha20-Poly1305 AEAD** encrypted secrets, configurable resource limits, cryptographically signed audit trail (90-day retention), HTTPS-only outbound, per-channel deny-by-default allowlists, cost auditing with daily budget thresholds.

**WASM story is real but limited.** NullClaw has a WASM runtime adapter (`RuntimeAdapter` vtable with `wasm` kind, using wasmtime). An edge MVP example lives at `examples/edge/cloudflare-worker/`—a hybrid pattern where networking/secrets live in the Cloudflare Worker host and agent logic runs as a replaceable Zig WASM module. This is the closest any project in the ecosystem gets to browser-adjacent deployment.

**OpenClaw compatibility**: NullClaw uses the same config structure as OpenClaw but with snake_case instead of camelCase. Providers under `models.providers`, default model under `agents.defaults.model.primary`, channels use `accounts` wrappers (`channels.telegram.accounts.main`). `nullclaw migrate openclaw` imports memory from OpenClaw workspaces into local SQLite vectors. **AIEOS v1.1** (AI Entity Object Specification) is NullClaw's JSON-based identity format—a programmatically validatable alternative to OpenClaw's markdown IDENTITY.md, covering psychology, linguistics, motivations, and ethical boundaries.

---

## How the three implementations compare at a glance

| Dimension | OpenClaw | PicoClaw | NullClaw |
|---|---|---|---|
| Language | TypeScript | Go | Zig |
| Binary size | ~28 MB (dist) | ~8 MB | 678 KB |
| RAM | >1 GB | <10 MB | ~1 MB |
| Boot (0.8 GHz) | >500s | <1s | <8ms |
| GitHub stars | 240K | 21K | 2.6K |
| Providers | 12+ core | 13+ | 22+ core, 50+ total |
| Channels | 14+ | 10+ | 17 |
| Memory backend | File (markdown) + vector | File only (markdown) | SQLite hybrid (vector + FTS5) |
| Plugin system | Full SDK (TypeScript) | Skills only (SKILL.md) | vtable interfaces (Zig) |
| WASM | None | None | Yes (wasmtime runtime) |
| Edge deployment | Node.js servers | Native binary on SBCs | Cloudflare Workers example |
| Security | DM pairing, Docker sandbox | Workspace sandbox, cmd blocking | Landlock/Firejail/Bubblewrap, encrypted secrets |
| Identity format | Markdown (IDENTITY.md) | Markdown (IDENTITY.md) | Markdown + AIEOS v1.1 JSON |
| Config format | JSON5, camelCase | JSON, snake_case | JSON, snake_case (OpenClaw-compatible) |
| License | MIT | MIT | MIT |

---

## WebClaw product design: browser-native, ephemeral, OpenClaw-compatible

WebClaw is a **WASM-first AI assistant runtime** that brings the OpenClaw spec into the browser. Compiled from Go using TinyGo, it runs entirely client-side with no server dependency, using a JS/TS plugin system for extensibility and an optional local agent bridge for machine-level operations.

### Core architecture

WebClaw's runtime compiles to a single `.wasm` module (**target: <2MB**) loaded by a thin JavaScript host. The host provides browser APIs (fetch, IndexedDB, Web Crypto, WebSocket, Service Worker) through WASM imports; the Go core implements the agent loop, config parsing, provider abstraction, and memory system.

```
┌─────────────────────────────────────────────────┐
│                    Browser Tab                    │
│  ┌───────────────────────────────────────────┐   │
│  │           JS/TS Plugin Runtime             │   │
│  │  (plugins, channel adapters, UI hooks)     │   │
│  └─────────────────┬─────────────────────────┘   │
│                    │ wasm-bindgen / JS interop     │
│  ┌─────────────────▼─────────────────────────┐   │
│  │         WebClaw Core (TinyGo → WASM)       │   │
│  │  ┌─────────┐ ┌──────────┐ ┌────────────┐  │   │
│  │  │ Agent   │ │ Provider │ │   Memory    │  │   │
│  │  │  Loop   │ │ Abstrac. │ │ (IndexedDB  │  │   │
│  │  │         │ │          │ │  + vectors) │  │   │
│  │  └─────────┘ └──────────┘ └────────────┘  │   │
│  │  ┌─────────┐ ┌──────────┐ ┌────────────┐  │   │
│  │  │ Config  │ │ Identity │ │   Tools    │  │   │
│  │  │ Parser  │ │ Loader   │ │ (sandboxed) │  │   │
│  │  └─────────┘ └──────────┘ └────────────┘  │   │
│  └───────────────────────────────────────────┘   │
│                    │ WebSocket                     │
│  ┌─────────────────▼─────────────────────────┐   │
│  │     Local Agent Bridge (optional)          │   │
│  │  ws://localhost:18800                       │   │
│  │  File I/O, shell exec, browser control     │   │
│  └───────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

**Three execution modes:**
1. **Pure browser** (ephemeral): All state in IndexedDB. No local bridge. Tools limited to web_search, web_fetch (via CORS proxy or service worker), memory operations, and JS-defined plugin tools. Sessions vanish when storage is cleared.
2. **Bridge-connected** (hybrid): WebSocket to `localhost:18800` local agent bridge (a small Go binary). Gains file I/O, shell exec, git, browser control via CDP, cron scheduling, and full OpenClaw tool compatibility.
3. **Service Worker** (persistent): Registers a Service Worker for background heartbeat execution, push notification handling, and offline-first operation. Agent survives tab closure.

### OpenClaw config compatibility layer

WebClaw parses both OpenClaw's camelCase JSON5 and PicoClaw/NullClaw's snake_case JSON. The config lives in IndexedDB under the key `webclaw:config` and can be imported/exported as a JSON file. A `webclaw migrate` function reads OpenClaw, PicoClaw, or NullClaw config files and transforms them into WebClaw's canonical format.

**Config schema** (superset, browser-adapted):

```json5
{
  identity: { name: "Clawd", emoji: "🦞", theme: "helpful lobster" },
  agents: {
    defaults: {
      model: { primary: "anthropic/claude-sonnet-4-5", fallbacks: [] },
      maxToolIterations: 20,
      temperature: 0.7,
      bootstrapMaxChars: 20000,
      bootstrapTotalMaxChars: 150000
    }
  },
  models: {
    providers: {
      anthropic: { apiKey: "sk-...", baseUrl: "https://api.anthropic.com" },
      openrouter: { apiKey: "sk-or-...", baseUrl: "https://openrouter.ai/api/v1" }
      // Keys encrypted with Web Crypto API (AES-GCM) in IndexedDB
    }
  },
  channels: {
    webchat: { enabled: true },       // Built-in browser UI
    bridge: { enabled: false, url: "ws://localhost:18800" }
  },
  tools: {
    profile: "browser",               // New profile: web_search, web_fetch, memory, canvas
    bridge: { enabled: false }         // Enables file/exec/git tools via bridge
  },
  memory: {
    backend: "indexeddb",              // or "bridge-sqlite" when connected
    vectorWeight: 0.7,
    keywordWeight: 0.3,
    embeddingProvider: "auto"          // Uses LLM provider's embedding endpoint
  },
  plugins: { entries: {}, load: { paths: [] } },
  session: { compaction: { mode: "default", memoryFlush: { enabled: true } } }
}
```

**Identity file compatibility**: WebClaw loads IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md from IndexedDB virtual filesystem (or from the local bridge's real filesystem). Import/export as a zip. Supports NullClaw's AIEOS v1.1 JSON format as an alternative.

### Provider abstraction in WASM

The Provider interface in WebClaw mirrors PicoClaw's `vendor/model-id` routing:

```go
type Provider interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamCompletion(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
    Embeddings(ctx context.Context, texts []string) ([][]float32, error)
    Models() []ModelInfo
}
```

All provider calls route through the browser's `fetch()` API (exposed to WASM via JS import). **CORS is the primary constraint.** Most LLM APIs (OpenAI, Anthropic, OpenRouter) allow browser-origin requests with API keys. For providers that don't, WebClaw routes through its optional CORS proxy Service Worker or the local bridge. Provider failover follows NullClaw's pattern: primary → retries (exponential backoff) → fallback chain → next model.

### Memory system: IndexedDB hybrid search

WebClaw ports NullClaw's hybrid merge strategy to the browser:

- **Vector store**: Embeddings (from provider embedding endpoints) stored as Float32Arrays in IndexedDB object store `webclaw:vectors`. Cosine similarity computed in WASM (pure Go math, no SIMD initially—future: WASM SIMD128).
- **Keyword index**: A TinyGo-compiled BM25 index over memory documents, stored in IndexedDB. No FTS5 (no SQLite in browser by default), but the tf-idf scoring achieves comparable results for the expected corpus size (<10K documents).
- **Hybrid merge**: Same 0.7/0.3 weighted combination as NullClaw. Configurable.
- **Bridge mode**: When connected to local bridge, delegates to SQLite with FTS5+vector (identical to NullClaw's implementation) for full-fidelity search.
- **Memory flush**: Before compaction, the agent promotes durable knowledge into memory documents (matching OpenClaw's behavior).

**Storage budget**: IndexedDB quotas vary by browser (typically 50% of disk on Chrome). WebClaw monitors usage and triggers hygiene (archival/purge of old memories) at 80% of quota.

### JS/TS plugin system

WebClaw's plugin system is the bridge between the Go WASM core and the JavaScript ecosystem. Plugins are ES modules loaded at runtime:

```typescript
// webclaw-plugin.d.ts
interface WebClawPluginAPI {
  registerTool(tool: ToolDefinition): void;
  registerChannel(channel: ChannelDefinition): void;
  registerHook(event: string, handler: HookHandler): void;
  registerService(service: ServiceDefinition): void;
  config: WebClawConfig;
  memory: { search(query: string): Promise<MemoryResult[]>; store(doc: MemoryDocument): Promise<void> };
  agent: { send(message: string): Promise<AgentResponse> };
  bridge: { exec(cmd: string): Promise<ExecResult>; readFile(path: string): Promise<string> } | null;
  logger: Logger;
}

interface ToolDefinition {
  name: string;
  description: string;
  parameters: JSONSchema;
  execute: (params: Record<string, unknown>) => Promise<ToolResult>;
}

// Plugin entry point
export default function(api: WebClawPluginAPI): void | Promise<void>;
```

**Plugin capabilities**: Register custom tools (available to the agent), custom channels (e.g., a WebRTC voice channel), lifecycle hooks (`message:received`, `message:sent`, `tool:before`, `tool:after`, `compaction:before`, `memory:flush`), and background services. Plugins run in the main thread (same origin) with access to DOM, fetch, and Web APIs. **No sandboxing beyond same-origin policy**—plugins are trusted code.

**Compatibility with OpenClaw plugins**: WebClaw can load OpenClaw plugin manifests (`openclaw.plugin.json`) and adapt the `configSchema` for its settings UI. Full API compatibility is not possible (OpenClaw plugins assume Node.js), but tool registrations and config schemas transfer.

### Local agent bridge protocol

The bridge is a lightweight Go binary (`webclaw-bridge`) that runs on the user's machine and exposes a WebSocket server at `ws://localhost:18800`. It implements the tools that cannot run in a browser:

- **File I/O**: `read_file`, `write_file`, `list_dir`, `edit_file` — scoped to a configurable workspace directory
- **Shell execution**: `exec` — with the same safety guards as PicoClaw (dangerous command blocking, workspace scoping)
- **Git operations**: `git_status`, `git_diff`, `git_commit`
- **Browser control**: CDP proxy to a local Chrome instance
- **Cron**: Persistent cron scheduler (survives browser closure)
- **System info**: Hardware discovery, OS details

**Bridge protocol** (JSON-RPC over WebSocket):

```json
{"jsonrpc": "2.0", "method": "tool.exec", "params": {"command": "ls -la", "cwd": "/workspace"}, "id": 1}
{"jsonrpc": "2.0", "result": {"stdout": "...", "stderr": "", "exitCode": 0}, "id": 1}
```

**Security**: Bridge requires pairing (6-digit OTP displayed in terminal, entered in browser). Bearer token stored in IndexedDB (encrypted with Web Crypto AES-GCM). Bridge workspace sandboxing mirrors PicoClaw's `restrict_to_workspace`. Bridge binds 127.0.0.1 only.

### What OpenClaw features WebClaw implements and what it drops

| OpenClaw Feature | WebClaw Status | Notes |
|---|---|---|
| Agent loop + tool dispatch | ✅ Full | Core WASM runtime |
| Provider abstraction | ✅ Full | Via fetch(), CORS-aware |
| Identity files (IDENTITY.md, SOUL.md, etc.) | ✅ Full | IndexedDB virtual FS |
| Memory (file-based + vector search) | ✅ Adapted | IndexedDB + WASM vector math |
| Config format (JSON5) | ✅ Full | Parses both camelCase and snake_case |
| Skills (SKILL.md) | ✅ Full | Loaded from IndexedDB or bridge |
| Tools (read/write/exec) | ⚡ Bridge-dependent | Pure browser mode: web_search, web_fetch, memory only |
| Subagents | ✅ Full | Web Workers for parallel execution |
| Canvas/A2UI | ✅ Native | Browser IS the canvas—renders directly in DOM |
| Browser control (CDP) | ⚡ Bridge-only | Requires local Chrome + bridge |
| Channels (Telegram, Discord, etc.) | ❌ Dropped | WebClaw IS the channel (webchat + bridge) |
| Heartbeat | ✅ Adapted | Service Worker-based scheduling |
| Cron | ⚡ Bridge-dependent | Service Worker for simple timers; bridge for persistent cron |
| Lobster workflows | ⚡ Bridge-dependent | Pipeline execution requires shell access |
| DM pairing | ❌ N/A | Single-user browser context |
| Mobile nodes | ❌ Dropped | WebClaw targets browser, not mobile apps |
| Plugin SDK | ✅ Redesigned | JS/TS native instead of Node.js |

### Performance budget and build pipeline

**TinyGo compilation**: `tinygo build -o webclaw.wasm -target wasm -gc=conservative -opt=2 ./cmd/webclaw/`. Target binary: **<2MB** (gzipped: <800KB). TinyGo's limitations: no `reflect` (use code generation for config parsing), no `net/http` (use JS fetch imports), limited `syscall/js` bridge (pre-define all import/export signatures).

**Build pipeline**:
1. `tinygo build` → `webclaw.wasm`
2. `wasm-opt -O3` → size optimization
3. `brotli` compression → `webclaw.wasm.br` (<800KB)
4. TypeScript plugin SDK compiled with `tsdown`
5. Host JS (`webclaw-host.js`) handles WASM instantiation, JS ↔ Go bridge, plugin loading
6. Bridge binary: standard `go build` → native binary for linux/darwin/windows

**Runtime memory**: Target <8MB WASM linear memory for the core runtime (comparable to PicoClaw's native footprint). IndexedDB for persistent state.

### Security model for the browser context

WebClaw's threat model differs fundamentally from server-side agents:

- **API key protection**: Keys encrypted with AES-256-GCM via Web Crypto API, stored in IndexedDB. Decryption requires a user-set passphrase (PBKDF2-derived key). Keys never exist in plaintext in JavaScript—decryption happens inside WASM memory (not inspectable from JS without deliberate effort).
- **Plugin trust**: Plugins execute in the same origin. No iframe sandbox (would break tool registration). Users must explicitly approve plugin installation. Plugin manifest declares required permissions.
- **Bridge authentication**: OTP pairing + bearer token + 127.0.0.1-only binding. Token rotation on every browser session.
- **Content Security Policy**: Strict CSP headers. Only whitelisted LLM API domains in `connect-src`.
- **Ephemeral mode**: Optional zero-persistence mode—all state in WASM linear memory only, nothing written to IndexedDB. Everything gone on tab close.

### Migration paths from existing implementations

**From OpenClaw**: Export workspace directory as zip → import into WebClaw (maps IDENTITY.md, SOUL.md, etc. into IndexedDB). Config transformation: `openclaw.json` → WebClaw config (key mapping: `models.providers` preserved, `channels` stripped to webchat-only, `tools.profile` → "browser"). Memory files imported; session transcripts optionally imported.

**From PicoClaw**: `~/.picoclaw/config.json` → WebClaw config (direct mapping: `agents`, `providers`, `tools` sections are structurally similar). Workspace files (IDENTITY.md, etc.) import directly. No vector memory to migrate (PicoClaw has none).

**From NullClaw**: Config transformation (snake_case → camelCase normalization). SQLite memory database: export as JSON snapshot via `nullclaw migrate export`, import into WebClaw's IndexedDB. AIEOS identity files import natively.

### Why this design works

The OpenClaw ecosystem proves that the core agent loop—context assembly, LLM dispatch, tool execution, memory management—is language-agnostic and runtime-portable. PicoClaw proved it runs in 10MB of Go. NullClaw proved it runs in 678KB of Zig with full vector memory. WebClaw proves it runs in a browser tab. The spec surface that matters is small: **config format, identity files, tool schemas, and provider routing**. Everything else is runtime-specific adaptation. The JS/TS plugin system solves extensibility without recompiling WASM, and the local bridge recovers the filesystem/shell capabilities that browsers intentionally lack. The result is an AI assistant that boots instantly from a URL, requires zero installation, persists knowledge across sessions via IndexedDB, and—when paired with the bridge—achieves feature parity with PicoClaw's native binary.

## Conclusion

The OpenClaw family demonstrates a clear pattern: the core agent architecture compresses remarkably well (from 240K-star TypeScript monolith to 678KB Zig binary) because the essential logic—prompt assembly, tool dispatch, provider failover, memory recall—is fundamentally simple. **WebClaw's key insight is that the browser is not a limitation but an advantage**: instant distribution (no install), native UI rendering (Canvas/A2UI becomes just DOM manipulation), built-in secure storage (IndexedDB + Web Crypto), and a vast plugin ecosystem (npm/JS). The local bridge pattern—already validated by OpenClaw's own node architecture and NullClaw's Cloudflare Worker edge MVP—cleanly separates "things browsers can do" from "things that need a local process." The ~2MB WASM binary sits between PicoClaw's 8MB and NullClaw's 678KB, which is the right trade-off for a runtime that gains the entire web platform in exchange.