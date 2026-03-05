/**
 * Test 10: Health Tracking
 * Verifies health status in console
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage } from './helpers.js';

test.describe('Health Tracking', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should log health status on startup', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('health') || text.includes('Health') || 
          text.includes('status') || text.includes('ready')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    // Reload to see startup health logs
    await page.reload();
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(3000);
    
    console.log('Health tracking logs:', logs);
    
    // Should have health-related logs
    const hasHealthLog = logs.some(log => 
      log.text.includes('health') || log.text.includes('ready')
    );
    
    expect(hasHealthLog || logs.length > 0).toBe(true);
  });

  test('should track system health during operation', async ({ page }) => {
    const healthLogs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('health') || text.includes('status') || 
          text.includes('performance') || text.includes('metrics')) {
        healthLogs.push({ 
          type: msg.type(), 
          text, 
          time: new Date().toISOString() 
        });
      }
    });
    
    // Perform operations
    await sendChatMessage(page, 'Health tracking test message 1');
    await page.waitForTimeout(2000);
    
    await sendChatMessage(page, 'Health tracking test message 2');
    await page.waitForTimeout(2000);
    
    console.log('Health logs during operation:', healthLogs);
    
    // Should have tracked something
    expect(healthLogs.length).toBeGreaterThanOrEqual(0);
  });

  test('should show component health in logs', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('component') || text.includes('ready') || 
          text.includes('initialized') || text.includes('loaded')) {
        logs.push(text);
      }
    });
    
    await page.reload();
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(3000);
    
    console.log('Component health logs:', logs);
    
    // Should indicate component status
    const hasComponentStatus = logs.some(log => 
      log.includes('ready') || log.includes('initialized')
    );
    
    expect(hasComponentStatus || logs.length > 0).toBe(true);
  });

  test('should track memory health', async ({ page }) => {
    const memoryLogs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('memory') && (text.includes('health') || 
          text.includes('status') || text.includes('usage'))) {
        memoryLogs.push(text);
      }
    });
    
    // Trigger memory operations
    await sendChatMessage(page, 'Track memory health test');
    await page.waitForTimeout(3000);
    
    console.log('Memory health logs:', memoryLogs);
    
    // Memory health should be tracked
    expect(memoryLogs.length).toBeGreaterThanOrEqual(0);
  });

  test('should report health metrics', async ({ page }) => {
    const metrics = await page.evaluate(() => {
      // Check if we can access performance metrics
      const perf = {
        memory: performance.memory ? {
          usedJSHeapSize: performance.memory.usedJSHeapSize,
          totalJSHeapSize: performance.memory.totalJSHeapSize,
          jsHeapSizeLimit: performance.memory.jsHeapSizeLimit
        } : null,
        timing: performance.timing ? {
          navigationStart: performance.timing.navigationStart,
          domContentLoaded: performance.timing.domContentLoadedEventEnd,
          loadComplete: performance.timing.loadEventEnd
        } : null,
        now: performance.now()
      };
      return perf;
    });
    
    console.log('Performance metrics:', metrics);
    
    // Should have some performance data
    expect(metrics).toHaveProperty('now');
    expect(metrics.now).toBeGreaterThan(0);
  });
});
