package agent

import (
	"strings"
	"unicode/utf8"
)

// EstimateTokens provides a more accurate token estimate than simple chars/4
// Uses a hybrid approach considering word length, punctuation, and formatting
//
// Rough guidelines:
// - Short words (1-4 chars): ~1 token
// - Medium words (5-8 chars): ~1-2 tokens
// - Long words (9+ chars): ~2-3 tokens
// - Code/formatting adds overhead
// - This is still an estimate; actual tokenization varies by model
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count runes (actual Unicode characters, not bytes)
	totalRunes := utf8.RuneCountInString(text)

	// Base estimate: ~0.75 tokens per word on average
	words := strings.Fields(text)
	tokenCount := 0

	for _, word := range words {
		length := utf8.RuneCountInString(word)

		// Token estimation based on word length
		// Rough approximation of BPE-style tokenization
		switch {
		case length <= 3:
			// Short words (the, and, of): usually 1 token
			tokenCount += 1
		case length <= 6:
			// Medium words (hello, world): 1-2 tokens
			tokenCount += 2
		case length <= 10:
			// Longer words (information): 2 tokens
			tokenCount += 2
		default:
			// Very long words (supercalifragilistic): split into subwords
			// ~2 characters per subword token on average
			tokenCount += (length / 2)
		}
	}

	// Add overhead for formatting
	// Each newline is roughly a token
	tokenCount += strings.Count(text, "\n")

	// Code blocks add overhead (``` + language + newline)
	codeBlockCount := strings.Count(text, "```") / 2
	tokenCount += codeBlockCount * 3

	// Special characters often get their own tokens
	tokenCount += strings.Count(text, "{") + strings.Count(text, "}")
	tokenCount += strings.Count(text, "[") + strings.Count(text, "]")

	// URL/links are token-heavy
	if strings.Contains(text, "http://") || strings.Contains(text, "https://") {
		// URLs typically ~10-20 tokens depending on length
		tokenCount += 10
	}

	// Sanity bounds
	if tokenCount < totalRunes/6 {
		// Lower bound: rarely less than 1 token per 6 characters
		tokenCount = totalRunes / 6
	}
	if tokenCount > totalRunes {
		// Upper bound: rarely more than 1 token per character
		tokenCount = totalRunes
	}

	return tokenCount
}

// EstimateMessageTokens estimates tokens for a single message including role overhead
func EstimateMessageTokens(role, content string) int {
	// Role overhead: ~4 tokens for role label and formatting
	overhead := 4

	// Special handling for different message types
	switch role {
	case "system":
		// System messages often have extra formatting
		overhead += 2
	case "tool":
		// Tool results have overhead for result formatting
		overhead += 3
	}

	return overhead + EstimateTokens(content)
}

// ValidateEstimate compares our estimate to actual (if available)
// Returns ratio of estimate to actual (1.0 = perfect match)
func ValidateEstimate(estimated, actual int) float64 {
	if actual == 0 {
		return 1.0
	}
	return float64(estimated) / float64(actual)
}
