//go:build js && wasm

package provider

import (
	"context"
	"fmt"
	"strings"
)

// Router routes requests to the appropriate provider based on model ID
type Router struct {
	providers map[string]*ProviderChain // Changed from Provider to ProviderChain
	config    *Config
	retry     RetryConfig
	fallbacks map[string]fallbackConfig // primary -> fallback mapping
}

// fallbackConfig holds fallback provider configuration
type fallbackConfig struct {
	providerName string
	modelID      string
}

// NewRouter creates a new provider router with the given configuration
func NewRouter(config *Config) *Router {
	r := &Router{
		providers: make(map[string]*ProviderChain),
		config:    config,
		retry:     DefaultRetryConfig(),
		fallbacks: make(map[string]fallbackConfig),
	}

	// Gemini Nano is always registered — availability is checked at runtime via JS.
	r.providers["gemini-nano"] = NewProviderChain(NewGeminiNanoProvider(), "local")

	// Register providers based on available API keys
	if config.AnthropicAPIKey != "" {
		anthropicProv := NewAnthropicProvider(config.AnthropicAPIKey)
		r.providers["anthropic"] = NewProviderChain(anthropicProv, "claude-sonnet-4-5")
	}

	if config.OpenAIAPIKey != "" {
		openaiProv := NewOpenAIProvider(config.OpenAIAPIKey)
		r.providers["openai"] = NewProviderChain(openaiProv, "gpt-4o-mini")
	}

	if config.OpenRouterAPIKey != "" {
		openrouterProv := NewOpenRouterProvider(
			config.OpenRouterAPIKey,
			config.HTTPReferer,
			config.XTitle,
		)
		r.providers["openrouter"] = NewProviderChain(openrouterProv, "anthropic/claude-3-haiku")
	}

	return r
}

// RegisterProvider manually registers a provider (useful for testing)
// Wraps the provider in a ProviderChain with retry support
func (r *Router) RegisterProvider(name string, p Provider) {
	// Get model ID from provider if available
	modelID := "default"
	if mp, ok := p.(interface{ GetModel() string }); ok {
		modelID = mp.GetModel()
	}

	// Wrap in ProviderChain with retry config
	chain := NewProviderChain(p, modelID)
	chain.SetRetryConfig(r.retry)
	r.providers[name] = chain
}

// HasProvider checks if a provider is available
func (r *Router) HasProvider(name string) bool {
	_, ok := r.providers[name]
	return ok
}

// RouteResult contains the provider and normalized model ID after routing
type RouteResult struct {
	Provider Provider
	ModelID  string // Normalized model ID (without vendor prefix)
	Vendor   string
}

// Route parses a vendor/model-id string and returns the appropriate provider
// Supported formats:
//   - "anthropic/claude-sonnet-4-5" → Anthropic provider
//   - "openai/gpt-4" → OpenAI provider
//   - "openrouter/anthropic/claude-sonnet-4-5" → OpenRouter provider with full model path
//   - "claude-sonnet-4-5" → defaults to Anthropic if available
func (r *Router) Route(modelID string) (*RouteResult, error) {
	if modelID == "" {
		return nil, ErrInvalidModel
	}

	// Parse vendor/model format
	vendor, model, err := ParseModelID(modelID)
	if err != nil {
		// No vendor prefix - try to infer from model name
		return r.routeByModelName(modelID)
	}

	// Normalize vendor aliases
	vendor = normalizeVendor(vendor)

	// Look up provider
	provider, ok := r.providers[vendor]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, vendor)
	}

	// Handle OpenRouter nested vendor format (openrouter/anthropic/claude-...)
	if vendor == "openrouter" && strings.Contains(model, "/") {
		// Keep the full path for OpenRouter (it routes to underlying providers)
		return &RouteResult{
			Provider: provider,
			ModelID:  model,
			Vendor:   vendor,
		}, nil
	}

	return &RouteResult{
		Provider: provider,
		ModelID:  model,
		Vendor:   vendor,
	}, nil
}

