//go:build js && wasm

package config

import (
	"encoding/json"
	"fmt"
	"time"
)

const ExportVersion = "1.0"

// ExportData represents the complete export format
type ExportData struct {
	Version       string                  `json:"version"`
	ExportDate    string                  `json:"export_date"`
	ExportVersion string                  `json:"export_version"`
	Config        *Config                 `json:"config"`
	IdentityFiles map[string]string       `json:"identity_files"`
	EncryptedKeys map[string]EncryptedKey `json:"encrypted_keys,omitempty"`
}

// EncryptedKey represents a stored encrypted API key
type EncryptedKey struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Salt       string `json:"salt"`
}

// IdentityFileProvider interface for getting identity files without direct import
type IdentityFileProvider interface {
	List() ([]string, error)
	GetContent(filename string) (string, error)
}

// ExportAll gathers config, identity files, and encrypted keys for export
func ExportAll(cfg *Config, idProvider IdentityFileProvider, ks KeyStoreExporter) (*ExportData, error) {
	data := &ExportData{
		Version:       "1",
		ExportDate:    time.Now().UTC().Format(time.RFC3339),
		ExportVersion: ExportVersion,
		Config:        cfg,
		IdentityFiles: make(map[string]string),
		EncryptedKeys: make(map[string]EncryptedKey),
	}

	// Export all identity files
	if idProvider != nil {
		files, err := idProvider.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list identity files: %w", err)
		}

		for _, filename := range files {
			content, err := idProvider.GetContent(filename)
			if err != nil {
				return nil, fmt.Errorf("failed to get identity file %s: %w", filename, err)
			}
			data.IdentityFiles[filename] = content
		}
	}

	// Export encrypted keys (if keystore provided)
	if ks != nil {
		for provider := range cfg.Providers {
			exists, err := ks.KeyExists(provider)
			if err != nil {
				continue // Skip on error
			}
			if exists {
				// Export what we have in config (encrypted key stored there)
				if providerCfg, ok := cfg.Providers[provider]; ok && providerCfg.APIKeyEncrypted != "" {
					// Parse the encrypted key format (stored as JSON)
					key, err := parseEncryptedKey(providerCfg.APIKeyEncrypted)
					if err == nil {
						data.EncryptedKeys[provider] = *key
					}
				}
			}
		}
	}

	return data, nil
}

// KeyStoreExporter interface for keystore operations without direct import
type KeyStoreExporter interface {
	KeyExists(provider string) (bool, error)
}

// parseEncryptedKey parses the encrypted key format
func parseEncryptedKey(data string) (*EncryptedKey, error) {
	// Format: stored as JSON in the APIKeyEncrypted field
	var key EncryptedKey
	if err := json.Unmarshal([]byte(data), &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// ExportToJSON serializes export data to JSON
func ExportToJSON(data *ExportData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// ExportMinimal creates a minimal export with just config (no identity/keys)
func ExportMinimal(cfg *Config) (*ExportData, error) {
	return &ExportData{
		Version:       "1",
		ExportDate:    time.Now().UTC().Format(time.RFC3339),
		ExportVersion: ExportVersion,
		Config:        cfg,
		IdentityFiles: make(map[string]string),
		EncryptedKeys: make(map[string]EncryptedKey),
	}, nil
}
