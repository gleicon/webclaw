# Phase 10: Browser-Based Local Model - Research

**Researched:** 2026-03-07
**Domain:** Browser-native LLM inference (WebGPU/WebLLM, WASM CPU fallback, Go provider integration)
**Confidence:** HIGH (core stack verified via official docs and npm) / MEDIUM (WASM CPU fallback path)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
*(CONTEXT.md contains no explicit ## Decisions section — all choices remain open for research recommendation)*

### Claude's Discretion
- Choice of inference library (WebLLM vs Transformers.js vs wllama)
- Which model(s) to pre-select as defaults
- Provider integration strategy (JS-side vs Go-side)
- Tool calling approach for local model
- Fallback behavior design

### Deferred Ideas (OUT OF SCOPE)
- Native Ollama/binary-based local inference (defeats browser-only goal)
- Mobile performance optimization (context notes phones likely too slow)
- Full fine-tuning of models for tool calling
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOCAL-01 | User can load and run a compact AI model (~100MB–2GB) in the browser via WebAssembly or WebGPU | WebLLM 0.2.81 supports Llama-3.2-3B-Instruct-q4f16_1-MLC (~2.2GB) and Gemma-2-2B-it-q4f16_1-MLC (~1.5GB); wllama handles GGUF on CPU |
| LOCAL-02 | Local model supports basic chat conversations without API keys or internet connection after download | WebLLM uses Cache API for model storage; works offline after first download |
| LOCAL-03 | Tool execution works with local model for simple automation tasks | WebLLM function calling is marked WIP; structured JSON output (JSON mode) is the reliable alternative for tool use with small models |
| LOCAL-04 | Seamless fallback: local model when offline, cloud providers when online | Router.RegisterProvider pattern already supports adding a "local" provider; offline detection via navigator.onLine |
| LOCAL-05 | Model can be downloaded and cached locally for subsequent sessions | WebLLM stores models in browser Cache API (CacheStorage); subsequent loads skip network, take 5–15s from disk |
</phase_requirements>

---

## Summary

Phase 10 adds a browser-native inference provider to WebClaw so users can run AI conversations completely offline. The established ecosystem choice is **WebLLM** (`@mlc-ai/web-llm` v0.2.81) — it uses WebGPU for GPU acceleration with a WASM compute fallback, implements an OpenAI-compatible streaming API, and stores models in the browser Cache API for offline use. WebGPU is now available in ~85% of desktop browsers (Chrome, Edge, Firefox 141+, Safari 26+), making it viable without being universal.

The critical architectural constraint for WebClaw is that the local model must run in a **separate JavaScript layer** from the Go WASM binary. WebLLM's engine is a JavaScript/TypeScript library; it cannot be imported into the Go provider package. The integration point is a new JS-side provider shim that speaks the same `postMessage` protocol already used by `worker.js`. The Go provider package gains a `LocalProvider` stub that delegates inference to this JS shim via the existing jsbridge pattern.

For browsers without WebGPU (Linux, some older machines), **wllama** (`@wllama/wllama` v2.3.7) provides a pure-WASM llama.cpp binding that runs GGUF models on CPU — slower (1–3 tok/s) but functional. The recommended architecture is: detect `navigator.gpu`, use WebLLM if available, fall back to wllama for CPU-only environments.

**Primary recommendation:** Use WebLLM as the primary inference engine in a dedicated Web Worker, with wllama as CPU fallback, integrated via a JS-side bridge shim that speaks the existing `worker.js` message protocol. Register as `"local"` vendor in the Go Router.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@mlc-ai/web-llm` | 0.2.81 | WebGPU-accelerated LLM inference in browser | 17.5k stars, OpenAI-compatible API, supports Llama/Gemma/Phi/Qwen, active development |
| `@wllama/wllama` | 2.3.7 | CPU-only WASM (llama.cpp) fallback | Active (published 10 days ago), GGUF format, auto multi-thread detection, worker-isolated |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `navigator.gpu` API | Browser built-in | Detect WebGPU availability | Feature-detect before choosing inference backend |
| Cache API (`caches`) | Browser built-in | Model file persistence across sessions | Used internally by WebLLM; also use for wllama GGUF files |
| OPFS (Origin Private File System) | Browser built-in | Alternative persistent storage | If Cache API eviction becomes a problem; experimental |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| WebLLM | Transformers.js | Transformers.js max practical model is ~1B params (ONNX format); no true chat models; better for embeddings only |
| wllama (CPU fallback) | Transformers.js text-generation | wllama supports full 3B models on CPU; Transformers.js limited to smaller models |
| Cache API (model storage) | IndexedDB | IndexedDB has per-row size limits and is not designed for binary blobs >100MB; Cache API is purpose-built for large response bodies |

**Installation:**
```bash
npm install @mlc-ai/web-llm @wllama/wllama
```

---

## Architecture Patterns

### Recommended Project Structure
```
static/
├── local-llm-worker.js      # New dedicated worker for WebLLM/wllama inference
├── webclaw-host.js          # Existing — add WebGPU detection + local provider init
├── worker.js                # Existing — unchanged (WASM agent loop worker)
internal/
├── provider/
│   ├── local.go             # New LocalProvider stub (delegates to JS via jsbridge)
│   └── router.go            # Existing — add "local" vendor registration
internal/
├── jsbridge/
│   └── local_bridge.go      # New — JS export for local model progress + inference
index.html                   # Add local model settings UI section
```

### Pattern 1: WebGPU Detection + Backend Selection (in webclaw-host.js)

**What:** Detect WebGPU at startup, select inference backend before instantiating local-llm-worker.
**When to use:** Always — runs once during app init.

```javascript
// Source: https://web.dev/articles/ai-chatbot-webllm + MDN WebGPU docs
async function detectLocalMLBackend() {
    if ('gpu' in navigator) {
        try {
            const adapter = await navigator.gpu.requestAdapter();
            if (adapter) return 'webgpu';  // WebLLM
        } catch (e) {
            // GPU present but failed — fall through
        }
    }
    return 'wasm';  // wllama
}
```

### Pattern 2: WebLLM Engine in Dedicated Worker

**What:** Instantiate WebLLM engine inside `local-llm-worker.js` so GPU work doesn't block UI.
**When to use:** WebGPU path.

```javascript
// Source: https://github.com/mlc-ai/web-llm (npm 0.2.81)
// local-llm-worker.js
import { MLCEngine } from "@mlc-ai/web-llm";

let engine = null;

self.onmessage = async function(event) {
    const { type, payload } = event.data;

    if (type === 'LOAD_MODEL') {
        engine = new MLCEngine();
        await engine.reload(payload.modelId, {
            initProgressCallback: (report) => {
                self.postMessage({ type: 'LOAD_PROGRESS', payload: report });
            }
        });
        self.postMessage({ type: 'MODEL_READY' });
    }

    if (type === 'CHAT_STREAM') {
        const stream = await engine.chat.completions.create({
            messages: payload.messages,
            stream: true,
            temperature: payload.temperature ?? 0.7,
        });
        for await (const chunk of stream) {
            const delta = chunk.choices[0]?.delta?.content ?? '';
            self.postMessage({ type: 'TOKEN', payload: { text: delta } });
        }
        self.postMessage({ type: 'COMPLETE' });
    }
};
```

### Pattern 3: wllama CPU Fallback in Worker

**What:** Use wllama for browsers without WebGPU, loading GGUF models from Cache API.
**When to use:** `navigator.gpu` unavailable.

```javascript
// Source: https://github.com/ngxson/wllama (npm 2.3.7)
// local-llm-worker.js (wasm branch)
import { Wllama } from "@wllama/wllama";

const wllama = new Wllama({
    'single-thread/wllama.wasm': '/vendor/wllama-single.wasm',
    'multi-thread/wllama.wasm':  '/vendor/wllama-multi.wasm',
});

await wllama.loadModelFromUrl(
    'https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf',
    { progressCallback: (pct) => self.postMessage({ type: 'LOAD_PROGRESS', payload: { progress: pct } }) }
);
```

### Pattern 4: Go LocalProvider Stub via jsbridge

**What:** LocalProvider implements `provider.Provider` but delegates completion to a JS callback.
**When to use:** When Go agent loop requests inference from the `local` vendor.

```go
// internal/provider/local.go
// Source: follows existing jsbridge pattern from internal/jsbridge/streaming.go
//go:build js && wasm

package provider

import (
    "context"
    "syscall/js"
)

// LocalProvider delegates to a JS-side WebLLM/wllama engine.
// Streaming tokens arrive via a JS callback registered in jsbridge/local_bridge.go.
type LocalProvider struct {
    modelID     string
    streamBridge js.Value // window.webclaw.local.streamChat
}

func (p *LocalProvider) Name() string { return "local" }
func (p *LocalProvider) MaxContextWindow(model string) int { return 4096 } // conservative

func (p *LocalProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
    ch := make(chan Token, 32)
    // Call JS bridge — tokens arrive via callback registered on js side
    // See jsbridge/local_bridge.go for callback registration
    go p.callJSStream(ctx, req, ch)
    return ch
}
```

### Pattern 5: Router "local" Vendor Registration

**What:** Register LocalProvider in the Router so `local/qwen2.5-1.5b` routes correctly.
**When to use:** During app startup when user has opted in to local model.

```go
// cmd/webclaw/main.go — add after existing provider registration
if config.LocalModel.Enabled {
    localProv := provider.NewLocalProvider(config.LocalModel.ModelID)
    router.RegisterProvider("local", localProv)
    // Signal JS side to initialize local-llm-worker
    js.Global().Get("webclaw").Get("local").Call("init", config.LocalModel.ModelID)
}
```

### Pattern 6: Progress Callback from JS to Go UI

**What:** JS local-llm-worker posts progress; webclaw-host.js forwards to main thread; Go exports update UI.
**When to use:** During model download and initial load from cache.

```javascript
// webclaw-host.js — forward local model progress to UI
localWorker.onmessage = function(e) {
    if (e.data.type === 'LOAD_PROGRESS') {
        const { progress, text } = e.data.payload;
        // Use existing event dispatch pattern from Phase 09
        window.dispatchEvent(new CustomEvent('webclaw:local-model-progress', {
            detail: { progress, text }
        }));
    }
};
```

### Anti-Patterns to Avoid

- **Importing `@mlc-ai/web-llm` into Go WASM:** Impossible — Go WASM cannot import ES modules. Always keep WebLLM in JS layer.
- **Storing model blobs in IndexedDB:** IndexedDB is not designed for multi-hundred-MB binary files. Use Cache API (WebLLM handles this) or OPFS.
- **Loading model on main thread:** Always use a dedicated Worker. Loading a 2GB model blocks the browser for 10–60 seconds.
- **Tool calling with JSON in function format on small models:** Smaller models (3B) have unreliable function-calling adherence. Use JSON mode + system prompt describing tools instead.
- **Sharing a single `local-llm-worker` with the existing `worker.js`:** The existing worker.js runs the WASM binary. WebLLM's engine is a separate JS runtime — they must be separate workers.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GPU-accelerated inference | Custom WebGPU shaders | `@mlc-ai/web-llm` | TVM compiled kernels, model quantization pipeline, shader compilation cache — years of work |
| CPU WASM inference | Custom llama.cpp WASM build | `@wllama/wllama` | Cross-browser threading, SIMD detection, GGUF parsing, KV cache — actively maintained |
| Model download + resume | Custom chunked fetch + reassembly | WebLLM's built-in download (Cache API) | Handles partial downloads, cache validation, progress reporting |
| Tokenization | Custom BPE tokenizer | Built into WebLLM/wllama | Model-specific tokenizer compiled into engine |
| Model format conversion | GGUF/ONNX/MLC conversion | Use pre-converted models from HuggingFace | Conversion requires Python + GPU, can't run in browser |

**Key insight:** The inference engine (WebLLM or wllama) is a complete runtime — treat it as a black-box inference service. WebClaw's job is the integration layer (provider stub, progress UI, config, fallback routing), not the inference math.

---

## Common Pitfalls

### Pitfall 1: Model Storage Eviction
**What goes wrong:** Browser evicts cached model files when disk space is low, especially on macOS/iOS Safari which aggressively evicts after 7 days of inactivity.
**Why it happens:** Cache API storage is "best effort" by default; no persistence guarantee.
**How to avoid:** Call `navigator.storage.persist()` to request persistent storage. Show user a warning if denied. Check `navigator.storage.estimate()` before download.
**Warning signs:** Users report model re-downloading on every visit; Safari on iOS consistently fails.

### Pitfall 2: WebGPU Unavailable but Detected as Available
**What goes wrong:** `navigator.gpu` exists but `requestAdapter()` returns null (no capable GPU, or driver too old).
**Why it happens:** WebGPU API is present but adapter request fails at runtime.
**How to avoid:** Always await `requestAdapter()` and check for null before declaring WebGPU available. Catch exceptions from `requestAdapter()`.
**Warning signs:** Engine initialization throws during `engine.reload()`.

### Pitfall 3: Two Workers Competing for Indexeddb
**What goes wrong:** Go WASM worker and local-llm-worker both try to write to the same IndexedDB stores.
**Why it happens:** Both workers share the same origin's IndexedDB.
**How to avoid:** Local model state (config, cache metadata) lives in JS only (`webclaw_local_model_config` localStorage key). No Go <-> local model coordination through IndexedDB.
**Warning signs:** IndexedDB transaction conflicts, random state corruption.

### Pitfall 4: Cold Start UX
**What goes wrong:** User sends a message, waits 5–15 seconds with no feedback while model initializes from cache.
**Why it happens:** Model must be loaded from Cache API into GPU VRAM each session — no lazy loading.
**How to avoid:** Pre-load the model on app startup (not on first message). Show a persistent "Local AI: Loading..." status indicator. Only expose "local" as provider option once model is ready.
**Warning signs:** Users think app is broken; no visible loading feedback.

### Pitfall 5: Tool Calling Unreliability on 3B Models
**What goes wrong:** 3B parameter models sometimes produce malformed JSON for tool calls, breaking the agent loop.
**Why it happens:** Small models lack the instruction-following precision of 70B+ models.
**How to avoid:** For LOCAL-03 (tool execution with local model), use **JSON mode** (`response_format: { type: "json_object" }`) plus a system prompt that describes tool schemas. Parse the structured JSON output into tool calls in JS before forwarding to Go agent loop.
**Warning signs:** Agent loop receives tool calls with missing required fields, null values, or non-JSON text.

### Pitfall 6: model_id Mismatch Between WebLLM and Config
**What goes wrong:** User config stores a model ID like `"llama-3.2-3b"` but WebLLM requires exact MLC format `"Llama-3.2-3B-Instruct-q4f16_1-MLC"`.
**Why it happens:** Model IDs in WebLLM are exact strings registered in `prebuiltAppConfig.model_list`.
**How to avoid:** Store the full WebLLM model ID in config. Provide a dropdown in settings populated from `webllm.prebuiltAppConfig.model_list` filtered to models <3GB. Never let users type model IDs manually.
**Warning signs:** `CreateMLCEngine` throws "model not found" errors.

---

## Code Examples

Verified patterns from official sources:

### WebLLM Engine Init with Progress (Main Thread — for settings preview)
```javascript
// Source: https://www.npmjs.com/package/@mlc-ai/web-llm (v0.2.81)
import { CreateMLCEngine } from "@mlc-ai/web-llm";

const engine = await CreateMLCEngine(
    "Llama-3.2-3B-Instruct-q4f16_1-MLC",
    {
        initProgressCallback: (report) => {
            console.log(`[local] ${report.text} (${Math.round(report.progress * 100)}%)`);
        }
    }
);
```

### WebLLM Streaming Chat Completion
```javascript
// Source: https://webllm.mlc.ai/docs/user/basic_usage.html
const stream = await engine.chat.completions.create({
    messages: [
        { role: "system", content: systemPrompt },
        { role: "user", content: userMessage }
    ],
    stream: true,
    temperature: 0.7,
    max_tokens: 512,
});

for await (const chunk of stream) {
    const delta = chunk.choices[0]?.delta?.content ?? '';
    onToken(delta);  // forward to Go worker via postMessage
}
```

### WebLLM JSON Mode for Structured Tool Output
```javascript
// Source: WebLLM OpenAI-compatible API (function calling WIP, JSON mode stable)
// Use this for LOCAL-03 tool calling instead of native function calling
const response = await engine.chat.completions.create({
    messages: [
        {
            role: "system",
            content: `You are a helpful assistant. When you need to use a tool, respond ONLY with valid JSON matching this schema: {"tool": "<tool_name>", "args": {...}}`
        },
        { role: "user", content: userMessage }
    ],
    response_format: { type: "json_object" },
    temperature: 0.1,  // Low temp for structured output
});
```

### wllama CPU Fallback Model Loading
```javascript
// Source: https://github.com/ngxson/wllama (npm @wllama/wllama v2.3.7)
import { Wllama } from "@wllama/wllama";

const wllama = new Wllama({
    'single-thread/wllama.wasm': '/vendor/wllama-single.wasm',
    'multi-thread/wllama.wasm': '/vendor/wllama-multi.wasm',
});

await wllama.loadModelFromUrl(modelUrl, {
    progressCallback: ({ loaded, total }) => {
        self.postMessage({ type: 'LOAD_PROGRESS', payload: { progress: loaded / total } });
    }
});

const result = await wllama.createCompletion(prompt, {
    nPredict: 512,
    temperature: 0.7,
    onNewToken: (token, piece) => {
        self.postMessage({ type: 'TOKEN', payload: { text: piece } });
    }
});
```

### WebGPU Detection with Graceful Fallback
```javascript
// Source: MDN WebGPU API + webgpu-webllm-app pattern
async function selectMLBackend() {
    if (!('gpu' in navigator)) return 'cpu';
    try {
        const adapter = await navigator.gpu.requestAdapter();
        if (!adapter) return 'cpu';
        return 'webgpu';
    } catch (e) {
        return 'cpu';
    }
}

// In webclaw-host.js startup:
const backend = await selectMLBackend();
window.webclaw = window.webclaw || {};
window.webclaw.localML = { backend, engine: null };
```

### Go LocalProvider → JS Bridge Pattern (follows existing jsbridge convention)
```go
//go:build js && wasm
// internal/jsbridge/local_bridge.go

package jsbridge

import "syscall/js"

// RegisterLocalModelBridge exposes Go callbacks for the local model JS worker.
// Pattern mirrors RegisterOAuthBridge from oauth_bridge.go.
func RegisterLocalModelBridge(onToken func(text string), onComplete func(), onError func(err string)) {
    webclaw := js.Global().Get("webclaw")
    obj := js.Global().Get("Object").New()

    obj.Set("onToken", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        if len(args) > 0 {
            onToken(args[0].String())
        }
        return nil
    }))

    obj.Set("onComplete", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        onComplete()
        return nil
    }))

    webclaw.Set("local", obj)
}
```

---

## Recommended Models (Verified Model IDs)

| WebLLM Model ID | Size | Quality | Use Case |
|-----------------|------|---------|----------|
| `Llama-3.2-3B-Instruct-q4f16_1-MLC` | ~2.2GB | Best chat quality in this tier | Default for WebGPU path |
| `Gemma-2-2B-it-q4f16_1-MLC` | ~1.5GB | Good multilingual, smaller download | Lower-end GPU users |
| `Phi-3.5-mini-instruct-q4f16_1-MLC` | ~2.5GB | Strong reasoning, Microsoft-tuned | Power users |
| `Qwen2.5-1.5B-Instruct-q4f16_1-MLC` | ~1.0GB | Fast, minimal RAM | CPU-class GPU or quick startup |

**wllama CPU path (GGUF from HuggingFace):**
| Model | Size | Speed (CPU) |
|-------|------|-------------|
| Qwen2.5-1.5B-Instruct Q4_K_M | ~1.0GB | ~3–5 tok/s (4-core) |
| SmolLM2-360M-Instruct Q4_K_M | ~230MB | ~8–12 tok/s | Fast but limited capability |

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| WebGPU Chrome-only (~30%) | WebGPU in all major browsers (~85% desktop) | 2025 Q4 | WASM-only fallback less critical |
| Models stored in IndexedDB | Cache API (CacheStorage) for model blobs | 2024 | No more IndexedDB blob size issues |
| Manual model loading from JS | WebLLM `CreateMLCEngine()` with auto-cache | 2024 | Download + cache + reload handled by library |
| Function calling not available | WebLLM function calling WIP (prelim support) | 2025 | JSON mode still recommended for 3B models |

**Deprecated/outdated:**
- Storing model files in IndexedDB: replaced by Cache API (WebLLM's default, purpose-built for large responses)
- Running WebLLM on main thread: always use dedicated Web Worker
- Transformers.js for chat models: limited to <1B, insufficient for meaningful chat

---

## Open Questions

1. **Model Context Window vs WebClaw History**
   - What we know: 3B models work well at 4K tokens; WebClaw cloud providers use 128K tokens
   - What's unclear: Does the existing conversation summarizer (20-message threshold) also work for local models, or does 4K context force earlier summarization?
   - Recommendation: Expose `MaxContextWindow()` returning 4096 from LocalProvider; the existing summarizer will trigger earlier and handle it automatically.

2. **Tool Calling Reliability at 3B Scale**
   - What we know: WebLLM function calling is marked WIP; JSON mode is stable; small models are unreliable with complex schemas
   - What's unclear: Which of WebClaw's 6 tools (web_fetch, web_search, memory_store, memory_search, file ops) are realistic with a 3B model in JSON mode?
   - Recommendation: Plan a restricted tool set for local provider (memory_search and memory_store only); web_fetch and web_search require reliable JSON formatting under pressure.

3. **Cross-Origin Model Caching**
   - What we know: Models cached by WebLLM (Cache API) are origin-scoped — cannot be shared between sites
   - What's unclear: If user deploys WebClaw on custom domain vs localhost vs static file, they re-download each model per origin
   - Recommendation: Document this limitation in settings UI. No code fix needed.

4. **Safari Storage Persistence**
   - What we know: Safari evicts Cache API storage for inactive origins; iOS has tighter limits
   - What's unclear: Whether Safari 26 (which ships WebGPU) also improved Cache API persistence guarantees
   - Recommendation: Always call `navigator.storage.persist()` at model load time; detect Safari and warn about potential eviction.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Playwright (existing in `test/`) |
| Config file | `playwright.config.js` (exists) |
| Quick run command | `npx playwright test test/10-local-model.spec.js --project=chromium` |
| Full suite command | `npx playwright test --project=chromium` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LOCAL-01 | WebGPU detection returns "webgpu" or "wasm" | smoke | `npx playwright test test/10-local-model.spec.js -k "backend detection" --project=chromium` | Wave 0 |
| LOCAL-01 | Model selector shows filtered model list in settings UI | smoke | `npx playwright test test/10-local-model.spec.js -k "model list" --project=chromium` | Wave 0 |
| LOCAL-02 | After model load, chat works without network (service worker intercept) | integration | `npx playwright test test/10-local-model.spec.js -k "offline chat" --project=chromium` | Wave 0 |
| LOCAL-03 | Local model returns valid JSON for memory_search tool call | unit (JS) | `node test/unit/local-tool-json.test.js` | Wave 0 |
| LOCAL-04 | Provider router routes "local/..." to LocalProvider stub | unit (Go) | N/A — Go tests not applicable in WASM context; manual verification | manual-only |
| LOCAL-04 | Fallback: when local model unavailable, cloud provider used | smoke | `npx playwright test test/10-local-model.spec.js -k "cloud fallback" --project=chromium` | Wave 0 |
| LOCAL-05 | Model persists in Cache API after page reload (no re-download) | integration | `npx playwright test test/10-local-model.spec.js -k "cache persistence" --project=chromium` | Wave 0 |

**Note on LOCAL-02 offline test:** Playwright's `page.route()` can intercept and fail all network requests to simulate offline. WebLLM model files served from Cache API will still load. This is the recommended approach rather than actual network disconnection.

**Note on LOCAL-04 Go routing:** The LocalProvider Go code cannot be unit-tested outside the WASM build target. Verification is through the Playwright smoke test that exercises the full path.

### Sampling Rate
- **Per task commit:** `npx playwright test test/10-local-model.spec.js -k "smoke" --project=chromium`
- **Per wave merge:** `npx playwright test --project=chromium`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `test/10-local-model.spec.js` — covers LOCAL-01 (backend detection, model list), LOCAL-02 (offline chat), LOCAL-04 (cloud fallback), LOCAL-05 (cache persistence)
- [ ] `test/unit/local-tool-json.test.js` — covers LOCAL-03 (JSON mode tool output parsing)
- [ ] `static/local-llm-worker.js` — does not exist; must be created in Wave 0
- [ ] `internal/provider/local.go` — does not exist; must be created in Wave 0
- [ ] `internal/jsbridge/local_bridge.go` — does not exist; must be created in Wave 0
- [ ] npm install: `npm install @mlc-ai/web-llm @wllama/wllama` — packages not yet in package.json

---

## Sources

### Primary (HIGH confidence)
- `@mlc-ai/web-llm` npm page (v0.2.81) — version, API surface, model support
- https://webllm.mlc.ai/docs/user/basic_usage.html — CreateMLCEngine, streaming API, model IDs
- https://web.dev/articles/ai-chatbot-webllm — Cache API storage, offline capability, model sizes
- https://github.com/ngxson/wllama — wllama v2.3.7, CPU WASM inference, GGUF support
- `@wllama/wllama` npm page (v2.3.7) — current version confirmation
- MDN Web API docs — WebGPU requestAdapter, Cache API, navigator.storage.persist()

### Secondary (MEDIUM confidence)
- https://localaimaster.com/blog/webllm-browser-ai-guide — ~65% WebGPU browser coverage (WebLLM's own stat; MDN/browser vendor announcements suggest ~85% desktop by late 2025)
- WebGPU browser support news (zircon.tech, webgpu.com) — Firefox 141+, Safari 26+ shipping WebGPU
- https://rxdb.info/articles/indexeddb-max-storage-limit.html — IndexedDB limitations vs Cache API for large files
- WebLLM GitHub issue #683 — model list confirmation for Llama-3.2-3B and Phi-3.5-mini model IDs

### Tertiary (LOW confidence)
- Token speed benchmarks (3–7 tok/s on WebGPU) — sourced from blog posts, not official benchmarks; hardware varies significantly
- WebLLM function calling "WIP" status — from search results and GitHub issue commentary; may have improved since research date

---

## Metadata

**Confidence breakdown:**
- Standard stack (WebLLM + wllama): HIGH — npm package versions verified, official docs read
- Model IDs: HIGH — confirmed via prebuiltAppConfig documentation and multiple search sources
- Architecture patterns (worker isolation, jsbridge): HIGH — follows existing WebClaw patterns exactly
- Tool calling via JSON mode: MEDIUM — WebLLM JSON mode confirmed stable; reliability on 3B models is empirical
- Token speed benchmarks: LOW — blog sources only, varies by hardware; treat as directional only
- Safari Cache API persistence: MEDIUM — documented behavior, exact Safari 26 changes unclear

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (WebLLM is actively developed; check npm for version updates before planning)
