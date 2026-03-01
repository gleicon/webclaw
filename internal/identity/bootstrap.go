//go:build js && wasm

package identity

import (
	"fmt"
	"strings"

	"github.com/gleicon/webclaw/internal/config"
)

// BootstrapResult holds the assembled system prompt and metadata
type BootstrapResult struct {
	SystemPrompt   string
	FilesLoaded    []string
	TotalChars     int
	MaxChars       int
	TruncatedFiles []string
}

// AssembleSystemPrompt loads identity files and assembles the system prompt
// Respects bootstrap limits from configuration
func AssembleSystemPrompt(store *Store, cfg *config.Config) (*BootstrapResult, error) {
	result := &BootstrapResult{
		FilesLoaded:    []string{},
		TruncatedFiles: []string{},
		MaxChars:       cfg.Identity.BootstrapTotalMaxChars,
	}

	// Load all identity files
	files := []struct {
		filename string
		section  string
	}{
		{"IDENTITY.md", "IDENTITY"},
		{"SOUL.md", "SOUL"},
		{"USER.md", "USER CONTEXT"},
		{"AGENTS.md", "AGENT CONFIGURATION"},
		{"TOOLS.md", "AVAILABLE TOOLS"},
		{"HEARTBEAT.md", "HEARTBEAT INSTRUCTIONS"},
	}

	var sections []string
	totalChars := 0
	maxPerFile := cfg.Identity.BootstrapMaxChars
	maxTotal := cfg.Identity.BootstrapTotalMaxChars

	// Build header
	header := fmt.Sprintf("You are %s, an AI assistant.\n\n", cfg.Identity.Name)
	sections = append(sections, header)
	totalChars += len(header)

	for _, f := range files {
		file, err := store.Get(f.filename)
		if err != nil {
			// Log but don't fail - file might not exist yet
			continue
		}

		if file == nil {
			// File doesn't exist, use empty string
			continue
		}

		content := file.Content
		truncated := false

		// Check per-file limit
		if len(content) > maxPerFile {
			content = content[:maxPerFile]
			lastNewline := strings.LastIndex(content, "\n")
			if lastNewline > 0 {
				content = content[:lastNewline] // Cut at newline
			}
			if content == "" {
				content = file.Content[:maxPerFile] // Fallback: hard cut
			}
			truncated = true
			result.TruncatedFiles = append(result.TruncatedFiles, f.filename)
		}

		// Check total limit
		section := fmt.Sprintf("[%s]\n%s\n\n", f.section, content)
		if totalChars+len(section) > maxTotal {
			// Can't fit this file, mark remaining as truncated
			for _, remaining := range files[len(result.FilesLoaded):] {
				result.TruncatedFiles = append(result.TruncatedFiles, remaining.filename)
			}
			break
		}

		sections = append(sections, section)
		totalChars += len(section)
		result.FilesLoaded = append(result.FilesLoaded, f.filename)

		if truncated {
			result.TruncatedFiles = append(result.TruncatedFiles, f.filename)
		}
	}

	// Add footer with bootstrap info
	footer := fmt.Sprintf("Bootstrap limits: %d/%d characters loaded from %d files",
		totalChars, maxTotal, len(result.FilesLoaded))
	if len(result.TruncatedFiles) > 0 {
		footer += fmt.Sprintf(" (%d files truncated)", len(result.TruncatedFiles))
	}
	sections = append(sections, footer)

	result.SystemPrompt = strings.Join(sections, "")
	result.TotalChars = totalChars

	return result, nil
}

// LoadIdentityFiles loads all identity files without assembly (for editing)
func LoadIdentityFiles(store *Store) (map[string]*IdentityFile, error) {
	files := make(map[string]*IdentityFile)
	filenames := []string{
		"IDENTITY.md",
		"SOUL.md",
		"USER.md",
		"AGENTS.md",
		"TOOLS.md",
		"HEARTBEAT.md",
	}

	for _, name := range filenames {
		file, err := store.Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", name, err)
		}
		files[name] = file
	}

	return files, nil
}

// CalculateBootstrapStats calculates stats for bootstrap display
func CalculateBootstrapStats(files map[string]*IdentityFile, cfg *config.Config) BootstrapStats {
	stats := BootstrapStats{
		Files:      make(map[string]FileStat),
		MaxPerFile: cfg.Identity.BootstrapMaxChars,
		MaxTotal:   cfg.Identity.BootstrapTotalMaxChars,
	}

	total := 0
	for name, file := range files {
		if file != nil {
			stat := FileStat{
				Size:     file.Size,
				WillLoad: file.Size <= cfg.Identity.BootstrapMaxChars,
			}
			if !stat.WillLoad {
				stat.TruncatedTo = cfg.Identity.BootstrapMaxChars
			}
			stats.Files[name] = stat
			if stat.WillLoad {
				total += file.Size
			} else {
				total += cfg.Identity.BootstrapMaxChars
			}
		}
	}

	stats.TotalSize = total
	stats.WillFit = total <= cfg.Identity.BootstrapTotalMaxChars

	return stats
}

// BootstrapStats holds statistics for bootstrap display
type BootstrapStats struct {
	Files      map[string]FileStat
	TotalSize  int
	MaxPerFile int
	MaxTotal   int
	WillFit    bool
}

// FileStat holds stats for a single file
type FileStat struct {
	Size        int
	WillLoad    bool
	TruncatedTo int
}

// GetSystemPromptTemplate returns the base template for system prompts
// (Used for documentation/testing)
func GetSystemPromptTemplate() string {
	return `You are {{name}}, an AI assistant.

[IDENTITY]
{{IDENTITY.md}}

[SOUL]
{{SOUL.md}}

[USER CONTEXT]
{{USER.md}}

[AGENT CONFIGURATION]
{{AGENTS.md}}

[AVAILABLE TOOLS]
{{TOOLS.md}}

[HEARTBEAT INSTRUCTIONS]
{{HEARTBEAT.md}}

Bootstrap limits: {{total}}/{{max}} characters
`
}
