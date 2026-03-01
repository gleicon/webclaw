//go:build js && wasm

package jsbridge

import (
	"syscall/js"
)

// TriggerDownload triggers a file download in the browser
func TriggerDownload(filename string, content []byte) {
	// Create blob from content
	uint8Array := js.Global().Get("Uint8Array").New(len(content))
	js.CopyBytesToJS(uint8Array, content)

	blob := js.Global().Get("Blob").New(
		js.Global().Get("Array").New(uint8Array),
		map[string]interface{}{"type": "application/json"},
	)

	// Create download URL
	url := js.Global().Get("URL").Call("createObjectURL", blob)

	// Create and click anchor element
	a := js.Global().Get("document").Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", filename)
	a.Call("click")

	// Cleanup
	js.Global().Get("URL").Call("revokeObjectURL", url)
}

// ReadFile reads a File object from a file input
// Returns a Promise that resolves to the file content
func ReadFile(file js.Value) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			// Create FileReader
			reader := js.Global().Get("FileReader").New()

			// Set up onload handler
			reader.Set("onload", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := reader.Get("result")
				if result.Type() == js.TypeString {
					// Result is a string (text content)
					resolve.Invoke(result)
				} else {
					// Result is ArrayBuffer
					uint8Array := js.Global().Get("Uint8Array").New(result)
					bytes := make([]byte, uint8Array.Length())
					js.CopyBytesToGo(bytes, uint8Array)
					resolve.Invoke(string(bytes))
				}
				return nil
			}))

			// Set up onerror handler
			reader.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				reject.Invoke(reader.Get("error"))
				return nil
			}))

			// Read as text
			reader.Call("readAsText", file)
		}()

		return nil
	}))
}
