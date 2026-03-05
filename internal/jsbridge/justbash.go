//go:build js && wasm

// Package jsbridge provides Go bindings for just-bash JavaScript bridge
// This enables file operations in the browser without requiring a local bridge binary
package jsbridge

import (
	"context"
	"fmt"
	"syscall/js"
	"time"
)

// JustBashResult represents the result of a just-bash command
type JustBashResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Success  bool
}

// JustBashFileInfo represents a file or directory entry
type JustBashFileInfo struct {
	Name        string
	Permissions string
	Owner       string
	Group       string
	Size        int64
	Date        string
	IsDirectory bool
	IsSymlink   bool
}

// JustBashSearchMatch represents a grep/search match
type JustBashSearchMatch struct {
	File string
	Line int
	Text string
}

// JustBashFsInfo represents filesystem information
type JustBashFsInfo struct {
	Filesystem string
	Size       string
	Used       string
	Available  string
	UsePercent string
	MountedOn  string
	Mode       string
}

var (
	// justBashBridge is the global JavaScript bridge object
	justBashBridge js.Value

	// isInitialized tracks if just-bash is ready
	isInitialized bool
)

// InitJustBash initializes the just-bash bridge
// This should be called from main() after the JavaScript bridge script is loaded
func InitJustBash(mode string, overlayRoot string) error {
	bridge := js.Global().Get("justBashBridge")
	if bridge.IsUndefined() || bridge.IsNull() {
		return fmt.Errorf("justBashBridge not found in global scope - ensure justbash-bridge.js is loaded")
	}

	justBashBridge = bridge

	// Initialize with options
	options := js.Global().Get("Object").New()
	options.Set("mode", mode)
	if overlayRoot != "" {
		options.Set("overlayRoot", overlayRoot)
	}

	// Call init and wait for promise
	promise := bridge.Call("init", options)

	// Wait for initialization
	result := awaitPromise(promise)
	if !result.Bool() {
		return fmt.Errorf("just-bash initialization failed")
	}

	isInitialized = true
	js.Global().Get("console").Call("log", "webclaw: just-bash bridge initialized")

	return nil
}

// IsJustBashReady returns true if just-bash is initialized and ready
func IsJustBashReady() bool {
	if !isInitialized || justBashBridge.IsUndefined() {
		return false
	}
	return justBashBridge.Call("isReady").Bool()
}

// ExecuteCommand runs a bash command via just-bash
func ExecuteCommand(ctx context.Context, command string, cwd string, env map[string]string) (*JustBashResult, error) {
	if !IsJustBashReady() {
		return nil, fmt.Errorf("just-bash not initialized")
	}

	options := js.Global().Get("Object").New()
	if cwd != "" {
		options.Set("cwd", cwd)
	}
	if len(env) > 0 {
		envObj := js.Global().Get("Object").New()
		for k, v := range env {
			envObj.Set(k, v)
		}
		options.Set("env", envObj)
	}

	// Set timeout from context
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline).Milliseconds()
		if timeout > 0 {
			options.Set("timeout", timeout)
		}
	}

	promise := justBashBridge.Call("exec", command, options)
	result := awaitPromise(promise)

	if result.IsUndefined() || result.IsNull() {
		return nil, fmt.Errorf("command execution returned undefined")
	}

	return &JustBashResult{
		Stdout:   result.Get("stdout").String(),
		Stderr:   result.Get("stderr").String(),
		ExitCode: result.Get("exitCode").Int(),
		Success:  result.Get("success").Bool(),
	}, nil
}

// ReadFile reads a file from the just-bash virtual filesystem
func JustBashReadFile(path string) (string, error) {
	if !IsJustBashReady() {
		return "", fmt.Errorf("just-bash not initialized")
	}

	promise := justBashBridge.Call("readFile", path)
	result := awaitPromise(promise)

	if result.IsUndefined() || result.IsNull() {
		return "", fmt.Errorf("failed to read file: %s", path)
	}

	return result.String(), nil
}

// WriteFile writes content to a file in the just-bash virtual filesystem
func JustBashWriteFile(path string, content string) error {
	if !IsJustBashReady() {
		return fmt.Errorf("just-bash not initialized")
	}

	promise := justBashBridge.Call("writeFile", path, content)
	result := awaitPromise(promise)

	if !result.Bool() {
		return fmt.Errorf("failed to write file: %s", path)
	}

	return nil
}

