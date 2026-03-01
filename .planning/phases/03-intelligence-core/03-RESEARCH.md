# Phase 3: Intelligence Core - Research Document

**Date:** 2026-03-01  
**Status:** Research Complete  
**Phase:** 3 - Intelligence Core  
**Requirements Covered:** PROV-01 through PROV-05, AGNT-01 through AGNT-04, MEM-01 through MEM-05

---

## Executive Summary

This research document provides technical patterns, library recommendations, and implementation guidance for Phase 3 of WebClaw: Intelligence Core. The phase focuses on three major subsystems:

1. **LLM Provider Routing** (PROV-*) - Routing vendor/model-id to correct APIs
2. **Agent Loop** (AGNT-*) - Managing conversation context and streaming
3. **Memory System** (MEM-*) - Vector storage, hybrid search, and quota management

---

## 1. LLM Provider API Patterns

### 1.1 Provider Architecture Overview

All three target providers (Anthropic, OpenAI, OpenRouter) use **Server-Sent Events (SSE)** for streaming responses. The architecture differs in event structure:

```
┌─────────────────────────────────────────────────────────────┐
│                    Provider Routing                          │
├─────────────────────────────────────────────────────────────┤
│  vendor/model-id  →  Provider  →  API Endpoint  →  Format   │
├─────────────────────────────────────────────────────────────┤
│  anthropic/*      →  Anthropic  →  /v1/messages   →  Claude  │
│  openai/*         →  OpenAI     →  /v1/chat       →  GPT     │
│  openrouter/*     →  OpenRouter   →  /v1/chat     →  Unified │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Anthropic Claude API Patterns

**Endpoint:** `POST https://api.anthropic.com/v1/messages`

**Key Characteristics:**
- Uses Messages API (not legacy completions)
- SSE format with event types: `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`
- Requires headers: `anthropic-version`, `x-api-key`, `content-type`

**Streaming Response Format:**
```
event: message_start
data: {"type":"message_start","message":{"id":"msg_01...","role":"assistant"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: message_stop
data: {"type":"message_stop"}
```

**Implementation Pattern:**
```javascript
// JS fetch bridge for Go WASM
async function jsFetchAnthropic(url, apiKey, body) {
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': apiKey,
      'anthropic-version': '2023-06-01'
    },
    body: JSON.stringify({
      model: body.model,
      max_tokens: body.max_tokens,
      messages: body.messages,
      stream: true
    })
  });
  
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    
    const chunk = decoder.decode(value, { stream: true });
    // Parse SSE events and call Go callback
    parseSSEEvents(chunk, goCallback);
  }
}
```

### 1.3 OpenAI API Patterns

**Endpoint:** `POST https://api.openai.com/v1/chat/completions`

**Key Characteristics:**
- OpenAI-compatible format (also used by OpenRouter)
- SSE format with data: prefixed JSON
- Requires headers: `Authorization: Bearer {token}`, `Content-Type`

**Streaming Response Format:**
```
data: {"id":"chatcmpl-...","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]
```

### 1.4 OpenRouter API Patterns

**Endpoint:** `POST https://openrouter.ai/api/v1/chat/completions`

**Key Characteristics:**
- OpenAI-compatible format
- Additional headers: `HTTP-Referer`, `X-Title`
- Supports provider routing via `provider` object
- Unified interface for 100+ models

**Provider Selection Options:**
```json
{
  "model": "anthropic/claude-sonnet-4-5",
  "provider": {
    "order": ["Anthropic", "AWS Bedrock"],
    "allow_fallbacks": true
  },
  "stream": true
}
```

### 1.5 Provider Failover Strategy (PROV-04)

**Recommended Pattern:** Exponential backoff with fallback chain

```
┌────────────────────────────────────────────────────────────┐
│                    Provider Failover Flow                    │
├────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Attempt Primary Provider                                 │
│     ↓                                                        │
│  2. If 5xx/rate-limit: Retry with exponential backoff      │
│     - Attempt 1: immediate                                   │
│     - Attempt 2: 1s delay                                    │
│     - Attempt 3: 2s delay                                    │
│     ↓                                                        │
│  3. If all retries fail: Switch to Fallback Model          │
│     ↓                                                        │
│  4. Notify user of model switch (non-blocking)              │
│                                                              │
└────────────────────────────────────────────────────────────┘
```

**Implementation:**
```go
// In Go WASM - called via JS bridge
type ProviderChain struct {
    Primary   ProviderConfig
    Fallbacks []ProviderConfig
    MaxRetries int
    BackoffMs  int
}

func (pc *ProviderChain) CallWithRetry(ctx context.Context, req CompletionRequest) (<-chan Token, error) {
    // Implementation: retry with backoff, then fall through to next provider
}
```

---

## 2. Web Worker Architecture for Streaming

### 2.1 Why Web Workers Are Essential

