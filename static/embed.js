// embed.js — Drop-in WebClaw chat widget for SaaS / third-party pages.
//
// Usage:
//   <script src="path/to/embed.js"></script>
//   <script>
//     webclaw.init({
//       wasmUrl:         '/static/webclaw.wasm',    // required
//       workerUrl:       '/static/worker.js',       // required
//       systemPrompt:    'You are a support agent…',
//       tools:           [],                        // allowlist; default = none
//       context:         { user: '…', page: '…' }, // injected each turn
//       proxyUrl:        '/api/ai-proxy',           // operator proxy (no raw key in browser)
//       model:           'gemini-nano/local',        // default; any vendor/model-id works
//       memoryNamespace: 'default',
//       position:        'bottom-right',            // or 'bottom-left'
//       theme:           { primaryColor: '#6366f1' },
//     });
//
//     // Update context as user navigates (SPA support).
//     // Replaces the entire context object — not merged. Pass all keys each call.
//     webclaw.setContext({ user: '…', page: '/billing' });
//   </script>

(function () {
    'use strict';

    if (window.__webclaw_embed_loaded) return;
    window.__webclaw_embed_loaded = true;

    // ── Config ────────────────────────────────────────────────────────────────

    let _config = null;
    let _worker = null;
    let _workerReady = false;
    let _liveContext = {};

    // Message types mirror worker.js
    const MSG = {
        INIT_WASM:    'INIT_WASM',
        START_STREAM: 'START_STREAM',
        ADD_MESSAGE:  'ADD_MESSAGE',
        ABORT_STREAM: 'ABORT_STREAM',
        WASM_READY:   'WASM_READY',
        WASM_ERROR:   'WASM_ERROR',
        TOKEN:        'TOKEN',
        COMPLETE:     'COMPLETE',
        ERROR:        'ERROR',
        STREAM_STARTED: 'STREAM_STARTED',
        TOOL_EVENT:   'TOOL_EVENT',
    };

    // ── Public API ────────────────────────────────────────────────────────────

    window.webclaw = window.webclaw || {};

    window.webclaw.init = function (config) {
        if (_config) { console.warn('[webclaw] init() called twice — ignored'); return; }
        _config = Object.assign({
            memoryNamespace: 'default',
            position: 'bottom-right',
            tools: [],
            theme: { primaryColor: '#6366f1' },
            systemPrompt: 'You are a helpful AI assistant.',
            context: {},
            model: 'gemini-nano/local',
        }, config);

        _liveContext = Object.assign({}, _config.context || {});

        if (!_config.wasmUrl || !_config.workerUrl) {
            console.error('[webclaw] init() requires wasmUrl and workerUrl');
            return;
        }

        _buildWidget();
        _startWorker();
    };

    window.webclaw.setContext = function (ctx) {
        _liveContext = Object.assign({}, ctx || {});
    };

    // ── Worker ────────────────────────────────────────────────────────────────

    function _startWorker() {
        _worker = new Worker(_config.workerUrl);
        _worker.onmessage = _onWorkerMessage;
        _worker.onerror = function (e) {
            console.error('[webclaw embed] worker error:', e.message);
            _setStatus('error', 'Worker failed — ' + e.message);
        };

        fetch(_config.wasmUrl)
            .then(function (r) { return r.arrayBuffer(); })
            .then(function (buf) {
                _worker.postMessage({ type: MSG.INIT_WASM, payload: { wasmBinary: buf } }, [buf]);
            })
            .catch(function (e) {
                console.error('[webclaw embed] failed to fetch WASM:', e);
                _setStatus('error', 'Failed to load AI — check wasmUrl');
            });
    }

    function _onWorkerMessage(e) {
        const { type, payload } = e.data;
        switch (type) {
            case MSG.WASM_READY:
                _workerReady = true;
                _setStatus('ready', 'Ready');
                break;
            case MSG.WASM_ERROR:
                _setStatus('error', 'AI init failed');
                console.error('[webclaw embed] WASM error:', payload);
                break;
            case MSG.TOKEN:
                _appendToken(payload.token);
                break;
            case MSG.COMPLETE:
                _onComplete();
                break;
            case MSG.ERROR:
                _onStreamError(payload.error);
                break;
            case MSG.TOOL_EVENT:
                _onToolEvent(payload);
                break;
        }
    }

    // ── Widget DOM (Shadow DOM for style isolation) ────────────────────────────

    let _shadow = null;
    let _panel = null;
    let _messageList = null;
    let _input = null;
    let _sendBtn = null;
    let _statusEl = null;
    let _assistantBubble = null; // bubble being streamed into
    let _isStreaming = false;

    function _buildWidget() {
        const primary = (_config.theme && _config.theme.primaryColor) || '#6366f1';
        const isLeft = _config.position === 'bottom-left';

        const host = document.createElement('div');
        host.id = 'webclaw-embed-host';
        host.style.cssText =
            'position:fixed;z-index:2147483647;' +
            (isLeft ? 'left:24px;' : 'right:24px;') +
            'bottom:24px;font-family:system-ui,sans-serif;';
        document.body.appendChild(host);

        _shadow = host.attachShadow({ mode: 'open' });

        // Minimal scoped styles — no Tailwind dependency
        const style = document.createElement('style');
        style.textContent = _widgetCSS(primary);
        _shadow.appendChild(style);

        const toggleBtn = document.createElement('button');
        toggleBtn.id = 'wc-toggle';
        toggleBtn.setAttribute('aria-label', 'Open AI assistant');
        toggleBtn.textContent = '✦';
        toggleBtn.addEventListener('click', _togglePanel);
        _shadow.appendChild(toggleBtn);

        _panel = document.createElement('div');
        _panel.id = 'wc-panel';
        _panel.setAttribute('aria-hidden', 'true');

        const header = document.createElement('div');
        header.id = 'wc-header';

        const title = document.createElement('span');
        title.textContent = 'AI Assistant';

        _statusEl = document.createElement('span');
        _statusEl.id = 'wc-status';
        _statusEl.textContent = 'Loading…';

        const closeBtn = document.createElement('button');
        closeBtn.id = 'wc-close';
        closeBtn.setAttribute('aria-label', 'Close');
        closeBtn.textContent = '×';
        closeBtn.addEventListener('click', _togglePanel);

        header.appendChild(title);
        header.appendChild(_statusEl);
        header.appendChild(closeBtn);

        _messageList = document.createElement('div');
        _messageList.id = 'wc-messages';
        _messageList.setAttribute('aria-live', 'polite');

        const inputArea = document.createElement('div');
        inputArea.id = 'wc-input-area';

        _input = document.createElement('textarea');
        _input.id = 'wc-input';
        _input.placeholder = 'Type a message…';
        _input.rows = 2;
        _input.setAttribute('aria-label', 'Message');
        _input.addEventListener('keydown', function (e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                _sendMessage();
            }
        });

        _sendBtn = document.createElement('button');
        _sendBtn.id = 'wc-send';
        _sendBtn.textContent = '↑';
        _sendBtn.setAttribute('aria-label', 'Send');
        _sendBtn.addEventListener('click', _sendMessage);

        inputArea.appendChild(_input);
        inputArea.appendChild(_sendBtn);

        _panel.appendChild(header);
        _panel.appendChild(_messageList);
        _panel.appendChild(inputArea);

        _shadow.appendChild(_panel);
    }

    function _togglePanel() {
        const hidden = _panel.getAttribute('aria-hidden') === 'true';
        _panel.setAttribute('aria-hidden', String(!hidden));
        _panel.style.display = hidden ? 'flex' : 'none';
        if (hidden && _input) _input.focus();
    }

    // ── Messaging ─────────────────────────────────────────────────────────────

    function _sendMessage() {
        if (!_workerReady || _isStreaming) return;
        const text = _input.value.trim();
        if (!text) return;
        _input.value = '';

        _addBubble('user', text);

        // Add to worker conversation history
        _worker.postMessage({ type: MSG.ADD_MESSAGE, payload: { role: 'user', content: text } });

        // Build effective system prompt = config systemPrompt + live context
        let effectiveSystem = _config.systemPrompt || '';
        const ctxKeys = Object.keys(_liveContext);
        if (ctxKeys.length > 0) {
            const ctxLines = ctxKeys.map(function (k) { return k + ': ' + _liveContext[k]; }).join('\n');
            effectiveSystem += '\n\n[Context]\n' + ctxLines;
        }

        _isStreaming = true;
        _sendBtn.disabled = true;
        _assistantBubble = _addBubble('assistant', '');
        _setStatus('streaming', '…');

        _worker.postMessage({
            type: MSG.START_STREAM,
            payload: {
                model: _config.model,
                systemPrompt: effectiveSystem,
                tools: _config.tools || [],
                memoryNamespace: _config.memoryNamespace,
                proxyUrl: _config.proxyUrl || null,
            }
        });
    }

    function _appendToken(text) {
        if (!_assistantBubble) return;
        const content = _assistantBubble.querySelector('.wc-bubble-content');
        if (content) content.textContent += text;
        _messageList.scrollTop = _messageList.scrollHeight;
    }

    function _onComplete() {
        _isStreaming = false;
        _sendBtn.disabled = false;
        _assistantBubble = null;
        _setStatus('ready', 'Ready');
    }

    function _onStreamError(msg) {
        _isStreaming = false;
        _sendBtn.disabled = false;
        if (_assistantBubble) {
            const content = _assistantBubble.querySelector('.wc-bubble-content');
            if (content && !content.textContent) {
                content.textContent = 'Error: ' + msg;
                _assistantBubble.classList.add('wc-error');
            }
        }
        _assistantBubble = null;
        _setStatus('ready', 'Ready');
    }

    function _onToolEvent(payload) {
        const indicator = document.createElement('div');
        indicator.className = 'wc-tool-event';
        const label = document.createElement('span');
        label.textContent = (payload.status === 'running' ? '⚙ ' : '✓ ') + payload.toolName;
        if (payload.summary) {
            const summary = document.createElement('span');
            summary.className = 'wc-tool-summary';
            summary.textContent = ' — ' + payload.summary;
            indicator.appendChild(label);
            indicator.appendChild(summary);
        } else {
            indicator.appendChild(label);
        }
        _messageList.appendChild(indicator);
        _messageList.scrollTop = _messageList.scrollHeight;
    }

    function _addBubble(role, text) {
        const wrapper = document.createElement('div');
        wrapper.className = 'wc-msg wc-msg-' + role;

        const bubble = document.createElement('div');
        bubble.className = 'wc-bubble';

        const content = document.createElement('span');
        content.className = 'wc-bubble-content';
        content.textContent = text;

        bubble.appendChild(content);
        wrapper.appendChild(bubble);
        _messageList.appendChild(wrapper);
        _messageList.scrollTop = _messageList.scrollHeight;
        return wrapper;
    }

    function _setStatus(state, text) {
        if (!_statusEl) return;
        _statusEl.textContent = text;
        _statusEl.setAttribute('data-state', state);
    }

    // ── CSS ───────────────────────────────────────────────────────────────────

    function _widgetCSS(primary) {
        return [
            '#wc-toggle{',
            '  width:52px;height:52px;border-radius:50%;',
            '  background:' + primary + ';border:none;cursor:pointer;',
            '  color:#fff;font-size:22px;display:flex;align-items:center;justify-content:center;',
            '  box-shadow:0 4px 12px rgba(0,0,0,.4);transition:transform .15s;',
            '}',
            '#wc-toggle:hover{transform:scale(1.08);}',
            '#wc-panel{',
            '  display:none;flex-direction:column;',
            '  position:absolute;bottom:64px;right:0;',
            '  width:340px;height:480px;',
            '  background:#1f2937;border:1px solid #374151;border-radius:12px;',
            '  box-shadow:0 8px 32px rgba(0,0,0,.5);overflow:hidden;',
            '}',
            '#wc-header{',
            '  display:flex;align-items:center;gap:8px;',
            '  padding:10px 12px;background:#111827;border-bottom:1px solid #374151;',
            '  font-size:13px;font-weight:600;color:#f3f4f6;flex-shrink:0;',
            '}',
            '#wc-status{',
            '  font-size:11px;font-weight:400;color:#6b7280;margin-left:4px;',
            '}',
            '#wc-status[data-state=ready]{color:#34d399;}',
            '#wc-status[data-state=streaming]{color:#fbbf24;}',
            '#wc-status[data-state=error]{color:#f87171;}',
            '#wc-close{',
            '  margin-left:auto;background:none;border:none;cursor:pointer;',
            '  color:#6b7280;font-size:18px;line-height:1;padding:0 2px;',
            '}',
            '#wc-close:hover{color:#f3f4f6;}',
            '#wc-messages{',
            '  flex:1;overflow-y:auto;padding:12px;display:flex;flex-direction:column;gap:8px;',
            '}',
            '.wc-msg{display:flex;}',
            '.wc-msg-user{justify-content:flex-end;}',
            '.wc-msg-assistant{justify-content:flex-start;}',
            '.wc-bubble{',
            '  max-width:80%;padding:8px 12px;border-radius:12px;',
            '  font-size:13px;line-height:1.5;color:#f3f4f6;white-space:pre-wrap;word-break:break-word;',
            '}',
            '.wc-msg-user .wc-bubble{background:' + primary + ';border-bottom-right-radius:3px;}',
            '.wc-msg-assistant .wc-bubble{background:#374151;border-bottom-left-radius:3px;}',
            '.wc-msg.wc-error .wc-bubble{background:#450a0a;color:#fca5a5;}',
            '.wc-tool-event{',
            '  font-size:11px;color:#6b7280;padding:2px 4px;',
            '}',
            '.wc-tool-summary{color:#9ca3af;}',
            '#wc-input-area{',
            '  display:flex;gap:6px;padding:10px 12px;',
            '  border-top:1px solid #374151;flex-shrink:0;',
            '}',
            '#wc-input{',
            '  flex:1;background:#111827;border:1px solid #374151;border-radius:8px;',
            '  color:#f3f4f6;font-size:13px;padding:6px 10px;resize:none;',
            '  font-family:inherit;outline:none;',
            '}',
            '#wc-input:focus{border-color:' + primary + ';}',
            '#wc-send{',
            '  width:34px;height:34px;border-radius:8px;background:' + primary + ';',
            '  border:none;cursor:pointer;color:#fff;font-size:16px;',
            '  display:flex;align-items:center;justify-content:center;flex-shrink:0;align-self:flex-end;',
            '}',
            '#wc-send:disabled{opacity:.4;cursor:default;}',
            '#wc-send:not(:disabled):hover{filter:brightness(1.1);}',
        ].join('\n');
    }

})();
