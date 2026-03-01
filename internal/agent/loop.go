//go:build js && wasm

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/memory"
	"github.com/gleicon/webclaw/internal/provider"
	"github.com/gleicon/webclaw/internal/tools"
)

// Provider defines the interface for LLM providers used by the agent loop.
// The callback receives full provider.Token structs so that tool_use metadata
// (ToolName, ToolInput, ToolUseID, FinishReason) can flow back to the caller.
type Provider interface {
	Stream(ctx context.Context, messages []Message, callback func(tok provider.Token)) error
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

	// router is the real LLM provider router.
	// When set, getProvider() returns a providerAdapter wrapping this router.
	// When nil, falls back to mockProvider for development/testing.
	router *provider.Router

	// toolRegistry holds all registered tools for dispatch.
	// When nil, tool calls return "tool registry not configured".
	toolRegistry *tools.Registry

	// workerBridge emits tool events to the UI via postMessage.
	// Using an interface avoids import cycle since worker_bridge.go imports agent.
	workerBridge interface{ EmitToolEvent(string, string, string, string) }
}

// SetRouter wires the real provider router so getProvider() returns a real LLM,
// not the mockProvider. Call this in main.go after constructing AgentLoop.
func (al *AgentLoop) SetRouter(r *provider.Router) {
	al.router = r
}

// SetToolRegistry wires the tool registry so tool calls are dispatched through it.
// Without this call, every tool invocation returns "tool registry not configured".
// Call this in main.go after constructing AgentLoop.
func (al *AgentLoop) SetToolRegistry(r *tools.Registry) {
	al.toolRegistry = r
}

// SetWorkerBridge wires the worker bridge so EmitToolEvent calls from the dispatch
// loop reach the UI. Call this in main.go after constructing both AgentLoop and
// the WorkerBridge instance returned by InitWorkerBridge.
func (al *AgentLoop) SetWorkerBridge(wb interface{ EmitToolEvent(string, string, string, string) }) {
	al.workerBridge = wb
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

// maxToolIterations is the maximum number of tool call rounds per Run().
// Prevents infinite loops in agentic tool use.
const maxToolIterations = 10

// Run executes a single agent turn with tool dispatch loop.
// 1. Assembles context (system prompt + identity + history)
// 2. Calls provider with streaming
// 3. If provider returns tool_use, dispatches through toolRegistry and loops
// 4. Emits tokens and tool events via worker bridge callbacks
// 5. Handles completion/errors
func (al *AgentLoop) Run(ctx context.Context, messages []Message, bridge *WorkerBridge) error {
	startTime := time.Now()

	js.Global().Get("console").Call("log", "webclaw: agent loop starting", al.providerName)

	// If no assembler is set, we need messages to be provided
	if al.assembler == nil && len(messages) == 0 {
		return fmt.Errorf("no context assembler and no messages provided")
	}

	// Get or create provider
	prov, err := al.getProvider()
	if err != nil {
		bridge.EmitError(fmt.Errorf("failed to get provider: %w", err))
		return err
	}

	// Use provided messages or assemble from context
	requestMessages := messages
	if len(requestMessages) == 0 {
		requestMessages = al.assembler.GetConversation().GetMessagesForAPI()
	}

	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		bridge.EmitError(fmt.Errorf("stream cancelled before start: %w", ctx.Err()))
		return ctx.Err()
	default:
	}

	var responseContent string
	tokenCount := 0
	firstTokenTime := time.Time{}

	// Tool dispatch loop — runs up to maxToolIterations.
	// On normal completion (FinishReason != "tool_use"), breaks and returns.
	for iter := 0; iter < maxToolIterations; iter++ {
		var lastTok provider.Token
		var iterContent string

		streamErr := prov.Stream(ctx, requestMessages, func(tok provider.Token) {
			// Track first token timing across iterations
			if tokenCount == 0 {
				firstTokenTime = time.Now()
				latency := firstTokenTime.Sub(startTime)
				js.Global().Get("console").Call("log", "webclaw: first token latency:", latency.Milliseconds(), "ms")
			}

			if tok.Text != "" {
				tokenCount++
				iterContent += tok.Text
				responseContent += tok.Text
				bridge.EmitToken(tok.Text)
			}
			lastTok = tok // always capture; final token carries FinishReason + tool metadata

			// Check for cancellation between tokens
			select {
			case <-ctx.Done():
				js.Global().Get("console").Call("log", "webclaw: stream aborted during token processing")
				return
			default:
			}
		})

		if streamErr != nil {
			if ctx.Err() != nil {
				js.Global().Get("console").Call("log", "webclaw: stream cancelled successfully")
				bridge.EmitComplete(true, responseContent)
				return nil
			}
			js.Global().Get("console").Call("error", "webclaw: stream error:", streamErr.Error())
			bridge.EmitError(fmt.Errorf("stream error: %w", streamErr))
			return streamErr
		}

		// --- Normal completion ---
		if lastTok.FinishReason != "tool_use" {
			duration := time.Since(startTime)
			js.Global().Get("console").Call("log", "webclaw: stream completed",
				"tokens:", tokenCount,
				"duration:", duration.Seconds(), "s",
				"tps:", float64(tokenCount)/duration.Seconds())

			if al.assembler != nil {
				al.assembler.AddAssistantResponse(responseContent)
				if summary, triggered := al.assembler.CheckAndSummarize(); triggered {
					js.Global().Get("console").Call("log", "webclaw: conversation summarized", summary.MessageCount)
				}
			}

			bridge.EmitComplete(true, responseContent)
			return nil
		}

		// --- Tool use detected ---
		toolName := lastTok.ToolName
		toolInput := lastTok.ToolInput
		toolUseID := lastTok.ToolUseID

		js.Global().Get("console").Call("log", "webclaw: tool use detected", toolName)

		// Emit "running" event before dispatch
		if al.workerBridge != nil {
			al.workerBridge.EmitToolEvent(toolName, "running", "Running "+toolName+"...", "")
		}

		// Dispatch through registry
		var result *tools.ToolResult
		if al.toolRegistry != nil {
			result, err = al.toolRegistry.Dispatch(ctx, toolName, toolInput)
			if err != nil {
				result = &tools.ToolResult{
					IsError:        true,
					Content:        err.Error(),
					DisplayContent: err.Error(),
					ToolName:       toolName,
					Status:         "error",
				}
			}
		} else {
			js.Global().Get("console").Call("warn", "webclaw: tool registry not configured")
			result = &tools.ToolResult{
				IsError:        true,
				Content:        "tool registry not configured",
				DisplayContent: "tool registry not configured",
				ToolName:       toolName,
				Status:         "error",
			}
		}

		// Emit "done" or "error" event after dispatch
		if al.workerBridge != nil {
			al.workerBridge.EmitToolEvent(toolName, result.Status, result.DisplayContent, result.Content)
		}

		// Inject tool_use + tool_result into message history for next LLM call.
		// Format as JSON strings since the Message.Content field is a string.
		toolUseJSON, _ := json.Marshal(map[string]interface{}{
			"type":  "tool_use",
			"id":    toolUseID,
			"name":  toolName,
			"input": toolInput,
		})
		toolResultJSON, _ := json.Marshal(map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": toolUseID,
			"content":     result.Content,
			"is_error":    result.IsError,
		})

		requestMessages = append(requestMessages,
			Message{Role: "assistant", Content: iterContent + string(toolUseJSON)},
			Message{Role: "user", Content: string(toolResultJSON)},
		)
		// Loop: next iteration sends updated history back to LLM
	}

	// Iteration limit reached — emit what we have and return error
	js.Global().Get("console").Call("warn", "webclaw: tool iteration limit reached", maxToolIterations)
	bridge.EmitComplete(true, responseContent)
	return fmt.Errorf("tool iteration limit (%d) reached", maxToolIterations)
}

