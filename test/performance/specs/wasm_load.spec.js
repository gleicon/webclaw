const { test, expect } = require('@playwright/test');

const WASM_LOAD_BUDGET = 2000; // 2 seconds
const WASM_LOAD_MAX = 2500;    // Hard limit

// App signals readiness via 'webclaw:host-ready' CustomEvent (detail.wasmLoaded = true)
// Register listener via addInitScript so it's installed before page JS runs

test.describe('WASM Load Performance', () => {
  test('WASM loads within budget', async ({ page }) => {
    await page.addInitScript(() => {
      performance.mark('nav-start');
      window.__webclawHostReady = false;
      window.addEventListener('webclaw:host-ready', () => {
        window.__webclawHostReady = true;
      }, { once: true });
    });

    await page.goto('/', { waitUntil: 'networkidle' });

    const wasmLoadTime = await page.evaluate(() => {
      return new Promise((resolve) => {
        if (window.__webclawHostReady) {
          performance.mark('wasm-ready');
          resolve(performance.measure('wasm-load', 'nav-start', 'wasm-ready').duration);
          return;
        }
        window.addEventListener('webclaw:host-ready', () => {
          performance.mark('wasm-ready');
          resolve(performance.measure('wasm-load', 'nav-start', 'wasm-ready').duration);
        }, { once: true });
        setTimeout(() => resolve(-1), 30000);
      });
    });

    expect(wasmLoadTime).toBeGreaterThan(0);
    expect(wasmLoadTime).toBeLessThan(WASM_LOAD_MAX);
    console.log(`WASM load: ${wasmLoadTime.toFixed(2)}ms`);
  });

  test('WASM cached load is faster', async ({ page }) => {
    // First load — wait for ready
    await page.addInitScript(() => {
      window.__webclawHostReady = false;
      window.addEventListener('webclaw:host-ready', () => {
        window.__webclawHostReady = true;
      }, { once: true });
    });
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => window.__webclawHostReady === true, { timeout: 30000 });

    // Reload and measure cached load
    await page.addInitScript(() => {
      performance.mark('reload-start');
      window.__webclawHostReady = false;
      window.addEventListener('webclaw:host-ready', () => {
        window.__webclawHostReady = true;
      }, { once: true });
    });
    await page.reload({ waitUntil: 'networkidle' });

    const cachedTime = await page.evaluate(() => {
      return new Promise((resolve) => {
        if (window.__webclawHostReady) {
          performance.mark('wasm-reload');
          resolve(performance.measure('wasm-reload', 'reload-start', 'wasm-reload').duration);
          return;
        }
        window.addEventListener('webclaw:host-ready', () => {
          performance.mark('wasm-reload');
          resolve(performance.measure('wasm-reload', 'reload-start', 'wasm-reload').duration);
        }, { once: true });
        setTimeout(() => resolve(-1), 30000);
      });
    });

    expect(cachedTime).toBeGreaterThan(0);
    expect(cachedTime).toBeLessThan(WASM_LOAD_BUDGET / 2);
  });
});
