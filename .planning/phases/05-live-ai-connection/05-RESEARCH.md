# Phase 5 Research: Live AI Provider Connection

**Researched:** 2026-03-01  
**Purpose:** Planning foundation for connecting stored API keys to live provider router

## Executive Summary

Phase 5 bridges the encrypted keystore (Phase 4) with the provider router (Phase 3). The core challenge: **keystore exists but router is initialized WITHOUT keys**. We need async key loading at startup with graceful degradation.

## 1. Keystore API Methods (Available Now)

From `internal/keystore/store.go`:

### Core Methods
- `NewKeyStore() (*KeyStore, error)` - Creates keystore, opens IndexedDB v5
- `RetrieveKey(provider, passphrase) (string, error)` - Decrypts and returns plaintext key
- `KeyExists(provider) (bool, error)` - Checks if key exists without decrypting
- `StoreKey(provider, apiKey, passphrase) error` - Encrypts and stores (already used by UI)

### Supporting Methods
- `ExportKey(provider) (*config.ExportedKey, error)` - Returns encrypted data for export
- `ImportKey(provider, ciphertext, iv, salt) error` - Stores pre-encrypted key
- `DeleteKey(provider) error` - Removes stored key
- `ClearKey(key string)` - Best-effort memory clearing (Go strings immutable limitation)

### Important Implementation Details
1. **Fixed passphrase (v1):** `const passphrase = "webclaw-v1-key"` (line 471 in main.go)
2. **Encryption:** AES-256-GCM via Web Crypto API with PBKDF2 key derivation
3. **Async pattern:** All IndexedDB operations return Promises, use goroutine+channel pattern
4. **Error cases:**
   - `stored == nil` → key not found
   - Decryption failure → wrong passphrase or corrupt data
   - Database blocked → other tabs open

## 2. Router Configuration Structure

From `internal/provider/router.go` and `config.go`:

### Config Structure
```go
type Config struct {
    HTTPReferer      string // "https://github.com/gleicon/webclaw"
    XTitle           string // "WebClaw"
    AnthropicAPIKey  string // Currently EMPTY at startup
    OpenAIAPIKey     string // Currently EMPTY at startup
    OpenRouterAPIKey string // Currently EMPTY at startup
}
```

### Router Behavior
- `NewRouter(config)` - Registers providers ONLY if API keys are non-empty
- `RegisterProvider(name, provider)` - Manual provider registration (for hot-swap)
- `HasProvider(name) bool` - Check if provider available
- Provider routing by model ID: `anthropic/claude-sonnet-4-5`, `openai/gpt-4o`, etc.
- Automatic vendor inference: `claude-*` → anthropic, `gpt-*` → openai

### Current Gap (main.go:60-65)
```go
routerConfig := &provider.Config{
    HTTPReferer: "https://github.com/gleicon/webclaw",
    XTitle:      "WebClaw",
    // NO API KEYS - they're in keystore but never loaded!
}
router := provider.NewRouter(routerConfig) // Empty providers
```

## 3. Async Initialization Patterns in Go WASM

From analyzing `main.go` and `keystore/store.go`:

### Pattern 1: Goroutine-Spawning Promise Handlers (Used in main.go)
```go
setKeyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
    return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
        resolve := resolveReject[0]
        reject := resolveReject[1]
        go func() { // Goroutine for async work
            ks, err := keystore.NewKeyStore()
            if err != nil {
                reject.Invoke(err.Error())
                return
            }
            // ... do work ...
            resolve.Invoke(js.Undefined())
        }()
        return nil
    }))
})
```

### Pattern 2: Channel-Based IndexedDB Operations (Used in keystore)
```go
func (ks *KeyStore) loadKey(provider string) (*StoredKey, error) {
    promise := jsbridge.IDBGet(ks.db, "keystore", provider)
    resultCh := make(chan js.Value, 1)
    errorCh := make(chan error, 1)
    
    promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        resultCh <- args[0]
        return nil
    })).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        errorCh <- fmt.Errorf("load failed: %v", args[0])
        return nil
    }))
    
    select {
    case result := <-resultCh:
        // ... process result ...
    case err := <-errorCh:
        return nil, err
    }
}
```

