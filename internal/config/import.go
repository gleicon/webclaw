//go:build js && wasm

package config

import (
	"encoding/json"
	"fmt"
)

// ImportFromJSON parses export data from JSON
func ImportFromJSON(data []byte) (*ExportData, error) {
	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse export JSON: %w", err)
	}
	return &export, nil
}

// ValidateExport checks if export data is valid and compatible
func ValidateExport(data *ExportData) error {
	if data.Version != "1" {
		return fmt.Errorf("unsupported export version: %s", data.Version)
	}

	if data.Config == nil {
		return fmt.Errorf("export missing config")
	}

	if data.Config.Version != CurrentVersion {
		return fmt.Errorf("config version mismatch: got %d, expected %d",
			data.Config.Version, CurrentVersion)
	}

	// Validate config
	if err := data.Config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Check for required identity files
	requiredFiles := []string{"IDENTITY.md", "SOUL.md"}
	for _, req := range requiredFiles {
		if _, ok := data.IdentityFiles[req]; !ok {
			return fmt.Errorf("export missing required identity file: %s", req)
		}
	}

	return nil
}

// IdentityFileImporter interface for storing identity files without direct import
type IdentityFileImporter interface {
	PutContent(filename string, content string) error
}

// ImportAll restores all data from export
func ImportAll(data *ExportData, storage *Storage, idImporter IdentityFileImporter) error {
	// Validate first
	if err := ValidateExport(data); err != nil {
		return err
	}

	// Import config
	if err := storage.SetConfig(data.Config); err != nil {
		return fmt.Errorf("failed to import config: %w", err)
	}

	// Import identity files
	if idImporter != nil {
		for filename, content := range data.IdentityFiles {
			if err := idImporter.PutContent(filename, content); err != nil {
				return fmt.Errorf("failed to import identity file %s: %w", filename, err)
			}
		}
	}

	return nil
}

// ImportConfigOnly imports only the config (not identity files)
func ImportConfigOnly(data *ExportData, storage *Storage) error {
	if data.Config == nil {
		return fmt.Errorf("export missing config")
	}

	if err := data.Config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return storage.SetConfig(data.Config)
}

// CanImport checks if the data can be imported (version compatible)
func CanImport(data *ExportData) (bool, string) {
	if data.Version != "1" {
		return false, fmt.Sprintf("unsupported version: %s", data.Version)
	}

	if data.Config == nil {
		return false, "missing config"
	}

	if data.Config.Version != CurrentVersion {
		return false, fmt.Sprintf("config version %d != %d",
			data.Config.Version, CurrentVersion)
	}

	return true, ""
}
