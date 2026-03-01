# Phase 4: Tools and Webchat UI - Context

**Gathered:** 2026-03-01
**Status:** Ready for planning

<domain>
## Phase Boundary

The developer can interact with the agent through a browser chat interface, use browser tools (`web_fetch`, `web_search`, `memory_store`, `memory_search`), and dogfood the full system from a browser tab — zero server dependency.

This phase delivers the user-facing experience: the chat UI, tool call display, settings panel, and identity file editor. The agent backend (providers, loop, memory) is already complete in Phase 3.

</domain>

<decisions>
## Implementation Decisions

### Chat UI Style
- Polished modern chat, not a minimal dev harness
- Use **Tailwind CSS via CDN** — no build step, works with existing vanilla JS + WASM architecture
- Ship the polished UI from day one (no throwaway prototype)

### Message Display
- Bubble layout with alignment: user messages right-aligned with accent color, agent messages left-aligned with neutral background
- Streaming tokens animate into the agent bubble as they arrive
- Clear user/agent turn separation

### Theme
- **Dark mode by default** — this is a developer tool
- Follows `prefers-color-scheme` as fallback; no manual toggle required for v1

### Layout
- **Three-tab navigation at top**: `Chat` | `Settings` | `Identity Files`
- Switching tabs replaces the main content area (no page reload)
- Two-column layout on the Chat tab: chat column (left/center) + tool activity side panel (always visible, right)

### Tool Call Side Panel
- Always visible alongside chat — not a slide-in or modal
- Updates live as tool calls happen
- Per tool call shows: tool name, status indicator (running / done / error), 1–2 line result summary
- Expandable for full details (collapsed by default)
- Covers all four browser tools: `web_fetch`, `web_search`, `memory_store`, `memory_search`

### Settings Tab
- API key entry for each provider (Anthropic, OpenAI, OpenRouter)
- Keys encrypted on save via existing Web Crypto bridge — never shown in plaintext after entry
- Provider/model defaults (synced with config in IndexedDB)
- Config export / import buttons (reuses existing `webclaw.exportImport` bridge)

### Provider & Model Selection
- **Dropdown in chat header** (always visible) for quick provider/model switching
- Changes take effect on next message (per-session override, does not write to config)
- Config-level defaults set in Settings tab

### Identity Files Tab
- List of 6 files on the left: IDENTITY.md, SOUL.md, USER.md, AGENTS.md, TOOLS.md, HEARTBEAT.md
- Click a file → opens in a textarea editor on the right
- Save button commits to IndexedDB via existing identity store bridge
- No file creation or deletion in v1 (edit only)

### Claude's Discretion
- Exact Tailwind color palette and spacing values
- Streaming cursor animation style
- Exact layout of the chat input area (attach button placement, send on Enter vs button)
- Empty state illustration/copy for first-run chat
- Tool panel collapse/expand interaction details

</decisions>

<specifics>
## Specific Ideas

- "Polished from the beginning" — user explicitly does not want a throwaway harness
- The side panel should feel like a live activity feed, not a log file
- Settings and Identity tabs should feel like panels in a developer tool (VS Code settings style is a reference point)

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `static/webclaw-host.js` — full streaming API: `startStream()`, `addMessage()`, `abortStream()`, events (`webclaw:host-ready`, `webclaw:config-ready`, `webclaw:identity-ready`)
- `webclaw.exportImport.exportConfig()` / `importConfig(jsonContent)` — already bridged from WASM, ready to wire into Settings tab buttons
- `static/worker.js` — Web Worker already handles streaming; Phase 4 just consumes tokens via `onToken` callback

### Established Patterns
- All WASM ↔ JS communication goes through `syscall/js` exports registered on the `window.webclaw` object
- Events flow via CustomEvent on `window` (webclaw:* namespace)
- File import/export uses the `webclaw:request-export` / `webclaw:request-import` event pattern already in `webclaw-host.js`
- New UI-triggered actions follow the same CustomEvent or direct `webclaw.*` call pattern

### Integration Points
- Chat input → `webclawHost.addMessage(role, content)` + `webclawHost.startStream({provider, model, onToken, onComplete, onError})`
- Abort button → `webclawHost.abortStream()`
- Header model dropdown → feeds `provider` and `model` params into `startStream()`
- Settings API key form → calls WASM keystore bridge (needs new JS export wired in `main.go`)
- Identity file editor → calls WASM identity store bridge (needs new JS export wired in `main.go`)
- `webclaw:host-ready` event → trigger UI initialization (show chat, populate model dropdown from config)

</code_context>

<deferred>
## Deferred Ideas

- Native messaging channels (Telegram, Discord, Slack) — explicitly out of scope per PROJECT.md
- Mobile-responsive layout — browser-first, mobile later
- Manual light/dark toggle — system preference is sufficient for v1
- Markdown rendering in chat bubbles — could add later, not required for dogfooding

</deferred>

---

*Phase: 04-tools-and-webchat-ui*
*Context gathered: 2026-03-01*
