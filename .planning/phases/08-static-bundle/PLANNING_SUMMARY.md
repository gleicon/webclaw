# Phase 8: Static Website Bundle - Planning Summary

## Overview

Created comprehensive plans for bundling WebClaw into a zero-dependency static website distribution. This enables users to run WebClaw without npm install, build steps, or external dependencies.

## Architecture Decisions

### Locked Decisions (User-Specified)

1. **Build Tool: Vite** (over Rollup/Parcel)
   - Chosen for: Fast builds, modern defaults, excellent WASM support
   - Rollup considered but Vite provides better developer experience

2. **Primary Format: Multi-file bundle**
   - Size: ~920KB (brotli compressed)
   - Trade-off: Multiple files vs single file for cache efficiency
   - Single-file mode offered as secondary option

3. **Zero External Dependencies**
   - All assets bundled (no CDN Tailwind)
   - Works with `file://` protocol
   - Can be opened directly in browser

### Claude's Discretion Areas

1. **WASM Compression Strategy**
   - Multi-file: Use brotli (.wasm.br) - best compression
   - Single-file ultimate: Use gzip with DecompressionStream API
   - Base64 adds ~33% overhead - acceptable for portability

2. **npm Package Scope**
   - Unscoped: `webclaw-static` (simpler)
   - Fallback: `@gleicon/webclaw` if taken

3. **CLI Features**
   - Serve command with port configuration
   - Auto-open browser option
   - Help documentation

## Plans Created

### Plan 08-01: Vite Bundler Setup
**Wave:** 1  
**Dependencies:** None  
**Objective:** Configure Vite to build production-ready multi-file bundle

**Key Tasks:**
1. Create vite.config.js with relative paths and WASM handling
2. Set up package.json with build scripts and dependencies
3. Configure Tailwind CSS compilation (replace CDN)
4. Create ES module entry points
5. Build and verify output structure

**Output Files:**
- `vite.config.js` - Vite configuration
- `package.json` - npm scripts and dependencies
- `tailwind.config.js`, `postcss.config.js` - CSS processing
- `src/main.js`, `src/styles.css` - Entry modules
- `dist-bundle/` - Built output (~920KB)

**Key Technical Challenges:**
- Web Workers must remain as separate files (loaded via `new Worker()`)
- Solution: Use `?worker` suffix or Blob URLs
- WASM files need copy plugin (not processed by Vite)
- Relative paths required for `file://` protocol support

### Plan 08-02: Single-File Inline Mode
**Wave:** 2  
**Dependencies:** 08-01  
**Objective:** Create single-file distribution for ultimate portability

**Key Tasks:**
1. Create WASM inlining utility script
2. Configure Vite for single-file output (inlineDynamicImports)
3. Implement Web Worker inlining via Blob URLs
4. Create optional WASM inlining with gzip + DecompressionStream
5. Build and verify both standard and ultimate modes

**Distribution Variants:**

| Variant | Files | Size | Use Case |
|---------|-------|------|----------|
| Standard | webclaw.html + webclaw.wasm.br | ~965KB | Folder distribution |
| Ultimate | webclaw-ultimate.html | ~1.3MB | Email, single file sharing |

**Technical Solutions:**
- **Worker inlining:** Embed worker.js as string constant, create Blob URL at runtime
- **WASM compression:** gzip instead of brotli for better JS decompressor support
- **Decompression:** Use native DecompressionStream API with pako fallback
- **Base64 overhead:** 33% size increase accepted for portability benefit

### Plan 08-03: Distribution Packaging
**Wave:** 2  
**Dependencies:** 08-01  
**Objective:** Create npm package, CLI, GitHub Actions, and Docker distribution

**Key Tasks:**
1. Configure package.json for npm publishing (bin entry, files whitelist)
2. Create CLI tool (`webclaw-static serve`) with zero dependencies
3. Set up GitHub Actions workflow (build → release → publish)
4. Create Dockerfile with nginx serving
5. Write comprehensive README with all distribution methods

**Distribution Channels:**

1. **npm (Primary)**
   - Command: `npx webclaw-static serve`
   - Benefits: No install, automatic updates, familiar to JS devs
   - Package includes: dist-bundle/, dist-singlefile/, bin/

