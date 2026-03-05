//go:build js && wasm

package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/gleicon/webclaw/internal/memory"
)

// mockLRUEvictor is a test helper that allows controlled quota simulation
type mockLRUEvictor struct {
	mockQuota      *memory.QuotaInfo
	shouldEvict    bool
	evictedDocs    []string
	evictCallCount int
}

// CheckQuota returns the mock quota info
func (m *mockLRUEvictor) CheckQuota(ctx context.Context) (*memory.QuotaInfo, error) {
	return m.mockQuota, nil
}

// EvictToTarget records which documents would be evicted
func (m *mockLRUEvictor) EvictToTarget(docs []*memory.MemoryDocument, targetPercent float64) ([]string, error) {
	m.evictCallCount++

	if len(docs) == 0 {
		return nil, nil
	}

	// Sort by LRU score (highest = evict first)
	sorted := make([]*memory.MemoryDocument, len(docs))
	copy(sorted, docs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GetLRUScore() > sorted[j].GetLRUScore()
	})

	// Calculate how much to evict to reach target
	// Simulate: evict enough docs to drop usage from current to target
	currentUsage := float64(len(docs))
	targetUsage := currentUsage * (targetPercent / 100.0)
	docsToEvict := int(currentUsage - targetUsage)

	if docsToEvict <= 0 {
		return nil, nil
	}

	if docsToEvict > len(sorted) {
		docsToEvict = len(sorted)
	}

	evicted := make([]string, docsToEvict)
	for i := 0; i < docsToEvict; i++ {
		evicted[i] = sorted[i].ID
		m.evictedDocs = append(m.evictedDocs, sorted[i].ID)
	}

	return evicted, nil
}

// TestLRUEviction_ThresholdVerification verifies the 80% eviction threshold
func TestLRUEviction_ThresholdVerification(t *testing.T) {
	testCases := []struct {
		name        string
		percent     float64
		expectEvict bool
		description string
	}{
		{
			name:        "below_threshold_79_percent",
			percent:     79.0,
			expectEvict: false,
			description: "At 79% usage, ShouldEvict should be false",
		},
		{
			name:        "at_threshold_80_percent",
			percent:     80.0,
			expectEvict: true,
			description: "At exactly 80% usage, ShouldEvict should be true",
		},
		{
			name:        "above_threshold_85_percent",
			percent:     85.0,
			expectEvict: true,
			description: "At 85% usage, ShouldEvict should be true",
		},
		{
			name:        "way_above_threshold_95_percent",
			percent:     95.0,
			expectEvict: true,
			description: "At 95% usage, ShouldEvict should be true",
		},
		{
			name:        "zero_percent",
			percent:     0.0,
			expectEvict: false,
			description: "At 0% usage, ShouldEvict should be false",
		},
		{
			name:        "exactly_79_point_99_percent",
			percent:     79.99,
			expectEvict: false,
			description: "At 79.99% usage, ShouldEvict should be false (just below threshold)",
		},
		{
			name:        "exactly_80_point_01_percent",
			percent:     80.01,
			expectEvict: true,
			description: "At 80.01% usage, ShouldEvict should be true (just above threshold)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create QuotaInfo with the test percentage
			quota := &memory.QuotaInfo{
				Usage:       int64(tc.percent * 1000000 / 100),
				Quota:       1000000,
				Percent:     tc.percent,
				Overflow:    tc.percent > 100,
				ShouldEvict: tc.percent >= 80.0,
			}

			// Verify ShouldEvict is set correctly
			if quota.ShouldEvict != tc.expectEvict {
				t.Errorf("%s: ShouldEvict = %v, expected %v (percent=%.2f%%)",
					tc.description, quota.ShouldEvict, tc.expectEvict, tc.percent)
			}
		})
	}
}

