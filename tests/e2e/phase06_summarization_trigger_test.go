//go:build js && wasm

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/agent"
	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/identity"
	"github.com/gleicon/webclaw/internal/provider"
)

// anthropicAgentProvider wraps provider.AnthropicProvider to match agent.Provider interface
type anthropicAgentProvider struct {
	anthropic *provider.AnthropicProvider
	model     string
}

// Stream implements agent.Provider interface by adapting Anthropic's channel-based streaming
func (aap *anthropicAgentProvider) Stream(ctx context.Context, messages []agent.Message, tools []map[string]interface{}, callback func(provider.Token)) error {
	// Convert agent messages to provider messages
	provMsgs := make([]provider.Message, len(messages))
	for i, m := range messages {
		provMsgs[i] = provider.Message{Role: m.Role, Content: m.Content}
	}

	req := provider.CompletionRequest{
		Model:       aap.model,
		Messages:    provMsgs,
		Tools:       tools,
		MaxTokens:   1024,
		Temperature: 0.7,
		Stream:      true,
	}

	// Get token channel from Anthropic provider
	tokenChan := aap.anthropic.Stream(ctx, req)

	// Consume channel and call callback for each token
	for tok := range tokenChan {
		if tok.FinishReason == "error" {
			return provider.ErrServerError
		}
		callback(tok)
	}

	return nil
}

func (aap *anthropicAgentProvider) GetName() string  { return "anthropic" }
func (aap *anthropicAgentProvider) GetModel() string { return aap.model }

