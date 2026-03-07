---
phase: 09-social-integrations
plan: "06"
subsystem: oauth-js-api
tags: [oauth, wasm, jsbridge, gap-closure, playwright]
dependency_graph:
  requires: [09-01, commit 3653b76]
  provides: [OAuth JS API on window.webclaw.oauth, passing smoke tests]
  affects: [dist/webclaw.wasm, internal/jsbridge/oauth_bridge.go, cmd/webclaw/main.go]
tech_stack:
  added: []
  patterns: [get-or-create JS object pattern, main thread OAuth goroutine, WASM JS bridge]
key_files:
  created: []
  modified:
    - cmd/webclaw/main.go
    - internal/jsbridge/oauth_bridge.go
    - dist/webclaw.wasm
    - dist/webclaw.wasm.br
    - .planning/phases/09-social-integrations/09-UAT.md
decisions:
  - "OAuth goroutine runs on main thread WASM (not worker) so js.Global() is window scope"
  - "RegisterOAuthBridge() uses webclaw.Get('oauth') to preserve JS-side openPopup before adding Go functions"
  - "300ms goroutine sleep sufficient for keystore readiness before OAuth init"
metrics:
  duration: "~15 min"
  completed: 2026-03-07
  tasks_completed: 3
  tasks_total: 3
  files_changed: 5
requirements: [INTEG-01, INTEG-02, INTEG-03, INTEG-04, AUTH-01, AUTH-02]
---

# Phase 09 Plan 06: OAuth JS API Gap Closure Summary

**One-liner:** Main thread OAuth goroutine with get-or-create JS object pattern exposes isConnected, getConnectionStatus, initiateConnection, disconnect on window.webclaw.oauth after WASM init.

## What Was Built

Closed UAT Gap 2: OAuth JS API (isConnected, getConnectionStatus, initiateConnection, disconnect) was not being exposed on window.webclaw.oauth after WASM init.

Gap 1 (WASM panic in idb_memory.go) had already been fixed in commit 3653b76. This plan focused on Gap 2.

### Root Cause of Gap 2

Two interacting failures:
1. When the main thread WASM panicked (due to idb_memory.go bug), the OAuth goroutine never ran
2. Even if the panic was fixed, the OAuth goroutine in the prior code was in the worker WASM context, where `js.Global()` returns worker scope (not window), so `RegisterJSExports()` saw `webclaw` as undefined and silently early-returned

### Fix Applied

**cmd/webclaw/main.go:** Added OAuth goroutine to the main thread init sequence:
- `oauth.Init()` — initializes provider configs
- `jsbridge.InitOAuthBridge()` — creates JS bridge and calls RegisterOAuthBridge()
- `oauth.NewTokenStore()` with nil/error check
- `jsbridge.GetOAuthBridge()` with nil check
- `oauth.NewOAuthManager()` — creates manager
- `oauthMgr.RegisterJSExports()` — adds isConnected/getConnectionStatus/initiateConnection/disconnect to window.webclaw.oauth
- Registers all 4 integration tool sets (Twitter, GitHub, Google, Notion)

**internal/jsbridge/oauth_bridge.go:** Fixed RegisterOAuthBridge() to use get-or-create pattern:
- Uses `webclaw.Get("oauth")` to fetch the existing oauth object set by webclaw-host.js
- Only creates a new object with `js.Global().Get("Object").New()` if existing one is undefined/null
- Removed the final `webclaw.Set("oauth", oauth)` overwrite that clobbered the JS-side openPopup function
- Then adds handleCallback and exchangeCode on the preserved existing object

## Verification Results

All 6 phase09-smoke.spec.js Playwright tests pass:

| Test | Name | Result |
|------|------|--------|
| 1 | App loads without crash errors | PASS |
| 2 | Connected Services section exists in DOM | PASS |
| 3 | All 4 provider cards rendered after host-ready | PASS |
| 4 | webclaw.oauth API exposed with isConnected, getConnectionStatus, initiateConnection, disconnect | PASS |
| 5 | isConnected('twitter') returns false (not undefined) | PASS |
| 6 | getConnectionStatus() returns array with all 4 providers | PASS |

Test 4 output confirmed: `oauthKeys: ["openPopup","handleCallback","exchangeCode","initiateConnection","disconnect","getConnectionStatus","isConnected"]` — all 7 functions present (JS-side: openPopup, handleCallback, exchangeCode; Go-side: initiateConnection, disconnect, getConnectionStatus, isConnected).

## UAT Status After This Plan

- Test 1 (Cold Start Smoke): issue → **pass** (Fixed in commit 3653b76)
- Test 20 (OAuth JS API): issue → **pass** (Fixed in this plan)
- Summary: 2 passed / 2 issues → **4 passed / 0 issues**

## Commits

| Hash | Message |
|------|---------|
| 04b7534 | fix(09): close UAT gaps — OAuth JS API wired on main thread, smoke tests passing |

## Deviations from Plan

None — plan executed exactly as written. The unstaged changes were already correct as verified in Task 1.

## Self-Check

- [x] cmd/webclaw/main.go modified: OAuth goroutine added to main thread
- [x] internal/jsbridge/oauth_bridge.go modified: get-or-create pattern applied
- [x] dist/webclaw.wasm built successfully
- [x] dist/webclaw.wasm.br compressed artifact present
- [x] 09-UAT.md updated: both gaps resolved, 4 passed / 0 issues
- [x] All 6 Playwright smoke tests pass
- [x] Single commit 04b7534 contains all 5 changed files
