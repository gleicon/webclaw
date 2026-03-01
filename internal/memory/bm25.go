//go:build js && wasm

package memory

import (
	"math"
	"strings"
	"unicode"
)

// BM25Index implements BM25 keyword search indexing.
// Uses standard BM25 formula with k1=1.2, b=0.75
type BM25Index struct {
	docCount   int                       // Total number of documents
	avgDocLen  float64                   // Average document length
	index      map[string]map[string]int // term -> docID -> term frequency
	docLengths map[string]int            // docID -> document length in tokens
}

// NewBM25Index creates a new BM25 index.
func NewBM25Index() *BM25Index {
	return &BM25Index{
		index:      make(map[string]map[string]int),
		docLengths: make(map[string]int),
	}
}

// AddDocument adds a document to the BM25 index.
func (b *BM25Index) AddDocument(docID string, content string) {
	// Tokenize and remove existing if present
	b.RemoveDocument(docID)

	// Tokenize content
	tokens := tokenize(content)
	if len(tokens) == 0 {
		return
	}

	// Update document length
	b.docLengths[docID] = len(tokens)

	// Count term frequencies
	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// Add to index
	for term, freq := range termFreq {
		if b.index[term] == nil {
			b.index[term] = make(map[string]int)
		}
		b.index[term][docID] = freq
	}

	// Update document count and average length
	b.docCount++
	b.updateAvgDocLen()
}

// RemoveDocument removes a document from the BM25 index.
func (b *BM25Index) RemoveDocument(docID string) {
	// Remove from all term postings
	for term, postings := range b.index {
		if _, exists := postings[docID]; exists {
			delete(postings, docID)
			// Clean up empty terms
			if len(postings) == 0 {
				delete(b.index, term)
			}
		}
	}

	// Remove document length
	if _, exists := b.docLengths[docID]; exists {
		delete(b.docLengths, docID)
		b.docCount--
		b.updateAvgDocLen()
	}
}

// Search performs a BM25 search and returns scored document IDs.
// Scores are normalized to 0-1 range.
func (b *BM25Index) Search(query string) map[string]float64 {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 || b.docCount == 0 {
		return make(map[string]float64)
	}

	// Score each document
	scores := make(map[string]float64)
	for _, term := range queryTokens {
		postings := b.index[term]
		if len(postings) == 0 {
			continue
		}

		// Calculate IDF
		df := float64(len(postings)) // Document frequency
		idf := math.Log(float64(b.docCount)-df+0.5) / (df + 0.5)

		// Score each document containing this term
		for docID, tf := range postings {
			docLen := float64(b.docLengths[docID])
			score := b.bm25Score(tf, docLen, idf)
			scores[docID] += score
		}
	}

	// Normalize scores to 0-1 range
	if len(scores) > 0 {
		maxScore := 0.0
		for _, score := range scores {
			if score > maxScore {
				maxScore = score
			}
		}
		if maxScore > 0 {
			for docID := range scores {
				scores[docID] /= maxScore
			}
		}
	}

	return scores
}

// bm25Score calculates the BM25 score for a single term-document pair.
func (b *BM25Index) bm25Score(tf int, docLen float64, idf float64) float64 {
	k1 := 1.2
	bParam := 0.75

	numerator := float64(tf) * (k1 + 1)
	denominator := float64(tf) + k1*(1-bParam+bParam*(docLen/b.avgDocLen))

	return idf * numerator / denominator
}

// updateAvgDocLen recalculates the average document length.
func (b *BM25Index) updateAvgDocLen() {
	if b.docCount == 0 {
		b.avgDocLen = 0
		return
	}

	totalLen := 0
	for _, length := range b.docLengths {
		totalLen += length
	}
	b.avgDocLen = float64(totalLen) / float64(b.docCount)
}

// tokenize splits text into lowercase tokens.
func tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split on non-alphanumeric characters
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			token := current.String()
			if len(token) > 1 { // Filter single-character tokens
				tokens = append(tokens, token)
			}
			current.Reset()
		}
	}

	// Don't forget the last token
	if current.Len() > 0 {
		token := current.String()
		if len(token) > 1 {
			tokens = append(tokens, token)
		}
	}

	return tokens
}
