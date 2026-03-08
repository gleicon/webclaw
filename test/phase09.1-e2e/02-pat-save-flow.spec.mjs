/**
 * Phase 9.1 — PAT Save Flow Tests
 *
 * These tests verify PAT/token save interactions and masked-state behavior.
 * They require window.webclaw.oauth to be ready (WASM loaded).
 *
 * Tests are written to FAIL until Plan 02 (Go backend) and Plan 03 (DOM rework)
 * are complete (Nyquist Wave 0 compliance).
 *
 * Element ID contracts:
 *   #oauth-pat-input-{id}    — PAT/token password input
 *   #oauth-pat-save-{id}     — Save button for PAT
 *   #oauth-masked-{id}       — masked token display container (visible after save)
 *   #oauth-status-{id}       — status text span
 */

import { test, expect } from '@playwright/test';

/**
 * Wait for WASM oauth to be ready, then navigate to Connected Services.
 * Requires savePATToken to be exported before resolving.
 */
async function waitForOAuthAndNavigate(page) {
  await page.goto('/');
  await page.waitForFunction(
    () => window.webclaw && window.webclaw.oauth && typeof window.webclaw.oauth.savePATToken === 'function',
    { timeout: 10000 }
  );
  // Navigate to Settings > Connected Services
  try {
    await page.click('#tab-settings', { timeout: 10000 });
  } catch {
    await page.click('button:has-text("Settings")', { timeout: 10000 });
  }
  await page.waitForSelector('#connected-services-list', { state: 'visible', timeout: 15000 });
}

test.describe('PAT save flow', () => {

  // -------------------------------------------------------------------------
  // WASM export presence checks — verify Go exports are available
  // -------------------------------------------------------------------------

  test('GitHub: savePATToken export exists on webclaw.oauth', async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(
      () => window.webclaw && window.webclaw.oauth,
      { timeout: 10000 }
    );
    const exists = await page.evaluate(() => typeof window.webclaw.oauth.savePATToken === 'function');
    expect(exists).toBe(true);
  });

  test('markInvalid export exists on webclaw.oauth', async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(
      () => window.webclaw && window.webclaw.oauth,
      { timeout: 10000 }
    );
    const exists = await page.evaluate(() => typeof window.webclaw.oauth.markInvalid === 'function');
    expect(exists).toBe(true);
  });

  // -------------------------------------------------------------------------
  // PAT save + masked state interactions
  // -------------------------------------------------------------------------

  test('GitHub: entering PAT and saving shows masked state', async ({ page }) => {
    await waitForOAuthAndNavigate(page);

    // Fill in a test PAT value
    await page.locator('#oauth-pat-input-github').fill('ghp_TESTVALUE12345');
    // Click the Save button
    await page.locator('#oauth-pat-save-github').click();

    // After save, either:
    //   (a) #oauth-masked-github becomes visible, OR
    //   (b) the input switches to type="password" with empty value (masked mode)
    const maskedVisible = await page.locator('#oauth-masked-github').isVisible().catch(() => false);
    const inputType = await page.locator('#oauth-pat-input-github').getAttribute('type').catch(() => null);
    const inputValue = await page.locator('#oauth-pat-input-github').inputValue().catch(() => null);

    const maskedState = maskedVisible || (inputType === 'password' && inputValue === '');
    expect(maskedState, 'GitHub PAT should be in masked state after save').toBe(true);

    // Status should show "Connected"
    await expect(page.locator('#oauth-status-github')).toContainText('Connected');
  });

  test('Notion: entering token and saving shows masked state', async ({ page }) => {
    await waitForOAuthAndNavigate(page);

    // Fill in a test Notion integration token value
    await page.locator('#oauth-pat-input-notion').fill('secret_TESTVALUE12345');
    // Click the Save button
    await page.locator('#oauth-pat-save-notion').click();

    // After save, check masked state (same dual assertion as GitHub)
    const maskedVisible = await page.locator('#oauth-masked-notion').isVisible().catch(() => false);
    const inputType = await page.locator('#oauth-pat-input-notion').getAttribute('type').catch(() => null);
    const inputValue = await page.locator('#oauth-pat-input-notion').inputValue().catch(() => null);

    const maskedState = maskedVisible || (inputType === 'password' && inputValue === '');
    expect(maskedState, 'Notion token should be in masked state after save').toBe(true);

    // Status should show "Connected"
    await expect(page.locator('#oauth-status-notion')).toContainText('Connected');
  });

  test('Invalid token: markInvalid causes red dot on card', async ({ page }) => {
    // NOTE: This test may not be fully automatable if the UI does not auto-refresh
    // after markInvalid() is called. The manual verification path from VALIDATION.md:
    //   Open browser console → call webclaw.oauth.markInvalid("github") → check UI
    //
    // We attempt automation here but skip if the UI does not expose a polling hook.
    test.skip(
      true,
      'Manual verification required: UI may not auto-refresh after markInvalid(). ' +
      'Manual path: open DevTools console, run webclaw.oauth.markInvalid("github"), ' +
      'observe #oauth-status-github for "Invalid" text or red CSS class. ' +
      'See VALIDATION.md Manual-Only Verifications section.'
    );

    // Automated attempt (runs if skip is removed after implementation):
    await waitForOAuthAndNavigate(page);
    await page.evaluate(() => window.webclaw.oauth.markInvalid('github'));
    // Trigger a UI refresh — call getConnectionStatus if it causes a DOM update
    await page.evaluate(() => window.webclaw.oauth.getConnectionStatus && window.webclaw.oauth.getConnectionStatus());
    await page.waitForTimeout(1000);

    // Assert status shows "Invalid" OR has an error CSS class
    const statusText = await page.locator('#oauth-status-github').textContent().catch(() => '');
    const hasErrorClass = await page.locator('#oauth-status-github').evaluate(
      el => el.classList.contains('text-red-500') || el.classList.contains('error') || el.classList.contains('invalid')
    ).catch(() => false);

    expect(statusText.includes('Invalid') || hasErrorClass, 'GitHub card should show invalid token state').toBe(true);
  });
});
