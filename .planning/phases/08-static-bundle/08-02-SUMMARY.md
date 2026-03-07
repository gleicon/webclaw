---
phase: 08-static-bundle
plan: 02
subsystem: build

tags: [vite, single-file, wasm, inline, base64]

# Dependency graph
requires:
  - phase: 08-01
    provides: Vite build infrastructure, static asset handling
provides:
  - Single-file HTML build mode
  - WASM inlining for ultimate portability
  - Web Worker Blob URL creation
affects: [distribution, deployment, packaging]

# Tech tracking
tech-stack:
  added: [base64-encoding, gzip-compression, blob-urls]
  patterns:
    [index-based-replacement, reverse-order-processing, content-escaping]

key-files:
  created:
    - scripts/inline-wasm.js - WASM base64 encoding and decompression loader
    - scripts/build-singlefile.js - Orchestrates single-file build with inlining
    - vite.singlefile.config.js - Vite configuration for single-file output
  modified:
    - package.json - Added build:singlefile and build:singlefile:ultimate scripts

key-decisions:
  - Used index-based string replacement instead of global regex to avoid matching patterns in inlined content
  - Process scripts/stylesheets in reverse order so indices don't shift during replacement
  - Escape < and > characters to \x3C and \x3E in inlined content to prevent nested HTML parsing
  - Use gzip compression (not brotli) for inline WASM to leverage DecompressionStream API

patterns-established:
  - Index-based HTML inlining: Use exec() to collect positions, then replace from end to start
  - Content escaping: Replace HTML metacharacters to prevent regex matching inside inlined content
  - Two-tier single-file: Standard (JS+CSS inline, WASM external) vs Ultimate (everything inline)

requirements-completed:
  - DIST-01

# Metrics
duration: 78min
completed: 2026-03-07
---

# Phase 08 Plan 02: Single-File Bundle Mode Summary

**Single-file distribution mode for WebClaw with inline JS/CSS/WASM, enabling email sharing and offline distribution via one HTML file**

## Performance

- **Duration:** 78 min
- **Started:** 2026-03-07T01:06:51Z
- **Completed:** 2026-03-07T02:25:00Z
- **Tasks:** 5
- **Files modified:** 4

## Accomplishments

- Created `scripts/inline-wasm.js` for base64 encoding and WASM decompression loader generation
- Created `scripts/build-singlefile.js` with index-based inlining to handle complex content
- Created `vite.singlefile.config.js` with inlineDynamicImports and cssCodeSplit disabled
- Implemented Web Worker inlining via Blob URLs with wasm_exec.js embedded
- Standard single-file: ~1.17MB HTML + external WASM files (portable with dependencies)
- Ultimate single-file: ~2.7MB standalone HTML with everything including WASM

## Task Commits

Each task was committed atomically:

1. **Task 1: Create WASM inlining utility** - `5a862e4` (feat)
2. **Task 2: Create single-file build configuration** - `3482f19` (feat)
3. **Task 3: Convert to ES modules** - `f5c0933` (refactor)
4. **Task 4: Fix vendor file paths** - `92ea43a` (fix)
5. **Task 5: Fix duplication issues with index-based replacement** - `3014593` (fix)

**Plan metadata:** `docs(08-02): complete single-file bundle mode` - TBD

## Files Created/Modified

- `scripts/inline-wasm.js` - WASM base64 encoding, gzip compression, DecompressionStream loader
- `scripts/build-singlefile.js` - Build orchestration with index-based inlining
- `vite.singlefile.config.js` - Vite config for single-file builds
- `package.json` - Added build:singlefile and build:singlefile:ultimate scripts

## Decisions Made

1. **Index-based replacement vs global regex**: Global regex replacement caused issues where vendor content (just-bash 1MB bundle) contained strings matching script tag patterns, leading to massive duplication (35x). Switched to collecting all matches with indices first, then replacing from end to start.

2. **Content escaping**: Even with index-based replacement, we escape `<` → `\x3C` and `>` → `\x3E` in inlined content to prevent any possibility of the content being interpreted as HTML structure.

3. **Gzip for inline WASM**: Used gzip instead of brotli for inline WASM compression because browsers have native DecompressionStream API support for gzip. Brotli would require bundling a decompressor library.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed script inlining duplication issue**

- **Found during:** Task 5 (Build verification)
- **Issue:** Vendor content (just-bash browser.js) contained strings matching `<script src="./static/...">` patterns, causing global regex replacement to match inside already-inlined content
- **Fix:** Rewrote inlineScripts() to use index-based replacement: collect all matches with exec(), then process from end to start using substring replacement at exact indices
- **Files modified:** scripts/build-singlefile.js
- **Verification:** Build now produces correct file sizes (1.17MB standard, 2.7MB ultimate instead of 7.6MB with duplication)

**2. [Rule 3 - Blocking] Converted build scripts to ES modules**

- **Found during:** Task 2 (First build attempt)
- **Issue:** Project uses `"type": "module"` in package.json, but scripts used CommonJS require/module.exports
- **Fix:** Converted both scripts to ES modules with import/export syntax
- **Files modified:** scripts/inline-wasm.js, scripts/build-singlefile.js
- **Verification:** npm run build:singlefile executes without module errors

**3. [Rule 3 - Blocking] Fixed vendor file path resolution**

- **Found during:** Task 2 (Build failed to find vendor files)
- **Issue:** Build script couldn't find vendor/browser.js (just-bash) because it wasn't copied to dist-singlefile/
- **Fix:** Added node_modules path resolution for vendor files that aren't copied by Vite in single-file mode
- **Files modified:** scripts/build-singlefile.js
- **Verification:** vendor/browser.js is correctly resolved and inlined

---

**Total deviations:** 3 auto-fixed (all blocking issues related to build process)
**Impact on plan:** All fixes necessary for correct build functionality. No scope creep.

## Issues Encountered

1. **Massive content duplication (35x):** The initial global regex replacement approach caused the vendor bundle content to be matched by subsequent regex patterns, leading to exponential duplication. Required complete rewrite of inlining logic to use position-based replacement.

2. **ES module vs CommonJS mismatch:** The project uses ES modules but the build scripts were written with CommonJS syntax, causing immediate runtime errors.

3. **WASM compression choice:** Initial plan considered brotli for better compression, but gzip is better for inline WASM because: (a) browsers have native DecompressionStream API for gzip, (b) no additional decompressor library needed, (c) compression ratio difference (~1.15MB vs ~1.16MB) is negligible for this use case.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both single-file build modes working:
  - Standard: `npm run build:singlefile` → webclaw.html + webclaw.wasm.br
  - Ultimate: `npm run build:singlefile:ultimate` → webclaw-ultimate.html
- All verification checks passing
- File sizes within expected ranges
- Worker inlining and WASM fetch interception working

Ready for Phase 08-03: Distribution (npm package, CLI, Docker)

---

_Phase: 08-static-bundle_
_Completed: 2026-03-07_
