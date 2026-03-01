# Phase 4: Tools and Webchat UI - Research

**Researched:** 2026-03-01
**Domain:** Vanilla JS/HTML chat UI + Go WASM browser tool implementations + syscall/js bridge extensions
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Chat UI Style:** Polished modern chat from day one — no throwaway harness. Tailwind CSS via CDN (no build step).
- **Message Display:** Bubble layout. User messages right-aligned with accent color. Agent messages left-aligned with neutral background. Streaming tokens animate into agent bubble as they arrive.
- **Theme:** Dark mode by default (developer tool). Follows `prefers-color-scheme` as fallback. No manual toggle required for v1.
- **Layout:** Three-tab navigation at top: `Chat` | `Settings` | `Identity Files`. Tab switching replaces main content area (no page reload). Two-column Chat tab: chat column (left/center) + tool activity side panel (always visible, right).
- **Tool Call Side Panel:** Always visible (not slide-in or modal). Updates live. Per tool call: tool name, status indicator (running / done / error), 1–2 line result summary. Expandable for full details (collapsed by default). Covers all four browser tools: `web_fetch`, `web_search`, `memory_store`, `memory_search`.
- **Settings Tab:** API key entry for Anthropic, OpenAI, OpenRouter. Keys encrypted on save via existing Web Crypto bridge — never shown in plaintext after entry. Provider/model defaults synced with config in IndexedDB. Config export / import buttons (reuses existing `webclaw.exportImport` bridge).
- **Provider & Model Selection:** Dropdown in chat header (always visible) for quick provider/model switching. Changes take effect on next message (per-session override, does not write to config). Config-level defaults set in Settings tab.
- **Identity Files Tab:** List of 6 files on left (IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md). Click → textarea editor on right. Save button commits to IndexedDB via existing identity store bridge. No file creation or deletion in v1 (edit only).
- **All WASM↔JS communication:** Through `syscall/js` exports registered on `window.webclaw` object.
- **Chat integration points:** `webclawHost.addMessage(role, content)` + `webclawHost.startStream({provider, model, onToken, onComplete, onError})`.
- **Keys encrypted via:** Existing keystore bridge.
- **Identity saves via:** Existing identity store bridge.

### Claude's Discretion
- Exact Tailwind color palette and spacing values
- Streaming cursor animation style
- Exact layout of the chat input area (attach button placement, send on Enter vs button)
- Empty state illustration/copy for first-run chat
- Tool panel collapse/expand interaction details

### Deferred Ideas (OUT OF SCOPE)
- Native messaging channels (Telegram, Discord, Slack) — explicitly out of scope per PROJECT.md
- Mobile-responsive layout — browser-first, mobile later
- Manual light/dark toggle — system preference is sufficient for v1
- Markdown rendering in chat bubbles — could add later, not required for dogfooding
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TOOL-01 | Agent can invoke `web_fetch` — fetches a URL via JS fetch(), returns content to agent | Tool registry pattern in Go + jsbridge.Fetch() already exists; need tool struct + registration |
| TOOL-02 | Agent can invoke `web_search` — queries DuckDuckGo and returns results | DuckDuckGo HTML search via jsbridge.Fetch(); no API key required; parse HTML response |
| TOOL-03 | Agent can invoke `memory_store` — stores a fact or document to memory | memory.Store interface exists; tool wraps StoreFact() from agent loop |
| TOOL-04 | Agent can invoke `memory_search` — recalls relevant memories for a query | memory.Store.Search() exists; tool wraps SearchMemory() from agent loop |
| TOOL-05 | Tool registry allows registering tools with name, description, JSON schema parameters, and execute function | New internal/tools package; pattern mirrors OpenClaw tool interface |
| TOOL-06 | Tool execution results have dual output: content fed back to LLM for next iteration, and display content for the UI | ToolResult struct with Content + DisplayContent fields; worker bridge extended to emit tool events |
| UI-01 | User can type a message and receive a streamed response in the browser | webclaw-host.js startStream() + onToken callback; streaming pipeline already functional |
| UI-02 | UI displays tool execution events (tool name, status, result summary) | New `webclaw:tool-event` CustomEvent fired from worker bridge; JS panel listens |
| UI-03 | User can view and edit identity files from a settings panel | New JS exports: `webclaw.identity.getFile(name)` + `webclaw.identity.putFile(name, content)` wired in main.go |
| UI-04 | User can configure provider API keys from a settings panel (keys encrypted on save) | New JS exports: `webclaw.keystore.setKey(provider, key)` + `webclaw.keystore.hasKey(provider)` wired in main.go |
| UI-05 | Conversation history is displayed with clear user/agent turn separation | DOM bubble rendering in index.html; existing onToken/onComplete callbacks sufficient |
</phase_requirements>

