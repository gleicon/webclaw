---
phase: 10
slug: browser-local-model
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Playwright (existing in `test/`) |
| **Config file** | `playwright.config.js` (exists) |
| **Quick run command** | `npx playwright test test/10-local-model.spec.js --project=chromium` |
| **Full suite command** | `npx playwright test --project=chromium` |
| **Estimated runtime** | ~60 seconds (model loading dominates) |

---

## Sampling Rate

- **After every task commit:** Run `npx playwright test test/10-local-model.spec.js -k "smoke" --project=chromium`
- **After every plan wave:** Run `npx playwright test --project=chromium`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 0 | LOCAL-01 | smoke | `npx playwright test test/10-local-model.spec.js -k "backend detection" --project=chromium` | Wave 0 | ⬜ pending |
| 10-01-02 | 01 | 0 | LOCAL-01 | smoke | `npx playwright test test/10-local-model.spec.js -k "model list" --project=chromium` | Wave 0 | ⬜ pending |
| 10-02-01 | 02 | 1 | LOCAL-02 | integration | `npx playwright test test/10-local-model.spec.js -k "offline chat" --project=chromium` | Wave 0 | ⬜ pending |
| 10-03-01 | 03 | 1 | LOCAL-03 | unit (JS) | `node test/unit/local-tool-json.test.js` | Wave 0 | ⬜ pending |
| 10-04-01 | 04 | 1 | LOCAL-04 | manual | N/A — Go WASM routing; verified through full path smoke | manual-only | ⬜ pending |
| 10-04-02 | 04 | 1 | LOCAL-04 | smoke | `npx playwright test test/10-local-model.spec.js -k "cloud fallback" --project=chromium` | Wave 0 | ⬜ pending |
| 10-05-01 | 05 | 2 | LOCAL-05 | integration | `npx playwright test test/10-local-model.spec.js -k "cache persistence" --project=chromium` | Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/10-local-model.spec.js` — Playwright specs for LOCAL-01 (backend detection, model list), LOCAL-02 (offline chat), LOCAL-04 (cloud fallback), LOCAL-05 (cache persistence)
- [ ] `test/unit/local-tool-json.test.js` — node unit test for LOCAL-03 (JSON mode tool output parsing)
- [ ] `static/local-llm-worker.js` — must be created as part of Wave 0 scaffolding
- [ ] `internal/provider/local.go` — LocalProvider stub; must exist before integration tests
- [ ] `internal/jsbridge/local_bridge.go` — jsbridge for model download progress callbacks
- [ ] `npm install @mlc-ai/web-llm @wllama/wllama` — packages not yet in package.json

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Provider router routes "local/..." model IDs to LocalProvider in WASM | LOCAL-04 | Go WASM unit tests not feasible outside browser build target | Load app, select local model, verify routing via console logs showing LocalProvider invoked |
| Token generation speed ≥ 3 tok/s on WebGPU hardware | LOCAL-02 | Hardware-dependent, no reliable automated benchmark in CI | Run on physical WebGPU-capable machine, measure token rate via UI or console timing |
| Safari Cache API persistence after browser restart | LOCAL-05 | Safari 26 behavior not fully documented; needs live device test | Open on Safari 26, download model, restart browser, reload page — verify no re-download |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
