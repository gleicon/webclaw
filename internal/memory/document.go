//go:build js && wasm

package memory

import (
	"encoding/json"
	"time"
)

// MemoryDocument represents a stored memory with vector embedding.
// Stored in IndexedDB with Float32Array embedding for efficient retrieval.
type MemoryDocument struct {
	// ID is a unique identifier (UUID or timestamp-based)
	ID string `json:"id"`

	// Content is the text content of the memory
	Content string `json:"content"`

	// Embedding is the 1536-dimension Float32Array embedding vector
	// text-embedding-3-small produces 1536 dimensions
	Embedding []float32 `json:"embedding"`

	// Metadata contains additional structured information
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Tokens is the approximate token count
	Tokens int `json:"tokens"`

	// AccessCount tracks how many times this memory has been retrieved
	AccessCount int `json:"access_count"`

	// LastAccessed is the timestamp of last retrieval
	LastAccessed time.Time `json:"last_accessed"`

	// CreatedAt is when this memory was first stored
	CreatedAt time.Time `json:"created_at"`

	// Importance is a user-defined priority (0-10, default 5)
	Importance int `json:"importance"`
}

// NewMemoryDocument creates a new memory document with defaults.
func NewMemoryDocument(id, content string, embedding []float32) *MemoryDocument {
	now := time.Now()
	return &MemoryDocument{
		ID:           id,
		Content:      content,
		Embedding:    embedding,
		Metadata:     make(map[string]interface{}),
		Tokens:       0, // Will be calculated
		AccessCount:  0,
		LastAccessed: now,
		CreatedAt:    now,
		Importance:   5,
	}
}

// Serialize converts the document to JSON for IndexedDB storage.
func (m *MemoryDocument) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// DeserializeMemoryDocument parses a JSON memory document from IndexedDB.
func DeserializeMemoryDocument(data []byte) (*MemoryDocument, error) {
	var doc MemoryDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// RecordAccess updates access statistics when memory is retrieved.
func (m *MemoryDocument) RecordAccess() {
	m.AccessCount++
	m.LastAccessed = time.Now()
}

// GetLRUScore calculates a score for LRU eviction (lower = evict first).
// Combines age, access count, and importance.
func (m *MemoryDocument) GetLRUScore() float64 {
	age := time.Since(m.CreatedAt).Hours()
	// Higher importance = lower score (keep longer)
	// More accesses = lower score (keep longer)
	// Older age = higher score (evict sooner)
	importanceWeight := float64(11 - m.Importance) // 1-11 range
	accessWeight := 1.0 / float64(m.AccessCount+1) // Inverse of access count
	return age + importanceWeight*10 - accessWeight*5
}

// MemorySearchResult represents a search result with relevance score.
type MemorySearchResult struct {
	Document *MemoryDocument `json:"document"`
	Score    float64         `json:"score"`
	// Source indicates which search method found this result
	Source string `json:"source"` // "vector", "keyword", "hybrid"
}

// SearchOptions configures memory search behavior.
type SearchOptions struct {
	// Limit is the maximum number of results to return
	Limit int

	// MinScore is the minimum relevance threshold (0-1)
	MinScore float64

	// VectorWeight is the weight for vector similarity (default 0.7)
	VectorWeight float64

	// KeywordWeight is the weight for BM25 (default 0.3)
	KeywordWeight float64

	// IncludeArchived includes archived memories in search
	IncludeArchived bool
}

// DefaultSearchOptions returns search options with sensible defaults.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit:         10,
		MinScore:      0.5,
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
	}
}

// QuotaInfo represents storage quota estimation.
type QuotaInfo struct {
	Usage       int64   `json:"usage"`        // Bytes used
	Quota       int64   `json:"quota"`        // Total quota bytes
	Percent     float64 `json:"percent"`      // Usage percentage
	Overflow    bool    `json:"overflow"`     // True if usage > quota
	ShouldEvict bool    `json:"should_evict"` // True if usage >= 80% threshold
}