---

## Summary

Phase 4 is primarily a **frontend assembly and bridge extension phase**, not a new backend infrastructure phase. The Go backend is complete through Phase 3: providers route, the agent loop streams, memory is persisted with hybrid search. What remains is (1) implementing the four browser tools as a tool registry in Go, (2) extending the WASM JS exports to cover identity file editing and API key management, (3) adding a tool event emission mechanism to the worker bridge, and (4) replacing the minimal `index.html` harness with a full Tailwind-CDN chat UI.

The Go tool registry (TOOL-05) is the foundation everything else depends on. Without it, tools cannot be registered, and tool results (TOOL-06) have no way to feed back into the agent loop or emit UI events. The UI work (UI-01 through UI-05) depends on both the tool events and the new JS bridge exports for identity and keystore.

The key architectural constraint from prior phases holds in Phase 4: **no `net/http` — all HTTP goes through `jsbridge.Fetch()`**. `web_fetch` and `web_search` must use the existing fetch bridge. The DuckDuckGo search tool uses `https://html.duckduckgo.com/html/?q=...` which is accessible without an API key and returns parseable HTML.

**Primary recommendation:** Build Phase 4 in three sequential concerns: (1) Go tool registry + four tool implementations + WASM bridge exports, (2) worker bridge tool event emission, (3) full index.html Tailwind chat UI wired to all bridges.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Tailwind CSS (CDN) | v3 (play CDN) | Utility-first CSS | No build step; locked by user decision; `<script src="https://cdn.tailwindcss.com">` |
| Go syscall/js | stdlib (Go 1.21+) | WASM ↔ JS bridge | Project standard; already used everywhere |
| Web Workers API | browser native | Non-blocking agent loop | Established in Phase 3 |
| IndexedDB | browser native | Config, identity, memory persistence | Established in Phase 2 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| DuckDuckGo HTML search | n/a (public URL) | web_search tool backend | No API key; access via `https://html.duckduckgo.com/html/?q=` |
| Web Crypto API | browser native | API key encryption | Established in Phase 2 via jsbridge.crypto |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Tailwind CDN | Bootstrap CDN | User locked Tailwind; no consideration needed |
| DuckDuckGo HTML | SerpAPI / Brave Search API | Would require API key; DuckDuckGo HTML is zero-dependency |
| Vanilla JS | Alpine.js or Preact via CDN | User chose vanilla JS + WASM architecture; framework adds complexity with no benefit |

**Installation (no npm, CDN only):**
```html
<!-- Tailwind CSS - add to index.html head -->
<script src="https://cdn.tailwindcss.com"></script>
```

---

## Architecture Patterns

### Project Structure for Phase 4

New files to create:
```
internal/tools/
├── registry.go        # Tool registry: register, list, dispatch
├── tool.go            # Tool interface + ToolResult type
├── web_fetch.go       # web_fetch tool implementation
├── web_search.go      # web_search tool implementation
├── memory_tools.go    # memory_store + memory_search tools
└── bridge.go          # WASM JS exports for tool registry

static/
└── (index.html replaces minimal harness with full chat UI)
```

Modifications to existing files:
```
cmd/webclaw/main.go           # Register new keystore + identity bridge exports
internal/agent/worker_bridge.go   # Add tool event emission (webclaw:tool-event)
```

### Pattern 1: Tool Interface and Registry

**What:** A central registry stores named tool definitions with JSON schema and an execute function. The agent loop calls `registry.Dispatch(name, params)` after parsing an LLM tool_use response.

**When to use:** Phase 4 is the first phase requiring the agent to execute tools. The registry must exist before individual tools can be wired.

