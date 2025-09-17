package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Test memory store basic operations
func TestMemoryStoreBasicOperations(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Test Set and Get
	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set value
	setResult := <-store.Set(ctx, key, value, time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	// Get value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Fatal("Value not found")
	}

	if string(getResult.Value) != string(value) {
		t.Errorf("Expected value %s, got %s", string(value), string(getResult.Value))
	}

	// Test TTL
	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Fatalf("Failed to get TTL: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttlResult.TTL)
	}

	// Test Exists
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence: %v", existsResult.Error)
	}
	if !existsResult.Exists {
		t.Error("Key should exist")
	}

	// Test Delete
	delResult := <-store.Del(ctx, key)
	if delResult.Error != nil {
		t.Fatalf("Failed to delete key: %v", delResult.Error)
	}

	// Verify deletion
	getResult = <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value after deletion: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Error("Value should not exist after deletion")
	}

	logx.Info("Memory store basic operations test completed")
}

// Test memory store batch operations
func TestMemoryStoreBatchOperations(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Test MSet and MGet
	ctx := context.Background()
	items := map[string][]byte{
		"batch-key-1": []byte("value-1"),
		"batch-key-2": []byte("value-2"),
		"batch-key-3": []byte("value-3"),
		"batch-key-4": []byte("value-4"),
	}

	// Set batch values
	msetResult := <-store.MSet(ctx, items, time.Minute)
	if msetResult.Error != nil {
		t.Fatalf("Failed to MSet values: %v", msetResult.Error)
	}

	// Get batch values
	keys := []string{"batch-key-1", "batch-key-2", "batch-key-3", "batch-key-4"}
	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Fatalf("Failed to MGet values: %v", mgetResult.Error)
	}

	if len(mgetResult.Values) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(mgetResult.Values))
	}

	for key, expectedValue := range items {
		if retrievedValue, exists := mgetResult.Values[key]; !exists {
			t.Errorf("Key %s not found in retrieved items", key)
		} else if string(retrievedValue) != string(expectedValue) {
			t.Errorf("Expected value %s for key %s, got %s", string(expectedValue), key, string(retrievedValue))
		}
	}

	logx.Info("Memory store batch operations test completed")
}

// Test memory store increment operations
func TestMemoryStoreIncrementOperations(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"

	// Test initial increment
	incrResult := <-store.IncrBy(ctx, key, 5, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to increment: %v", incrResult.Error)
	}
	if incrResult.Result != 5 {
		t.Errorf("Expected value 5, got %d", incrResult.Result)
	}

	// Test subsequent increment
	incrResult = <-store.IncrBy(ctx, key, 3, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to increment: %v", incrResult.Error)
	}
	if incrResult.Result != 8 {
		t.Errorf("Expected value 8, got %d", incrResult.Result)
	}

	// Test negative increment
	incrResult = <-store.IncrBy(ctx, key, -2, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to decrement: %v", incrResult.Error)
	}
	if incrResult.Result != 6 {
		t.Errorf("Expected value 6, got %d", incrResult.Result)
	}

	// Verify final value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get counter value: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Fatal("Counter should exist")
	}

	// The value should be stored as bytes, so we need to check the raw bytes
	expectedBytes := []byte("6")
	if string(getResult.Value) != string(expectedBytes) {
		t.Errorf("Expected counter value %s, got %s", string(expectedBytes), string(getResult.Value))
	}

	logx.Info("Memory store increment operations test completed")
}

