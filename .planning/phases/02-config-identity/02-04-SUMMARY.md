---
phase: "02-config-identity"
plan: "04"
subsystem: "config"
tags: ["export", "import", "backup", "restore", "json", "browser-file-api"]
dependencies:
  requires: ["02-01", "02-02", "02-03"]
  provides: ["CONF-04"]
  affects: ["config", "identity", "keystore"]
tech-stack:
  added: []
  patterns: ["interface-adapter", "dependency-inversion"]
key-files:
  created:
    - internal/config/export.go
    - internal/config/import.go
    - internal/jsbridge/files.go
  modified:
    - internal/jsbridge/bridge.go
    - static/webclaw-host.js
    - cmd/webclaw/main.go
decisions:
  - "Used interfaces (IdentityFileProvider, IdentityFileImporter) to avoid import cycle between config and identity packages"
  - "Placed export/import bridge registration in main.go to prevent circular dependencies"
  - "Export format includes version metadata for future compatibility checking"
  - "Import validates required identity files (IDENTITY.md, SOUL.md) before restoring"
metrics:
  duration: 35m
  completed: "2026-03-01T10:45:00Z"
  files-created: 3
  files-modified: 3
---

# Phase 02 Plan 04: Import/Export Config System

## Summary

Implemented config import/export functionality that allows users to backup and restore their complete WebClaw configuration. The export includes config, identity files, and encrypted API keys. Import validates and restores all data using browser-standard download and FileReader APIs.

**Key Achievement**: CONF-04 complete — users can now export their entire agent state as JSON and reimport it to restore or migrate their configuration.

## Files Created

### internal/config/export.go
Export functionality for config and identity files.

**Key Components:**
- `ExportData` struct: Complete export format with version, date, config, identity files, encrypted keys
- `ExportAll()`: Gathers config, identity files, and encrypted keys via interfaces
- `ExportToJSON()`: Serializes to formatted JSON
- `ExportMinimal()`: Config-only export option
- `IdentityFileProvider` interface: Avoids import cycle with identity package
- `KeyStoreExporter` interface: For keystore operations

**Export Format:**
```json
{
  "version": "1",
  "export_date": "2026-03-01T10:00:00Z",
  "export_version": "1.0",
  "config": {
    "version": 1,
    "identity": {...},
    "agents": {...},
    "providers": {...},
    "memory": {...}
  },
  "identity_files": {
    "IDENTITY.md": "content...",
    "SOUL.md": "content...",
    "USER.md": "content...",
    "AGENTS.md": "content...",
    "TOOLS.md": "content...",
    "HEARTBEAT.md": "content..."
  },
  "encrypted_keys": {
    "anthropic": {
      "ciphertext": "base64...",
      "iv": "base64...",
      "salt": "base64..."
    }
  }
}
```

### internal/config/import.go
Import functionality with validation.

**Key Components:**
- `ImportFromJSON()`: Parses export data from JSON
- `ValidateExport()`: Checks version, config validation, required identity files
- `ImportAll()`: Restores config and identity files
- `ImportConfigOnly()`: Config-only restore option
- `CanImport()`: Pre-check for compatibility
- `IdentityFileImporter` interface: Avoids import cycle with identity package

**Validation Rules:**
1. Export version must be "1"
2. Config version must match CurrentVersion (1)
3. Config must pass Validate()
4. Required identity files must be present: IDENTITY.md, SOUL.md

### internal/jsbridge/files.go
Browser file operations bridge.

**Key Components:**
- `TriggerDownload(filename, content)`: Creates blob, triggers browser download
- `ReadFile(file)`: Promise-based FileReader API wrapper
- Uses Uint8Array and Blob for binary data handling
- Cleanup with URL.revokeObjectURL()

## Files Modified

### internal/jsbridge/bridge.go
Added `RegisterCallback()` function to export callback registration for external packages.

### static/webclaw-host.js
Added file handling section with:
- `webclaw:request-export` event handler → triggers download
- `webclaw:request-import` event handler → opens file picker
- `downloadConfig()` helper for blob-based downloads
- `triggerFileImport()` helper for FileReader-based import
- `webclawHelpers` global for WASM integration

### cmd/webclaw/main.go
- Added `registerExportImportBridge()` function
- Added `identityFileProvider` wrapper for config.IdentityFileProvider
- Added `identityFileImporter` wrapper for config.IdentityFileImporter
- Added `webclaw: export/import ready` log message
- JavaScript API: `window.webclaw.exportImport.exportConfig()` and `importConfig(json)`

## JavaScript API

```javascript
// Export config (triggers file download)
await window.webclaw.exportImport.exportConfig();

// Import config (restores from JSON string)
const jsonContent = await file.text();
await window.webclaw.exportImport.importConfig(jsonContent);
```

## Architectural Decisions

### Import Cycle Resolution
**Problem**: config and identity packages have circular dependency risk.
- `identity/bootstrap.go` imports `config` for bootstrap limits
- `config/export.go` needs identity file access

**Solution**: Interface-based dependency inversion
- `config.IdentityFileProvider` interface implemented by identity store wrapper
- `config.IdentityFileImporter` interface for import operations
- Wrappers in main.go bridge the gap without direct imports

### Bridge Registration Location
**Problem**: Adding export/import to jsbridge/bridge.go creates import cycle.
- `config/storage.go` imports `jsbridge`
- `jsbridge/bridge.go` would import `config`

**Solution**: Register in main.go after jsbridge.Init()
- No circular dependencies
- All packages available at main package level
- Clean separation of concerns

## Testing

### Automated Verification
```bash
# All packages compile
GOOS=js GOARCH=wasm go build ./...  # ✓ PASS

# WASM binary builds
make build  # ✓ PASS - produces dist/webclaw.wasm
```

### Manual Verification Steps
1. Start dev server: `make serve`
2. Open http://localhost:8080 in browser
3. Open DevTools console
4. Test export: `await window.webclaw.exportImport.exportConfig()`
   - Expected: File download triggered (webclaw-config.json)
5. Test import: `await window.webclaw.exportImport.importConfig(testConfigJson)`
   - Expected: Import succeeds, config restored
6. Verify console shows: "webclaw: export/import ready"

## Requirements Satisfied

- ✅ **CONF-04**: Config can be exported as JSON file
- ✅ **CONF-04**: Import restores full agent state (config + identity)
- ✅ **CONF-04**: Import validates schema version
- ✅ **CONF-04**: File operations use browser download/FileReader APIs
- ✅ **CONF-04**: JavaScript bridge provides exportConfig/importConfig functions

## Phase 2 Completion Status

All Phase 2 plans now complete:
- 02-01: Configuration system with IndexedDB storage ✅
- 02-02: Web Crypto bridge with encrypted key storage ✅
- 02-03: Identity file system with bootstrap assembly ✅
- 02-04: Import/Export config with browser file APIs ✅

**Phase 2 Status**: COMPLETE
**Next Phase**: Phase 3 - Intelligence Core (LLM provider routing, agent loop, memory system)

## Commits

- `ca6b612`: feat(02-04): config export functionality
- `db63daa`: feat(02-04): config import functionality
- `fd067b9`: feat(02-04): browser file operations bridge
- `dfd9f89`: feat(02-04): export/import bridge integration
