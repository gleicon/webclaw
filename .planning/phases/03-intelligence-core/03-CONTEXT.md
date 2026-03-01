# Phase 3: Intelligence Core - Context

**Gathered:** 2026-03-01
**Status:** Ready for planning
**Source:** Research + Requirements

<domain>
## Phase Boundary

Phase 3 implements the intelligence core of WebClaw — the ability to:
1. Route LLM calls to providers (Anthropic, OpenAI, OpenRouter)
2. Stream responses token-by-token via Web Worker
3. Manage conversation context with automatic summarization
4. Store and retrieve memories with hybrid vector+BM25 search
5. Handle storage quotas with memory archival

This phase transforms WebClaw from a static configuration system into a working AI agent.

## Key Subsystems

### LLM Provider System (PROV-01 to PROV-05)
- Route vendor/model-id strings to correct provider
- Implement SSE streaming for all providers
- Handle provider failover with exponential backoff
- Support Anthropic, OpenAI, and OpenRouter

### Agent Loop (AGNT-01 to AGNT-04)
- Turn-based conversation execution
- Context assembly (system prompt + identity + history)
- Web Worker for non-blocking streaming
- Automatic summarization when thresholds exceeded

### Memory System (MEM-01 to MEM-05)
- Vector embedding storage in IndexedDB
- Hybrid search: cosine similarity (0.7) + BM25 (0.3)
- Memory archival at 80% quota
- Provider-based embeddings (OpenAI text-embedding-3-small)

</domain>

<decisions>
## Implementation Decisions

### Locked Decisions (from research)
- **Provider APIs:** Use SSE streaming for all providers
- **Embeddings:** Use provider APIs (not local) for v1
- **Search:** Custom BM25+cosine (no external library)
- **Streaming:** Web Worker architecture (non-blocking)
- **Summarization:** Progressive sliding window approach

### Claude's Discretion
- Provider interface design (common interface for all providers)
- Web Worker communication protocol (postMessage vs shared memory)
- Memory storage schema (Float32Array in IndexedDB)
- BM25 term frequency calculation (custom implementation)
- Conversation history data structure (slices vs linked list)

</decisions>

<specifics>
## Specific Requirements

### Provider Routing (PROV-01)
- Parse vendor/model-id format
- Map to correct provider implementation
- No net/http imports (use syscall/js fetch bridge)

### Streaming (PROV-03, AGNT-04)
- Token-by-token streaming to UI
- Run in Web Worker to avoid blocking main thread
- SSE parsing for Anthropic and OpenAI formats

### Summarization (AGNT-02, AGNT-03)
- Trigger: 20 messages OR 75% of context window
- Strategy: Progressive summarization
- Output: Compressed history replaces full history

### Memory Search (MEM-02)
- Hybrid scoring: 0.7 * cosine_similarity + 0.3 * bm25_score
- Return ranked results
- Merge and deduplicate

### Quota Management (MEM-05)
- Monitor IndexedDB usage
- LRU eviction at 80% threshold
- Archive to compressed storage

</specifics>

<deferred>
## Deferred Ideas

- Local embedding models (deferred to v2)
- Full-text search library (Orama/deferred to v2)
- Multi-modal memory (images/audio deferred)
- Distributed memory (future architecture)

</deferred>

---

*Phase: 03-intelligence-core*
*Context gathered: 2026-03-01*