**Requirements:** AGNT-04 (Agent loop runs in Web Worker)

- LLM streaming can take 10-60 seconds
- Token-by-token processing on main thread blocks UI
- Go WASM runs synchronously in the main thread by default
- Web Workers provide true parallelism for streaming I/O

### 2.2 Architecture Pattern

```
┌──────────────────────────────────────────────────────────────┐
│                     Browser Environment                       │
│  ┌──────────────────┐      ┌──────────────────────────────┐   │
│  │   Main Thread    │      │      Web Worker              │   │
│  │                  │      │                              │   │
│  │  ┌──────────┐   │      │  ┌──────────┐  ┌──────────┐  │   │
│  │  │   UI     │◄──┼──────┼──┤  Agent   │  │  WASM    │  │   │
│  │  │ Thread   │   │      │  │  Loop    │  │  Module  │  │   │
│  │  └──────────┘   │      │  └──────────┘  └──────────┘  │   │
│  │        ▲        │      │        │            │        │   │
│  │        │        │      │        ▼            ▼        │   │
│  │  ┌─────┴──────┐ │      │  ┌────────────────────────┐  │   │
│  │  │  Message   │ │      │  │  Streaming Fetch     │  │   │
│  │  │  Channel   │ │      │  │  - JS fetch() bridge   │  │   │
│  │  └────────────┘ │      │  └────────────────────────┘  │   │
│  └──────────────────┘      └──────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### 2.3 Web Worker Implementation Pattern

**worker.js:**
```javascript
// Web Worker for WebClaw Agent Loop
importScripts('wasm_exec.js');

let wasmInstance = null;

// Initialize WASM in worker
async function initWasm(wasmBinary) {
  const go = new Go();
  const result = await WebAssembly.instantiate(wasmBinary, go.importObject);
  wasmInstance = result.instance;
  go.run(wasmInstance);
}

// Message passing protocol
self.onmessage = async (e) => {
  const { type, payload } = e.data;
  
  switch (type) {
    case 'INIT_WASM':
      await initWasm(payload.wasmBinary);
      self.postMessage({ type: 'WASM_READY' });
      break;
      
    case 'START_STREAM':
      // Call into Go WASM - sets up streaming
      globalThis.webclawStartStream(payload.conversationId, payload.messages);
      break;
      
    case 'USER_MESSAGE':
      // Add user message to active conversation
      globalThis.webclawAddMessage(payload.conversationId, payload.message);
      break;
      
    case 'ABORT_STREAM':
      globalThis.webclawAbortStream(payload.conversationId);
      break;
  }
};

// Called from Go WASM when tokens arrive
globalThis.onClawToken = (conversationId, token) => {
  self.postMessage({
    type: 'TOKEN',
    payload: { conversationId, token }
  });
};

// Called when complete
globalThis.onClawComplete = (conversationId, summary) => {
  self.postMessage({
    type: 'COMPLETE',
    payload: { conversationId, summary }
  });
};
```

**Main thread integration:**
```javascript
// main.js
const worker = new Worker('worker.js', { type: 'module' });

worker.onmessage = (e) => {
  const { type, payload } = e.data;
  
  switch (type) {
    case 'TOKEN':
      // Append token to UI
      ui.appendToken(payload.conversationId, payload.token);
      break;
    case 'COMPLETE':
      ui.markComplete(payload.conversationId);
      break;
  }
};

// Start conversation
worker.postMessage({
  type: 'START_STREAM',
  payload: {
    conversationId: 'conv-123',
    messages: [{ role: 'user', content: 'Hello' }]
  }
});
```

### 2.4 WASM Streaming Bridge

**Go side (exposed to JS):**
```go
// internal/agent/worker_bridge.go
package agent

import (
    "syscall/js"
    "context"
)

// RegisterWorkerCallbacks registers JS-callable functions for Web Worker
func RegisterWorkerCallbacks() {
    js.Global().Set("webclawStartStream", js.FuncOf(startStream))
    js.Global().Set("webclawAddMessage", js.FuncOf(addMessage))
    js.Global().Set("webclawAbortStream", js.FuncOf(abortStream))
}

func startStream(this js.Value, args []js.Value) interface{} {
    conversationId := args[0].String()
    messagesJS := args[1]
    
    // Convert JS messages to Go types
    messages := parseMessages(messagesJS)
    
    // Start streaming in goroutine
    go func() {
        ctx := context.Background()
        for token := range agent.StreamResponse(ctx, messages) {
            // Call back to JS
            js.Global().Call("onClawToken", conversationId, token)
        }
        js.Global().Call("onClawComplete", conversationId)
    }()
    
    return nil
}
```

---

## 3. Conversation Summarization Strategies

### 3.1 When to Summarize (AGNT-02)

**Thresholds:**
- Message count: 20 messages
- Token count: 75% of model's context window
- Character count: 150K characters (matches bootstrap limit)

**Detection Pattern:**
```go
const (
    MaxMessagesBeforeSummarize = 20
    ContextWindowPercentile    = 0.75
)

