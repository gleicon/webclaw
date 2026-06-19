# Getting started

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.25+ | compiles WASM binary |
| Node.js | 20+ | Tailwind CSS, Vite bundler |
| brotli | any | compresses WASM for distribution |

On macOS: `brew install go node brotli`

## Local development

```bash
git clone https://github.com/gleicon/webclaw.git
cd webclaw

npm install

# Compile WASM + Tailwind CSS + brotli
make build

# Copy WASM to the static dir served by the dev server
cp dist/webclaw.wasm static/webclaw.wasm

# Start the Go dev server
make serve          # http://localhost:8080
```

After changing Go source files, run `make build && cp dist/webclaw.wasm static/webclaw.wasm` and reload.

After changing Tailwind classes in `index.html`, run `make css` and reload.

The dev server is the Go binary in `cmd/devserver/`, not Vite. It serves `static/` with correct WASM MIME types and a `/vendor/browser.js` handler. `npm run dev` does not handle WASM correctly.

## Make targets

| Target | Description |
|---|---|
| `make build` | Tailwind CSS + Go WASM compile + brotli compress |
| `make css` | Tailwind only (`src/styles/main.css` → `static/main.css`) |
| `make serve` | Start Go dev server on port 8080 |
| `make test` | Go unit tests (no browser) |
| `make test-wasm` | WASM-tagged provider tests via Node |
| `make test-all` | Both test targets |
| `make docker-build` | Build Docker image `webclaw:latest` |
| `make docker-run` | Run image on http://localhost:8080 |
| `make clean` | Remove build artifacts |

## Testing

```bash
make test           # fast, no browser, covers agent + tool logic
make test-wasm      # runs provider tests compiled to WASM via Node
make test-all       # both
```

Run `make test-all` before `make build` to catch logic errors without a full WASM compile.

## Production build

### Static hosting (Netlify, Vercel, S3, etc.)

```bash
make build
cp dist/webclaw.wasm static/webclaw.wasm
npm run build       # Vite bundle → dist-bundle/
```

Upload `dist-bundle/` to your host. Set these response headers:

| File | Content-Type | Content-Encoding |
|---|---|---|
| `*.wasm` | `application/wasm` | — |
| `*.wasm.br` | `application/wasm` | `br` |

`deploy/nginx.conf` is a working nginx configuration for self-hosted deployments.

### Docker

The `Dockerfile` does the full build inside the image (no pre-built artifacts needed):

- Stage 1: `golang:1.26-alpine` + Node + brotli — builds WASM and Vite bundle
- Stage 2: `nginx:alpine` — serves `dist-bundle/`

```bash
make docker-build                        # → webclaw:latest
make docker-run                          # http://localhost:8080
docker run -p 3000:80 webclaw:latest     # custom port
docker tag webclaw:latest registry/webclaw:v1.0
docker push registry/webclaw:v1.0
```

If `golang:1.26-alpine` is not on Docker Hub, update the `FROM` line in `Dockerfile` to the latest available 1.25+ tag.

## Chrome Built-in AI (Gemini Nano)

Runs inference on-device with no API key. Requires Chrome 138+ and a one-time model download (~4 GB).

Enable:
1. `chrome://flags` → search **Prompt API** → set **Enabled**
2. Restart Chrome

For public deployments that need the Prompt API without the flag, obtain an Origin Trial token from the Chrome Origin Trials portal and paste it in **Settings → Browser AI**. WebClaw stores it in `localStorage` and injects it into the page `<meta>` tag on load.

## Embed widget configuration

`webclaw.init()` options:

| Option | Required | Default | Description |
|---|---|---|---|
| `wasmUrl` | yes | — | path to `webclaw.wasm` |
| `workerUrl` | yes | — | path to `worker.js` |
| `systemPrompt` | no | `'You are a helpful assistant.'` | system prompt prepended to every session |
| `proxyUrl` | no | — | OpenAI-compatible proxy URL; operator supplies auth server-side |
| `model` | no | `'gemini-nano/local'` | provider/model string |
| `tools` | no | `[]` | tool allowlist; empty = no tools |
| `context` | no | `{}` | initial context object injected into system prompt |
| `memoryNamespace` | no | `'default'` | IndexedDB key prefix for this widget instance |
| `position` | no | `'bottom-right'` | `'bottom-right'` or `'bottom-left'` |
| `theme.primaryColor` | no | `'#6366f1'` | CSS color for buttons and user bubbles |

`webclaw.setContext(obj)` replaces the live context object; call it on each navigation. The new context is included in the next stream request.
