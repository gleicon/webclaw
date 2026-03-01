---
phase: 02-config-identity
plan: 01
subsystem: config
completed_at: 2026-03-01T01:35:16Z
---

# Phase 02 Plan 01: Configuration System Foundation

## Summary

Built the configuration system foundation for WebClaw — config struct with JSON serialization, IndexedDB persistence layer, and first-run detection. This enables the agent to store and retrieve configuration data in the browser.

**One-liner:** Config struct with snake_case JSON, IndexedDB persistence, auto-created default config on first run with events dispatched for JS integration.

---

## Files Created

| File | Purpose | Lines |
|------|---------|-------|
| `internal/config/config.go` | Config struct, defaults, validation, JSON serialization | 158 |
| `internal/config/storage.go` | IndexedDB persistence layer | 189 |

## Files Modified

| File | Changes |
|------|---------|
| `internal/jsbridge/indexeddb.go` | Added IDBOpen, IDBGet, IDBPut, IDBDelete functions |
| `cmd/webclaw/main.go` | Added initializeConfig() with first-run detection |

---

## Config Schema

### Main Config Structure
```go
type Config struct {
    Version   int                     `json:"version"`
    Identity  IdentityConfig          `json:"identity"`
    Agents    AgentsConfig            `json:"agents"`
    Providers map[string]ProviderConfig `json:"providers"`
    Memory    MemoryConfig            `json:"memory"`
    CreatedAt time.Time               `json:"created_at"`
    UpdatedAt time.Time               `json:"updated_at"`
}
```

### Default Values
| Setting | Default |
|---------|---------|
| Identity.Name | "WebClaw" |
| Identity.BootstrapMaxChars | 20000 |
| Identity.BootstrapTotalMaxChars | 150000 |
| Agents.DefaultModel | "anthropic/claude-sonnet-4-5" |
| Agents.MaxToolIterations | 10 |
| Agents.Temperature | 0.7 |
| Memory.Enabled | true |
| Memory.MaxMemories | 10000 |
| Memory.EmbeddingModel | "openai/text-embedding-3-small" |

### Default Providers
- **anthropic**: https://api.anthropic.com
- **openai**: https://api.openai.com
- **openrouter**: https://openrouter.ai/api

---

## IndexedDB Structure

- **Database**: `webclaw`
- **Version**: 1
- **Object Store**: `config`
- **Key Path**: `key`
- **Config Key**: `webclaw:config`

Storage operations use Promise-based async patterns with goroutine-spawn for non-blocking IndexedDB access.

---

## First-Run Flow

1. WASM loads, jsbridge.Init() registers bridges
2. `initializeConfig()` called from main()
3. Storage opens IndexedDB connection
4. Check `ConfigExists()` — queries IndexedDB for config
5. If no config:
   - Create `DefaultConfig()`
   - Save to IndexedDB with timestamps
   - Dispatch `webclaw:first-run` event
   - Log "webclaw: created default config (first run)"
6. If config exists:
   - Load config from IndexedDB
   - Validate config
   - Dispatch `webclaw:config-ready` event
   - Log "webclaw: config loaded"

---

## Events Dispatched

| Event | When | Detail |
|-------|------|--------|
| `webclaw:first-run` | First run, default config created | `{version, identity}` |
| `webclaw:config-ready` | Config loaded successfully | `{version, identity}` |

Note: Full config object not passed in event detail (syscall/js can't convert Go structs). Use storage API to retrieve full config.

---

## Validation Rules

- Version must match `CurrentVersion` (1)
- Identity.Name cannot be empty
- Agents.Temperature must be 0-2
- Agents.MaxToolIterations must be 1-50
- All provider BaseURLs must be non-empty

---

## Test Results

| Test | Result |
|------|--------|
| `GOOS=js GOARCH=wasm go build ./...` | ✅ PASS |
| `make build` produces dist/webclaw.wasm | ✅ PASS |
| `node test/test-wasm.js` | ✅ PASS |
| Config saved to IndexedDB (first-run) | ✅ PASS |
| "webclaw: created default config (first run)" log | ✅ PASS |

**Build artifacts:**
- `dist/webclaw.wasm`: 3373.6 KB
- `dist/webclaw.wasm.br`: 699.7 KB (79.3% compression)

---

## Deviations from Plan

### Auto-fixed Issues

**[Rule 1 - Bug] Fixed IndexedDB objectStoreNames contains() call**
- **Found during:** Task 2 testing
- **Issue:** Line 45 used `db.Call("objectStoreNames", "contains").Invoke(ConfigStore)` but `objectStoreNames` is a property returning a DOMStringList, not a function
- **Fix:** Changed to `db.Get("objectStoreNames").Call("contains", ConfigStore)`
- **Files modified:** `internal/config/storage.go`
- **Commit:** 0448adb

**[Rule 1 - Bug] Fixed CustomEvent invalid value error**
- **Found during:** Task 3 testing
- **Issue:** Passing Go struct (`cfg`) directly to JS via `map[string]interface{}{"config": cfg}` causes "ValueOf: invalid value" panic
- **Fix:** Only pass primitive values in event detail. Changed to pass `{version, identity}` only.
- **Files modified:** `cmd/webclaw/main.go`
- **Commit:** 0448adb

---

## Commits

| Hash | Message |
|------|---------|
| e80140d | feat(02-01): create config struct with defaults and validation |
| d109f45 | feat(02-01): add IndexedDB storage layer for config persistence |
| 42f723a | feat(02-01): add first-run config initialization |
| 0448adb | fix(02-01): fix IndexedDB storage and event dispatch bugs |

---

## Requirements Satisfied

| Requirement | Status | Evidence |
|-------------|--------|----------|
| CONF-01 | ✅ | JSON config with snake_case tags implemented |
| CONF-02 | ✅ | Config persisted to IndexedDB under "webclaw:config" |
| CONF-03 | ✅ | Config covers identity, agents, providers, memory |
| First-run detection | ✅ | Creates default config when none exists |
| Event dispatching | ✅ | webclaw:first-run and webclaw:config-ready events |
| WASM compilation | ✅ | All code compiles under GOOS=js GOARCH=wasm |

---

## Notes for Plan 02-02 (Crypto Bridge)

- Config contains `APIKeyEncrypted` field ready for encrypted storage
- Storage layer supports Get/Put/Delete for future key management
- Identity config has bootstrap limits that will need adjustment per agent

---

## Self-Check: PASSED

✅ All created files exist:
- `/Users/gleicon/code/go/src/github.com/gleicon/webclaw/internal/config/config.go`
- `/Users/gleicon/code/go/src/github.com/gleicon/webclaw/internal/config/storage.go`

✅ All modified files updated:
- `/Users/gleicon/code/go/src/github.com/gleicon/webclaw/internal/jsbridge/indexeddb.go`
- `/Users/gleicon/code/go/src/github.com/gleicon/webclaw/cmd/webclaw/main.go`

✅ All commits exist:
- e80140d, d109f45, 42f723a, 0448adb

✅ Build passes: `GOOS=js GOARCH=wasm go build ./...`
✅ Tests pass: `node test/test-wasm.js`