// routeByModelName attempts to route based on model name patterns
func (r *Router) routeByModelName(model string) (*RouteResult, error) {
	modelLower := strings.ToLower(model)

	// Anthropic models
	if strings.Contains(modelLower, "claude") {
		if provider, ok := r.providers["anthropic"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  model,
				Vendor:   "anthropic",
			}, nil
		}
		// Fallback to OpenRouter if available
		if provider, ok := r.providers["openrouter"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  "anthropic/" + model,
				Vendor:   "openrouter",
			}, nil
		}
	}

	// OpenAI models
	if strings.HasPrefix(modelLower, "gpt-") {
		if provider, ok := r.providers["openai"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  model,
				Vendor:   "openai",
			}, nil
		}
		// Fallback to OpenRouter if available
		if provider, ok := r.providers["openrouter"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  "openai/" + model,
				Vendor:   "openrouter",
			}, nil
		}
	}

	// Meta models
	if strings.Contains(modelLower, "llama") {
		if provider, ok := r.providers["openrouter"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  "meta-llama/" + model,
				Vendor:   "openrouter",
			}, nil
		}
	}

	// Google models
	if strings.Contains(modelLower, "gemini") {
		if provider, ok := r.providers["openrouter"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  "google/" + model,
				Vendor:   "openrouter",
			}, nil
		}
	}

	// Mistral models
	if strings.Contains(modelLower, "mistral") || strings.Contains(modelLower, "mixtral") {
		if provider, ok := r.providers["openrouter"]; ok {
			return &RouteResult{
				Provider: provider,
				ModelID:  "mistralai/" + model,
				Vendor:   "openrouter",
			}, nil
		}
	}

	// Default to first available provider
	for vendor, provider := range r.providers {
		return &RouteResult{
			Provider: provider,
			ModelID:  model,
			Vendor:   vendor,
		}, nil
	}

	return nil, ErrProviderNotFound
}

// normalizeVendor normalizes vendor name aliases
func normalizeVendor(vendor string) string {
	switch strings.ToLower(vendor) {
	case "anthropic", "claude":
		return "anthropic"
	case "openai", "gpt":
		return "openai"
	case "openrouter", "router":
		return "openrouter"
	default:
		return strings.ToLower(vendor)
	}
}

// Complete routes and executes a completion request
func (r *Router) Complete(ctx context.Context, modelID string, req CompletionRequest) (*Token, error) {
	route, err := r.Route(modelID)
	if err != nil {
		return nil, err
	}

	// Update request with normalized model ID
	req.Model = route.ModelID

	return route.Provider.Complete(ctx, req)
}

// Stream routes and executes a streaming completion request
func (r *Router) Stream(ctx context.Context, modelID string, req CompletionRequest) (<-chan Token, error) {
	route, err := r.Route(modelID)
	if err != nil {
		return nil, err
	}

	// Update request with normalized model ID
	req.Model = route.ModelID

	return route.Provider.Stream(ctx, req), nil
}

// Embed routes and executes an embedding request
// Note: Only OpenAI supports embeddings in current implementation
func (r *Router) Embed(ctx context.Context, modelID string, input string) ([]float32, error) {
	route, err := r.Route(modelID)
	if err != nil {
		return nil, err
	}

	return route.Provider.Embed(ctx, input)
}

// ConditionalProvider is implemented by providers whose availability depends on
// runtime conditions (e.g. browser API support). AvailableProviders skips any
// provider that returns false.
type ConditionalProvider interface {
	IsAvailable() bool
}

// AvailableProviders returns registered provider names that are currently available.
// Providers implementing ConditionalProvider are only included if IsAvailable() is true.
func (r *Router) AvailableProviders() []string {
	names := make([]string, 0, len(r.providers))
	for name, chain := range r.providers {
		if cp, ok := chain.primary.(ConditionalProvider); ok {
			if !cp.IsAvailable() {
				continue
			}
		}
		names = append(names, name)
	}
	return names
}

// GetProvider returns a specific provider by name
func (r *Router) GetProvider(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}

