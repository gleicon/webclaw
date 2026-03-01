---
phase: 02-config-identity
plan: 03
phase_name: "Configuration and Identity"
plan_name: "Identity File System"
subsystem: identity
tags: [identity, indexeddb, bootstrap, system-prompt]
depends_on: [02-02]
requires: [IDNT-01, IDNT-02, IDNT-04]
provides: [identity-storage, bootstrap-assembly, default-identity]
affects: [LLM system prompts, first-run experience]
tech_stack:
  added:
    - "Go 1.25 WASM with syscall/js"
    - "IndexedDB identity object store"
    - "SHA256 checksums for change detection"
  patterns:
    - "Promise-based async IndexedDB operations"
    - "Truncation at newlines for clean cuts"
    - "Event dispatch for JS integration"
key_files:
  created:
    - internal/identity/files.go
    - internal/identity/defaults.go
    - internal/identity/bootstrap.go
  modified:
    - cmd/webclaw/main.go
decisions:
  - "Six identity files: IDENTITY, SOUL, USER, AGENTS, TOOLS, HEARTBEAT"
  - "20K per file limit, 150K total bootstrap limit (configurable)"
  - "Checksum-based change detection for identity files"
  - "webclaw:identity-ready event for JS integration"
metrics:
  duration_minutes: 8
  tasks: 4
  files_created: 3
  files_modified: 1
  lines_added: ~842
  wasm_size_bytes: 3740136
  wasm_compressed_bytes: 777902
---

# Phase 2 Plan 03: Identity File System - Summary

**Status:** ✅ Complete  
**Completed:** 2026-03-01  
**One-liner:** Virtual filesystem in IndexedDB storing six identity files with bootstrap assembly for system prompt injection

---

## What Was Built

### 1. Identity File Storage Layer (`internal/identity/files.go`)

A complete CRUD storage system for identity files in IndexedDB:

| Component | Purpose |
|-----------|---------|
| `IdentityFile` struct | File with content, size, timestamp, SHA256 checksum |
| `Store` struct | IndexedDB connection management |
| `Get(filename)` | Retrieve file by name |
| `Put(file)` | Store file with auto-updated metadata |
| `Delete(filename)` | Remove file |
| `List()` | Get all filenames |
| `Exists(filename)` | Check file existence |
| `LoadDefaults()` | Create default files on first run |
| `Close()` | Clean shutdown |

**Key features:**
- Uses `identity` object store in IndexedDB
- Automatic SHA256 checksums for integrity
- Automatic size calculation
- Timestamp tracking (ModifiedAt)
- Promise-based async operations via `jsbridge`

### 2. Default Identity Content (`internal/identity/defaults.go`)

Six default identity files with realistic OpenClaw-compatible content:

| File | Purpose | Size |
|------|---------|------|
| `IDENTITY.md` | Core identity, purpose, constraints | ~1.2KB |
| `SOUL.md` | Personality, tone, communication style | ~1.1KB |
| `USER.md` | User profile template (editable) | ~0.8KB |
| `AGENTS.md` | Agent configuration, model settings | ~0.9KB |
| `TOOLS.md` | Tool descriptions and usage | ~1.4KB |
| `HEARTBEAT.md` | Periodic tasks, event triggers | ~1.0KB |

**Helper functions:**
- `DefaultFiles()` - Returns map of all defaults
- `IsIdentityFile(filename)` - Validation helper

### 3. Bootstrap Assembly (`internal/identity/bootstrap.go`)

System prompt assembly with configurable limits:

| Function | Purpose |
|----------|---------|
| `AssembleSystemPrompt(store, config)` | Loads files, applies limits, builds prompt |
| `LoadIdentityFiles(store)` | Load all files without assembly |
| `CalculateBootstrapStats(files, config)` | Pre-flight size checking |
| `GetSystemPromptTemplate()` | Documentation template |

**Bootstrap limits (from config):**
- Per file: 20,000 chars (cuts at newline when possible)
- Total: 150,000 chars
- Tracks truncated files in `BootstrapResult`

**Template structure:**
```
You are {name}, an AI assistant.

[IDENTITY]
{IDENTITY.md content}

[SOUL]
{SOUL.md content}

[USER CONTEXT]
{USER.md content}

[AGENT CONFIGURATION]
{AGENTS.md content}

[AVAILABLE TOOLS]
{TOOLS.md content}

[HEARTBEAT INSTRUCTIONS]
{HEARTBEAT.md content}

Bootstrap limits: {used}/{max} characters loaded from {count} files
```

### 4. First-Run Initialization (`cmd/webclaw/main.go`)

Integration into main initialization flow:

```
main()
  ├── jsbridge.Init()
  ├── initializeConfig()      ✓ (existing)
  ├── initializeKeystore()    ✓ (existing)
  └── initializeIdentity()    ✓ NEW
        ├── NewStore()
        ├── List() files
        ├── if empty: LoadDefaults()
        └── Dispatch webclaw:identity-ready event
```

