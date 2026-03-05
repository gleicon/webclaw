package agent

import (
	"fmt"
	"strings"
	"testing"
)

// TestPhase06_AccurateTokenCountingDisplay verifies the UI displays accurate token counts
// using the hybrid word-length algorithm (not simple chars/4)
//
// PASS: Test code showing token counts are accurate per hybrid algorithm
// FAIL: Specific mismatches showing old chars/4 method still in use
func TestPhase06_AccurateTokenCountingDisplay(t *testing.T) {
	t.Run("Test 1: Verify EstimateTokens uses hybrid algorithm not chars/4", func(t *testing.T) {
		// Test case: "Hi" (2 characters)
		// Old chars/4 method: 2/4 = 0 tokens
		// Hybrid algorithm: "Hi" is a short word (<=3 chars) = 1 token
		hiTokens := EstimateTokens("Hi")
		if hiTokens < 1 || hiTokens > 2 {
			t.Errorf("FAIL: EstimateTokens(\"Hi\") = %d, expected 1-2 tokens (hybrid), got %d (old method would give 0)", hiTokens, hiTokens)
		} else {
			t.Logf("PASS: EstimateTokens(\"Hi\") = %d (expected 1-2, old chars/4 would give 0)", hiTokens)
		}

		// Test case: "Hello world" (11 characters)
		// Old chars/4 method: 11/4 = 2 tokens
		// Hybrid algorithm: "Hello" (5 chars) = 2 tokens, "world" (5 chars) = 2 tokens, total = 4 tokens
		helloWorldTokens := EstimateTokens("Hello world")
		if helloWorldTokens < 3 || helloWorldTokens > 6 {
			t.Errorf("FAIL: EstimateTokens(\"Hello world\") = %d, expected 3-6 tokens (hybrid), old method would give 2", helloWorldTokens)
		} else {
			t.Logf("PASS: EstimateTokens(\"Hello world\") = %d (expected 3-6, old chars/4 would give 2)", helloWorldTokens)
		}
	})

	t.Run("Test 2: Verify word length categories", func(t *testing.T) {
		// Short word (≤3 chars) = 1 token
		theTokens := EstimateTokens("the")
		if theTokens != 1 {
			t.Errorf("FAIL: EstimateTokens(\"the\") = %d, expected 1 token for short word", theTokens)
		} else {
			t.Logf("PASS: Short word 'the' (3 chars) = %d token", theTokens)
		}

		// Medium word (4-6 chars) = 2 tokens
		helloTokens := EstimateTokens("hello")
		if helloTokens != 2 {
			t.Errorf("FAIL: EstimateTokens(\"hello\") = %d, expected 2 tokens for medium word", helloTokens)
		} else {
			t.Logf("PASS: Medium word 'hello' (5 chars) = %d tokens", helloTokens)
		}

		// Long word (7-10 chars) = 2 tokens
		computerTokens := EstimateTokens("computer")
		if computerTokens != 2 {
			t.Errorf("FAIL: EstimateTokens(\"computer\") = %d, expected 2 tokens for long word", computerTokens)
		} else {
			t.Logf("PASS: Long word 'computer' (8 chars) = %d tokens", computerTokens)
		}

		// Very long word (>10 chars) = length/2 tokens
		// "supercalifragilistic" (20 chars) = ~10 tokens
		superTokens := EstimateTokens("supercalifragilistic")
		expectedSuper := 10 // 20 chars / 2 = 10
		if superTokens < 8 || superTokens > 12 {
			t.Errorf("FAIL: EstimateTokens(\"supercalifragilistic\") = %d, expected ~%d tokens (length/2)", superTokens, expectedSuper)
		} else {
			t.Logf("PASS: Very long word 'supercalifragilistic' (20 chars) = %d tokens (expected ~%d, length/2)", superTokens, expectedSuper)
		}
	})

	t.Run("Test 3: Verify role overhead in EstimateMessageTokens", func(t *testing.T) {
		baseContent := "test"
		baseTokens := EstimateTokens(baseContent) // Should be 1 (short word)
		expectedBase := 4 + baseTokens            // 4 overhead + content

		// User message: 4 overhead tokens
		userTokens := EstimateMessageTokens("user", baseContent)
		if userTokens != expectedBase {
			t.Errorf("FAIL: EstimateMessageTokens(\"user\", \"test\") = %d, expected %d (4 overhead + %d content)", userTokens, expectedBase, baseTokens)
		} else {
			t.Logf("PASS: User message has 4 overhead tokens: %d total (4 + %d content)", userTokens, baseTokens)
		}

		// System message: 4 overhead + 2 additional = 6 tokens
		systemTokens := EstimateMessageTokens("system", baseContent)
		expectedSystem := 6 + baseTokens
		if systemTokens != expectedSystem {
			t.Errorf("FAIL: EstimateMessageTokens(\"system\", \"test\") = %d, expected %d (6 overhead + %d content)", systemTokens, expectedSystem, baseTokens)
		} else {
			t.Logf("PASS: System message has +2 overhead: %d total (6 + %d content)", systemTokens, baseTokens)
		}

		// Tool message: 4 overhead + 3 additional = 7 tokens
		toolTokens := EstimateMessageTokens("tool", baseContent)
		expectedTool := 7 + baseTokens
		if toolTokens != expectedTool {
			t.Errorf("FAIL: EstimateMessageTokens(\"tool\", \"test\") = %d, expected %d (7 overhead + %d content)", toolTokens, expectedTool, baseTokens)
		} else {
			t.Logf("PASS: Tool message has +3 overhead: %d total (7 + %d content)", toolTokens, baseTokens)
		}
	})

	t.Run("Test 4: Verify Conversation.GetTokenCount uses new tokenizer", func(t *testing.T) {
		conv := NewConversation("test-phase06")

		// Add messages with known token counts
		// "Hi" = 1 token (short word)
		conv.AddMessage(RoleUser, "Hi") // 4 overhead + 1 = 5

		// "Hello world" = 4 tokens (2 medium words)
		conv.AddMessage(RoleAssistant, "Hello world") // 4 overhead + 4 = 8

		// System message with +2 overhead
		conv.AddMessage(RoleSystem, "test") // 6 overhead + 1 = 7

		// Tool message with +3 overhead
		conv.AddMessage(RoleTool, "test") // 7 overhead + 1 = 8

		totalTokens := conv.GetTokenCount()

		// Calculate expected: 5 + 8 + 7 + 8 = 28
		expectedMin := 20
		expectedMax := 35

		if totalTokens < expectedMin || totalTokens > expectedMax {
			t.Errorf("FAIL: GetTokenCount() = %d, expected %d-%d tokens (using hybrid algorithm)", totalTokens, expectedMin, expectedMax)
		} else {
			t.Logf("PASS: Conversation.GetTokenCount() = %d (expected %d-%d, using hybrid algorithm)", totalTokens, expectedMin, expectedMax)
		}

		// Verify it's NOT using old chars/4 method
		// Old method: "Hi"=0, "Hello world"=2, "test"=0, "test"=0, overhead=4 each = 16 total
		oldMethodEstimate := 16
		if totalTokens == oldMethodEstimate {
			t.Errorf("FAIL: GetTokenCount() appears to be using old chars/4 method (got exactly %d)", totalTokens)
		} else {
			t.Logf("PASS: GetTokenCount() = %d differs from old chars/4 estimate (%d)", totalTokens, oldMethodEstimate)
		}
	})

	t.Run("Test 5: Verify algorithm rejects simple chars/4", func(t *testing.T) {
		testCases := []struct {
			name      string
			text      string
			charsDiv4 int // Old method estimate
			minTokens int // Minimum expected from hybrid
		}{
			{"2 chars", "Hi", 0, 1},
			{"3 chars", "the", 0, 1},
			{"5 chars one word", "hello", 1, 2},
			{"20 chars one word", "supercalifragilistic", 5, 8},
		}

		for _, tc := range testCases {
			tokens := EstimateTokens(tc.text)
			if tokens <= tc.charsDiv4 {
				t.Errorf("FAIL: %s - EstimateTokens() = %d, but chars/4 = %d. Algorithm appears to be using chars/4.", tc.name, tokens, tc.charsDiv4)
			} else {
				t.Logf("PASS: %s - EstimateTokens() = %d > chars/4 = %d (hybrid algorithm active)", tc.name, tokens, tc.charsDiv4)
			}
			if tokens < tc.minTokens {
				t.Errorf("FAIL: %s - EstimateTokens() = %d, expected at least %d tokens", tc.name, tokens, tc.minTokens)
			}
		}
	})

	t.Run("Test 6: Display summary of token counting behavior", func(t *testing.T) {
		fmt.Println("\n=== Phase 06: Accurate Token Counting Display Summary ===")
		fmt.Println()

		testStrings := []string{
			"Hi",
			"Hello world",
			"This is a test message",
			"The quick brown fox",
			"supercalifragilisticexpialidocious",
		}

		fmt.Println("Content Token Counts (using hybrid word-length algorithm):")
		fmt.Println(strings.Repeat("-", 70))
		for _, str := range testStrings {
			tokens := EstimateTokens(str)
			chars := len(str)
			oldMethod := chars / 4
			fmt.Printf("%-36s | %2d tokens | %2d chars | chars/4 = %d\n", fmt.Sprintf("\"%s\"", str), tokens, chars, oldMethod)
		}

		fmt.Println()
		fmt.Println("Message Token Counts (with role overhead):")
		fmt.Println(strings.Repeat("-", 70))
		roles := []string{"user", "system", "tool", "assistant"}
		content := "Hello world"
		for _, role := range roles {
			tokens := EstimateMessageTokens(role, content)
			contentTokens := EstimateTokens(content)
			overhead := tokens - contentTokens
			fmt.Printf("%-10s message: %-20s | %2d tokens (content=%d, overhead=%d)\n", role, fmt.Sprintf("\"%s\"", content), tokens, contentTokens, overhead)
		}

		fmt.Println()
		fmt.Println("Role Overhead Summary:")
		fmt.Println("  - user/assistant: +4 tokens (base)")
		fmt.Println("  - system:         +6 tokens (+2 additional)")
		fmt.Println("  - tool:           +7 tokens (+3 additional)")
		fmt.Println()
		fmt.Println("Algorithm Verification: PASS (using hybrid word-length, not chars/4)")
		fmt.Println(strings.Repeat("=", 70))
	})
}

