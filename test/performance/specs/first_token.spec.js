const { test, expect } = require('@playwright/test');

const FIRST_TOKEN_BUDGET = 1000; // 1 second
const FIRST_TOKEN_MAX = 1500;    // Hard limit

test.describe('First Token Performance', () => {
  test.beforeEach(async ({ page }) => {
    // Register listener before navigation — app fires 'webclaw:host-ready' on init
    await page.addInitScript(() => {
      window.__webclawHostReady = false;
      window.addEventListener('webclaw:host-ready', () => {
        window.__webclawHostReady = true;
      }, { once: true });
    });
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => window.__webclawHostReady === true, { timeout: 30000 });
  });

  test('First token arrives within budget', async ({ page }) => {
    // Check if API key configured
    const hasKey = await page.evaluate(() => {
      const cfg = JSON.parse(localStorage.getItem('webclaw:config') || '{}');
      return !!(cfg.providers?.openrouter?.apiKey);
    });

    if (!hasKey) {
      console.log('Skipping: No API key');
      test.skip();
    }

    const input = await page.locator('textarea, [data-testid="message-input"]').first();
    await input.fill('Say hello');

    await page.evaluate(() => performance.mark('submit'));
    await page.locator('button[type="submit"]').first().click();

    const latency = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const check = () => {
          const msgs = document.querySelectorAll('.assistant-message, [data-testid="assistant-message"]');
          if (msgs.length > 0 && msgs[msgs.length - 1].textContent.trim().length > 0) {
            performance.mark('first-token');
            resolve(performance.measure('latency', 'submit', 'first-token').duration);
            return;
          }
          setTimeout(check, 10);
        };
        setTimeout(() => resolve(-1), 30000);
        check();
      });
    });

    expect(latency).toBeGreaterThan(0);
    expect(latency).toBeLessThan(FIRST_TOKEN_MAX);
    console.log(`First token: ${latency.toFixed(2)}ms`);
  });
});
