# Phase 8: Static Website Bundle Design

## User Intent

Create a **zero-dependency static website bundle** for WebClaw that can be:
1. Opened directly (file:// or served statically)
2. Distributed as a single folder or zipped file
3. Used immediately without npm install or build steps
4. Contains ALL dependencies bundled (WASM, JS, CSS, just-bash when integrated)

## Architecture Options Presented

### Option A: ES Modules + CDN (Rejected)
- **Why rejected:** Requires internet connection, external dependencies violate "zero external dependencies" goal

### Option B: Bundled with Vite/Rollup (Primary)
- **Use case:** Multi-file bundle, clean separation, cache-friendly
- **Build step:** Required for distribution, but users get pre-built bundle
- **Best for:** Regular users who want offline capability

### Option C: Inline Everything (Secondary)
- **Use case:** Single HTML file, email-able, ultimate portability
- **Build step:** Required, more complex bundling
- **Best for:** Sharing, demos, emergency access

## Decisions Made

### Locked Decisions

1. **Build Tool: Vite**
   - Rationale: Fast, modern, excellent WASM support, built-in brotli/gzip compression, ES modules output
   - Rollup considered but Vite provides faster dev experience and better defaults

2. **Primary Distribution: Multi-file bundle**
   - Rationale: WASM is 865KB brotli compressed, inlining it adds ~33% overhead (base64 = 1.15MB)
   - Single-file mode offered as secondary option for special use cases

3. **just-bash Integration: Bundled library**
   - Rationale: When Phase 7a completes, just-bash will be tree-shaken and bundled
   - No CDN fallback for static bundle

4. **CSS: Tailwind CDN replaced with compiled CSS**
   - Current index.html uses `cdn.tailwindcss.com`
   - Static bundle will compile only used Tailwind classes (~10KB vs 300KB full CDN)

### Claude's Discretion Areas

1. **npm package structure:** Standard package.json with bin entry for CLI
2. **GitHub Actions:** Automated build and release workflow
3. **Docker:** Single static file served by nginx (optional, low priority)
4. **CLI commands:** `npx webclaw-static serve` (static server), `npx webclaw-static build` (custom build)

## Deferred Ideas

- **WASM SIMD optimization:** Future optimization, not v1
- **Service Worker for offline PWA:** Out of scope for v1 bundle
- **Plugin marketplace integration:** Requires server component
- **Auto-updater:** Requires network, contradicts offline-first

## Key Questions to Answer

1. **just-bash bundle size:** Need to check @jstz-dev/just-bash package for tree-shaking potential
2. **WASM encoding:** Keep as .wasm.br file vs inline base64 trade-off
3. **Distribution formats:** npm package, GitHub releases, single-file download
4. **Versioning strategy:** Semantic versioning for bundle releases

## Trade-off Analysis

| Format | Size | Portability | Build Required | Best For |
|--------|------|-------------|----------------|----------|
| Multi-file | ~920KB | Good (folder) | No (pre-built) | Most users |
| Single-file | ~1.2MB | Excellent (1 file) | No (pre-built) | Sharing/demos |
| CDN-based | ~50KB | Poor (needs net) | No | Development only |
| Source + build | ~4.3MB | Poor | Yes | Contributors |

## Success Criteria

1. **Download and run:** User can download zip, extract, double-click index.html, and it works
2. **No external requests:** All assets bundled, DevTools Network tab shows only local files
3. **just-bash works:** When Phase 7a is integrated, file operations work without npm install
4. **Multiple formats:** At least multi-file and single-file bundles available
5. **npm install optional:** `npm install webclaw-static` works but isn't required

## Phase Boundaries

**Starts after:** Phase 6 complete (WASM pipeline, agent loop, memory system working)  
**Can run parallel to:** Phase 7a (just-bash integration)  
**Must complete before:** Phase 7 (Local Bridge Binary) public release

## Next Steps

1. Create 08-01-PLAN.md — Vite bundler setup with multi-file output
2. Create 08-02-PLAN.md — Single-file inline mode
3. Create 08-03-PLAN.md — npm package and distribution