// TestPhase06_ConversationSummarizationTrigger tests the complete summarization flow
// with real LLM calls. This is Phase 06 of the WebClaw implementation.
//
// Requirements:
//   - ANTHROPIC_API_KEY environment variable set (from .env.test)
//   - WASM environment (browser or Node.js with wasm_exec.js)
//
// Test Flow:
//  1. Create ContextAssembler with real Anthropic summarizer
//  2. Add 18 messages (approaching 20 threshold)
//  3. Add 2 more messages to trigger summarization (20+ messages)
//  4. Verify CheckAndSummarize is called and returns true
//  5. Verify summarization completes (LLM call succeeds)
//  6. Verify conversation is compacted: summary + last 2 messages
func TestPhase06_ConversationSummarizationTrigger(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping Phase 06 test: ANTHROPIC_API_KEY not set in environment")
	}

	t.Log("=== Phase 06: Conversation Summarization Trigger Test ===")
	t.Logf("API Key present: %s...%s", apiKey[:10], apiKey[len(apiKey)-4:])

	// Step 1: Create ContextAssembler with real Anthropic summarizer
	t.Log("\n[Step 1] Creating ContextAssembler with Anthropic summarizer...")

	cfg := &config.Config{
		Identity: config.IdentityConfig{
			Name: "WebClaw",
		},
	}
	store := &identity.Store{}

	// Create assembler
	assembler := agent.NewContextAssembler(cfg, store)

	// Create real Anthropic provider for summarization
	anthropicProvider := provider.NewAnthropicProvider(apiKey)
	agentProvider := &anthropicAgentProvider{
		anthropic: anthropicProvider,
		model:     "claude-3-haiku-20240307", // Cheapest model for testing
	}
	summarizer := agent.NewSummarizer(agentProvider)
	assembler.SetSummarizer(summarizer)

	// Get conversation and verify default threshold
	conv := assembler.GetConversation()
	if conv.MaxMessages != 20 {
		t.Fatalf("Expected default MaxMessages=20, got %d", conv.MaxMessages)
	}

	t.Log("✓ ContextAssembler created with Anthropic summarizer")
	t.Logf("✓ Conversation threshold: %d messages", conv.MaxMessages)

	// Step 2: Add 18 messages (approaching threshold)
	t.Log("\n[Step 2] Adding 18 messages (approaching 20 threshold)...")

	for i := 0; i < 18; i++ {
		msg := conv.AddUserMessage(generateTestMessage(i))
		t.Logf("  Added message %d: %s (ID: %s)", i+1, truncate(msg.Content, 40), msg.ID)
	}

	msgCount := conv.GetMessageCount()
	if msgCount != 18 {
		t.Fatalf("Expected 18 messages, got %d", msgCount)
	}

	needsSum := conv.NeedsSummarization()
	if needsSum {
		t.Fatal("Should NOT need summarization at 18 messages (threshold is 20)")
	}

	t.Logf("✓ Added 18 messages, current count: %d", msgCount)
	t.Logf("✓ NeedsSummarization() = %v (expected: false)", needsSum)

	// Step 3: Add 2 more messages to trigger summarization (20+ messages)
	t.Log("\n[Step 3] Adding 2 more messages to trigger summarization...")

	for i := 18; i < 20; i++ {
		msg := conv.AddUserMessage(generateTestMessage(i))
		t.Logf("  Added message %d: %s (ID: %s)", i+1, truncate(msg.Content, 40), msg.ID)
	}

	msgCount = conv.GetMessageCount()
	if msgCount != 20 {
		t.Fatalf("Expected 20 messages, got %d", msgCount)
	}

	needsSum = conv.NeedsSummarization()
	if !needsSum {
		t.Fatal("Should need summarization at 20 messages (threshold is 20)")
	}

	t.Logf("✓ Added 2 more messages, current count: %d", msgCount)
	t.Logf("✓ NeedsSummarization() = %v (expected: true)", needsSum)

	// Step 4: Call CheckAndSummarize and verify it triggers
	t.Log("\n[Step 4] Calling CheckAndSummarize()...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	summary, triggered := assembler.CheckAndSummarize(ctx)
	duration := time.Since(startTime)

	if !triggered {
		t.Fatal("CheckAndSummarize did not trigger summarization")
	}

	if summary == nil {
		t.Fatal("CheckAndSummarize returned nil summary")
	}

	t.Logf("✓ CheckAndSummarize triggered: %v", triggered)
	t.Logf("✓ CheckAndSummarize duration: %v", duration)

	// Step 5: Verify summarization completed (LLM call succeeded)
	t.Log("\n[Step 5] Verifying summarization completed...")

	if summary.Content == "" {
		t.Fatal("Summary content is empty - LLM call may have failed")
	}

	if summary.MessageCount != 20 {
		t.Errorf("Expected summary.MessageCount=20, got %d", summary.MessageCount)
	}

	t.Logf("✓ Summary created with %d messages", summary.MessageCount)
	t.Logf("✓ Summary length: %d characters", len(summary.Content))
	t.Logf("✓ Summary preview: %s", truncate(summary.Content, 80))

	// Step 6: Verify conversation is compacted: summary + last 2 messages
	t.Log("\n[Step 6] Verifying conversation compaction...")

	// Get final conversation state
	finalConv := assembler.GetConversation()
	finalMsgCount := finalConv.GetMessageCount()

	// Should have exactly 2 messages (last 2 preserved for continuity)
	if finalMsgCount != 2 {
		t.Fatalf("Expected 2 messages after compaction (last 2 kept), got %d", finalMsgCount)
	}

	// Verify conversation has the summary set
	if finalConv.Summary == nil {
		t.Fatal("Conversation.Summary is nil after compaction")
	}

	if finalConv.Summary.Content == "" {
		t.Fatal("Conversation.Summary.Content is empty")
	}

	// Verify the 2 remaining messages exist
	messages := finalConv.GetMessages()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages in conversation, got %d", len(messages))
	}

	// Check that the messages have content and proper roles
	for i, msg := range messages {
		if msg.Content == "" {
			t.Errorf("Message %d has empty content", i)
		}
		if msg.Role != "user" && msg.Role != "assistant" {
			t.Errorf("Message %d has unexpected role: %s", i, msg.Role)
		}
		t.Logf("  ✓ Preserved message %d: role=%s, content=%s", i, msg.Role, truncate(msg.Content, 40))
	}

	t.Logf("✓ Conversation compacted successfully")
	t.Logf("✓ Final state: 1 summary + %d recent messages", finalMsgCount)
	t.Logf("✓ Summary preserved: %s...", truncate(finalConv.Summary.Content, 50))

	// Final verification
	t.Log("\n=== Phase 06 Test Results ===")
	t.Logf("Initial messages: 20")
	t.Logf("Threshold triggered: Yes (at 20 messages)")
	t.Logf("Summarization triggered: %v", triggered)
	t.Logf("LLM call succeeded: Yes (summary length: %d chars)", len(summary.Content))
	t.Logf("Conversation compacted: Yes (%d messages remaining)", finalMsgCount)
	t.Logf("Summary stored in conversation: Yes")
	t.Logf("Continuity preserved: Yes (last 2 messages kept)")
	t.Logf("Total test duration: %v", duration)

	t.Log("\n=== CONSOLE OUTPUT EXPECTED ===")
	t.Log("webclaw: summarization triggered - 20 messages")
	t.Log("webclaw: summarization complete - X tokens in Y s")
	t.Log("webclaw: conversation compacted - summary + 2 recent messages")

	t.Log("\n✅ PHASE 06 TEST PASSED")
}