### Pattern 3: Sequential Initialization with Error Tolerance (main.go:18-85)
1. Each initializer runs sequentially
2. Errors are logged but NOT fatal
3. System continues with partial functionality
4. Event dispatch for UI state updates

### Key Insight for Phase 5
Router initialization must be **deferred** or **updated** after keystore is ready. Options:
1. **Option A:** Initialize router with empty keys, async update with `RegisterProvider()`
2. **Option B:** Block until keystore ready (not recommended - blocks UI)
3. **Option C:** Two-phase init: empty router → async key load → reconfigure

**Recommended:** Option A - Start with empty router, async populate with `RegisterProvider()`

## 4. Error Handling Strategy

### Error Categories

#### A. Keystore Not Ready
- IndexedDB still opening
- Store version mismatch (schema upgrade in progress)
- Other tab blocking database

**Handling:** Retry with timeout, or proceed without keys (degraded mode)

#### B. Key Not Found
- User hasn't entered API key yet
- Key was deleted
- First run scenario

**Handling:** UI shows "Enter API key in Settings" toast, agent uses mock responses or shows error

#### C. Decryption Failure
- Wrong passphrase (v2 user-derived key scenario)
- Corrupt data in IndexedDB

**Handling:** Log error, clear corrupted key, prompt user to re-enter

#### D. Invalid API Key (Runtime)
- 401 Unauthorized from provider
- Rate limited (429)
- Key revoked

**Handling:** Provider returns error via stream, UI shows toast with specific message

### Implementation Pattern
```go
func loadProviderKeysAsync(router *provider.Router) {
    go func() {
        ks, err := keystore.NewKeyStore()
        if err != nil {
            logError("keystore unavailable", err)
            return // Degraded: router stays empty
        }
        
        providers := []string{"anthropic", "openai", "openrouter"}
        for _, p := range providers {
            exists, _ := ks.KeyExists(p)
            if !exists {
                continue // Skip - no key stored
            }
            
            key, err := ks.RetrieveKey(p, passphrase)
            if err != nil {
                logError(fmt.Sprintf("failed to load %s key", p), err)
                continue // Skip this provider
            }
            
            // Register with router
            switch p {
            case "anthropic":
                router.RegisterProvider("anthropic", provider.NewAnthropicProvider(key))
            case "openai":
                router.RegisterProvider("openai", provider.NewOpenAIProvider(key))
            case "openrouter":
                router.RegisterProvider("openrouter", provider.NewOpenRouterProvider(key, ...))
            }
            
            // Clear key from memory (best effort)
            keystore.ClearKey(key)
        }
        
        // Notify UI that providers are ready
        dispatchProvidersReadyEvent(router.AvailableProviders())
    }()
}
```

## 5. Real API Testing Approach

### Testing Strategy (without burning credits)

#### Phase 1: Connection Validation (Minimal Cost)
1. **Test with Anthropic first** - Most reliable streaming, clear error messages
2. **Use cheapest model:** `claude-3-haiku-20240307` (lowest cost, fast)
3. **Single token request:** Set `max_tokens: 1` for validation only
4. **Health check endpoint:** Some providers have `/health` or cheap validation calls

#### Phase 2: Streaming Validation
1. **Short conversation:** "Say hello" → expect streaming response
2. **Verify SSE parsing:** Check event formats match provider implementations
3. **Tool call test:** Single `web_fetch` tool call to verify tool integration

#### Phase 3: Error Scenarios
1. **Missing key:** Expect clear error message in UI
2. **Invalid key:** Use obviously fake key `sk-invalid123`, expect 401
3. **Rate limit simulation:** Use tiny rate-limited test account if available

### Mock Provider for Development
```go
// Already exists in codebase - use when no real keys available
type mockProvider struct{}
func (m *mockProvider) Complete(...) (*Token, error) {
    return &Token{Text: "[Mock] Provider not configured. Enter API key in Settings."}, nil
}
```

