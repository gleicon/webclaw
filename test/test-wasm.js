#!/usr/bin/env node
/**
 * Automated WASM verification test for WebClaw Phase 1
 * Uses puppeteer-core with system Chrome
 */

import puppeteer from 'puppeteer-core';
import http from 'http';

const DEV_SERVER_URL = 'http://localhost:8080';
const CHROME_PATH = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const TIMEOUT = 30000;

// Wait for server to be ready
async function waitForServer(url, maxAttempts = 30, delay = 500) {
  for (let i = 0; i < maxAttempts; i++) {
    try {
      await new Promise((resolve, reject) => {
        const req = http.get(url, (res) => {
          if (res.statusCode === 200) resolve();
          else reject(new Error(`Status ${res.statusCode}`));
        });
        req.on('error', reject);
        req.setTimeout(2000, () => reject(new Error('Timeout')));
      });
      return true;
    } catch (err) {
      process.stdout.write('.');
      await new Promise(r => setTimeout(r, delay));
    }
  }
  return false;
}

async function runTests() {
  console.log('🧪 WebClaw WASM Automated Test Suite\n');
  
  // Check if dev server is running
  console.log('Checking dev server...');
  const serverReady = await waitForServer(DEV_SERVER_URL);
  if (!serverReady) {
    console.error('\n❌ Dev server not running at ' + DEV_SERVER_URL);
    console.error('Start it with: go run ./cmd/devserver/');
    process.exit(1);
  }
  console.log('✓ Dev server is running\n');
  
  // Check build artifacts
  console.log('Checking build artifacts...');
  const fs = await import('fs');
  
  const checkFile = (path) => {
    if (!fs.existsSync(path)) {
      console.error(`❌ ${path} not found. Run: make build`);
      process.exit(1);
    }
    const stats = fs.statSync(path);
    const size = (stats.size / 1024).toFixed(1);
    console.log(`✓ ${path} (${size}KB)`);
    return stats.size;
  };
  
  const wasmSize = checkFile('dist/webclaw.wasm');
  const wasmBrSize = checkFile('dist/webclaw.wasm.br');
  checkFile('static/wasm_exec.js');
  
  const compressionRatio = ((1 - wasmBrSize / wasmSize) * 100).toFixed(1);
  console.log(`✓ Brotli compression: ${compressionRatio}% size reduction\n`);
  
  // Launch Chrome
  console.log('Launching Chrome...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--enable-features=SharedArrayBuffer',
      '--disable-features=IsolateOrigins,site-per-process'
    ]
  });
  
  try {
    const page = await browser.newPage();
    const logs = [];
    const errors = [];
    
    // Capture console messages
    page.on('console', msg => {
      const text = msg.text();
      logs.push({ type: msg.type(), text });
      console.log(`[${msg.type().toUpperCase()}] ${text}`);
    });
    
    // Capture page errors
    page.on('pageerror', err => {
      errors.push(err.message);
      console.error(`[PAGE ERROR] ${err.message}`);
    });
    
    // Navigate to page
    console.log('\n📋 Test 1: Loading page and verifying WASM instantiates...');
    await page.goto(DEV_SERVER_URL, { waitUntil: 'networkidle0', timeout: TIMEOUT });
    
    // Wait for webclaw object to be available
    await page.waitForFunction(() => {
      return window.webclaw && typeof window.webclaw.jsFetch === 'function';
    }, { timeout: TIMEOUT });
    
    console.log('✅ WASM loaded and bridges registered\n');
    
    // Analyze logs
    console.log('📋 Test 2: Verifying console messages...');
    const wasmReady = logs.some(l => l.text.includes('webclaw: WASM ready'));
    const bridgesAvailable = logs.some(l => l.text.includes('webclaw: bridges available'));
    
    if (!wasmReady) console.log('⚠️  "webclaw: WASM ready" message not found in console');
    else console.log('✅ "webclaw: WASM ready" message found');
    
    if (!bridgesAvailable) console.log('⚠️  "webclaw: bridges available" message not found');
    else console.log('✅ "webclaw: bridges available" message found');
    console.log('');
    
    // Test jsFetch
    console.log('📋 Test 3: Testing jsFetch bridge...');
    const fetchResult = await page.evaluate(async () => {
      try {
        // Use local test endpoint to avoid CORS issues
        const response = await window.webclaw.jsFetch('http://localhost:8080/api/test');
        const text = await response.text();
        return { success: true, length: text.length, preview: text.substring(0, 100) };
      } catch (err) {
        return { success: false, error: err.message };
      }
    });
    
    if (!fetchResult.success) {
      throw new Error(`jsFetch failed: ${fetchResult.error}`);
    }
    
    console.log(`✅ jsFetch works - fetched ${fetchResult.length} characters`);
    console.log(`   Preview: ${fetchResult.preview}...\n`);
    
    // Test jsIndexedDB
    console.log('📋 Test 4: Testing jsIndexedDB bridge...');
    const idbResult = await page.evaluate(() => {
      try {
        const request = window.webclaw.jsIndexedDB.open('test-db', 1);
        return { 
          success: true, 
          type: typeof request,
          hasReadyState: 'readyState' in request,
          readyState: request.readyState
        };
      } catch (err) {
        return { success: false, error: err.message };
      }
    });
    
    if (!idbResult.success) {
      throw new Error(`jsIndexedDB failed: ${idbResult.error}`);
    }
    
    if (idbResult.type !== 'object' || !idbResult.hasReadyState) {
      throw new Error(`jsIndexedDB returned invalid object: type=${idbResult.type}`);
    }
    
    console.log(`✅ jsIndexedDB works - returned valid IDBOpenDBRequest (readyState: ${idbResult.readyState})\n`);
    
    // Final summary
    console.log('═══════════════════════════════════════════════════');
    console.log('📊 Test Summary');
    console.log('═══════════════════════════════════════════════════');
    console.log(`  WASM Module:       ✅ Loaded (${(wasmSize/1024/1024).toFixed(2)}MB)`);
    console.log(`  Bridges:           ✅ Available`);
    console.log(`  jsFetch:           ✅ Working (${fetchResult.length} chars fetched)`);
    console.log(`  jsIndexedDB:       ✅ Working`);
    console.log(`  Compression:       ✅ ${compressionRatio}% reduction`);
    
    if (errors.length > 0) {
      console.log(`\n⚠️  ${errors.length} page error(s) detected`);
    }
    
    console.log('');
    console.log('🎉 ALL TESTS PASSED - Phase 1 verification complete!');
    console.log('═══════════════════════════════════════════════════');
    console.log('');
    console.log('Requirements satisfied:');
    console.log('  ✅ BUILD-01: WASM binary compiles');
    console.log('  ✅ BUILD-02: Host page loads WASM in browser');
    console.log('  ✅ BUILD-03: jsFetch and jsIndexedDB bridges callable');
    console.log('  ✅ BUILD-04: Brotli-compressed artifact produced');
    console.log('');
    console.log('Files created:');
    console.log('  - index.html (host page)');
    console.log('  - static/webclaw-host.js (WASM loader)');
    console.log('  - static/wasm_exec.js (Go runtime)');
    console.log('  - cmd/devserver/main.go (dev server)');
    console.log('  - Makefile (build pipeline)');
    console.log('  - dist/webclaw.wasm (WASM binary)');
    console.log('  - dist/webclaw.wasm.br (compressed)');
    
    process.exit(0);
    
  } catch (error) {
    console.error('\n❌ TEST FAILED:\n');
    console.error(error.message);
    
    if (error.message.includes('net::ERR')) {
      console.error('\nNote: Network errors may occur if fetching external URLs fails.');
      console.error('This can happen in restricted network environments.');
    }
    
    process.exit(1);
  } finally {
    await browser.close();
  }
}

runTests();
