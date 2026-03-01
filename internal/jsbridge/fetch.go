//go:build js && wasm

package jsbridge

import "syscall/js"

// fetch is a js.FuncOf callback that wraps window.fetch().
// The URL string is extracted BEFORE spawning the goroutine because
// args are GC-eligible once the callback returns to JS.
// Returns a Promise that resolves/rejects based on the underlying window.fetch result.
func fetch(this js.Value, args []js.Value) interface{} {
	url := args[0].String() // extract BEFORE goroutine; args invalid after return
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]
		go func() { // REQUIRED: without this, event loop deadlocks
			result := js.Global().Call("fetch", url)
			result.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				resolve.Invoke(args[0])
				return nil
			})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				reject.Invoke(args[0])
				return nil
			}))
		}()
		return nil
	}))
}
