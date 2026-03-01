# Phase 2: Configuration and Identity — Research

## Overview

Phase 2 adds persistent configuration, encrypted API key storage, and identity file management. This requires bridging the Web Crypto API to Go/WASM, implementing IndexedDB storage patterns, and creating a virtual filesystem for identity files.

## Technical Questions

### 1. Web Crypto API in Go/WASM

**Question:** How do we call Web Crypto API methods from Go WASM for AES-256-GCM encryption?

**Pattern:** Use `syscall/js` to call `crypto.subtle` methods through the jsbridge pattern established in Phase 1.

**Key methods needed:**
- `crypto.subtle.generateKey()` - Generate AES-GCM keys
- `crypto.subtle.encrypt()` - Encrypt data with AES-GCM
- `crypto.subtle.decrypt()` - Decrypt data with AES-GCM  
- `crypto.subtle.deriveKey()` - PBKDF2 key derivation from passphrase
- `crypto.subtle.importKey()` - Import raw key data
- `crypto.subtle.exportKey()` - Export key to raw bytes

**Implementation approach:**
```go
// jsbridge/crypto.go
func EncryptAESGCM(plaintext []byte, key js.Value) ([]byte, error) {
    // Call crypto.subtle.encrypt() via syscall/js
    // Returns Promise, resolve with ciphertext + auth tag
}

func DeriveKeyPBKDF2(passphrase string, salt []byte) (js.Value, error) {
    // Use crypto.subtle.importKey() to import passphrase
    // Use crypto.subtle.deriveKey() with PBKDF2 params
    // Returns derived AES-GCM key
}
```

**Security considerations:**
- Keys should only exist as `js.Value` references (opaque handles)
- Never export keys to Go []byte except during the brief moment of key derivation
- Use 256-bit keys, random IVs, and include authentication tags

### 2. IndexedDB Configuration Storage

**Question:** How do we structure IndexedDB for config storage?

**Database structure:**
```
Database: webclaw
Version: 1

Object Stores:
  - config: { keyPath: "key" }
    * Keys: "webclaw:config", "webclaw:keys", "webclaw:identity"
    
  - identity_files: { keyPath: "filename" }
    * Files: IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md
    
  - memories: { keyPath: "id", indexes: ["timestamp", "embedding"] }
    * For Phase 3 memory system
```

**Config schema (JSON):**
```json
{
  "version": 1,
  "identity": {
    "name": "WebClaw",
    "bootstrapMaxChars": 20000,
    "bootstrapTotalMaxChars": 150000
  },
  "agents": {
    "defaultModel": "anthropic/claude-sonnet-4-5",
    "maxToolIterations": 10,
    "temperature": 0.7,
    "bootstrapLimits": {
      "maxCharsPerFile": 20000,
      "maxTotalChars": 150000
    }
  },
  "providers": {
    "anthropic": {
      "apiKeyEncrypted": "base64_encrypted_key",
      "baseUrl": "https://api.anthropic.com"
    },
    "openai": {
      "apiKeyEncrypted": "base64_encrypted_key", 
      "baseUrl": "https://api.openai.com"
    },
    "openrouter": {
      "apiKeyEncrypted": "base64_encrypted_key",
      "baseUrl": "https://openrouter.ai/api"
    }
  },
  "memory": {
    "enabled": true,
    "maxMemories": 10000,
    "embeddingModel": "openai/text-embedding-3-small"
  }
}
```

**Storage pattern:**
- Config stored as JSON blob under key "webclaw:config"
- API keys encrypted before storage (never plaintext)
- Encryption key derived from user passphrase via PBKDF2
- Salt stored alongside encrypted data for key derivation

### 3. Encryption Architecture

**Question:** How do we ensure API keys never exist as plaintext in JavaScript?

**Architecture:**
1. **Key Derivation:**
   - User enters passphrase in browser UI
   - Passphrase sent to WASM via JS bridge
   - WASM calls Web Crypto to derive 256-bit AES key via PBKDF2
   - Salt is randomly generated and stored with encrypted data

2. **Encryption Flow:**
   ```
   User enters API key in browser UI
           ↓
   JS sends { key: "api-key-value", passphrase: "user-pass" } to WASM
           ↓
   WASM derives encryption key via PBKDF2 (passphrase + salt)
           ↓
   WASM encrypts API key via AES-GCM
           ↓
   WASM returns { encryptedKey: "base64", salt: "base64" } to JS
           ↓
   JS stores encryptedKey + salt in IndexedDB
           ↓
   JS discards plaintext key from memory
   ```

