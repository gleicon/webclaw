//go:build js && wasm

package memory

import (
	"math"
	"sort"
)

// HybridSearcher combines vector similarity and BM25 keyword search.
type HybridSearcher struct {
	bm25          *BM25Index
	vectorWeight  float64
	keywordWeight float64
}

// NewHybridSearcher creates a new hybrid searcher.
func NewHybridSearcher(bm25 *BM25Index, vectorWeight, keywordWeight float64) *HybridSearcher {
	// Normalize weights to sum to 1
	total := vectorWeight + keywordWeight
	if total == 0 {
		vectorWeight = 0.7
		keywordWeight = 0.3
		total = 1.0
	}

	return &HybridSearcher{
		bm25:          bm25,
		vectorWeight:  vectorWeight / total,
		keywordWeight: keywordWeight / total,
	}
}

// Search performs hybrid search and returns ranked results.
func (h *HybridSearcher) Search(query string, queryEmbedding []float32, docs []*MemoryDocument, opts SearchOptions) []*MemorySearchResult {
	if len(docs) == 0 {
		return []*MemorySearchResult{}
	}

	// Get BM25 scores
	bm25Scores := h.bm25.Search(query)

	// Create result map to merge vector and keyword scores
	type combinedResult struct {
		doc          *MemoryDocument
		vectorScore  float64
		keywordScore float64
		hybridScore  float64
	}

	results := make(map[string]*combinedResult)

	// Initialize with all documents
	for _, doc := range docs {
		results[doc.ID] = &combinedResult{
			doc:          doc,
			vectorScore:  0,
			keywordScore: 0,
			hybridScore:  0,
		}
	}

	// Apply BM25 scores
	for docID, score := range bm25Scores {
		if r, exists := results[docID]; exists {
			r.keywordScore = score
		}
	}

	// Apply vector similarity scores if embedding available
	if len(queryEmbedding) > 0 {
		for _, doc := range docs {
			if len(doc.Embedding) > 0 {
				similarity := cosineSimilarity(queryEmbedding, doc.Embedding)
				// Normalize similarity from [-1, 1] to [0, 1]
				normalized := (similarity + 1) / 2
				results[doc.ID].vectorScore = normalized
			}
		}
	}

	// Calculate hybrid scores
	for _, r := range results {
		r.hybridScore = h.vectorWeight*r.vectorScore + h.keywordWeight*r.keywordScore
	}

	// Convert to slice and sort
	var sorted []*combinedResult
	for _, r := range results {
		sorted = append(sorted, r)
	}

	// Sort by hybrid score (descending)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].hybridScore > sorted[j].hybridScore
	})

	// Filter by minimum score and limit
	var finalResults []*MemorySearchResult
	for _, r := range sorted {
		if r.hybridScore < opts.MinScore {
			continue
		}

		// Determine source
		source := "hybrid"
		if r.vectorScore == 0 {
			source = "keyword"
		} else if r.keywordScore == 0 {
			source = "vector"
		}

		finalResults = append(finalResults, &MemorySearchResult{
			Document: r.doc,
			Score:    r.hybridScore,
			Source:   source,
		})

		if len(finalResults) >= opts.Limit {
			break
		}
	}

	return finalResults
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Returns value in range [-1, 1] where 1 = identical, 0 = orthogonal, -1 = opposite.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// VectorSearch performs pure vector similarity search.
// Returns top-K results above minimum similarity threshold.
func VectorSearch(queryEmbedding []float32, docs []*MemoryDocument, limit int, minSimilarity float64) []*MemorySearchResult {
	if len(queryEmbedding) == 0 || len(docs) == 0 {
		return []*MemorySearchResult{}
	}

	type scoredDoc struct {
		doc   *MemoryDocument
		score float64
	}

	scored := make([]scoredDoc, 0, len(docs))
	for _, doc := range docs {
		if len(doc.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, doc.Embedding)
		// Normalize to [0, 1]
		normalized := (similarity + 1) / 2

		if normalized >= minSimilarity {
			scored = append(scored, scoredDoc{doc: doc, score: normalized})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Take top K
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Convert to results
	results := make([]*MemorySearchResult, len(scored))
	for i, s := range scored {
		results[i] = &MemorySearchResult{
			Document: s.doc,
			Score:    s.score,
			Source:   "vector",
		}
	}

	return results
}