### Testing Checklist
- [ ] WASM loads without crashing
- [ ] Keystore opens successfully
- [ ] Keys load from storage (check console logs)
- [ ] Router registers providers (check AvailableProviders())
- [ ] Anthropic API call succeeds (1 token)
- [ ] Streaming delivers tokens incrementally
- [ ] Tool calls work end-to-end
- [ ] Provider switching updates active provider
- [ ] Missing key shows clear error
- [ ] Invalid key shows 401 error

## Key Decisions Required

1. **Passphrase for v1:** Use existing fixed string `webclaw-v1-key` (DECISION: YES)
2. **Key caching:** Decrypt once at startup vs. decrypt per request (DECISION: Decrypt once, register provider, clear key)
3. **Hot-swap support:** Can user change API key without reload? (DECISION: Yes, via Settings tab → re-register provider)
4. **Provider status UI:** Show dots for connected providers? (DECISION: Yes, green/red status indicators)
5. **Fallback behavior:** If no providers configured, show mock or error? (DECISION: Show mock with clear "Configure API key" message)

## Implementation Sketch

```go
// In main.go, after router initialization (line 65):
router := provider.NewRouter(routerConfig)
agentLoop.SetRouter(router)

// NEW: Async key loading
loadProviderKeysAsync(router)

func loadProviderKeysAsync(router *provider.Router) {
    go func() {
        ks, err := keystore.NewKeyStore()
        if err != nil {
            js.Global().Get("console").Call("warn", "keystore load failed:", err.Error())
            return
        }
        
        passphrase := "webclaw-v1-key" // Fixed v1 passphrase
        providersLoaded := []string{}
        
        // Try to load each provider key
        if key, err := ks.RetrieveKey("anthropic", passphrase); err == nil {
            router.RegisterProvider("anthropic", provider.NewAnthropicProvider(key))
            providersLoaded = append(providersLoaded, "anthropic")
            keystore.ClearKey(key)
        }
        
        if key, err := ks.RetrieveKey("openai", passphrase); err == nil {
            router.RegisterProvider("openai", provider.NewOpenAIProvider(key))
            providersLoaded = append(providersLoaded, "openai")
            keystore.ClearKey(key)
        }
        
        if key, err := ks.RetrieveKey("openrouter", passphrase); err == nil {
            router.RegisterProvider("openrouter", provider.NewOpenRouterProvider(
                key, "https://github.com/gleicon/webclaw", "WebClaw"))
            providersLoaded = append(providersLoaded, "openrouter")
            keystore.ClearKey(key)
        }
        
        // Dispatch event to update UI
        js.Global().Call("dispatchEvent",
            js.Global().Get("CustomEvent").New("webclaw:providers-ready",
                map[string]interface{}{
                    "providers": providersLoaded,
                    "count":     len(providersLoaded),
                }))
    }()
}
```

## Requirements Traceability

| Requirement | Phase 5 Implementation |
|-------------|------------------------|
| **PROV-01** | Router routing with `vendor/model-id` format - Keys enable real routing vs mock |
| **PROV-02** | HTTP via syscall/js - Provider implementations already use fetch bridge |
| **SEC-02** | Decrypt in WASM memory - Keys decrypted via `RetrieveKey`, never touch JS |

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Keys don't load (IndexedDB timing) | Retry with exponential backoff, max 3 attempts |
| Memory leak (keys in Go strings) | `ClearKey()` best effort, short-lived key variables |
| Provider 401 after load | Runtime error handling, UI toast with re-auth prompt |
| Multiple tabs blocking DB | Event to notify user: "Close other WebClaw tabs" |
| Async race condition | Router methods check `HasProvider()` before use |

## Next Steps for Planning

1. Create `05-PLAN.md` with exact code changes to `main.go`
2. Define JS event `webclaw:providers-ready` for UI consumption
3. Add provider status indicators to Settings UI
4. Implement health check / test connection button
5. Write integration tests for key load → API call flow

---

*Research completed: 2026-03-01*  
*Ready for 05-PLAN.md creation*