```go
// Source: project convention — mirrors OpenClaw tool interface pattern
// File: internal/tools/tool.go

// ToolResult is the dual-output result from tool execution
type ToolResult struct {
    // Content is fed back to the LLM as the tool_result message
    Content string
    // DisplayContent is shown in the UI tool panel (may differ from Content)
    DisplayContent string
    // IsError signals the LLM that the tool failed
    IsError bool
    // ToolName and Status are for UI event emission
    ToolName string
    Status   string // "running" | "done" | "error"
}

// Tool defines a single callable tool
type Tool struct {
    Name        string
    Description string
    InputSchema map[string]interface{} // JSON Schema object
    Execute     func(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

// Registry manages available tools
type Registry struct {
    tools map[string]*Tool
}
```

### Pattern 2: Tool Event Emission via CustomEvent

**What:** When the agent loop executes a tool, it fires `webclaw:tool-event` on the window (main thread). The worker cannot fire CustomEvents directly (no DOM access), so events must be posted through the worker bridge message channel and re-dispatched on the main thread.

**When to use:** Every tool execution — before (status: "running") and after (status: "done" or "error").

```javascript
// Source: existing webclaw-host.js handleWorkerMessage pattern
// Extended to handle TOOL_EVENT message type

// In handleWorkerMessage (webclaw-host.js):
case MSG_TYPES.TOOL_EVENT:
    window.dispatchEvent(new CustomEvent('webclaw:tool-event', {
        detail: {
            toolName: payload.toolName,
            status: payload.status,     // "running" | "done" | "error"
            summary: payload.summary,   // 1-2 line display text
            full: payload.full          // full result for expandable detail
        }
    }));
    break;
```

```go
// In worker_bridge.go — new EmitToolEvent function:
func (wb *WorkerBridge) EmitToolEvent(name, status, summary, full string) {
    // Posts TOOL_EVENT message to main thread via onToolEvent callback
    if wb.onToolEvent != nil {
        wb.onToolEvent(name, status, summary, full)
    }
}
```

### Pattern 3: New JS Bridge Exports in main.go

**What:** Identity file editing and API key management need new JS exports on `window.webclaw`. Following the established pattern from Phase 2 (exportImport bridge), new exports are registered in `main.go`.

**When to use:** UI-03 (identity editing) and UI-04 (API key settings).

```go
// Source: cmd/webclaw/main.go — following registerExportImportBridge pattern

// webclaw.identity.getFile(filename) → Promise<string>
// webclaw.identity.putFile(filename, content) → Promise<void>
// webclaw.identity.listFiles() → Promise<string[]>

// webclaw.keystore.setKey(provider, apiKey) → Promise<void>
// webclaw.keystore.hasKey(provider) → Promise<bool>
// webclaw.keystore.getKey(provider) → Promise<string>  (returns encrypted blob, NOT plaintext)
```

### Pattern 4: Tailwind Dark-Mode Chat UI Structure

**What:** The chat UI is a single HTML file with Tailwind CDN. State (current tab, conversation history, tool events) is managed in vanilla JS. No framework needed — the WASM bridge provides the data layer.

**When to use:** Replacing the minimal `index.html` harness for UI-01 through UI-05.

```html
<!-- Source: Tailwind CDN docs + project dark-mode decision -->
<!-- Structure overview for chat layout -->
<body class="bg-gray-900 text-gray-100 h-screen flex flex-col">
  <!-- Tab Bar -->
  <nav class="flex border-b border-gray-700">
    <button id="tab-chat" class="tab-btn px-6 py-3 ...">Chat</button>
    <button id="tab-settings" class="tab-btn px-6 py-3 ...">Settings</button>
    <button id="tab-identity" class="tab-btn px-6 py-3 ...">Identity Files</button>
  </nav>

  <!-- Chat Tab: two-column layout -->
  <div id="view-chat" class="flex flex-1 overflow-hidden">
    <!-- Chat column -->
    <div class="flex flex-col flex-1 overflow-hidden">
      <!-- Provider dropdown in chat header -->
      <div class="px-4 py-2 border-b border-gray-700 flex items-center gap-2">
        <select id="model-selector" class="bg-gray-800 ...">...</select>
      </div>
      <!-- Message list -->
      <div id="messages" class="flex-1 overflow-y-auto p-4 space-y-3">...</div>
      <!-- Input area -->
      <div class="p-4 border-t border-gray-700">
        <div class="flex gap-2">
          <textarea id="user-input" class="flex-1 bg-gray-800 ..."></textarea>
          <button id="send-btn" class="bg-indigo-600 ...">Send</button>
        </div>
      </div>
    </div>
    <!-- Tool activity panel — always visible -->
    <div id="tool-panel" class="w-72 border-l border-gray-700 overflow-y-auto p-3">
      <h2 class="text-sm font-semibold text-gray-400 mb-2">Tool Activity</h2>
      <div id="tool-events" class="space-y-2">...</div>
    </div>
  </div>

  <!-- Settings Tab -->
  <div id="view-settings" class="hidden p-6 space-y-6">...</div>

  <!-- Identity Files Tab -->
  <div id="view-identity" class="hidden flex h-full">...</div>
</body>
```

