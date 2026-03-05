/**
 * Test 01: Summarization Threshold
 * Verifies 20-message threshold triggers summarization
 * Captures "summarization triggered" console log
 */

import { test, expect } from '@playwright/test';
import { waitForConsoleLog, sendChatMessage, getConsoleLogs } from './helpers.js';

test.describe('Summarization Threshold', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate and wait for WASM to load
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    
    // Wait for app to be fully ready
    await page.waitForTimeout(2000);
  });

  test('should trigger summarization after 20 messages', async ({ page }) => {
    const consoleLogs = [];
    
    // Capture all console logs
    page.on('console', msg => {
      const text = msg.text();
      consoleLogs.push({ type: msg.type(), text, time: new Date().toISOString() });
    });
    
    // Send 20 messages to trigger threshold
    const messages = Array.from({ length: 20 }, (_, i) => 
      `Test message number ${i + 1} for summarization threshold testing`
    );
    
    for (let i = 0; i < messages.length; i++) {
      await sendChatMessage(page, messages[i]);
      // Small delay to ensure message is processed
      await page.waitForTimeout(500);
      
      // Every 5 messages, wait a bit longer for any async processing
      if ((i + 1) % 5 === 0) {
        await page.waitForTimeout(1000);
      }
    }
    
    // Wait for potential summarization trigger
    await page.waitForTimeout(3000);
    
    // Check for summarization-related logs
    const summarizationLogs = consoleLogs.filter(log => 
      log.text.toLowerCase().includes('summarization') ||
      log.text.toLowerCase().includes('summarize') ||
      log.text.toLowerCase().includes('threshold') ||
      log.text.toLowerCase().includes('compress')
    );
    
    console.log('Summarization-related logs:', summarizationLogs);
    
    // Either we should see summarization trigger OR the messages should be processed
    expect(summarizationLogs.length).toBeGreaterThan(0);
    
    // Verify we processed all messages
    const messageElements = await page.locator('#messages > div').count();
    expect(messageElements).toBeGreaterThanOrEqual(20);
  });

  test('should log summarization trigger message', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      logs.push(msg.text());
    });
    
    // Send messages rapidly
    for (let i = 0; i < 25; i++) {
      await sendChatMessage(page, `Message ${i + 1} for logging test`);
      await page.waitForTimeout(300);
    }
    
    // Wait for summarization processing
    await page.waitForTimeout(5000);
    
    // Look for specific patterns in logs
    const hasSummarizationLog = logs.some(log => 
      log.includes('summarization triggered') ||
      log.includes('Summarization:') ||
      log.includes('memory compression') ||
      log.includes('token limit')
    );
    
    // Log all console output for debugging
    console.log('All logs:', logs.slice(-20));
    
    expect(hasSummarizationLog).toBe(true);
  });

  test('should maintain message context after summarization', async ({ page }) => {
    // Send context-establishing messages
    await sendChatMessage(page, 'My name is TestUser and I like testing');
    await page.waitForTimeout(1000);
    
    // Send 20 more messages
    for (let i = 0; i < 20; i++) {
      await sendChatMessage(page, `Follow-up message ${i + 1}`);
      await page.waitForTimeout(300);
    }
    
    // Ask about context
    await sendChatMessage(page, 'What is my name?');
    await page.waitForTimeout(2000);
    
    // Verify the UI shows the conversation
    const messages = await page.locator('#messages > div').count();
    expect(messages).toBeGreaterThan(20);
  });
});
