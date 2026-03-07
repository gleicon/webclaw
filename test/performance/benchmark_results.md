# WebClaw Performance Benchmarks

**Last Updated:** TBD

## Performance Budgets

| Metric | Target | Maximum | Status |
|--------|--------|---------|--------|
| WASM Load | <2000ms | 2500ms | pending |
| First Token | <1000ms | 1500ms | pending |

## Current Results

| Browser | WASM First | WASM Cached |
|---------|------------|-------------|
| Chrome | TBD | TBD |
| Firefox | TBD | TBD |
| Safari | TBD | TBD |

## Running Benchmarks

```bash
cd test/performance
npm install
npm test
```

To run only WASM load tests:

```bash
cd test/performance
npx playwright test specs/wasm_load.spec.js --project=chromium
```

To run only first-token latency tests (requires a configured API key):

```bash
cd test/performance
npx playwright test specs/first_token.spec.js --project=chromium
```

## Measurement Methodology

### WASM Load Time

Measured from `performance.mark('nav-start')` (set before navigation) to
detection of either `window.wasmReady === true` or `[data-wasm-ready]` element.

The application must set one of these signals when the Go WASM module is
fully initialized and ready to handle calls.

### First Token Latency

Measured from `performance.mark('submit')` (set just before clicking send)
to detection of non-empty text in the latest `.assistant-message` element.

This test is skipped automatically if no OpenRouter API key is configured in
`localStorage['webclaw:config']`.

## CI Schedule

- Every PR push: WASM load tests (chromium only)
- Every push to main: WASM load tests
- Nightly (02:00 UTC): Full suite
- On-demand: `workflow_dispatch` trigger in GitHub Actions UI