2. **GitHub Releases (Secondary)**
   - Trigger: Git tag push (`v*.*.*`)
   - Assets:
     - webclaw-v{version}.zip (multi-file)
     - webclaw-v{version}-singlefile.zip (single-folder)
     - webclaw-v{version}-ultimate.html (standalone)
   - Auto-generated release notes

3. **Docker (Tertiary)**
   - Command: `docker run -p 8080:80 gleicon/webclaw`
   - Multi-stage build for minimal size (~25MB)
   - nginx with brotli support

**CLI Commands:**
- `webclaw-static serve [port]` - Start static server (default 8080)
- `webclaw-static serve --open` - Serve and open browser
- `webclaw-static --help` - Show usage

## Bundle Size Analysis

### Current Assets
```
webclaw.wasm          4.2MB  (raw)
webclaw.wasm.br       865KB  (brotli compressed)
wasm_exec.js          17KB   (Go runtime)
webclaw-host.js       11KB   (Host bridge)
worker.js             8.7KB  (Web Worker)
index.html            47KB   (UI)
Tailwind CSS          ~300KB (CDN, uncompiled)
```

### Target Bundle Sizes

| Format | JS+CSS | WASM | Total | Notes |
|--------|--------|------|-------|-------|
| Development | ~75KB | 4.2MB | 4.3MB | Uncompressed |
| Multi-file | ~30KB | 865KB | ~920KB | Brotli, minified |
| Single-file | ~40KB | 865KB | ~965KB | Inlined JS+CSS |
| Ultimate | - | - | ~1.3MB | Everything inline + base64 |

### Size Optimizations

1. **Tailwind CSS:** CDN (~300KB) → Compiled (~15KB) = 95% reduction
2. **JS Minification:** Terser with dead code elimination
3. **WASM Compression:** Brotli reduces 4.2MB → 865KB (79% reduction)
4. **Code Splitting:** Manual chunks for caching

## just-bash Integration Considerations

When Phase 7a (just-bash Filesystem) completes, it will be integrated into the static bundle:

**Current Plan 07a-01 approach:**
- npm dependency: `@jstz-dev/just-bash`
- Loaded via: `import { Bash } from '@jstz-dev/just-bash'` or CDN

**Static Bundle Integration:**
- just-bash will be tree-shaken and bundled by Vite
- Estimated additional size: 50-200KB (depends on commands used)
- No external CDN requests - fully bundled

**Update Path:**
1. Add `@jstz-dev/just-bash` to package.json dependencies
2. Import in src/main.js
3. Rebuild - Vite handles tree-shaking automatically
4. Update size documentation

## Build Configuration Summary

### vite.config.js (Multi-file)
```javascript
{
  base: './',                    // file:// protocol support
  build: {
    target: 'es2020',
    outDir: 'dist-bundle',
    assetsInlineLimit: 4096,   // Don't inline WASM
    rollupOptions: {
      output: {
        manualChunks: {          // Code splitting
          'wasm-exec': ['./static/wasm_exec.js']
        }
      }
    }
  },
  plugins: [
    viteStaticCopy({             // Copy WASM files
      targets: [
        { src: 'dist/webclaw.wasm.br', dest: '' }
      ]
    })
  ]
}
```

### vite.singlefile.config.js
```javascript
{
  base: './',
  build: {
    outDir: 'dist-singlefile',
    rollupOptions: {
      output: {
        inlineDynamicImports: true,  // Single chunk
        manualChunks: undefined
      }
    },
    assetsInlineLimit: 10000000,      // Inline everything
    cssCodeSplit: false
  }
}
```

## File Structure

### Source (Development)
```
webclaw/
├── index.html              # Entry HTML (uses CDN Tailwind)
├── static/
│   ├── wasm_exec.js        # Go WASM runtime
│   ├── webclaw-host.js     # Host bridge
│   └── worker.js           # Web Worker
├── dist/
│   ├── webclaw.wasm        # 4.2MB
│   └── webclaw.wasm.br     # 865KB
├── vite.config.js          # Build config
└── package.json            # Dependencies
```

### Distribution (Production)
```
dist-bundle/                # ~920KB
├── index.html              # ~2KB (references assets)
├── assets/
│   ├── index-*.js          # ~25KB (bundled JS)
│   ├── index-*.css         # ~15KB (compiled Tailwind)
│   └── wasm_exec-*.js      # ~17KB
└── webclaw.wasm.br         # 865KB

dist-singlefile/            
├── webclaw.html            # ~120KB (inline JS+CSS)
├── webclaw.wasm.br         # 865KB
└── webclaw-ultimate.html   # ~1.3MB (everything inline)
```

