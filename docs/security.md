# Security model

## Data locality

All data stays in the browser. Nothing is sent to any server operated by WebClaw.

Outbound connections from the page are:
- LLM provider API endpoints (Anthropic, OpenAI, OpenRouter, or your `proxyUrl`)
- OAuth authorization servers when a user connects an integration
- The target URLs of tool calls (web fetch, web search) initiated by the agent

There are no analytics calls, no telemetry, and no session logging.

## Storage

WebClaw uses three browser storage mechanisms:

| Store | Contents | Scope |
|---|---|---|
| IndexedDB `webclaw` db, `keystore` object store | Encrypted API keys | Origin |
| IndexedDB `webclaw` db, `oauth` object store | Encrypted OAuth tokens | Origin |
| IndexedDB `webclaw` db, `identity` object store | Identity files (system prompt, MEMORY.md, etc.) | Origin |
| `localStorage` | OAuth client IDs (non-secret), UI state, origin trial token | Origin |

"Origin" means `scheme://host:port`. Data stored at `https://example.com` is not accessible from `http://example.com` or any other origin. This is enforced by the browser's same-origin policy, not by application code.

If you are running two separate deployments of WebClaw (e.g., on different domains or ports), each has a completely separate IndexedDB and localStorage. They cannot see each other's data.

## API key encryption

Keys are encrypted before being written to IndexedDB. The encryption path:

```
passphrase (constant in WASM binary)
    │
    ▼
SubtleCrypto.importKey (PBKDF2 base key)
    │
    ▼
SubtleCrypto.deriveKey
  algorithm: PBKDF2
  hash:      SHA-256
  salt:      16 random bytes (crypto.getRandomValues, per key)
  iterations: 100,000   (OWASP 2023 recommendation)
  output:    AES-GCM 256-bit key
    │
    ▼
SubtleCrypto.encrypt
  algorithm: AES-GCM
  iv:        12 random bytes (crypto.getRandomValues, per key)
    │
    ▼
StoredKey { ciphertext, iv, salt }  →  IndexedDB
```

The encryption and decryption happen inside WASM linear memory. The plaintext key is passed to the provider HTTP client inside WASM and is not exposed to JavaScript at any point. After use, `keystore.ClearKey()` overwrites the string bytes in WASM memory before releasing the reference.

The encryption passphrase is a compile-time constant (`webclaw-v1-key`) embedded in the WASM binary. This means the encryption protects against direct IndexedDB inspection or database export, but not against someone who has the WASM binary and can extract the passphrase. The primary threat model is casual credential exposure (e.g., someone looking at browser DevTools or exporting a profile), not a targeted attack on a machine the attacker already controls.

## OAuth tokens

OAuth tokens are stored in the same `webclaw` IndexedDB under the `oauth` object store. Tokens are encrypted with the same PBKDF2/AES-GCM scheme as API keys.

OAuth flows use PKCE (Proof Key for Code Exchange). No client secret is required or stored. The authorization callback URL is `about:blank`; the auth code is extracted via `postMessage` when the popup navigates.

Client IDs (not secrets) are stored in `localStorage` under `webclaw_oauth_clientids`. Client IDs are not sensitive, but they are still scoped to the page origin.

## Instance isolation

Each running WASM instance has its own memory space. WebClaw runs two instances per page load:

- Main thread WASM — handles config, keystore, OAuth bridge
- Web Worker WASM — handles the agent loop and LLM streaming

Both instances read from the same IndexedDB (same origin), but they do not share WASM linear memory. A key decrypted in the worker is not accessible in the main thread, and vice versa.

If a user opens two tabs of the same WebClaw deployment, they share the same IndexedDB (same origin) but each tab has independent WASM memory. Conversations are not shared between tabs unless the user explicitly exports and imports.

Two different deployments on different origins cannot access each other's data. This boundary is enforced by the browser.

## Embed widget

When WebClaw is embedded via `embed.js`, the same-origin rules apply to the origin that serves the WASM and worker files, not the host page's origin. The Shadow DOM widget cannot be read or styled by the host page's JavaScript or CSS. However, the host page JavaScript can call `webclaw.setContext()` to pass contextual data into the system prompt — treat this as user-provided input, not trusted data.

In production embeds, use `proxyUrl` instead of storing an API key in the browser. The proxy handles provider authentication server-side; the browser never sees a raw API key.

## What WebClaw does not protect against

- A user (or malware) with full control of the browser profile can export IndexedDB data. The encryption only slows inspection; it does not prevent it from a fully compromised machine.
- The passphrase `webclaw-v1-key` is in the WASM binary. Anyone with the binary can derive the encryption key.
- The Web Worker has network access. Tool calls (web fetch, web search, just-bash bridge) make outbound requests from the browser. These are scoped by browser CSP if you deploy one.
- Memory snapshots or heap dumps of the browser process will contain plaintext keys while they are in use.