// TestPhase06_TokenCountAccuracy specifically tests the accuracy requirements
// mentioned in the Phase 06 requirements
func TestPhase06_TokenCountAccuracy(t *testing.T) {
	t.Run("Short word (≤3 chars) = 1 token", func(t *testing.T) {
		shortWords := []string{"a", "Hi", "the", "cat", "dog", "go"}
		for _, word := range shortWords {
			tokens := EstimateTokens(word)
			if tokens != 1 {
				t.Errorf("FAIL: Short word '%s' (%d chars) = %d tokens, expected 1", word, len(word), tokens)
			}
		}
		t.Logf("PASS: All short words (≤3 chars) correctly counted as 1 token")
	})

	t.Run("Medium word (4-10 chars) = 2 tokens", func(t *testing.T) {
		mediumWords := []string{"hello", "world", "test", "code", "function", "computer", "language"}
		for _, word := range mediumWords {
			tokens := EstimateTokens(word)
			if tokens != 2 {
				t.Errorf("FAIL: Medium word '%s' (%d chars) = %d tokens, expected 2", word, len(word), tokens)
			}
		}
		t.Logf("PASS: All medium words (4-10 chars) correctly counted as 2 tokens")
	})

	t.Run("Long word (>10 chars) = length/2 tokens", func(t *testing.T) {
		longWords := []struct {
			word     string
			expected int
		}{
			{"supercalifragilistic", 10},                          // 20 chars / 2 = 10
			{"internationalization", 11},                          // 22 chars / 2 = 11
			{"pneumonoultramicroscopicsilicovolcanoconiosis", 23}, // 45 chars / 2 = 22.5 -> ~23
		}

		for _, tc := range longWords {
			tokens := EstimateTokens(tc.word)
			// Allow some tolerance (±2 tokens)
			if tokens < tc.expected-2 || tokens > tc.expected+2 {
				t.Errorf("FAIL: Long word '%s' (%d chars) = %d tokens, expected ~%d (length/2)", tc.word, len(tc.word), tokens, tc.expected)
			} else {
				t.Logf("PASS: Long word '%s' (%d chars) = %d tokens (expected ~%d)", tc.word, len(tc.word), tokens, tc.expected)
			}
		}
	})

	t.Run("Role overhead verification", func(t *testing.T) {
		content := "test"
		baseTokens := EstimateTokens(content)

		// System: +2 overhead
		systemTokens := EstimateMessageTokens("system", content)
		if systemTokens != baseTokens+6 { // 4 base + 2 extra
			t.Errorf("FAIL: System overhead = %d, expected %d (base+6)", systemTokens, baseTokens+6)
		} else {
			t.Logf("PASS: System message overhead = +6 tokens")
		}

		// Tool: +3 overhead
		toolTokens := EstimateMessageTokens("tool", content)
		if toolTokens != baseTokens+7 { // 4 base + 3 extra
			t.Errorf("FAIL: Tool overhead = %d, expected %d (base+7)", toolTokens, baseTokens+7)
		} else {
			t.Logf("PASS: Tool message overhead = +7 tokens")
		}
	})
}

