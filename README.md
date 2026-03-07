# WebClaw Static

WebClaw AI agent as a zero-dependency static website bundle. Runs entirely in the browser with WebAssembly.

## Quick Start

### Option 1: npx (no install)

```bash
npx webclaw-static serve
# Opens at http://localhost:8080
```

### Option 2: npm install

```bash
npm install -g webclaw-static
webclaw-static serve --open
```

### Option 3: Download

Download the latest release:

- [webclaw-v1.0.0.zip](https://github.com/gleicon/webclaw/releases) - Multi-file bundle (best for servers)
- [webclaw-v1.0.0-singlefile.zip](https://github.com/gleicon/webclaw/releases) - Single-folder distribution
- [webclaw-v1.0.0-ultimate.html](https://github.com/gleicon/webclaw/releases) - Standalone HTML file

Extract and open `index.html` (or `webclaw.html`) in any modern browser.

### Option 4: Docker

```bash
docker run -p 8080:80 gleicon/webclaw:latest
```

Then open http://localhost:8080

## Features

- **Zero dependencies**: No Node.js, no build step, no npm install required
- **Offline capable**: Works without internet once loaded
- **AI Providers**: Anthropic Claude, OpenAI GPT, OpenRouter
- **Secure**: API keys encrypted in browser storage
- **Tools**: Web search, web fetch, memory management, file operations (79+ bash commands)
- **File size**: ~1MB total (920KB compressed)

## Usage

1. Open WebClaw in browser
2. Go to Settings tab
3. Enter your API key for at least one provider (Anthropic, OpenAI, or OpenRouter)
4. Return to Chat tab and start conversing

## CLI Commands

```bash
# Serve the multi-file bundle (optimized)
webclaw-static serve

# Serve on custom port
webclaw-static serve --port=3000

# Serve and open browser automatically
webclaw-static serve --open

# Serve single-file version
webclaw-static serve --singlefile

# Serve ultimate standalone version
webclaw-static serve --ultimate

# Open WebClaw in browser (file:// or localhost)
webclaw-static open

# Show help
webclaw-static --help
```

## File Sizes

| Format      | Size   | Use Case                              |
| ----------- | ------ | ------------------------------------- |
| Multi-file  | ~920KB | Web server hosting (best performance) |
| Single-file | ~1MB   | Folder distribution, file sharing     |
| Ultimate    | ~1.3MB | Email attachment, single file sharing |

## Browser Compatibility

- Chrome 90+
- Firefox 90+
- Safari 15+
- Edge 90+

Requires WebAssembly and IndexedDB support.

## Security

- API keys encrypted at rest using Web Crypto API (AES-256-GCM)
- All processing happens client-side
- No data sent to external servers (except AI provider APIs)
- Keys never exist as plaintext in JavaScript - only in WASM linear memory

## Distribution Channels

### npm Registry

```bash
# Run without installing
npx webclaw-static serve

# Install globally
npm install -g webclaw-static
```

Package: `webclaw-static`

### GitHub Releases

Download pre-built bundles from the [releases page](https://github.com/gleicon/webclaw/releases).

Releases are automatically created when pushing a tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Docker Hub

```bash
# Run latest
docker run -p 8080:80 gleicon/webclaw:latest

# Run specific version
docker run -p 8080:80 gleicon/webclaw:1.0.0
```

Image: `gleicon/webclaw`

## Building from Source

```bash
# Clone repository
git clone https://github.com/gleicon/webclaw.git
cd webclaw

# Install dependencies
npm install

# Build all variants
npm run build:all

# Output:
# - dist-bundle/ (multi-file, optimized)
# - dist-singlefile/ (single-file and ultimate HTML)
```

## Development

```bash
# Start dev server
npm run dev

# Build and preview
npm run build
npm run preview

# Build single-file version
npm run build:singlefile

# Build ultimate version (WASM inlined)
npm run build:singlefile:ultimate
```

## Architecture

WebClaw consists of three main components:

1. **Go Core (WASM)**: Compiled from Go to WebAssembly using TinyGo
2. **JavaScript Host**: Thin layer providing browser APIs to WASM
3. **Web Worker**: Handles streaming LLM responses without blocking UI

All components run in the browser - no server required.

## Contributing

Contributions welcome! Please open an issue or pull request on GitHub.

## License

MIT License - see LICENSE file for details.

## Links

- Repository: https://github.com/gleicon/webclaw
- Issues: https://github.com/gleicon/webclaw/issues
- npm: https://www.npmjs.com/package/webclaw-static
- Docker Hub: https://hub.docker.com/r/gleicon/webclaw
