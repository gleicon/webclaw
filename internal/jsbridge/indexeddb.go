//go:build js && wasm

package jsbridge

import "syscall/js"

// indexedDBOpen wraps indexedDB.open(dbName, version).
// Phase 1: thin wrapper for smoke test callability only.
// Full IndexedDB operations (config, memory) come in Phases 2-3.
func indexedDBOpen(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.Null()
	}
	dbName := args[0].String()
	version := 1
	if len(args) > 1 {
		version = args[1].Int()
	}
	return js.Global().Get("indexedDB").Call("open", dbName, version)
}
