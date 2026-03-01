//go:build js && wasm

// Package crypto provides Web Crypto API access for Go WASM
package crypto

import (
	"fmt"
	"syscall/js"
)

// subtleCrypto returns the crypto.subtle object
func subtleCrypto() js.Value {
	return js.Global().Get("crypto").Get("subtle")
}

// GenerateKey generates a new AES-GCM key
// Returns a Promise that resolves to a CryptoKey
func GenerateKey(length int) js.Value {
	algorithm := map[string]interface{}{
		"name":   "AES-GCM",
		"length": length,
	}
	usages := []string{"encrypt", "decrypt"}
	return subtleCrypto().Call("generateKey", algorithm, true, usages)
}

// ImportKey imports raw key bytes into a CryptoKey
// extractable: whether the key can be exported later
func ImportKey(keyBytes []byte, extractable bool) js.Value {
	// Create Uint8Array from key bytes
	keyData := js.Global().Get("Uint8Array").New(len(keyBytes))
	js.CopyBytesToJS(keyData, keyBytes)

	algorithm := map[string]interface{}{
		"name": "AES-GCM",
	}
	usages := []string{"encrypt", "decrypt"}

	return subtleCrypto().Call("importKey", "raw", keyData, algorithm, extractable, usages)
}

// ExportKey exports a CryptoKey to raw bytes
// Returns a Promise that resolves to an ArrayBuffer
func ExportKey(key js.Value) js.Value {
	return subtleCrypto().Call("exportKey", "raw", key)
}

// DeriveKey derives an AES-GCM key from a base key using PBKDF2
// baseKey: imported passphrase
// salt: random salt bytes
// iterations: PBKDF2 iteration count (recommend 100000)
// Returns a Promise that resolves to a CryptoKey
func DeriveKey(baseKey js.Value, salt []byte, iterations int) js.Value {
	// Create Uint8Array from salt
	saltData := js.Global().Get("Uint8Array").New(len(salt))
	js.CopyBytesToJS(saltData, salt)

	deriveParams := map[string]interface{}{
		"name":       "PBKDF2",
		"salt":       saltData,
		"iterations": iterations,
		"hash":       "SHA-256",
	}

	keyAlgorithm := map[string]interface{}{
		"name":   "AES-GCM",
		"length": 256,
	}

	usages := []string{"encrypt", "decrypt"}
	return subtleCrypto().Call("deriveKey", deriveParams, baseKey, keyAlgorithm, false, usages)
}

// ImportKeyPBKDF2 imports a passphrase for PBKDF2 derivation
func ImportKeyPBKDF2(passphrase []byte) js.Value {
	passData := js.Global().Get("Uint8Array").New(len(passphrase))
	js.CopyBytesToJS(passData, passphrase)

	algorithm := map[string]interface{}{
		"name": "PBKDF2",
	}
	usages := []string{"deriveKey"}
	return subtleCrypto().Call("importKey", "raw", passData, algorithm, false, usages)
}

// EncryptAESGCM encrypts plaintext with AES-GCM
// key: CryptoKey
// iv: 12-byte initialization vector
// plaintext: data to encrypt
// Returns a Promise that resolves to an ArrayBuffer (ciphertext + auth tag)
func EncryptAESGCM(key js.Value, iv []byte, plaintext []byte) js.Value {
	// Create Uint8Array from IV and plaintext
	ivData := js.Global().Get("Uint8Array").New(len(iv))
	js.CopyBytesToJS(ivData, iv)

	plainData := js.Global().Get("Uint8Array").New(len(plaintext))
	js.CopyBytesToJS(plainData, plaintext)

	algorithm := map[string]interface{}{
		"name": "AES-GCM",
		"iv":   ivData,
	}

	return subtleCrypto().Call("encrypt", algorithm, key, plainData)
}

// DecryptAESGCM decrypts ciphertext with AES-GCM
// key: CryptoKey
// iv: 12-byte initialization vector
// ciphertext: encrypted data (includes auth tag)
// Returns a Promise that resolves to an ArrayBuffer
func DecryptAESGCM(key js.Value, iv []byte, ciphertext []byte) js.Value {
	// Create Uint8Array from IV and ciphertext
	ivData := js.Global().Get("Uint8Array").New(len(iv))
	js.CopyBytesToJS(ivData, iv)

	cipherData := js.Global().Get("Uint8Array").New(len(ciphertext))
	js.CopyBytesToJS(cipherData, ciphertext)

	algorithm := map[string]interface{}{
		"name": "AES-GCM",
		"iv":   ivData,
	}

	return subtleCrypto().Call("decrypt", algorithm, key, cipherData)
}

// GetRandomValues fills a byte slice with cryptographically random values
func GetRandomValues(bytes []byte) {
	if len(bytes) == 0 {
		return
	}

	// Create Uint8Array view of the slice
	arr := js.Global().Get("Uint8Array").New(len(bytes))
	js.Global().Get("crypto").Call("getRandomValues", arr)
	js.CopyBytesToGo(bytes, arr)
}

// WaitForPromise waits for a JS Promise and returns the result or error
// This uses the goroutine-spawn pattern to avoid blocking the event loop
func WaitForPromise(promise js.Value) (js.Value, error) {
	resultCh := make(chan js.Value, 1)
	errorCh := make(chan error, 1)

	// Set up handlers before any async operations
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		select {
		case resultCh <- args[0]:
		default:
		}
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		select {
		case errorCh <- fmt.Errorf("%v", args[0]):
		default:
		}
		return nil
	}))

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return js.Undefined(), err
	}
}

// ArrayBufferToBytes converts a JS ArrayBuffer to Go []byte
func ArrayBufferToBytes(buf js.Value) []byte {
	uint8Array := js.Global().Get("Uint8Array").New(buf)
	bytes := make([]byte, uint8Array.Length())
	js.CopyBytesToGo(bytes, uint8Array)
	return bytes
}