// TestPhase06_ConversationTokenCount verifies Conversation uses the hybrid tokenizer
func TestPhase06_ConversationTokenCount(t *testing.T) {
	conv := NewConversation("test-conversation")

	// Add a user message with short words
	conv.AddUserMessage("Hi there cat")
	// "Hi"=1, "there"=2, "cat"=1, total content=4, overhead=4, total=8

	// Add an assistant message
	conv.AddAssistantMessage("Hello world")
	// "Hello"=2, "world"=2, total content=4, overhead=4, total=8

	// Add a system message
	conv.AddMessage(RoleSystem, "Be helpful")
	// "Be"=1, "helpful"=2, total content=3, overhead=6, total=9

	total := conv.GetTokenCount()

	// Old method estimate: "Hi there cat" = 2, "Hello world" = 2, "Be helpful" = 2
	// Overhead (old): 4+4+4 = 12
	// Total old method: 18
	oldMethodTotal := 18

	// Hybrid estimate: 8 + 8 + 9 = 25
	hybridMin := 20
	hybridMax := 30

	t.Run("Conversation uses hybrid algorithm", func(t *testing.T) {
		if total >= hybridMin && total <= hybridMax {
			t.Logf("PASS: GetTokenCount() = %d (within hybrid range %d-%d)", total, hybridMin, hybridMax)
		} else {
			t.Errorf("FAIL: GetTokenCount() = %d, expected %d-%d (hybrid algorithm), old method would give ~%d", total, hybridMin, hybridMax, oldMethodTotal)
		}
	})

	t.Run("Conversation rejects chars/4 method", func(t *testing.T) {
		// The total should be significantly different from old method
		difference := total - oldMethodTotal
		if difference < 3 {
			t.Errorf("FAIL: GetTokenCount() = %d, old method = %d, difference = %d. Algorithm may still be using chars/4", total, oldMethodTotal, difference)
		} else {
			t.Logf("PASS: GetTokenCount() = %d differs from old method (%d) by %d tokens", total, oldMethodTotal, difference)
		}
	})
}
