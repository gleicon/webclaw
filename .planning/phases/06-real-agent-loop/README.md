# Phase 6: Real Agent Loop

**Phase ID:** 06-real-agent-loop  
**Goal:** Make WebClaw a real OpenClaw implementation with working tool_use loop, real summarization, and complete memory flush

## Overview

WebClaw currently has 80% of an OpenClaw implementation. The infrastructure is impressive:
- ✅ Agent loop with tool dispatch
- ✅ Providers with SSE streaming
- ✅ Tool registry
- ✅ Context management framework
- ✅ Memory system with hybrid search
- ✅ Summarizer implementation

**The critical missing piece (15%):** The tool_use protocol isn't complete:
- ❌ Providers don't send tool definitions to LLM
- ❌ Providers don't parse tool_use from responses
- ❌ Summarization is a placeholder (doesn't call LLM)
- ❌ Memory flush before summarization not implemented

This phase completes the remaining 15% to make WebClaw a true OpenClaw implementation.

## Plans

| Plan | Objective | Files | Wave | Requirements |
|------|-----------|-------|------|--------------|
| 06-01 | Provider tool support: send tool definitions, parse tool_use | 4 files | 1 | AGNT-01 |
| 06-02 | Wire tool registry to provider calls | 1 file | 2 | AGNT-01 |
| 06-03 | Real summarization with LLM calls | 2 files | 3 | AGNT-02, AGNT-03 |
| 06-04 | Memory flush before summarization | 3 files | 4 | MEM-04 |
| 06-05 | Accurate token counting + E2E tests | 3 files | 5 | AGNT-02, AGNT-03, AGNT-04 |

**Total:** 5 plans, 13 files, ~8-11 hours estimated

## Wave Structure

```
Wave 1 (06-01): Provider tool support
    ↓
Wave 2 (06-02): Agent loop wiring
    ↓
Wave 3 (06-03): Real summarization
    ↓
Wave 4 (06-04): Memory flush
    ↓
Wave 5 (06-05): Token counting + E2E tests
```

## Key Files Modified

| File | Changes | Plans |
|------|---------|-------|
| internal/provider/provider.go | Add Tools field to CompletionRequest | 06-01 |
| internal/provider/anthropic.go | Include tools in request, parse tool_use | 06-01 |
| internal/provider/openai.go | Include tools in request, parse tool_calls | 06-01 |
| internal/provider/openrouter.go | Pass through tools | 06-01 |
| internal/agent/loop.go | Pass tools to provider, wire summarizer | 06-02, 06-03 |
| internal/agent/context.go | Real summarization, memory flush | 06-03, 06-04 |
| internal/agent/tokenizer.go | Create new file for accurate counting | 06-05 |
| internal/agent/conversation.go | Use new tokenizer | 06-05 |
| internal/memory/flush.go | Verify/implement flush | 06-04 |
| internal/identity/memory_writer.go | AppendToMemoryFile | 06-04 |
| cmd/webclaw/main.go | Wire all components | 06-03, 06-04, 06-05 |
| tests/e2e/full_agent_loop_test.go | E2E test | 06-05 |

## Success Criteria

### Phase 6-1 (Provider Tool Support)
- [ ] CompletionRequest has Tools field
- [ ] Anthropic sends tools in API request
- [ ] Anthropic parses tool_use from streaming response
- [ ] OpenAI sends tools in API request
- [ ] OpenAI parses tool_calls from streaming response
- [ ] Token has ToolName, ToolInput, ToolUseID when tool_use detected

### Phase 6-2 (Agent Loop Wiring)
- [ ] Provider interface accepts tools parameter
- [ ] AgentLoop.Run() gets tools from registry
- [ ] Tools flow: registry → loop → provider → LLM

### Phase 6-3 (Real Summarization)
- [ ] CheckAndSummarize calls real LLM summarizer
- [ ] Summarization triggers at correct thresholds
- [ ] After summarization: summary + last 2 messages kept

### Phase 6-4 (Memory Flush)
- [ ] Facts extracted before summarization
- [ ] Facts stored to memory store
- [ ] Facts written to MEMORY.md
- [ ] No data loss when conversation summarized

### Phase 6-5 (Integration)
- [ ] Accurate token counting (better than chars/4)
- [ ] Full E2E agent loop test passes
- [ ] All components wired correctly
- [ ] Console logging shows tool flow

## Technical Details

### Tool Use Protocols

**Anthropic Messages API:**
```json
// Request
{
  "model": "claude-sonnet-4-5",
  "messages": [...],
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web",
      "input_schema": {"type": "object", "properties": {...}}
    }
  ]
}

// Response (streaming)
event: content_block_start
data: {"type":"content_block_start","content_block":{"type":"tool_use","id":"tool_123","name":"web_search"}}

event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"query\":\"...\"}"}}
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
        "description": "Search the web",
        "parameters": {"type": "object", "properties": {...}}
      }
    }
  ]
}

// Response (streaming)
data: {"choices":[{"delta":{"tool_calls":[{"function":{"name":"web_search"}}]}}]}
data: {"choices":[{"finish_reason":"tool_calls"}]}
```

### Summarization Flow

```
1. User sends message
2. Check summarization thresholds
3. If exceeded:
   a. Extract key facts → Store to memory
   b. Call LLM to summarize conversation
   c. Replace history: summary + last 2 messages
4. Send request to LLM (with tools)
5. Receive response (may include tool_use)
6. If tool_use:
   a. Emit "running" event
   b. Dispatch tool
   c. Emit "done" event
   d. Inject tool_result
   e. Loop back to step 4 (up to max iterations)
7. If no tool_use (stop/finish):
   a. Stream response to UI
   b. Add to conversation history
```

## Dependencies

**Required:**
- Phase 5 (Live AI Connection) - COMPLETE ✓

**Enables:**
- Phase 7 (Polish & Release) - Full OpenClaw compatibility

## OpenClaw Compatibility

After Phase 6, WebClaw matches OpenClaw's core behavior:

| Feature | OpenClaw | WebClaw After Phase 6 |
|---------|----------|----------------------|
| Tool protocol | tool_use → execute → tool_result | ✅ Same |
| Tool definitions | Injected into system prompt | ✅ Via API tools field |
| Context management | 20 msg / 75% threshold | ✅ Same |
| Summarization | LLM-based with history replacement | ✅ Same |
| Memory flush | Before compaction | ✅ Same |
| Streaming | Token-by-token | ✅ Same |

## Execution Order

```bash
# Wave 1: Provider tool support
cat .planning/phases/06-real-agent-loop/06-01-PLAN.md

# Wave 2: Agent loop wiring
cat .planning/phases/06-real-agent-loop/06-02-PLAN.md

# Wave 3: Real summarization
cat .planning/phases/06-real-agent-loop/06-03-PLAN.md

# Wave 4: Memory flush
cat .planning/phases/06-real-agent-loop/06-04-PLAN.md

# Wave 5: Integration
cat .planning/phases/06-real-agent-loop/06-05-PLAN.md
```

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Provider API differences | Abstract tool handling, provider-specific implementations |
| Tool loop infinite loops | Hard limit at 10 iterations |
| Token counting inaccuracy | Make configurable, document approximation |
| Summarization failures | Fail gracefully, continue without summary |
| Memory flush blocking | Run async (goroutine), don't block summarization |

## Testing Strategy

1. **Unit tests:** Each provider's tool parsing
2. **Integration tests:** Tool flow from registry to provider
3. **E2E tests:** Full conversation with tool use
4. **Manual testing:** Real API keys required for live validation

## Estimated Effort

| Plan | Hours | Tasks |
|------|-------|-------|
| 06-01 | 2-3 | 5 tasks (provider changes) |
| 06-02 | 1 | 5 tasks (wiring) |
| 06-03 | 1-2 | 5 tasks (summarization) |
| 06-04 | 1 | 5 tasks (memory flush) |
| 06-05 | 2-3 | 5 tasks (token counting + E2E) |
| **Total** | **7-10** | **25 tasks** |

## Post-Phase 6

After completing Phase 6:
1. Update ROADMAP.md to mark Phase 6 complete
2. Run full E2E test with real API keys
3. Performance test (token latency, tool execution time)
4. Begin Phase 7 (Polish & Release)

---

**Last Updated:** 2026-03-02  
**Status:** Ready for execution
