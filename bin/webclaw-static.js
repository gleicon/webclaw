#!/usr/bin/env node

/**
 * WebClaw Static - CLI Tool
 *
 * Zero-dependency HTTP server for serving the WebClaw static bundle.
 * Supports serving multi-file and single-file bundles with auto-open browser.
 */

import http from "http";
import fs from "fs";
import path from "path";
import { exec } from "child_process";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const DEFAULT_PORT = 8080;
const DIST_DIR = path.join(__dirname, "..", "dist-bundle");
const SINGLEFILE_DIR = path.join(__dirname, "..", "dist-singlefile");

const MIME_TYPES = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".mjs": "text/javascript",
  ".css": "text/css",
  ".wasm": "application/wasm",
  ".br": "application/brotli",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".otf": "font/otf",
  ".eot": "application/vnd.ms-fontobject",
};

function showHelp() {
  console.log(`
WebClaw Static - AI agent that runs in your browser

Usage:
  webclaw-static serve [options]    Start static server
  webclaw-static open               Open WebClaw in browser
  webclaw-static --help             Show this help

Options:
  --port=<number>   Server port (default: 8080)
  --open, -o        Auto-open browser
  --singlefile      Serve single-file version
  --ultimate        Serve ultimate single-file version

Examples:
  npx webclaw-static serve
  npx webclaw-static serve --port=3000 --open
  npx webclaw-static serve --singlefile
  npx webclaw-static open
`);
}

function parseArgs(args) {
  const options = {
    port: DEFAULT_PORT,
    open: false,
    singlefile: false,
    ultimate: false,
    help: false,
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === "--help" || arg === "-h") {
      options.help = true;
    } else if (arg === "--open" || arg === "-o") {
      options.open = true;
    } else if (arg === "--singlefile") {
      options.singlefile = true;
    } else if (arg === "--ultimate") {
      options.ultimate = true;
    } else if (arg.startsWith("--port=")) {
      const portNum = parseInt(arg.split("=")[1], 10);
      if (!isNaN(portNum) && portNum > 0 && portNum < 65536) {
        options.port = portNum;
      } else {
        console.error(`Invalid port number: ${arg}`);
        process.exit(1);
      }
    } else if (arg === "--port") {
      const nextArg = args[i + 1];
      if (nextArg && !isNaN(parseInt(nextArg, 10))) {
        const portNum = parseInt(nextArg, 10);
        if (portNum > 0 && portNum < 65536) {
          options.port = portNum;
          i++; // Skip next arg
        }
      }
    }
  }

  return options;
}

