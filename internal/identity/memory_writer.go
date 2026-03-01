//go:build js && wasm

package identity

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

// MemoryFact represents a single fact stored in MEMORY.md
type MemoryFact struct {
	Content     string                 `json:"content"`
	Category    string                 `json:"category"`
	Confidence  float64                `json:"confidence"`
	Source      string                 `json:"source"`
	ExtractedAt time.Time              `json:"extracted_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryWriter handles writing extracted facts to MEMORY.md in IndexedDB
// Compatible with OpenClaw memory flush format
type MemoryWriter struct {
	store *Store
}

// NewMemoryWriter creates a new memory writer using the identity store
func NewMemoryWriter(store *Store) *MemoryWriter {
	return &MemoryWriter{
		store: store,
	}
}

// MemoryFlushResult contains information about the flush operation
type MemoryFlushResult struct {
	FactsWritten int
	Timestamp    time.Time
	Success      bool
	Error        error
}

// FlushFactsToMEMORY writes extracted facts to MEMORY.md file in IndexedDB
// Follows OpenClaw memory flush format with timestamp headers
func (mw *MemoryWriter) FlushFactsToMEMORY(facts []MemoryFact) (*MemoryFlushResult, error) {
	result := &MemoryFlushResult{
		Timestamp: time.Now(),
		Success:   false,
	}

	if len(facts) == 0 {
		result.Success = true
		return result, nil
	}

	// Get existing MEMORY.md content
	existingContent, err := mw.getExistingMEMORYContent()
	if err != nil {
		result.Error = fmt.Errorf("failed to read existing MEMORY.md: %w", err)
		return result, result.Error
	}

	// Format new facts section
	newSection := mw.formatFactsSection(facts)

	// Combine with existing content
	updatedContent := mw.mergeContent(existingContent, newSection)

	// Write back to MEMORY.md
	if err := mw.writeMEMORYContent(updatedContent); err != nil {
		result.Error = fmt.Errorf("failed to write MEMORY.md: %w", err)
		return result, result.Error
	}

	result.FactsWritten = len(facts)
	result.Success = true

	// Log to browser console
	js.Global().Get("console").Call("log",
		fmt.Sprintf("webclaw: flushed %d facts to MEMORY.md", len(facts)))

	return result, nil
}

// getExistingMEMORYContent retrieves current MEMORY.md content from IndexedDB
func (mw *MemoryWriter) getExistingMEMORYContent() (string, error) {
	file, err := mw.store.Get("MEMORY.md")
	if err != nil {
		return "", fmt.Errorf("failed to get MEMORY.md: %w", err)
	}

	if file == nil {
		// MEMORY.md doesn't exist yet, return empty string
		return "", nil
	}

	return file.Content, nil
}

// writeMEMORYContent writes content to MEMORY.md in IndexedDB
func (mw *MemoryWriter) writeMEMORYContent(content string) error {
	file := &IdentityFile{
		Filename: "MEMORY.md",
		Content:  content,
	}

	if err := mw.store.Put(file); err != nil {
		return fmt.Errorf("failed to write MEMORY.md: %w", err)
	}

	return nil
}

// formatFactsSection formats extracted facts for MEMORY.md
// Uses OpenClaw-compatible format with timestamp header
func (mw *MemoryWriter) formatFactsSection(facts []MemoryFact) string {
	var parts []string

	// Add timestamp header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	parts = append(parts, fmt.Sprintf("## Memory Flush - %s", timestamp))
	parts = append(parts, "")

	// Group facts by category
	byCategory := groupFactsByCategory(facts)

	// Write each category
	categories := []string{"user_preference", "decision", "fact", "action_item", "topic"}
	for _, category := range categories {
		if categoryFacts, ok := byCategory[category]; ok && len(categoryFacts) > 0 {
			parts = append(parts, fmt.Sprintf("### %s", formatCategoryName(category)))
			parts = append(parts, "")

			for _, fact := range categoryFacts {
				parts = append(parts, mw.formatFact(fact))
			}

			parts = append(parts, "")
		}
	}

	// Add any uncategorized facts
	if uncategorized, ok := byCategory[""]; ok && len(uncategorized) > 0 {
		parts = append(parts, "### Other")
		parts = append(parts, "")

		for _, fact := range uncategorized {
			parts = append(parts, mw.formatFact(fact))
		}

		parts = append(parts, "")
	}

	return strings.Join(parts, "\n")
}

// formatFact formats a single fact for MEMORY.md
func (mw *MemoryWriter) formatFact(fact MemoryFact) string {
	var parts []string

	// Main fact line with optional confidence indicator
	confidenceIndicator := ""
	if fact.Confidence < 0.7 {
		confidenceIndicator = " (low confidence)"
	} else if fact.Confidence >= 0.95 {
		confidenceIndicator = " (high confidence)"
	}

	parts = append(parts, fmt.Sprintf("- %s%s", fact.Content, confidenceIndicator))

	// Source attribution (optional, helps with traceability)
	if fact.Source != "" {
		parts = append(parts, fmt.Sprintf("  - Source: %s", fact.Source))
	}

	// Metadata if present
	if len(fact.Metadata) > 0 {
		metaJSON, _ := json.Marshal(fact.Metadata)
		if len(metaJSON) < 100 { // Only show if not too long
			parts = append(parts, fmt.Sprintf("  - Metadata: %s", string(metaJSON)))
		}
	}

	return strings.Join(parts, "\n")
}

// mergeContent combines existing MEMORY.md with new section
// Places new content at the top (most recent first)
func (mw *MemoryWriter) mergeContent(existing, newSection string) string {
	if existing == "" {
		// First time - create header and add content
		var parts []string
		parts = append(parts, "# MEMORY")
		parts = append(parts, "")
		parts = append(parts, "Persistent memory of important facts, preferences, and decisions from conversations.")
		parts = append(parts, "This file is automatically updated by the memory system.")
		parts = append(parts, "")
		parts = append(parts, "---")
		parts = append(parts, "")
		parts = append(parts, newSection)

		return strings.Join(parts, "\n")
	}

	// Find the position after the header to insert new content
	// Look for the first "## Memory Flush" or add at the end
	lines := strings.Split(existing, "\n")

	// Find where to insert (after the initial header/description)
	insertPos := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "# MEMORY") {
			insertPos = i + 1
		}
		// Skip past the description block
		if strings.HasPrefix(line, "---") && insertPos > 0 {
			insertPos = i + 1
			break
		}
	}

	// Insert new section
	var result []string
	result = append(result, lines[:insertPos]...)
	result = append(result, "")
	result = append(result, newSection)
	result = append(result, lines[insertPos:]...)

	return strings.Join(result, "\n")
}

// groupFactsByCategory organizes facts by their category
func groupFactsByCategory(facts []MemoryFact) map[string][]MemoryFact {
	result := make(map[string][]MemoryFact)

	for _, fact := range facts {
		category := fact.Category
		if category == "" {
			category = "other"
		}
		result[category] = append(result[category], fact)
	}

	return result
}

// formatCategoryName converts category ID to display name
func formatCategoryName(category string) string {
	switch category {
	case "user_preference":
		return "User Preferences"
	case "decision":
		return "Decisions"
	case "fact":
		return "Facts"
	case "action_item":
		return "Action Items"
	case "topic":
		return "Topics"
	default:
		return strings.Title(category)
	}
}

// ReadMEMORY reads the current content of MEMORY.md
// Useful for displaying memory in the UI
func (mw *MemoryWriter) ReadMEMORY() (string, error) {
	return mw.getExistingMEMORYContent()
}

// ClearMEMORY clears all content from MEMORY.md
// Use with caution - this deletes all persistent memory
func (mw *MemoryWriter) ClearMEMORY() error {
	return mw.writeMEMORYContent("")
}

// ExtractAndFlush is a convenience method that extracts facts and immediately flushes them
// This is the typical flow: extract from conversation → flush to MEMORY.md
func (mw *MemoryWriter) ExtractAndFlush(extractor func() ([]MemoryFact, error)) (*MemoryFlushResult, error) {
	facts, err := extractor()
	if err != nil {
		return &MemoryFlushResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     err,
		}, err
	}

	return mw.FlushFactsToMEMORY(facts)
}
