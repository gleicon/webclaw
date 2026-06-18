//go:build js && wasm

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"syscall/js"
)

// GeminiNanoProvider uses Chrome's built-in LanguageModel API.
// No API key required — availability is gated by browser + hardware.
// Calls window.webclaw.geminiNano.streamPrompt (registered in worker.js).
type GeminiNanoProvider struct{}

func NewGeminiNanoProvider() *GeminiNanoProvider {
	return &GeminiNanoProvider{}
}

func (p *GeminiNanoProvider) Name() string { return "gemini-nano" }

// IsAvailable implements ConditionalProvider.
// Returns true only when Chrome exposes window.LanguageModel (Chrome 138+).
func (p *GeminiNanoProvider) IsAvailable() bool {
	lm := js.Global().Get("LanguageModel")
	return !lm.IsUndefined() && !lm.IsNull()
}

// MaxContextWindow returns 9216: Chrome's shared input+output token budget.
func (p *GeminiNanoProvider) MaxContextWindow(_ string) int { return 9216 }

func (p *GeminiNanoProvider) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("gemini-nano: embeddings not supported")
}

func (p *GeminiNanoProvider) Complete(ctx context.Context, req CompletionRequest) (*Token, error) {
	var full string
	var last Token
	for tok := range p.Stream(ctx, req) {
		if tok.FinishReason == "error" {
			return nil, errors.New(tok.Text)
		}
		full += tok.Text
		last = tok
	}
	last.Text = full
	return &last, nil
}

// geminiHistoryMsg is the JSON shape LanguageModel.create() expects for initialPrompts.
type geminiHistoryMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// parseMessagesForGemini splits a CompletionRequest message slice into the
// three parts Chrome's LanguageModel API requires: a system prompt, an
// initialPrompts history, and the final user message to send.
func parseMessagesForGemini(msgs []Message) (systemPrompt, userMsg string, history []geminiHistoryMsg) {
	for i, msg := range msgs {
		switch msg.Role {
		case "system":
			systemPrompt = msg.Content
		case "user":
			if i == len(msgs)-1 {
				userMsg = msg.Content
			} else {
				history = append(history, geminiHistoryMsg{Role: "user", Content: msg.Content})
			}
		case "assistant":
			history = append(history, geminiHistoryMsg{Role: "assistant", Content: msg.Content})
		}
	}
	return
}

func (p *GeminiNanoProvider) Stream(ctx context.Context, req CompletionRequest) <-chan Token {
	tokenChan := make(chan Token, 10)

	go func() {
		defer close(tokenChan)

		nano := js.Global().Get("webclaw").Get("geminiNano")
		if nano.IsUndefined() || nano.IsNull() {
			tokenChan <- Token{FinishReason: "error", Text: "gemini-nano: browser API bridge not ready"}
			return
		}

		systemPrompt, userMsg, history := parseMessagesForGemini(req.Messages)

		historyJSON, err := json.Marshal(history)
		if err != nil {
			tokenChan <- Token{FinishReason: "error", Text: "gemini-nano: marshal history: " + err.Error()}
			return
		}

		doneChan := make(chan struct{})
		var closeOnce sync.Once
		var lastErrMsg string
		var errMu sync.Mutex

		closeDone := func(errMsg string) {
			closeOnce.Do(func() {
				if errMsg != "" {
					errMu.Lock()
					lastErrMsg = errMsg
					errMu.Unlock()
				}
				close(doneChan)
			})
		}

		// Callbacks are released explicitly after JS finishes — never via defer.
		// The JS async function outlives this goroutine on ctx cancellation;
		// releasing while JS is still calling a callback panics the runtime.
		onToken := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 {
				// Drop token silently if caller cancelled rather than blocking.
				select {
				case tokenChan <- Token{Text: args[0].String()}:
				case <-ctx.Done():
				}
			}
			return nil
		})

		onDone := js.FuncOf(func(_ js.Value, _ []js.Value) any {
			closeDone("")
			return nil
		})

		onError := js.FuncOf(func(_ js.Value, args []js.Value) any {
			msg := "unknown error"
			if len(args) > 0 {
				msg = args[0].String()
			}
			closeDone(msg)
			return nil
		})

		// Fire-and-forget: streamPrompt is async, returns a Promise we ignore.
		// Callbacks wire the results back into Go channels.
		nano.Call("streamPrompt",
			systemPrompt,
			string(historyJSON),
			userMsg,
			onToken,
			onDone,
			onError,
		)

		ctxCancelled := false
		select {
		case <-doneChan:
		case <-ctx.Done():
			ctxCancelled = true
			<-doneChan // wait for JS to finish before releasing callbacks
		}

		onToken.Release()
		onDone.Release()
		onError.Release()

		errMu.Lock()
		e := lastErrMsg
		errMu.Unlock()

		switch {
		case ctxCancelled:
			// Caller moved on; send non-blocking so we don't hang if nobody reads.
			select {
			case tokenChan <- Token{FinishReason: "error", Text: "gemini-nano: context cancelled"}:
			default:
			}
		case e != "":
			tokenChan <- Token{FinishReason: "error", Text: "gemini-nano: " + e}
		default:
			tokenChan <- Token{FinishReason: "stop"}
		}
	}()

	return tokenChan
}
