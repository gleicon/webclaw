/**
 * Test 08: Storage Hygiene
 * Checks IndexedDB quota via browser API
 */

import { test, expect } from '@playwright/test';

test.describe('Storage Hygiene', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: 30000 });
    await page.waitForTimeout(2000);
  });

  test('should report storage quota via browser API', async ({ page }) => {
    const quota = await page.evaluate(async () => {
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        const estimate = await navigator.storage.estimate();
        return {
          usage: estimate.usage,
          quota: estimate.quota,
          usageDetails: estimate.usageDetails,
          persisted: navigator.storage.persisted ? await navigator.storage.persisted() : null
        };
      }
      return null;
    });
    
    console.log('Storage quota:', quota);
    
    if (quota) {
      expect(quota).toHaveProperty('usage');
      expect(quota).toHaveProperty('quota');
      expect(quota.quota).toBeGreaterThan(0);
      expect(quota.usage).toBeGreaterThanOrEqual(0);
    }
  });

  test('should have access to IndexedDB', async ({ page }) => {
    const idbAvailable = await page.evaluate(() => {
      return 'indexedDB' in window;
    });
    
    expect(idbAvailable).toBe(true);
  });

  test('should list IndexedDB databases', async ({ page }) => {
    const databases = await page.evaluate(async () => {
      if ('databases' in indexedDB) {
        return await indexedDB.databases();
      }
      return [];
    });
    
    console.log('IndexedDB databases:', databases);
    
    // Should return an array (may be empty or have webclaw databases)
    expect(Array.isArray(databases)).toBe(true);
  });

  test('should handle storage persistence request', async ({ page }) => {
    const persistence = await page.evaluate(async () => {
      if ('storage' in navigator && 'persist' in navigator.storage) {
        const persisted = await navigator.storage.persist();
        return { persisted, supported: true };
      }
      return { persisted: false, supported: false };
    });
    
    console.log('Storage persistence:', persistence);
    
    // Should either support persistence or gracefully handle lack thereof
    expect(persistence).toHaveProperty('persisted');
  });

  test('should allow opening IndexedDB connections', async ({ page }) => {
    const canOpenDB = await page.evaluate(() => {
      return new Promise((resolve) => {
        const request = indexedDB.open('test-hygiene-db', 1);
        
        request.onsuccess = () => {
          const db = request.result;
          db.close();
          resolve(true);
        };
        
        request.onerror = () => resolve(false);
      });
    });
    
    expect(canOpenDB).toBe(true);
  });

  test('should clean up test databases', async ({ page }) => {
    // Create a test database
    await page.evaluate(() => {
      return new Promise((resolve) => {
        const request = indexedDB.open('cleanup-test-db', 1);
        request.onupgradeneeded = (event) => {
          const db = event.target.result;
          db.createObjectStore('test-store');
        };
        request.onsuccess = () => resolve();
      });
    });
    
    // Delete it
    const deleted = await page.evaluate(() => {
      return new Promise((resolve) => {
        const request = indexedDB.deleteDatabase('cleanup-test-db');
        request.onsuccess = () => resolve(true);
        request.onerror = () => resolve(false);
      });
    });
    
    expect(deleted).toBe(true);
  });

  test('should monitor storage growth', async ({ page }) => {
    const initialQuota = await page.evaluate(async () => {
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        return await navigator.storage.estimate();
      }
      return null;
    });
    
    // Write some data
    await page.evaluate(() => {
      return new Promise((resolve) => {
        const request = indexedDB.open('growth-test-db', 1);
        request.onupgradeneeded = (event) => {
          const db = event.target.result;
          db.createObjectStore('data');
        };
        request.onsuccess = (event) => {
          const db = event.target.result;
          const transaction = db.transaction(['data'], 'readwrite');
          const store = transaction.objectStore('data');
          
          // Write test data
          for (let i = 0; i < 100; i++) {
            store.put({ id: i, data: 'x'.repeat(1000) }, i);
          }
          
          transaction.oncomplete = () => {
            db.close();
            resolve();
          };
        };
      });
    });
    
    const finalQuota = await page.evaluate(async () => {
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        return await navigator.storage.estimate();
      }
      return null;
    });
    
    console.log('Initial storage:', initialQuota);
    console.log('Final storage:', finalQuota);
    
    if (initialQuota && finalQuota) {
      // Usage may have increased
      expect(finalQuota.usage).toBeGreaterThanOrEqual(initialQuota.usage);
    }
    
    // Cleanup
    await page.evaluate(() => {
      return new Promise((resolve) => {
        const request = indexedDB.deleteDatabase('growth-test-db');
        request.onsuccess = () => resolve();
        request.onerror = () => resolve();
      });
    });
  });
});
