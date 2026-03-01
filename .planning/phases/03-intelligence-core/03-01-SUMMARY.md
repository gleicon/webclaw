---
phase: 03-intelligence-core
plan: 01
subsystem: llm-provider
tags: [anthropic, openai, openrouter, sse, wasm, streaming]

# Dependency graph
requires:
  - phase: 02-config-identity
    provides: Config system, API key storage
provides:
  - Provider interface for Anthropic, OpenAI, OpenRouter
  - SSE streaming via syscall/js fetch bridge
  - vendor/model-id routing with fallback inference
  - Exponential backoff retry with failover chains
affects:
  - 03-agent-loop
  - 03-memory-system
  - 04-webchat-ui

# Tech tracking
tech-stack:
  added: [syscall/js fetch streaming, SSE parser, provider chain pattern]
  patterns: [Strategy pattern for providers, Chain of Responsibility for failover]

key-files:
  created:
    - internal/provider/provider.go - Core interface and SSE parser
    - internal/provider/anthropic.go - Anthropic Messages API with streaming
    - internal/provider/openai.go - OpenAI Chat Completions with embeddings
    - internal/provider/openrouter.go - OpenRouter multi-model routing
    - internal/provider/router.go - vendor/model-id parsing and routing
    - internal/provider/failover.go - Exponential backoff and provider chains
    - internal/jsbridge/streaming.go - JS ReadableStream to Go channel bridge
  modified:
    - internal/jsbridge/fetch.go - Enhanced with full HTTP options
    - internal/jsbridge/bridge.go - Updated to use RegisterFetchCallback

key-decisions:
  - "All HTTP calls use syscall/js fetch bridge - no net/http in WASM"
  - "SSE parsing implemented in Go for cross-browser compatibility"
  - "Provider chain pattern allows primary→fallback with transparent failover"
  - "Router infers vendor from model names (claude→anthropic, gpt→openai)"

patterns-established:
  - "Provider interface: Complete(), Stream(), Embed(), Name(), MaxContextWindow()"
  - "Streaming returns <-chan Token for backpressure and cancellation"
  - "RetryConfig for exponential backoff: 1s, 2s, 4s with jitter consideration"
  - "ParseModelID handles vendor/model-id and nested openrouter/anthropic/model"

requirements-completed:
  - PROV-01
  - PROV-02
  - PROV-04
  - PROV-05

# Metrics
duration: 4min
completed: 2026-03-01
---

# Phase 03 Plan 01: LLM Provider System Summary

**Complete LLM provider routing with Anthropic Messages API, OpenAI Chat Completions, and OpenRouter multi-model support - all using syscall/js fetch bridge for browser-native streaming.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-01T19:26:33Z
- **Completed:** 2026-03-01T19:31:13Z
- **Tasks:** 7 completed
- **Files created:** 8
- **Files modified:** 2

## Accomplishments

- Provider interface with Complete(), Stream(), Embed(), Name(), MaxContextWindow() methods
- Anthropic provider with Messages API and SSE streaming (content_block_delta, message_stop events)
- OpenAI provider with Chat Completions, embeddings, and [DONE] terminator handling
- OpenRouter provider with HTTP-Referer and X-Title headers, multi-model routing
- Router with vendor/model-id parsing and model name inference
- Failover chains with exponential backoff (1s, 2s, 4s) and provider fallback
- JS fetch bridge with streaming support via ReadableStream reader

## Task Commits

1. **Task 1: Define Provider Interface** - `3029c4d` (feat)
2. **Task 2: JS Fetch Bridge** - `7cb9d3a` (feat) - also fixed idb_memory.go return values
3. **Task 3: Anthropic Provider** - `3441851` (feat)
4. **Task 4: OpenAI Provider** - `edc0bc3` (feat)
5. **Task 5: OpenRouter Provider** - `733127b` (feat)
6. **Task 6: Provider Router** - `59f7162` (feat)
7. **Task 7: Failover with Exponential Backoff** - `a0dda6a` (feat)

## Files Created/Modified

### Created
- `internal/provider/provider.go` - Core interface, Token/Message types, SSEParser
- `internal/provider/anthropic.go` - Anthropic Messages API, streaming parser
- `internal/provider/openai.go` - OpenAI Chat Completions, embeddings
- `internal/provider/openrouter.go` - OpenRouter with multi-model support
- `internal/provider/router.go` - vendor/model-id routing, model inference
- `internal/provider/failover.go` - ProviderChain, RetryConfig, exponential backoff
- `internal/jsbridge/streaming.go` - StreamingReader, SSEStreamingReader

### Modified
- `internal/jsbridge/fetch.go` - Enhanced Fetch() with headers/body, FetchStream()
- `internal/jsbridge/bridge.go` - Updated to use RegisterFetchCallback()
- `internal/jsbridge/idb_memory.go` - Fixed return value issues
- `internal/memory/idb_store.go` - Moved to appropriate location

## Decisions Made

- **All HTTP through syscall/js:** No net/http imports in provider package - uses jsbridge.Fetch() exclusively
- **SSE parsing in Go:** Rather than relying on EventSource API, implement SSE parsing for flexibility with different provider formats
- **Provider chains for failover:** Primary → Retry (3x) → Fallback pattern handles both transient errors and provider outages
- **Model name inference:** Router can identify vendor from model names (claude-* → anthropic, gpt-* → openai) for convenience

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed package organization for memory/jsbridge files**
- **Found during:** Task 2 (JS Fetch Bridge)
- **Issue:** idb_store.go and idb_memory.go were in wrong directories causing package conflicts
- **Fix:** Moved idb_store.go to jsbridge/, idb_memory.go to memory/ with correct package declarations
- **Files modified:** internal/jsbridge/idb_memory.go, internal/memory/idb_store.go
- **Verification:** GOOS=js GOARCH=wasm go build passes
- **Committed in:** 7cb9d3a

**2. [Rule 1 - Bug] Fixed return value mismatches in idb_memory.go**
- **Found during:** Task 2 (JS Fetch Bridge)
- **Issue:** Some js.FuncOf callbacks returned `return` instead of `return nil`
- **Fix:** Changed `return` to `return nil` in 3 locations
- **Files modified:** internal/jsbridge/idb_memory.go
- **Verification:** Build passes with no errors
- **Committed in:** 7cb9d3a

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** No scope creep - fixes required for compilation

## Issues Encountered

None - all code compiled successfully after auto-fixes.

## User Setup Required

None - no external service configuration required. API keys will be configured through the identity system (Phase 2).

## Next Phase Readiness

- ✅ Provider routing complete
- ✅ Streaming infrastructure ready
- ✅ Retry/failover patterns established
- ⚠️  Ready for 03-02: Agent Loop integration
- ⚠️  Ready for 03-03: Memory System integration

---
*Phase: 03-intelligence-core*
*Completed: 2026-03-01*