type Conversation struct {
    Messages []Message
    Summary  *Summary
}

func (c *Conversation) NeedsSummarization(modelMaxTokens int) bool {
    if len(c.Messages) >= MaxMessagesBeforeSummarize {
        return true
    }
    
    totalTokens := c.EstimateTokenCount()
    if float64(totalTokens) > float64(modelMaxTokens)*ContextWindowPercentile {
        return true
    }
    
    return false
}

func (c *Conversation) EstimateTokenCount() int {
    // Conservative estimate: ~4 characters per token
    totalChars := 0
    for _, msg := range c.Messages {
        totalChars += len(msg.Content)
    }
    return totalChars / 4
}
```

### 3.2 Summarization Strategies (AGNT-03)

**Strategy 1: Progressive Summarization (Recommended)**

Instead of replacing the entire history, maintain a sliding window:

```
┌─────────────────────────────────────────────────────────────┐
│              Progressive Summarization Model                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  [Summary of turns 1-15]                                     │
│  [Turn 16 - User]                                            │
│  [Turn 16 - Assistant]                                       │
│  [Turn 17 - User]                                            │
│  [Turn 17 - Assistant]                                       │
│  ... (recent turns retained in full)                        │
│                                                              │
│  When threshold hit again:                                   │
│  → Summarize turns 16-17 into previous summary               │
│  → Keep last 3 turns in full                                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Implementation:**
```go
type SlidingWindow struct {
    Summary        string          // Condensed history
    RecentMessages []Message       // Full messages (last N turns)
    MaxRecentTurns int             // Keep last 3-5 turns in full
}

func (sw *SlidingWindow) Summarize(ctx context.Context, provider Provider) error {
    // Build prompt for summarization
    prompt := fmt.Sprintf(`
        Summarize the following conversation concisely, preserving:
        - Key facts and decisions
        - User preferences mentioned
        - Outstanding questions or tasks
        
        Previous summary:
        %s
        
        Recent conversation:
        %s
        
        Provide a condensed summary in 2-4 paragraphs.
    `, sw.Summary, formatMessages(sw.RecentMessages))
    
    // Call LLM for summarization
    summary, err := provider.Complete(ctx, prompt)
    if err != nil {
        return err
    }
    
    // Update sliding window
    sw.Summary = summary
    sw.RecentMessages = nil // Reset recent, will fill with new messages
    
    return nil
}
```

**Strategy 2: Extractive Key Points**

For critical conversations, extract key facts before summarization:

```go
type MemoryExtractor struct {
    KeyFacts    []string
    Decisions   []string
    Preferences map[string]string
}

func (me *MemoryExtractor) Extract(ctx context.Context, messages []Message) error {
    // Use LLM to extract structured facts
    extractionPrompt := `
        From this conversation, extract:
        1. Key facts stated by the user (as bullet points)
        2. Decisions made by the assistant (as bullet points)
        3. User preferences or constraints (as key: value pairs)
        
        Format as JSON.
    `
    
    // Store extracted facts to memory system
    for _, fact := range me.KeyFacts {
        memory.Store(ctx, MemoryDocument{
            Content: fact,
            Type:    "fact",
            Source:  conversationId,
        })
    }
    
    return nil
}
```

### 3.3 OpenClaw-Compatible Memory Flush (MEM-04)

**Requirement:** Before compaction, durable knowledge must be flushed to MEMORY.md

**Pattern:**
```go
func (c *Conversation) CompactAndFlush(ctx context.Context) error {
    // 1. Extract durable knowledge
    extractor := &MemoryExtractor{}
    if err := extractor.Extract(ctx, c.Messages); err != nil {
        return err
    }
    
    // 2. Flush to MEMORY.md
    memoryDoc := buildMemoryDocument(extractor)
    if err := identity.WriteMemoryFile(memoryDoc); err != nil {
        return err
    }
    
    // 3. Summarize conversation
    if err := c.SlidingWindow.Summarize(ctx, provider); err != nil {
        return err
    }
    
    // 4. Clear old messages (keeping summary + recent)
    c.Messages = c.SlidingWindow.RecentMessages
    
    return nil
}
```

---

## 4. Vector Embedding Storage in IndexedDB

### 4.1 Storage Schema (MEM-01)

**Database Design:**

```
┌─────────────────────────────────────────────────────────────┐
│              IndexedDB Schema: webclaw:memory                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Object Stores:                                              │
│                                                              │
│  1. memories (primary)                                       │
│     - keyPath: id (UUID)                                     │
│     - content: string                                        │
│     - embedding: Float32Array                                │
│     - metadata: object (source, timestamp, type)             │
│     - tokens: number (for quota tracking)                    │
│     - accessCount: number (for LRU eviction)                 │
│     - lastAccessed: timestamp                                │
│                                                              │
│  2. memory_index                                             │
│     - For keyword search (BM25)                              │
│     - Inverted index: term → [memoryIds]                     │
│     - Token frequencies per document                         │
│                                                              │
│  3. archives                                                 │
│     - Compressed/older memories                              │
│     - Serialized as JSON blobs                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Go WASM Bridge:**
```go
// internal/memory/storage.go
package memory

