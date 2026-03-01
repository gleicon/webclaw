//go:build js && wasm

package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"syscall/js"
	"time"
)

// OpenAIEmbedder generates embeddings using OpenAI's text-embedding-3-small model.
type OpenAIEmbedder struct {
	apiKey     string
	model      string
	dimensions int
	baseURL    string
}

// NewOpenAIEmbedder creates a new OpenAI embedder.
func NewOpenAIEmbedder(apiKey string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:     apiKey,
		model:      "text-embedding-3-small",
		dimensions: 1536,
		baseURL:    "https://api.openai.com/v1",
	}
}

// NewOpenAIEmbedderWithModel creates a new OpenAI embedder with custom model.
func NewOpenAIEmbedderWithModel(apiKey, model string, dimensions int) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		baseURL:    "https://api.openai.com/v1",
	}
}

// Embed generates a vector embedding for the given text.
// Uses text-embedding-3-small which produces 1536-dimensional vectors.
func (o *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	if text == "" {
		return make([]float32, o.dimensions), nil
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"input": text,
		"model": o.model,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the request via JS fetch bridge
	resultChan := make(chan embeddingResult, 1)
	errChan := make(chan error, 1)

	// Use global fetch via syscall/js
	go func() {
		fetch := js.Global().Get("fetch")
		url := o.baseURL + "/embeddings"
		options := js.Global().Get("Object").New()

		// Set method
		options.Set("method", "POST")

		// Set headers
		headers := js.Global().Get("Object").New()
		headers.Set("Authorization", "Bearer "+o.apiKey)
		headers.Set("Content-Type", "application/json")
		options.Set("headers", headers)

		// Set body
		options.Set("body", string(jsonBody))

		promise := fetch.Invoke(url, options)

		promise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			response := args[0]

			// Check if response is OK
			if !response.Get("ok").Bool() {
				errChan <- fmt.Errorf("embeddings API returned error: %d", response.Get("status").Int())
				return nil
			}

			// Get JSON response
			response.Call("json").Call("then", js.FuncOf(func(_ js.Value, jsonArgs []js.Value) interface{} {
				data := jsonArgs[0]

				// Parse embedding from response
				embeddings := data.Get("data")
				if embeddings.IsUndefined() || embeddings.IsNull() || embeddings.Length() == 0 {
					errChan <- fmt.Errorf("no embeddings in response")
					return nil
				}

				firstEmbedding := embeddings.Index(0).Get("embedding")
				if firstEmbedding.IsUndefined() || firstEmbedding.IsNull() {
					errChan <- fmt.Errorf("no embedding data")
					return nil
				}

				// Convert JS array to Go []float32
				length := firstEmbedding.Length()
				embedding := make([]float32, length)
				for i := 0; i < length; i++ {
					embedding[i] = float32(firstEmbedding.Index(i).Float())
				}

				resultChan <- embeddingResult{embedding: embedding}
				return nil
			})).Call("catch", js.FuncOf(func(_ js.Value, catchArgs []js.Value) interface{} {
				errChan <- fmt.Errorf("failed to parse response: %v", catchArgs[0])
				return nil
			}))

			return nil
		})).Call("catch", js.FuncOf(func(_ js.Value, catchArgs []js.Value) interface{} {
			errChan <- fmt.Errorf("fetch failed: %v", catchArgs[0])
			return nil
		}))
	}()

	select {
	case result := <-resultChan:
		return result.embedding, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for embeddings API")
	}
}

// embeddingResult is a helper struct for channel communication.
type embeddingResult struct {
	embedding []float32
}

// MockEmbedder is a mock embedder for testing that returns predictable embeddings.
type MockEmbedder struct {
	Dimensions int
}

// NewMockEmbedder creates a mock embedder for testing.
func NewMockEmbedder(dimensions int) *MockEmbedder {
	return &MockEmbedder{Dimensions: dimensions}
}

// Embed generates a deterministic mock embedding based on text content.
func (m *MockEmbedder) Embed(text string) ([]float32, error) {
	// Create a deterministic embedding based on text hash
	embedding := make([]float32, m.Dimensions)

	// Simple hash-based embedding for testing
	var hash uint32 = 5381
	for _, c := range text {
		hash = ((hash << 5) + hash) + uint32(c)
	}

	// Fill embedding with values derived from hash
	for i := 0; i < m.Dimensions; i++ {
		// Create a pseudo-random but deterministic value
		hash = ((hash << 5) + hash) + uint32(i)
		// Normalize to range [-1, 1]
		embedding[i] = float32(int32(hash)) / float32(^uint32(0)>>1)
	}

	return embedding, nil
}

// CosineSimilarity calculates cosine similarity between two float32 vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt is a helper function since math.Sqrt takes float64.
func sqrt(x float64) float64 {
	// Simple Newton-Raphson method for square root
	if x == 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// normalizeVector normalizes a float32 vector to unit length.
func normalizeVector(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}

	if sum == 0 {
		return v
	}

	norm := sqrt(sum)
	result := make([]float32, len(v))
	for i, x := range v {
		result[i] = float32(float64(x) / norm)
	}

	return result
}

// SerializeEmbedding converts a float32 slice to bytes for storage.
func SerializeEmbedding(embedding []float32) []byte {
	buf := new(bytes.Buffer)
	for _, f := range embedding {
		bits := math.Float32bits(f)
		buf.Write([]byte{
			byte(bits),
			byte(bits >> 8),
			byte(bits >> 16),
			byte(bits >> 24),
		})
	}
	return buf.Bytes()
}

// DeserializeEmbedding converts bytes back to float32 slice.
func DeserializeEmbedding(data []byte, dimensions int) []float32 {
	if len(data) < dimensions*4 {
		return nil
	}

	embedding := make([]float32, dimensions)
	for i := 0; i < dimensions; i++ {
		offset := i * 4
		bits := uint32(data[offset]) |
			uint32(data[offset+1])<<8 |
			uint32(data[offset+2])<<16 |
			uint32(data[offset+3])<<24
		embedding[i] = float32(bits)
	}

	return embedding
}
