# WebClaw

## What This Is

WebClaw is a TinyGo-compiled WASM AI agent runtime that brings the OpenClaw spec natively into the browser. It runs entirely client-side with no server dependency, using IndexedDB for persistent memory, the browser's fetch() API for LLM provider calls, and a JS/TS plugin system for extensibility. An optional local bridge binary (`webclaw-bridge`) unlocks file I/O, shell execution, and git operations via WebSocket when needed.

## Core Value

A fully functional OpenClaw-compatible AI assistant that boots from a URL in a browser tab — zero install, instant distribution, no Node.js or server required.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] TinyGo WASM core compiles to <2MB and loads in a browser tab
- [ ] Agent loop: context assembly (identity files + history) → LLM dispatch → tool execution → memory management
- [ ] Provider abstraction routes `vendor/model-id` strings to API calls via browser fetch()
- [ ] OpenClaw config format (JSON5 camelCase + snake_case variants) parsed inside WASM
- [ ] Identity file system (IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md) loaded from IndexedDB virtual FS
- [ ] Memory system: hybrid vector + BM25 keyword search over IndexedDB, 0.7/0.3 weighted merge
- [ ] API keys encrypted at rest via Web Crypto AES-GCM, decrypted inside WASM linear memory
- [ ] Browser tool profile: web_search, web_fetch, memory_search, memory_store
- [ ] JS/TS plugin SDK: registerTool, registerHook, registerChannel, registerService
- [ ] Local agent bridge binary (`webclaw-bridge`): file I/O, shell exec, git, CDP, cron via ws://localhost:18800
- [ ] Bridge pairing: 6-digit OTP + bearer token + 127.0.0.1-only binding
- [ ] Service Worker mode for background heartbeat and tab-close persistence
- [ ] OpenClaw/PicoClaw/NullClaw config migration (`webclaw migrate`)
- [ ] Webchat channel: built-in browser UI for interacting with the agent

### Out of Scope

- Native messaging channels (Telegram, Discord, Slack, WhatsApp) — WebClaw IS the channel (webchat + bridge)
- Mobile app — browser-first, mobile later
- DM pairing flow — single-user browser context, not applicable
- WASM SIMD128 for vector math — future optimization, not v1
- Full OpenClaw plugin API compatibility — Node.js API cannot run in browser; JS/TS SDK is the replacement

## Context

WebClaw fills the only gap in the OpenClaw ecosystem: no implementation runs in a browser. The family tree:
- **OpenClaw** (TypeScript, 240K stars): runs as a Gateway process, >1GB RAM, >500s boot on edge hardware
- **PicoClaw** (Go, 21K stars): <10MB, <1s boot, targets RISC-V/ARM SBCs — no WASM
- **NullClaw** (Zig, 2.6K stars): 678KB binary, <8ms boot, has wasmtime runtime but targets Cloudflare Workers edge — not browser-native

WebClaw's insight: the browser is not a limitation — it's an advantage (instant distribution, native DOM canvas, IndexedDB + Web Crypto, vast npm ecosystem). The local bridge cleanly separates "things browsers can do" from "things that need a local process," validated by OpenClaw's own node architecture and NullClaw's Cloudflare Worker edge MVP.

**Technical constraints:**
- TinyGo limitations: no `reflect` (use codegen for config parsing), no `net/http` (use JS fetch imports), limited `syscall/js` (pre-define all import/export signatures)
- CORS is the primary runtime constraint for LLM APIs — most providers (Anthropic, OpenAI, OpenRouter) allow browser-origin requests with API keys
- IndexedDB quotas vary by browser (~50% of disk on Chrome); hygiene triggers at 80% usage

**Target v1 user:** The developer (dogfooding). Proving the concept works for personal use before expanding.

## Constraints

- **Tech stack**: Full Go (`GOOS=js GOARCH=wasm`) for WASM core — not TinyGo; standard Go for bridge binary; TypeScript for plugin SDK
- **WASM size**: ~5-15MB uncompressed, ~3-5MB brotli — acceptable tradeoff for full Go runtime (reflect, encoding/json, goroutines)
- **HTTP in WASM**: `net/http` unavailable in browser Go; all outbound calls go through `syscall/js` fetch() interop
- **Build pipeline**: `GOOS=js GOARCH=wasm go build` → wasm-opt → brotli; standard `go build` for bridge
- **Security**: API keys never in JS plaintext (encrypted via Web Crypto AES-GCM, decrypted inside WASM); bridge binds 127.0.0.1 only; strict CSP

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Full Go WASM over TinyGo | TinyGo lacks `reflect` and `encoding/json` — essential for config parsing and agent flexibility. Binary size tradeoff (~5MB compressed) is acceptable. | ✓ Good |
| Rebuild core cleanly (not fork PicoClaw) | PicoClaw's structs are shaped around its channel SDKs and design assumptions. Clean rebuild lets us design around browser constraints from day one, informed by PicoClaw's patterns without inheriting its baggage. | ✓ Good |
| Bridge interface: REST + WebSocket | REST for simple ops (file read, secrets) — curl-testable and easy to debug. WebSocket for streaming (shell exec, live output). Separation of concerns. | ✓ Good |
| IndexedDB over localStorage | Capacity (localStorage ~5MB limit); structured storage; async API fits WASM async model | — Pending |
| BM25 instead of FTS5 | No SQLite in browser by default; BM25 achieves comparable results for <10K doc corpus | — Pending |
| JS/TS plugin SDK over Node.js compatibility | Browser environment requires native JS APIs; attempting Node.js compat adds too much complexity | — Pending |

---
*Last updated: 2026-02-28 after initialization*