function serve(options) {
  // Determine which directory to serve
  let serveDir = DIST_DIR;
  let entryFile = "index.html";

  if (options.ultimate) {
    serveDir = SINGLEFILE_DIR;
    entryFile = "webclaw-ultimate.html";
    if (!fs.existsSync(path.join(serveDir, entryFile))) {
      console.error(
        `Ultimate bundle not found: ${path.join(serveDir, entryFile)}`,
      );
      console.error("Run: npm run build:singlefile:ultimate");
      process.exit(1);
    }
  } else if (options.singlefile) {
    serveDir = SINGLEFILE_DIR;
    entryFile = "webclaw.html";
    if (!fs.existsSync(path.join(serveDir, entryFile))) {
      console.error(
        `Single-file bundle not found: ${path.join(serveDir, entryFile)}`,
      );
      console.error("Run: npm run build:singlefile");
      process.exit(1);
    }
  }

  // Check if directory exists
  if (!fs.existsSync(serveDir)) {
    console.error(`Bundle directory not found: ${serveDir}`);
    console.error("Run: npm run build:all");
    process.exit(1);
  }

  const server = http.createServer((req, res) => {
    // Security: prevent directory traversal
    let requestedPath = decodeURIComponent(req.url || "/");
    if (requestedPath.includes("..")) {
      res.writeHead(403);
      return res.end("Forbidden");
    }

    // Determine file path
    let filePath;
    if (requestedPath === "/") {
      filePath = path.join(serveDir, entryFile);
    } else {
      filePath = path.join(serveDir, requestedPath);
    }

    // Check if file exists and is within serve directory
    if (!filePath.startsWith(serveDir)) {
      res.writeHead(403);
      return res.end("Forbidden");
    }

    // Check file existence
    if (!fs.existsSync(filePath)) {
      // For SPAs, serve index.html for non-existent routes
      if (options.singlefile || options.ultimate) {
        res.writeHead(404);
        return res.end("Not found");
      } else {
        filePath = path.join(serveDir, "index.html");
      }
    }

    // If directory, look for index.html
    if (fs.statSync(filePath).isDirectory()) {
      filePath = path.join(filePath, "index.html");
    }

    // Check again after potential changes
    if (!fs.existsSync(filePath)) {
      res.writeHead(404);
      return res.end("Not found");
    }

    // Determine content type
    const ext = path.extname(filePath).toLowerCase();
    let contentType = MIME_TYPES[ext] || "application/octet-stream";

    // Handle brotli compressed files
    const isBrotli = ext === ".br" || filePath.endsWith(".wasm.br");

    // Read and serve file
    fs.readFile(filePath, (err, content) => {
      if (err) {
        res.writeHead(500);
        return res.end("Internal server error");
      }

      const headers = {};

      if (isBrotli) {
        headers["Content-Encoding"] = "br";
        headers["Content-Type"] = "application/wasm";
      } else {
        headers["Content-Type"] = contentType;
      }

      // Add cache headers for static assets
      if (!ext.includes(".html")) {
        headers["Cache-Control"] = "public, max-age=31536000"; // 1 year
      }

      res.writeHead(200, headers);
      res.end(content);
    });
  });

  server.listen(options.port, () => {
    const url = `http://localhost:${options.port}`;
    console.log(`WebClaw serving at ${url}`);

    if (options.singlefile) {
      console.log("Mode: Single-file bundle");
    } else if (options.ultimate) {
      console.log("Mode: Ultimate standalone HTML");
    } else {
      console.log("Mode: Multi-file bundle (optimized)");
    }

    console.log("Press Ctrl+C to stop");

    if (options.open) {
      const command =
        process.platform === "darwin"
          ? `open ${url}`
          : process.platform === "win32"
            ? `start ${url}`
            : `xdg-open ${url}`;
      exec(command, (err) => {
        if (err) {
          console.error(`Could not open browser: ${err.message}`);
        }
      });
    }
  });

  // Graceful shutdown
  process.on("SIGINT", () => {
    console.log("\nShutting down...");
    server.close(() => {
      process.exit(0);
    });
  });
}

function openBrowser() {
  // Try to find the best way to open WebClaw
  const possiblePaths = [
    path.join(DIST_DIR, "index.html"),
    path.join(SINGLEFILE_DIR, "webclaw.html"),
    path.join(SINGLEFILE_DIR, "webclaw-ultimate.html"),
  ];

  let filePath = null;
  for (const p of possiblePaths) {
    if (fs.existsSync(p)) {
      filePath = p;
      break;
    }
  }

  if (!filePath) {
    console.error("No WebClaw bundle found. Run: npm run build:all");
    process.exit(1);
  }

  const command =
    process.platform === "darwin"
      ? `open "file://${filePath}"`
      : process.platform === "win32"
        ? `start "" "file://${filePath}"`
        : `xdg-open "file://${filePath}"`;

  exec(command, (err) => {
    if (err) {
      console.error(`Could not open browser: ${err.message}`);
      console.log(`Please open this file manually: ${filePath}`);
      process.exit(1);
    }
    console.log(`Opening ${filePath}`);
  });
}

// Main entry point
function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  if (!command || command === "--help" || command === "-h") {
    showHelp();
    process.exit(0);
  }

  const options = parseArgs(args.slice(1));

  if (options.help) {
    showHelp();
    process.exit(0);
  }

  if (command === "serve") {
    serve(options);
  } else if (command === "open") {
    openBrowser();
  } else {
    console.error(`Unknown command: ${command}`);
    showHelp();
    process.exit(1);
  }
}

main();
