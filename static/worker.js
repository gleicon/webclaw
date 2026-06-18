// worker.js — Web Worker for running agent loop without blocking main thread
// This worker loads the WASM module and handles streaming LLM communication

// Gemini Nano bridge — wraps Chrome's built-in LanguageModel API into a
// callback-friendly surface callable from Go via syscall/js.
// Chunks from promptStreaming are cumulative — we slice each delta manually.
self.webclaw = self.webclaw || {};
self.webclaw.geminiNano = {
    async streamPrompt(systemPrompt, historyJSON, userMsg, onToken, onDone, onError) {
        let session = null;
        try {
            if (typeof LanguageModel === 'undefined') {
                onError('Chrome LanguageModel API not available — enable via Origin Trial or chrome://flags');
                return;
            }

            let history = [];
            try { history = JSON.parse(historyJSON); } catch (_) {}

            const opts = {};
            if (systemPrompt) opts.systemPrompt = systemPrompt;
            if (history.length > 0) opts.initialPrompts = history;

            session = await LanguageModel.create(opts);

            let prevLen = 0;
            const stream = session.promptStreaming(userMsg);
            for await (const chunk of stream) {
                const delta = chunk.slice(prevLen);
                prevLen = chunk.length;
                if (delta) onToken(delta);
            }
            onDone();
        } catch (e) {
            onError(e.message || String(e));
        } finally {
            if (session) { try { session.destroy(); } catch (_) {} }
        }
    }
};

let wasmModule = null;
let wasmInstance = null;
let goRuntime = null;
let isStreaming = false;
let abortController = null;

// Message types between main thread and worker
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

// Handle messages from main thread
self.onmessage = function(event) {
    const { type, payload } = event.data;
    
    switch (type) {
        case MSG_TYPES.INIT_WASM:
            handleInitWasm(payload);
            break;
        case MSG_TYPES.START_STREAM:
            handleStartStream(payload);
            break;
        case MSG_TYPES.ADD_MESSAGE:
            handleAddMessage(payload);
            break;
        case MSG_TYPES.ABORT_STREAM:
            handleAbortStream();
            break;
        default:
            console.error('[worker] Unknown message type:', type);
    }
};

// Initialize WASM module
async function handleInitWasm(payload) {
    try {
        if (!payload || !payload.wasmBinary) {
            throw new Error('WASM binary not provided');
        }
        
        console.log('[worker] Initializing WASM...');
        
        // Import wasm_exec.js functionality
        importScripts('wasm_exec.js');
        
        goRuntime = new Go();
        
        // Polyfill for browsers without instantiateStreaming
        if (!WebAssembly.instantiateStreaming) {
            WebAssembly.instantiateStreaming = async (resp, importObject) => {
                const source = await (await resp).arrayBuffer();
                return await WebAssembly.instantiate(source, importObject);
            };
        }
        
        // Instantiate WASM from provided binary
        const wasmBuffer = new Uint8Array(payload.wasmBinary);
        const wasmResponse = new Response(wasmBuffer, {
            headers: { 'Content-Type': 'application/wasm' }
        });
        
        const result = await WebAssembly.instantiateStreaming(
            wasmResponse,
            goRuntime.importObject
        );
        
        wasmInstance = result.instance;
        
        // Start Go runtime (this runs main.go asynchronously)
        goRuntime.run(wasmInstance);
        
        // Wait for WASM to signal readiness via global callback
        await waitForWasmReady();
        
        console.log('[worker] WASM initialized successfully');
        
        self.postMessage({
            type: MSG_TYPES.WASM_READY,
            payload: { success: true }
        });
        
    } catch (error) {
        console.error('[worker] WASM initialization failed:', error);
        self.postMessage({
            type: MSG_TYPES.WASM_ERROR,
            payload: { 
                error: error.message,
                stack: error.stack 
            }
        });
    }
}

// Wait for WASM to signal it's ready
function waitForWasmReady() {
    return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            reject(new Error('WASM initialization timeout (30s)'));
        }, 30000);
        
        // Check periodically for webclaw global
        const checkInterval = setInterval(() => {
            if (self.webclaw && self.webclaw.workerBridge) {
                clearInterval(checkInterval);
                clearTimeout(timeout);
                
                // Register callbacks for streaming
                registerStreamingCallbacks();
                resolve();
            }
        }, 100);
    });
}

