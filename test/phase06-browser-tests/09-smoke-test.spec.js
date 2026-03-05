/**
 * Test 09: Smoke Test
 * Verifies startup console logs (all components ready)
 */

import { test, expect } from '@playwright/test';

test.describe('Smoke Test', () => {
  test('should show all components ready on startup', async ({ page }) => {
    const logs = [];
    const errors = [];
    
    page.on('console', msg => {
      logs.push({ 
        type: msg.type(), 
        text: msg.text(),
        time: new Date().toISOString()
      });
    });
    
    page.on('pageerror', err => {
      errors.push(err.message);
    });
    
    // Navigate and wait for load
    await page.goto('/');
    
    // Wait for WASM
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    
    // Wait for app initialization
    await page.waitForTimeout(3000);
    
    console.log('=== SMOKE TEST LOGS ===');
    logs.forEach(log => console.log(`[${log.type.toUpperCase()}] ${log.text}`));
    console.log('=======================');
    
    // Check for critical components
    const hasWASMReady = logs.some(l => l.text.includes('WASM') || l.text.includes('wasm'));
    const hasBridges = logs.some(l => l.text.includes('bridge') || l.text.includes('jsFetch'));
    const hasUI = await page.locator('#messages').isVisible();
    
    expect(hasWASMReady).toBe(true);
    expect(hasBridges).toBe(true);
    expect(hasUI).toBe(true);
    expect(errors.length).toBe(0);
  });

  test('should have all UI components visible', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(3000);
    
    // Check all major UI elements
    const elements = {
      messages: '#messages',
      input: '#user-input',
      sendBtn: '#send-btn',
      modelSelector: '#model-selector',
      toolEvents: '#tool-events',
      tabChat: '#tab-chat',
      tabSettings: '#tab-settings',
      tabIdentity: '#tab-identity'
    };
    
    for (const [name, selector] of Object.entries(elements)) {
      const visible = await page.locator(selector).isVisible().catch(() => false);
      console.log(`${name}: ${visible ? 'visible' : 'NOT visible'}`);
      expect(visible, `${name} should be visible`).toBe(true);
    }
  });

  test('should have working tab navigation', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(3000);
    
    // Test Settings tab
    await page.click('#tab-settings');
    await page.waitForTimeout(500);
    const settingsVisible = await page.locator('#view-settings').isVisible();
    expect(settingsVisible).toBe(true);
    
    // Test Identity tab
    await page.click('#tab-identity');
    await page.waitForTimeout(500);
    const identityVisible = await page.locator('#view-identity').isVisible();
    expect(identityVisible).toBe(true);
    
    // Back to Chat
    await page.click('#tab-chat');
    await page.waitForTimeout(500);
    const chatVisible = await page.locator('#view-chat').isVisible();
    expect(chatVisible).toBe(true);
  });

  test('should load without JavaScript errors', async ({ page }) => {
    const jsErrors = [];
    
    page.on('pageerror', error => {
      jsErrors.push(error.message);
    });
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        jsErrors.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForTimeout(5000);
    
    console.log('JavaScript errors:', jsErrors);
    
    expect(jsErrors.filter(e => !e.includes('favicon')).length).toBe(0);
  });

  test('should have WASM binary loaded', async ({ page }) => {
    await page.goto('/');
    
    const wasmLoaded = await page.waitForFunction(() => {
      return window.webclaw && 
             typeof window.webclaw.jsFetch === 'function' &&
             typeof window.webclaw.jsIndexedDB === 'object';
    }, { timeout: 30000 });
    
    expect(wasmLoaded).toBeTruthy();
  });
});
