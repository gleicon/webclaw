---
phase: 06-real-agent-loop
plan: 01
subsystem: provider
tags: [tools, llm, streaming, anthropic, openai, openrouter]

# Dependency graph
requires:
  - phase: 05-live-ai-connection
    provides: Provider interface and streaming infrastructure
provides:
  - Tool definitions in CompletionRequest
  - Anthropic tool_use event parsing
  - OpenAI tool_calls parsing
  - OpenRouter tool passthrough
  - Token with FinishReason="tool_use" and ToolName/ToolInput/ToolUseID
affects: [06-real-agent-loop]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Generic tool schema converted to provider-specific formats"
    - "Accumulating partial JSON from streaming deltas"
    - "Unified Token with FinishReason tool_use across providers"

key-files:
  created:
    - internal/provider/anthropic_test.go
    - internal/provider/openai_test.go
  modified:
    - internal/provider/provider.go
    - internal/provider/anthropic.go
    - internal/provider/openai.go
    - internal/provider/openrouter.go

key-decisions:
  - "Used FinishReason='tool_use' consistently across all providers (Anthropic uses 'tool_use' natively, OpenAI/OpenRouter use 'tool_calls' but we normalize to 'tool_use')"
  - "Accumulate JSON fragments during streaming, parse at message_stop/finish"
  - "Convert generic map[string]interface{} tools to provider-specific formats at request time"
  - "Handle one tool at a time per token (simplifies agent loop integration)"

patterns-established:
  - "Tool conversion helpers: convertToOpenAITools, convertToOpenRouterTools with getString/getMap helpers"
  - "Streaming tool tracking: map[int]*toolCallState for accumulating across delta events"
  - "SSE event structs include all needed fields for content_block_start, content_block_delta, etc."
  - "Test coverage: request serialization, event parsing, JSON accumulation, Token fields"

requirements-completed:
  - AGNT-01

duration: 3min
completed: "2026-03-04"
---

# Phase 06 Plan 01: Provider-Side Tool Support Summary

**Provider-side tool support: LLM requests include tool definitions and streaming responses parse tool_use/tool_calls into unified Token format.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-04T00:06:57Z
- **Completed:** 2026-03-04T00:10:01Z
- **Tasks:** 5
- **Files modified:** 6 (4 modified, 2 created)

## Accomplishments

- Added `Tools []map[string]interface{}` field to CompletionRequest with `omitempty` JSON tag
- Implemented Anthropic tool_use parsing: content_block_start detection, input_json_delta accumulation, message_stop handling
- Implemented OpenAI tool_calls parsing: ToolCalls delta tracking, finish_reason detection, arguments accumulation
- Updated OpenRouter to pass tools through and handle tool_calls (OpenAI-compatible format)
- Created comprehensive tests for both Anthropic and OpenAI provider tool handling

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend CompletionRequest with Tools field** - `4dc8dbf` (feat)
2. **Task 2: Implement Anthropic tool_use parsing** - `83119ba` (feat)
3. **Task 3: Implement OpenAI tool_calls parsing** - `5758f04` (feat)
4. **Task 4: Update OpenRouter to pass through tools** - `9f5adb4` (feat)
5. **Task 5: Add provider tests for tool use** - `e3ae6fd` (test)

**Plan metadata:** `TBD` (docs: complete plan)

## Files Created/Modified

- `internal/provider/provider.go` - Added Tools field to CompletionRequest
- `internal/provider/anthropic.go` - Added tool_use parsing with content_block_start/content_block_delta handling
- `internal/provider/openai.go` - Added tool_calls parsing with ToolCalls tracking
- `internal/provider/openrouter.go` - Added tool passthrough with OpenAI-compatible format
- `internal/provider/anthropic_test.go` - Tests for Anthropic tool request/response handling
- `internal/provider/openai_test.go` - Tests for OpenAI tool request/response handling

## Decisions Made

- **FinishReason normalization:** Used "tool_use" consistently across all providers, even though OpenAI/OpenRouter use "tool_calls" in their API. The provider layer normalizes to the common "tool_use" value.
- **JSON accumulation:** Accumulate partial JSON from streaming deltas (input_json_delta for Anthropic, function.arguments for OpenAI), then parse complete JSON at message_stop/finish.
- **One tool per token:** Handle one tool at a time per Token response. This simplifies agent loop integration. Multi-tool scenarios would require additional handling.
- **Tool conversion at request time:** Convert generic `[]map[string]interface{}` tools to provider-specific formats when building requests, not at storage time.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

Testing tool use requires live API keys:

- **ANTHROPIC_API_KEY** - Get from Anthropic Dashboard → API Keys
- **OPENAI_API_KEY** - Get from OpenAI Dashboard → API Keys

No external service configuration needed for the tool support itself.

## Next Phase Readiness

- **Ready for:** Agent loop integration with tool calling
- **Provider interface:** Complete with tool support
- **All three providers:** Support tools in requests and tool_use responses
- **Tests:** Verify request serialization, event parsing, and Token field population

---
*Phase: 06-real-agent-loop*
*Completed: 2026-03-04*

## Self-Check: PASSED

All created/modified files verified on disk:
- ✓ internal/provider/provider.go
- ✓ internal/provider/anthropic.go
- ✓ internal/provider/openai.go
- ✓ internal/provider/openrouter.go
- ✓ internal/provider/anthropic_test.go
- ✓ internal/provider/openai_test.go
- ✓ .planning/phases/06-real-agent-loop/06-01-SUMMARY.md

All commits verified:
- 4dc8dbf: Task 1 - Tools field
- 83119ba: Task 2 - Anthropic tool_use
- 5758f04: Task 3 - OpenAI tool_calls
- 9f5adb4: Task 4 - OpenRouter tools
- e3ae6fd: Task 5 - Provider tests
- 999ace8: Metadata updates
