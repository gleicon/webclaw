//go:build js && wasm

package memory

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"sort"
	"syscall/js"
	"time"

	"github.com/gleicon/webclaw/internal/jsbridge"
)

// LRUEvictor handles memory eviction based on LRU policy.
type LRUEvictor struct{}

// NewLRUEvictor creates a new LRU evictor.
func NewLRUEvictor() *LRUEvictor {
	return &LRUEvictor{}
}

// EvictToTarget removes memories until quota usage is below target percent.
func (e *LRUEvictor) EvictToTarget(db js.Value, targetPercent float64) error {
	// Get all memories
	getAllPromise := jsbridge.MemoryGetAll(db)

	docsChan := make(chan []*jsbridge.MemoryDocument, 1)
	errChan := make(chan error, 1)

	getAllPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		docs := jsbridge.MemoryDocumentsFromJSArray(args[0])
		docsChan <- docs
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to get memories for eviction: %v", args[0])
		return nil
	}))

	var docs []*jsbridge.MemoryDocument
	select {
	case docs = <-docsChan:
	case err := <-errChan:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout getting memories for eviction")
	}

	if len(docs) == 0 {
		return nil
	}

	// Sort by LRU score (higher score = evict first)
	sort.Slice(docs, func(i, j int) bool {
		return e.getLRUScore(docs[i]) > e.getLRUScore(docs[j])
	})

	// Evict until we hit the target
	evicted := 0
	for _, doc := range docs {
		// Check current quota
		quotaPromise := jsbridge.GetStorageQuota()
		quotaChan := make(chan *jsbridge.QuotaInfo, 1)

		quotaPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			info := jsbridge.QuotaInfoFromJS(args[0])
			quotaChan <- info
			return nil
		}))

		var quota *jsbridge.QuotaInfo
		select {
		case quota = <-quotaChan:
		case <-time.After(5 * time.Second):
			continue
		}

		// If we're below target, stop evicting
		if quota.Percent < targetPercent {
			break
		}

		// Archive before deletion
		if err := e.archiveAndDelete(db, doc); err != nil {
			// Log error but continue with next document
			fmt.Printf("Failed to archive memory %s: %v\n", doc.ID, err)
			continue
		}
		evicted++
	}

	fmt.Printf("Evicted %d memories to reach target quota\n", evicted)
	return nil
}

// getLRUScore calculates an LRU score for a document.
// Higher score = more likely to be evicted.
func (e *LRUEvictor) getLRUScore(doc *jsbridge.MemoryDocument) float64 {
	createdAt, _ := time.Parse(time.RFC3339, doc.CreatedAt)
	age := time.Since(createdAt).Hours()

	// Higher importance = lower score (keep longer)
	// More accesses = lower score (keep longer)
	// Older age = higher score (evict sooner)
	importanceWeight := float64(11 - doc.Importance)
	accessWeight := 1.0 / float64(doc.AccessCount+1)
	return age + importanceWeight*10 - accessWeight*5
}

// archiveAndDelete archives a memory to compressed storage then deletes it.
func (e *LRUEvictor) archiveAndDelete(db js.Value, doc *jsbridge.MemoryDocument) error {
	// Serialize document
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// Compress
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(data); err != nil {
		gzWriter.Close()
		return fmt.Errorf("failed to compress document: %w", err)
	}
	gzWriter.Close()
	compressed := buf.Bytes()

	// Store in archive
	archivePromise := jsbridge.ArchivePut(db, doc.ID, compressed)
	archiveChan := make(chan struct{}, 1)
	archiveErrChan := make(chan error, 1)

	archivePromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		archiveChan <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		archiveErrChan <- fmt.Errorf("failed to archive: %v", args[0])
		return nil
	}))

	select {
	case <-archiveChan:
		// Archive successful, now delete
	case err := <-archiveErrChan:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout archiving")
	}

	// Delete from main store
	deletePromise := jsbridge.MemoryDelete(db, doc.ID)
	deleteChan := make(chan struct{}, 1)
	deleteErrChan := make(chan error, 1)

	deletePromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		deleteChan <- struct{}{}
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		deleteErrChan <- fmt.Errorf("failed to delete: %v", args[0])
		return nil
	}))

	select {
	case <-deleteChan:
		return nil
	case err := <-deleteErrChan:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout deleting")
	}
}
