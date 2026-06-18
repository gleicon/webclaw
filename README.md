# WebClaw

AI agent that runs entirely in your browser via WebAssembly. No server required. API keys stay in your browser, encrypted at rest.

## Try it

### Fastest: Docker (no build step)

```bash
# Build image — compiles Go WASM + Vite bundle inside the container
make docker-build

# Serve at http://localhost:8080
make docker-run
```

### From source (dev mode)

Prerequisites: Go 1.25+, Node 20+, brotli CLI.

```bash
git clone https://github.com/gleicon/webclaw.git
cd webclaw

npm install

# Compile WASM + CSS (re-run after Go or Tailwind changes)
make build
cp dist/webclaw.wasm static/webclaw.wasm

# Start the Go dev server
make serve           # http://localhost:8080
```

The Go devserver (`make serve`) is the primary dev entry point — it handles WASM MIME types and serves everything from `static/`. Do **not** use `npm run dev` for WASM work.

## First use

1. Open **Settings → API Keys**
2. Enter an API key for at least one provider (Anthropic, OpenAI, or OpenRouter)
   — **or** use Chrome 138+ with Gemini Nano enabled (no key needed, see below)
3. Switch to the **Chat** tab and start a conversation

## Providers

| Provider | Key required | Notes |
|---|---|---|
| Anthropic | Yes | Claude models |
| OpenAI | Yes | GPT models |
| OpenRouter | Yes | 100+ models via one key |
| Chrome Built-in / Gemini Nano | No | Chrome 138+, local inference, ~4 GB model download |

### Chrome Built-in AI (Gemini Nano)

Runs on-device — no API key, no network after first model download.

**Enable it:**
1. Open Chrome, navigate to `chrome://flags`
2. Search **Prompt API**, set to **Enabled**
3. Restart Chrome

The model selector shows **"Chrome Built-in / Gemini Nano — Chrome 138+ only"** on non-Chrome browsers so the option is always visible as a hint.

**Origin Trial token** (optional, for public deployments): Settings → Browser AI → paste your token from the Chrome Origin Trials portal. WebClaw stores it in `localStorage` and injects it at page load.

## Embed in your own site

Drop a chat widget into any page with a single script tag:

```html
<script src="/static/embed.js"></script>
<script>
  webclaw.init({
    wasmUrl:      '/static/webclaw.wasm',   // required
    workerUrl:    '/static/worker.js',      // required
    systemPrompt: 'You are a support agent for Acme Corp.',
    proxyUrl:     '/api/ai-proxy',          // server-side proxy — no raw key in browser
    model:        'gemini-nano/local',      // default; any provider/model works
    position:     'bottom-right',          // or 'bottom-left'
    theme:        { primaryColor: '#6366f1' },
  });

  // Update context as the user navigates (SPA support)
  webclaw.setContext({ user: 'alice', page: '/billing' });
</script>
```

The widget uses Shadow DOM for CSS isolation from the host page. Use `proxyUrl` in production so API keys stay server-side.

## Building

```bash
make build          # CSS + compile Go WASM + brotli compress
make css            # CSS only (after adding new Tailwind classes)
```

After `make build`, copy the binary if you want the dev server to serve it:

```bash
cp dist/webclaw.wasm static/webclaw.wasm
```

### Full Vite bundle (for static hosting)

```bash
npm run build       # outputs to dist-bundle/
```

The Vite build copies WASM, worker, vendor, and embed files automatically. `dist-bundle/` can be served from any static host or CDN.

## Testing

```bash
make test           # Go unit tests — no browser needed, fast
make test-wasm      # WASM-tagged provider tests via Node (requires node in PATH)
make test-all       # both
```

Run tests before `make build` to catch logic errors without a full WASM compile cycle.

## Docker

The Dockerfile does a complete build inside the container:

```
Stage 1: golang:1.26-alpine + Node
  → GOOS=js GOARCH=wasm go build
  → brotli compress WASM
  → npm run build (Vite bundle)

Stage 2: nginx:alpine
  → serves dist-bundle/ on port 80
```

```bash
make docker-build               # builds image tagged webclaw:latest
make docker-run                 # serves on http://localhost:8080
docker run -p 3000:80 webclaw:latest   # custom port
```

If `golang:1.26-alpine` is not available, update the `FROM` line in `Dockerfile` to match your installed Go version (must be ≥ 1.25).

## Deploying

### Static hosting (Netlify, Vercel, S3 + CloudFront, etc.)

```bash
make build
cp dist/webclaw.wasm static/webclaw.wasm
npm run build
# upload dist-bundle/ to your host
```

Set these response headers for WASM files:

| File pattern | Content-Type | Content-Encoding |
|---|---|---|
| `*.wasm` | `application/wasm` | — |
| `*.wasm.br` | `application/wasm` | `br` |

`deploy/nginx.conf` in this repo is a ready-to-use nginx configuration.

### Docker / self-hosted

```bash
make docker-build
docker tag webclaw:latest your-registry/webclaw:v1.0
docker push your-registry/webclaw:v1.0
```

## Architecture

```
index.html + static/main.css       ← UI (Tailwind, vanilla JS)
       │
       ├── main thread WASM         ← config, keystore, OAuth JS bridge
       │   static/webclaw.wasm
       │
       └── Web Worker               ← agent loop, LLM streaming
           static/worker.js
           └── worker WASM instance ← same binary, second instance
```

Both contexts load the same WASM binary. `console.log` calls from Go appear **twice** in the browser console — this is expected, not a bug.

The full agent loop (tool calls, memory, summarization) runs in the Web Worker so the UI stays responsive during long responses.

## Security

- API keys encrypted in IndexedDB with Web Crypto (AES-256-GCM)
- Keys only exist as plaintext inside WASM linear memory — never in JS
- OAuth uses PKCE — no client secret stored anywhere
- `embed.js` accepts a `proxyUrl` instead of a raw key, keeping secrets server-side

## Browser support

Requires WebAssembly and IndexedDB.

| Browser | Chat | Gemini Nano |
|---|---|---|
| Chrome 90+ | ✓ | — |
| Chrome 138+ | ✓ | ✓ (enable via flags) |
| Firefox 90+ | ✓ | — |
| Safari 15+ | ✓ | — |
| Edge 90+ | ✓ | — |
