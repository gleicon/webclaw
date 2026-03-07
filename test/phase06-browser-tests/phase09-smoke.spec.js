/**
 * Phase 09: Social Integrations - Smoke Tests
 * Does NOT require live OAuth credentials.
 */

import { test, expect } from '@playwright/test';

const BASE_URL = 'http://localhost:8080';

async function waitForOAuth(page, timeout = 20000) {
  return page.waitForFunction(() => {
    return window.webclaw &&
           window.webclaw.oauth &&
           typeof window.webclaw.oauth.isConnected === 'function';
  }, { timeout });
}

test.describe('Phase 09: Social Integrations - Infrastructure', () => {

  test('1. App loads without crash errors', async ({ page }) => {
    const crashErrors = [];
    page.on('pageerror', err => crashErrors.push(err.message));
    await page.goto(BASE_URL);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    const crashTypes = crashErrors.filter(e =>
      e.includes('TypeError') || e.includes('ReferenceError') || e.includes('SyntaxError')
    );
    console.log('Total page errors:', crashErrors.length, 'Crash errors:', crashTypes);
    expect(crashTypes).toHaveLength(0);
  });

  test('2. Connected Services section exists in DOM', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('#connected-services-section')).toBeAttached();
    await expect(page.locator('#connected-services-list')).toBeAttached();
    console.log('Connected Services section found in DOM');
  });

  test('3. All 4 provider cards are rendered after host-ready', async ({ page }) => {
    await page.goto(BASE_URL);
    await waitForOAuth(page);
    await page.waitForTimeout(1000);

    const listContent = await page.evaluate(() => {
      const list = document.getElementById('connected-services-list');
      return list ? list.innerHTML : '';
    });

    const providers = ['twitter', 'google', 'github', 'notion'];
    for (const p of providers) {
      const found = listContent.toLowerCase().includes(p);
      console.log('Provider "' + p + '" in list: ' + found);
      expect(found, 'Provider card for "' + p + '" not found').toBe(true);
    }
  });

  test('4. webclaw.oauth API is exposed with all methods', async ({ page }) => {
    await page.goto(BASE_URL);
    await waitForOAuth(page);

    const status = await page.evaluate(() => ({
      oauth: typeof window.webclaw.oauth,
      oauthKeys: Object.keys(window.webclaw.oauth),
    }));

    console.log('WASM oauth status:', JSON.stringify(status));
    expect(status.oauth).toBe('object');
    expect(status.oauthKeys).toContain('isConnected');
    expect(status.oauthKeys).toContain('getConnectionStatus');
    expect(status.oauthKeys).toContain('initiateConnection');
    expect(status.oauthKeys).toContain('disconnect');
  });

  test('5. Unconnected provider returns false from isConnected', async ({ page }) => {
    await page.goto(BASE_URL);
    await waitForOAuth(page);

    const isConnected = await page.evaluate(() => window.webclaw.oauth.isConnected('twitter'));
    console.log('twitter isConnected (should be false):', isConnected);
    expect(isConnected).toBe(false);
  });

  test('6. getConnectionStatus returns array with all 4 providers', async ({ page }) => {
    await page.goto(BASE_URL);
    await waitForOAuth(page);

    const status = await page.evaluate(async () => {
      try { return await window.webclaw.oauth.getConnectionStatus(); }
      catch (e) { return 'error: ' + e.message; }
    });

    console.log('Connection status:', JSON.stringify(status));
    expect(Array.isArray(status)).toBe(true);
    const providerNames = status.map(s => s.provider);
    for (const p of ['twitter', 'google', 'github', 'notion']) {
      expect(providerNames, 'Status should include ' + p).toContain(p);
    }
  });

});