import (
    "syscall/js"
    "encoding/json"
    "time"
)

type MemoryDocument struct {
    ID            string    `json:"id"`
    Content       string    `json:"content"`
    Embedding     []float32 `json:"embedding"`
    Metadata      Metadata  `json:"metadata"`
    Tokens        int       `json:"tokens"`
    AccessCount   int       `json:"access_count"`
    LastAccessed  time.Time `json:"last_accessed"`
}

func (m *MemoryDocument) Store(ctx context.Context) error {
    // Call JS IndexedDB bridge
    docJSON, _ := json.Marshal(m)
    promise := js.Global().Call("webclawMemoryStore", string(docJSON))
    
    // Wait for promise resolution
    return awaitPromise(promise)
}
```

**JavaScript Bridge:**
```javascript
// Bridge: JS side
async function webclawMemoryStore(docJSON) {
  const db = await openMemoryDB();
  const doc = JSON.parse(docJSON);
  
  // Store in memories object store
  await db.put('memories', {
    id: doc.id,
    content: doc.content,
    embedding: new Float32Array(doc.embedding),
    metadata: doc.metadata,
    tokens: doc.tokens,
    access_count: doc.access_count,
    last_accessed: Date.now()
  });
  
  // Update keyword index for BM25
  await updateKeywordIndex(db, doc);
  
  // Check quota and evict if needed
  await checkAndEvict(db);
  
  return { success: true };
}
```

### 4.2 Embedding Generation (MEM-03)

**Options:**

1. **Provider Embeddings** (Recommended for v1)
   - Use active LLM provider's embedding endpoint
   - Anthropic: No native embeddings, use text-content for search
   - OpenAI: `text-embedding-3-small` (1536 dimensions)
   - OpenRouter: Various models available

2. **Local Embeddings** (Future optimization)
   - Libraries: `client-vector-search`, `@xenova/transformers`
   - Models: `gte-small` (384d), `nomic-embed-text-v1` (768d)
   - Trade-off: ~30-500MB download vs. API cost

**Implementation:**
```go
// internal/memory/embeddings.go
package memory

type EmbeddingProvider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type OpenAIEmbedder struct {
    APIKey string
    Model  string // "text-embedding-3-small"
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // Call JS fetch bridge to OpenAI embeddings API
    // POST https://api.openai.com/v1/embeddings
    // Returns [][]float32
}

// Dimension sizes to store in schema
const (
    EmbeddingDimOpenAI3Small = 1536
    EmbeddingDimOpenAI3Large = 3072
)
```

---

## 5. Hybrid Search: Cosine Similarity + BM25

### 5.1 Algorithm Overview (MEM-02)

**Hybrid Search Formula:**
```
FinalScore = (0.7 × CosineScore) + (0.3 × BM25Score)
```

Where:
- CosineScore: Vector similarity (0-1 range)
- BM25Score: Normalized keyword relevance (0-1 range)

### 5.2 Cosine Similarity Implementation

```go
// internal/memory/search.go
package memory

import "math"