**Events dispatched:**
- `webclaw:identity-ready` with `filesCreated: 6` (first run)
- `webclaw:identity-ready` with `filesLoaded: N` (subsequent runs)

**Console logs:**
- `webclaw: created default identity files (first run)`
- `webclaw: identity files loaded`

---

## Verification Results

### Automated Verification

```bash
# 1. Full WASM build
$ GOOS=js GOARCH=wasm go build ./...
✓ Compiles without errors

# 2. Makefile build
$ make build
✓ dist/webclaw.wasm: 3,740,136 bytes
✓ dist/webclaw.wasm.br: 777,902 bytes (brotli compressed)
```

### Manual Browser Test

```javascript
// Check identity files were created
const idb = await window.indexedDB.open("webclaw");
const tx = idb.transaction("identity", "readonly");
const store = tx.objectStore("identity");
const files = await store.getAll();
console.log(files); // Should show 6 identity files

// Check events
document.addEventListener('webclaw:identity-ready', (e) => {
  console.log('Identity ready:', e.detail);
  // { filesCreated: 6, event: "first-run" }
  // or
  // { filesLoaded: 6, event: "loaded" }
});
```

---

## Success Criteria Check

| Criterion | Status | Evidence |
|-----------|--------|----------|
| IDNT-01: Six identity files stored in IndexedDB | ✅ | `LoadDefaults()` creates IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md |
| IDNT-02: Bootstrap assembly with limits | ✅ | `AssembleSystemPrompt()` with 20K per file, 150K total limits |
| IDNT-04: Default files created on first run | ✅ | `initializeIdentity()` checks `List()` and calls `LoadDefaults()` if empty |
| Events dispatched | ✅ | `webclaw:identity-ready` with `filesCreated` or `filesLoaded` |
| All code compiles under WASM | ✅ | `GOOS=js GOARCH=wasm go build ./...` exits 0 |

---

## Architecture

### Identity File Storage

```
IndexedDB
└── webclaw (database)
    ├── config (object store)     [existing]
    ├── keystore (object store)   [existing]
    └── identity (object store)   [NEW - this plan]
        ├── keyPath: filename
        └── records:
            ├── filename: "IDENTITY.md"
            ├── content: "# IDENTITY\n..."
            ├── size: 1234
            ├── modified_at: "2026-03-01T10:30:00Z"
            └── checksum: "sha256:abc123..."
```

### System Prompt Assembly Flow

```
LLM Request
    │
    ▼
AssembleSystemPrompt(store, config)
    │
    ├── Get IDENTITY.md ───┐
    ├── Get SOUL.md ───────┤
    ├── Get USER.md ───────┤ Truncate to 20K at newline
    ├── Get AGENTS.md ─────┤
    ├── Get TOOLS.md ──────┤
    └── Get HEARTBEAT.md ──┘
    │
    ├── Check total <= 150K
    │   └── If exceeded: stop loading, mark remaining as truncated
    │
    └── Build system prompt string
        │
        ▼
    [System Prompt]
```

---

## Deviations from Plan

**None** - plan executed exactly as written.

All code matches the specification in `02-03-PLAN.md`:
- Files created with exact functions specified
- Bootstrap limits match config defaults
- Events dispatched as specified
- Console logging added for debugging

---

## Notes for Plan 02-04 (Import/Export)

The identity file system is ready for import/export features:

**What 02-04 can build on:**
- `LoadIdentityFiles()` returns map of all files (perfect for export)
- `Store.Put()` handles updates (perfect for import)
- `IsIdentityFile()` validates filenames
- Checksums can detect conflicts during import

**Suggested import/export formats:**
- JSON bundle: `{ "IDENTITY.md": "...", "SOUL.md": "...", ... }`
- Individual markdown files with YAML frontmatter
- ZIP archive of .md files

**Conflict resolution:**
- Compare checksums to detect changes
- Option: overwrite, skip, or merge
- UI needed for user decision on conflicts

---

## Commits

| Hash | Message | Files |
|------|---------|-------|
| e49fb97 | feat(02-03): identity file storage and default content | `internal/identity/files.go`, `internal/identity/defaults.go` |
| 2a83f76 | feat(02-03): bootstrap assembly for system prompts | `internal/identity/bootstrap.go` |
| d23a056 | feat(02-03): first-run identity initialization in main | `cmd/webclaw/main.go` |

---

## Self-Check: PASSED

- [x] `internal/identity/files.go` exists with Store struct
- [x] `internal/identity/defaults.go` exists with all 6 default functions
- [x] `internal/identity/bootstrap.go` exists with AssembleSystemPrompt
- [x] `cmd/webclaw/main.go` updated with initializeIdentity()
- [x] All files compile under `GOOS=js GOARCH=wasm`
- [x] `make build` produces valid WASM
- [x] All commits created and tracked

---

**Next Plan:** 02-04 (Import/Export for identity files) or Phase 03 (Memory System)
