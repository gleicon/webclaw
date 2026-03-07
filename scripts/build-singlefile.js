#!/usr/bin/env node
/**
 * Single-File Build Script for WebClaw
 *
 * Orchestrates the creation of a single HTML file containing all JS and CSS.
 * Handles Web Worker inlining via Blob URLs for true single-file distribution.
 *
 * Modes:
 *   Standard: HTML with inline JS+CSS, WASM external (~120KB HTML + 865KB WASM)
 *   Ultimate: HTML with EVERYTHING including WASM base64 (~1.3MB standalone)
 *
 * Usage:
 *   node scripts/build-singlefile.js              # Standard mode
 *   INLINE_WASM=true node scripts/build-singlefile.js  # Ultimate mode
 *   npm run build:singlefile                      # Via package.json
 *   npm run build:singlefile:ultimate             # Ultimate via package.json
 */

const { build } = require("vite");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

// Import WASM inlining utility
const { inlineWASM } = require("./inline-wasm.js");

// Build configuration
const BUILD_DIR = "dist-singlefile";
const WASM_DIR = "dist";
const STATIC_DIR = "static";
const OUTPUT_FILE = "webclaw.html";
const OUTPUT_ULTIMATE_FILE = "webclaw-ultimate.html";

/**
 * Main build function
 */
async function buildSingleFile() {
  const inlineWASMFlag = process.env.INLINE_WASM === "true";
  const mode = inlineWASMFlag ? "ULTIMATE" : "STANDARD";

  console.log(`[build-singlefile] Starting ${mode} mode build...`);

  try {
    // Step 1: Run Vite build with single-file config
    console.log("[build-singlefile] Step 1: Vite build...");
    await build({ configFile: "vite.singlefile.config.js" });

    // Step 2: Read the built HTML
    const htmlPath = path.join(BUILD_DIR, "index.html");
    if (!fs.existsSync(htmlPath)) {
      throw new Error(`Built HTML not found: ${htmlPath}`);
    }
    let html = fs.readFileSync(htmlPath, "utf8");
    console.log(
      `[build-singlefile] Step 2: Read HTML (${(html.length / 1024).toFixed(2)}KB)`,
    );

    // Step 3: Inline external scripts
    html = await inlineScripts(html);

    // Step 4: Inline external stylesheets
    html = await inlineStylesheets(html);

    // Step 5: Inline Web Worker
    html = inlineWorker(html);

    // Step 6: Copy static files that are referenced
    html = copyStaticAssets(html);

    // Step 7: Handle WASM (copy or inline based on mode)
    if (inlineWASMFlag) {
      console.log(
        "[build-singlefile] Step 7: Inlining WASM (ultimate mode)...",
      );
      const wasmPath = path.join(WASM_DIR, "webclaw.wasm.br");
      html = inlineWASM(html, wasmPath);
    } else {
      console.log("[build-singlefile] Step 7: Copying WASM file...");
      copyWASM();
    }

    // Step 8: Update paths for file:// protocol compatibility
    html = fixPathsForFileProtocol(html);

    // Step 9: Write final HTML
    const outputFileName = inlineWASMFlag ? OUTPUT_ULTIMATE_FILE : OUTPUT_FILE;
    const outputPath = path.join(BUILD_DIR, outputFileName);
    fs.writeFileSync(outputPath, html);
    console.log(
      `[build-singlefile] Step 9: Wrote ${outputFileName} (${(html.length / 1024).toFixed(2)}KB)`,
    );

    // Step 10: Report file sizes
    reportFileSizes();

    console.log(`[build-singlefile] ✓ Build complete: ${outputPath}`);

    // Step 11: Verification
    await verifyBuild(outputPath, inlineWASMFlag);
  } catch (err) {
    console.error("[build-singlefile] Build failed:", err);
    process.exit(1);
  }
}

/**
 * Inline external script references into the HTML
 */
async function inlineScripts(html) {
  console.log("[build-singlefile] Step 3: Inlining external scripts...");

  // Find all external script references
  const scriptRegex = /<script[^>]+src=["']([^"']+)["'][^>]*><\/script>/g;
  let match;
  let inlinedCount = 0;

  // Collect all matches first
  const matches = [];
  while ((match = scriptRegex.exec(html)) !== null) {
    matches.push({ fullMatch: match[0], src: match[1] });
  }

  // Process each script
  for (const { fullMatch, src } of matches) {
    // Resolve the script path
    let scriptPath;
    if (src.startsWith("./")) {
      scriptPath = path.join(BUILD_DIR, src.slice(2));
    } else if (src.startsWith("/")) {
      scriptPath = path.join(".", src);
    } else {
      scriptPath = path.join(BUILD_DIR, src);
    }

    // Also check in static directory for original files
    if (!fs.existsSync(scriptPath)) {
      const staticPath = path.join(STATIC_DIR, path.basename(src));
      if (fs.existsSync(staticPath)) {
        scriptPath = staticPath;
      }
    }

    if (fs.existsSync(scriptPath)) {
      const content = fs.readFileSync(scriptPath, "utf8");
      // Replace with inline script
      const typeAttr = fullMatch.includes('type="module"')
        ? ' type="module"'
        : "";
      const inlineScript = `<script${typeAttr}>${content}</script>`;
      html = html.replace(fullMatch, inlineScript);
      inlinedCount++;
      console.log(
        `  ✓ Inlined: ${src} (${(content.length / 1024).toFixed(2)}KB)`,
      );
    } else {
      console.warn(`  ⚠ Script not found: ${scriptPath}`);
    }
  }

  console.log(`[build-singlefile] Inlined ${inlinedCount} scripts`);
  return html;
}