// TestPhase06_ThresholdVariations tests summarization at different thresholds
func TestPhase06_ThresholdVariations(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	testCases := []struct {
		name          string
		threshold     int
		messagesToAdd int
		shouldTrigger bool
	}{
		{
			name:          "Exactly at threshold",
			threshold:     10,
			messagesToAdd: 10,
			shouldTrigger: true,
		},
		{
			name:          "One below threshold",
			threshold:     10,
			messagesToAdd: 9,
			shouldTrigger: false,
		},
		{
			name:          "One above threshold",
			threshold:     10,
			messagesToAdd: 11,
			shouldTrigger: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			store := &identity.Store{}
			assembler := agent.NewContextAssembler(cfg, store)

			anthropicProvider := provider.NewAnthropicProvider(apiKey)
			agentProvider := &anthropicAgentProvider{
				anthropic: anthropicProvider,
				model:     "claude-3-haiku-20240307",
			}
			summarizer := agent.NewSummarizer(agentProvider)
			assembler.SetSummarizer(summarizer)

			conv := assembler.GetConversation()
			conv.SetThresholds(tc.threshold, 0.75, 1000)

			// Add messages
			for i := 0; i < tc.messagesToAdd; i++ {
				conv.AddUserMessage(generateTestMessage(i))
			}

			needsSum := conv.NeedsSummarization()
			if needsSum != tc.shouldTrigger {
				t.Errorf("NeedsSummarization() = %v, expected %v", needsSum, tc.shouldTrigger)
			}

			t.Logf("Threshold=%d, Messages=%d, NeedsSummarization=%v (expected=%v)",
				tc.threshold, tc.messagesToAdd, needsSum, tc.shouldTrigger)
		})
	}
}

// TestPhase06_Summarization_EdgeCases tests edge cases for summarization
func TestPhase06_Summarization_EdgeCases(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	t.Run("Empty conversation should not trigger", func(t *testing.T) {
		cfg := &config.Config{}
		store := &identity.Store{}
		assembler := agent.NewContextAssembler(cfg, store)

		anthropicProvider := provider.NewAnthropicProvider(apiKey)
		agentProvider := &anthropicAgentProvider{
			anthropic: anthropicProvider,
			model:     "claude-3-haiku-20240307",
		}
		summarizer := agent.NewSummarizer(agentProvider)
		assembler.SetSummarizer(summarizer)

		ctx := context.Background()
		summary, triggered := assembler.CheckAndSummarize(ctx)

		if triggered {
			t.Error("Should not trigger summarization on empty conversation")
		}
		if summary != nil {
			t.Error("Should not return summary for empty conversation")
		}
	})

	t.Run("Below threshold should not trigger", func(t *testing.T) {
		cfg := &config.Config{}
		store := &identity.Store{}
		assembler := agent.NewContextAssembler(cfg, store)

		anthropicProvider := provider.NewAnthropicProvider(apiKey)
		agentProvider := &anthropicAgentProvider{
			anthropic: anthropicProvider,
			model:     "claude-3-haiku-20240307",
		}
		summarizer := agent.NewSummarizer(agentProvider)
		assembler.SetSummarizer(summarizer)

		conv := assembler.GetConversation()

		// Add 5 messages (well below 20 threshold)
		for i := 0; i < 5; i++ {
			conv.AddUserMessage("Test message " + string(rune('0'+i)))
		}

		ctx := context.Background()
		summary, triggered := assembler.CheckAndSummarize(ctx)

		if triggered {
			t.Error("Should not trigger summarization below threshold")
		}
		if summary != nil {
			t.Error("Should not return summary when not triggered")
		}

		// All messages should still be there
		if conv.GetMessageCount() != 5 {
			t.Errorf("Expected 5 messages preserved, got %d", conv.GetMessageCount())
		}
	})

	t.Run("Verify last 2 message continuity", func(t *testing.T) {
		cfg := &config.Config{}
		store := &identity.Store{}
		assembler := agent.NewContextAssembler(cfg, store)

		anthropicProvider := provider.NewAnthropicProvider(apiKey)
		agentProvider := &anthropicAgentProvider{
			anthropic: anthropicProvider,
			model:     "claude-3-haiku-20240307",
		}
		summarizer := agent.NewSummarizer(agentProvider)
		assembler.SetSummarizer(summarizer)

		conv := assembler.GetConversation()

		// Add messages with specific content we can verify
		for i := 0; i < 18; i++ {
			conv.AddUserMessage("Old message " + string(rune('0'+i%10)))
			conv.AddAssistantMessage("Old response " + string(rune('0'+i%10)))
		}
		// Add last 2 specific messages
		conv.AddUserMessage("FINAL USER QUESTION")
		conv.AddAssistantMessage("FINAL ASSISTANT ANSWER")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, triggered := assembler.CheckAndSummarize(ctx)
		if !triggered {
			t.Fatal("Summarization should have triggered")
		}

		// Verify the last 2 messages are preserved
		messages := conv.GetMessages()
		if len(messages) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(messages))
		}

		// Check content (order may vary based on implementation)
		foundUser := false
		foundAssistant := false
		for _, msg := range messages {
			if msg.Content == "FINAL USER QUESTION" && msg.Role == "user" {
				foundUser = true
			}
			if msg.Content == "FINAL ASSISTANT ANSWER" && msg.Role == "assistant" {
				foundAssistant = true
			}
		}

		if !foundUser {
			t.Error("Final user question not found in preserved messages")
		}
		if !foundAssistant {
			t.Error("Final assistant answer not found in preserved messages")
		}

		t.Log("✓ Last 2 message continuity verified")
	})
}

