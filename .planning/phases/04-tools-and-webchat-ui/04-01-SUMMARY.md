---
phase: 04-tools-and-webchat-ui
plan: 01
subsystem: tools
tags: [tools, registry, agent-loop, provider-routing, tool-dispatch]
dependency_graph:
  requires:
    - 03-01 (provider routing)
    - 03-02 (agent loop, worker bridge)
    - 03-04 (memory system)
  provides:
    - Tool and ToolResult types
    - Registry with dispatch and Anthropic schema generation
    - web_fetch, web_search, memory_store, memory_search tools
    - AgentLoop tool dispatch loop with EmitToolEvent integration
  affects:
    - internal/agent/loop.go (Provider interface signature changed)
    - internal/provider/provider.go (Token struct extended with tool fields)
tech_stack:
  added:
    - internal/tools/ package (tool.go, registry.go, web_fetch.go, web_search.go, memory_tools.go)
  patterns:
    - MemoryAgent interface to avoid tools/agent circular import
    - WorkerBridgeIface as anonymous interface in AgentLoop struct field
    - providerAdapter bridging channel-based router to callback-based Provider interface
    - Tool dispatch loop up to maxToolIterations=10 with graceful iteration limit
key_files:
  created:
    - internal/tools/tool.go
    - internal/tools/registry.go
    - internal/tools/registry_test.go
    - internal/tools/web_fetch.go
    - internal/tools/web_search.go
    - internal/tools/memory_tools.go
    - internal/tools/tools_meta_test.go
  modified:
    - internal/agent/loop.go
    - internal/agent/worker_bridge.go
    - internal/agent/summarizer.go
    - internal/provider/provider.go
decisions:
  - "MemoryAgent interface in tools package avoids circular import between tools and agent"
  - "WorkerBridgeIface as anonymous interface field avoids agent importing itself"
  - "Provider interface callback changed from func(string) to func(provider.Token) to carry tool_use metadata"
  - "Tool dispatch loop in Run() handles tool_use up to maxToolIterations before returning error"
  - "providerAdapter bridges provider.Router channel-based stream to Agent callback-based interface"
metrics:
  duration_minutes: 6
  completed_date: "2026-03-01"
  tasks_completed: 4
  files_created: 7
  files_modified: 4
---

# Phase 4 Plan 1: Tool Registry and Agent Dispatch Loop Summary

**One-liner:** Go tool registry with web_fetch/web_search/memory tools and Agent tool dispatch loop wired to real provider.Router via providerAdapter.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Tool types and registry | 03212a9 | internal/tools/tool.go, registry.go, registry_test.go |
| 2 | Four browser tool implementations | 4bac30e | web_fetch.go, web_search.go, memory_tools.go, tools_meta_test.go |
| 3 | Wire real provider router into AgentLoop | a6490ab | loop.go (SetRouter, providerAdapter), provider.go (Token fields) |
| 4 | Agent loop tool dispatch loop | a6490ab | loop.go (dispatch loop, EmitToolEvent, maxToolIterations) |

## What Was Built

### internal/tools/ package

A new package providing the tool execution infrastructure:

- **tool.go** (`//go:build js && wasm`): `Tool` struct (Name, Description, InputSchema, Execute func) and `ToolResult` struct (Content, DisplayContent, IsError, ToolName, Status) — dual-output design where Content feeds the LLM and DisplayContent shows in the UI.

- **registry.go** (`//go:build js && wasm`): `Registry` with `sync.RWMutex`-protected tool map. `Register()`, `Dispatch()`, `ToAPISchema()` (Anthropic-format schema), `List()`.

- **web_fetch.go**: `NewWebFetchTool()` — fetches URLs via `jsbridge.Fetch()` (never `net/http`). Truncates DisplayContent to 200 chars.

- **web_search.go**: `NewWebSearchTool()` — fetches `https://html.duckduckgo.com/html/?q={query}` via jsbridge.Fetch, extracts up to 5 result titles/snippets via substring parsing (no html.Parse which is unavailable in WASM stdlib).

- **memory_tools.go**: `NewMemoryStoreTool(MemoryAgent)` and `NewMemorySearchTool(MemoryAgent)` — accept a `MemoryAgent` interface (StoreFact + SearchMemory) to avoid a circular import between `tools` and `agent`.

