---
phase: 04-tools-and-webchat-ui
plan: 02
subsystem: ui
tags: [wasm, javascript, bridge, worker, indexeddb, keystore, tools, agent]

# Dependency graph
requires:
  - phase: 04-01-tools-and-webchat-ui
    provides: "tools.Registry, NewWebFetchTool, NewWebSearchTool, NewMemoryStoreTool, NewMemorySearchTool, AgentLoop.SetRouter/SetToolRegistry/SetWorkerBridge"
  - phase: 02-config-identity
    provides: "identity.NewStore, keystore.NewKeyStore, jsbridge.RegisterCallback pattern"
  - phase: 03-intelligence-core
    provides: "agent.AgentLoop, provider.Router, worker bridge streaming"
provides:
  - "webclaw.identity.getFile/putFile/listFiles JS exports (Promise-returning)"
  - "webclaw.keystore.setKey/hasKey JS exports (Promise-returning)"
  - "WorkerBridge.EmitToolEvent via onToolEvent callback (not direct postMessage)"
  - "TOOL_EVENT message type in worker.js and webclaw-host.js"
  - "webclaw:tool-event CustomEvent dispatched on window by webclaw-host.js"
  - "globalAgentLoop wired with real provider router, tool registry, and worker bridge"
  - "InitWorkerBridge() returns *WorkerBridge for main.go wiring"
affects: ["04-03-chat-ui", "static/webchat-host.js", "static/worker.js"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "js.FuncOf + Promise pattern for async WASM exports: every JS-facing function returns a Promise wrapping a goroutine"
    - "Callback registration pattern: WASM exposes registerCallback(name, fn) so worker.js can wire event handlers without direct DOM access"
    - "Tool event channel: WASM onToolEvent callback -> worker.js postMessage TOOL_EVENT -> main thread webclaw:tool-event CustomEvent"
    - "globalAgentLoop singleton: pre-configured loop reused per stream, not recreated per request"

key-files:
  created: []
  modified:
    - "cmd/webclaw/main.go"
    - "internal/agent/worker_bridge.go"
    - "static/worker.js"
    - "static/webclaw-host.js"

key-decisions:
  - "v1 keystore passphrase is fixed string 'webclaw-v1-key' — keys are still encrypted at rest but not user-derived; v2 will prompt user"
  - "onToolEvent uses callback pattern (not direct postMessage) so WASM in worker context posts via worker.js, keeping JS boundary clean"
  - "globalAgentLoop singleton in worker_bridge.go; handleStartStream reuses it if set, falls back to fresh unconfigured loop if nil"
  - "InitWorkerBridge() changed to return *WorkerBridge so main.go can call agentLoop.SetWorkerBridge(workerBridgeInstance)"
  - "Identity and keystore bridges registered before InitWorkerBridge in main() so webclaw global exists when bridge functions are set"

patterns-established:
  - "Promise-returning JS bridge: js.FuncOf -> Promise.New -> goroutine -> resolve/reject"
  - "WASM tool event emission: EmitToolEvent -> onToolEvent callback -> worker.js postMessage -> webclaw-host.js CustomEvent"

requirements-completed:
  - UI-03
  - UI-04
  - TOOL-06

# Metrics
duration: 10min
completed: 2026-03-01
---

# Phase 4 Plan 02: JS Bridge Extensions and Agent Loop Wiring Summary

**WASM JS bridge extended with identity file editing (webclaw.identity.*) and API key management (webclaw.keystore.*), tool events routed from WASM callback through worker postMessage to main-thread CustomEvent, and AgentLoop wired with real provider router + tool registry + worker bridge in main.go**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-01T21:04:13Z
- **Completed:** 2026-03-01T21:14:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added webclaw.identity.getFile/putFile/listFiles as Promise-returning WASM exports for chat UI Settings tab
- Added webclaw.keystore.setKey/hasKey as Promise-returning WASM exports for API key management in Settings tab
- Implemented TOOL_EVENT channel: EmitToolEvent -> onToolEvent callback -> TOOL_EVENT postMessage -> webclaw:tool-event CustomEvent
- Wired AgentLoop with real provider.Router, tools.Registry (all 4 browser tools), and WorkerBridge in main.go
- Changed InitWorkerBridge() to return *WorkerBridge so main.go can complete the wiring triangle

## Task Commits

Each task was committed atomically:

1. **Task 1: Tool event emission across worker boundary** - `121c949` (feat)
2. **Task 2: Identity bridge, keystore bridge, and agent loop wiring** - `ce6cf9c` (feat)

## Files Created/Modified

- `cmd/webclaw/main.go` - Added registerIdentityBridge(), registerKeystoreBridge(), agentLoop wiring (SetRouter/SetToolRegistry/SetWorkerBridge), globalAgentLoop registration
- `internal/agent/worker_bridge.go` - Added onToolEvent field, updated EmitToolEvent to use callback, added "onToolEvent" registerCallback case, InitWorkerBridge returns *WorkerBridge, added globalAgentLoop + SetGlobalAgentLoop
- `static/worker.js` - Added TOOL_EVENT to MSG_TYPES, registered onToolEvent callback in registerStreamingCallbacks()
- `static/webclaw-host.js` - Added TOOL_EVENT to MSG_TYPES, added TOOL_EVENT case to handleWorkerMessage dispatching webclaw:tool-event CustomEvent

## Decisions Made

- v1 passphrase for keystore is fixed string `"webclaw-v1-key"` — keys still encrypted at rest (AES-256-GCM), but passphrase not user-derived in v1 for simplicity
- onToolEvent uses callback registration pattern (not direct `self.postMessage`) so the message is always posted from the correct JS context (the worker)
- globalAgentLoop singleton approach: pre-configured loop (with router, tools, workerBridge) stored as package-level var; handleStartStream reuses it if set, falls back to fresh unconfigured loop
- InitWorkerBridge() return type changed from void to *WorkerBridge to enable post-construction wiring in main.go

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] globalAgentLoop singleton required for SetWorkerBridge wiring to take effect**
- **Found during:** Task 2 (main.go wiring)
- **Issue:** handleStartStream created a NEW AgentLoop per stream, discarding the SetRouter/SetToolRegistry/SetWorkerBridge wiring done in main(). The plan showed wiring the setters but the stream handler would ignore them.
- **Fix:** Added `var globalAgentLoop *AgentLoop` and `SetGlobalAgentLoop()` to worker_bridge.go. main() calls `agent.SetGlobalAgentLoop(agentLoop)` after wiring. handleStartStream uses globalAgentLoop if non-nil.
- **Files modified:** internal/agent/worker_bridge.go
- **Verification:** WASM build passes; agentLoop.SetRouter/SetToolRegistry/SetWorkerBridge called in main.go
- **Committed in:** ce6cf9c (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for wiring to take effect at runtime. No scope creep.

## Issues Encountered

None beyond the globalAgentLoop deviation documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 04-03 (Chat UI) can now use webclaw.identity.* and webclaw.keystore.* JS APIs
- Plan 04-03 can listen for webclaw:tool-event CustomEvents to populate the tool activity panel
- All four browser tools (web_fetch, web_search, memory_store, memory_search) registered
- Real provider router wired — users need to call webclaw.keystore.setKey('anthropic', key) to activate LLM calls

---
*Phase: 04-tools-and-webchat-ui*
*Completed: 2026-03-01*