// ValidateModelID checks if a model ID can be routed
func (r *Router) ValidateModelID(modelID string) error {
	_, err := r.Route(modelID)
	return err
}

// ModelInfo contains information about a model
type ModelInfo struct {
	ID         string
	Vendor     string
	Provider   string
	MaxContext int
	Available  bool
}

// GetModelInfo returns information about a model
func (r *Router) GetModelInfo(modelID string) (*ModelInfo, error) {
	route, err := r.Route(modelID)
	if err != nil {
		return nil, err
	}

	return &ModelInfo{
		ID:         route.ModelID,
		Vendor:     route.Vendor,
		Provider:   route.Provider.Name(),
		MaxContext: route.Provider.MaxContextWindow(route.ModelID),
		Available:  true,
	}, nil
}

// SetFallback configures a fallback provider for a primary provider
// When the primary fails after retries, the fallback provider will be used
func (r *Router) SetFallback(primaryName string, fallbackName string, fallbackModel string) {
	r.fallbacks[primaryName] = fallbackConfig{
		providerName: fallbackName,
		modelID:      fallbackModel,
	}

	// Apply fallback to the chain if both providers exist
	if primaryChain, ok := r.providers[primaryName]; ok {
		if fallbackChain, ok := r.providers[fallbackName]; ok {
			// Extract the actual provider from the fallback chain
			primaryChain.SetFallback(fallbackChain, fallbackModel)
		}
	}
}

// SetRetryConfig sets the retry configuration for all providers in the router
func (r *Router) SetRetryConfig(config RetryConfig) {
	r.retry = config
	// Update all existing provider chains
	for _, chain := range r.providers {
		chain.SetRetryConfig(config)
	}
}

// ChainResult combines multiple providers into a fallback chain
// This is the foundation for the failover implementation
type ChainResult struct {
	Primary  Provider
	Fallback Provider
	ModelID  string
}

// CreateChain creates a provider chain with fallback for a model
func (r *Router) CreateChain(modelID string, fallbackModelID string) (*ChainResult, error) {
	primary, err := r.Route(modelID)
	if err != nil {
		return nil, fmt.Errorf("primary model: %w", err)
	}

	var fallback *RouteResult
	if fallbackModelID != "" {
		fallback, err = r.Route(fallbackModelID)
		if err != nil {
			return nil, fmt.Errorf("fallback model: %w", err)
		}
	}

	return &ChainResult{
		Primary:  primary.Provider,
		Fallback: fallback.Provider,
		ModelID:  primary.ModelID,
	}, nil
}

// compile check for routerProvider
var _ Provider = (*routerProvider)(nil)

// Router implements Provider interface for convenience
type routerProvider struct {
	router  *Router
	modelID string
}

// Complete implements Provider by routing to appropriate provider
func (rp *routerProvider) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	return rp.router.Complete(ctx, rp.modelID, req)
}

// Stream implements Provider by routing to appropriate provider
func (rp *routerProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	ch, err := rp.router.Stream(ctx, rp.modelID, req)
	if err != nil {
		// Return error as token
		result := make(chan Token, 1)
		result <- Token{FinishReason: "error", Text: err.Error()}
		close(result)
		return result
	}
	return ch
}

// Embed implements Provider by routing to appropriate provider
func (rp *routerProvider) Embed(ctx context.Context, input string) ([]float32, error) {
	return rp.router.Embed(ctx, rp.modelID, input)
}

// Name returns "router" as the provider name
func (rp *routerProvider) Name() string {
	return "router"
}

// MaxContextWindow returns the context window for the routed model
func (rp *routerProvider) MaxContextWindow(model string) int {
	route, err := rp.router.Route(rp.modelID)
	if err != nil {
		return 4096 // Conservative default
	}
	return route.Provider.MaxContextWindow(route.ModelID)
}

// NewRouterProvider creates a provider that uses the router internally
// This allows the router to be used anywhere a Provider is expected
func NewRouterProvider(router *Router, modelID string) Provider {
	return &routerProvider{
		router:  router,
		modelID: modelID,
	}
}
