//go:build js && wasm

package agent

import (
	"context"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// workerBridge holds the callbacks registered from the worker
var workerBridge = &WorkerBridge{
	activeStreams: make(map[string]context.CancelFunc),
}

// globalAgentLoop is the singleton AgentLoop instance created in main.go.
// When set, handleStartStream reuses this loop (with its pre-configured router,
// toolRegistry, and workerBridge) instead of creating a fresh unconfigured loop.
var globalAgentLoop *AgentLoop

// SetGlobalAgentLoop stores the pre-configured AgentLoop for use by handleStartStream.
// Call this in main.go after creating the AgentLoop and calling SetRouter,
// SetToolRegistry, and SetWorkerBridge.
func SetGlobalAgentLoop(loop *AgentLoop) {
	globalAgentLoop = loop
}

// WorkerBridge provides the interface between WASM and the Web Worker
// Callbacks are set by worker.js and called by the agent loop
type WorkerBridge struct {
	onToken     func(token string)
	onComplete  func(result js.Value)
	onError     func(err error)
	onToolEvent func(name, status, summary, full string)

	// Track active streams for cancellation
	activeStreams map[string]context.CancelFunc
}

// InitWorkerBridge registers the worker bridge functions on the global webclaw object.
// Called from main.go during initialization.
// Returns the *WorkerBridge instance so main.go can wire it into the AgentLoop via SetWorkerBridge.
func InitWorkerBridge() *WorkerBridge {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		// Create webclaw global if it doesn't exist
		js.Global().Set("webclaw", js.Global().Get("Object").New())
		webclaw = js.Global().Get("webclaw")
	}

	// Create workerBridge object
	bridge := js.Global().Get("Object").New()

	// Export functions that worker.js will call
	// These are called from JS, spawn goroutines for async work

	// webclaw.workerBridge.startStream(payload)
	startStreamFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			js.Global().Get("console").Call("error", "startStream: missing payload")
			return js.Undefined()
		}

		payload := args[0]

		// Spawn goroutine to avoid blocking
		go func() {
			handleStartStream(payload)
		}()

		return js.Undefined()
	})
	jsbridge.RegisterCallback(startStreamFn)
	bridge.Set("startStream", startStreamFn)

	// webclaw.workerBridge.addMessage(role, content)
	addMessageFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			js.Global().Get("console").Call("error", "addMessage: missing arguments")
			return js.Undefined()
		}

		role := args[0].String()
		content := args[1].String()

		go func() {
			handleAddMessage(role, content)
		}()

		return js.Undefined()
	})
	jsbridge.RegisterCallback(addMessageFn)
	bridge.Set("addMessage", addMessageFn)

	// webclaw.workerBridge.abortStream()
	abortStreamFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			handleAbortStream()
		}()

		return js.Undefined()
	})
	jsbridge.RegisterCallback(abortStreamFn)
	bridge.Set("abortStream", abortStreamFn)

	// Register callback setters (worker.js calls these)
	// Initial placeholder values so worker.js can detect they're registered
	bridge.Set("onToken", js.Undefined())
	bridge.Set("onComplete", js.Undefined())
	bridge.Set("onError", js.Undefined())
	bridge.Set("onToolEvent", js.Undefined())

	// Export setter functions that worker.js can call
	registerCallbackFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return js.Undefined()
		}

		callbackName := args[0].String()
		callback := args[1]

		switch callbackName {
		case "onToken":
			workerBridge.onToken = func(token string) {
				callback.Invoke(token)
			}
		case "onComplete":
			workerBridge.onComplete = func(result js.Value) {
				callback.Invoke(result)
			}
		case "onError":
			workerBridge.onError = func(err error) {
				errObj := js.Global().Get("Object").New()
				errObj.Set("message", err.Error())
				errObj.Set("code", "STREAM_ERROR")
				callback.Invoke(errObj)
			}
		case "onToolEvent":
			workerBridge.onToolEvent = func(name, status, summary, full string) {
				callback.Invoke(name, status, summary, full)
			}
		}

		// Also set on the bridge object so worker.js can see it's registered
		bridge.Set(callbackName, callback)

		return js.Undefined()
	})
	jsbridge.RegisterCallback(registerCallbackFn)
	bridge.Set("registerCallback", registerCallbackFn)

	webclaw.Set("workerBridge", bridge)

	js.Global().Get("console").Call("log", "webclaw: worker bridge initialized")

	return workerBridge
}

