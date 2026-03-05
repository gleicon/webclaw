// WebClaw Static Bundle Entry Point
// This file serves as the Vite entry point for the static bundle

// Note: just-bash will be integrated in Phase 7a
// For now, this file establishes the build structure

// Import CSS (will be processed by Vite)
import './styles/main.css';

// Log initialization
console.log('[WebClaw Static] Entry point loaded');
console.log('[WebClaw Static] Build: Phase 8 - Static Bundle');
console.log('[WebClaw Static] Note: just-bash integration planned for Phase 7a');

// The existing webclaw-host.js will be loaded via script tag in index.html
// This ensures backward compatibility with the current architecture

// Export version info for debugging
window.webclawStatic = {
  version: '0.1.0',
  build: 'phase-8-static-bundle',
  features: [
    'WASM runtime',
    'LLM providers (Anthropic, OpenAI, OpenRouter)',
    'Memory system (BM25 + embeddings)',
    'Tool registry',
    'Identity files',
    'Webchat UI'
  ],
  pending: [
    'just-bash file operations (Phase 7a)',
    'Local bridge binary (Phase 7b)'
  ]
};
