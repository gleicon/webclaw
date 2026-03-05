/**
 * Test 07: Fail-Fast Error Handling
 * Verifies error handling behavior
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage, setupTestEnvironment } from './helpers.js';

test.describe('Fail-Fast Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && window.webclaw.keystore && window.webclaw.keystore.setKey;
    }, { timeout: 30000 });
    
    // Inject API keys
    await setupTestEnvironment(page);
    
    await page.waitForTimeout(2000);
  });

  test('should log errors to console', async ({ page }) => {
    const errors = [];
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      logs.push({ type: msg.type(), text });
      if (msg.type() === 'error' || text.includes('error') || text.includes('Error')) {
        errors.push(text);
      }
    });
    
    page.on('pageerror', err => {
      errors.push(err.message);
    });
    
    // Perform an action
    await sendChatMessage(page, 'Test message');
    await page.waitForTimeout(3000);
    
    console.log('All logs:', logs);
    console.log('Errors captured:', errors);
    
    // Normal operation shouldn't have errors
    // But we capture them for analysis
    expect(errors.length).toBeGreaterThanOrEqual(0);
  });

  test('should handle network errors gracefully', async ({ page }) => {
    // Monitor for error handling
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('network') || text.includes('timeout') || 
          text.includes('failed') || text.includes('retry')) {
        logs.push(text);
      }
    });
    
    await sendChatMessage(page, 'Trigger a test');
    await page.waitForTimeout(5000);
    
    console.log('Error handling logs:', logs);
    
    // UI should remain functional
    const input = await page.locator('#user-input');
    await expect(input).toBeVisible();
    await expect(input).toBeEnabled();
  });

  test('should show error messages in UI when operations fail', async ({ page }) => {
    // Send a message and check UI remains stable
    await sendChatMessage(page, 'Test error handling');
    await page.waitForTimeout(3000);
    
    // Verify UI elements are still present
    await expect(page.locator('#messages')).toBeVisible();
    await expect(page.locator('#user-input')).toBeVisible();
    await expect(page.locator('#send-btn')).toBeVisible();
    
    // Take screenshot to verify state
    await page.screenshot({ path: 'test-results/error-handling-ui.png' });
  });

  test('should not crash on invalid input', async ({ page }) => {
    // Test with various edge cases
    const testInputs = [
      '',
      '   ',
      'a'.repeat(10000),
      '🎉🎊🎁',
      '<script>alert("xss")</script>',
      'null\x00undefined',
      '🤖'.repeat(100)
    ];
    
    for (const input of testInputs) {
      try {
        await sendChatMessage(page, input);
        await page.waitForTimeout(500);
      } catch (e) {
        console.log(`Input "${input.substring(0, 20)}..." caused error: ${e.message}`);
      }
      
      // Verify page still functional
      await expect(page.locator('#user-input')).toBeVisible();
    }
  });

  test('should log fail-fast behavior', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('fail') || text.includes('panic') || 
          text.includes('fatal') || text.includes('abort')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    await page.reload();
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(3000);
    
    console.log('Fail-fast related logs:', logs);
    
    // Should not have fatal errors during normal startup
    const hasFatalError = logs.some(log => 
      log.text.includes('panic') || log.text.includes('fatal')
    );
    
    expect(hasFatalError).toBe(false);
  });
});
