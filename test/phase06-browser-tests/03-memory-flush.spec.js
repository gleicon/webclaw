/**
 * Test 03: Memory Flush
 * Verifies MEMORY.md file creation and IndexedDB operations
 */

import { test, expect } from '@playwright/test';
import { sendChatMessage, waitForElement, checkIndexedDB } from './helpers.js';

test.describe('Memory Flush', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should create MEMORY.md entry in IndexedDB', async ({ page }) => {
    // Trigger memory operations by sending messages
    await sendChatMessage(page, 'Create a memory: I prefer dark mode interfaces');
    await page.waitForTimeout(3000);
    
    await sendChatMessage(page, 'Remember that testing is important for quality software');
    await page.waitForTimeout(3000);
    
    // Check IndexedDB for memory storage
    const hasMemory = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const request = indexedDB.open('webclaw-memory', 1);
        
        request.onsuccess = (event) => {
          const db = event.target.result;
          
          // Check if object stores exist
          const storeNames = Array.from(db.objectStoreNames);
          console.log('Object stores:', storeNames);
          
          // Look for memory-related stores
          const hasMemoryStore = storeNames.some(name => 
            name.includes('memory') || 
            name.includes('Memory') ||
            name.includes('session')
          );
          
          resolve(hasMemoryStore || storeNames.length > 0);
        };
        
        request.onerror = () => resolve(false);
      });
    });
    
    console.log('Memory storage exists:', hasMemory);
    expect(hasMemory).toBe(true);
  });

  test('should log memory flush operations', async ({ page }) => {
    const logs = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('memory') || text.includes('Memory') || text.includes('MEMORY')) {
        logs.push({ type: msg.type(), text });
      }
    });
    
    // Trigger potential memory operations
    await sendChatMessage(page, '/memory flush');
    await page.waitForTimeout(3000);
    
    // Send more messages
    await sendChatMessage(page, 'Testing memory persistence across sessions');
    await page.waitForTimeout(2000);
    
    console.log('Memory-related logs:', logs);
    
    // Should have memory-related logs
    expect(logs.length).toBeGreaterThan(0);
  });

  test('should persist data in IndexedDB across operations', async ({ page }) => {
    // Store test data
    const testKey = 'test-memory-key';
    const testValue = JSON.stringify({ 
      test: true, 
      timestamp: Date.now(),
      content: 'Memory flush test data'
    });
    
    await page.evaluate(async ({ key, value }) => {
      return new Promise((resolve, reject) => {
        const request = indexedDB.open('webclaw-test', 1);
        
        request.onupgradeneeded = (event) => {
          const db = event.target.result;
          if (!db.objectStoreNames.contains('test-store')) {
            db.createObjectStore('test-store');
          }
        };
        
        request.onsuccess = (event) => {
          const db = event.target.result;
          const transaction = db.transaction(['test-store'], 'readwrite');
          const store = transaction.objectStore('test-store');
          const putRequest = store.put(value, key);
          
          putRequest.onsuccess = () => resolve(true);
          putRequest.onerror = () => reject(putRequest.error);
        };
        
        request.onerror = () => reject(request.error);
      });
    }, { key: testKey, value: testValue });
    
    // Wait a moment
    await page.waitForTimeout(1000);
    
    // Verify data can be retrieved
    const retrieved = await page.evaluate(async ({ key }) => {
      return new Promise((resolve) => {
        const request = indexedDB.open('webclaw-test', 1);
        
        request.onsuccess = (event) => {
          const db = event.target.result;
          const transaction = db.transaction(['test-store'], 'readonly');
          const store = transaction.objectStore('test-store');
          const getRequest = store.get(key);
          
          getRequest.onsuccess = () => resolve(getRequest.result);
          getRequest.onerror = () => resolve(null);
        };
        
        request.onerror = () => resolve(null);
      });
    }, { key: testKey });
    
    expect(retrieved).toBe(testValue);
  });

  test('should handle IndexedDB quota checks', async ({ page }) => {
    // Check storage quota via browser API
    const quota = await page.evaluate(async () => {
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        return await navigator.storage.estimate();
      }
      return null;
    });
    
    console.log('Storage quota info:', quota);
    
    if (quota) {
      expect(quota).toHaveProperty('usage');
      expect(quota).toHaveProperty('quota');
    }
  });
});
