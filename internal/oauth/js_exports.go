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

	// Export setClientId — configures a provider's client ID at runtime
	// Client IDs are public values (they appear in the OAuth URL), so no encryption needed.
	// Persistence is handled by the caller (JS localStorage).
	setClientIdFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return js.Undefined()
		}
		provider := args[0].String()
		clientId := args[1].String()
		if err := SetProviderClientID(provider, clientId); err != nil {
			js.Global().Get("console").Call("warn", "webclaw: oauth.setClientId:", err.Error())
		}
		return js.Undefined()
	})
	oauthObj.Set("setClientId", setClientIdFn)

	// Export savePATToken — saves a Personal Access Token for github or notion
	// Called from JS: await webclaw.oauth.savePATToken("github", "ghp_...")
	savePATFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return js.Undefined()
		}
		provider := args[0].String()
		pat := args[1].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				err := m.SavePAT(provider, pat)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				// Clear invalid flag when a new token is saved
				m.ClearInvalid(provider)
				resolve.Invoke(js.Undefined())
			}()

			return nil
		}))
	})
	oauthObj.Set("savePATToken", savePATFn)

	// Export markInvalid — marks a provider's token as invalid (called by tools on 401/403)
	// Called from JS: webclaw.oauth.markInvalid("github")
	markInvalidFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}
		provider := args[0].String()
		m.MarkInvalid(provider)
		return js.Undefined()
	})
	oauthObj.Set("markInvalid", markInvalidFn)
}