// Test memory store complex data structures
func TestMemoryStoreComplexDataStructures(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Test complex data structures
	testCases := []struct {
		key   string
		value []byte
	}{
		{
			key:   "json-data",
			value: []byte(`{"name": "John", "age": 30, "city": "New York"}`),
		},
		{
			key:   "large-data",
			value: []byte(fmt.Sprintf("large-data-%s", string(make([]byte, 1000)))),
		},
		{
			key:   "binary-data",
			value: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
		},
		{
			key:   "unicode-data",
			value: []byte("Hello, 世界! 🌍"),
		},
	}

	for i, tc := range testCases {
		// Set value
		setResult := <-store.Set(context.Background(), tc.key, tc.value, time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set value %d: %v", i, setResult.Error)
		}

		// Get value
		getResult := <-store.Get(context.Background(), tc.key)
		if getResult.Error != nil {
			t.Fatalf("Failed to get value %d: %v", i, getResult.Error)
		}
		if !getResult.Exists {
			t.Fatalf("Value %d not found", i)
		}
		if string(getResult.Value) != string(tc.value) {
			t.Errorf("Expected value %s, got %s", string(tc.value), string(getResult.Value))
		}
	}

	logx.Info("Memory store complex data structures test completed")
}

// Test memory store concurrent operations
func TestMemoryStoreConcurrentOperations(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Perform concurrent operations
	const numGoroutines = 10
	const operationsPerGoroutine = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("concurrent-memory-%d-%d", goroutineID, j)
				value := []byte(fmt.Sprintf("concurrent-value-%d-%d", goroutineID, j))

				// Set value
				setResult := <-store.Set(ctx, key, value, time.Minute)
				if setResult.Error != nil {
					t.Errorf("Goroutine %d: Failed to set value %d: %v", goroutineID, j, setResult.Error)
					return
				}

				// Get value
				getResult := <-store.Get(ctx, key)
				if getResult.Error != nil {
					t.Errorf("Goroutine %d: Failed to get value %d: %v", goroutineID, j, getResult.Error)
					return
				}
				if !getResult.Exists {
					t.Errorf("Goroutine %d: Value %d not found", goroutineID, j)
					return
				}
				if string(getResult.Value) != string(value) {
					t.Errorf("Goroutine %d: Expected value %s, got %s", goroutineID, string(value), string(getResult.Value))
					return
				}

				// Increment counter
				counterKey := fmt.Sprintf("concurrent-counter-%d", goroutineID)
				incrResult := <-store.IncrBy(ctx, counterKey, 1, time.Minute)
				if incrResult.Error != nil {
					t.Errorf("Goroutine %d: Failed to increment counter: %v", goroutineID, incrResult.Error)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete properly
	wg.Wait()

	logx.Info("Memory store concurrent operations test completed")
}

// Test memory store statistics
func TestMemoryStoreStatistics(t *testing.T) {
	// Create memory store with statistics enabled
	config := cachex.DefaultMemoryConfig()
	config.EnableStats = true
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Perform operations to generate statistics
	ctx := context.Background()
	keys := []string{"stats-key-1", "stats-key-2", "stats-key-3"}
	values := [][]byte{[]byte("value-1"), []byte("value-2"), []byte("value-3")}

	// Set values
	for i, key := range keys {
		setResult := <-store.Set(ctx, key, values[i], time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set value %d: %v", i, setResult.Error)
		}
	}

	// Get values (hits)
	for i, key := range keys {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Fatalf("Failed to get value %d: %v", i, getResult.Error)
		}
		if !getResult.Exists {
			t.Fatalf("Value %d not found", i)
		}
		if string(getResult.Value) != string(values[i]) {
			t.Errorf("Expected value %s, got %s", string(values[i]), string(getResult.Value))
		}
	}

	// Get non-existent value (miss)
	getResult := <-store.Get(ctx, "non-existent")
	if getResult.Error != nil {
		t.Fatalf("Failed to get non-existent value: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Error("Non-existent value should not be found")
	}

	// Get statistics
	stats := store.GetStats()
	if stats == nil {
		t.Fatal("Expected statistics, got nil")
	}

	logx.Info("Memory store statistics",
		logx.Int64("hits", stats.Hits),
		logx.Int64("misses", stats.Misses),
		logx.Int64("evictions", stats.Evictions),
		logx.Int64("expirations", stats.Expirations),
		logx.Int64("size", stats.Size),
		logx.Int64("memory_usage", stats.MemoryUsage))

	// Verify statistics are reasonable
	if stats.Hits < 3 {
		t.Errorf("Expected at least 3 hits, got %d", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("Expected at least 1 miss, got %d", stats.Misses)
	}

	logx.Info("Memory store statistics test completed")
}

// Test memory store configurations
func TestMemoryStoreConfigurations(t *testing.T) {
	// Test different configurations
	configs := []*cachex.MemoryConfig{
		cachex.DefaultMemoryConfig(),
		cachex.HighPerformanceMemoryConfig(),
		cachex.ResourceConstrainedMemoryConfig(),
	}

	for i, config := range configs {
		// Create memory store
		store, err := cachex.NewMemoryStore(config)
		if err != nil {
			t.Fatalf("Failed to create memory store with config %d: %v", i, err)
		}

		// Test basic operations
		ctx := context.Background()
		key := fmt.Sprintf("config-test-%d", i)
		value := []byte(fmt.Sprintf("config-value-%d", i))

		setResult := <-store.Set(ctx, key, value, time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set value with config %d: %v", i, setResult.Error)
		}

		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Fatalf("Failed to get value with config %d: %v", i, getResult.Error)
		}
		if !getResult.Exists {
			t.Fatalf("Value not found with config %d", i)
		}

		if string(getResult.Value) != string(value) {
			t.Errorf("Expected value %s with config %d, got %s", string(value), i, string(getResult.Value))
		}

		store.Close()
	}

	logx.Info("Memory store configurations test completed")
}

// Test memory store eviction behavior
func TestMemoryStoreEvictionBehavior(t *testing.T) {
	// Create memory store with small capacity
	config := cachex.DefaultMemoryConfig()
	config.MaxSize = 3
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Fill the cache to capacity
	keys := []string{"evict-1", "evict-2", "evict-3"}
	for i, key := range keys {
		value := []byte(fmt.Sprintf("value-%d", i+1))
		setResult := <-store.Set(context.Background(), key, value, time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set value %d: %v", i, setResult.Error)
		}
	}

	// Verify all keys exist
	for i, key := range keys {
		getResult := <-store.Get(context.Background(), key)
		if getResult.Error != nil {
			t.Fatalf("Failed to get value %d: %v", i, getResult.Error)
		}
		if !getResult.Exists {
			t.Fatalf("Value %d not found", i)
		}
	}

	// Add one more key to trigger eviction
	newKey := "evict-4"
	newValue := []byte("value-4")
	setResult := <-store.Set(context.Background(), newKey, newValue, time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set new value: %v", setResult.Error)
	}

	// Verify new key exists
	getResult := <-store.Get(context.Background(), newKey)
	if getResult.Error != nil {
		t.Fatalf("Failed to get new value: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Fatal("New value not found")
	}

	// Check that at least one old key was evicted
	evictedCount := 0
	for _, key := range keys {
		getResult := <-store.Get(context.Background(), key)
		if getResult.Error != nil {
			t.Fatalf("Failed to check evicted key %s: %v", key, getResult.Error)
		}
		if !getResult.Exists {
			evictedCount++
		}
	}

	if evictedCount == 0 {
		t.Error("Expected at least one key to be evicted")
	}

	logx.Info("Memory store eviction behavior test completed")
}

// Test memory store TTL operations
func TestMemoryStoreTTLOperations(t *testing.T) {
	// Create memory store
	config := cachex.DefaultMemoryConfig()
	config.EvictionPolicy = cachex.EvictionPolicyTTL
	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	// Test TTL operations
	key := "ttl-test"
	value := []byte("ttl-value")

	// Set value with short TTL
	shortTTL := 100 * time.Millisecond
	setResult := <-store.Set(context.Background(), key, value, shortTTL)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value with TTL: %v", setResult.Error)
	}

	// Verify value exists
	getResult := <-store.Get(context.Background(), key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Fatal("Value not found")
	}

	// Check TTL
	ttlResult := <-store.TTL(context.Background(), key)
	if ttlResult.Error != nil {
		t.Fatalf("Failed to get TTL: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttlResult.TTL)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Verify value is expired
	getResult = <-store.Get(context.Background(), key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get expired value: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Error("Value should be expired")
	}

	logx.Info("Memory store TTL operations test completed")
}