### Tests

- **registry_test.go** (`//go:build !js`): Inline registry contract tests — Dispatch returns result for registered tool, error for unknown, ToAPISchema returns Anthropic-compatible schema, List returns all names.

- **tools_meta_test.go** (`//go:build !js`): Metadata tests for all four tool constructors — validates Name, required params in InputSchema, empty-param error behavior.

### internal/agent/loop.go Changes

**Provider interface** — callback signature changed from `func(string)` to `func(provider.Token)` to carry tool_use metadata (FinishReason, ToolName, ToolInput, ToolUseID) back to Run().

**AgentLoop struct additions:**
- `router *provider.Router` — set via `SetRouter()`, enables real LLM calls
- `toolRegistry *tools.Registry` — set via `SetToolRegistry()`, dispatches tool calls
- `workerBridge interface{ EmitToolEvent(string, string, string, string) }` — set via `SetWorkerBridge()`, emits tool events to UI

**providerAdapter** — wraps `provider.Router` (channel-based) to satisfy the `Provider` interface (callback-based). Converts `agent.Message` slices to `provider.Message` slices.

**Tool dispatch loop in Run():**
1. Stream from provider, capturing last token
2. If `lastToken.FinishReason != "tool_use"` → normal completion, break
3. If `tool_use` → emit "running" via workerBridge, Dispatch() through toolRegistry, emit "done"/"error"
4. Inject tool_use + tool_result as JSON into message history
5. Loop up to `maxToolIterations = 10`; return error if limit reached

### internal/agent/worker_bridge.go

Added `EmitToolEvent(toolName, status, summary, full string)` method to `WorkerBridge` — posts a JS object with `{type: "tool_event", toolName, status, summary, full}` via `postMessage`.

### internal/provider/provider.go

Extended `Token` struct with tool-use fields: `ToolName string`, `ToolInput map[string]interface{}`, `ToolUseID string`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing] MemoryAgent interface instead of *agent.AgentLoop parameter**
- **Found during:** Task 2
- **Issue:** Plan specified `NewMemoryStoreTool(loop *agent.AgentLoop)` but `agent` imports `tools` in Task 3, creating a circular import
- **Fix:** Created `MemoryAgent` interface in tools package with `StoreFact` and `SearchMemory` — `AgentLoop` satisfies it without importing `tools`
- **Files modified:** internal/tools/memory_tools.go
- **Commit:** 4bac30e

**2. [Rule 1 - Bug] Provider callback signature change propagated to summarizer.go**
- **Found during:** Task 3
- **Issue:** Changing `Provider.Stream` callback from `func(string)` to `func(provider.Token)` broke summarizer.go (uses Provider interface)
- **Fix:** Updated summarizer.go Stream callbacks and mockSummarizerProvider.Stream to use `provider.Token` (linter applied this automatically)
- **Files modified:** internal/agent/summarizer.go
- **Commit:** a6490ab

**3. [Rule 2 - Missing] getProvider() called once before tool loop**
- **Found during:** Task 4
- **Issue:** The plan pseudocode showed `getProvider()` inside the loop iteration, but the provider doesn't need to change between tool calls
- **Fix:** getProvider() called once before the loop; same provider instance used for all iterations (simpler and correct)
- **Impact:** Minor structural deviation, same behavior

## Self-Check: PASSED

Files created/modified:
- FOUND: internal/tools/tool.go
- FOUND: internal/tools/registry.go
- FOUND: internal/tools/registry_test.go
- FOUND: internal/tools/web_fetch.go
- FOUND: internal/tools/web_search.go
- FOUND: internal/tools/memory_tools.go
- FOUND: internal/tools/tools_meta_test.go
- FOUND: internal/agent/loop.go
- FOUND: internal/agent/worker_bridge.go
- FOUND: internal/agent/summarizer.go
- FOUND: internal/provider/provider.go

Commits verified:
- 03212a9: feat(04-01): add Tool, ToolResult types and Registry
- 4bac30e: feat(04-01): add four browser tool implementations and metadata tests
- a6490ab: feat(04-01): wire real provider router and implement tool dispatch loop
