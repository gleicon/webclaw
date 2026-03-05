import { defineConfig } from 'vite';
import { resolve } from 'path';
import compression from 'vite-plugin-compression';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

// https://vitejs.dev/config/
export default defineConfig({
  // Root directory for source files
  root: '.',
  
  // Base URL - use relative paths for file:// compatibility
  base: './',
  
  // Build configuration
  build: {
    // Output directory
    outDir: 'dist-bundle',
    
    // Clean output directory before build
    emptyOutDir: true,
    
    // Target modern browsers (ES2020)
    target: 'es2020',
    
    // Module format
    format: 'es',
    
    // Minification settings
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: false, // Keep console logs for debugging
        drop_debugger: true,
      },
      format: {
        comments: false,
      },
    },
    
    // Rollup options for chunking
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index-vite.html'),
      },
      output: {
        // Entry chunk naming
        entryFileNames: 'assets/[name]-[hash].js',
        // Chunk naming for code splitting
        chunkFileNames: 'assets/[name]-[hash].js',
        // Asset naming (images, fonts, etc.)
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name.split('.');
          const ext = info[info.length - 1];
          
          // Special handling for WASM files
          if (ext === 'wasm' || ext === 'wasm.br') {
            return 'assets/[name][extname]';
          }
          
          return 'assets/[name]-[hash][extname]';
        },
      },
    },
    
    // Copy WASM files to dist
    assetsInlineLimit: 0, // Don't inline any assets
    
    // Source maps for debugging
    sourcemap: true,
    
    // Chunk size warning limit (in kbs)
    chunkSizeWarningLimit: 500,
  },
  
  // Plugins
  plugins: [
    // Brotli compression for assets
    compression({
      algorithm: 'brotliCompress',
      ext: '.br',
      threshold: 1024,
      deleteOriginFile: false,
    }),
    
    // Gzip compression fallback
    compression({
      algorithm: 'gzip',
      ext: '.gz',
      threshold: 1024,
      deleteOriginFile: false,
    }),
  ],
  
  // Development server
  server: {
    port: 8080,
    host: true,
    open: true,
    // Enable CORS for development
    cors: true,
    // Hot module replacement
    hmr: true,
  },
  
  // Preview server (for testing production build)
  preview: {
    port: 8080,
    host: true,
    open: true,
  },
  
  // CSS configuration
  css: {
    postcss: {
      plugins: [
        tailwindcss,
        autoprefixer,
      ],
    },
    // Extract CSS to separate files
    devSourcemap: true,
  },
  
  // Resolve aliases
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  
  // Optimize dependencies
  optimizeDeps: {
    exclude: [],
  },
  
  // Handle WASM files
  assetsInclude: ['**/*.wasm', '**/*.wasm.br'],
  
  // Esbuild options
  esbuild: {
    target: 'es2020',
    format: 'esm',
  },
});