// handleStartStream processes the START_STREAM message from the worker
func handleStartStream(payload js.Value) {
	js.Global().Get("console").Call("log", "webclaw: starting stream")

	// Extract parameters from payload
	var providerName, model string
	var messages []Message

	if !payload.IsUndefined() && !payload.IsNull() {
		if providerVal := payload.Get("provider"); !providerVal.IsUndefined() {
			providerName = providerVal.String()
		}
		if modelVal := payload.Get("model"); !modelVal.IsUndefined() {
			model = modelVal.String()
		}
	}

	// Create cancellable context for this stream
	ctx, cancel := context.WithCancel(context.Background())
	streamID := generateStreamID()
	workerBridge.activeStreams[streamID] = cancel

	// Start the agent loop.
	// Use the global pre-configured loop (with router/toolRegistry/workerBridge)
	// if one was set in main.go. Otherwise fall back to a new unconfigured loop.
	go func() {
		defer delete(workerBridge.activeStreams, streamID)

		loop := globalAgentLoop
		if loop == nil {
			loop = NewAgentLoop(providerName, model)
		}
		err := loop.Run(ctx, messages, workerBridge)
		if err != nil {
			if workerBridge.onError != nil {
				workerBridge.onError(err)
			}
		}
	}()
}

// handleAddMessage adds a message to the conversation history
func handleAddMessage(role, content string) {
	// This will be implemented in the agent loop
	// For now, just log it
	js.Global().Get("console").Call("log", "webclaw: adding message", role)

	// Store message for context assembly
	// (implementation in context.go)
}

// handleAbortStream cancels the active stream
func handleAbortStream() {
	js.Global().Get("console").Call("log", "webclaw: aborting stream")

	// Cancel all active streams
	for id, cancel := range workerBridge.activeStreams {
		js.Global().Get("console").Call("log", "webclaw: cancelling stream", id)
		cancel()
	}

	// Clear the map
	workerBridge.activeStreams = make(map[string]context.CancelFunc)
}

// generateStreamID creates a unique stream identifier
func generateStreamID() string {
	// Simple timestamp-based ID using Date.now() static method
	return js.Global().Get("Date").Call("now").String()
}

// EmitToken sends a token to the UI via the worker callback
func (wb *WorkerBridge) EmitToken(token string) {
	if wb.onToken != nil {
		wb.onToken(token)
	}
}

// EmitToolEvent emits a tool event to the UI via the onToolEvent callback.
// worker.js registers onToolEvent via registerCallback('onToolEvent', fn).
// The callback posts a TOOL_EVENT postMessage to the main thread, which
// webclaw-host.js then dispatches as a webclaw:tool-event CustomEvent on window.
//
// Call BEFORE dispatch with status="running", AFTER with status="done" or "error".
// toolName: name of the tool being invoked
// status: "running", "done", or "error"
// summary: short human-readable description for the UI activity panel
// full: full content (may be long; used for LLM context)
func (wb *WorkerBridge) EmitToolEvent(toolName, status, summary, full string) {
	if wb.onToolEvent != nil {
		wb.onToolEvent(toolName, status, summary, full)
	}
}

// EmitComplete signals stream completion
func (wb *WorkerBridge) EmitComplete(success bool, content string) {
	if wb.onComplete != nil {
		result := js.Global().Get("Object").New()
		result.Set("success", success)
		result.Set("content", content)
		wb.onComplete(result)
	}
}

// EmitError sends an error to the UI
func (wb *WorkerBridge) EmitError(err error) {
	if wb.onError != nil {
		wb.onError(err)
	}
}
