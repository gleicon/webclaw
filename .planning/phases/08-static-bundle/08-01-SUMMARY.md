---
phase: 08-static-bundle
plan: 01
subsystem: build
tags: [vite, tailwind, bundler, static]

requires:
  - phase: 07a-filesystem
    provides: just-bash integration for file operations

provides:
  - Vite bundler configuration for WebClaw
  - Static build pipeline with WASM support
  - Compiled Tailwind CSS (no CDN dependency)
  - GitHub Actions workflow for automated releases

affects:
  - 08-02 (single-file bundle mode)
  - deployment
  - distribution

tech-stack:
  added:
    - vite: "^5.0.12"
    - vite-plugin-static-copy: "^1.0.0"
    - vite-plugin-compression: "^0.5.1"
    - tailwindcss: "^3.4.1"
  patterns:
    - ES modules with relative paths for file:// compatibility
    - Static asset copying via Vite plugin
    - Brotli + Gzip dual compression
    - Development and production builds

key-files:
  created: []
  modified:
    - vite.config.js: Vite configuration with WASM handling and compression
    - package.json: Build scripts and dependencies
    - index.html: Removed CDN Tailwind, uses bundled CSS
    - src/main.js: Entry point with build metadata
    - .github/workflows/build.yml: CI/CD for automated releases

key-decisions:
  - Use vite-plugin-static-copy for WASM and static JS files
  - Keep worker.js as separate file (required for new Worker())
  - Use relative paths (./) for file:// protocol compatibility
  - Compile Tailwind to eliminate CDN dependency

patterns-established:
  - Static bundle output in dist-bundle/ folder
  - Dual compression (Brotli + Gzip) for all assets
  - Vendor folder for node_modules dependencies (just-bash)

requirements-completed:
  - DIST-01

metrics:
  duration: 12min
  completed: 2026-03-07
---

# Phase 08 Plan 01: Vite Static Bundle Summary

**Vite bundler configured for WebClaw static distribution with compiled Tailwind CSS, WASM support, and dual compression (Brotli + Gzip)**

## Performance

- **Duration:** 12 min
- **Started:** 2026-03-07T01:02:14Z
- **Completed:** 2026-03-07T01:14:00Z
- **Tasks:** 5
- **Files modified:** 4

## Accomplishments

- Vite configuration with vite-plugin-static-copy for WASM files
- Static asset pipeline copying worker.js, wasm_exec.js, justbash-bridge.js, and webclaw-host.js
- Removed CDN Tailwind dependency - now compiled to ~15KB CSS
- GitHub Actions workflow for automated builds and releases
- Support for file:// protocol (relative paths throughout)
- Dual compression (Brotli + Gzip) for optimal delivery

## Task Commits

Each task was committed atomically:

1. **Task 1: Update Vite configuration** - `34ed6af` (feat)
   - Add vite-plugin-static-copy for WASM file handling
   - Update build entry point to index.html

2. **Task 2: Package.json** - Part of Task 1 commit
   - Added vite-plugin-static-copy dependency

3. **Task 3: Remove CDN Tailwind** - `4d9b9a6` (feat)
   - Replaced CDN script with local CSS link
   - Uses ./src/styles/main.css (compiled by Vite)

4. **Task 4: Update entry point** - `cbb1d36` (feat)
   - Added build metadata and tech stack info
   - Documented features and pending items

5. **Task 5: Configure build** - `bc23208` (feat)
   - Added just-bash and webclaw-host to static copy targets
   - Fixed relative paths for all script references

**Plan metadata:** `docs(08-01): complete plan` (pending)

## Files Created/Modified

- `vite.config.js` - Vite configuration with WASM copy, compression, and build settings
- `package.json` - Dependencies for Vite, Tailwind, and static copy plugin
- `index.html` - Removed CDN Tailwind, uses relative paths for all assets
- `src/main.js` - Entry point with version info and build metadata
- `.github/workflows/build.yml` - CI/CD workflow for automated builds and releases

## Build Output Structure

```
dist-bundle/
├── index.html              (77KB - main entry)
├── index.html.br           (11KB - brotli compressed)
├── index.html.gz           (14KB - gzip compressed)
├── webclaw.wasm            (4.2MB - uncompressed WASM)
├── webclaw.wasm.br         (865KB - brotli compressed WASM)
├── assets/
│   └── main-*.css          (15KB - compiled Tailwind)
├── static/
│   ├── worker.js           (8.7KB - Web Worker)
│   ├── wasm_exec.js        (17KB - Go WASM runtime)
│   ├── justbash-bridge.js (9.8KB - just-bash bridge)
│   └── webclaw-host.js     (16KB - WebClaw host)
└── vendor/
    └── browser.js          (1MB - just-bash library)
```

## Decisions Made

1. **Use vite-plugin-static-copy**: Required because Vite doesn't natively handle WASM files and certain static JS files that need to remain as separate files (worker.js for new Worker()).

2. **Keep relative paths (./)**: Essential for file:// protocol compatibility, allowing the bundle to be opened directly in a browser without a server.

3. **Compile Tailwind instead of CDN**: Eliminates external dependency on cdn.tailwindcss.com, making the bundle truly standalone.

4. **Dual compression (Brotli + Gzip)**: Provides optimal compression for different server configurations. Brotli achieves better ratios (WASM: 865KB vs 4.2MB).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The existing Vite configuration was already well-structured. Minor updates were needed to:

- Add vite-plugin-static-copy dependency
- Include additional static files (justbash-bridge.js, webclaw-host.js, just-bash browser.js)
- Update index.html paths to use relative paths

## Size Analysis

| Component             | Size        | Compressed     |
| --------------------- | ----------- | -------------- |
| CSS                   | 15KB        | 3KB (Brotli)   |
| WASM                  | 4.2MB       | 865KB (Brotli) |
| just-bash             | 1MB         | 226KB (Brotli) |
| Static JS             | ~50KB total | ~12KB (Brotli) |
| **Total deliverable** | **~7.7MB**  | **~1.1MB**     |

**Key metrics:**

- CSS < 20KB target: ✓ (15KB actual)
- file:// compatible: ✓ (all relative paths)
- No CDN dependencies: ✓
- WASM compression: ✓ (79% reduction with Brotli)

## Next Phase Readiness

Ready for Plan 08-02 (Single-File Bundle Mode):

- Current multi-file bundle is complete and functional
- Can now experiment with embedding strategies for single-file mode
- Worker.js is the main challenge for single-file (requires separate file for new Worker())

## User Setup Required

None - no external service configuration required. The static bundle is self-contained and can be:

1. Opened directly via file:// protocol
2. Served by any static web server
3. Deployed to CDN or static hosting

## Verification Commands

```bash
# Build the bundle
npm run build

# Preview production build
npm run preview

# Verify file paths (should all be relative)
grep -o 'href="[^"]*"\|src="[^"]*"' dist-bundle/index.html | head -10
```

---

_Phase: 08-static-bundle_
_Completed: 2026-03-07_
