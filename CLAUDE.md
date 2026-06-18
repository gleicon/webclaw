# WebClaw Development Guidelines

## Architecture Overview

WebClaw is a Go/WASM browser app with a dual-server dev setup:

- **Go devserver** (`make serve`) — static file server on port 8080. Serves compiled assets from the project root. Does NOT compile CSS or JS.
- **Vite dev server** (`npm run dev`) — processes CSS/JS on the fly. Runs on port 8080, but falls back to 8081 if 8080 is occupied.

The Go devserver is the primary dev entry point. Use it with `make serve`.

## WASM Architecture

The app loads the same WASM binary (`static/webclaw.wasm`) in two contexts:

1. **Main thread** — handles config, identity, keystore, OAuth JS bridge
2. **Web worker** (`static/worker.js`) — handles agent loop / LLM streaming

Because both run `main()`, all `console.log` calls from Go appear **twice** in the browser console. This is expected, not a bug.

## CSS

### How it works
- Source: `src/styles/main.css` (raw `@tailwind` directives — browsers cannot use this directly)
- Compiled output: `static/main.css` (what the Go devserver serves)
- `index.html` links to `./static/main.css`

### Rules
- After adding new Tailwind classes to `index.html` or any source file, run:
  ```
  make css
  ```
- Only use valid Tailwind classes. The gray scale is `gray-100` through `gray-900` — there is no `gray-850`.
- `make build` automatically runs `make css` first.

## Makefile Targets

| Target | What it does |
|--------|-------------|
| `make build` | Compile CSS + WASM + brotli compress |
| `make css` | Compile Tailwind CSS only (`src/styles/main.css` -> `static/main.css`) |
| `make serve` | Run the Go static file devserver on port 8080 |
| `make clean` | Remove all build artifacts including `static/main.css` |

## Static Asset Serving

`vite-plugin-static-copy` only copies files during `vite build` — it does NOT serve files in `vite dev` mode. Two fixes are in place for dev:

1. **`vite.config.js`** — `serveVendorInDev()` plugin middleware serves `node_modules/just-bash/dist/bundle/browser.js` at `/vendor/browser.js` when using `npm run dev`.
2. **`cmd/devserver/main.go`** — explicit handler serves the same file at `/vendor/browser.js` when using `make serve`.

If you add new `node_modules` files that need to be served in dev, add handlers in both places.

## OAuth Integration

### Client IDs
- Providers (Twitter, Google, GitHub, Notion) are registered at startup with **empty** client IDs.
- Users must enter their OAuth App's Client ID in Settings -> Connected Services.
- Client IDs are stored in `localStorage` under key `webclaw_oauth_clientids` and applied to WASM on every page load.
- `window.webclaw.oauth.setClientId(provider, clientId)` sets the ID at runtime.

### No client secret needed
This app uses OAuth 2.0 **PKCE** flow. The client secret is replaced by the code verifier/challenge pair. Only the Client ID is required.

### OAuth App setup (all providers)
Set the Authorization callback URL / redirect URI to: `about:blank`

The popup flow extracts the auth code via `postMessage` when the popup navigates.

## HTML / JS Safety Guidelines

### Build UI with DOM methods, not string injection
The security hook blocks dynamic string-based DOM injection to prevent XSS. Always use `createElement` + `textContent` + `appendChild` when building elements with variable content.

For hardcoded status badges or icons (trusted static strings), direct assignment is acceptable.

## Building WASM

```bash
make build       # full build: CSS + WASM + brotli
make serve       # start Go devserver
```

After rebuilding WASM, copy to the static dir:
```bash
cp dist/webclaw.wasm static/webclaw.wasm
```

The devserver serves `static/webclaw.wasm`. The `dist/` directory is the build output; `static/` is what gets served.
