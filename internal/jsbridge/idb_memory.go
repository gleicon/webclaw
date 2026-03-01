//go:build js && wasm

package jsbridge

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/memory"
)

// MemoryDB is the IndexedDB database name for memory storage.
const MemoryDB = "webclaw_memory"

// MemoryDBVersion is the schema version for memory database.
const MemoryDBVersion = 1

// MemoryStoreNames for different object stores.
const (
	MemoryStoreMain     = "memories"     // Active memories
	MemoryStoreIndex    = "memory_index" // BM25 inverted index
	MemoryStoreArchives = "archives"     // Archived memories
)

// MemoryDBOpen opens the memory database with proper object stores.
// Returns a Promise that resolves to the IDBDatabase.
func MemoryDBOpen() js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			request := js.Global().Get("indexedDB").Call("open", MemoryDB, MemoryDBVersion)

			request.Set("onupgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				db := request.Get("result")

				// Create main memories store with auto-increment
				if !db.Call("objectStoreNames").Call("contains", MemoryStoreMain).Bool() {
					store := db.Call("createObjectStore", MemoryStoreMain, map[string]interface{}{
						"keyPath": "id",
					})
					// Index for last_accessed for eviction queries
					store.Call("createIndex", "by_accessed", "last_accessed", map[string]interface{}{
						"unique": false,
					})
				}

				// Create BM25 index store
				if !db.Call("objectStoreNames").Call("contains", MemoryStoreIndex).Bool() {
					db.Call("createObjectStore", MemoryStoreIndex, map[string]interface{}{
						"keyPath": "term",
					})
				}

				// Create archives store
				if !db.Call("objectStoreNames").Call("contains", MemoryStoreArchives).Bool() {
					db.Call("createObjectStore", MemoryStoreArchives, map[string]interface{}{
						"keyPath": "id",
					})
				}

				return nil
			}))

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

// MemoryPut stores a MemoryDocument in IndexedDB.
// Returns a Promise that resolves when complete.
func MemoryPut(db js.Value, doc *memory.MemoryDocument) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			// Serialize document to JS object via JSON
			data, err := doc.Serialize()
			if err != nil {
				reject.Invoke(err.Error())
				return
			}

			// Parse JSON to JS object
			var jsObj map[string]interface{}
			if err := json.Unmarshal(data, &jsObj); err != nil {
				reject.Invoke(err.Error())
				return
			}

			transaction := db.Call("transaction", MemoryStoreMain, "readwrite")
			store := transaction.Call("objectStore", MemoryStoreMain)
			request := store.Call("put", jsObj)

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

// MemoryGet retrieves a MemoryDocument by ID from IndexedDB.
// Returns a Promise that resolves to *MemoryDocument or nil if not found.
func MemoryGet(db js.Value, id string) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", MemoryStoreMain, "readonly")
			store := transaction.Call("objectStore", MemoryStoreMain)
			request := store.Call("get", id)

			request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := request.Get("result")
				if result.IsUndefined() || result.IsNull() {
					resolve.Invoke(js.Null())
					return nil
				}

				// Convert JS object to JSON string, then to Go struct
				jsonStr := js.Global().Get("JSON").Call("stringify", result).String()
				doc, err := memory.DeserializeMemoryDocument([]byte(jsonStr))
				if err != nil {
					reject.Invoke(err.Error())
					return nil
				}

				resolve.Invoke(js.ValueOf(doc))
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

// MemoryDelete removes a MemoryDocument by ID from IndexedDB.
// Returns a Promise that resolves when complete.
func MemoryDelete(db js.Value, id string) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", MemoryStoreMain, "readwrite")
			store := transaction.Call("objectStore", MemoryStoreMain)
			request := store.Call("delete", id)

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

// MemoryGetAll retrieves all MemoryDocuments from IndexedDB.
// Returns a Promise that resolves to []*MemoryDocument.
func MemoryGetAll(db js.Value) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			transaction := db.Call("transaction", MemoryStoreMain, "readonly")
			store := transaction.Call("objectStore", MemoryStoreMain)
			request := store.Call("getAll")

			request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				results := request.Get("result")
				if results.IsUndefined() || results.IsNull() {
					resolve.Invoke(js.Global().Get("Array").New())
					return nil
				}

				// Convert each JS object to Go struct
				length := results.Get("length").Int()
				docs := make([]*memory.MemoryDocument, 0, length)

				for i := 0; i < length; i++ {
					item := results.Index(i)
					jsonStr := js.Global().Get("JSON").Call("stringify", item).String()
					doc, err := memory.DeserializeMemoryDocument([]byte(jsonStr))
					if err != nil {
						// Log error but continue with other documents
						fmt.Printf("Error deserializing memory document: %v\n", err)
						continue
					}
					docs = append(docs, doc)
				}

				resolve.Invoke(js.ValueOf(docs))
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

// ArchivePut stores a compressed memory archive.
// Returns a Promise that resolves when complete.
func ArchivePut(db js.Value, id string, compressedData []byte) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			// Create archive object
			archive := map[string]interface{}{
				"id":         id,
				"data":       compressedData,
				"archivedAt": js.Global().Get("Date").New().Call("toISOString").String(),
			}

			transaction := db.Call("transaction", MemoryStoreArchives, "readwrite")
			store := transaction.Call("objectStore", MemoryStoreArchives)
			request := store.Call("put", archive)

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

// GetStorageQuota returns storage quota information using navigator.storage.estimate().
// Returns a Promise that resolves to memory.QuotaInfo.
func GetStorageQuota() js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			storage := js.Global().Get("navigator").Get("storage")
			if storage.IsUndefined() || storage.IsNull() {
				reject.Invoke("navigator.storage not available")
				return
			}

			estimatePromise := storage.Call("estimate")
			if estimatePromise.IsUndefined() || estimatePromise.IsNull() {
				reject.Invoke("navigator.storage.estimate not available")
				return
			}

			estimatePromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				estimate := args[0]
				usage := int64(estimate.Get("usage").Float())
				quota := int64(estimate.Get("quota").Float())

				// Handle cases where quota might be undefined (in some browsers/privacy modes)
				if quota == 0 {
					// Use a default high value to prevent false overflow
					quota = 1024 * 1024 * 1024 // 1GB default
				}

				percent := float64(0)
				if quota > 0 {
					percent = float64(usage) / float64(quota) * 100
				}

				quotaInfo := memory.QuotaInfo{
					Usage:    usage,
					Quota:    quota,
					Percent:  percent,
					Overflow: usage > quota,
				}

				resolve.Invoke(js.ValueOf(quotaInfo))
				return nil
			})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				reject.Invoke(args[0])
				return nil
			}))
		}()

		return nil
	}))
}
