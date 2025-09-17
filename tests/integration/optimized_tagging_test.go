package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// TestOptimizedTagManager_Integration tests the optimized tag manager with real stores
func TestOptimizedTagManager_Integration(t *testing.T) {
	// Test with memory store
	t.Run("MemoryStore", func(t *testing.T) {
		store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		})
		if err != nil {
			t.Fatalf("Failed to create memory store: %v", err)
		}
		defer store.Close()

		testOptimizedTagManager(t, store, "MemoryStore")
	})

	// Test with Redis store if available
	t.Run("RedisStore", func(t *testing.T) {
		store := createRedisStoreForTagging(t)
		if store == nil {
			t.Skip("Redis not available, skipping test")
		}
		defer store.Close()

		testOptimizedTagManager(t, store, "RedisStore")
	})
}

func testOptimizedTagManager(t *testing.T, store cachex.Store, storeName string) {
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false, // Disable for testing
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    true,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager with %s: %v", storeName, err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Test basic operations
	t.Run("BasicOperations", func(t *testing.T) {
		// Add tags
		err := tm.AddTags(ctx, "basic-key1", "basic-tag1", "basic-tag2", "basic-tag3")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		err = tm.AddTags(ctx, "basic-key2", "basic-tag2", "basic-tag3", "basic-tag4")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		// Get tags by key
		tags, err := tm.GetTagsByKey(ctx, "basic-key1")
		if err != nil {
			t.Errorf("GetTagsByKey() failed: %v", err)
		}
		if len(tags) != 3 {
			t.Errorf("Expected 3 tags for basic-key1, got %d", len(tags))
		}

		// Get keys by tag
		keys, err := tm.GetKeysByTag(ctx, "basic-tag2")
		if err != nil {
			t.Errorf("GetKeysByTag() failed: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("Expected 2 keys for basic-tag2, got %d", len(keys))
		}

		// Remove tags
		err = tm.RemoveTags(ctx, "basic-key1", "basic-tag1")
		if err != nil {
			t.Errorf("RemoveTags() failed: %v", err)
		}

		tags, err = tm.GetTagsByKey(ctx, "basic-key1")
		if err != nil {
			t.Errorf("GetTagsByKey() failed: %v", err)
		}
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags for basic-key1 after removal, got %d", len(tags))
		}

		// Remove key
		err = tm.RemoveKey(ctx, "basic-key1")
		if err != nil {
			t.Errorf("RemoveKey() failed: %v", err)
		}

		tags, err = tm.GetTagsByKey(ctx, "basic-key1")
		if err != nil {
			t.Errorf("GetTagsByKey() failed: %v", err)
		}
		if len(tags) != 0 {
			t.Errorf("Expected 0 tags for basic-key1 after removal, got %d", len(tags))
		}
	})

	// Test three-tier batch processing
	t.Run("ThreeTierBatchProcessing", func(t *testing.T) {
		// Very small operation (≤5 tags) - should use direct processing
		err := tm.AddTags(ctx, "small-key", "tag1", "tag2", "tag3")
		if err != nil {
			t.Errorf("AddTags() for small operation failed: %v", err)
		}

		// Medium operation (≤batch size) - should use batch buffer
		mediumTags := make([]string, 50)
		for i := 0; i < 50; i++ {
			mediumTags[i] = fmt.Sprintf("tag-%d", i)
		}
		err = tm.AddTags(ctx, "medium-key", mediumTags...)
		if err != nil {
			t.Errorf("AddTags() for medium operation failed: %v", err)
		}

		// Large operation (>batch size) - should use direct processing
		largeTags := make([]string, 150)
		for i := 0; i < 150; i++ {
			largeTags[i] = fmt.Sprintf("large-tag-%d", i)
		}
		err = tm.AddTags(ctx, "large-key", largeTags...)
		if err != nil {
			t.Errorf("AddTags() for large operation failed: %v", err)
		}

		// Verify all operations worked
		smallTags, err := tm.GetTagsByKey(ctx, "small-key")
		if err != nil {
			t.Errorf("GetTagsByKey() for small operation failed: %v", err)
		}
		if len(smallTags) != 3 {
			t.Errorf("Expected 3 tags for small operation, got %d", len(smallTags))
		}

		mediumResultTags, err := tm.GetTagsByKey(ctx, "medium-key")
		if err != nil {
			t.Errorf("GetTagsByKey() for medium operation failed: %v", err)
		}
		if len(mediumResultTags) != 50 {
			t.Errorf("Expected 50 tags for medium operation, got %d", len(mediumResultTags))
		}

		largeResultTags, err := tm.GetTagsByKey(ctx, "large-key")
		if err != nil {
			t.Errorf("GetTagsByKey() for large operation failed: %v", err)
		}
		if len(largeResultTags) != 150 {
			t.Errorf("Expected 150 tags for large operation, got %d", len(largeResultTags))
		}
	})

	// Test memory usage tracking
	t.Run("MemoryUsageTracking", func(t *testing.T) {
		stats, err := tm.GetStats()
		if err != nil {
			t.Errorf("GetStats() failed: %v", err)
			return
		}

		if stats.MemoryUsage <= 0 {
			t.Errorf("Memory usage should be tracked, got %d", stats.MemoryUsage)
		}

		if stats.TotalKeys <= 0 {
			t.Errorf("Total keys should be tracked, got %d", stats.TotalKeys)
		}

		if stats.TotalTags <= 0 {
			t.Errorf("Total tags should be tracked, got %d", stats.TotalTags)
		}

		logx.Info("Optimized tag manager stats",
			logx.Int64("memory_usage", stats.MemoryUsage),
			logx.Int64("total_keys", stats.TotalKeys),
			logx.Int64("total_tags", stats.TotalTags),
			logx.Int64("add_tags_count", stats.AddTagsCount),
			logx.Int64("remove_tags_count", stats.RemoveTagsCount),
			logx.Int64("batch_operations", stats.BatchOperations))
	})

	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		// Add some data
		err := tm.AddTags(ctx, "persist-key1", "persist-tag1", "persist-tag2")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		err = tm.AddTags(ctx, "persist-key2", "persist-tag2", "persist-tag3")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		// Verify data is accessible
		tags, err := tm.GetTagsByKey(ctx, "persist-key1")
		if err != nil {
			t.Errorf("GetTagsByKey() failed: %v", err)
		}
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags for persist-key1, got %d", len(tags))
		}

		keys, err := tm.GetKeysByTag(ctx, "persist-tag2")
		if err != nil {
			t.Errorf("GetKeysByTag() failed: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("Expected 2 keys for persist-tag2, got %d", len(keys))
		}
	})

	// Test invalidation
	t.Run("Invalidation", func(t *testing.T) {
		// Add some data to store
		setResult := <-store.Set(ctx, "invalidate-key1", []byte("value1"), 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Store.Set() failed: %v", setResult.Error)
		}

		setResult = <-store.Set(ctx, "invalidate-key2", []byte("value2"), 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Store.Set() failed: %v", setResult.Error)
		}

		// Add tags to keys
		err := tm.AddTags(ctx, "invalidate-key1", "invalidate-tag")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		err = tm.AddTags(ctx, "invalidate-key2", "invalidate-tag")
		if err != nil {
			t.Errorf("AddTags() failed: %v", err)
		}

		// Invalidate by tag
		err = tm.InvalidateByTag(ctx, "invalidate-tag")
		if err != nil {
			t.Errorf("InvalidateByTag() failed: %v", err)
		}

		// Verify keys were invalidated
		keys, err := tm.GetKeysByTag(ctx, "invalidate-tag")
		if err != nil {
			t.Errorf("GetKeysByTag() failed: %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("Expected 0 keys after invalidation, got %d", len(keys))
		}
	})

	logx.Info("Optimized tag manager integration test completed successfully")
}

