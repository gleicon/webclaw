//go:build js && wasm

package agent

import (
	"context"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/memory"
)

// Provider defines the interface for LLM providers
// This is a placeholder until the actual provider interface is implemented
type Provider interface {
	Stream(ctx context.Context, messages []Message, callback func(token string)) error
	GetName() string
	GetModel() string
}

// AgentLoop orchestrates the agent turn: context → provider → response
// Runs in a goroutine spawned from the worker bridge
type AgentLoop struct {
	providerName string
	model        string
	assembler    *ContextAssembler
	memory       memory.Store
	embedder     memory.Embedder
}

// NewAgentLoop creates a new agent loop for the specified provider
func NewAgentLoop(providerName, model string) *AgentLoop {
	return &AgentLoop{
		providerName: providerName,
		model:        model,
	}
}

// NewAgentLoopWithAssembler creates an agent loop with a pre-configured context assembler
func NewAgentLoopWithAssembler(providerName, model string, assembler *ContextAssembler) *AgentLoop {
	return &AgentLoop{
		providerName: providerName,
		model:        model,
		assembler:    assembler,
	}
}

// Run executes a single agent turn
// 1. Assembles context (system prompt + identity + history)
// 2. Calls provider with streaming
// 3. Emits tokens via worker bridge callbacks
// 4. Handles completion/errors
func (al *AgentLoop) Run(ctx context.Context, messages []Message, bridge *WorkerBridge) error {
	startTime := time.Now()

	js.Global().Get("console").Call("log", "webclaw: agent loop starting", al.providerName)

	// If no assembler is set, we need messages to be provided
	if al.assembler == nil && len(messages) == 0 {
		return fmt.Errorf("no context assembler and no messages provided")
	}

	// Get or create provider
	provider, err := al.getProvider()
	if err != nil {
		bridge.EmitError(fmt.Errorf("failed to get provider: %w", err))
		return err
	}

	// Use provided messages or assemble from context
	var requestMessages []Message
	if len(messages) > 0 {
		requestMessages = messages
	} else {
		// Need user message to assemble context
		// This path is for when we have an assembler but no explicit messages
		// The last message should be the user query
		requestMessages = al.assembler.GetConversation().GetMessagesForAPI()
	}

	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		bridge.EmitError(fmt.Errorf("stream cancelled before start: %w", ctx.Err()))
		return ctx.Err()
	default:
	}

	// Start the provider stream
	var responseContent string
	tokenCount := 0
	firstTokenTime := time.Time{}

	streamErr := provider.Stream(ctx, requestMessages, func(token string) {
		// Track first token timing
		if tokenCount == 0 {
			firstTokenTime = time.Now()
			latency := firstTokenTime.Sub(startTime)
			js.Global().Get("console").Call("log", "webclaw: first token latency:", latency.Milliseconds(), "ms")
		}

		tokenCount++
		responseContent += token

		// Emit token to UI via worker bridge
		bridge.EmitToken(token)

		// Check for cancellation between tokens
		select {
		case <-ctx.Done():
			// Stream is being aborted, stop processing tokens
			js.Global().Get("console").Call("log", "webclaw: stream aborted during token processing")
			return
		default:
		}
	})

	if streamErr != nil {
		// Check if this was a cancellation error
		if ctx.Err() != nil {
			js.Global().Get("console").Call("log", "webclaw: stream cancelled successfully")
			bridge.EmitComplete(true, responseContent) // Emit what we have so far
			return nil
		}

		js.Global().Get("console").Call("error", "webclaw: stream error:", streamErr.Error())
		bridge.EmitError(fmt.Errorf("stream error: %w", streamErr))
		return streamErr
	}

	// Stream completed successfully
	duration := time.Since(startTime)
	js.Global().Get("console").Call("log", "webclaw: stream completed",
		"tokens:", tokenCount,
		"duration:", duration.Seconds(), "s",
		"tps:", float64(tokenCount)/duration.Seconds())

	// Add assistant response to conversation history if we have an assembler
	if al.assembler != nil {
		al.assembler.AddAssistantResponse(responseContent)

		// Check if we need summarization
		if summary, triggered := al.assembler.CheckAndSummarize(); triggered {
			js.Global().Get("console").Call("log", "webclaw: conversation summarized", summary.MessageCount)
		}
	}

	// Signal completion
	bridge.EmitComplete(true, responseContent)

	return nil
}

// getProvider returns a provider instance based on configuration
// This is a placeholder that returns a mock provider for now
func (al *AgentLoop) getProvider() (Provider, error) {
	// For now, return a mock provider that simulates streaming
	// In a real implementation, this would look up the provider from a registry
	return &mockProvider{
		name:  al.providerName,
		model: al.model,
	}, nil
}