// getProvider returns a provider instance based on configuration.
// When a real router is configured (via SetRouter), returns a providerAdapter
// wrapping that router. Falls back to mockProvider when no router is set
// (allows tests and development without API keys).
func (al *AgentLoop) getProvider() (Provider, error) {
	if al.router != nil {
		return &providerAdapter{
			router: al.router,
			name:   al.providerName,
			model:  al.model,
		}, nil
	}
	// Fallback: mock provider when no router configured
	return &mockProvider{
		name:  al.providerName,
		model: al.model,
	}, nil
}

// providerAdapter wraps provider.Router to match the agent.Provider interface.
// The agent loop uses callback-based streaming; provider.Router uses channels.
// This adapter bridges the two by consuming the channel and calling the callback.
type providerAdapter struct {
	router *provider.Router
	name   string
	model  string
}

func (pa *providerAdapter) Stream(ctx context.Context, messages []Message, callback func(tok provider.Token)) error {
	// Convert agent []Message to provider []Message
	provMsgs := make([]provider.Message, len(messages))
	for i, m := range messages {
		provMsgs[i] = provider.Message{Role: m.Role, Content: m.Content}
	}
	req := provider.CompletionRequest{
		Model:       pa.model,
		Messages:    provMsgs,
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
	}
	ch, err := pa.router.Stream(ctx, pa.model, req)
	if err != nil {
		return err
	}
	for tok := range ch {
		if tok.FinishReason == "error" {
			return fmt.Errorf("provider error: %s", tok.Text)
		}
		callback(tok)
	}
	return nil
}

func (pa *providerAdapter) GetName() string { return pa.name }
func (pa *providerAdapter) GetModel() string { return pa.model }

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

func (mp *mockProvider) Stream(ctx context.Context, messages []Message, callback func(tok provider.Token)) error {
	// Simulate streaming with a mock response
	mockResponse := "This is a mock response from the " + mp.name + " provider using model " + mp.model + ". "
	mockResponse += "The agent loop is working correctly and streaming tokens to the UI."

	// Stream tokens with small delay
	words := splitIntoWords(mockResponse)
	for i, word := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond): // 50ms between tokens
			tok := provider.Token{Text: word + " "}
			// Mark the last token with stop reason
			if i == len(words)-1 {
				tok.FinishReason = "stop"
			}
			callback(tok)
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
