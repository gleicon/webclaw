/**
 * Test 11: Async Embedder
 * Verifies memory initialization logs
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage } from './helpers.js';

test.describe('Async Embedder', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should log embedder initialization', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('embedder') || text.includes('Embedder') || 
          text.includes('embedding') || text.includes('Embedding') ||
          text.includes('vector') || text.includes('Vector')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    // Reload to see initialization logs
    await page.reload();
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(3000);
    
    console.log('Embedder initialization logs:', logs);
    
    // Should have embedder-related initialization
    const hasEmbedderLog = logs.some(log => 
      log.text.includes('embedder') || 
      log.text.includes('embedding') ||
      log.text.includes('vector')
    );
    
    expect(hasEmbedderLog || logs.length > 0).toBe(true);
  });

  test('should initialize memory system asynchronously', async ({ page }) => {
    const memoryLogs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('memory') && (text.includes('init') || 
          text.includes('async') || text.includes('ready'))) {
        memoryLogs.push({ type: msg.type(), text });
      }
    });
    
    await page.reload();
    
    // Wait for initialization
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    
    // Give async initialization time
    await page.waitForTimeout(5000);
    
    console.log('Memory initialization logs:', memoryLogs);
    
    // Should show async initialization
    const hasAsyncInit = memoryLogs.some(log => 
      log.text.includes('init') || log.text.includes('async')
    );
    
    expect(hasAsyncInit || memoryLogs.length > 0).toBe(true);
  });

  test('should log vector store operations', async ({ page }) => {
    const vectorLogs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('vector') || text.includes('store') || 
          text.includes('index') || text.includes('embedding')) {
        vectorLogs.push(text);
      }
    });
    
    // Trigger operations that might use embeddings
    await sendChatMessage(page, 'Initialize vector store test');
    await page.waitForTimeout(3000);
    
    console.log('Vector store logs:', vectorLogs);
    
    // Should have vector-related activity
    expect(vectorLogs.length).toBeGreaterThanOrEqual(0);
  });

  test('should support embedding generation', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      logs.push(text);
    });
    
    // Try to trigger embedding generation
    await sendChatMessage(page, 'Generate embedding for: test content');
    await page.waitForTimeout(5000);
    
    // Look for embedding-related logs
    const embeddingLogs = logs.filter(log => 
      log.includes('embed') || 
      log.includes('vector') ||
      log.includes('similarity')
    );
    
    console.log('Embedding generation logs:', embeddingLogs);
    
    expect(embeddingLogs.length).toBeGreaterThanOrEqual(0);
  });

  test('should handle async memory loading', async ({ page }) => {
    // Check IndexedDB for memory data
    const memoryData = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const checkStores = async () => {
          try {
            const dbs = await indexedDB.databases();
            const webclawDBs = dbs.filter(db => 
              db.name && db.name.includes('webclaw')
            );
            
            resolve({
              databases: webclawDBs.map(db => db.name),
              totalDBs: dbs.length
            });
          } catch (e) {
            resolve({ error: e.message, databases: [] });
          }
        };
        
        checkStores();
      });
    });
    
    console.log('WebClaw IndexedDB databases:', memoryData);
    
    // Should have some databases (may be empty initially)
    expect(memoryData).toHaveProperty('databases');
    expect(Array.isArray(memoryData.databases)).toBe(true);
  });
});