### Anti-Patterns to Avoid

- **Blocking the main thread with WASM initialization:** The existing pattern (Web Worker for streaming, main thread for config/identity) must be preserved. Never call `startStream` from main thread.
- **Storing API keys in plaintext JS variables:** Key entry form must immediately pass to `webclaw.keystore.setKey()` and clear the input value; never read back in JS.
- **DOM string building for chat messages:** Use `createElement` or a template element — never `innerHTML += userInput` (XSS risk even in a local dev tool).
- **Direct window.webclaw access before `webclaw:host-ready` event fires:** UI initialization must wait for this event.
- **net/http in tool implementations:** `web_fetch` and `web_search` must use `jsbridge.Fetch()`, not standard library HTTP.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP requests in browser tools | Custom XHR wrapper | `jsbridge.Fetch()` (existing) | Already handles WASM-to-JS promise bridging, timeout, error handling |
| API key encryption | Custom crypto in tool bridge | `webclaw.crypto.encrypt()` (existing jsbridge) | AES-256-GCM with PBKDF2 already implemented and tested in Phase 2 |
| Streaming SSE parsing | Custom SSE reader | Existing `provider.SSEParser` + `provider.StreamReader` | Already handles multi-line SSE, \r\n edge cases, incomplete chunks |
| Config persistence | Custom IndexedDB wrapper | `config.NewStorage()` / `identity.NewStore()` (existing) | Full error handling, goroutine-spawn pattern already established |
| Tab switching | Router library | 3-line JS toggle with `hidden` class | Three tabs — zero complexity to justify a library |
| Markdown rendering | Marked.js or similar | None — deferred per user decision | User explicitly deferred markdown rendering |

**Key insight:** The Go backend is feature-complete for Phase 4. The new work is wiring (bridge extensions + tool registry) and UI surface. Resist the urge to add new persistence layers or complex state management.

---

## Common Pitfalls

### Pitfall 1: Tool Events Crossing the Worker Boundary

**What goes wrong:** The agent loop (and tool execution) runs inside the Web Worker which has no DOM access. Attempting to fire `new CustomEvent(...)` or `window.dispatchEvent(...)` from worker.js or from WASM running in the worker will silently fail or throw.

**Why it happens:** Workers are sandboxed — they have `self` not `window`. CustomEvents are DOM APIs.

**How to avoid:** All tool events must be posted as structured messages from worker to main thread (`self.postMessage({type: 'TOOL_EVENT', ...})`), then re-dispatched as CustomEvents by `webclaw-host.js` on the main thread. Add `TOOL_EVENT` to `MSG_TYPES` in both `worker.js` and `webclaw-host.js`.

**Warning signs:** Tool panel never updates even though console shows tool executions completing; no errors but no events.

### Pitfall 2: DuckDuckGo HTML Parsing Fragility

**What goes wrong:** The `web_search` tool fetches `https://html.duckduckgo.com/html/?q=query` and parses the response. DuckDuckGo may return CAPTCHA pages, rate-limit responses, or change their HTML structure.

**Why it happens:** DuckDuckGo HTML endpoint is unofficial — no SLA or stable schema guarantee.

**How to avoid:** Parse conservatively — extract `<a class="result__a">` links and `.result__snippet` text. Wrap in robust error handling. Return graceful degradation (empty results with error message in DisplayContent) rather than panicking. Keep the tool result Content useful to the LLM even on partial failure.

