// webclaw-host.js — WASM loader and bridge harness with Web Worker support
// Handles both main-thread WASM (for config/identity) and worker-thread WASM (for streaming)

(function() {
    'use strict';
    
    // Global state
    const state = {
        wasmLoaded: false,
        worker: null,
        workerReady: false,
        isStreaming: false,
        streamCallbacks: {
            onToken: null,
            onComplete: null,
            onError: null
        }
    };
    
    // Message types (must match worker.js)
    const MSG_TYPES = {
        // Main -> Worker
        INIT_WASM: 'INIT_WASM',
        START_STREAM: 'START_STREAM',
        ADD_MESSAGE: 'ADD_MESSAGE',
        ABORT_STREAM: 'ABORT_STREAM',

        // Worker -> Main
        WASM_READY: 'WASM_READY',
        WASM_ERROR: 'WASM_ERROR',
        TOKEN: 'TOKEN',
        COMPLETE: 'COMPLETE',
        ERROR: 'ERROR',
        STREAM_STARTED: 'STREAM_STARTED',
        STREAM_ABORTED: 'STREAM_ABORTED',
        TOOL_EVENT: 'TOOL_EVENT'
    };
    
    // Initialize main thread WASM (for config, identity, crypto)
    async function initMainThreadWASM() {
        const go = new Go();
        
        // Polyfill for browsers without instantiateStreaming
        if (!WebAssembly.instantiateStreaming) {
            WebAssembly.instantiateStreaming = async (resp, importObject) => {
                const source = await (await resp).arrayBuffer();
                return await WebAssembly.instantiate(source, importObject);
            };
        }
        
        try {
            const result = await WebAssembly.instantiateStreaming(
                fetch("dist/webclaw.wasm"),
                go.importObject
            );
            go.run(result.instance);
            state.wasmLoaded = true;
            console.log('[host] Main thread WASM loaded');
        } catch (err) {
            console.error("[host] Failed to load main thread WASM:", err);
            throw err;
        }
    }
    
    // Initialize Web Worker for streaming
    async function initWorker() {
        return new Promise((resolve, reject) => {
            try {
                console.log('[host] Creating Web Worker...');
                
                // Create worker
                state.worker = new Worker('static/worker.js');
                
                // Handle messages from worker
                state.worker.onmessage = handleWorkerMessage;
                state.worker.onerror = (err) => {
                    console.error('[host] Worker error:', err);
                    reject(err);
                };
                
                // Fetch WASM binary to pass to worker
                fetch('dist/webclaw.wasm')
                    .then(r => r.arrayBuffer())
                    .then(wasmBuffer => {
                        console.log('[host] Sending INIT_WASM to worker...');
                        state.worker.postMessage({
                            type: MSG_TYPES.INIT_WASM,
                            payload: { wasmBinary: wasmBuffer }
                        });
                    })
                    .catch(err => reject(err));
                
                // Wait for WASM_READY
                const checkReady = (event) => {
                    if (event.data && event.data.type === MSG_TYPES.WASM_READY) {
                        state.worker.removeEventListener('message', checkReady);
                        state.workerReady = true;
                        console.log('[host] Worker WASM ready');
                        resolve();
                    } else if (event.data && event.data.type === MSG_TYPES.WASM_ERROR) {
                        state.worker.removeEventListener('message', checkReady);
                        reject(new Error(event.data.payload.error));
                    }
                };
                state.worker.addEventListener('message', checkReady);
                
            } catch (err) {
                reject(err);
            }
        });
    }
    
    // Handle messages from worker
    function handleWorkerMessage(event) {
        const { type, payload } = event.data;
        
        switch (type) {
            case MSG_TYPES.TOKEN:
                if (state.streamCallbacks.onToken) {
                    state.streamCallbacks.onToken(payload.token);
                }
                break;
                
            case MSG_TYPES.COMPLETE:
                state.isStreaming = false;
                if (state.streamCallbacks.onComplete) {
                    state.streamCallbacks.onComplete(payload);
                }
                break;
                
            case MSG_TYPES.ERROR:
                state.isStreaming = false;
                console.error('[host] Stream error:', payload);
                if (state.streamCallbacks.onError) {
                    state.streamCallbacks.onError(payload);
                }
                break;
                
            case MSG_TYPES.STREAM_STARTED:
                state.isStreaming = true;
                console.log('[host] Stream started:', payload.timestamp);
                break;
                
            case MSG_TYPES.STREAM_ABORTED:
                state.isStreaming = false;
                console.log('[host] Stream aborted:', payload.timestamp);
                break;
                
            case MSG_TYPES.WASM_ERROR:
                console.error('[host] Worker WASM error:', payload);
                break;

            case MSG_TYPES.TOOL_EVENT:
                window.dispatchEvent(new CustomEvent('webclaw:tool-event', {
                    detail: event.data.payload  // { toolName, status, summary, full }
                }));
                break;

            default:
                console.log('[host] Unknown message from worker:', type, payload);
        }
    }
    
    // Start a streaming LLM request
    function startStream(options = {}) {
        if (!state.workerReady) {
            throw new Error('Worker not initialized. Call initWorker() first.');
        }
        
        if (state.isStreaming) {
            throw new Error('Stream already in progress');
        }
        
        // Set callbacks
        state.streamCallbacks.onToken = options.onToken || null;
        state.streamCallbacks.onComplete = options.onComplete || null;
        state.streamCallbacks.onError = options.onError || null;
        
        console.log('[host] Starting stream...');
        
        state.worker.postMessage({
            type: MSG_TYPES.START_STREAM,
            payload: {
                provider: options.provider || 'mock',
                model: options.model || 'default',
                messages: options.messages || []
            }
        });
    }
    
    // Add a message to conversation history
    function addMessage(role, content) {
        if (!state.workerReady) {
            throw new Error('Worker not initialized');
        }
        
        state.worker.postMessage({
            type: MSG_TYPES.ADD_MESSAGE,
            payload: { role, content }
        });
    }
    
    // Abort the current stream
    function abortStream() {
        if (!state.workerReady) {
            throw new Error('Worker not initialized');
        }
        
        if (!state.isStreaming) {
            console.log('[host] No stream to abort');
            return;
        }
        
        console.log('[host] Sending abort...');
        state.worker.postMessage({
            type: MSG_TYPES.ABORT_STREAM
        });
    }
    
    // File input handling for import/export
    function setupFileHandling() {
        window.addEventListener('webclaw:request-export', (e) => {
            const { filename, content } = e.detail;
            downloadConfig(filename || 'webclaw-config.json', content);
        });
        
        window.addEventListener('webclaw:request-import', () => {
            triggerFileImport();
        });
    }
    
    function downloadConfig(filename, content) {
        const blob = new Blob([content], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }
    
    function triggerFileImport() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json,application/json';
        input.style.display = 'none';
        
        input.onchange = async (e) => {
            const file = e.target.files[0];
            if (!file) return;
            
            try {
                const content = await file.text();
                window.dispatchEvent(new CustomEvent('webclaw:import-data', {
                    detail: { content, filename: file.name }
                }));
            } catch (err) {
                console.error('Failed to read file:', err);
            }
            
            document.body.removeChild(input);
        };
        
        document.body.appendChild(input);
        input.click();
    }
    
    // Initialize everything
    async function init() {
        console.log('[host] Initializing WebClaw...');
        
        try {
            // Load main thread WASM
            await initMainThreadWASM();
            
            // Initialize worker
            await initWorker();
            
            // Setup file handling
            setupFileHandling();
            
            console.log('[host] WebClaw initialized successfully');
            console.log('[host] Available APIs:', Object.keys(window.webclaw || {}));
            
            // Dispatch ready event
            window.dispatchEvent(new CustomEvent('webclaw:host-ready', {
                detail: { wasmLoaded: true, workerReady: true }
            }));
            
        } catch (err) {
            console.error('[host] Initialization failed:', err);
            throw err;
        }
    }
    
    // Expose public API
    window.webclawHost = {
        init,
        startStream,
        addMessage,
        abortStream,
        getState: () => ({ ...state }),
        
        // Helper to check if everything is ready
        isReady: () => state.wasmLoaded && state.workerReady,
        
        // Helper to get streaming status
        isStreaming: () => state.isStreaming
    };
    
    // Auto-initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        // DOM already loaded
        init();
    }
    
})();