func cosineSimilarity(a, b []float32) float32 {
    if len(a) != len(b) {
        return 0
    }
    
    var dotProduct, normA, normB float32
    for i := 0; i < len(a); i++ {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Optimized search with early termination
func (s *MemoryStore) VectorSearch(
    ctx context.Context, 
    query []float32, 
    minSimilarity float32,
    topK int,
) ([]SearchResult, error) {
    // Load all memories (or use HNSW index for large datasets)
    memories, err := s.loadAllMemories(ctx)
    if err != nil {
        return nil, err
    }
    
    results := make([]SearchResult, 0, topK)
    
    for _, mem := range memories {
        similarity := cosineSimilarity(query, mem.Embedding)
        
        if similarity >= minSimilarity {
            results = append(results, SearchResult{
                Memory:    mem,
                Score:     similarity,
                Algorithm: "cosine",
            })
        }
    }
    
    // Sort by score descending
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    return results, nil
}
```

### 5.3 BM25 Implementation

**BM25 Formula:**
```
score(D,Q) = Σ IDF(q_i) × (f(q_i,D) × (k1 + 1)) / (f(q_i,D) + k1 × (1 - b + b × |D|/avgDL))

Where:
- IDF(q_i) = log((N - n(q_i) + 0.5) / (n(q_i) + 0.5))
- N = total documents
- n(q_i) = documents containing term q_i
- f(q_i,D) = term frequency in document D
- |D| = document length
- avgDL = average document length
- k1 = 1.2 (term saturation parameter)
- b = 0.75 (length normalization parameter)
```

**JavaScript Implementation (in browser):**
```javascript
// internal/memory/bm25.js
class BM25Index {
  constructor(k1 = 1.2, b = 0.75) {
    this.k1 = k1;
    this.b = b;
    this.documents = new Map();
    this.invertedIndex = new Map();
    this.avgDocLength = 0;
    this.totalDocLength = 0;
    this.N = 0;
  }
  
  tokenize(text) {
    return text.toLowerCase()
      .replace(/[^a-z0-9\s]/g, '')
      .split(/\s+/)
      .filter(t => t.length > 2);
  }
  
  addDocument(id, text) {
    const tokens = this.tokenize(text);
    const docLength = tokens.length;
    
    // Update stats
    this.totalDocLength += docLength;
    this.N += 1;
    this.avgDocLength = this.totalDocLength / this.N;
    
    // Store document
    this.documents.set(id, { tokens, length: docLength });
    
    // Update inverted index
    const termFreq = new Map();
    for (const token of tokens) {
      termFreq.set(token, (termFreq.get(token) || 0) + 1);
    }
    
    for (const [term, freq] of termFreq) {
      if (!this.invertedIndex.has(term)) {
        this.invertedIndex.set(term, new Map());
      }
      this.invertedIndex.get(term).set(id, freq);
    }
  }
  
  search(query, topK = 10) {
    const queryTerms = this.tokenize(query);
    const scores = new Map();
    
    for (const term of queryTerms) {
      const postings = this.invertedIndex.get(term);
      if (!postings) continue;
      
      const n = postings.size; // docs containing term
      const idf = Math.log((this.N - n + 0.5) / (n + 0.5));
      
      for (const [docId, freq] of postings) {
        const doc = this.documents.get(docId);
        const numerator = freq * (this.k1 + 1);
        const denominator = freq + this.k1 * (1 - this.b + this.b * (doc.length / this.avgDocLength));
        
        const score = idf * (numerator / denominator);
        scores.set(docId, (scores.get(docId) || 0) + score);
      }
    }
    
    // Normalize scores to 0-1 range
    const maxScore = Math.max(...scores.values(), 1);
    const normalized = new Map();
    for (const [id, score] of scores) {
      normalized.set(id, score / maxScore);
    }
    
    // Return top K
    return Array.from(normalized.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, topK)
      .map(([id, score]) => ({ id, score }));
  }
}
```

### 5.4 Hybrid Search Merging

```go
// internal/memory/hybrid.go
package memory

const (
    VectorWeight = 0.7
    KeywordWeight = 0.3
)

func (s *MemoryStore) HybridSearch(
    ctx context.Context,
    query string,
    queryEmbedding []float32,
    topK int,
) ([]SearchResult, error) {
    // Run searches in parallel
    var vectorResults, keywordResults []SearchResult
    var err error
    
    errChan := make(chan error, 2)
    
    go func() {
        vectorResults, err = s.VectorSearch(ctx, queryEmbedding, 0.6, topK*2)
        errChan <- err
    }()
    
    go func() {
        keywordResults, err = s.KeywordSearch(ctx, query, topK*2)
        errChan <- err
    }()
    
    // Wait for both
    for i := 0; i < 2; i++ {
        if e := <-errChan; e != nil {
            return nil, e
        }
    }
    
    // Merge and rerank
    merged := make(map[string]SearchResult)
    
    // Add vector results
    for _, r := range vectorResults {
        r.Score = r.Score * VectorWeight
        merged[r.Memory.ID] = r
    }
    
    // Add keyword results
    for _, r := range keywordResults {
        if existing, ok := merged[r.Memory.ID]; ok {
            // Already exists, combine scores
            existing.Score += r.Score * KeywordWeight
            existing.Algorithm = "hybrid"
            merged[r.Memory.ID] = existing
        } else {
            r.Score = r.Score * KeywordWeight
            merged[r.Memory.ID] = r
        }
    }
    
    // Convert to slice and sort
    results := make([]SearchResult, 0, len(merged))
    for _, r := range merged {
        results = append(results, r)
    }
    
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    // Update access stats
    for _, r := range results {
        s.updateAccessStats(ctx, r.Memory.ID)
    }
    
    return results, nil
}
```

---

## 6. Memory Management and Quota Handling

### 6.1 Quota Detection (MEM-05)

**IndexedDB Quota API:**
```javascript
// Check storage quota
async function checkStorageQuota() {
  if ('storage' in navigator && 'estimate' in navigator.storage) {
    const estimate = await navigator.storage.estimate();
    const usage = estimate.usage || 0;
    const quota = estimate.quota || Infinity;
    const percentUsed = (usage / quota) * 100;
    
    return {
      usage,
      quota,
      percentUsed,
      remaining: quota - usage
    };
  }
  
  // Fallback: estimate based on known limits
  return {
    usage: await estimateIndexedDBSize(),
    quota: 60 * 1024 * 1024 * 1024, // 60GB (typical Chrome limit)
    percentUsed: 0,
    remaining: Infinity
  };
}
```

### 6.2 Eviction Strategy

**LRU + Access Frequency Hybrid:**

```go
// internal/memory/eviction.go
package memory

const (
    QuotaThresholdPercent = 80
    TargetUsagePercent    = 60 // Evict down to 60%
)

func (s *MemoryStore) checkAndEvict(ctx context.Context) error {
    quota, err := s.getStorageQuota(ctx)
    if err != nil {
        return err
    }
    
    if quota.PercentUsed < QuotaThresholdPercent {
        return nil // No eviction needed
    }
    
    // Calculate how much to evict
    targetSize := int64(float64(quota.Quota) * (TargetUsagePercent / 100))
    bytesToEvict := quota.Usage - targetSize
    
    // Get candidates sorted by LRU + low access count
    candidates, err := s.getEvictionCandidates(ctx, bytesToEvict)
    if err != nil {
        return err
    }
    
    // Archive before delete (for potential recovery)
    for _, mem := range candidates {
        if err := s.archiveMemory(ctx, mem); err != nil {
            log.Printf("Failed to archive memory %s: %v", mem.ID, err)
        }
        
        if err := s.deleteMemory(ctx, mem.ID); err != nil {
            log.Printf("Failed to delete memory %s: %v", mem.ID, err)
        }
    }
    
    return nil
}

func (s *MemoryStore) getEvictionCandidates(
    ctx context.Context, 
    bytesToEvict int64,
) ([]MemoryDocument, error) {
    // Load all memories with stats
    allMemories, err := s.loadAllMemories(ctx)
    if err != nil {
        return nil, err
    }
    
    // Score each memory: lower = more likely to evict
    type scoredMemory struct {
        doc   MemoryDocument
        score float64
    }
    
    scored := make([]scoredMemory, len(allMemories))
    now := time.Now()
    
    for i, mem := range allMemories {
        // Age factor (hours since last access)
        ageHours := now.Sub(mem.LastAccessed).Hours()
        
        // Access frequency factor
        accessScore := float64(mem.AccessCount) / (ageHours + 1)
        
        // Combined score (lower = better candidate for eviction)
        scored[i] = scoredMemory{
            doc:   mem,
            score: ageHours/(accessScore+1) + float64(mem.Tokens)/1000,
        }
    }
    
    // Sort by score descending (highest score = evict first)
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })
    
    // Select candidates until we meet eviction target
    var candidates []MemoryDocument
    var accumulatedSize int64
    
    for _, sm := range scored {
        if accumulatedSize >= bytesToEvict {
            break
        }
        candidates = append(candidates, sm.doc)
        accumulatedSize += int64(sm.doc.Tokens * 4) // Approx 4 bytes per token
    }
    
    return candidates, nil
}
```

### 6.3 Archival Strategy

**Compression and Archive Storage:**

```javascript
// Archive memories as compressed blobs
async function archiveMemories(memories) {
  const archive = {
    version: '1.0',
    archivedAt: Date.now(),
    memories: memories,
    compressed: false
  };
  
  // Convert to string
  const jsonStr = JSON.stringify(archive);
  
  // Compress using CompressionStream if available
  let compressed;
  if ('CompressionStream' in window) {
    const stream = new Blob([jsonStr]).stream();
    const compressedStream = stream.pipeThrough(new CompressionStream('gzip'));
    const response = new Response(compressedStream);
    compressed = await response.arrayBuffer();
    archive.compressed = true;
  }
  
  // Store in archives object store
  const db = await openMemoryDB();
  await db.put('archives', {
    id: `archive-${Date.now()}`,
    data: compressed || jsonStr,
    memoryCount: memories.length,
    compressed: archive.compressed,
    createdAt: Date.now()
  });
}
```

---

## 7. Library Recommendations

### 7.1 For Hybrid Search (MEM-02)

| Library | Size | Pros | Cons |
|---------|------|------|------|
| **@orama/orama** | ~2KB | Full-text + vector + hybrid, fast, well-maintained | Additional dependency |
| **client-vector-search** | ~30MB* | Local embeddings, IndexedDB persistence | Large model download |
| **kyr0/vectorstore** | ~500MB* | Good for large datasets, multilingual | Very large, Node-focused |
| **Custom BM25 + Cosine** | ~5KB | Minimal, full control, no deps | Implementation effort |

*Model sizes separate

**Recommendation:** Start with custom implementation for BM25 + cosine similarity. Orama is excellent if we want to add dependency.

### 7.2 For Embeddings (MEM-03)

| Approach | Latency | Cost | Privacy | Size |
|----------|---------|------|---------|------|
| **OpenAI API** | ~100-300ms | $0.02/1M tokens | Low | 0 |
| **@xenova/transformers** | ~50-500ms** | Free | High | 30-500MB |
| **Provider Embeddings** | Varies | Included | Medium | 0 |

**Depends on model

**Recommendation:** Use provider embeddings (via OpenRouter or OpenAI) for v1. Local embedding support can be added as plugin in v2.

### 7.3 For Web Workers

No external libraries needed - use native Web Worker API with postMessage protocol.

---

## 8. Common Pitfalls to Avoid

### 8.1 Streaming Pitfalls

| Pitfall | Solution |
|---------|----------|
| **Memory leaks from open readers** | Always call `reader.releaseLock()` when done |
| **Missing SSE boundary handling** | Events can span multiple chunks - buffer partial lines |
| **CORS issues with credentials** | Use `credentials: 'include'` only when necessary |
| **Not handling abort signals** | Pass AbortController signal to fetch for cancellation |
| **Zombie streams on errors** | Wrap in try/finally to ensure cleanup |

**Proper SSE Parsing:**
```javascript
function createSSEParser(callback) {
  let buffer = '';
  
  return {
    process(chunk) {
      buffer += chunk;
      const lines = buffer.split('\n');
      buffer = lines.pop(); // Keep incomplete line in buffer
      
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data === '[DONE]') {
            callback({ done: true });
          } else {
            try {
              callback({ data: JSON.parse(data) });
            } catch (e) {
              console.error('Invalid SSE data:', data);
            }
          }
        }
      }
    }
  };
}
```

### 8.2 IndexedDB Pitfalls

| Pitfall | Solution |
|---------|----------|
| **Storing Float32Array directly** | Wrap in object or use Array.from() before JSON.stringify |
| **Not handling version upgrades** | Implement onupgradeneeded with migration logic |
| **Large transaction timeouts** | Batch operations in chunks of 100-1000 |
| **Quota exceeded errors** | Catch and trigger eviction |
| **Concurrent access conflicts** | Use single writer pattern or proper locking |

### 8.3 WASM-JS Bridge Pitfalls

| Pitfall | Solution |
|---------|----------|
| **Passing Go structs to JS** | Convert to primitives or JSON strings |
| **Memory leaks from js.Func** | Call .Release() when done |
| **Blocking the main thread** | Use goroutines + callbacks, never long-running ops |
| **Not handling JS exceptions** | Wrap JS calls in defer/recover |

---

## 9. Architecture Recommendations

### 9.1 Recommended Package Structure

```
internal/
├── provider/
│   ├── provider.go          # Provider interface
│   ├── anthropic.go         # Anthropic implementation
│   ├── openai.go            # OpenAI implementation
│   ├── openrouter.go        # OpenRouter implementation
│   └── router.go            # vendor/model-id routing
├── agent/
│   ├── loop.go              # Agent loop orchestration
│   ├── context.go           # Context/history management
│   ├── summarizer.go        # Summarization logic
│   └── worker_bridge.go     # Web Worker integration
├── memory/
│   ├── store.go             # IndexedDB storage interface
│   ├── embedding.go         # Embedding generation
│   ├── search.go            # Cosine similarity search
│   ├── bm25.go              # BM25 keyword search
│   ├── hybrid.go            # Hybrid search merger
│   └── eviction.go          # Quota management
└── jsbridge/
    ├── fetch.go             # syscall/js fetch bridge
    ├── idb.go               # IndexedDB bridge
    └── worker.go            # Web Worker helpers
