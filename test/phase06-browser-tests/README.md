# Phase 06 Browser E2E Tests

Comprehensive browser-based end-to-end tests for WebClaw Phase 06 using Playwright.

## Overview

These tests run in a REAL browser (Chromium) and verify actual UI behavior, console logs, IndexedDB operations, and user interactions. They are TRUE E2E tests, not unit tests or mocked tests.

## Key Capabilities

- **Console Log Capture**: Captures all `console.log` output from webclaw
- **IndexedDB Access**: Direct browser API access to verify storage
- **UI Interaction**: Real clicks, typing, and navigation
- **Screenshot on Failure**: Automatic visual debugging
- **Video Recording**: Captures test execution

## File Structure

```
test/
├── package.json                    # Updated with Playwright
├── playwright.config.js            # Playwright configuration
├── run-phase06-e2e.js              # Test runner with dev server management
├── phase06-browser-tests/
│   ├── helpers.js                  # Shared test utilities
│   ├── 01-summarization.spec.js    # 20-message threshold
│   ├── 02-token-counting.spec.js   # UI token display
│   ├── 03-memory-flush.spec.js     # MEMORY.md + IndexedDB
│   ├── 04-tool-registry.spec.js    # Tool console logs
│   ├── 05-memory-search.spec.js    # Memory search tool
│   ├── 06-provider-failover.spec.js # Provider switching
│   ├── 07-fail-fast.spec.js        # Error handling
│   ├── 08-storage-hygiene.spec.js  # IndexedDB quota
│   ├── 09-smoke-test.spec.js       # Startup verification
│   ├── 10-health-tracking.spec.js  # Health status logs
│   └── 11-async-embedder.spec.js   # Memory initialization
```

## Installation

```bash
cd test
npm install
npx playwright install chromium
```

## Running Tests

### Automated (Recommended)
```bash
# Runs full suite with automatic dev server
npm run test:phase06
```

### Headed Mode (See Browser)
```bash
npm run test:phase06:headed
```

### Individual Test Files
```bash
npx playwright test phase06-browser-tests/09-smoke-test.spec.js
```

### Debug Mode
```bash
npm run test:phase06:debug
```

### UI Mode
```bash
npm run test:phase06:ui
```

## Test Descriptions

### 01-summarization.spec.js
- **Purpose**: Verifies 20-message threshold triggers summarization
- **Tests**:
  - Sends 20 messages and captures "summarization triggered" console log
  - Logs summarization trigger message
  - Maintains message context after summarization

### 02-token-counting.spec.js
- **Purpose**: Verifies token counts display in UI
- **Tests**:
  - Displays token count after sending message
  - Shows token metrics in console
  - Tracks input vs output tokens
  - Updates token count dynamically

### 03-memory-flush.spec.js
- **Purpose**: Verifies MEMORY.md and IndexedDB operations
- **Tests**:
  - Creates MEMORY.md entry in IndexedDB
  - Logs memory flush operations
  - Persists data across operations
  - Handles IndexedDB quota checks

### 04-tool-registry.spec.js
- **Purpose**: Verifies tool console logs and tool_use trigger
- **Tests**:
  - Logs tool registration on startup
  - Displays tool activity in side panel
  - Logs tool_use events
  - Shows tool names in console
  - Triggers tool_use for memory operations

### 05-memory-search.spec.js
- **Purpose**: Uses memory search tool in UI
- **Tests**:
  - Searches memory and returns results
  - Logs memory search operations
  - Handles no results gracefully
  - Displays memory search in tool activity panel
  - Supports semantic memory search

### 06-provider-failover.spec.js
- **Purpose**: Verifies provider initialization
- **Tests**:
  - Initializes primary provider on startup
  - Logs provider selection
  - Displays selected model in UI
  - Handles model switching
  - Shows provider status in console

### 07-fail-fast.spec.js
- **Purpose**: Verifies error handling
- **Tests**:
  - Logs errors to console
  - Handles network errors gracefully
  - Shows error messages in UI
  - Does not crash on invalid input
  - Logs fail-fast behavior

### 08-storage-hygiene.spec.js
- **Purpose**: Checks IndexedDB quota via browser API
- **Tests**:
  - Reports storage quota
  - Has access to IndexedDB
  - Lists IndexedDB databases
  - Handles storage persistence request
  - Opens IndexedDB connections
  - Cleans up test databases
  - Monitors storage growth

### 09-smoke-test.spec.js
- **Purpose**: Verifies startup logs
- **Tests**:
  - Shows all components ready on startup
  - Has all UI components visible
  - Working tab navigation
  - No JavaScript errors
  - WASM binary loaded

### 10-health-tracking.spec.js
- **Purpose**: Verifies health status in console
- **Tests**:
  - Logs health status on startup
  - Tracks system health during operation
  - Shows component health in logs
  - Tracks memory health
  - Reports health metrics

### 11-async-embedder.spec.js
- **Purpose**: Verifies memory initialization logs
- **Tests**:
  - Logs embedder initialization
  - Initializes memory system asynchronously
  - Logs vector store operations
  - Supports embedding generation
  - Handles async memory loading

## Test Configuration

### playwright.config.js
- **Base URL**: `http://localhost:8080`
- **Workers**: 1 (sequential execution)
- **Retries**: 2 on CI
- **Screenshot**: On failure
- **Video**: On first retry
- **Trace**: On first retry

### Environment Variables
- `HEADLESS=false`: Run in headed mode
- `CI=true`: CI mode with extra reporters

## Console Log Capture

All tests capture console logs automatically. Key patterns to watch for:

```javascript
// Summarization
"summarization triggered"
"Summarization:"
"threshold"

// Tokens
"token count"
"input tokens"
"output tokens"

// Tools
"tool_use"
"calling tool"
"tool registered"

// Memory
"memory flush"
"MEMORY.md"
"IndexedDB"

// Health
"health"
"ready"
"initialized"

// Embedder
"embedder"
"embedding"
"vector"
```

## IndexedDB Access

Tests can directly access IndexedDB via `page.evaluate()`:

```javascript
const data = await page.evaluate(async () => {
  const request = indexedDB.open('webclaw-memory');
  // ... operations
});
```

## Adding New Tests

1. Create new `.spec.js` file in `phase06-browser-tests/`
2. Import helpers: `import { sendChatMessage } from './helpers.js'`
3. Use `test.beforeEach` to navigate and wait for WASM
4. Capture console logs with `page.on('console', ...)`
5. Use UI helpers for interaction
6. Run with `npx playwright test phase06-browser-tests/your-test.spec.js`

## Troubleshooting

### Dev Server Not Starting
```bash
cd .. && go run ./cmd/devserver/
# In another terminal:
cd test && npm run test:phase06
```

### Playwright Not Installed
```bash
cd test && npx playwright install chromium
```

### Tests Failing
- Check `test-results/` for screenshots
- Check `playwright-report/` for HTML report
- View trace: `npx playwright show-trace playwright-report/trace.zip`

## Integration with CI

```yaml
# .github/workflows/test.yml
- name: Run Phase 06 E2E Tests
  run: |
    cd test
    npm ci
    npx playwright install chromium
    npm run test:phase06
```