// TestPhase06_SummaryContent validates that summary contains key information
func TestPhase06_SummaryContent(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	cfg := &config.Config{}
	store := &identity.Store{}
	assembler := agent.NewContextAssembler(cfg, store)

	anthropicProvider := provider.NewAnthropicProvider(apiKey)
	agentProvider := &anthropicAgentProvider{
		anthropic: anthropicProvider,
		model:     "claude-3-haiku-20240307",
	}
	summarizer := agent.NewSummarizer(agentProvider)
	assembler.SetSummarizer(summarizer)

	conv := assembler.GetConversation()

	// Add messages with specific topics to verify in summary
	messages := []string{
		"My name is Alice and I work at TechCorp",
		"I need help with Go programming",
		"We discussed error handling patterns",
		"The best practice is to use context for cancellation",
		"I should refactor the codebase to use interfaces",
		"Performance optimization is important for our API",
		"We decided to use PostgreSQL for the database",
		"The deployment should use Docker containers",
		"Monitoring with Prometheus and Grafana",
		"Let's implement rate limiting for security",
		"The architecture should be microservices-based",
		"We need to add comprehensive test coverage",
		"Documentation should be in Markdown format",
		"API versioning strategy needs to be defined",
		"Caching with Redis will improve performance",
		"Authentication should use JWT tokens",
		"We agreed on a code review process",
		"The deadline is next Friday for the MVP",
		"Let's discuss scalability for high traffic",
		"Final decision: proceed with the current plan",
	}

	for _, msg := range messages {
		conv.AddUserMessage(msg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	summary, triggered := assembler.CheckAndSummarize(ctx)
	if !triggered {
		t.Fatal("Summarization should have triggered")
	}

	if summary == nil || summary.Content == "" {
		t.Fatal("Summary should not be empty")
	}

	// Verify summary contains key information
	content := strings.ToLower(summary.Content)

	keyTerms := []string{
		"alice", "techcorp", "go", "programming",
		"error handling", "interfaces", "performance",
		"postgresql", "docker", "monitoring",
		"security", "microservices", "test coverage",
		"api", "redis", "authentication", "jwt",
	}

	foundTerms := 0
	for _, term := range keyTerms {
		if strings.Contains(content, term) {
			foundTerms++
		}
	}

	t.Logf("Summary content: %s", truncate(summary.Content, 200))
	t.Logf("Key terms found in summary: %d/%d", foundTerms, len(keyTerms))

	// Summary should contain at least some key terms
	if foundTerms < 3 {
		t.Errorf("Summary should contain at least 3 key terms, found %d", foundTerms)
	}

	t.Log("✓ Summary content validation passed")
}

// generateTestMessage creates a unique test message for each index
func generateTestMessage(index int) string {
	topics := []string{
		"architecture", "performance", "security", "scalability",
		"deployment", "monitoring", "testing", "optimization",
		"refactoring", "documentation", "api design", "database",
	}

	topic := topics[index%len(topics)]
	return "Let's discuss the " + topic + " aspects of the system. Message number " + string(rune('0'+index%10))
}

// truncate truncates a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