```

### 9.2 Provider Interface Design

```go
// internal/provider/provider.go
package provider

import "context"

// Token represents a streamed token
type Token struct {
    Content string
    Done    bool
    Error   error
}

// CompletionRequest represents a chat completion request
type CompletionRequest struct {
    Model       string
    Messages    []Message
    MaxTokens   int
    Temperature float64
    Stream      bool
    Tools       []Tool
}

// Provider is the interface for LLM providers
type Provider interface {
    // Complete returns a single non-streaming response
    Complete(ctx context.Context, req CompletionRequest) (*Message, error)
    
    // Stream returns a channel of tokens
    Stream(ctx context.Context, req CompletionRequest) (<-chan Token, error)
    
    // Embed generates embeddings for texts
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    
    // Name returns the provider identifier
    Name() string
    
    // MaxContextWindow returns the model's context limit
    MaxContextWindow(model string) int
}

// Router routes vendor/model-id to the appropriate provider
type Router struct {
    providers map[string]Provider
    configs   map[string]ProviderConfig
}

func (r *Router) Route(vendorModel string) (Provider, string, error) {
    // Parse "anthropic/claude-sonnet-4-5" format
    parts := strings.Split(vendorModel, "/")
    if len(parts) != 2 {
        return nil, "", fmt.Errorf("invalid model format: %s", vendorModel)
    }
    
    vendor, model := parts[0], parts[1]
    provider, ok := r.providers[vendor]
    if !ok {
        return nil, "", fmt.Errorf("unknown vendor: %s", vendor)
    }
    
    return provider, model, nil
}
```

### 9.3 Memory Store Interface

```go
// internal/memory/store.go
package memory

