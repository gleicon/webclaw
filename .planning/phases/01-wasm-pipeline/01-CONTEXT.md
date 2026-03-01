# Phase 1: WASM Pipeline - Context

**Gathered:** 2026-02-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Compile a Go binary to WASM, load and instantiate it in a browser tab, wire up the `jsFetch` and `jsIndexedDB` JS-to-Go bridges via `syscall/js`, and produce a brotli-compressed artifact. This phase delivers nothing user-visible — it is pure infrastructure. All other phases depend on this working correctly.

The host page (`index.html` + `webclaw-host.js`) is a developer-facing loader, not a user UI. No agent logic, no config, no chat UI belongs here.

</domain>

<decisions>
## Implementation Decisions

### Project structure
- `cmd/webclaw/main.go` as the WASM entry point (single binary target: `GOOS=js GOARCH=wasm`)
- Internal packages under `internal/` (not `pkg/`) — all code is private until Phase 4 when a public API surface is needed
- Bridge code lives in `internal/jsbridge/` — centralized package for all `syscall/js` interaction
- No flat layout — structured from the start to avoid painful refactors when Phases 2–4 add complexity

### JS bridge design
- Centralized `internal/jsbridge` package exposes Go-level interfaces: `Fetch(url, opts) (Response, error)` and `IndexedDB() DBHandle`
- `syscall/js` calls are confined to this package — business logic in other packages never imports `syscall/js` directly
- Bridge functions are registered in `main.go` via `js.Global().Set(...)` on startup, making them callable from JS
- Two bridge surfaces: Go calling JS (via `syscall/js`) and JS calling Go (exported functions)

### Host page scope
- `index.html`: minimal — loads `wasm_exec.js`, `webclaw-host.js`, and starts the WASM module
- `webclaw-host.js`: loads the `.wasm` binary (brotli-decompressed at serve time or pre-decompressed), instantiates via `WebAssembly.instantiateStreaming`, calls `go.run(instance)`
- No styling, no UI elements — developer harness only
- Host page does include a small JS smoke test section: calls `jsFetch` and `jsIndexedDB` from JS console to verify bridges work

### Smoke test strategy
- Verification is browser-based: open `index.html`, open DevTools console, call `window.webclaw.jsFetch("https://example.com")` and `window.webclaw.jsIndexedDB.open("test")`
- Go's `main()` registers a `webclaw.ready` callback callable from JS to confirm WASM initialized
- Round-trip confirmed when: JS calls Go bridge → Go executes JS API → result returned to Go → Go writes result back to JS
- No external test framework in Phase 1 — manual console verification is sufficient

### Brotli artifact
- Build script (`Makefile` or `build.sh`) runs `brotli --best` on the compiled `.wasm` file
- Output: `dist/webclaw.wasm` (raw) and `dist/webclaw.wasm.br` (brotli-compressed)
- Host page loads `.wasm.br` if the server sends `Content-Encoding: br`; falls back to `.wasm` for local file:// development
- Brotli tool: system `brotli` CLI (not a Go implementation) — keep build dependencies minimal

### Go toolchain
- Standard Go (`GOOS=js GOARCH=wasm`) — NOT TinyGo (explicitly out of scope per requirements)
- `wasm_exec.js` ships with the Go standard library — copy it into `static/` as part of build
- Go version: whatever is current stable (1.22+); document in `go.mod`

### Claude's Discretion
- Exact Makefile targets and build script structure
- Whether `webclaw-host.js` uses `fetch()` or `XMLHttpRequest` for loading the WASM binary
- Error message text in the host page
- Whether to use `go generate` for the build step
- Directory names for static assets (`static/`, `web/`, `dist/`)

</decisions>

<specifics>
## Specific Ideas

- The `jsFetch` bridge must route ALL outbound HTTP through JS `fetch()` — no `net/http` usage ever (this is enforced by Go's WASM target not having a working `net` stack, but the bridge makes the pattern explicit and reusable for Phase 3)
- The `jsIndexedDB` bridge is a thin wrapper in Phase 1 — just enough to prove callability. Full IndexedDB operations (config storage, memory) happen in Phases 2 and 3
- NullClaw's edge deployment pattern (Cloudflare Worker wraps WASM module) is a useful reference for how the host/WASM boundary should be designed, even though WebClaw targets browser tabs directly

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — this is a greenfield project. No existing components.

### Established Patterns
- The `wasm_exec.js` runtime support file is provided by the Go SDK — it must be served alongside the WASM binary
- Go's `syscall/js` package is the only sanctioned way to call JS APIs from Go WASM — no CGo, no WASI

### Integration Points
- `main.go` is the WASM init: registers JS-callable functions and blocks on a channel to keep the Go runtime alive
- `webclaw-host.js` is the JS init: fetches WASM, instantiates, runs Go, then calls any post-init JS hooks
- Future phases (2-4) will import `internal/jsbridge` for all browser API access

</code_context>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope. All WASM infrastructure decisions needed for this phase are captured above.

</deferred>

---

*Phase: 01-wasm-pipeline*
*Context gathered: 2026-02-28*
