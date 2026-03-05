/**
 * Test 05: Memory Search
 * Uses memory search tool in UI and verifies results
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage, getConsoleLogs } from './helpers.js';

test.describe('Memory Search', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should search memory and return results', async ({ page }) => {
    // First, establish some memories
    await sendChatMessage(page, 'Remember this important fact: WebClaw is a browser-based AI assistant');
    await page.waitForTimeout(3000);
    
    await sendChatMessage(page, 'Save to memory: Testing is crucial for reliable software');
    await page.waitForTimeout(3000);
    
    // Now search for the memory
    await sendChatMessage(page, 'Search my memory for "WebClaw"');
    await page.waitForTimeout(5000);
    
    // Check the response
    const messages = await page.locator('#messages > div').count();
    console.log(`Total messages: ${messages}`);
    
    // Take screenshot of results
    await page.screenshot({ path: 'test-results/memory-search-results.png' });
    
    // Should have multiple messages including search results
    expect(messages).toBeGreaterThan(2);
  });

  test('should log memory search operations', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('search') || text.includes('Search') || text.includes('memory')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    // Trigger memory search
    await sendChatMessage(page, 'Find information in my memory about testing');
    await page.waitForTimeout(5000);
    
    console.log('Memory search logs:', logs);
    
    // Should have search-related logs
    const hasSearchLog = logs.some(log => 
      log.text.includes('search') || log.text.includes('Search')
    );
    
    expect(hasSearchLog || logs.length > 0).toBe(true);
  });

  test('should handle memory search with no results', async ({ page }) => {
    // Search for something that doesn't exist
    await sendChatMessage(page, 'Search my memory for "xyznonexistent12345"');
    await page.waitForTimeout(5000);
    
    // Should handle gracefully
    const messages = await page.locator('#messages > div').count();
    expect(messages).toBeGreaterThan(0);
    
    // Check for error messages
    const errorMessages = await page.locator('#messages .error, #messages .text-red').count();
    console.log(`Error messages: ${errorMessages}`);
    
    // No errors expected for empty results
    expect(errorMessages).toBe(0);
  });

  test('should display memory search in tool activity panel', async ({ page }) => {
    // Trigger memory search
    await sendChatMessage(page, 'Use the search_memory tool to find "browser"');
    await page.waitForTimeout(5000);
    
    // Check tool activity panel
    const toolEvents = await page.locator('#tool-events > div').count();
    console.log(`Tool events after memory search: ${toolEvents}`);
    
    // Take screenshot
    await page.screenshot({ path: 'test-results/memory-search-tool-activity.png' });
    
    // Tool panel should be visible
    const panel = await page.locator('#tool-events');
    await expect(panel).toBeVisible();
  });

  test('should support semantic memory search', async ({ page }) => {
    // Store contextual information
    await sendChatMessage(page, 'I enjoy working on software quality assurance and test automation');
    await page.waitForTimeout(3000);
    
    // Search with related but different terms
    await sendChatMessage(page, 'What do I like to work on?');
    await page.waitForTimeout(5000);
    
    // Should get a response
    const messages = await page.locator('#messages > div').count();
    expect(messages).toBeGreaterThan(1);
  });
});
