---
phase: 08-static-bundle
plan: 06
status: complete
completed: "2026-03-07"
---

# Plan 08-06 Summary: Error Telemetry System

## What Was Built

Lightweight error and event telemetry system in `internal/telemetry/` with localStorage persistence, integrated into the agent loop and provider router.

## Key Files Created

- `internal/telemetry/errors.go` — `ErrorRecord`, `EventRecord` types; `ErrorLevel` constants; `SanitizeContext()` redacts sensitive fields
- `internal/telemetry/telemetry.go` — `Collector` with async `StoreError`/`StoreEvent` (non-blocking goroutines), `GetReport()` for debug export, `Storage` interface
- `internal/telemetry/storage.go` — `LocalStorageBackend`: stores JSON arrays in localStorage, auto-trims to 100 errors / 50 events, handles corrupted data
- `internal/telemetry/global.go` — singleton `GlobalTelemetry`, auto-initialized with localStorage backend
- `internal/agent/loop.go` — tool execution errors + stream errors + iteration limit warnings recorded
- `internal/provider/failover.go` — final failover failures recorded

## Design Decisions

- localStorage over IndexedDB to avoid version bump complexity
- All sensitive fields (`api_key`, `token`, `password`, `secret`, `key`) redacted by `SanitizeContext` before storage
- Non-blocking async pattern: `StoreError`/`StoreEvent` spawn goroutines so telemetry never blocks the agent loop
- Auto-trim retention: 100 errors max, 50 events max (circular buffer via slice truncation)

## Commits

- `e30e936` feat(08-06): create telemetry package with core types and Collector
- `80827be` feat(08-06): implement localStorage-based telemetry storage backend
- `9645e68` feat(08-06): integrate telemetry into agent loop and provider router

## Self-Check: PASSED

- Errors collected with context (timestamp, type, sanitized context)
- Graceful degradation: telemetry failures don't propagate to agent loop
- No API keys in telemetry (SanitizeContext covers all known key field names)
- GetReport() available for debug export
