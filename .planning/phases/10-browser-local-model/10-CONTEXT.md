# Phase 10: Browser-Based Local Model

## Goal

Enable WebClaw to run a lightweight local AI model directly in the browser for offline/basic chat and automation capabilities

## Context

### Problem Statement

WebClaw currently requires internet connectivity and API keys for AI providers (Anthropic, OpenAI, OpenRouter). Users want:

- Offline capability for basic tasks
- Privacy-first conversations that don't leave the browser
- Lower latency for simple queries
- Reduced dependency on paid API services

### Technical Constraints

#### Browser Environment

- WebAssembly (WASM) compilation target
- Limited memory (typically 2-4GB available)
- No file system access without bridge binary
- IndexedDB for persistent storage (model caching)
- Web Workers for background processing

#### Model Requirements

- Must fit in browser memory after WebClaw's ~50MB WASM + JS footprint
- Target: 100-500MB quantized model
- Acceptable latency: <5s for first token, streaming thereafter
- Minimum viable: 3B-7B parameter model (quantized to 4-bit)

#### Candidate Models

1. **Llama 3.2 3B** - Meta's compact instruction-tuned model (~2GB 4-bit)
2. **Qwen 2.5 3B** - Alibaba's multilingual model (~2GB 4-bit)
3. **Phi-3 Mini** - Microsoft's 3.8B model (~2.5GB 4-bit)
4. **Gemma 2 2B** - Google's lightweight model (~1.5GB 4-bit)

### Prior Art

#### WebLLM (Apache 2.0)

- https://github.com/mlc-ai/web-llm
- Loads models via WebGPU
- Uses Apache TVM runtime compiled to WASM
- Supports Llama, Mistral, Phi, Qwen families
- ~3-7B models run at acceptable speeds on modern GPUs
- Can run on CPU via WASM fallback (slower)

#### Transformers.js (Apache 2.0)

- https://huggingface.co/docs/transformers.js
- ONNX Runtime Web for inference
- Smaller models only (typically <1B parameters)
- Good for embeddings, classification, summarization
- Limited chat capability

#### Ollama + Web Interface

- Requires local binary (defeats browser-native goal)
- Can communicate via WebSocket
- Not suitable for true browser-only deployment

### Integration Points

#### Provider Router

The local model should integrate with the existing provider router as a new provider type:

- Provider name: `local` or `browser-local`
- Model ID: e.g., `local/llama-3.2-3b`
- Same streaming interface as cloud providers
- Tool support via existing tool registry

#### Memory System

- Reuse existing memory store for context
- May need reduced context window (4K-8K tokens vs 128K)
- BM25 + embeddings for memory search still work

#### Configuration

- Add `local_models` section to config
- Store model metadata (size, capability, download URL)
- Cache location: IndexedDB
- Optional: Preload on first run

### Open Questions

1. **Performance Baseline**: What's the minimum acceptable token/second on mid-tier laptops?
2. **Model Selection**: Which model provides best chat quality at ~2-3GB size?
3. **WebGPU Support**: How to gracefully fall back to WASM when WebGPU unavailable?
4. **Warm-up Time**: Model loading from IndexedDB can take 5-10 seconds - acceptable UX?
5. **Tool Calling**: Can smaller models reliably use tools? Fine-tuning needed?

### Success Criteria

- [ ] Local model provider registered with router
- [ ] Model download and caching in IndexedDB
- [ ] Basic chat works offline (no API calls)
- [ ] Tool execution functional with local model
- [ ] Auto-fallback to cloud when local unavailable
- [ ] <10s load time from cache on subsequent visits
- [ ] 3+ tokens/second generation speed on modern hardware

### Potential Blockers

1. WebGPU browser support (~70% of users as of 2024)
2. Model licensing (Llama requires acceptance, others Apache/MIT)
3. Storage quota (models may exceed 200MB IndexedDB limit in Safari)
4. Mobile performance (likely too slow on phones)

### Files to Modify

- `cmd/webclaw/main.go` - Add local provider registration
- `internal/provider/` - New local provider implementation
- `internal/config/` - Local model configuration
- `internal/jsbridge/` - Model download progress callbacks
- `index.html` - Settings UI for model management
- `webclaw-host.js` - WebGPU detection and model loading

## Dependencies

### Phase 6: Real Agent Loop

Required for:

- Agent loop structure
- Tool registry integration
- Memory system

### Phase 7/7a: File Operations

Useful for:

- Storing downloaded models
- Model management UI

### Optional: Phase 8

Distribution improvements helpful for:

- Larger bundle size accommodation
- CDN hosting for model files

## Research Needed

1. Benchmark WebLLM vs Transformers.js for chat quality
2. Test IndexedDB storage limits across browsers
3. Evaluate model quantization approaches
4. Design provider failover when local model fails
5. Create minimal viable model testing protocol

## Notes

This phase extends WebClaw's "browser-native, no server" philosophy to the AI layer itself. While local models won't match cloud quality, they enable:

- Private conversations
- Offline productivity
- Zero API costs for basic tasks
- Better resilience when connectivity poor

The goal is "good enough" local inference for 80% of simple use cases, with seamless upgrade to cloud for complex tasks.