/**
 * Inline external stylesheet references into the HTML
 */
async function inlineStylesheets(html) {
  console.log("[build-singlefile] Step 4: Inlining stylesheets...");

  // Find all external stylesheet references
  const linkRegex =
    /<link[^>]+rel=["']stylesheet["'][^>]+href=["']([^"']+)["'][^>]*>/g;
  let match;
  let inlinedCount = 0;

  const matches = [];
  while ((match = linkRegex.exec(html)) !== null) {
    matches.push({ fullMatch: match[0], href: match[1] });
  }

  for (const { fullMatch, href } of matches) {
    let cssPath;
    if (href.startsWith("./")) {
      cssPath = path.join(BUILD_DIR, href.slice(2));
    } else if (href.startsWith("/")) {
      cssPath = path.join(".", href);
    } else {
      cssPath = path.join(BUILD_DIR, href);
    }

    if (fs.existsSync(cssPath)) {
      const content = fs.readFileSync(cssPath, "utf8");
      const inlineStyle = `<style>${content}</style>`;
      html = html.replace(fullMatch, inlineStyle);
      inlinedCount++;
      console.log(
        `  ✓ Inlined: ${href} (${(content.length / 1024).toFixed(2)}KB)`,
      );
    } else {
      console.warn(`  ⚠ Stylesheet not found: ${cssPath}`);
    }
  }

  console.log(`[build-singlefile] Inlined ${inlinedCount} stylesheets`);
  return html;
}

/**
 * Inline Web Worker by converting it to a Blob URL creator
 */
function inlineWorker(html) {
  console.log("[build-singlefile] Step 5: Inlining Web Worker...");

  const workerPath = path.join(STATIC_DIR, "worker.js");
  if (!fs.existsSync(workerPath)) {
    console.warn("  ⚠ Worker file not found, skipping worker inlining");
    return html;
  }

  // Read worker code
  let workerCode = fs.readFileSync(workerPath, "utf8");

  // Also need to inline wasm_exec.js since worker imports it
  const wasmExecPath = path.join(STATIC_DIR, "wasm_exec.js");
  if (fs.existsSync(wasmExecPath)) {
    const wasmExecCode = fs.readFileSync(wasmExecPath, "utf8");
    // Replace the importScripts call with inline code
    workerCode = workerCode.replace(
      /importScripts\(['"]\.\/wasm_exec\.js['"]\);/,
      `// wasm_exec.js inlined\n${wasmExecCode}`,
    );
    console.log("  ✓ Inlined wasm_exec.js into worker");
  }

  // Escape the worker code for JavaScript string embedding
  const escapedWorkerCode = workerCode
    .replace(/\\/g, "\\\\")
    .replace(/`/g, "\\`")
    .replace(/\$/g, "\\$");

  // Create the inline worker loader
  const inlineWorkerLoader = `
// Inline Web Worker Loader - Auto-generated by build-singlefile.js
(function() {
    'use strict';
    
    const __WORKER_CODE__ = \`${escapedWorkerCode}\`;
    
    window.__createWebClawWorker = function() {
        const blob = new Blob([__WORKER_CODE__], { type: 'application/javascript' });
        const url = URL.createObjectURL(blob);
        return new Worker(url, { type: 'module' });
    };
})();
`;

  // Inject the worker loader into the HTML
  // Find a good injection point (before the first script tag)
  const firstScriptMatch = html.match(/<script[^>]*>/);
  if (firstScriptMatch) {
    const injectionPoint = firstScriptMatch.index;
    const before = html.substring(0, injectionPoint);
    const after = html.substring(injectionPoint);
    html = before + `<script>${inlineWorkerLoader}</script>\n` + after;
    console.log("  ✓ Injected worker loader");
  }

  // Replace new Worker('static/worker.js') with __createWebClawWorker()
  const originalWorkerRegex =
    /new Worker\(['"](?:\.\/)?static\/worker\.js['"]\)/;
  if (originalWorkerRegex.test(html)) {
    html = html.replace(originalWorkerRegex, "window.__createWebClawWorker()");
    console.log("  ✓ Replaced Worker instantiation with inline version");
  }

  return html;
}

/**
 * Copy static assets that are referenced but not inlined
 */
function copyStaticAssets(html) {
  console.log("[build-singlefile] Step 6: Copying static assets...");

  // Copy vendor files if referenced
  const vendorFiles = [
    "vendor/browser.js", // just-bash browser bundle
  ];

  for (const file of vendorFiles) {
    if (html.includes(file)) {
      const srcPath = path.join(".", file);
      const destPath = path.join(BUILD_DIR, file);

      if (fs.existsSync(srcPath)) {
        fs.mkdirSync(path.dirname(destPath), { recursive: true });
        fs.copyFileSync(srcPath, destPath);
        console.log(`  ✓ Copied: ${file}`);
      } else {
        console.warn(`  ⚠ Not found: ${file}`);
      }
    }
  }

  return html;
}

/**
 * Copy WASM files to build directory
 */
function copyWASM() {
  const wasmFiles = ["webclaw.wasm", "webclaw.wasm.br"];

  for (const file of wasmFiles) {
    const srcPath = path.join(WASM_DIR, file);
    const destPath = path.join(BUILD_DIR, file);

    if (fs.existsSync(srcPath)) {
      fs.copyFileSync(srcPath, destPath);
      const size = (fs.statSync(destPath).size / 1024).toFixed(2);
      console.log(`  ✓ Copied: ${file} (${size}KB)`);
    } else {
      console.warn(`  ⚠ WASM file not found: ${srcPath}`);
    }
  }
}

/**
 * Fix paths for file:// protocol compatibility
 * Convert relative paths to work when opened directly
 */
function fixPathsForFileProtocol(html) {
  console.log(
    "[build-singlefile] Step 8: Fixing paths for file:// protocol...",
  );

  // Update WASM fetch paths to be relative to current directory
  // The html is in dist-singlefile/, WASM is alongside it
  html = html.replace(
    /fetch\(['"]dist\/webclaw\.wasm['"]\)/g,
    "fetch('webclaw.wasm')",
  );

  // Update worker.js references (though they should be replaced by now)
  html = html.replace(/['"]\.\/static\/worker\.js['"]/g, "'./worker.js'");

  // Ensure vendor paths are relative
  html = html.replace(/src=["']\.\/vendor\//g, 'src="./vendor/');

  return html;
}

/**
 * Report file sizes in the build directory
 */
function reportFileSizes() {
  console.log("[build-singlefile] File sizes:");

  const files = [
    OUTPUT_FILE,
    OUTPUT_ULTIMATE_FILE,
    "webclaw.wasm",
    "webclaw.wasm.br",
  ];

  for (const file of files) {
    const filePath = path.join(BUILD_DIR, file);
    if (fs.existsSync(filePath)) {
      const size = (fs.statSync(filePath).size / 1024).toFixed(2);
      console.log(`  ${file}: ${size}KB`);
    }
  }
}

/**
 * Verify the build output
 */
async function verifyBuild(outputPath, isUltimate) {
  console.log("[build-singlefile] Verifying build...");

  const html = fs.readFileSync(outputPath, "utf8");

  // Check 1: No external script references (except possibly vendor)
  const externalScripts = html.match(/<script[^>]+src=["'][^"']+["'][^>]*>/g);
  if (externalScripts) {
    const nonVendorScripts = externalScripts.filter(
      (s) => !s.includes("vendor"),
    );
    if (nonVendorScripts.length > 0) {
      console.warn("  ⚠ Found external script references:", nonVendorScripts);
    } else {
      console.log("  ✓ All scripts inlined (vendor files excluded)");
    }
  } else {
    console.log("  ✓ All scripts inlined");
  }

  // Check 2: No external stylesheet references
  const externalStyles = html.match(
    /<link[^>]+rel=["']stylesheet["'][^>]+href=["'][^"']+["'][^>]*>/g,
  );
  if (externalStyles) {
    console.warn("  ⚠ Found external stylesheet references:", externalStyles);
  } else {
    console.log("  ✓ All stylesheets inlined");
  }

  // Check 3: Worker inlining
  if (html.includes("__createWebClawWorker")) {
    console.log("  ✓ Worker inlining detected");
  } else {
    console.warn("  ⚠ Worker inlining not detected");
  }

  // Check 4: WASM inlining (ultimate mode)
  if (isUltimate) {
    if (html.includes("__WASM_BASE64__") || html.includes("decompressGzip")) {
      console.log("  ✓ WASM inlining detected");
    } else {
      console.warn("  ⚠ WASM inlining not detected");
    }
  }

  // Check 5: WASM fetch interceptor
  if (html.includes("fetch interceptor")) {
    console.log("  ✓ WASM fetch interceptor detected");
  }

  console.log("[build-singlefile] ✓ Verification complete");
}

// Run the build
buildSingleFile();