import "context"

type MemoryStore interface {
    // Store saves a memory document
    Store(ctx context.Context, doc *MemoryDocument) error
    
    // Get retrieves a memory by ID
    Get(ctx context.Context, id string) (*MemoryDocument, error)
    
    // Search performs hybrid vector + keyword search
    Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)
    
    // Delete removes a memory
    Delete(ctx context.Context, id string) error
    
    // Archive moves old memories to compressed storage
    Archive(ctx context.Context, before time.Time) error
    
    // CheckQuota returns storage usage statistics
    CheckQuota(ctx context.Context) (*QuotaInfo, error)
    
    // Evict removes memories to free space
    Evict(ctx context.Context, targetBytes int64) error
}

type SearchQuery struct {
    Text            string
    Embedding       []float32
    TopK            int
    MinSimilarity   float32
    Filter          map[string]interface{}
}
```

---

## 10. Implementation Checklist

### Phase 3 Plans

Based on this research, Phase 3 should be divided into 4 plans:

#### Plan 03-01: LLM Provider System (PROV-*)
- [ ] Provider interface definition
- [ ] Anthropic provider implementation
- [ ] OpenAI provider implementation
- [ ] OpenRouter provider implementation
- [ ] Vendor/model-id router
- [ ] Failover and retry logic
- [ ] JS fetch bridge with streaming

#### Plan 03-02: Agent Loop & Streaming (AGNT-*)
- [ ] Web Worker infrastructure
- [ ] Agent loop orchestration
- [ ] Conversation context management
- [ ] Token streaming to UI
- [ ] Tool call handling framework

#### Plan 03-03: Conversation Management (AGNT-02, AGNT-03, MEM-04)
- [ ] Context window monitoring
- [ ] Summarization triggers
- [ ] Progressive summarization implementation
- [ ] Memory flush to MEMORY.md
- [ ] Sliding window conversation model

#### Plan 03-04: Memory System (MEM-*)
- [ ] IndexedDB memory storage schema
- [ ] Vector embedding storage (Float32Array)
- [ ] Cosine similarity search
- [ ] BM25 keyword index
- [ ] Hybrid search (0.7/0.3 weighting)
- [ ] Quota monitoring
- [ ] LRU eviction with archival

---

## 11. Testing Strategy

### 11.1 Unit Tests (Go WASM-compatible)

```go
// internal/memory/search_test.go
package memory

