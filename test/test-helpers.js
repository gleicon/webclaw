/**
 * Test Helpers for injecting API keys and setting up the test environment
 */

import { readFileSync, existsSync } from 'fs';
import { join } from 'path';

/**
 * Load API keys from .env.test file
 */
export function loadTestAPIKeys() {
  const envPath = join(process.cwd(), '..', '.env.test');
  if (!existsSync(envPath)) {
    console.warn('⚠️  .env.test not found - tests may fail without API keys');
    return {};
  }
  
  const env = {};
  const content = readFileSync(envPath, 'utf-8');
  const lines = content.split('\n');
  
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    
    const match = trimmed.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (match) {
      const [, key, value] = match;
      env[key] = value;
    }
  }
  
  return env;
}

/**
 * Inject API keys into browser's IndexedDB keystore using the WASM bridge
 * Must be called via page.evaluate()
 */
export async function injectAPIKeys(page, keys) {
  await page.evaluate(async (apiKeys) => {
    // Wait for WASM and keystore bridge to be ready
    let attempts = 0;
    while (!window.webclaw || !window.webclaw.keystore || !window.webclaw.keystore.setKey) {
      attempts++;
      if (attempts > 50) {
        throw new Error('webclaw.keystore bridge not available after 5s');
      }
      await new Promise(r => setTimeout(r, 100));
    }
    
    // Inject each key
    for (const [provider, key] of Object.entries(apiKeys)) {
      if (key && key.startsWith('sk-')) {
        try {
          await window.webclaw.keystore.setKey(provider, key);
          console.log(`[test] Injected API key for ${provider}`);
        } catch (err) {
          console.error(`[test] Failed to inject key for ${provider}:`, err);
        }
      }
    }
    
    return true;
  }, keys);
}

/**
 * Setup test environment - inject API keys before tests
 */
export async function setupTestEnvironment(page) {
  const keys = loadTestAPIKeys();
  
  if (keys.ANTHROPIC_API_KEY) {
    await injectAPIKeys(page, {
      'anthropic': keys.ANTHROPIC_API_KEY
    });
  }
  
  if (keys.OPENAI_API_KEY) {
    await injectAPIKeys(page, {
      'openai': keys.OPENAI_API_KEY
    });
  }
  
  return keys;
}

/**
 * Send a chat message via the UI
 */
export async function sendChatMessage(page, message) {
  // Wait for input to be available
  await page.waitForSelector('#user-input', { state: 'visible', timeout: 10000 });
  
  // Type the message
  await page.fill('#user-input', message);
  
  // Click send or press Enter
  const sendButton = await page.$('#send-btn');
  if (sendButton) {
    await sendButton.click();
  } else {
    await page.press('#user-input', 'Enter');
  }
  
  // Wait for message to appear
  await page.waitForTimeout(500);
}

/**
 * Wait for console log matching a pattern
 */
export async function waitForConsoleLog(page, pattern, timeout = 10000) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error(`Timeout waiting for log matching: ${pattern}`));
    }, timeout);
    
    const handler = (msg) => {
      const text = msg.text();
      if (pattern.test(text)) {
        clearTimeout(timer);
        page.off('console', handler);
        resolve(text);
      }
    };
    
    page.on('console', handler);
  });
}

/**
 * Get all messages from the chat
 */
export async function getChatMessages(page) {
  const messages = await page.locator('#messages > div').all();
  return messages.map(async (msg) => {
    const text = await msg.textContent();
    const classes = await msg.getAttribute('class');
    return { text, classes };
  });
}

/**
 * Wait for streaming to complete
 */
export async function waitForStreamComplete(page, timeout = 30000) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error('Timeout waiting for stream to complete'));
    }, timeout);
    
    const checkStreaming = async () => {
      const sendBtn = await page.$('#send-btn');
      const abortBtn = await page.$('#abort-btn');
      
      // Streaming is complete when Send button is visible and Abort is hidden
      if (sendBtn && !(await abortBtn?.isVisible())) {
        clearTimeout(timer);
        resolve();
        return;
      }
      
      setTimeout(checkStreaming, 500);
    };
    
    checkStreaming();
  });
}