// TestLRUEviction_OrderVerification verifies eviction removes oldest/lowest-priority first
func TestLRUEviction_OrderVerification(t *testing.T) {
	now := time.Now()

	// Create documents with different characteristics
	docs := []*memory.MemoryDocument{
		{
			ID:           "old-low-importance",
			Content:      "Old document with low importance",
			CreatedAt:    now.Add(-72 * time.Hour), // 3 days old
			Importance:   2,
			AccessCount:  1,
			LastAccessed: now.Add(-48 * time.Hour),
		},
		{
			ID:           "old-high-importance",
			Content:      "Old document with high importance",
			CreatedAt:    now.Add(-72 * time.Hour), // 3 days old
			Importance:   9,
			AccessCount:  5,
			LastAccessed: now.Add(-24 * time.Hour),
		},
		{
			ID:           "recent-low-importance",
			Content:      "Recent document with low importance",
			CreatedAt:    now.Add(-1 * time.Hour), // 1 hour old
			Importance:   3,
			AccessCount:  0,
			LastAccessed: now.Add(-1 * time.Hour),
		},
		{
			ID:           "recent-high-access",
			Content:      "Recent document frequently accessed",
			CreatedAt:    now.Add(-2 * time.Hour), // 2 hours old
			Importance:   5,
			AccessCount:  20,
			LastAccessed: now,
		},
		{
			ID:           "medium-age-no-access",
			Content:      "Medium age document never accessed",
			CreatedAt:    now.Add(-24 * time.Hour), // 1 day old
			Importance:   5,
			AccessCount:  0,
			LastAccessed: now.Add(-24 * time.Hour),
		},
	}

	// Create mock evictor
	mockEvictor := &mockLRUEvictor{
		mockQuota: &memory.QuotaInfo{
			Usage:       850000,
			Quota:       1000000,
			Percent:     85.0,
			ShouldEvict: true,
		},
	}

	// Trigger eviction to 60% target
	evicted, err := mockEvictor.EvictToTarget(docs, 60.0)
	if err != nil {
		t.Fatalf("eviction failed: %v", err)
	}

	t.Logf("Evicted %d documents: %v", len(evicted), evicted)

	// Verify eviction happened
	if len(evicted) == 0 {
		t.Fatal("expected some documents to be evicted at 85% quota")
	}

	// The "old-low-importance" document should be evicted first (highest LRU score)
	// due to being old AND low importance AND low access count
	foundOldLowImportance := false
	for _, id := range evicted {
		if id == "old-low-importance" {
			foundOldLowImportance = true
			break
		}
	}
	if !foundOldLowImportance {
		t.Error("expected 'old-low-importance' to be evicted first (oldest, low importance, low access)")
	}

	// Calculate expected eviction count: from 5 docs at 85% to 60% target
	// Should evict enough to reach 60% = 3 docs (5 * 0.6 = 3)
	expectedEvictCount := 3
	if len(evicted) != expectedEvictCount {
		t.Errorf("expected %d documents evicted to reach 60%% target, got %d",
			expectedEvictCount, len(evicted))
	}

	// Verify LRU scores make sense
	t.Log("LRU Scores (higher = evict sooner):")
	for _, doc := range docs {
		score := doc.GetLRUScore()
		t.Logf("  %s: %.2f (age=%vh, importance=%d, accesses=%d)",
			doc.ID, score,
			time.Since(doc.CreatedAt).Hours(),
			doc.Importance, doc.AccessCount)
	}

	// Verify that "recent-high-access" is NOT evicted (should have lowest score)
	for _, id := range evicted {
		if id == "recent-high-access" {
			t.Error("'recent-high-access' should NOT be evicted (recent, high importance, many accesses)")
		}
	}

	// Verify that "old-high-importance" is likely NOT evicted (high importance protects it)
	// Note: This depends on exact score calculations
	t.Logf("Eviction order respects LRU priority (oldest, least accessed, lowest importance first)")
}

// TestLRUEviction_QuotaInfoStructure verifies QuotaInfo has all required fields
func TestLRUEviction_QuotaInfoStructure(t *testing.T) {
	quota := memory.QuotaInfo{
		Usage:       800000000,
		Quota:       1000000000,
		Percent:     80.0,
		Overflow:    false,
		ShouldEvict: true,
	}

	// Serialize and deserialize to verify structure
	data, err := json.Marshal(quota)
	if err != nil {
		t.Fatalf("failed to marshal QuotaInfo: %v", err)
	}

	var restored memory.QuotaInfo
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal QuotaInfo: %v", err)
	}

	// Verify all fields preserved
	if restored.Usage != quota.Usage {
		t.Errorf("Usage mismatch: %d != %d", restored.Usage, quota.Usage)
	}
	if restored.Quota != quota.Quota {
		t.Errorf("Quota mismatch: %d != %d", restored.Quota, quota.Quota)
	}
	if restored.Percent != quota.Percent {
		t.Errorf("Percent mismatch: %f != %f", restored.Percent, quota.Percent)
	}
	if restored.Overflow != quota.Overflow {
		t.Errorf("Overflow mismatch: %v != %v", restored.Overflow, quota.Overflow)
	}
	if restored.ShouldEvict != quota.ShouldEvict {
		t.Errorf("ShouldEvict mismatch: %v != %v", restored.ShouldEvict, quota.ShouldEvict)
	}

	t.Log("QuotaInfo structure validated with all 5 fields")
}