// Register callbacks that WASM will call during streaming
function registerStreamingCallbacks() {
    if (!self.webclaw || !self.webclaw.workerBridge) {
        console.error('[worker] workerBridge not available');
        return;
    }
    
    // Register all callbacks using the registerCallback pattern
    // This properly wires them to the Go WorkerBridge struct
    
    // Called by WASM when a token is received from LLM
    self.webclaw.workerBridge.registerCallback('onToken', function(token) {
        self.postMessage({
            type: MSG_TYPES.TOKEN,
            payload: { token }
        });
    });
    
    // Called by WASM when stream completes (naturally or by error)
    self.webclaw.workerBridge.registerCallback('onComplete', function(result) {
        console.log('[worker] onComplete called, resetting isStreaming');
        isStreaming = false;
        self.postMessage({
            type: MSG_TYPES.COMPLETE,
            payload: result || { success: true }
        });
    });
    
    // Called by WASM when an error occurs
    self.webclaw.workerBridge.registerCallback('onError', function(error) {
        console.log('[worker] onError called, resetting isStreaming');
        isStreaming = false;
        self.postMessage({
            type: MSG_TYPES.ERROR,
            payload: {
                error: error.message || error,
                code: error.code || 'UNKNOWN'
            }
        });
    });

    // Called by WASM when a tool starts or completes
    self.webclaw.workerBridge.registerCallback('onToolEvent', function(toolName, status, summary, full) {
        self.postMessage({
            type: MSG_TYPES.TOOL_EVENT,
            payload: { toolName, status, summary, full }
        });
    });

    console.log('[worker] Streaming callbacks registered');
}

// Start a streaming LLM request
async function handleStartStream(payload) {
    try {
        if (!self.webclaw || !self.webclaw.workerBridge) {
            throw new Error('WASM not initialized');
        }
        
        if (isStreaming) {
            throw new Error('Stream already in progress');
        }
        
        isStreaming = true;
        
        console.log('[worker] Starting stream...');
        
        // Call into WASM to start the agent loop
        // This spawns a goroutine that will call the provider
        self.webclaw.workerBridge.startStream(payload);
        
        self.postMessage({
            type: MSG_TYPES.STREAM_STARTED,
            payload: { timestamp: Date.now() }
        });
        
    } catch (error) {
        isStreaming = false;
        console.error('[worker] Failed to start stream:', error);
        self.postMessage({
            type: MSG_TYPES.ERROR,
            payload: { 
                error: error.message,
                code: 'START_FAILED'
            }
        });
    }
}

// Add a message to the conversation history
function handleAddMessage(payload) {
    try {
        if (!self.webclaw || !self.webclaw.workerBridge) {
            throw new Error('WASM not initialized');
        }
        
        const { role, content } = payload;
        
        console.log('[worker] Adding message:', role);
        
        self.webclaw.workerBridge.addMessage(role, content);
        
    } catch (error) {
        console.error('[worker] Failed to add message:', error);
        self.postMessage({
            type: MSG_TYPES.ERROR,
            payload: { 
                error: error.message,
                code: 'ADD_MESSAGE_FAILED'
            }
        });
    }
}

// Abort the current stream
function handleAbortStream() {
    try {
        if (!isStreaming) {
            console.log('[worker] No stream to abort');
            return;
        }
        
        if (!self.webclaw || !self.webclaw.workerBridge) {
            throw new Error('WASM not initialized');
        }
        
        console.log('[worker] Aborting stream...');
        
        self.webclaw.workerBridge.abortStream();
        
        isStreaming = false;
        
        self.postMessage({
            type: MSG_TYPES.STREAM_ABORTED,
            payload: { timestamp: Date.now() }
        });
        
    } catch (error) {
        console.error('[worker] Failed to abort stream:', error);
        self.postMessage({
            type: MSG_TYPES.ERROR,
            payload: { 
                error: error.message,
                code: 'ABORT_FAILED'
            }
        });
    }
}

// Handle errors that bubble up to the worker
self.onerror = function(error) {
    console.error('[worker] Unhandled error:', error);
    self.postMessage({
        type: MSG_TYPES.ERROR,
        payload: { 
            error: error.message || 'Unknown worker error',
            code: 'WORKER_ERROR',
            stack: error.stack
        }
    });
};

// Signal worker is loaded
console.log('[worker] Worker script loaded, waiting for INIT_WASM');
