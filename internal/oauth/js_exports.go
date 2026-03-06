//go:build js && wasm

package oauth

import (
	"encoding/json"
	"syscall/js"
)

// RegisterJSExports exports OAuthManager methods to JavaScript
// This is called after the OAuthManager is created with a valid token store
func (m *OAuthManager) RegisterJSExports() {
	webclaw := js.Global().Get("webclaw")
	oauthObj := webclaw.Get("oauth")

	if oauthObj.IsUndefined() || oauthObj.IsNull() {
		return
	}

	// Export initiateConnection
	initiateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}
		provider := args[0].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				err := m.InitiateConnection(provider)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				resolve.Invoke(js.Undefined())
			}()

			return nil
		}))
	})
	oauthObj.Set("initiateConnection", initiateFn)

	// Export disconnect
	disconnectFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}
		provider := args[0].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				err := m.Disconnect(provider)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				resolve.Invoke(js.Undefined())
			}()

			return nil
		}))
	})
	oauthObj.Set("disconnect", disconnectFn)

	// Export getConnectionStatus
	statusFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]

			go func() {
				statuses := m.ListConnections()

				// Convert to JSON and then to JS object
				jsonData, _ := json.Marshal(statuses)
				jsData := js.Global().Get("JSON").Call("parse", string(jsonData))
				resolve.Invoke(jsData)
			}()

			return nil
		}))
	})
	oauthObj.Set("getConnectionStatus", statusFn)

	// Export isConnected helper
	isConnectedFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}
		provider := args[0].String()
		return js.ValueOf(m.IsConnected(provider))
	})
	oauthObj.Set("isConnected", isConnectedFn)
}
