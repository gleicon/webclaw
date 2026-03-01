---
phase: 05-live-ai-connection
plan: 02
subsystem: provider
tags: [router, provider, hot-swap, events, wasm]

# Dependency graph
requires:
  - phase: 05-live-ai-connection
    provides: Async keystore initialization pattern (05-01)
provides:
  - webclaw:providers-ready event dispatch
  - HasProvider() availability check
  - Mock fallback for missing providers
  - Clear error messages for UI
  - Hot-swap capable provider registration
affects:
  - 05-03-live-ai-connection
  - UI event handling
  - Settings tab integration

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Event-driven initialization: CustomEvent dispatch for provider readiness"
    - "Defensive programming: HasProvider() check before API calls"
    - "Graceful degradation: Mock provider fallback when no providers configured"
    - "Hot-swap pattern: RegisterProvider/UnregisterProvider for runtime key updates"

key-files:
  created: []
  modified:
    - cmd/webclaw/main.go
    - internal/agent/loop.go

key-decisions:
  - "Dispatch event at end of async loader rather than per-provider (reduces event noise)"
  - "Check HasProvider() in providerAdapter.Stream for early failure (fail fast)"
  - "Use separate noProvidersMock type for clearer user messaging vs test mockProvider"
  - "Router.AvailableProviders() provides provider list for event dispatch"

patterns-established:
  - "Provider lifecycle: Router starts empty, async loader registers providers, event notifies UI"
  - "Graceful fallback: Router exists but empty → noProvidersMock → helpful Settings message"
  - "Defensive API calls: Always check availability before expensive network operations"

requirements-completed:
  - PROV-01
  - PROV-02

# Metrics
duration: 1min
completed: 2026-03-01
---

# Phase 05 Plan 02: Router Configuration with Live Keys Summary

**Provider router configured with async hot-swap registration, availability checks, and UI event notifications for live AI streaming**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-01T23:23:27Z
- **Completed:** 2026-03-01T23:24:40Z
- **Tasks:** 10
- **Files modified:** 2

## Accomplishments

- Router initialized with empty keys (non-blocking startup preserved)
- `webclaw:providers-ready` CustomEvent dispatched with provider list and count
- `HasProvider()` check added to providerAdapter for fail-fast API call validation
- `noProvidersMock` fallback returns helpful message directing users to Settings
- Clear error message when provider unavailable: "Please add your API key in Settings"
- Hot-swap support foundation via `RegisterProvider()` for runtime key updates

## Task Commits

Each task was committed atomically:

1. **Task 7: Dispatch webclaw:providers-ready event** - `cedd22f` (feat)
2. **Task 8: Add HasProvider check in providerAdapter** - `6922ba3` (feat)
3. **Task 9-10: Mock fallback and graceful error handling** - `91b2b5a` (feat)

**Plan metadata:** (pending final commit)

_Note: Tasks 1-6 were already implemented in previous work (05-01 foundation) and verified in place_

## Files Created/Modified

- `cmd/webclaw/main.go` - Added `webclaw:providers-ready` event dispatch at end of `loadProviderKeysAsync()`
- `internal/agent/loop.go` - Added `HasProvider()` check in providerAdapter, `noProvidersMock` fallback, updated `getProvider()` to check `AvailableProviders()`

## Decisions Made

1. **Event dispatch timing**: Dispatched once at end of async loader with full provider list, rather than per-provider events. Reduces event noise and gives UI complete picture at once.

2. **Fail-fast vs fail-slow**: Added `HasProvider()` check at start of `providerAdapter.Stream()` rather than letting the router fail later. Provides clearer error message and avoids unnecessary setup work.

3. **Separate mock types**: Created `noProvidersMock` distinct from `mockProvider`. The former is for production (guides users to Settings), the latter for development/testing.

## Deviations from Plan

None - plan executed exactly as written. Tasks 1-6 were already implemented in 05-01 foundation and verified in place. Tasks 7-10 implemented as specified.

## Issues Encountered

None. All changes straightforward extensions of existing patterns.

## User Setup Required

None - no external service configuration required. The implementation enables users to configure API keys in the Settings tab, which triggers the existing `webclaw.keystore.setKey()` bridge.

## Next Phase Readiness

- **Ready for 05-03**: Live tool execution during streaming
- Provider router fully configured with hot-swap capability
- UI can listen for `webclaw:providers-ready` to show/hide provider-specific UI
- Clear user messaging when providers unavailable

---
*Phase: 05-live-ai-connection*
*Completed: 2026-03-01*
