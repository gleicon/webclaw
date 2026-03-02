# WebClaw Phase 5 - Comprehensive Testing Checklist

## Pre-Flight Setup

```bash
# 1. Build the WASM
make clean && make build

# 2. Start the static server
make serve

# 3. Open browser to http://localhost:8080

# 4. Open DevTools (F12) → Console tab
```

---

## Test 1: Demo Mode (No API Keys) ✅

**Purpose:** Verify graceful fallback when no keys configured

**Steps:**
1. Open `http://localhost:8080` in fresh browser (or clear IndexedDB)
2. Check Settings tab shows all providers with 🔴 red status dots
3. Check for "Demo Mode" banner at top
4. In Chat tab, type: `Say "hello"` and press Enter

**Expected:**
- Response: Demo mode message
- Console shows: `webclaw: no persisted key for [provider]`
- No errors in console

**Pass Criteria:** ✅ Demo mode works, clear messaging

---

## Test 2: Add OpenAI API Key ✅

**Purpose:** Verify provider registration and live streaming

**Prerequisites:** OpenAI API key (https://platform.openai.com/api-keys)

**Steps:**
1. Go to Settings tab
2. Paste API key in "OpenAI API Key" field
3. Click Save
4. Check Settings shows 🟢 green dot for OpenAI
5. Click 🧪 Test Connection button

**Expected:**
- Console shows: `webclaw: registered provider: openai`
- Console shows: `[OpenAI] Stream started: status= 200`
- Test button returns to 🧪 (not stuck on ⏳)
- Success toast appears

**Pass Criteria:** ✅ Provider works, streaming successful

---

## Test 3: Live Chat with OpenAI ✅

**Purpose:** Verify end-to-end real API call with streaming

**Steps:**
1. Ensure OpenAI key configured
2. Check model selector auto-switched to `openai/gpt-4o`
3. Type: `Say "test successful"` and press Enter

**Expected:**
- Console shows: `[OpenAI] Stream: model= gpt-4o`
- Tokens stream gradually (not all at once)
- Response contains: "test successful"
- Stream completes with: `webclaw: stream completed tokens: X`

**Verify Streaming:**
- Watch response appear **gradually**
- Check Network tab for SSE events

**Pass Criteria:** ✅ Real API call, incremental streaming

---

## Test 4: Add Anthropic API Key ⏸️

**Purpose:** Verify Anthropic provider (requires credits)

**Prerequisites:** Anthropic API key with credits

**Steps:**
1. Go to Settings tab
2. Paste API key in "Anthropic API Key" field
3. Click Save
4. Check Settings shows 🟢 green dot for Anthropic
5. Click 🧪 Test Connection

**Expected:**
- Console shows: `webclaw: registered provider: anthropic`
- Console shows: `[Anthropic] Stream started: status= 200`
- Success toast appears

**Alternative:** If no Anthropic credits, skip this test

---

## Test 5: Provider Auto-Switching ✅

**Purpose:** Verify UI auto-selects available provider

**Steps:**
1. Clear all keys (or use fresh browser)
2. Add ONLY OpenAI key
3. Check Chat tab model selector

**Expected:**
- Model auto-switches to first OpenAI model
- Console shows: `[UI] Auto-switching to available provider: openai`

**Pass Criteria:** ✅ Dynamic routing works

---

## Test 6: Invalid API Key Error (401) ✅

**Purpose:** Verify 401 error handling

**Steps:**
1. Go to Settings
2. Enter: `sk-invalid123456789` in OpenAI field
3. Save settings
4. Click 🧪 Test Connection

**Expected:**
- Error toast: "Invalid API key"
- Console shows: `[OpenAI] Error response body: {...}`
- UI stays responsive (not frozen)

**Pass Criteria:** ✅ Clear error message, graceful handling

---

## Test 7: Chat with Invalid Key ✅

**Purpose:** Verify chat fails gracefully

**Steps:**
1. Enter invalid OpenAI key
2. Try to send chat message

**Expected:**
- Error toast: "Invalid API key - please check Settings"
- Message not sent
- UI allows retry after fixing key

**Pass Criteria:** ✅ Error handled, UI responsive

---

## Test 8: Settings Persistence ✅

**Purpose:** Verify keys survive page refresh

**Steps:**
1. Add valid OpenAI key
2. Verify green dot appears
3. Refresh browser (F5)
4. Wait for WASM to load

**Expected:**
- Green dot reappears automatically
- Console shows: `webclaw: loaded persisted key for openai`
- No re-entry of key needed

**Pass Criteria:** ✅ Persistence works, auto-load on startup

---

## Test 9: Key Security ✅

**Purpose:** Verify keys never exposed in JavaScript

**Steps:**
1. With valid key configured, open DevTools → Console
2. Run: `localStorage.getItem('openai-api-key')`
3. Run: `indexedDB.open('webclaw-keystore')`
4. Search DevTools Sources for your API key

**Expected:**
- localStorage returns `null`
- IndexedDB shows encrypted data (not plaintext)
- API key NOT found anywhere in Sources

**Pass Criteria:** ✅ Keys encrypted at rest

---

## Test 10: Remove All Keys (Regression) ✅

**Purpose:** Verify graceful return to demo mode

**Steps:**
1. Have valid key configured
2. Clear all API key fields in Settings
3. Save settings
4. Send a chat message

**Expected:**
- Returns to demo mode response
- Demo banner reappears
- No console errors
- Can re-add keys later

**Pass Criteria:** ✅ Clean fallback to demo mode

---

## Test 11: Multiple Provider Support ✅

**Purpose:** Verify multiple providers can coexist

**Steps:**
1. Add OpenAI key
2. Add Anthropic key (if available)
3. Check Settings shows both green
4. Console shows: `[UI] Providers ready: 2 providers available`
5. Switch between models and chat

**Expected:**
- Both providers registered
- Model dropdown shows all models
- Can switch between providers in chat

**Pass Criteria:** ✅ Multiple providers work

---

## Test 12: Memory Clearing (Security) ✅

**Purpose:** Verify best-effort memory clearing

**Note:** Hard to fully verify, but code exists in:
- `keystore.ClearKey(apiKey)` called after registration
- Comment notes "best effort" (Go/WASM memory management limitations)

**Pass Criteria:** ✅ Code implemented

---

## Test 13: Tool Calls End-to-End ⏸️

**Purpose:** Verify tools work with live provider

**Steps:**
1. With OpenAI configured
2. Type: `Fetch https://example.com and tell me the title`

**Expected:**
- Tool Activity Panel shows `web_fetch` running
- Tool result passed to LLM
- Response references actual fetched content

**Alternative:** Can verify in demo mode that tool structure works

---

## Test 14: Rate Limiting (429) ⏸️

**Purpose:** Verify 429 error handling

**Approach:**
- Rapid requests OR use quota-limited key

**Expected:**
- Error toast: "Rate limited - please wait"
- Console shows HTTP 429
- UI suggests waiting

**Alternative:** Hard to trigger intentionally

---

## Summary

| Test | Status | Notes |
|------|--------|-------|
| 1. Demo Mode | ⬜ | |
| 2. OpenAI Key | ⬜ | Requires OpenAI key |
| 3. OpenAI Chat | ⬜ | Requires OpenAI key |
| 4. Anthropic Key | ⬜ | Requires Anthropic credits |
| 5. Auto-Switch | ⬜ | |
| 6. Invalid Key | ⬜ | |
| 7. Chat Error | ⬜ | |
| 8. Persistence | ⬜ | |
| 9. Security | ⬜ | |
| 10. Regression | ⬜ | |
| 11. Multiple | ⬜ | Optional |
| 12. Memory | ✅ | Code verified |
| 13. Tools | ⬜ | Optional |
| 14. Rate Limit | ⬜ | Optional |

## Post-Test: Deployment Package

After tests pass:

```bash
./deploy.sh ./dist-static
```

**Files to deploy:**
- `index.html`
- `static/wasm_exec.js`
- `dist/webclaw.wasm.br`

**Host anywhere:** GitHub Pages, Netlify, Vercel, S3, etc.

---

**Ready to run tests?** Start with Test 1 (clear browser data first).