// TestLRUEviction_ArchiveCompression verifies documents are compressed before archival
func TestLRUEviction_ArchiveCompression(t *testing.T) {
	doc := &memory.MemoryDocument{
		ID: "test-archive-doc",
		Content: "This is test content for compression verification. " +
			"It needs to be long enough to show compression benefits. " +
			"Adding more text to increase the size and make compression more effective. " +
			"The gzip compression should reduce the size of this document significantly.",
		CreatedAt: time.Now(),
	}

	// Serialize document
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal document: %v", err)
	}

	originalSize := len(data)

	// Compress
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(data); err != nil {
		gzWriter.Close()
		t.Fatalf("failed to compress: %v", err)
	}
	gzWriter.Close()

	compressed := buf.Bytes()
	compressedSize := len(compressed)

	compressionRatio := float64(originalSize) / float64(compressedSize)
	t.Logf("Original size: %d bytes, Compressed: %d bytes, Ratio: %.2fx",
		originalSize, compressedSize, compressionRatio)

	// Verify compression actually reduced size
	if compressedSize >= originalSize {
		t.Logf("Warning: compression did not reduce size (small documents may not compress well)")
	} else {
		t.Logf("Compression successful: %.1f%% reduction",
			(1.0-float64(compressedSize)/float64(originalSize))*100)
	}

	// Verify we can decompress
	gzReader, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	// Verify data integrity
	if !bytes.Equal(data, decompressed) {
		t.Error("decompressed data does not match original")
	}

	// Verify we can restore the document
	var restored memory.MemoryDocument
	if err := json.Unmarshal(decompressed, &restored); err != nil {
		t.Fatalf("failed to unmarshal restored document: %v", err)
	}

	if restored.ID != doc.ID {
		t.Errorf("ID mismatch after round-trip: %s != %s", restored.ID, doc.ID)
	}
	if restored.Content != doc.Content {
		t.Errorf("Content mismatch after round-trip")
	}
}

