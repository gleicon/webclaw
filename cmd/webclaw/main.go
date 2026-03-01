//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

func main() {
	jsbridge.Init()
	js.Global().Get("console").Call("log", "webclaw: WASM ready")
	<-make(chan struct{}) // block forever — Go runtime exits when main() returns
}
