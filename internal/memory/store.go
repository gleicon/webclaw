//go:build js && wasm

package memory

import (
	"context"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// Store defines the interface for memory storage operations.
type Store interface {
	// Store saves a memory document with embedding.
	Store(doc *MemoryDocument) error

	// Get retrieves a memory document by ID.
	Get(id string) (*MemoryDocument, error)

	// Delete removes a memory document by ID.
	Delete(id string) error

	// Search finds memories matching the query.
	Search(query string, opts SearchOptions) ([]*MemorySearchResult, error)

	// GetAll retrieves all memory documents.
	GetAll() ([]*MemoryDocument, error)

	// CheckQuota returns current storage quota information.
	CheckQuota() (*QuotaInfo, error)

	// EvictIfNeeded removes memories if quota is exceeded.
	EvictIfNeeded() error
}

// memoryStore implements Store using IndexedDB.
type memoryStore struct {
	db       js.Value
	embedder Embedder
	evictor  *LRUEvictor
	bm25     *BM25Index
}

// Embedder generates embeddings for content.
type Embedder interface {
	// Embed generates a vector embedding for the given text.
	Embed(text string) ([]float32, error)
}

// NewMemoryStore creates a new memory store with the given embedder.
func NewMemoryStore(embedder Embedder) (Store, error) {
	// Open IndexedDB
	dbPromise := jsbridge.MemoryDBOpen()

	// Wait for the promise to resolve
	dbChan := make(chan js.Value, 1)
	errChan := make(chan error, 1)

	dbPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		dbChan <- args[0]
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to open memory database: %v", args[0])
		return nil
	}))

	select {
	case db := <-dbChan:
		return &memoryStore{
			db:       db,
			embedder: embedder,
			evictor:  NewLRUEvictor(),
			bm25:     NewBM25Index(),
		}, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout opening memory database")
	}
}

// SetEmbedder sets the embedder for generating embeddings.
// PHASE 6-6: Allows async enablement of embeddings after initialization.
func (s *memoryStore) SetEmbedder(embedder Embedder) {
	s.embedder = embedder
}

// Store saves a memory document with embedding.
func (s *memoryStore) Store(doc *MemoryDocument) error {
	// PHASE 6-6: Check quota before storing (MEM-05)
	// Trigger eviction at 80% quota threshold
	if s.evictor != nil {
		quota, err := s.evictor.CheckQuota(context.Background())
		if err == nil && quota.ShouldEvict {
			js.Global().Get("console").Call("log",
				"webclaw: storage at", int(quota.Percent), "%- triggering LRU eviction")
			if err := s.EvictIfNeeded(); err != nil {
				js.Global().Get("console").Call("error",
					"webclaw: eviction failed:", err.Error())
			}
		}
	}

	// Generate embedding if not provided
	if len(doc.Embedding) == 0 && s.embedder != nil {
		embedding, err := s.embedder.Embed(doc.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		doc.Embedding = embedding
	}

	// Calculate token count (rough estimate: ~4 chars per token)
	if doc.Tokens == 0 {
		doc.Tokens = len(doc.Content) / 4
	}

	// Update timestamps
	doc.LastAccessed = time.Now()

	// Store in IndexedDB using the bridge struct
	bridgeDoc := jsbridge.MemoryDocument{
		ID:           doc.ID,
		Content:      doc.Content,
		Embedding:    doc.Embedding,
		Tokens:       doc.Tokens,
		AccessCount:  doc.AccessCount,
		LastAccessed: doc.LastAccessed.Format(time.RFC3339),
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
		Importance:   doc.Importance,
	}
	if doc.Metadata != nil {
		bridgeDoc.Metadata = doc.Metadata
	}

	putPromise := jsbridge.MemoryPut(s.db, bridgeDoc)

	// Wait for completion
	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	putPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		doneChan <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to store memory: %v", args[0])
		return nil
	}))

	select {
	case <-doneChan:
		// Update BM25 index
		s.bm25.AddDocument(doc.ID, doc.Content)
		return nil
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout storing memory")
	}
}

// Get retrieves a memory document by ID.
func (s *memoryStore) Get(id string) (*MemoryDocument, error) {
	getPromise := jsbridge.MemoryGet(s.db, id)

	docChan := make(chan *MemoryDocument, 1)
	errChan := make(chan error, 1)

	getPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if args[0].IsNull() || args[0].IsUndefined() {
			docChan <- nil
			return nil
		}

		// Convert JS object to bridge document
		bridgeDoc := jsbridge.MemoryDocumentFromJS(args[0])
		if bridgeDoc == nil {
			docChan <- nil
			return nil
		}

		// Convert to memory package document
		doc := s.bridgeToMemoryDoc(bridgeDoc)
		docChan <- doc
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to get memory: %v", args[0])
		return nil
	}))

	select {
	case doc := <-docChan:
		if doc != nil {
			doc.RecordAccess()
			// Update access stats in background
			go s.updateAccessStats(doc)
		}
		return doc, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout getting memory")
	}
}