**Warning signs:** Agent loop crashes on search; empty result sets with no error surfaced to user.

### Pitfall 3: Identity Store and Keystore Without New JS Exports

**What goes wrong:** The identity `Store` and `keystore.KeyStore` are initialized in `main.go` but their JS bridge functions only cover export/import config. UI-03 and UI-04 require new JS exports — `webclaw.identity.*` and `webclaw.keystore.*` — that do not yet exist. The UI will call them before they are registered.

**Why it happens:** Phase 2 only needed these for import/export workflows; direct file-by-file editing was deferred to Phase 4.

**How to avoid:** Register new exports in `main.go` following the same pattern as `registerExportImportBridge()`. Ensure they are registered before `webclaw:host-ready` fires. UI JavaScript must guard against undefined methods and show a loading state.

**Warning signs:** `TypeError: webclaw.identity.getFile is not a function` in DevTools when clicking the Identity Files tab.

### Pitfall 4: Tailwind CDN Dark Mode Class Conflicts

**What goes wrong:** Tailwind CDN's default dark mode uses `prefers-color-scheme` media query, but explicit `dark:` utility classes require the `darkMode: 'class'` config or the CDN's `media` default. On CDN, dark mode utilities like `dark:bg-gray-800` only apply when the user's OS is in dark mode (media query), not when you add a `dark` class to `<html>`.

**Why it happens:** The user decided "dark mode by default" following `prefers-color-scheme`. This aligns with Tailwind CDN's default `darkMode: 'media'` behavior — no config change needed as long as we're not adding a toggle.

**How to avoid:** Use standard dark: utilities. They will apply automatically for users whose OS is dark mode (which is the expected case for a developer tool). For the minority case, provide a sensible light mode fallback using base color classes. Do NOT attempt to force dark mode with a class override since that requires Tailwind config which the CDN doesn't easily support.

**Warning signs:** `dark:` classes not applying even in dark OS mode — check the CDN script URL is correct (`https://cdn.tailwindcss.com`, not the old v2 CDN).

### Pitfall 5: WASM Startup Race Between main.go Exports and UI Init

**What goes wrong:** `webclaw-host.js` dispatches `webclaw:host-ready` after both main thread WASM and worker WASM are ready. However, new identity/keystore exports registered in `main.go` may not be ready if the identity/keystore initialization fails silently (they have `// Don't exit` error handling). UI code that immediately calls these exports on `host-ready` may find undefined functions.

**Why it happens:** The existing `initializeIdentity()` and `initializeKeystore()` log errors but return without halting — the `webclaw` object exists but subkeys may not be set.

**How to avoid:** Register bridge exports unconditionally at the top of `main.go` init sequence (before the heavy identity/keystore async operations). Bridge functions should return Promise-rejected errors if the underlying store is unavailable, rather than not existing at all.

**Warning signs:** `webclaw.identity` is defined but `webclaw.identity.getFile` throws because the store initialization failed partway.

---

## Code Examples

Verified patterns from existing project code:

### Tool Registry and Dispatch

```go
// Source: internal pattern — following agent/worker_bridge.go goroutine-spawn convention
// File: internal/tools/registry.go

type Registry struct {
    tools map[string]*Tool
    mu    sync.RWMutex
}

func (r *Registry) Register(t *Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[t.Name] = t
}

func (r *Registry) Dispatch(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
    r.mu.RLock()
    t, ok := r.tools[name]
    r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
    return t.Execute(ctx, params)
}

func (r *Registry) ToAPISchema() []map[string]interface{} {
    // Returns tool definitions in Anthropic/OpenAI tool_use schema format
    // Used when building the system prompt or API request
}
```

### web_fetch Tool Implementation

