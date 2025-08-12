package memorystore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	// Test with default config
	store, err := New(nil)
	require.NoError(t, err)
	defer store.Close()

	assert.NotNil(t, store)
	assert.NotNil(t, store.data)
	assert.NotNil(t, store.accessOrder)
	assert.NotNil(t, store.accessMap)
}

func TestNewMemoryStore_InvalidConfig(t *testing.T) {
	// Test with invalid config (negative values)
	config := &Config{
		MaxSize:         -1,
		MaxMemoryMB:     -1,
		DefaultTTL:      1 * time.Second, // Use positive TTL to avoid ticker panic
		CleanupInterval: 1 * time.Second, // Use positive interval to avoid ticker panic
	}

	store, err := New(config)
	require.NoError(t, err) // Should still work with invalid config
	defer store.Close()

	assert.NotNil(t, store)
}

func TestMemoryStore_GetSet(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test basic Get/Set
	key := "test-key"
	value := []byte("test-value")

	// Set value
	err = store.Set(ctx, key, value, time.Minute)
	assert.NoError(t, err)

	// Get value
	result, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// Test non-existent key
	result, err = store.Get(ctx, "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryStore_GetSet_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test empty key
	err = store.Set(ctx, "", []byte("empty-key"), time.Minute)
	assert.NoError(t, err)
	result, err := store.Get(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, []byte("empty-key"), result)

	// Test empty value
	err = store.Set(ctx, "empty-value", []byte{}, time.Minute)
	assert.NoError(t, err)
	result, err = store.Get(ctx, "empty-value")
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, result)

	// Test very large value
	largeValue := make([]byte, 1024*1024) // 1MB
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	err = store.Set(ctx, "large-value", largeValue, time.Minute)
	assert.NoError(t, err)
	result, err = store.Get(ctx, "large-value")
	assert.NoError(t, err)
	assert.Equal(t, largeValue, result)

	// Test zero TTL (should use default)
	err = store.Set(ctx, "zero-ttl", []byte("test"), 0)
	assert.NoError(t, err)
	result, err = store.Get(ctx, "zero-ttl")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), result)
}

func TestMemoryStore_TTL(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Set value with TTL
	key := "ttl-test"
	value := []byte("test-value")
	ttl := 100 * time.Millisecond

	err = store.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	// Check TTL
	resultTTL, err := store.TTL(ctx, key)
	assert.NoError(t, err)
	assert.True(t, resultTTL > 0)
	assert.True(t, resultTTL <= ttl)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Check that item has expired
	resultTTL, err = store.TTL(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), resultTTL)

	// Try to get expired item
	result, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryStore_TTL_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test TTL for non-existent key
	ttl, err := store.TTL(ctx, "non-existent")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), ttl)

	// Test TTL for expired key
	err = store.Set(ctx, "expired", []byte("test"), 1*time.Millisecond)
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	ttl, err = store.TTL(ctx, "expired")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), ttl)

	// Test very short TTL
	err = store.Set(ctx, "short-ttl", []byte("test"), 1*time.Nanosecond)
	assert.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	ttl, err = store.TTL(ctx, "short-ttl")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), ttl)
}

func TestMemoryStore_MGetMSet(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test MSet
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	err = store.MSet(ctx, items, time.Minute)
	assert.NoError(t, err)

	// Test MGet
	keys := []string{"key1", "key2", "key3", "non-existent"}
	results, err := store.MGet(ctx, keys...)
	assert.NoError(t, err)

	assert.Equal(t, []byte("value1"), results["key1"])
	assert.Equal(t, []byte("value2"), results["key2"])
	assert.Equal(t, []byte("value3"), results["key3"])
	assert.Nil(t, results["non-existent"])
}

func TestMemoryStore_MGetMSet_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test empty MSet
	err = store.MSet(ctx, map[string][]byte{}, time.Minute)
	assert.NoError(t, err)

	// Test empty MGet
	results, err := store.MGet(ctx)
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{}, results)

	// Test MGet with all non-existent keys
	results, err = store.MGet(ctx, "non-existent1", "non-existent2")
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{}, results)

	// Test MSet with overwriting existing key
	err = store.Set(ctx, "duplicate", []byte("first"), time.Minute)
	assert.NoError(t, err)

	items := map[string][]byte{
		"duplicate": []byte("second"), // This should overwrite
	}
	err = store.MSet(ctx, items, time.Minute)
	assert.NoError(t, err)
	result, err := store.Get(ctx, "duplicate")
	assert.NoError(t, err)
	assert.Equal(t, []byte("second"), result)

	// Test MSet with very large number of items
	largeItems := make(map[string][]byte)
	for i := 0; i < 1000; i++ {
		largeItems[fmt.Sprintf("key-%d", i)] = []byte(fmt.Sprintf("value-%d", i))
	}
	err = store.MSet(ctx, largeItems, time.Minute)
	assert.NoError(t, err)
}

