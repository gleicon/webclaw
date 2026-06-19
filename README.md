# WebClaw

Go/WASM browser client for LLM providers. Runs without a server. API keys are encrypted in IndexedDB and decrypted inside WASM linear memory — they never appear in JS.

## Quick start

**Docker** (no local toolchain required):

```bash
make docker-build   # compiles Go WASM + Vite bundle inside the image
make docker-run     # serves on http://localhost:8080
```

**From source** (requires Go 1.25+, Node 20+, brotli):

```bash
npm install
make build
cp dist/webclaw.wasm static/webclaw.wasm
make serve          # http://localhost:8080
```

Use `make serve` (Go devserver), not `npm run dev`. The Go devserver sets the correct WASM MIME types.

First run: open **Settings → API Keys** and enter a key for at least one provider.

## Providers

| Provider | Key required | Notes |
|---|---|---|
| Anthropic | yes | Claude models |
| OpenAI | yes | GPT models |
| OpenRouter | yes | 100+ models |
| Chrome Built-in (Gemini Nano) | no | Chrome 138+; enable via `chrome://flags` → Prompt API |

## Embed widget

`static/embed.js` is a drop-in chat widget using Shadow DOM for style isolation.

```html
<script src="/static/embed.js"></script>
<script>
webclaw.init({
  wasmUrl:   '/static/webclaw.wasm',  // required
  workerUrl: '/static/worker.js',     // required
  proxyUrl:  '/api/ai-proxy',         // recommended in production; keeps keys off the client
  model:     'gemini-nano/local',     // default; any provider/model string works
  position:  'bottom-right',          // or 'bottom-left'
  theme:     { primaryColor: '#6366f1' },
});
// Update context on navigation (SPA support)
webclaw.setContext({ user: 'alice', page: '/billing' });
</script>
```

## Architecture

```
index.html + static/main.css     — UI (Tailwind, vanilla JS)
  │
  ├── main thread WASM            — config, keystore, OAuth bridge
  │   static/webclaw.wasm
  │
  └── Web Worker                  — agent loop, LLM streaming
      static/worker.js
      └── worker WASM instance    — same binary, second instance
```

Both threads load the same WASM binary. Go `console.log` calls appear twice in the browser console — this is expected.

## Security

- API keys encrypted at rest: IndexedDB + Web Crypto AES-256-GCM
- Keys exist as plaintext only inside WASM linear memory, never in JS
- OAuth: PKCE flow — no client secret stored anywhere
- Embed widget: `proxyUrl` keeps raw API keys off the browser entirely

## Browser support

Requires WebAssembly and IndexedDB.

| Browser | Support |
|---|---|
| Chrome 90+ | full |
| Chrome 138+ | full + Gemini Nano (enable via chrome://flags → Prompt API) |
| Firefox 90+ | full |
| Safari 15+ | full |
| Edge 90+ | full |

---

[Getting started guide](docs/getting-started.md) — local dev, testing, production deployment.
