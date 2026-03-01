# Phase 5: Live AI Provider Connection - Context

**Gathered:** 2026-03-01
**Status:** Ready for planning
**Source:** Technical gap analysis

## Phase Boundary

Phase 5 closes the loop between infrastructure and reality. We have:
- ✅ Encrypted API key storage (keystore)
- ✅ Provider implementations with HTTP fetch
- ✅ UI for entering and saving keys
- ✅ Router infrastructure

But we're missing:
- ❌ Loading keys from keystore into router at startup
- ❌ Async initialization sequence (keystore → router)
- ❌ Real API testing and error handling
- ❌ Provider credential refresh/update

This phase makes WebClaw actually talk to Claude, GPT-4o, or other models instead of returning mock responses.

## Implementation Decisions

### Architecture Decisions (LOCKED)
- **Async initialization**: Keystore must be ready before router is configured
- **No blocking on startup**: Router starts with empty keys, gets populated async
- **Graceful degradation**: Missing keys → clear error, not crash
- **Provider hot-swap**: Changing model in dropdown updates active provider config

### Claude's Discretion
- How to handle keystore passphrase (v1: fixed, v2: user prompt)
- Error message formatting for UI
- Retry/backoff logic for failed API calls
- Whether to cache keys in memory (vs. fetch every request)

## Specific Ideas

### Initialization Flow
1. WASM loads
2. Keystore opens (IndexedDB v5)
3. Check each provider for stored keys
4. Retrieve and decrypt keys (async)
5. Configure router with live keys
6. Agent loop now uses real providers

### Key Management
- Keys stored encrypted at rest
- Decrypted only in WASM memory during use
- Never cached in JavaScript
- Cleared from memory after API call (best effort)

### Testing Strategy
1. Test with Anthropic API (most reliable)
2. Verify streaming works end-to-end
3. Test error cases: missing key, invalid key, rate limit
4. Verify provider switching works (change model dropdown)

### UI Integration
- Show "Connecting..." during async init
- Display provider status (green dot = connected, red = error)
- Error toast for missing/invalid API key

## Deferred Ideas

**Phase 6+:**
- Key rotation/revocation UI
- Multiple key profiles (work vs. personal)
- Provider-specific settings (temperature, max_tokens)
- API usage tracking/display

## Technical Notes

### Current Gap
```go
// main.go line 60-65
routerConfig := &provider.Config{
    HTTPReferer: "https://github.com/gleicon/webclaw",
    XTitle:      "WebClaw",
    // NO API KEYS! They're in keystore but never loaded.
}
```

### Solution Sketch
```go
// In initializeKeystore() or new async init function:
ks, _ := keystore.NewKeyStore()
// Retrieve each key
anthropicKey, _ := ks.RetrieveKey("anthropic", passphrase)
openaiKey, _ := ks.RetrieveKey("openai", passphrase)
// Pass to router
routerConfig.AnthropicAPIKey = anthropicKey
routerConfig.OpenAIAPIKey = openaiKey
```

---

*Phase: 05-live-ai-connection*
*Context gathered: 2026-03-01*