## npm Package Structure
```
webclaw-static/
├── package.json            # npm manifest
├── bin/
│   └── webclaw-static.js   # CLI entry (executable)
├── dist-bundle/            # Pre-built multi-file
├── dist-singlefile/        # Pre-built single-file
├── README.md               # Distribution docs
└── LICENSE
```

## Wave Structure

```
Wave 1 (Parallel):
├── 08-01: Vite bundler setup

Wave 2 (Depends on Wave 1):
├── 08-02: Single-file inline mode
└── 08-03: Distribution packaging
```

## Success Criteria by Plan

### 08-01 Success
- [ ] Vite config with relative paths and WASM handling
- [ ] Build produces dist-bundle/ with all assets
- [ ] No CDN dependencies (Tailwind compiled)
- [ ] file:// protocol compatible
- [ ] Total size <1MB

### 08-02 Success
- [ ] Single-file build scripts work
- [ ] webclaw.html inlines JS+CSS, references external WASM
- [ ] webclaw-ultimate.html contains everything
- [ ] Worker inlining via Blob URLs
- [ ] Both formats work with file://

### 08-03 Success
- [ ] CLI serves WebClaw on localhost
- [ ] npm package valid (`npm pack` succeeds)
- [ ] GitHub Actions workflow exists
- [ ] Docker image builds successfully
- [ ] README documents all 4 distribution methods

## Trade-offs and Decisions

### Build Tool: Vite vs Rollup vs Parcel
| Tool | Pros | Cons | Decision |
|------|------|------|----------|
| Vite | Fast, modern, WASM support | Newer ecosystem | ✓ Selected |
| Rollup | Mature, smaller output | Slower builds | Considered |
| Parcel | Zero config | Less control | Not selected |

### WASM Distribution: External vs Inline
| Format | Size | Portability | Decision |
|--------|------|-------------|----------|
| External (.wasm.br) | 865KB | Good (folder) | ✓ Primary |
| Inline base64 | ~1.15MB | Excellent (1 file) | Secondary |

### Worker Strategy: Separate vs Blob URL
| Strategy | Complexity | Performance | Decision |
|----------|------------|-------------|----------|
| Separate file | Simple | Native | ✓ Multi-file |
| Blob URL | Complex | Good | ✓ Single-file |

## User Setup Requirements

### For npm Publishing
- npm account
- NPM_TOKEN secret in GitHub repo settings
- Package name availability check: `npm view webclaw-static`

### For GitHub Releases
- GitHub Actions enabled in repo settings
- Write permissions for Actions (Settings → Actions → General)

### For Docker Hub (Optional)
- Docker Hub account
- DOCKER_USERNAME and DOCKER_PASSWORD secrets

## Next Steps

1. **Execute Plan 08-01:** Set up Vite bundler (can start immediately)
2. **Parallel Development:** Can run alongside Phase 7a (just-bash)
3. **No Blockers:** Phase 8 depends only on Phase 6 (complete)
4. **Execute Phase:** `/gsd-execute-phase 08-static-bundle`

## Documentation Created

1. **08-CONTEXT.md** - Design decisions and trade-offs
2. **08-01-PLAN.md** - Vite bundler setup (5 tasks)
3. **08-02-PLAN.md** - Single-file inline mode (5 tasks)
4. **08-03-PLAN.md** - Distribution packaging (5 tasks)
5. **ROADMAP.md updated** - Phase 8 with 3 plans documented

## Estimated Effort

| Plan | Tasks | Est. Time |
|------|-------|-----------|
| 08-01 | 5 | 30-45 min |
| 08-02 | 5 | 45-60 min |
| 08-03 | 5 | 30-45 min |
| **Total** | **15** | **2-2.5 hours** |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Worker Blob URL CSP issues | High | Test in multiple browsers, use proper MIME types |
| WASM compression compatibility | Medium | Provide uncompressed fallback, test widely |
| npm package name taken | Low | Use scoped package `@gleicon/webclaw` |
| file:// protocol restrictions | Medium | Document clearly, recommend http when possible |
| just-bash size increase | Low | Tree-shaking, measure before/after |

---

*Planning completed: 2026-03-05*  
*Phase 8 ready for execution*