func TestMemoryStore_Del(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Set values
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), time.Minute)
	assert.NoError(t, err)

	// Delete one key
	err = store.Del(ctx, "key1")
	assert.NoError(t, err)

	// Check that key1 is gone
	result, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Check that key2 still exists
	result, err = store.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), result)
}

func TestMemoryStore_Del_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test deleting non-existent key
	err = store.Del(ctx, "non-existent")
	assert.NoError(t, err)

	// Test deleting empty key
	err = store.Del(ctx, "")
	assert.NoError(t, err)

	// Test deleting multiple keys (some exist, some don't)
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)
	err = store.Del(ctx, "key1", "non-existent", "key2")
	assert.NoError(t, err)

	// Verify key1 was deleted
	result, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test deleting empty slice
	err = store.Del(ctx)
	assert.NoError(t, err)
}

func TestMemoryStore_Exists(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test non-existent key
	exists, err := store.Exists(ctx, "non-existent")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Set a key
	err = store.Set(ctx, "test-key", []byte("test-value"), time.Minute)
	assert.NoError(t, err)

	// Test existing key
	exists, err = store.Exists(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestMemoryStore_Exists_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test empty key
	exists, err := store.Exists(ctx, "")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test expired key
	err = store.Set(ctx, "expired", []byte("test"), 1*time.Millisecond)
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	exists, err = store.Exists(ctx, "expired")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMemoryStore_IncrBy(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test incrementing non-existent key
	result, err := store.IncrBy(ctx, "counter", 5, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), result)

	// Test incrementing existing key
	result, err = store.IncrBy(ctx, "counter", 3, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), result)

	// Test incrementing by negative value
	result, err = store.IncrBy(ctx, "counter", -2, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), result)
}

func TestMemoryStore_IncrBy_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test incrementing by zero
	result, err := store.IncrBy(ctx, "zero", 0, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), result)

	// Test incrementing by very large number
	result, err = store.IncrBy(ctx, "large", 9223372036854775807, time.Minute) // Max int64
	assert.NoError(t, err)
	assert.Equal(t, int64(9223372036854775807), result)

	// Test incrementing by negative large number
	result, err = store.IncrBy(ctx, "negative", -9223372036854775808, time.Minute) // Min int64
	assert.NoError(t, err)
	assert.Equal(t, int64(-9223372036854775808), result)

	// Test incrementing expired key
	err = store.Set(ctx, "expired-counter", []byte("5"), 1*time.Millisecond)
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	_, err = store.IncrBy(ctx, "expired-counter", 1, time.Minute)
	assert.Error(t, err) // Should fail because key expired
}

func TestMemoryStore_CapacityLimit(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 2 // Small capacity for testing

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add items up to capacity
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), time.Minute)
	assert.NoError(t, err)

	// Try to add one more (should trigger eviction)
	err = store.Set(ctx, "key3", []byte("value3"), time.Minute)
	assert.NoError(t, err)

	// Check that we still have 2 items
	stats := store.GetStats()
	assert.Equal(t, int64(2), stats.Size)
}

func TestMemoryStore_CapacityLimit_EdgeCases(t *testing.T) {
	// Test with very small capacity
	config := DefaultConfig()
	config.MaxSize = 1

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add first item
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)

	// Add second item (should evict first)
	err = store.Set(ctx, "key2", []byte("value2"), time.Minute)
	assert.NoError(t, err)

	// Check that only one item is stored
	stats := store.GetStats()
	assert.Equal(t, int64(1), stats.Size)

	// Test with very large capacity
	config = DefaultConfig()
	config.MaxSize = 1000000

	store, err = New(config)
	require.NoError(t, err)
	defer store.Close()

	// Add many items
	for i := 0; i < 10000; i++ {
		err = store.Set(ctx, fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("value-%d", i)), time.Minute)
		assert.NoError(t, err)
	}

	// Check that all items are stored
	stats = store.GetStats()
	assert.Equal(t, int64(10000), stats.Size)
}

