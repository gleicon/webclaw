/**
 * Phase 9.1 — Connected Services Card Structure (Smoke Tests)
 *
 * These tests assert specific DOM element IDs, text content, and card structure
 * that Plans 02 and 03 will create. They are written to FAIL until implementation
 * is complete (Nyquist Wave 0 compliance).
 *
 * Tests do NOT wait for window.webclaw — they test DOM structure only.
 * Navigate to Settings > Connected Services and assert element presence/content.
 *
 * Element ID contracts (from Plan 01 interfaces):
 *   #connected-services-list        — container for all provider cards
 *   #oauth-provider-{id}            — provider card wrapper row
 *   #oauth-badge-{id}               — auth-type badge per card
 *   #oauth-pat-input-{id}           — PAT/token password input (github, notion)
 *   #oauth-pat-save-{id}            — Save button for PAT
 *   #oauth-pat-toggle-{id}          — Show/hide toggle for PAT input
 *   #oauth-pat-update-{id}          — Update token button (shown when masked)
 *   #oauth-redirect-callout-{id}    — redirect URI callout (twitter, google)
 *   #oauth-masked-{id}              — masked token display container
 *   #oauth-clientid-{id}            — client ID input (twitter, google)
 *   #oauth-btn-{id}                 — connect/disconnect button
 *   #oauth-status-{id}              — status text span
 */

import { test, expect } from '@playwright/test';

/**
 * Navigate to Settings > Connected Services section.
 * Waits for #connected-services-list to be visible.
 */
async function navigateToConnectedServices(page) {
  await page.goto('/');
  // Click the Settings tab — try data-tab attribute first, then id, then text
  try {
    await page.click('#tab-settings', { timeout: 10000 });
  } catch {
    await page.click('button:has-text("Settings")', { timeout: 10000 });
  }
  // Wait for the connected-services list container to appear
  await page.waitForSelector('#connected-services-list', { state: 'visible', timeout: 15000 });
}

test.describe('Connected Services card structure (smoke)', () => {
  test.beforeEach(async ({ page }) => {
    await navigateToConnectedServices(page);
  });

  // -------------------------------------------------------------------------
  // PAT-flow providers: GitHub and Notion use Access Token / PAT inputs
  // -------------------------------------------------------------------------

  test('GitHub card: PAT input present, no Connect button', async ({ page }) => {
    // PAT input must be visible for GitHub
    await expect(page.locator('#oauth-pat-input-github')).toBeVisible();
    // GitHub should NOT have an OAuth Connect button (it uses PAT, not PKCE flow)
    await expect(page.locator('#oauth-btn-github')).toHaveCount(0);
  });

  test('Notion card: token input present, no Connect button', async ({ page }) => {
    // Token input must be visible for Notion
    await expect(page.locator('#oauth-pat-input-notion')).toBeVisible();
    // Notion should NOT have an OAuth Connect button (it uses API token)
    await expect(page.locator('#oauth-btn-notion')).toHaveCount(0);
  });

  // -------------------------------------------------------------------------
  // PKCE-flow providers: Twitter and Google use Client ID + redirect callout
  // -------------------------------------------------------------------------

  test('Twitter card: Client ID field and redirect URI callout visible', async ({ page }) => {
    await expect(page.locator('#oauth-clientid-twitter')).toBeVisible();
    await expect(page.locator('#oauth-redirect-callout-twitter')).toBeVisible();
    // Callout must contain the PKCE redirect URI
    await expect(page.locator('#oauth-redirect-callout-twitter')).toContainText('about:blank');
  });

  test('Google card: Client ID field and redirect URI callout visible', async ({ page }) => {
    await expect(page.locator('#oauth-clientid-google')).toBeVisible();
    await expect(page.locator('#oauth-redirect-callout-google')).toBeVisible();
    // Callout must contain the PKCE redirect URI
    await expect(page.locator('#oauth-redirect-callout-google')).toContainText('about:blank');
  });

  // -------------------------------------------------------------------------
  // Auth-type badges — each card must declare its authentication method
  // -------------------------------------------------------------------------

  test('Twitter card: auth-type badge shows OAuth 2.0', async ({ page }) => {
    await expect(page.locator('#oauth-badge-twitter')).toBeVisible();
    await expect(page.locator('#oauth-badge-twitter')).toHaveText('OAuth 2.0');
  });

  test('Google card: auth-type badge shows OAuth 2.0', async ({ page }) => {
    await expect(page.locator('#oauth-badge-google')).toBeVisible();
    await expect(page.locator('#oauth-badge-google')).toHaveText('OAuth 2.0');
  });

  test('GitHub card: auth-type badge shows Access Token', async ({ page }) => {
    await expect(page.locator('#oauth-badge-github')).toBeVisible();
    await expect(page.locator('#oauth-badge-github')).toHaveText('Access Token');
  });

  test('Notion card: auth-type badge shows Access Token', async ({ page }) => {
    await expect(page.locator('#oauth-badge-notion')).toBeVisible();
    await expect(page.locator('#oauth-badge-notion')).toHaveText('Access Token');
  });
});
