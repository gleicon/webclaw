# Phase 6: Real Agent Loop - Implementation Plan

**Phase ID:** 06-real-agent-loop  
**Goal:** Make WebClaw a real OpenClaw implementation with working tool_use loop, real summarization, and complete memory flush

**Requirements Addressed:** AGNT-01, AGNT-02, AGNT-03, AGNT-04, MEM-04, PROV-03 (full)

---

## Executive Summary

WebClaw currently has 80% of an OpenClaw implementation. The infrastructure is solid, but the **critical missing piece** is the actual tool_use protocol:

1. **Missing:** Tool definitions sent to LLM (providers don't include `tools` in API requests)
2. **Missing:** Tool_use response parsing (providers don't extract `tool_use` blocks from Anthropic or `tool_calls` from OpenAI)
3. **Missing:** Real summarization wiring (placeholder exists, not connected)
4. **Missing:** Memory flush to MEMORY.md before summarization

This phase completes the agent loop to match OpenClaw's behavior.

---

## Phase Breakdown

### Phase 6-1: Provider Tool Support (AGNT-01 Critical)
**Goal:** Providers send tool definitions and parse tool_use responses

**Files to Modify:**
- `internal/provider/provider.go` - Add Tools field to CompletionRequest
- `internal/provider/anthropic.go` - Include tools in request, parse content_block_start/tool_use
- `internal/provider/openai.go` - Include tools in request, parse tool_calls
- `internal/provider/openrouter.go` - Pass through tools

**Key Changes:**

1. **CompletionRequest adds Tools field:**
```go
type CompletionRequest struct {
    Model       string
    Messages    []Message
    Tools       []map[string]interface{}  // NEW: Tool definitions for LLM
    MaxTokens   int
    Temperature float64
    Stream      bool
}
```

2. **Anthropic provider changes:**
   - Request: Add `tools` array with `{name, description, input_schema}`
   - Response parsing: Handle `content_block_start` with `type: "tool_use"`
   - Extract: `id`, `name`, `input` from tool_use block
   - Return Token with `FinishReason: "tool_use"`, populated `ToolName`, `ToolInput`, `ToolUseID`

3. **OpenAI provider changes:**
   - Request: Add `tools` array with `{type: "function", function: {name, description, parameters}}`
   - Response parsing: Handle `finish_reason: "tool_calls"`
   - Extract: `tool_calls[].id`, `tool_calls[].function.name`, `tool_calls[].function.arguments`
   - Return Token with `FinishReason: "tool_use"` (mapped from "tool_calls")

**Verification:**
- Send message "search for Go documentation"
- LLM should return `tool_use` requesting `web_search` tool
- Tool executes and result feeds back into conversation

---

### Phase 6-2: Agent Loop Wiring (AGNT-01 Completion)
**Goal:** Wire tool registry to provider calls

**Files to Modify:**
- `internal/agent/loop.go` - Pass tool schemas to provider
- `internal/agent/provider_adapter.go` (if exists) or create in loop.go

**Key Changes:**

1. **AgentLoop.Run() passes tools to provider:**
```go
// Before calling provider, get tool schemas from registry
var toolSchemas []map[string]interface{}
if al.toolRegistry != nil {
    toolSchemas = al.toolRegistry.ToAPISchema()
}

// Pass in CompletionRequest
req := provider.CompletionRequest{
    Model:     al.model,
    Messages:  requestMessages,
    Tools:     toolSchemas,  // NEW
    MaxTokens: 4096,
    Stream:    true,
}
```

2. **providerAdapter.Stream() forwards tools:**
```go
func (pa *providerAdapter) Stream(ctx context.Context, messages []Message, tools []map[string]interface{}, callback func(tok provider.Token)) error {
    provMsgs := convertMessages(messages)
    req := provider.CompletionRequest{
        Model:    pa.model,
        Messages: provMsgs,
        Tools:    tools,  // Forward to actual provider
        // ...
    }
    // ...
}
```

**Verification:**
- Start conversation
- Send: "What tools do you have available?"
- Agent should know about registered tools (web_fetch, web_search, etc.)

---

### Phase 6-3: Real Summarization (AGNT-02, AGNT-03)
**Goal:** Wire summarizer to actually call LLM when threshold hit

**Files to Modify:**
- `internal/agent/context.go` - Replace placeholder CheckAndSummarize()
- `internal/agent/loop.go` - Call summarization before adding assistant response

**Current State:**
```go
// context.go - Currently returns placeholder
func (ca *ContextAssembler) CheckAndSummarize() (*Summary, bool) {
    if ca.conversation.NeedsSummarization() {
        // PLACEHOLDER - doesn't actually call LLM
        summary := &Summary{
            Content: fmt.Sprintf("Placeholder summary: %d messages", msgCount),
        }
        return summary, true
    }
    return nil, false
}
```

**Required Changes:**

1. **ContextAssembler gets Summarizer:**
```go
type ContextAssembler struct {
    config        *config.Config
    identityStore *identity.Store
    conversation  *Conversation
    summarizer    *Summarizer  // NEW
}
```

2. **Real summarization implementation:**
```go
func (ca *ContextAssembler) CheckAndSummarize(ctx context.Context) (*Summary, bool) {
    if !ca.conversation.NeedsSummarization() {
        return nil, false
    }
    
    // Call real summarizer with LLM
    result, err := ca.summarizer.SummarizeConversation(ctx, ca.conversation)
    if err != nil {
        // Log error, don't block
        jsLog("Summarization failed:", err)
        return nil, false
    }
    
    // Create summary from result
    summary := &Summary{
        ID:           generateMessageID(),
        Content:      result.Summary,
        MessageCount: result.MessageCount,
        CreatedAt:    time.Now(),
    }
    
    ca.conversation.SetSummary(summary)
    ca.conversation.ClearMessages()
    
    return summary, true
}
```

3. **Loop calls summarization:**
```go
// In AgentLoop.Run(), before adding assistant response:
if ca.assembler != nil {
    // Check and trigger summarization if needed
    if summary, triggered := ca.assembler.CheckAndSummarize(ctx); triggered {
        jsLog("Summarization triggered:", summary.MessageCount, "messages summarized")
    }
    
    ca.assembler.AddAssistantResponse(responseContent)
}
```

**Verification:**
- Have 20-message conversation (or adjust threshold to 5 for testing)
- Send 21st message
- Should trigger summarization (visible in console)
- Conversation history should be replaced with summary

---

### Phase 6-4: Memory Flush Before Summarization (MEM-04)
**Goal:** Flush durable knowledge to MEMORY.md before summarizing

**Files to Modify:**
- `internal/memory/flush.go` (exists, verify implementation)
- `internal/agent/context.go` - Call memory flush before summarizing
- `internal/identity/memory_writer.go` - Verify MEMORY.md writing

**Implementation:**

1. **Before summarization, extract facts and store to memory:**
```go
func (ca *ContextAssembler) CheckAndSummarize(ctx context.Context) (*Summary, bool) {
    if !ca.conversation.NeedsSummarization() {
        return nil, false
    }
    
    // PHASE 6-4: Flush important facts to memory before losing them
    if ca.memoryStore != nil {
        facts, err := ca.summarizer.ExtractKeyFacts(ctx, ca.conversation.GetMessages())
        if err == nil && len(facts) > 0 {
            for _, fact := range facts {
                doc := memory.NewMemoryDocument(
                    generateMemoryID(),
                    fact,
                    nil, // No embedding for now
                )
                doc.Metadata = map[string]interface{}{
                    "type":   "conversation_fact",
                    "source": "pre_summarization_flush",
                }
                ca.memoryStore.Store(doc)
            }
            
            // Also write to MEMORY.md identity file
            if ca.identityStore != nil {
                memoryContent := strings.Join(facts, "\n\n")
                ca.identityStore.AppendToMemoryFile(memoryContent)
            }
        }
    }
    
    // Then proceed with summarization...
}
```

**Verification:**
- Have conversation with factual information ("My name is Alice")
- Trigger summarization
- Check MEMORY.md contains the fact
- Check memory store contains the fact

---

### Phase 6-5: Token Counting & Context Window (AGNT-02)
**Goal:** Accurate token counting instead of char/4 heuristic

**Files to Modify:**
- `internal/agent/conversation.go` - Replace estimateTokens()
- `internal/agent/tokenizer.go` - Create new file with tiktoken-style counting

**Implementation:**

1. **Simple tokenizer (no external deps):**
```go
// tokenizer.go
package agent

import (
    "strings"
    "unicode/utf8"
)

// EstimateTokens provides a more accurate token estimate than chars/4
// Uses a hybrid approach: words + punctuation + unicode handling
func EstimateTokens(text string) int {
    if text == "" {
        return 0
    }
    
    // Count words (roughly 1-2 tokens per word depending on length)
    words := strings.Fields(text)
    tokenCount := 0
    
    for _, word := range words {
        // Short words (1-4 chars): ~1 token
        // Medium words (5-8 chars): ~1.5 tokens
        // Long words (9+ chars): ~2 tokens
        // Very long words: split and estimate
        length := utf8.RuneCountInString(word)
        switch {
        case length <= 4:
            tokenCount += 1
        case length <= 8:
            tokenCount += 2
        case length <= 12:
            tokenCount += 3
        default:
            tokenCount += (length / 4) + 1
        }
    }
    
    // Add overhead for formatting
    tokenCount += strings.Count(text, "\n") // Newlines
    tokenCount += strings.Count(text, "```") / 2 // Code blocks
    
    return tokenCount
}
```

2. **Update Conversation to use accurate counting:**
```go
func (c *Conversation) GetTokenCount() int {
    total := 0
    for _, msg := range c.Messages {
        // Add role overhead (~4 tokens per message for role + formatting)
        total += 4
        total += EstimateTokens(msg.Content)
    }
    return total
}
```

**Verification:**
- Compare token estimates before/after
- Test with known text samples
- Verify threshold triggers at correct time

---

### Phase 6-6: Integration & E2E Testing
**Goal:** Wire everything together and test full flow

**Files to Modify:**
- `cmd/webclaw/main.go` - Wire summarizer to context assembler
- `static/worker.js` - Ensure tool events display correctly
- `tests/e2e/tool_loop_test.go` - Create comprehensive E2E test

**Integration Steps:**

1. **main.go wiring:**
```go
// After creating AgentLoop and router
summarizer := agent.NewSummarizer(routerProvider)
contextAssembler.SetSummarizer(summarizer)
contextAssembler.SetMemoryStore(memoryStore) // If available
```

2. **E2E Test:**
```go
func TestToolUseLoop(t *testing.T) {
    // Setup: Create agent loop with real provider (mock for test)
    // Send: "Search for Go programming language"
    // Expect:
    //   1. LLM returns tool_use for web_search
    //   2. Tool executes
    //   3. Result injected
    //   4. LLM returns final response
    //   5. No more tool_use
}
```

**Verification:**
- Full conversation with tool use works end-to-end
- Summarization triggers correctly
- Memory flush happens before summarization
- UI shows tool execution status

---

## Technical Approach

### Provider Tool Protocols

**Anthropic Messages API:**
```json
// Request
{
  "model": "claude-sonnet-4-5",
  "messages": [...],
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web for information",
      "input_schema": {
        "type": "object",
        "properties": {
          "query": {"type": "string"}
        },
        "required": ["query"]
      }
    }
  ]
}

