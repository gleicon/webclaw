#!/usr/bin/env node

/**
 * WebClaw Static - Serve command
 * Quick server for the static bundle
 */

import { createServer } from 'vite';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const rootDir = resolve(__dirname, '..');

async function serve() {
  const port = process.env.PORT || 8080;
  const host = process.env.HOST || 'localhost';
  
  console.log(`
╔══════════════════════════════════════════════════════════╗
║                    WebClaw Static                       ║
║              Browser-native AI Assistant                 ║
╚══════════════════════════════════════════════════════════╝
  `);
  
  const server = await createServer({
    root: rootDir,
    base: './',
    server: {
      port,
      host,
      open: true,
    },
  });
  
  await server.listen();
  
  const addresses = server.resolvedUrls;
  console.log('\n📦 WebClaw is running at:');
  addresses.local.forEach(url => console.log(`   Local:   ${url}`));
  addresses.network.forEach(url => console.log(`   Network: ${url}`));
  
  console.log('\nPress Ctrl+C to stop\n');
}

serve().catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
