//go:build js && wasm

package jsbridge

import (
	"encoding/json"
	"syscall/js"
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

// MemoryDocument represents a memory document for IndexedDB serialization.
// This is defined here to avoid import cycles with the memory package.
type MemoryDocument struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Embedding    []float32              `json:"embedding"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tokens       int                    `json:"tokens"`
	AccessCount  int                    `json:"access_count"`
	LastAccessed string                 `json:"last_accessed"`
	CreatedAt    string                 `json:"created_at"`
	Importance   int                    `json:"importance"`
}

// QuotaInfo represents storage quota estimation.
type QuotaInfo struct {
	Usage    int64   `json:"usage"`    // Bytes used
	Quota    int64   `json:"quota"`    // Total quota bytes
	Percent  float64 `json:"percent"`  // Usage percentage
	Overflow bool    `json:"overflow"` // True if usage > quota
}

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
				if !db.Get("objectStoreNames").Call("contains", MemoryStoreMain).Bool() {
					store := db.Call("createObjectStore", MemoryStoreMain, map[string]interface{}{
						"keyPath": "id",
					})
					// Index for last_accessed for eviction queries
					store.Call("createIndex", "by_accessed", "last_accessed", map[string]interface{}{
						"unique": false,
					})
				}

				// Create BM25 index store
				if !db.Get("objectStoreNames").Call("contains", MemoryStoreIndex).Bool() {
					db.Call("createObjectStore", MemoryStoreIndex, map[string]interface{}{
						"keyPath": "term",
					})
				}

				// Create archives store
				if !db.Get("objectStoreNames").Call("contains", MemoryStoreArchives).Bool() {
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
// The doc parameter should be a MemoryDocument (or compatible map).
// Returns a Promise that resolves when complete.
func MemoryPut(db js.Value, doc interface{}) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
		resolve := resolveReject[0]
		reject := resolveReject[1]

		go func() {
			// Serialize document to JSON, then to JS object
			data, err := json.Marshal(doc)
			if err != nil {
				reject.Invoke(err.Error())
				return
			}

			// Parse JSON to JS object
			jsObj := js.Global().Get("JSON").Call("parse", string(data))

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

// MemoryGet retrieves a document by ID from IndexedDB.
// Returns a Promise that resolves to the raw JS object or null if not found.
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
				resolve.Invoke(result)
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

// MemoryDelete removes a document by ID from IndexedDB.
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

// MemoryGetAll retrieves all documents from IndexedDB.
// Returns a Promise that resolves to array of raw JS objects.
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
				resolve.Invoke(results)
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
// Returns a Promise that resolves to QuotaInfo.
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

				// Return as plain JS object
				result := js.Global().Get("Object").New()
				result.Set("usage", usage)
				result.Set("quota", quota)
				result.Set("percent", percent)
				result.Set("overflow", usage > quota)

				resolve.Invoke(result)
				return nil
			})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				reject.Invoke(args[0])
				return nil
			}))
		}()

		return nil
	}))
}

// MemoryDocumentFromJS converts a JS memory document to our Go struct.
func MemoryDocumentFromJS(val js.Value) *MemoryDocument {
	if val.IsUndefined() || val.IsNull() {
		return nil
	}

	// Convert JS object to JSON string, then parse
	jsonStr := js.Global().Get("JSON").Call("stringify", val).String()

	var doc MemoryDocument
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		return nil
	}

	return &doc
}

// MemoryDocumentsFromJSArray converts a JS array of documents to Go slice.
func MemoryDocumentsFromJSArray(val js.Value) []*MemoryDocument {
	if val.IsUndefined() || val.IsNull() {
		return []*MemoryDocument{}
	}

	length := val.Get("length").Int()
	docs := make([]*MemoryDocument, 0, length)

	for i := 0; i < length; i++ {
		item := val.Index(i)
		if doc := MemoryDocumentFromJS(item); doc != nil {
			docs = append(docs, doc)
		}
	}

	return docs
}

// QuotaInfoFromJS converts a JS quota info object to Go struct.
func QuotaInfoFromJS(val js.Value) *QuotaInfo {
	if val.IsUndefined() || val.IsNull() {
		return nil
	}

	return &QuotaInfo{
		Usage:    int64(val.Get("usage").Float()),
		Quota:    int64(val.Get("quota").Float()),
		Percent:  val.Get("percent").Float(),
		Overflow: val.Get("overflow").Bool(),
	}
}