func TestCosineSimilarity(t *testing.T) {
    a := []float32{1, 0, 0}
    b := []float32{0, 1, 0}
    
    score := cosineSimilarity(a, b)
    if score != 0 {
        t.Errorf("Expected 0 for orthogonal vectors, got %f", score)
    }
    
    c := []float32{1, 0, 0}
    score = cosineSimilarity(a, c)
    if score != 1 {
        t.Errorf("Expected 1 for identical vectors, got %f", score)
    }
}
```

### 11.2 Integration Tests

```javascript
// tests/agent_integration.test.js
// Run with Playwright or similar

describe('Agent Loop', () => {
  test('streaming response delivers tokens', async () => {
    const tokens = [];
    
    worker.onmessage = (e) => {
      if (e.data.type === 'TOKEN') {
        tokens.push(e.data.payload.token);
      }
    };
    
    worker.postMessage({
      type: 'START_STREAM',
      payload: { messages: [{ role: 'user', content: 'Hi' }] }
    });
    
    await waitFor(() => tokens.length > 0, { timeout: 5000 });
    expect(tokens.length).toBeGreaterThan(0);
  });
});
```

### 11.3 Performance Benchmarks

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token streaming latency | <100ms first token | Time from request to first TOKEN event |
| Hybrid search | <50ms for 1K memories | Time from query to results |
| Memory store | <20ms per document | Time to persist with index update |
| Summarization | <5s for 20 messages | Time to generate summary |
| Quota check | <10ms | Time to check storage estimate |

---

## RESEARCH COMPLETE

This document provides comprehensive technical guidance for implementing Phase 3: Intelligence Core. All success criteria (PROV-*, AGNT-*, MEM-*) have been researched with concrete implementation patterns, library recommendations, and testing strategies.

**Key Decisions:**
1. Use custom BM25 + cosine implementation for hybrid search (minimal deps)
2. Use provider embeddings (not local models) for v1
3. Implement Web Worker architecture for non-blocking streaming
4. Use progressive summarization with sliding window
5. LRU + access-frequency hybrid for eviction

**Next Steps:**
Proceed to plan creation based on the 4-plan structure outlined in Section 10.
