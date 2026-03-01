//go:build js && wasm

package config

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	CurrentVersion                = 1
	ConfigKey                     = "webclaw:config"
	DefaultBootstrapMaxChars      = 20000
	DefaultBootstrapTotalMaxChars = 150000
	DefaultMaxToolIterations      = 10
	DefaultTemperature            = 0.7
	DefaultMaxMemories            = 10000
)

// Config is the main configuration structure
type Config struct {
	Version   int                       `json:"version"`
	Identity  IdentityConfig            `json:"identity"`
	Agents    AgentsConfig              `json:"agents"`
	Providers map[string]ProviderConfig `json:"providers"`
	Memory    MemoryConfig              `json:"memory"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

// IdentityConfig holds identity-related settings
type IdentityConfig struct {
	Name                   string `json:"name"`
	BootstrapMaxChars      int    `json:"bootstrap_max_chars,omitempty"`
	BootstrapTotalMaxChars int    `json:"bootstrap_total_max_chars,omitempty"`
}

// AgentsConfig holds agent behavior settings
type AgentsConfig struct {
	DefaultModel      string  `json:"default_model"`
	MaxToolIterations int     `json:"max_tool_iterations"`
	Temperature       float64 `json:"temperature"`
}

// ProviderConfig holds LLM provider settings
type ProviderConfig struct {
	APIKeyEncrypted string `json:"api_key_encrypted,omitempty"`
	BaseURL         string `json:"base_url"`
}

// MemoryConfig holds memory system settings
type MemoryConfig struct {
	Enabled        bool   `json:"enabled"`
	MaxMemories    int    `json:"max_memories"`
	EmbeddingModel string `json:"embedding_model"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *Config {
	return &Config{
		Version: CurrentVersion,
		Identity: IdentityConfig{
			Name:                   "WebClaw",
			BootstrapMaxChars:      DefaultBootstrapMaxChars,
			BootstrapTotalMaxChars: DefaultBootstrapTotalMaxChars,
		},
		Agents: AgentsConfig{
			DefaultModel:      "anthropic/claude-sonnet-4-5",
			MaxToolIterations: DefaultMaxToolIterations,
			Temperature:       DefaultTemperature,
		},
		Providers: map[string]ProviderConfig{
			"anthropic": {
				BaseURL: "https://api.anthropic.com",
			},
			"openai": {
				BaseURL: "https://api.openai.com",
			},
			"openrouter": {
				BaseURL: "https://openrouter.ai/api",
			},
		},
		Memory: MemoryConfig{
			Enabled:        true,
			MaxMemories:    DefaultMaxMemories,
			EmbeddingModel: "openai/text-embedding-3-small",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate checks the config for errors
func (c *Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version: %d (expected %d)", c.Version, CurrentVersion)
	}

	if c.Identity.Name == "" {
		return fmt.Errorf("identity.name cannot be empty")
	}

	if c.Agents.Temperature < 0 || c.Agents.Temperature > 2 {
		return fmt.Errorf("agents.temperature must be between 0 and 2")
	}

	if c.Agents.MaxToolIterations < 1 || c.Agents.MaxToolIterations > 50 {
		return fmt.Errorf("agents.max_tool_iterations must be between 1 and 50")
	}

	// Validate providers
	for name, provider := range c.Providers {
		if provider.BaseURL == "" {
			return fmt.Errorf("providers.%s.base_url cannot be empty", name)
		}
	}

	return nil
}

// UpdateTimestamps updates the CreatedAt and UpdatedAt fields
func (c *Config) UpdateTimestamps() {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = time.Now()
}

// ToJSON serializes the config to JSON bytes
func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromJSON deserializes JSON bytes to a Config
func FromJSON(data []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &c, nil
}

// MergeProvider merges a provider config into the config
func (c *Config) MergeProvider(name string, provider ProviderConfig) {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	existing := c.Providers[name]
	if provider.BaseURL != "" {
		existing.BaseURL = provider.BaseURL
	}
	if provider.APIKeyEncrypted != "" {
		existing.APIKeyEncrypted = provider.APIKeyEncrypted
	}
	c.Providers[name] = existing
	c.UpdateTimestamps()
}
