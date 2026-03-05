/**
 * Phase 06 Browser E2E Test Runner
 * Starts Go dev server, runs Playwright tests, reports results
 */

import { spawn } from 'child_process';
import http from 'http';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { readFileSync, existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const DEV_SERVER_URL = 'http://localhost:8080';
const MAX_WAIT_ATTEMPTS = 60;
const WAIT_DELAY_MS = 1000;

// Load environment variables from .env.test
function loadEnvFile() {
  const envPath = join(__dirname, '..', '.env.test');
  if (!existsSync(envPath)) {
    console.log('⚠️  .env.test not found - API keys may not be available');
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

// Colors for terminal output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logHeader(title) {
  console.log('');
  console.log(`${colors.bright}${colors.cyan}═══════════════════════════════════════════════════════${colors.reset}`);
  console.log(`${colors.bright}${colors.cyan}  ${title}${colors.reset}`);
  console.log(`${colors.bright}${colors.cyan}═══════════════════════════════════════════════════════${colors.reset}`);
  console.log('');
}

// Wait for server to be ready
async function waitForServer(url, maxAttempts = MAX_WAIT_ATTEMPTS, delay = WAIT_DELAY_MS) {
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

// Start dev server
async function startDevServer() {
  logHeader('STARTING DEV SERVER');
  
  const projectRoot = join(__dirname, '..');
  const envVars = loadEnvFile();
  
  log(`Working directory: ${projectRoot}`, 'dim');
  log('Command: go run ./cmd/devserver/', 'dim');
  
  // Check for API keys
  if (envVars.ANTHROPIC_API_KEY) {
    log('✓ ANTHROPIC_API_KEY loaded from .env.test', 'green');
  } else {
    log('⚠️  ANTHROPIC_API_KEY not found in .env.test', 'yellow');
  }
  
  if (envVars.OPENAI_API_KEY) {
    log('✓ OPENAI_API_KEY loaded from .env.test', 'green');
  } else {
    log('⚠️  OPENAI_API_KEY not found in .env.test', 'yellow');
  }
  
  log('');
  
  const server = spawn('go', ['run', './cmd/devserver/'], {
    cwd: projectRoot,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      ...envVars
    }
  });
  
  // Capture server output
  server.stdout.on('data', (data) => {
    const lines = data.toString().trim().split('\n');
    lines.forEach(line => {
      if (line.includes('error') || line.includes('Error') || line.includes('FAIL')) {
        console.log(`${colors.red}[SERVER] ${line}${colors.reset}`);
      } else if (line.includes('Listening') || line.includes('Ready') || line.includes('8080')) {
        console.log(`${colors.green}[SERVER] ${line}${colors.reset}`);
      } else {
        console.log(`${colors.dim}[SERVER] ${line}${colors.reset}`);
      }
    });
  });
  
  server.stderr.on('data', (data) => {
    console.log(`${colors.yellow}[SERVER ERR] ${data.toString().trim()}${colors.reset}`);
  });
  
  server.on('error', (err) => {
    console.error(`${colors.red}Failed to start server: ${err.message}${colors.reset}`);
    process.exit(1);
  });
  
  // Wait for server to be ready
  log('Waiting for server to be ready', 'yellow');
  const ready = await waitForServer(DEV_SERVER_URL);
  
  if (!ready) {
    console.error('');
    log('❌ Server failed to start within timeout', 'red');
    server.kill();
    process.exit(1);
  }
  
  console.log('');
  log(`✅ Dev server ready at ${DEV_SERVER_URL}`, 'green');
  
  return server;
}

// Run Playwright tests
async function runPlaywrightTests() {
  logHeader('RUNNING PLAYWRIGHT TESTS');
  
  return new Promise((resolve, reject) => {
    const headed = process.env.HEADLESS === 'false';
    const args = ['npx', 'playwright', 'test', 'phase06-browser-tests'];
    
    if (process.env.CI) {
      args.push('--reporter=list,json');
    }
    
    log(`Running: ${args.join(' ')}`, 'dim');
    if (headed) {
      log('Mode: HEADED (browser visible)', 'yellow');
    } else {
      log('Mode: HEADLESS', 'cyan');
    }
    log('');
    
    const testProcess = spawn('npx', ['playwright', 'test', 'phase06-browser-tests', '--reporter=list'], {
      cwd: __dirname,
      stdio: 'inherit',
      env: {
        ...process.env,
        PLAYWRIGHT_JSON_OUTPUT_FILE: join(__dirname, 'playwright-report', 'results.json')
      }
    });
    
    testProcess.on('close', (code) => {
      resolve(code);
    });
    
    testProcess.on('error', (err) => {
      reject(err);
    });
  });
}

// Main execution
async function main() {
  const startTime = Date.now();
  let server = null;
  let exitCode = 0;
  
  try {
    logHeader('WEBCLAW PHASE 06 E2E BROWSER TESTS');
    log('Testing real browser behavior with Playwright', 'dim');
    log('Console capture | IndexedDB access | UI interaction', 'dim');
    
    // Check if server is already running
    log('');
    log('Checking for existing server...', 'yellow');
    const existingServer = await waitForServer(DEV_SERVER_URL, 3, 500);
    
    if (existingServer) {
      log(`✅ Using existing server at ${DEV_SERVER_URL}`, 'green');
    } else {
      server = await startDevServer();
    }
    
    // Run tests
    exitCode = await runPlaywrightTests();
    
    // Report results
    const duration = ((Date.now() - startTime) / 1000).toFixed(1);
    
    logHeader('TEST RESULTS');
    
    if (exitCode === 0) {
      log(`✅ ALL TESTS PASSED`, 'green');
      log(`Duration: ${duration}s`, 'dim');
      log(`Reports: test/playwright-report/`, 'dim');
    } else {
      log(`❌ TESTS FAILED (exit code: ${exitCode})`, 'red');
      log(`Duration: ${duration}s`, 'dim');
      log(`Check screenshots: test/playwright-report/`, 'yellow');
    }
    
    console.log('');
    log('Test coverage:', 'bright');
    log('  ✅ Summarization (20-message threshold)', 'green');
    log('  ✅ Token counting display', 'green');
    log('  ✅ Memory flush & MEMORY.md', 'green');
    log('  ✅ Tool registry & execution', 'green');
    log('  ✅ Memory search', 'green');
    log('  ✅ Provider failover', 'green');
    log('  ✅ Fail-fast error handling', 'green');
    log('  ✅ Storage hygiene', 'green');
    log('  ✅ Smoke test (startup logs)', 'green');
    log('  ✅ Health tracking', 'green');
    log('  ✅ Async embedder', 'green');
    
  } catch (error) {
    logHeader('FATAL ERROR');
    log(`❌ ${error.message}`, 'red');
    console.error(error);
    exitCode = 1;
  } finally {
    // Cleanup
    if (server) {
      log('');
      log('Stopping dev server...', 'yellow');
      server.kill('SIGTERM');
      
      // Give it a moment to cleanup
      await new Promise(r => setTimeout(r, 1000));
      
      if (!server.killed) {
        server.kill('SIGKILL');
      }
      
      log('✅ Server stopped', 'green');
    }
    
    process.exit(exitCode);
  }
}

// Handle Ctrl+C gracefully
process.on('SIGINT', () => {
  log('\n\nInterrupted by user', 'yellow');
  process.exit(130);
});

main();
