const { test, expect } = require('@playwright/test');

const WASM_LOAD_BUDGET = 2000; // 2 seconds
const WASM_LOAD_MAX = 2500;    // Hard limit

test.describe('WASM Load Performance', () => {
  test('WASM loads within budget', async ({ page }) => {
    await page.evaluate(() => performance.mark('nav-start'));
    await page.goto('/', { waitUntil: 'networkidle' });

    const wasmLoadTime = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const check = () => {
          if (window.wasmReady || document.querySelector('[data-wasm-ready]')) {
            performance.mark('wasm-ready');
            resolve(performance.measure('wasm-load', 'nav-start', 'wasm-ready').duration);
            return;
          }
          setTimeout(check, 10);
        };
        setTimeout(() => resolve(-1), 30000);
        check();
      });
    });

    expect(wasmLoadTime).toBeGreaterThan(0);
    expect(wasmLoadTime).toBeLessThan(WASM_LOAD_MAX);
    console.log(`WASM load: ${wasmLoadTime.toFixed(2)}ms`);
  });

  test('WASM cached load is faster', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => window.wasmReady);

    await page.evaluate(() => performance.mark('reload-start'));
    await page.reload({ waitUntil: 'networkidle' });

    const cachedTime = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const check = () => {
          if (window.wasmReady) {
            performance.mark('wasm-ready');
            resolve(performance.measure('wasm-reload', 'reload-start', 'wasm-ready').duration);
            return;
          }
          setTimeout(check, 10);
        };
        setTimeout(() => resolve(-1), 30000);
        check();
      });
    });

    expect(cachedTime).toBeGreaterThan(0);
    expect(cachedTime).toBeLessThan(WASM_LOAD_BUDGET / 2);
  });
});