// SetAssembler sets the context assembler for conversation management
func (al *AgentLoop) SetAssembler(assembler *ContextAssembler) {
	al.assembler = assembler
}

// GetAssembler returns the current context assembler
func (al *AgentLoop) GetAssembler() *ContextAssembler {
	return al.assembler
}

// mockProvider is a placeholder provider for testing
// Streams mock tokens to verify the pipeline works
type mockProvider struct {
	name  string
	model string
}

func (mp *mockProvider) Stream(ctx context.Context, messages []Message, callback func(token string)) error {
	// Simulate streaming with a mock response
	mockResponse := "This is a mock response from the " + mp.name + " provider using model " + mp.model + ". "
	mockResponse += "The agent loop is working correctly and streaming tokens to the UI."

	// Stream tokens with small delay
	tokens := []string{}
	for _, word := range splitIntoWords(mockResponse) {
		tokens = append(tokens, word+" ")
	}

	for _, token := range tokens {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond): // 50ms between tokens
			callback(token)
		}
	}

	return nil
}

func (mp *mockProvider) GetName() string {
	return mp.name
}

func (mp *mockProvider) GetModel() string {
	return mp.model
}

// splitIntoWords splits a string into words for token simulation
func splitIntoWords(text string) []string {
	// Simple word split - in production would use actual tokenization
	words := []string{}
	start := 0
	for i, c := range text {
		if c == ' ' || c == '.' || c == ',' || c == '!' || c == '?' {
			if i > start {
				words = append(words, text[start:i])
			}
			if c != ' ' {
				words = append(words, string(c))
			}
			start = i + 1
		}
	}
	if start < len(text) {
		words = append(words, text[start:])
	}
	return words
}

// SetMemoryStore sets the memory store for the agent loop.
func (al *AgentLoop) SetMemoryStore(store memory.Store) {
	al.memory = store
}

// SetEmbedder sets the embedding generator for memory operations.
func (al *AgentLoop) SetEmbedder(embedder memory.Embedder) {
	al.embedder = embedder
}

// SearchMemory searches for relevant memories based on query.
func (al *AgentLoop) SearchMemory(query string, limit int) ([]*memory.MemorySearchResult, error) {
	if al.memory == nil {
		return nil, fmt.Errorf("memory store not initialized")
	}

	opts := memory.SearchOptions{
		Limit:         limit,
		MinScore:      0.5,
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
	}

	return al.memory.Search(query, opts)
}

// StoreFact stores a fact in memory.
func (al *AgentLoop) StoreFact(content string, metadata map[string]interface{}) error {
	if al.memory == nil {
		return fmt.Errorf("memory store not initialized")
	}

	// Generate embedding if embedder is available
	var embedding []float32
	if al.embedder != nil {
		var err error
		embedding, err = al.embedder.Embed(content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Create memory document
	doc := memory.NewMemoryDocument(
		generateMemoryID(),
		content,
		embedding,
	)

	if metadata != nil {
		doc.Metadata = metadata
	}

	// Store in memory
	if err := al.memory.Store(doc); err != nil {
		return fmt.Errorf("failed to store fact: %w", err)
	}

	// Check if eviction is needed
	go al.memory.EvictIfNeeded()

	return nil
}

// generateMemoryID creates a unique memory ID.
func generateMemoryID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

// EnhanceContextWithMemory searches memory and adds relevant facts to context.
func (al *AgentLoop) EnhanceContextWithMemory(query string) ([]*memory.MemorySearchResult, error) {
	results, err := al.SearchMemory(query, 5)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		js.Global().Get("console").Call("log", "webclaw: found relevant memories:", len(results))
		for i, result := range results {
			js.Global().Get("console").Call("log", fmt.Sprintf("  %d. Score: %.3f - %s", i+1, result.Score, result.Document.Content))
		}
	}

	return results, nil
}

// ExtractAndStoreFacts extracts facts from conversation and stores them in memory.
// This is called after a successful response to capture important information.
func (al *AgentLoop) ExtractAndStoreFacts(userMessage, assistantResponse string) {
	if al.memory == nil {
		return
	}

	// Simple fact extraction: store key information from the conversation
	// In a real implementation, this would use an LLM to extract facts

	// Store user message as a memory if it's substantial
	if len(userMessage) > 50 {
		go func() {
			err := al.StoreFact(userMessage, map[string]interface{}{
				"type":     "user_message",
				"source":   "conversation",
				"response": assistantResponse,
			})
			if err != nil {
				js.Global().Get("console").Call("error", "webclaw: failed to store user message:", err.Error())
			}
		}()
	}
}
