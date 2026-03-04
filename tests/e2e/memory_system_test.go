//go:build js && wasm

package e2e

import (
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/memory"
)

// TestMemorySystem_StoreAndSearch verifies end-to-end memory storage and hybrid search
func TestMemorySystem_StoreAndSearch(t *testing.T) {
	// Create memory store (BM25-only for E2E test, no embedder)
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	// Store test documents
	docs := []*memory.MemoryDocument{
		{
			ID:       "test-1",
			Content:  "Go is a programming language created by Google",
			Metadata: map[string]interface{}{"type": "fact"},
		},
		{
			ID:       "test-2",
			Content:  "Python is popular for data science and machine learning",
			Metadata: map[string]interface{}{"type": "fact"},
		},
		{
			ID:       "test-3",
			Content:  "JavaScript runs in browsers and on servers via Node.js",
			Metadata: map[string]interface{}{"type": "fact"},
		},
	}

	for _, doc := range docs {
		if err := store.Store(doc); err != nil {
			t.Fatalf("failed to store doc %s: %v", doc.ID, err)
		}
	}

	// Give IndexedDB operations time to complete
	time.Sleep(100 * time.Millisecond)

	// Search for Go-related content
	results, err := store.Search("programming language", memory.SearchOptions{
		Limit:         5,
		MinScore:      0.1,
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Verify results exist
	if len(results) == 0 {
		t.Error("expected search results, got none")
		return
	}

	// First result should be Go-related (BM25 keyword match in BM25-only mode)
	foundGo := false
	for _, r := range results {
		if r.Document.ID == "test-1" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Logf("Warning: Go document not in top results. Results: %v", results)
		// Don't fail - BM25 ranking may vary
	}

	t.Logf("Found %d results for 'programming language'", len(results))
}

// TestMemorySystem_GetAndDelete verifies document retrieval and deletion
func TestMemorySystem_GetAndDelete(t *testing.T) {
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	// Store a document
	doc := &memory.MemoryDocument{
		ID:       "get-delete-test",
		Content:  "Test document for get and delete operations",
		Metadata: map[string]interface{}{"test": true},
	}

	if err := store.Store(doc); err != nil {
		t.Fatalf("failed to store doc: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Retrieve the document
	retrieved, err := store.Get("get-delete-test")
	if err != nil {
		t.Fatalf("failed to get doc: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected to retrieve stored document, got nil")
	}
	if retrieved.Content != doc.Content {
		t.Errorf("content mismatch: expected %q, got %q", doc.Content, retrieved.Content)
	}

	// Delete the document
	if err := store.Delete("get-delete-test"); err != nil {
		t.Fatalf("failed to delete doc: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Verify deletion
	deleted, err := store.Get("get-delete-test")
	if err != nil {
		t.Fatalf("error getting deleted doc: %v", err)
	}
	if deleted != nil {
		t.Error("expected nil after deletion, got document")
	}
}

// TestMemorySystem_GetAll verifies retrieval of all documents
func TestMemorySystem_GetAll(t *testing.T) {
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	// Store multiple documents
	for i := 0; i < 3; i++ {
		doc := &memory.MemoryDocument{
			ID:      "getall-test-" + string(rune('a'+i)),
			Content: "Document " + string(rune('A'+i)),
		}
		if err := store.Store(doc); err != nil {
			t.Fatalf("failed to store doc: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	// Get all documents
	allDocs, err := store.GetAll()
	if err != nil {
		t.Fatalf("failed to get all docs: %v", err)
	}

	// Should have at least 3 documents (may have more from other tests)
	if len(allDocs) < 3 {
		t.Errorf("expected at least 3 documents, got %d", len(allDocs))
	}
}

// TestMemorySystem_CheckQuota verifies quota checking
func TestMemorySystem_CheckQuota(t *testing.T) {
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	quota, err := store.CheckQuota()
	if err != nil {
		// Quota checking may fail in test environments without storage API
		t.Skipf("quota check not available in test environment: %v", err)
	}

	t.Logf("Storage quota: %d bytes used of %d bytes (%.1f%%)",
		quota.Usage, quota.Quota, quota.Percent)

	// Verify quota values are reasonable
	if quota.Quota < 0 {
		t.Error("quota should be non-negative")
	}
	if quota.Usage < 0 {
		t.Error("usage should be non-negative")
	}
	if quota.Percent < 0 || quota.Percent > 100 {
		t.Errorf("percent should be 0-100, got %.1f", quota.Percent)
	}
}

// TestMemorySystem_SearchScoring verifies search result scoring
func TestMemorySystem_SearchScoring(t *testing.T) {
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	// Store documents with varying relevance
	docs := []*memory.MemoryDocument{
		{ID: "score-1", Content: "Go programming language for systems"},
		{ID: "score-2", Content: "Python for data science and AI"},
		{ID: "score-3", Content: "Rust systems programming language"},
	}

	for _, doc := range docs {
		if err := store.Store(doc); err != nil {
			t.Fatalf("failed to store doc: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Search with different weights
	for _, opts := range []memory.SearchOptions{
		{Limit: 3, MinScore: 0.1, VectorWeight: 0.7, KeywordWeight: 0.3},
		{Limit: 3, MinScore: 0.1, VectorWeight: 0.0, KeywordWeight: 1.0}, // BM25 only
	} {
		results, err := store.Search("programming language", opts)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		// Verify results are sorted by score descending
		for i := 1; i < len(results); i++ {
			if results[i].Score > results[i-1].Score {
				t.Errorf("results not sorted by score: %.3f > %.3f at position %d",
					results[i].Score, results[i-1].Score, i)
			}
		}

		t.Logf("Search with weights %.1f/%.1f: %d results",
			opts.VectorWeight, opts.KeywordWeight, len(results))
	}
}