// Response (streaming)
event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_123","name":"web_search","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query\": \"Go programming"}}}

event: message_stop
data: {"type":"message_stop"}
```

**OpenAI Chat Completions API:**
```json
// Request
{
  "model": "gpt-4o",
  "messages": [...],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "web_search",
        "description": "Search the web for information",
        "parameters": {
          "type": "object",
          "properties": {
            "query": {"type": "string"}
          },
          "required": ["query"]
        }
      }
    }
  ]
}

// Response (streaming)
data: {"id":"chatcmpl-123","choices":[{"delta":{"content":null,"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"web_search","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-123","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\": \"Go programming\"}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-123","choices":[{"delta":{},"finish_reason":"tool_calls"}]}
```

### Summarization Flow

```
1. User sends message
2. AgentLoop.Run() starts
3. Check token count / message count
4. If threshold exceeded:
   a. Extract key facts → Store to memory
   b. Call LLM to summarize conversation
   c. Replace history with summary + last 2 messages
5. Send request to LLM (with tools)
6. Receive response (may include tool_use)
7. If tool_use:
   a. Emit "running" event
   b. Dispatch tool
   c. Emit "done" event
   d. Inject tool_result
   e. Loop back to step 5 (up to max iterations)
8. If no tool_use (stop/finish):
   a. Stream response to UI
   b. Add to conversation history
   c. Emit complete
```

---

## Success Criteria Per Phase

| Phase | Success Criteria | Verification |
|-------|-----------------|--------------|
| 6-1 | Provider sends tool definitions, parses tool_use | Test: "Use web_search tool" → triggers web_search |
| 6-2 | Agent loop passes tools to provider | Test: Agent knows available tools |
| 6-3 | Real summarization with LLM call | Test: 21 messages → triggers summarization |
| 6-4 | Memory flush before summarization | Test: Facts appear in MEMORY.md |
| 6-5 | Accurate token counting | Test: Compare estimates to actual |
| 6-6 | Full E2E tool loop works | Test: 3-turn conversation with tool use |

---

## Key Files to Modify

| File | Changes |
|------|---------|
| `internal/provider/provider.go` | Add Tools field to CompletionRequest |
| `internal/provider/anthropic.go` | Include tools in request, parse tool_use blocks |
| `internal/provider/openai.go` | Include tools in request, parse tool_calls |
| `internal/provider/openrouter.go` | Pass through tools |
| `internal/agent/loop.go` | Pass tools to provider, wire summarizer |
| `internal/agent/context.go` | Real summarization, memory flush |
| `internal/agent/tokenizer.go` | Create new file for accurate token counting |
| `cmd/webclaw/main.go` | Wire summarizer to context assembler |
| `tests/e2e/tool_loop_test.go` | Create comprehensive E2E tests |

---

## Estimated Effort

| Phase | Files | Tasks | Est. Time |
|-------|-------|-------|-----------|
| 6-1 | 4 | Add Tools field, implement Anthropic tool_use, implement OpenAI tool_calls | 2-3 hrs |
| 6-2 | 1 | Wire tool registry to provider calls | 1 hr |
| 6-3 | 2 | Wire summarizer, replace placeholder | 1-2 hrs |
| 6-4 | 2 | Implement memory flush | 1 hr |
| 6-5 | 2 | Create tokenizer, update estimates | 1 hr |
| 6-6 | 3 | Integration, E2E tests, debugging | 2-3 hrs |
| **Total** | **14** | **~20 tasks** | **8-11 hrs** |

---

## Dependencies

**Must complete first:**
- Phase 5 (Live AI Connection) - COMPLETE ✓

**This phase enables:**
- Phase 7 (Polish & Release) - Full OpenClaw compatibility

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Provider API changes | Abstract tool handling, keep provider-specific code isolated |
| Token counting inaccuracy | Make it configurable, document it's approximate |
| Summarization failures | Fail gracefully, continue without summary |
| Tool loop infinite loops | Hard limit at 10 iterations, timeout on each call |

---

## OpenClaw Compatibility Notes

This phase makes WebClaw compatible with OpenClaw's core agent behavior:

1. **Tool Use Protocol** - Matches OpenClaw's tool_use → execute → tool_result loop
2. **Context Management** - Matches 20 message / 75% threshold pattern
3. **Summarization** - Matches LLM-based summarization with history replacement
4. **Memory Flush** - Matches MEMORY.md persistence before compaction

After this phase, WebClaw will be a true browser-based OpenClaw implementation.
