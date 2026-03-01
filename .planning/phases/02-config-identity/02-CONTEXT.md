---
phase: 02-config-identity
created: 2026-02-28
scope: Research knowledge for Phase 2 planning
---

# Phase 2: Configuration and Identity — Context

## What We're Building

Phase 2 adds three core capabilities to the WebClaw WASM runtime:

1. **Configuration System** (CONF-01 to CONF-04)
   - JSON config with snake_case/camelCase support
   - Persisted in IndexedDB under `webclaw:config`
   - Covers identity, agents, providers, memory settings
   - Import/export as JSON file

2. **Encrypted Key Storage** (SEC-01 to SEC-03)
   - AES-256-GCM encryption via Web Crypto API
   - PBKDF2 key derivation from user passphrase
   - Keys never exist as plaintext in JavaScript
   - Decryption happens inside WASM linear memory

3. **Identity File System** (IDNT-01 to IDNT-04)
   - Virtual filesystem in IndexedDB
   - Six default identity files (IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md)
   - Bootstrap injection into system prompts
   - User-editable via browser UI (Phase 4)

## Technical Foundation from Phase 1

We have:
- `internal/jsbridge` package with fetch() and IndexedDB bridges
- `cmd/webclaw/main.go` WASM entry point
- Build pipeline: `make build` produces dist/webclaw.wasm
- Dev server: `go run ./cmd/devserver/` serves on :8080
- Test infrastructure: `node test/test-wasm.js` validates WASM loading

## New Components Needed

### Go Packages

```
internal/
  config/          # Config loading, validation, persistence
    - config.go    # Config struct, defaults, JSON serialization
    - storage.go   # IndexedDB persistence via jsbridge
    
  crypto/          # Web Crypto API wrapper (jsbridge extension)
    - aes.go       # AES-256-GCM encrypt/decrypt
    - pbkdf2.go    # Key derivation from passphrase
    - bridge.go    # JS bridge to crypto.subtle
    
  identity/        # Identity file virtual filesystem
    - files.go     # IdentityFile struct, CRUD operations
    - bootstrap.go # File injection into system prompts
    - defaults.go  # Default content for the 6 identity files
    
  keystore/        # Encrypted API key management
    - store.go     # Key storage/retrieval interface
    - encryption.go # High-level encrypt/decrypt API
```

### JavaScript Additions

- Extend `jsbridge` to expose Web Crypto methods
- Add IndexedDB object store operations beyond simple open()
- File import/export via FileReader and download triggers

### UI Elements (Phase 4 prep)

- Settings panel structure (stub for now)
- Passphrase modal
- Identity file editor (stub)

## Key Interfaces

### Config Interface

```go
type Config struct {
    Version  int              `json:"version"`
    Identity IdentityConfig   `json:"identity"`
    Agents   AgentsConfig     `json:"agents"`
    Providers map[string]ProviderConfig `json:"providers"`
    Memory   MemoryConfig     `json:"memory"`
}

type ProviderConfig struct {
    APIKeyEncrypted string `json:"apiKeyEncrypted"`
    BaseURL         string `json:"baseUrl"`
}
```

### Crypto Interface

```go
type Crypto interface {
    // Derive encryption key from passphrase
    DeriveKey(passphrase string, salt []byte) (KeyHandle, error)
    
    // Encrypt plaintext, returns ciphertext+tag
    Encrypt(plaintext []byte, key KeyHandle) (ciphertext []byte, iv []byte, err error)
    
    // Decrypt ciphertext, returns plaintext
    Decrypt(ciphertext []byte, iv []byte, key KeyHandle) (plaintext []byte, err error)
}

// KeyHandle is an opaque reference to a JS CryptoKey object
type KeyHandle = js.Value
```

### Identity File Interface

```go
type IdentityStore interface {
    Get(filename string) (IdentityFile, error)
    Put(file IdentityFile) error
    Delete(filename string) error
    List() ([]string, error)
    Exists(filename string) (bool, error)
}

type IdentityFile struct {
    Filename   string    `json:"filename"`
    Content    string    `json:"content"`
    ModifiedAt time.Time `json:"modified_at"`
}
```

## Constraints & Decisions

### From Requirements

- PBKDF2: 100,000 iterations (OWASP 2023)
- Salt: 16 bytes (128 bits)
- IV: 12 bytes (96 bits, GCM standard)
- Tag: 128 bits (16 bytes)
- Config format: JSON (human-readable)
- Identity format: Markdown (OpenClaw compatible)
- Bootstrap limits: 20K chars per file, 150K total

### Technical Constraints

- All Web Crypto calls must use goroutine-spawn pattern (prevents deadlock)
- IndexedDB operations are async, require promise handling
- Plaintext keys only in WASM memory, zeroed after use
- Go's GC may retain memory, but WASM sandbox limits exposure

## Integration Points

### With Phase 1

- Extend `internal/jsbridge` with crypto methods
- Use existing IndexedDB bridge for config storage
- Build system unchanged (`make build` still works)

### For Phase 3 (Intelligence Core)

- Config provides provider routing info (PROV-01)
- Identity files feed into system prompts (AGNT-01)
- Keystore provides decrypted API keys for provider calls
- Memory config prepares for vector storage (MEM-01)

### For Phase 4 (UI)

- Config storage enables settings persistence
- Identity store enables file editing
- Keystore enables secure key entry
- Import/export enables config portability

## Testing Strategy

1. **Unit tests** (Go native, no WASM)
   - Config serialization/deserialization
   - Key derivation correctness (known test vectors)
   - Encryption round-trip

2. **Integration tests** (Node.js + headless Chrome)
   - Full config save/load cycle
   - Encryption/decryption via Web Crypto
   - IndexedDB persistence
   - Import/export round-trip

3. **Security tests**
   - Verify keys don't leak to JS console
   - Verify memory is zeroed after use
   - Attempt key extraction from WASM memory

## Success Criteria

From ROADMAP.md:
1. ✅ User can load a JSON config and it is persisted to IndexedDB
2. ✅ User can export and reimport config as a JSON file
3. ✅ On first run, agent prompts for a passphrase; subsequent runs require it
4. ✅ API keys are never readable as plaintext in browser DevTools
5. ✅ Default identity files are loaded from IndexedDB and user can edit them

## Research References

- Web Crypto API: https://www.w3.org/TR/WebCryptoAPI/
- PBKDF2: NIST SP 800-132
- AES-GCM: NIST SP 800-38D
- IndexedDB: MDN documentation
- OWASP Password Storage: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html

---
*Phase 2 Context Document*
*Updated: 2026-02-28*