```go
// Source: internal/jsbridge/fetch.go (Fetch function) — already established
// File: internal/tools/web_fetch.go

func NewWebFetchTool() *Tool {
    return &Tool{
        Name:        "web_fetch",
        Description: "Fetch the content of a URL and return it as text",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "url": map[string]interface{}{
                    "type":        "string",
                    "description": "The URL to fetch",
                },
            },
            "required": []string{"url"},
        },
        Execute: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
            url, _ := params["url"].(string)
            resp, err := jsbridge.Fetch(url, jsbridge.FetchOptions{Method: "GET"})
            if err != nil {
                return &ToolResult{IsError: true, Content: err.Error(),
                    DisplayContent: "fetch failed: " + err.Error()}, nil
            }
            content := string(resp.Body)
            summary := content
            if len(summary) > 200 {
                summary = summary[:200] + "..."
            }
            return &ToolResult{
                Content:        content,
                DisplayContent: fmt.Sprintf("HTTP %d — %s", resp.Status, summary),
                ToolName:       "web_fetch",
                Status:         "done",
            }, nil
        },
    }
}
```

### Tool Event via Worker Bridge

```go
// Source: internal/agent/worker_bridge.go EmitToken/EmitComplete pattern
// New method to add to WorkerBridge:

func (wb *WorkerBridge) EmitToolEvent(toolName, status, summary, full string) {
    if wb.onToolEvent != nil {
        wb.onToolEvent(toolName, status, summary, full)
    }
}
```

```javascript
// Source: static/worker.js — extending MSG_TYPES
const MSG_TYPES = {
    // ... existing types ...
    TOOL_EVENT: 'TOOL_EVENT',  // NEW
};

// In registerStreamingCallbacks():
self.webclaw.workerBridge.onToolEvent = function(toolName, status, summary, full) {
    self.postMessage({
        type: MSG_TYPES.TOOL_EVENT,
        payload: { toolName, status, summary, full }
    });
};
```

```javascript
// Source: static/webclaw-host.js handleWorkerMessage — add case:
case MSG_TYPES.TOOL_EVENT:
    window.dispatchEvent(new CustomEvent('webclaw:tool-event', {
        detail: event.data.payload
    }));
    break;
```

### Identity Bridge Registration in main.go

```go
// Source: cmd/webclaw/main.go registerExportImportBridge() — same pattern
// File: cmd/webclaw/main.go (new function registerIdentityBridge)

func registerIdentityBridge() {
    webclaw := js.Global().Get("webclaw")
    identityObj := js.Global().Get("Object").New()

    // webclaw.identity.getFile(filename) → Promise<string>
    getFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        filename := args[0].String()
        return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, rr []js.Value) interface{} {
            go func() {
                store, _ := identity.NewStore()
                defer store.Close()
                file, err := store.Get(filename)
                if err != nil {
                    rr[1].Invoke(err.Error())
                    return
                }
                if file == nil {
                    rr[0].Invoke("")
                    return
                }
                rr[0].Invoke(file.Content)
            }()
            return nil
        }))
    })
    jsbridge.RegisterCallback(getFn)
    identityObj.Set("getFile", getFn)

    // webclaw.identity.putFile(filename, content) → Promise<void>
    // ... similar pattern ...

    webclaw.Set("identity", identityObj)
}
```

### Tailwind Dark Mode Chat Bubble

```html
<!-- Source: Tailwind CDN v3 utility class conventions -->
<!-- User message bubble (right-aligned, accent color) -->
<div class="flex justify-end">
  <div class="max-w-[70%] rounded-2xl rounded-tr-sm px-4 py-2 bg-indigo-600 text-white text-sm">
    {{ message.content }}
  </div>
</div>

<!-- Agent message bubble (left-aligned, neutral) -->
<div class="flex justify-start">
  <div class="max-w-[70%] rounded-2xl rounded-tl-sm px-4 py-2 bg-gray-700 text-gray-100 text-sm">
    {{ message.content }}
    <!-- Streaming cursor -->
    <span id="cursor" class="inline-block w-2 h-4 bg-gray-400 animate-pulse ml-0.5 align-middle"></span>
  </div>
</div>
```

### Tab Switching (vanilla JS)

