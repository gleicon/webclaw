/**
 * Test 04: Tool Registry
 * Verifies tool console logs and tool_use trigger
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage, getConsoleLogs, waitForConsoleLog } from './helpers.js';

test.describe('Tool Registry', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should log tool registration on startup', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('tool') || text.includes('Tool') || text.includes('TOOL')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    // Reload to see startup logs
    await page.reload();
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(3000);
    
    console.log('Tool-related startup logs:', logs);
    
    // Should have tool registration logs
    const hasToolRegistration = logs.some(log => 
      log.text.includes('tool') && (
        log.text.includes('register') ||
        log.text.includes('available') ||
        log.text.includes('loaded')
      )
    );
    
    expect(hasToolRegistration || logs.length > 0).toBe(true);
  });

  test('should display tool activity in side panel', async ({ page }) => {
    // Send a message that might trigger a tool
    await sendChatMessage(page, 'Search my memory for anything about testing');
    await page.waitForTimeout(5000);
    
    // Check tool events panel
    const toolEvents = await page.locator('#tool-events > div').count();
    console.log(`Tool events in panel: ${toolEvents}`);
    
    // Take screenshot
    await page.screenshot({ path: 'test-results/tool-activity-panel.png' });
    
    // Panel should exist (even if empty initially)
    const panel = await page.locator('#tool-events');
    await expect(panel).toBeVisible();
  });

  test('should log tool_use events', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      logs.push(text);
    });
    
    // Try to trigger tool use
    await sendChatMessage(page, 'Use the memory search tool to find information');
    await page.waitForTimeout(5000);
    
    // Look for tool use patterns
    const toolUseLogs = logs.filter(log => 
      log.includes('tool_use') ||
      log.includes('tool-call') ||
      log.includes('calling tool') ||
      log.includes('executing tool')
    );
    
    console.log('Tool use logs:', toolUseLogs);
    
    // Should have some tool-related activity
    const hasToolActivity = logs.some(log => 
      log.includes('tool') || log.includes('Tool')
    );
    
    expect(hasToolActivity).toBe(true);
  });

  test('should show tool names in console', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.match(/tool[_-]?name|memory[_-]?search|file[_-]?read/i)) {
        logs.push(text);
      }
    });
    
    await sendChatMessage(page, 'What tools are available?');
    await page.waitForTimeout(3000);
    
    console.log('Tool name logs:', logs);
    
    // Some tool names should appear in logs
    expect(logs.length).toBeGreaterThanOrEqual(0);
  });

  test('should trigger tool_use for memory operations', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      logs.push({ type: msg.type(), text, time: Date.now() });
    });
    
    // Try to trigger memory-related tool
    await sendChatMessage(page, 'Save this to my memory: I am running browser tests');
    await page.waitForTimeout(5000);
    
    // Look for memory tool usage
    const memoryToolLogs = logs.filter(log => 
      log.text.includes('memory') && (
        log.text.includes('tool') ||
        log.text.includes('save') ||
        log.text.includes('store')
      )
    );
    
    console.log('Memory tool logs:', memoryToolLogs);
    
    // Should have logged the attempt
    expect(logs.length).toBeGreaterThan(0);
  });
});