// ListDir lists directory contents
func JustBashListDir(path string, showAll bool, longFormat bool) ([]JustBashFileInfo, error) {
	if !IsJustBashReady() {
		return nil, fmt.Errorf("just-bash not initialized")
	}

	options := js.Global().Get("Object").New()
	options.Set("all", showAll)
	options.Set("long", longFormat)

	promise := justBashBridge.Call("listDir", path, options)
	result := awaitPromise(promise)

	if result.IsUndefined() || result.IsNull() {
		return nil, fmt.Errorf("failed to list directory: %s", path)
	}

	// Convert JS array to Go slice
	length := result.Get("length").Int()
	entries := make([]JustBashFileInfo, length)

	for i := 0; i < length; i++ {
		item := result.Index(i)
		entries[i] = JustBashFileInfo{
			Name:        item.Get("name").String(),
			Permissions: item.Get("permissions").String(),
			Owner:       item.Get("owner").String(),
			Group:       item.Get("group").String(),
			Size:        int64(item.Get("size").Int()),
			Date:        item.Get("date").String(),
			IsDirectory: item.Get("isDirectory").Bool(),
			IsSymlink:   item.Get("isSymlink").Bool(),
		}
	}

	return entries, nil
}

// SearchFiles searches for patterns in files
func JustBashSearchFiles(pattern string, path string, recursive bool, ignoreCase bool) ([]JustBashSearchMatch, error) {
	if !IsJustBashReady() {
		return nil, fmt.Errorf("just-bash not initialized")
	}

	options := js.Global().Get("Object").New()
	options.Set("recursive", recursive)
	options.Set("ignoreCase", ignoreCase)
	options.Set("lineNumber", true)

	promise := justBashBridge.Call("searchFiles", pattern, path, options)
	result := awaitPromise(promise)

	if result.IsUndefined() || result.IsNull() {
		return nil, fmt.Errorf("search failed")
	}

	// Convert JS array to Go slice
	length := result.Get("length").Int()
	matches := make([]JustBashSearchMatch, length)

	for i := 0; i < length; i++ {
		item := result.Index(i)
		matches[i] = JustBashSearchMatch{
			File: item.Get("file").String(),
			Line: item.Get("line").Int(),
			Text: item.Get("text").String(),
		}
	}

	return matches, nil
}

// GetFsInfo returns filesystem information
func JustBashGetFsInfo() (*JustBashFsInfo, error) {
	if !IsJustBashReady() {
		return nil, fmt.Errorf("just-bash not initialized")
	}

	promise := justBashBridge.Call("getFsInfo")
	result := awaitPromise(promise)

	if result.IsUndefined() || result.IsNull() {
		return nil, fmt.Errorf("failed to get filesystem info")
	}

	return &JustBashFsInfo{
		Filesystem: result.Get("filesystem").String(),
		Size:       result.Get("size").String(),
		Used:       result.Get("used").String(),
		Available:  result.Get("available").String(),
		UsePercent: result.Get("usePercent").String(),
		MountedOn:  result.Get("mountedOn").String(),
		Mode:       result.Get("mode").String(),
	}, nil
}

// GetCwd returns the current working directory
func JustBashGetCwd() string {
	if !IsJustBashReady() {
		return "/home/user"
	}

	return justBashBridge.Call("getCwd").String()
}

// ChangeDir changes the current working directory
func JustBashChangeDir(path string) error {
	if !IsJustBashReady() {
		return fmt.Errorf("just-bash not initialized")
	}

	promise := justBashBridge.Call("changeDir", path)
	result := awaitPromise(promise)

	if !result.Bool() {
		return fmt.Errorf("failed to change directory: %s", path)
	}

	return nil
}

// GetMode returns the current filesystem mode
func JustBashGetMode() string {
	if !IsJustBashReady() {
		return "virtual"
	}

	return justBashBridge.Call("getMode").String()
}

// awaitPromise waits for a JavaScript Promise to resolve
func awaitPromise(promise js.Value) js.Value {
	// Create channels for result
	resultCh := make(chan js.Value, 1)

	// Set up handlers
	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- args[0]
		return nil
	})
	defer thenFunc.Release()

	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultCh <- js.Undefined()
		return nil
	})
	defer catchFunc.Release()

	// Chain promise
	promise.Call("then", thenFunc).Call("catch", catchFunc)

	// Block until result
	return <-resultCh
}

// RegisterJustBashCallbacks registers callbacks for just-bash readiness
func RegisterJustBashCallbacks(onReady func(), onError func(error)) {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() {
		webclaw = js.Global().Get("Object").New()
		js.Global().Set("webclaw", webclaw)
	}

	webclaw.Set("onJustBashReady", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && args[0].Bool() {
			isInitialized = true
			if onReady != nil {
				onReady()
			}
		} else {
			var errMsg string
			if len(args) > 1 {
				errMsg = args[1].String()
			} else {
				errMsg = "just-bash initialization failed"
			}
			if onError != nil {
				onError(fmt.Errorf(errMsg))
			}
		}
		return nil
	}))
}
