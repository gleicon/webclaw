# WebClaw: The Full Vision

## What WebClaw Is

**WebClaw is the browser-native implementation of the OpenClaw specification.** It brings AI agent capabilities directly into the browser with zero installation, instant distribution, and full privacy.

### The OpenClaw Ecosystem

| Project | Language | Target | Size | Status |
|---------|----------|--------|------|--------|
| **OpenClaw** | TypeScript/Node.js | Server/Gateway | >1GB RAM, >500s boot | Original (240K stars) |
| **PicoClaw** | Go | RISC-V/ARM SBCs | <10MB, <1s boot | 21K stars |
| **NullClaw** | Zig | Cloudflare Workers | 678KB, <8ms boot | 2.6K stars |
| **WebClaw** | Go/WASM | Browser | ~3-5MB compressed | ✅ **v1.0 Ready** |

**WebClaw's Unique Value:** The browser isn't a limitation — it's an advantage. Instant distribution, native DOM access, IndexedDB persistence, Web Crypto security.

---

## ✅ COMPLETED: Phase 5 - Live AI Connection

**Status:** All tests passed, production ready

**What's Working:**
- ✅ Anthropic API (claude-sonnet, claude-opus) - direct browser CORS
- ✅ OpenAI API (gpt-4o, gpt-3.5-turbo) - direct browser access
- ✅ Encrypted API key storage (IndexedDB + Web Crypto AES-GCM)
- ✅ Real streaming responses (Server-Sent Events)
- ✅ Tool calls (web_fetch, web_search, memory_store/search)
- ✅ Pause/resume providers manually
- ✅ Auto-failover on invalid keys/no credits
- ✅ Static deployment (GitHub Pages, Netlify, S3)

**Files to Deploy:**
- `index.html` (~50KB)
- `static/wasm_exec.js` (~20KB)
- `dist/webclaw.wasm.br` (~887KB compressed)

---

## 📋 PLANNED: Future Phases

### Phase 6: Local Bridge Binary (webclaw-bridge)
**Goal:** Unlock capabilities browsers can't do natively

**Features:**
- File I/O (read/write local files)
- Shell execution (run commands, build tools)
- Git operations (clone, commit, push)
- Chrome DevTools Protocol (CDP) access
- WebSocket connection: `ws://localhost:18800`
- Security: 6-digit OTP + bearer token, 127.0.0.1-only

**Use Cases:**
- "Analyze my codebase" (reads local files via bridge)
- "Commit these changes" (git commands via bridge)
- "Run the tests" (shell execution via bridge)

---

### Phase 7: JS/TS Plugin SDK
**Goal:** Extensible plugin system for custom tools and integrations

**Plugin API:**
```typescript
// Example plugin
registerTool('my_api', async (args) => {
  const response = await fetch(args.endpoint);
  return response.json();
});

registerHook('message:received', (msg) => {
  console.log('Got message:', msg);
});

registerService('weather', {
  async getForecast(city) { /* ... */ }
});
```

**Features:**
- ES modules loaded at runtime from URLs
- Plugin manifest with permission declarations
- Lifecycle hooks (message:received, tool:before, etc.)
- Sandboxed execution environment

---

### Phase 8: Service Worker Mode
**Goal:** Background execution and persistence

**Features:**
- Agent survives tab closure
- Background heartbeat execution
- Resume on tab reopen
- Push notifications (when bridge is connected)

---

### Phase 9: Migration Tools
**Goal:** Import from other OpenClaw implementations

**Features:**
- OpenClaw workspace zip import
- PicoClaw config.json import
- NullClaw SQLite memory export import

---

### Phase 10: Advanced Features
**Goal:** Production polish and enterprise features

**Potential Features:**
- Multiple agent personas (switch identities)
- Conversation branching (fork/merge chat threads)
- Voice input/output (Web Speech API)
- Image generation integration (DALL-E, Midjourney)
- Multi-modal conversations (images, audio)
- Agent-to-agent communication (swarm mode)

---

## ❌ EXPLICITLY OUT OF SCOPE

Per the OpenClaw specification, these are NOT planned for WebClaw:

### Native Messaging Channels
**Why:** WebClaw IS the channel
- ❌ Telegram bot integration
- ❌ Discord bot integration  
- ❌ Slack bot integration
- ❌ WhatsApp integration

**Alternative:** WebClaw provides webchat UI + optional local bridge. Users interact via browser, not messaging platforms.

### Mobile App
**Why:** Browser-first architecture
- ❌ iOS/Android native apps
- ❌ React Native port

**Alternative:** WebClaw works in mobile browsers. PWA support may be added later.

### Full Node.js Compatibility
**Why:** Browser environment constraints
- ❌ Node.js API compatibility layer
- ❌ npm package installation in WASM

**Alternative:** JS/TS plugin SDK is the replacement, designed for browser constraints.

---

## 🤔 What About "OpenClaw Marketplace"?

**Current Status:** Not defined in the OpenClaw specification

**Likely Implementation:**
- Plugin registry (like npm but for WebClaw plugins)
- Curated tool collections
- Community-contributed integrations

**Would Require:**
- Phase 7 (Plugin SDK) completed
- Security review process
- Hosting infrastructure

**Not on current roadmap** — could be community-driven after v1.0 release.

---

## 🎯 Recommendation

**Current State:** WebClaw v1.0 is **production-ready** for personal use.

**Options:**

1. **Ship Now** (Phase 7 minimal)
   - README documentation
   - Deploy to GitHub Pages
   - Share with users
   - Gather feedback

2. **Add Bridge First** (Phase 6)
   - File I/O and shell access
   - More compelling for developers
   - Still browser-first

3. **Add Plugins First** (Phase 7)
   - Extensibility from day one
   - Community can build integrations
   - More complex to implement

**My Suggestion:** 
Ship Phase 7 (minimal polish) NOW. WebClaw is already useful:
- Private AI conversations in browser
- No server costs
- Instant deployment
- Working Anthropic + OpenAI

Then add Phase 6 (bridge) as v1.1 feature based on user feedback.

---

## Next Decision

**Which direction?**
1. 📦 **Ship v1.0 now** (Phase 7 minimal)
2. 🔧 **Build bridge first** (Phase 6)
3. 🔌 **Build plugins first** (Phase 7 full)
4. 📋 **Something else?**