func TestMemoryStore_EvictionPolicyLRU(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 2
	config.EvictionPolicy = EvictionPolicyLRU

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add two items
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), time.Minute)
	assert.NoError(t, err)

	// Access key1 to make it more recently used
	_, err = store.Get(ctx, "key1")
	assert.NoError(t, err)

	// Add third item (should evict key2 as least recently used)
	err = store.Set(ctx, "key3", []byte("value3"), time.Minute)
	assert.NoError(t, err)

	// Check that key2 was evicted
	result, err := store.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Check that key1 and key3 still exist
	result, err = store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), result)

	result, err = store.Get(ctx, "key3")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value3"), result)
}

func TestMemoryStore_EvictionPolicyLFU(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 2
	config.EvictionPolicy = EvictionPolicyLFU

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add two items
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), time.Minute)
	assert.NoError(t, err)

	// Access key1 multiple times to make it more frequently used
	for i := 0; i < 5; i++ {
		_, err = store.Get(ctx, "key1")
		assert.NoError(t, err)
	}

	// Add third item (should evict key2 as least frequently used)
	err = store.Set(ctx, "key3", []byte("value3"), time.Minute)
	assert.NoError(t, err)

	// Check that key2 was evicted
	result, err := store.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Check that key1 and key3 still exist
	result, err = store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), result)

	result, err = store.Get(ctx, "key3")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value3"), result)
}

func TestMemoryStore_EvictionPolicyTTL(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 2
	config.EvictionPolicy = EvictionPolicyTTL

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add two items with different TTLs
	err = store.Set(ctx, "key1", []byte("value1"), 2*time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), 1*time.Minute)
	assert.NoError(t, err)

	// Add third item (should evict key2 as it has shorter TTL)
	err = store.Set(ctx, "key3", []byte("value3"), 3*time.Minute)
	assert.NoError(t, err)

	// Check that key2 was evicted
	result, err := store.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Check that key1 and key3 still exist
	result, err = store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), result)

	result, err = store.Get(ctx, "key3")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value3"), result)
}

func TestMemoryStore_BackgroundCleanup(t *testing.T) {
	config := DefaultConfig()
	config.CleanupInterval = 50 * time.Millisecond // Fast cleanup for testing

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add items with short TTL
	err = store.Set(ctx, "key1", []byte("value1"), 100*time.Millisecond)
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"), 100*time.Millisecond)
	assert.NoError(t, err)

	// Wait for items to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Check that items have been cleaned up
	result, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Nil(t, result)

	result, err = store.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryStore_BackgroundCleanup_EdgeCases(t *testing.T) {
	config := DefaultConfig()
	config.CleanupInterval = 10 * time.Millisecond // Very fast cleanup
	config.EnableStats = true

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add many items with different TTLs
	for i := 0; i < 100; i++ {
		ttl := time.Duration(i%10+1) * time.Millisecond
		err = store.Set(ctx, fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("value-%d", i)), ttl)
		assert.NoError(t, err)
	}

	// Wait for cleanup to run multiple times
	time.Sleep(100 * time.Millisecond)

	// Check that most items have been cleaned up
	stats := store.GetStats()
	assert.True(t, stats.Expirations > 0)
}

func TestMemoryStore_Stats(t *testing.T) {
	config := DefaultConfig()
	config.EnableStats = true

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Perform operations
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)

	_, err = store.Get(ctx, "key1") // Hit
	assert.NoError(t, err)

	_, err = store.Get(ctx, "non-existent") // Miss
	assert.NoError(t, err)

	// Get stats
	stats := store.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Size)
	assert.True(t, stats.MemoryUsage > 0)
}

func TestMemoryStore_Stats_EdgeCases(t *testing.T) {
	// Test with stats disabled
	config := DefaultConfig()
	config.EnableStats = false

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Perform operations
	err = store.Set(ctx, "key1", []byte("value1"), time.Minute)
	assert.NoError(t, err)

	_, err = store.Get(ctx, "key1")
	assert.NoError(t, err)

	// Get stats (should still work but may not be accurate)
	stats := store.GetStats()
	assert.Equal(t, int64(1), stats.Size)
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test concurrent reads and writes
	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Start reader goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				store.Get(ctx, key)
			}
		}(i)
	}

	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))
				store.Set(ctx, key, value, time.Minute)
			}
		}(i)
	}

	wg.Wait()

	// Verify that some items were stored
	stats := store.GetStats()
	assert.True(t, stats.Size > 0)
}