// TestOptimizedTagManager_Performance tests performance characteristics
func TestOptimizedTagManager_Performance(t *testing.T) {
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    true,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Performance test for AddTags
	t.Run("AddTagsPerformance", func(t *testing.T) {
		start := time.Now()
		numOperations := 1000

		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("perf-key-%d", i)
			tags := []string{fmt.Sprintf("perf-tag-%d", i%10)}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				t.Errorf("AddTags() failed: %v", err)
			}
		}

		duration := time.Since(start)
		opsPerSecond := float64(numOperations) / duration.Seconds()

		logx.Info("AddTags performance test completed")

		if opsPerSecond < 1000 {
			t.Errorf("AddTags performance too low: %.2f ops/sec", opsPerSecond)
		}
	})

	// Performance test for GetKeysByTag
	t.Run("GetKeysByTagPerformance", func(t *testing.T) {
		start := time.Now()
		numOperations := 1000

		for i := 0; i < numOperations; i++ {
			tag := fmt.Sprintf("perf-tag-%d", i%10)
			_, err := tm.GetKeysByTag(ctx, tag)
			if err != nil {
				t.Errorf("GetKeysByTag() failed: %v", err)
			}
		}

		duration := time.Since(start)
		opsPerSecond := float64(numOperations) / duration.Seconds()

		logx.Info("GetKeysByTag performance test completed")

		if opsPerSecond < 5000 {
			t.Errorf("GetKeysByTag performance too low: %.2f ops/sec", opsPerSecond)
		}
	})

	// Performance test for GetTagsByKey
	t.Run("GetTagsByKeyPerformance", func(t *testing.T) {
		start := time.Now()
		numOperations := 1000

		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("perf-key-%d", i%100)
			_, err := tm.GetTagsByKey(ctx, key)
			if err != nil {
				t.Errorf("GetTagsByKey() failed: %v", err)
			}
		}

		duration := time.Since(start)
		opsPerSecond := float64(numOperations) / duration.Seconds()

		logx.Info("GetTagsByKey performance test completed")

		if opsPerSecond < 5000 {
			t.Errorf("GetTagsByKey performance too low: %.2f ops/sec", opsPerSecond)
		}
	})
}