// bridgeToMemoryDoc converts a jsbridge MemoryDocument to memory.MemoryDocument.
func (s *memoryStore) bridgeToMemoryDoc(bridge *jsbridge.MemoryDocument) *MemoryDocument {
	createdAt, _ := time.Parse(time.RFC3339, bridge.CreatedAt)
	lastAccessed, _ := time.Parse(time.RFC3339, bridge.LastAccessed)

	return &MemoryDocument{
		ID:           bridge.ID,
		Content:      bridge.Content,
		Embedding:    bridge.Embedding,
		Metadata:     bridge.Metadata,
		Tokens:       bridge.Tokens,
		AccessCount:  bridge.AccessCount,
		LastAccessed: lastAccessed,
		CreatedAt:    createdAt,
		Importance:   bridge.Importance,
	}
}

// updateAccessStats updates the access count in IndexedDB (background operation).
func (s *memoryStore) updateAccessStats(doc *MemoryDocument) {
	// Fire and forget - don't block on this
	bridgeDoc := jsbridge.MemoryDocument{
		ID:           doc.ID,
		Content:      doc.Content,
		Embedding:    doc.Embedding,
		Tokens:       doc.Tokens,
		AccessCount:  doc.AccessCount,
		LastAccessed: doc.LastAccessed.Format(time.RFC3339),
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
		Importance:   doc.Importance,
	}
	if doc.Metadata != nil {
		bridgeDoc.Metadata = doc.Metadata
	}
	jsbridge.MemoryPut(s.db, bridgeDoc)
}

// Delete removes a memory document by ID.
func (s *memoryStore) Delete(id string) error {
	deletePromise := jsbridge.MemoryDelete(s.db, id)

	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	deletePromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		doneChan <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to delete memory: %v", args[0])
		return nil
	}))

	select {
	case <-doneChan:
		// Remove from BM25 index
		s.bm25.RemoveDocument(id)
		return nil
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout deleting memory")
	}
}

// Search finds memories matching the query using hybrid search.
func (s *memoryStore) Search(query string, opts SearchOptions) ([]*MemorySearchResult, error) {
	// Get query embedding
	var queryEmbedding []float32
	if s.embedder != nil {
		embedding, err := s.embedder.Embed(query)
		if err == nil {
			queryEmbedding = embedding
		}
	}

	// Get all memories for search
	allDocs, err := s.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get memories for search: %w", err)
	}

	// Perform hybrid search
	searcher := NewHybridSearcher(s.bm25, opts.VectorWeight, opts.KeywordWeight)
	results := searcher.Search(query, queryEmbedding, allDocs, opts)

	return results, nil
}

// GetAll retrieves all memory documents.
func (s *memoryStore) GetAll() ([]*MemoryDocument, error) {
	getAllPromise := jsbridge.MemoryGetAll(s.db)

	docsChan := make(chan []*MemoryDocument, 1)
	errChan := make(chan error, 1)

	getAllPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		// Convert JS array to Go slice
		bridgeDocs := jsbridge.MemoryDocumentsFromJSArray(args[0])

		// Convert to memory package documents
		docs := make([]*MemoryDocument, len(bridgeDocs))
		for i, bridge := range bridgeDocs {
			docs[i] = s.bridgeToMemoryDoc(bridge)
		}

		docsChan <- docs
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to get all memories: %v", args[0])
		return nil
	}))

	select {
	case docs := <-docsChan:
		return docs, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout getting all memories")
	}
}

// CheckQuota returns current storage quota information.
func (s *memoryStore) CheckQuota() (*QuotaInfo, error) {
	quotaPromise := jsbridge.GetStorageQuota()

	quotaChan := make(chan *QuotaInfo, 1)
	errChan := make(chan error, 1)

	quotaPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		bridgeInfo := jsbridge.QuotaInfoFromJS(args[0])

		// Convert to memory package QuotaInfo
		info := &QuotaInfo{
			Usage:    bridgeInfo.Usage,
			Quota:    bridgeInfo.Quota,
			Percent:  bridgeInfo.Percent,
			Overflow: bridgeInfo.Overflow,
		}

		quotaChan <- info
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to get quota: %v", args[0])
		return nil
	}))

	select {
	case info := <-quotaChan:
		return info, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout checking quota")
	}
}

// EvictIfNeeded removes memories if quota exceeds threshold.
func (s *memoryStore) EvictIfNeeded() error {
	quota, err := s.CheckQuota()
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	// Trigger eviction at 80% threshold
	if quota.Percent >= 80 {
		return s.evictor.EvictToTarget(s.db, 60)
	}

	return nil
}