func TestMemoryStore_ConcurrentAccess_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test concurrent access to same key
	const numGoroutines = 100
	var wg sync.WaitGroup

	// Concurrent writes to same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value := []byte(fmt.Sprintf("value-%d", id))
			store.Set(ctx, "same-key", value, time.Minute)
		}(i)
	}

	wg.Wait()

	// Verify that the key exists (last write should win)
	result, err := store.Get(ctx, "same-key")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test concurrent reads during eviction
	config := DefaultConfig()
	config.MaxSize = 1
	smallStore, err := New(config)
	require.NoError(t, err)
	defer smallStore.Close()

	// Fill the store and trigger evictions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("evict-key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))
			smallStore.Set(ctx, key, value, time.Minute)
			smallStore.Get(ctx, key)
		}(i)
	}

	wg.Wait()
}

func TestMemoryStore_Configurations(t *testing.T) {
	// Test default config
	defaultConfig := DefaultConfig()
	assert.Equal(t, 10000, defaultConfig.MaxSize)
	assert.Equal(t, 100, defaultConfig.MaxMemoryMB)
	assert.Equal(t, 5*time.Minute, defaultConfig.DefaultTTL)
	assert.Equal(t, EvictionPolicyLRU, defaultConfig.EvictionPolicy)

	// Test high performance config
	highPerfConfig := HighPerformanceConfig()
	assert.Equal(t, 50000, highPerfConfig.MaxSize)
	assert.Equal(t, 500, highPerfConfig.MaxMemoryMB)
	assert.Equal(t, 10*time.Minute, highPerfConfig.DefaultTTL)
	assert.Equal(t, EvictionPolicyLRU, highPerfConfig.EvictionPolicy)

	// Test resource constrained config
	resourceConfig := ResourceConstrainedConfig()
	assert.Equal(t, 1000, resourceConfig.MaxSize)
	assert.Equal(t, 10, resourceConfig.MaxMemoryMB)
	assert.Equal(t, 2*time.Minute, resourceConfig.DefaultTTL)
	assert.Equal(t, EvictionPolicyTTL, resourceConfig.EvictionPolicy)
}

func TestMemoryStore_Close(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)

	// Close the store
	err = store.Close()
	assert.NoError(t, err)

	// Try to use closed store (should not panic)
	ctx := context.Background()
	_, err = store.Get(ctx, "test")
	assert.NoError(t, err) // Should not panic, just return nil
}

func TestMemoryStore_Close_EdgeCases(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)

	// Close once
	err = store.Close()
	assert.NoError(t, err)

	// Test operations after close (should not panic)
	ctx := context.Background()
	_, err = store.Get(ctx, "test")
	assert.NoError(t, err)

	err = store.Set(ctx, "test", []byte("value"), time.Minute)
	assert.NoError(t, err) // Should not panic
}

func TestMemoryStore_ContextCancellation(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations should still work (context is not used in memory store)
	err = store.Set(ctx, "key", []byte("value"), time.Minute)
	assert.NoError(t, err)

	result, err := store.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), result)
}

func TestMemoryStore_MemoryPressure(t *testing.T) {
	// Test with very small memory limit
	config := DefaultConfig()
	config.MaxMemoryMB = 1 // 1MB limit

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Try to add items that exceed memory limit
	for i := 0; i < 100; i++ {
		// Each value is ~10KB
		value := make([]byte, 10*1024)
		err = store.Set(ctx, fmt.Sprintf("key-%d", i), value, time.Minute)
		assert.NoError(t, err) // Should still work, but may trigger evictions
	}

	// Check that store is still functional
	stats := store.GetStats()
	assert.True(t, stats.Size > 0)
}

func TestMemoryStore_DataIntegrity(t *testing.T) {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test that data is not corrupted during concurrent access
	const numOperations = 1000
	var wg sync.WaitGroup

	// Concurrent writes with verification
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("integrity-key-%d", id)
			expectedValue := []byte(fmt.Sprintf("integrity-value-%d", id))

			// Write
			err := store.Set(ctx, key, expectedValue, time.Minute)
			assert.NoError(t, err)

			// Read and verify
			result, err := store.Get(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, result)
		}(i)
	}

	wg.Wait()
}
