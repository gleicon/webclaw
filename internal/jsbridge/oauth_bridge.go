//go:build js && wasm

package jsbridge

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

// OAuthCallbackData holds the result of an OAuth popup flow
type OAuthCallbackData struct {
	Code      string `json:"code"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	ErrorDesc string `json:"error_description,omitempty"`
	Provider  string `json:"provider"`
}

// JSOAuthBridge provides JavaScript bridge for OAuth popup flow
// This is called by the OAuthManager to open popups and handle callbacks
type JSOAuthBridge struct {
	pendingCallbacks map[string]chan *OAuthCallbackData
}

// NewJSOAuthBridge creates a new OAuth bridge
func NewJSOAuthBridge() *JSOAuthBridge {
	return &JSOAuthBridge{
		pendingCallbacks: make(map[string]chan *OAuthCallbackData),
	}
}

// OpenOAuthPopup opens an OAuth popup and waits for the callback
// Returns the authorization code or an error
func (b *JSOAuthBridge) OpenOAuthPopup(authURL, provider, state string) (*OAuthCallbackData, error) {
	// Create a unique callback channel for this request
	callbackCh := make(chan *OAuthCallbackData, 1)
	b.pendingCallbacks[state] = callbackCh

	// Clean up after timeout or completion
	defer func() {
		delete(b.pendingCallbacks, state)
	}()

	// Call JavaScript to open popup
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		return nil, fmt.Errorf("webclaw object not found")
	}

	oauth := webclaw.Get("oauth")
	if oauth.IsUndefined() || oauth.IsNull() {
		return nil, fmt.Errorf("webclaw.oauth not found")
	}

	openPopup := oauth.Get("openPopup")
	if openPopup.IsUndefined() || openPopup.IsNull() {
		return nil, fmt.Errorf("webclaw.oauth.openPopup not found")
	}

	// Create promise handler
	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	// Call JS function
	go func() {
		promise := openPopup.Invoke(authURL, provider, state)
		if promise.IsUndefined() || promise.IsNull() {
			errorCh <- fmt.Errorf("openPopup returned undefined")
			return
		}

		// Handle promise
		promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultCh <- args[0]
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errMsg := "popup failed"
			if len(args) > 0 && !args[0].IsUndefined() {
				errMsg = args[0].String()
			}
			errorCh <- fmt.Errorf("%s", errMsg)
			return nil
		}))
	}()

	// Wait for result with timeout
	select {
	case result := <-resultCh:
		// Parse result
		jsonStr := js.Global().Get("JSON").Call("stringify", result).String()
		var data OAuthCallbackData
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			return nil, fmt.Errorf("failed to parse OAuth result: %w", err)
		}
		return &data, nil

	case err := <-errorCh:
		return nil, err

	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("OAuth popup timed out after 2 minutes")
	}
}

// HandleOAuthCallback is called from JavaScript when the popup sends a message
// This can be registered as a global callback
func (b *JSOAuthBridge) HandleOAuthCallback(data *OAuthCallbackData) {
	if ch, ok := b.pendingCallbacks[data.State]; ok {
		ch <- data
	}
}

// RegisterOAuthBridge registers OAuth bridge functions on window.webclaw.oauth
func RegisterOAuthBridge() js.Func {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		// webclaw object should already exist from bridge.go Init()
		return js.Func{}
	}

	// Get or create oauth object (preserve existing JS functions like openPopup)
	oauth := webclaw.Get("oauth")
	if oauth.IsUndefined() || oauth.IsNull() {
		oauth = js.Global().Get("Object").New()
		webclaw.Set("oauth", oauth)
	}

	// Register callback handler for OAuth results from popup
	handleCallbackFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}

		// Parse callback data from JS object
		jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()
		var data OAuthCallbackData
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			fmt.Println("[oauth] Failed to parse callback data:", err)
			return js.Undefined()
		}

		// Dispatch custom event that Go can listen for
		event := js.Global().Get("CustomEvent").New("webclaw:oauth-callback", map[string]interface{}{
			"detail": args[0],
		})
		js.Global().Call("dispatchEvent", event)

		return js.Undefined()
	})
	oauth.Set("handleCallback", handleCallbackFn)

	// Exchange code for token (called by popup callback page)
	exchangeCodeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 4 {
			return js.Undefined()
		}

		provider := args[0].String()
		code := args[1].String()
		codeVerifier := args[2].String()
		configJSON := args[3].String()

		// Create promise for async operation
		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				// Parse provider config
				var config map[string]interface{}
				if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
					reject.Invoke(fmt.Sprintf("failed to parse config: %v", err))
					return
				}

				// Build token request
				tokenURL := config["token_url"].(string)
				clientID := config["client_id"].(string)
				redirectURI := "about:blank"

				// Create form data
				params := map[string]string{
					"grant_type":    "authorization_code",
					"code":          code,
					"redirect_uri":  redirectURI,
					"code_verifier": codeVerifier,
					"client_id":     clientID,
				}

				// Use fetch API via jsbridge
				// This is handled in JS since it requires CORS
				result := map[string]interface{}{
					"token_url": tokenURL,
					"params":    params,
					"provider":  provider,
				}
				resultJSON, _ := json.Marshal(result)

				// Return the request details for JS to execute
				resolve.Invoke(string(resultJSON))
			}()

			return nil
		}))
	})
	oauth.Set("exchangeCode", exchangeCodeFn)

	return handleCallbackFn
}

// Global bridge instance
var globalOAuthBridge *JSOAuthBridge

// InitOAuthBridge initializes the OAuth bridge
// Called from main.go during startup
func InitOAuthBridge() {
	globalOAuthBridge = NewJSOAuthBridge()
	RegisterOAuthBridge()
}

// GetOAuthBridge returns the global OAuth bridge instance
func GetOAuthBridge() *JSOAuthBridge {
	return globalOAuthBridge
}