3. **Decryption Flow:**
   ```
   Agent needs API key for LLM call
           ↓
   WASM requests encrypted key from JS
           ↓
   JS fetches { encryptedKey, salt } from IndexedDB
           ↓
   JS prompts user for passphrase (if not cached)
           ↓
   JS sends { encryptedKey, salt, passphrase } to WASM
           ↓
   WASM derives key via PBKDF2
           ↓
   WASM decrypts API key
           ↓
   WASM uses key for fetch() call
           ↓
   WASM zeros memory containing plaintext key
   ```

**Zero-knowledge design:**
- Plaintext API keys only exist in WASM linear memory briefly
- Go's garbage collector may retain memory, but it's WASM sandboxed
- Memory scrubbing: explicitly overwrite key bytes with zeros after use

### 4. Identity File System

**Question:** How do we implement a virtual filesystem for identity files in IndexedDB?

**Design:**
- Store identity files as blobs in IndexedDB object store
- Provide CRUD operations: Create, Read, Update, Delete
- Lazy loading: only load files when needed for prompt assembly
- Default files created on first run if not present

**Identity file contents:**
- `IDENTITY.md` - Core identity, name, purpose, constraints
- `SOUL.md` - Personality, tone, behavioral patterns  
- `USER.md` - User profile, preferences, context
- `AGENTS.md` - Agent configurations, models, settings
- `TOOLS.md` - Tool descriptions and schemas
- `HEARTBEAT.md` - Periodic execution instructions

**File structure:**
```go
type IdentityFile struct {
    Filename    string    `json:"filename"`
    Content     string    `json:"content"`
    Size        int       `json:"size"`
    ModifiedAt  time.Time `json:"modified_at"`
    Checksum    string    `json:"checksum"` // For change detection
}
```

**Bootstrap injection:**
- Before each LLM call, assemble system prompt
- Load identity files from IndexedDB
- Truncate each file to `bootstrapMaxChars` (default 20K)
- Ensure total across all files <= `bootstrapTotalMaxChars` (default 150K)
- Inject into system prompt template

### 5. Import/Export Pattern

**Question:** How do we implement config import/export?

**Export:**
- Fetch config + identity files from IndexedDB
- Create JSON structure with all data
- Trigger browser download via `URL.createObjectURL()` + `<a download>`
- Include encrypted keys (still encrypted, passphrase needed to use)

**Import:**
- User selects JSON file via `<input type="file">`
- Read file via FileReader API
- Validate schema version
- Write config + identity files to IndexedDB
- If encrypted keys present, prompt for passphrase on first use

### 6. First-Run Flow

**Question:** How do we detect and handle first-run initialization?

**Detection:**
- Check IndexedDB for "webclaw:config" key
- If missing, trigger first-run setup

**First-run steps:**
1. Generate default identity files
2. Prompt user to set passphrase
3. Create empty config with defaults
4. Show settings UI for API key entry
5. Encrypt and store any provided API keys

## Research Sources

- Web Crypto API spec: https://www.w3.org/TR/WebCryptoAPI/
- IndexedDB best practices: https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Using_IndexedDB
- Go syscall/js: https://pkg.go.dev/syscall/js
- PBKDF2 recommendations: NIST SP 800-132 (minimum 10,000 iterations, SHA-256)
- AES-GCM: NIST SP 800-38D (96-bit IV, 128-bit auth tag)

## Decisions

1. **Database name:** `webclaw` (single database for all data)
2. **PBKDF2 iterations:** 100,000 (OWASP 2023 recommendation)
3. **Salt length:** 16 bytes (128 bits)
4. **IV length:** 12 bytes (96 bits, GCM standard)
5. **Tag length:** 128 bits (16 bytes)
6. **Config serialization:** JSON (human-readable, debuggable)
7. **Identity files:** Markdown format (maintains OpenClaw compatibility)

## Open Questions

- [ ] Should we support multiple passphrases for different security levels?
- [ ] How do we handle passphrase change (re-encrypt all keys)?
- [ ] Should identity files have version history?
- [ ] What migration path for config format changes?

---
*Research for: Phase 2 - Configuration and Identity*
*Date: 2026-02-28*
