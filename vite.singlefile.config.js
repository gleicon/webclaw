import { defineConfig } from "vite";
import { resolve } from "path";
import tailwindcss from "tailwindcss";
import autoprefixer from "autoprefixer";

/**
 * Vite Configuration for Single-File Build
 *
 * This config produces a single HTML file with all JS and CSS inlined.
 * WASM is kept external for the standard build, but can be inlined for "ultimate" mode.
 *
 * Key differences from base vite.config.js:
 * - inlineDynamicImports: true (no code splitting)
 * - cssCodeSplit: false (inline CSS)
 * - assetsInlineLimit: 10MB (inline all assets except WASM)
 * - No compression plugins (we'll compress the final single file if needed)
 */

export default defineConfig({
  // Root directory for source files
  root: ".",

  // Base URL - use relative paths for file:// compatibility
  base: "./",

  // Build configuration
  build: {
    // Output directory for single-file build
    outDir: "dist-singlefile",

    // Clean output directory before build
    emptyOutDir: true,

    // Target modern browsers (ES2020)
    target: "es2020",

    // Module format
    format: "es",

    // Minification settings
    minify: "terser",
    terserOptions: {
      compress: {
        drop_console: false,
        drop_debugger: true,
      },
      format: {
        comments: false,
      },
    },

    // Rollup options for single-file output
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
      },
      output: {
        // Inline all dynamic imports into a single bundle
        inlineDynamicImports: true,

        // No manual chunks (everything in one file)
        manualChunks: undefined,

        // Entry chunk naming (will be inlined anyway)
        entryFileNames: "assets/[name]-[hash].js",

        // Asset naming
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name.split(".");
          const ext = info[info.length - 1];

          // Keep WASM files separate
          if (ext === "wasm" || ext === "br") {
            return "[name][extname]";
          }

          return "assets/[name]-[hash][extname]";
        },
      },
    },

    // Inline all assets under 10MB (catches JS and CSS)
    assetsInlineLimit: 10 * 1024 * 1024,

    // Don't split CSS into separate files
    cssCodeSplit: false,

    // Source maps for debugging
    sourcemap: false, // No source maps for production single-file

    // Chunk size warning (expect large bundle with inline WASM)
    chunkSizeWarningLimit: 2000,
  },

  // Plugins - no compression, no static copy (we handle that in build script)
  plugins: [],

  // Development server
  server: {
    port: 8080,
    host: true,
    open: true,
    cors: true,
    hmr: true,
  },

  // Preview server
  preview: {
    port: 8080,
    host: true,
    open: true,
  },

  // CSS configuration
  css: {
    postcss: {
      plugins: [tailwindcss, autoprefixer],
    },
    devSourcemap: true,
  },

  // Resolve aliases
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src"),
    },
  },

  // Optimize dependencies
  optimizeDeps: {
    exclude: [],
  },

  // Handle WASM files
  assetsInclude: ["**/*.wasm", "**/*.wasm.br"],

  // Esbuild options
  esbuild: {
    target: "es2020",
    format: "esm",
  },
});
