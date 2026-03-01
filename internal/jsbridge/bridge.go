//go:build js && wasm

package jsbridge

import "syscall/js"

// liveCallbacks keeps js.Func values alive to prevent GC.
// Startup-registered functions are never Released.
var liveCallbacks []js.Func

// Init registers the jsFetch and jsIndexedDB bridges on window.webclaw
// and fires the "webclaw:ready" CustomEvent. Called once from main().
func Init() {
	webclaw := js.Global().Get("Object").New()

	fetchFn := js.FuncOf(fetch)
	liveCallbacks = append(liveCallbacks, fetchFn)
	webclaw.Set("jsFetch", fetchFn)

	idb := js.Global().Get("Object").New()
	idbOpenFn := js.FuncOf(indexedDBOpen)
	liveCallbacks = append(liveCallbacks, idbOpenFn)
	idb.Set("open", idbOpenFn)
	webclaw.Set("jsIndexedDB", idb)

	js.Global().Set("webclaw", webclaw)

	js.Global().Call("dispatchEvent",
		js.Global().Get("CustomEvent").New("webclaw:ready"))
}