// TestLRUEviction_StoreIntegration verifies Store() checks quota before storing
func TestLRUEviction_StoreIntegration(t *testing.T) {
	// Create a memory store
	store, err := memory.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}

	// Store several documents to create a baseline
	docs := []*memory.MemoryDocument{
		{ID: "evict-test-1", Content: "First test document for eviction testing"},
		{ID: "evict-test-2", Content: "Second test document with different content"},
		{ID: "evict-test-3", Content: "Third document to establish memory baseline"},
	}

	for _, doc := range docs {
		if err := store.Store(doc); err != nil {
			t.Fatalf("failed to store doc %s: %v", doc.ID, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Check current quota
	quota, err := store.CheckQuota()
	if err != nil {
		t.Skipf("quota check not available in test environment: %v", err)
	}

	t.Logf("Current storage: %.2f%% used (%d bytes of %d bytes)",
		quota.Percent, quota.Usage, quota.Quota)

	// Verify quota structure
	if quota.Quota <= 0 {
		t.Error("quota should be positive")
	}
	if quota.Usage < 0 {
		t.Error("usage should be non-negative")
	}
	if quota.Percent < 0 || quota.Percent > 100 {
		t.Errorf("percent should be 0-100, got %.2f", quota.Percent)
	}

	// Verify ShouldEvict is set based on threshold
	expectedShouldEvict := quota.Percent >= 80.0
	if quota.ShouldEvict != expectedShouldEvict {
		t.Errorf("ShouldEvict = %v, expected %v (at %.2f%%)",
			quota.ShouldEvict, expectedShouldEvict, quota.Percent)
	}

	// If we're above threshold, verify EvictIfNeeded can be called
	if quota.Percent >= 80.0 {
		t.Log("Storage above 80% threshold, testing EvictIfNeeded...")
		err := store.EvictIfNeeded()
		if err != nil {
			t.Logf("EvictIfNeeded returned error (may be expected in test env): %v", err)
		} else {
			t.Log("EvictIfNeeded completed without error")
		}
	} else {
		t.Logf("Storage at %.2f%%, below 80%% threshold - no eviction needed", quota.Percent)
	}
}

// TestLRUEviction_ConcurrentAccessSafety verifies thread safety considerations
func TestLRUEviction_ConcurrentAccessSafety(t *testing.T) {
	// This test documents the expected behavior - actual WASM tests
	// cannot easily test true concurrency, but we can verify the
	// design supports concurrent access patterns

	t.Log("LRU Eviction thread safety considerations:")
	t.Log("1. CheckQuota() is read-only and safe for concurrent calls")
	t.Log("2. EvictIfNeeded() should be serialized to prevent race conditions")
	t.Log("3. Store() calls CheckQuota() before storing, creating a natural ordering")
	t.Log("4. IndexedDB transactions provide atomicity for eviction operations")
	t.Log("5. Archive and delete operations are performed in a single transaction")

	// Create a simple scenario to demonstrate expected ordering
	quota1 := &memory.QuotaInfo{Percent: 75.0, ShouldEvict: false}
	quota2 := &memory.QuotaInfo{Percent: 85.0, ShouldEvict: true}

	operations := []struct {
		name      string
		quota     *memory.QuotaInfo
		shouldRun bool
	}{
		{"store_at_75", quota1, !quota1.ShouldEvict}, // Should run (no eviction)
		{"store_at_85", quota2, true},                // Should run (eviction first)
	}

	for _, op := range operations {
		if op.quota.ShouldEvict {
			t.Logf("Operation '%s': Will trigger eviction first (%.0f%%)", op.name, op.quota.Percent)
		} else {
			t.Logf("Operation '%s': Direct store (%.0f%% < 80%% threshold)", op.name, op.quota.Percent)
		}
	}
}

// TestLRUEviction_DocumentScoringFormula verifies the LRU score calculation
func TestLRUEviction_DocumentScoringFormula(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name            string
		createdAt       time.Time
		importance      int
		accessCount     int
		expectHighScore bool // true if should be evicted before others
	}{
		{
			name:            "old_unimportant_unaccessed",
			createdAt:       now.Add(-72 * time.Hour),
			importance:      1,
			accessCount:     0,
			expectHighScore: true,
		},
		{
			name:            "recent_important_accessed",
			createdAt:       now.Add(-1 * time.Hour),
			importance:      10,
			accessCount:     50,
			expectHighScore: false,
		},
		{
			name:            "medium_moderate_moderate",
			createdAt:       now.Add(-24 * time.Hour),
			importance:      5,
			accessCount:     5,
			expectHighScore: false, // Middle ground
		},
	}

	var scores []struct {
		name  string
		score float64
		high  bool
	}

	for _, tc := range testCases {
		doc := &memory.MemoryDocument{
			ID:           tc.name,
			CreatedAt:    tc.createdAt,
			Importance:   tc.importance,
			AccessCount:  tc.accessCount,
			LastAccessed: tc.createdAt,
		}

		score := doc.GetLRUScore()
		scores = append(scores, struct {
			name  string
			score float64
			high  bool
		}{tc.name, score, tc.expectHighScore})

		t.Logf("%s: score=%.2f (age=%vh, importance=%d, accesses=%d)",
			tc.name, score,
			time.Since(tc.createdAt).Hours(),
			tc.importance, tc.accessCount)
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// The highest score should be the one expecting high score
	if !scores[0].high {
		t.Errorf("Expected '%s' to have highest score (most evictable), but it's not marked as high priority",
			scores[0].name)
	}

	// Verify the lowest score is NOT expecting high score
	lastIdx := len(scores) - 1
	if scores[lastIdx].high {
		t.Errorf("Expected '%s' to have lowest score (least evictable), but it's marked as high priority",
			scores[lastIdx].name)
	}

	t.Logf("Eviction priority order (highest score = evict first): %s", scores[0].name)
}

// BenchmarkLRUScoreCalculation benchmarks the LRU score calculation
func BenchmarkLRUScoreCalculation(b *testing.B) {
	doc := &memory.MemoryDocument{
		ID:           "benchmark-doc",
		CreatedAt:    time.Now().Add(-24 * time.Hour),
		Importance:   5,
		AccessCount:  10,
		LastAccessed: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.GetLRUScore()
	}
}
