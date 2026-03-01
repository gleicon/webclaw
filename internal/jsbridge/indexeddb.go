//go:build js && wasm

package jsbridge

import (
	"fmt"
	"syscall/js"
)

// OpenWebClawDB opens the webclaw IndexedDB and ensures all required object stores exist.
// This is the single source of truth for database schema initialization.
// All packages (identity, keystore, config, memory) should use this instead of IDBOpen directly.
func OpenWebClawDB() (js.Value, error) {
	req := IDBOpen("webclaw", 4)

	dbCh := make(chan js.Value, 1)
	errCh := make(chan error, 1)

	// Handle schema upgrades - creates ALL required object stores
	req.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db := event.Get("target").Get("result")

		// Create identity object store
		if !db.Get("objectStoreNames").Call("contains", "identity").Bool() {
			db.Call("createObjectStore", "identity", map[string]interface{}{
				"keyPath": "filename",
			})
		}

		// Create keystore object store
		if !db.Get("objectStoreNames").Call("contains", "keystore").Bool() {
			db.Call("createObjectStore", "keystore", map[string]interface{}{
				"keyPath": "provider",
			})
		}

		// Create config object store (for future use)
		if !db.Get("objectStoreNames").Call("contains", "config").Bool() {
			db.Call("createObjectStore", "config", map[string]interface{}{
				"keyPath": "key",
			})
		}

		return nil
	}))

	req.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		dbCh <- req.Get("result")
		return nil
	}))

	req.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- fmt.Errorf("failed to open database: %v", req.Get("error"))
		return nil
	}))

	req.Set("onblocked", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errCh <- fmt.Errorf("database blocked - close other tabs")
		return nil
	}))

	select {
	case db := <-dbCh:
		return db, nil
	case err := <-errCh:
		return js.Null(), err
	}
}

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

// IDBOpen opens a database, returns the IDBOpenDBRequest
func IDBOpen(dbName string, version int) js.Value {
	return js.Global().Get("indexedDB").Call("open", dbName, version)
}

// IDBGet retrieves a value from an object store by key
// Returns a Promise that resolves to the value or undefined
func IDBGet(db js.Value, storeName string, key string) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", storeName, "readonly")
			store := transaction.Call("objectStore", storeName)
			request := store.Call("get", key)

			request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := request.Get("result")
				if result.IsUndefined() || result.IsNull() {
					resolve.Invoke(js.Null())
				} else {
					resolve.Invoke(result)
				}
				return nil
			}))

			request.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				reject.Invoke(request.Get("error"))
				return nil
			}))
		}()

		return nil
	}))
}

// IDBPut stores a value in an object store
// Returns a Promise that resolves when complete
func IDBPut(db js.Value, storeName string, value interface{}) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", storeName, "readwrite")
			store := transaction.Call("objectStore", storeName)
			request := store.Call("put", value)

			request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				resolve.Invoke(request.Get("result"))
				return nil
			}))

			request.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				reject.Invoke(request.Get("error"))
				return nil
			}))
		}()

		return nil
	}))
}

// IDBDelete removes a value from an object store by key
// Returns a Promise that resolves when complete
func IDBDelete(db js.Value, storeName string, key string) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", storeName, "readwrite")
			store := transaction.Call("objectStore", storeName)
			request := store.Call("delete", key)

			request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				resolve.Invoke(js.Undefined())
				return nil
			}))

			request.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				reject.Invoke(request.Get("error"))
				return nil
			}))
		}()

		return nil
	}))
}