```javascript
// Source: project vanilla JS convention — minimal, no framework
const TABS = ['chat', 'settings', 'identity'];

function switchTab(active) {
    TABS.forEach(tab => {
        document.getElementById(`tab-${tab}`).classList.toggle('border-b-2', tab === active);
        document.getElementById(`tab-${tab}`).classList.toggle('border-indigo-500', tab === active);
        document.getElementById(`view-${tab}`).classList.toggle('hidden', tab !== active);
    });
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Tailwind CDN v2 (separate script tags) | Tailwind CDN v3 unified play script | Tailwind v3 | Single `<script src="https://cdn.tailwindcss.com">` tag handles everything |
| DuckDuckGo Instant Answers API | DuckDuckGo HTML search endpoint | 2023 | Instant Answers API deprecated; use `html.duckduckgo.com/html/?q=` |
| Tailwind dark mode via `dark` class | Default dark mode via `prefers-color-scheme` media query | Tailwind v3 | CDN defaults to media mode; class mode requires config override |

**Deprecated/outdated:**
- `https://unpkg.com/tailwindcss@2/dist/tailwind.min.css` — Tailwind v2 CDN: outdated, missing v3 utilities. Use `https://cdn.tailwindcss.com` instead.

---

## Open Questions

1. **Agent loop tool integration — is the loop currently wired to a real provider or still mock?**
   - What we know: `loop.go` calls `al.getProvider()` which returns `&mockProvider{}` (hardcoded).
   - What's unclear: Whether Phase 3 execution wired the real provider router into `getProvider()` before the mock was left in. The Phase 3 summary says provider routing is complete, but `loop.go` still shows the mock.
   - Recommendation: The plan for Phase 4 must include a task to wire `provider.NewRouter()` into `AgentLoop.getProvider()`. This is a prerequisite for any real LLM call from the UI.

2. **Tool call protocol — does the current agent loop handle LLM tool_use responses?**
   - What we know: The agent loop streams tokens but has no tool call detection or dispatch logic. The `Run()` method calls `provider.Stream()` and accumulates tokens only.
   - What's unclear: Whether the Phase 3 work added tool_use response parsing upstream of the loop, or whether this is entirely new in Phase 4.
   - Recommendation: Phase 4 must add tool call response detection to the agent loop. After collecting a full streamed response, the loop must check if the LLM requested a tool call (Anthropic: `stop_reason: "tool_use"`, OpenAI: `finish_reason: "tool_calls"`), dispatch through the registry, inject the tool_result, and loop again up to maxToolIterations.

3. **Keystore JS export — which function signature to use?**
   - What we know: `keystore.KeyStore` exists with encrypt/decrypt. The `webclaw.crypto.encrypt/decrypt` bridge exists. Phase 2 did not add `webclaw.keystore.setKey/getKey` exports.
   - What's unclear: Whether to expose raw key storage functions or a higher-level "store provider API key" abstraction.
   - Recommendation: Use provider-keyed abstraction: `webclaw.keystore.setKey(provider, plaintext)` encrypts and stores under `keystore:${provider}`; `webclaw.keystore.getKey(provider)` returns plaintext (decrypted in WASM, passed as string). The Settings UI form immediately passes the input value to setKey and clears it.

---

## Sources

### Primary (HIGH confidence)
- Existing codebase — all source files read directly; patterns and constraints are verified facts
- `internal/jsbridge/bridge.go`, `fetch.go` — jsbridge API surface
- `internal/agent/worker_bridge.go`, `loop.go` — agent loop and worker bridge
- `internal/provider/router.go`, `provider.go` — provider routing interface
- `static/webclaw-host.js`, `static/worker.js` — worker message protocol
- `cmd/webclaw/main.go` — WASM main and bridge registration pattern
- `.planning/phases/04-tools-and-webchat-ui/04-CONTEXT.md` — locked user decisions

### Secondary (MEDIUM confidence)
- Tailwind CSS v3 CDN documentation — `https://cdn.tailwindcss.com` unified CDN script behavior
- DuckDuckGo HTML endpoint — `https://html.duckduckgo.com/html/?q=` publicly known, widely used as zero-API-key search

### Tertiary (LOW confidence)
- DuckDuckGo HTML response structure (result selectors) — unofficial, subject to change; validate during implementation

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all core tech is in the existing codebase; Tailwind CDN is locked by user decision
- Architecture: HIGH — bridge patterns are established; tool registry follows existing conventions
- Pitfalls: HIGH (worker boundary, WASM startup race) / MEDIUM (DuckDuckGo HTML fragility) — established from code inspection
- Open questions: MEDIUM — identified gaps in loop.go that need confirmation during planning

**Research date:** 2026-03-01
**Valid until:** 2026-04-01 (stable stack; Tailwind CDN v3 and browser APIs are stable)
