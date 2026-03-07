// WebClaw Static Bundle Entry Point
// This file serves as the Vite entry point for the static bundle

// Import CSS (processed by Vite + Tailwind)
import "./styles/main.css";

// Log initialization
console.log("[WebClaw Static] Entry point loaded");
console.log("[WebClaw Static] Build: Phase 8 - Static Bundle");
console.log("[WebClaw Static] CSS: Tailwind compiled + minified");

// Note: wasm_exec.js and webclaw-host.js are loaded via script tags in index.html
// This ensures proper initialization order (WASM runtime must be available before host)
// The just-bash integration is available via global window.JustBash

// Export version info for debugging
window.webclawStatic = {
  version: "0.1.0",
  build: "phase-8-static-bundle",
  buildDate: new Date().toISOString(),
  features: [
    "WASM runtime (Go)",
    "LLM providers (Anthropic, OpenAI, OpenRouter)",
    "Memory system (BM25 + embeddings)",
    "Tool registry",
    "Identity files",
    "Webchat UI",
    "File operations (just-bash)",
  ],
  tech: {
    bundler: "Vite",
    css: "Tailwind CSS (compiled)",
    wasm: "Go 1.21+",
    compression: "Brotli + Gzip",
  },
};

console.log("[WebClaw Static] Version:", window.webclawStatic.version);
console.log("[WebClaw Static] Build date:", window.webclawStatic.buildDate);
