//go:build js && wasm

package jsbridge

import (
	"encoding/base64"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/crypto"
)

// liveCallbacks keeps js.Func values alive to prevent GC.
// Startup-registered functions are never Released.
var liveCallbacks []js.Func

// RegisterCallback registers a js.Func to prevent garbage collection.
// Exported for use by other packages that need to register JS callbacks.
func RegisterCallback(fn js.Func) {
	liveCallbacks = append(liveCallbacks, fn)
}

// Init registers the jsFetch and jsIndexedDB bridges on window.webclaw
// and fires the "webclaw:ready" CustomEvent. Called once from main().
func Init() {
	webclaw := js.Global().Get("Object").New()

	fetchFn := RegisterFetchCallback()
	liveCallbacks = append(liveCallbacks, fetchFn)
	webclaw.Set("jsFetch", fetchFn)

	idb := js.Global().Get("Object").New()
	idbOpenFn := js.FuncOf(indexedDBOpen)
	liveCallbacks = append(liveCallbacks, idbOpenFn)
	idb.Set("open", idbOpenFn)
	webclaw.Set("jsIndexedDB", idb)

	// Add crypto bridge
	cryptoObj := js.Global().Get("Object").New()

	// Encrypt function: webclaw.crypto.encrypt(plaintext, passphrase)
	encryptFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return js.Undefined()
		}
		plaintext := args[0].String()
		passphrase := args[1].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				encrypted, err := crypto.EncryptWithPassphrase([]byte(plaintext), passphrase)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				// Return as JS object
				result := js.Global().Get("Object").New()
				result.Set("ciphertext", base64.StdEncoding.EncodeToString(encrypted.Ciphertext))
				result.Set("iv", base64.StdEncoding.EncodeToString(encrypted.IV))
				result.Set("salt", base64.StdEncoding.EncodeToString(encrypted.Salt))
				resolve.Invoke(result)
			}()

			return nil
		}))
	})
	liveCallbacks = append(liveCallbacks, encryptFn)
	cryptoObj.Set("encrypt", encryptFn)

	// Decrypt function: webclaw.crypto.decrypt(ciphertext, iv, salt, passphrase)
	decryptFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 4 {
			return js.Undefined()
		}
		ciphertext := args[0].String()
		iv := args[1].String()
		salt := args[2].String()
		passphrase := args[3].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				// Decode base64
				ct, _ := base64.StdEncoding.DecodeString(ciphertext)
				ivBytes, _ := base64.StdEncoding.DecodeString(iv)
				saltBytes, _ := base64.StdEncoding.DecodeString(salt)

				encrypted := &crypto.EncryptedData{
					Ciphertext: ct,
					IV:         ivBytes,
					Salt:       saltBytes,
				}

				plaintext, err := crypto.DecryptWithPassphrase(encrypted, passphrase)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				resolve.Invoke(string(plaintext))
			}()

			return nil
		}))
	})
	liveCallbacks = append(liveCallbacks, decryptFn)
	cryptoObj.Set("decrypt", decryptFn)

	webclaw.Set("crypto", cryptoObj)
	js.Global().Set("webclaw", webclaw)

	js.Global().Call("dispatchEvent",
		js.Global().Get("CustomEvent").New("webclaw:ready"))
}
