package agent

import (
	"testing"
)

func TestEstimateTokensAccuracy(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		minTokens int // Minimum reasonable estimate
		maxTokens int // Maximum reasonable estimate
	}{
		{
			name:      "empty",
			text:      "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "short sentence",
			text:      "Hello world",
			minTokens: 3,
			maxTokens: 10,
		},
		{
			name:      "medium paragraph",
			text:      "The quick brown fox jumps over the lazy dog. This is a test sentence.",
			minTokens: 10,
			maxTokens: 25,
		},
		{
			name:      "with newlines",
			text:      "Line 1\nLine 2\nLine 3",
			minTokens: 6,
			maxTokens: 15,
		},
		{
			name:      "code block",
			text:      "```go\nfmt.Println(\"hello\")\n```",
			minTokens: 8,
			maxTokens: 20,
		},
		{
			name:      "long word",
			text:      "supercalifragilisticexpialidocious",
			minTokens: 10,
			maxTokens: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.minTokens || got > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
					tt.text, got, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimateMessageTokensAccuracy(t *testing.T) {
	tests := []struct {
		role      string
		content   string
		minTokens int
	}{
		{"user", "Hello", 5},              // 4 overhead + ~1 for content
		{"system", "You are helpful", 10}, // 6 overhead + ~4 for content
		{"assistant", "Let me help", 10},  // 4 overhead + ~6 for content
	}

	for _, tt := range tests {
		got := EstimateMessageTokens(tt.role, tt.content)
		if got < tt.minTokens {
			t.Errorf("EstimateMessageTokens(%q, %q) = %d, want >= %d",
				tt.role, tt.content, got, tt.minTokens)
		}
	}
}

// Benchmark to ensure tokenizer is fast
func BenchmarkEstimateTokensSpeed(b *testing.B) {
	text := "The quick brown fox jumps over the lazy dog. " +
		"This sentence is repeated to make a longer text for benchmarking. " +
		"Tokenization should be fast enough for real-time use."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(text)
	}
}
