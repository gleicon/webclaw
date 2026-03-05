/**
 * Helper functions for Phase 06 browser tests
 * Shared utilities for UI interaction and console capture
 */

import { expect } from '@playwright/test';

/**
 * Wait for a specific console log message
 */
export async function waitForConsoleLog(page, pattern, timeout = 10000) {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      reject(new Error(`Timeout waiting for console log matching: ${pattern}`));
    }, timeout);
    
    const handler = (msg) => {
      const text = msg.text();
      if (pattern instanceof RegExp ? pattern.test(text) : text.includes(pattern)) {
        clearTimeout(timeoutId);
        page.off('console', handler);
        resolve({ type: msg.type(), text });
      }
    };
    
    page.on('console', handler);
  });
}

/**
 * Get all console logs captured so far
 */
export async function getConsoleLogs(page) {
  const logs = [];
  
  page.on('console', msg => {
    logs.push({
      type: msg.type(),
      text: msg.text(),
      location: msg.location(),
      time: new Date().toISOString()
    });
  });
  
  return logs;
}

/**
 * Send a chat message via the UI
 */
export async function sendChatMessage(page, message) {
  // Find input and send button
  const input = page.locator('#user-input');
  const sendBtn = page.locator('#send-btn');
  
  // Ensure elements are visible
  await expect(input).toBeVisible();
  await expect(sendBtn).toBeVisible();
  
  // Type message
  await input.fill(message);
  
  // Click send
  await sendBtn.click();
  
  // Wait a moment for message to be processed
  await page.waitForTimeout(500);
}

/**
 * Wait for an element to appear
 */
export async function waitForElement(page, selector, timeout = 10000) {
  const element = page.locator(selector);
  await element.waitFor({ state: 'visible', timeout });
  return element;
}

/**
 * Check IndexedDB for specific data
 */
export async function checkIndexedDB(page, dbName, storeName, key) {
  return await page.evaluate(async ({ dbName, storeName, key }) => {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(dbName);
      
      request.onsuccess = (event) => {
        const db = event.target.result;
        
        // Check if store exists
        if (!db.objectStoreNames.contains(storeName)) {
          resolve(null);
          return;
        }
        
        const transaction = db.transaction([storeName], 'readonly');
        const store = transaction.objectStore(storeName);
        const getRequest = store.get(key);
        
        getRequest.onsuccess = () => resolve(getRequest.result);
        getRequest.onerror = () => reject(getRequest.error);
      };
      
      request.onerror = () => reject(request.error);
    });
  }, { dbName, storeName, key });
}

/**
 * Get all data from an IndexedDB store
 */
export async function getAllFromStore(page, dbName, storeName) {
  return await page.evaluate(async ({ dbName, storeName }) => {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(dbName);
      
      request.onsuccess = (event) => {
        const db = event.target.result;
        
        if (!db.objectStoreNames.contains(storeName)) {
          resolve([]);
          return;
        }
        
        const transaction = db.transaction([storeName], 'readonly');
        const store = transaction.objectStore(storeName);
        const getAllRequest = store.getAll();
        
        getAllRequest.onsuccess = () => resolve(getAllRequest.result);
        getAllRequest.onerror = () => reject(getAllRequest.error);
      };
      
      request.onerror = () => reject(request.error);
    });
  }, { dbName, storeName });
}

/**
 * Clear all IndexedDB databases (useful for cleanup)
 */
export async function clearIndexedDB(page) {
  return await page.evaluate(async () => {
    const databases = await indexedDB.databases();
    
    for (const db of databases) {
      if (db.name) {
        await new Promise((resolve) => {
          const request = indexedDB.deleteDatabase(db.name);
          request.onsuccess = () => resolve();
          request.onerror = () => resolve();
        });
      }
    }
    
    return databases.length;
  });
}

/**
 * Wait for WASM to be fully loaded and ready
 */
export async function waitForWASMReady(page, timeout = 30000) {
  await page.waitForFunction(() => {
    return window.webclaw && 
           typeof window.webclaw.jsFetch === 'function' &&
           typeof window.webclaw.jsIndexedDB === 'object';
  }, { timeout });
  
  // Additional wait for any async initialization
  await page.waitForTimeout(1000);
}

/**
 * Capture screenshot with descriptive name
 */
export async function captureScreenshot(page, name) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const filename = `${name}-${timestamp}.png`;
  await page.screenshot({ path: `test-results/${filename}` });
  return filename;
}

/**
 * Switch to a specific tab
 */
export async function switchTab(page, tabId) {
  const tabButton = page.locator(`#tab-${tabId}`);
  await tabButton.click();
  await page.waitForTimeout(500);
  
  // Verify the corresponding view is visible
  const view = page.locator(`#view-${tabId}`);
  await expect(view).toBeVisible();
}
