package unit

import (
	"fmt"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

func TestCacheItemPool(t *testing.T) {
	pool := cachex.NewCacheItemPool()
	if pool == nil {
		t.Fatal("NewCacheItemPool returned nil")
	}

	// Test normal Get/Put cycle
	item := pool.Get()
	if item == nil {
		t.Fatal("Get returned nil item")
	}
	if item.Value != nil {
		t.Error("New item should have nil Value")
	}
	if !item.ExpiresAt.IsZero() {
		t.Error("New item should have zero ExpiresAt")
	}
	if item.AccessCount != 0 {
		t.Error("New item should have zero AccessCount")
	}
	if !item.LastAccess.IsZero() {
		t.Error("New item should have zero LastAccess")
	}
	if item.Size != 0 {
		t.Error("New item should have zero Size")
	}

	// Modify the item
	item.Value = []byte("test")
	item.ExpiresAt = time.Now().Add(time.Hour)
	item.AccessCount = 5
	item.LastAccess = time.Now()
	item.Size = 100

	// Put it back
	pool.Put(item)

	// Get it again and verify it was reset
	item2 := pool.Get()
	if item2.Value != nil {
		t.Error("Item should be reset after Put")
	}
	if !item2.ExpiresAt.IsZero() {
		t.Error("Item should be reset after Put")
	}
	if item2.AccessCount != 0 {
		t.Error("Item should be reset after Put")
	}
	if !item2.LastAccess.IsZero() {
		t.Error("Item should be reset after Put")
	}
	if item2.Size != 0 {
		t.Error("Item should be reset after Put")
	}
}

func TestCacheItemPoolNilHandling(t *testing.T) {
	pool := cachex.NewCacheItemPool()

	// Test Get with nil pool (should not panic)
	var nilPool *cachex.CacheItemPool
	item := nilPool.Get()
	if item == nil {
		t.Fatal("Get should return a new item even with nil pool")
	}

	// Test Put with nil pool (should not panic)
	nilPool.Put(nil)
	// Note: We can't test with cacheItem directly as it's unexported

	// Test Put with nil item
	pool.Put(nil) // Should not panic
}

func TestAsyncResultPool(t *testing.T) {
	pool := cachex.NewAsyncResultPool()
	if pool == nil {
		t.Fatal("NewAsyncResultPool returned nil")
	}

	// Test normal Get/Put cycle
	result := pool.Get()
	if result == nil {
		t.Fatal("Get returned nil result")
	}
	if result.Result != 0 {
		t.Error("New result should have zero Result")
	}
	if result.Values != nil {
		t.Error("New result should have nil Values")
	}
	if result.Error != nil {
		t.Error("New result should have nil Error")
	}

	// Modify the result
	result.Result = 42
	result.Values = map[string][]byte{"key": []byte("value")}
	result.Error = fmt.Errorf("test error")

	// Put it back
	pool.Put(result)

	// Get it again and verify it was reset
	result2 := pool.Get()
	if result2.Result != 0 {
		t.Error("Result should be reset after Put")
	}
	if result2.Values != nil {
		t.Error("Result should be reset after Put")
	}
	if result2.Error != nil {
		t.Error("Result should be reset after Put")
	}
}

func TestAsyncResultPoolNilHandling(t *testing.T) {
	pool := cachex.NewAsyncResultPool()

	// Test Get with nil pool (should not panic)
	var nilPool *cachex.AsyncResultPool
	result := nilPool.Get()
	if result == nil {
		t.Fatal("Get should return a new result even with nil pool")
	}

	// Test Put with nil pool (should not panic)
	nilPool.Put(nil)
	nilPool.Put(&cachex.AsyncResult{})

	// Test Put with nil result
	pool.Put(nil) // Should not panic
}

func TestBufferPool(t *testing.T) {
	pool := cachex.NewBufferPool()
	if pool == nil {
		t.Fatal("NewBufferPool returned nil")
	}

	// Test normal Get/Put cycle
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Get returned nil buffer")
	}
	if len(buf) != 0 {
		t.Error("New buffer should have zero length")
	}
	if cap(buf) < 1024 {
		t.Error("New buffer should have at least 1024 capacity")
	}

	// Modify the buffer
	buf = append(buf, "test"...)

	// Put it back
	pool.Put(&buf)

	// Get it again and verify it was reset
	buf2 := pool.Get()
	if len(buf2) != 0 {
		t.Error("Buffer should be reset after Put")
	}
}

func TestBufferPoolNilHandling(t *testing.T) {
	pool := cachex.NewBufferPool()

	// Test Get with nil pool (should not panic)
	var nilPool *cachex.BufferPool
	buf := nilPool.Get()
	if buf == nil {
		t.Fatal("Get should return a new buffer even with nil pool")
	}

	// Test Put with nil pool (should not panic)
	nilPool.Put(nil)
	nilPool.Put(&[]byte{})

	// Test Put with nil buffer
	pool.Put(nil) // Should not panic
}

func TestBufferPoolGetWithCapacity(t *testing.T) {
	pool := cachex.NewBufferPool()

	// Test with capacity smaller than default
	buf := pool.GetWithCapacity(512)
	if cap(buf) < 1024 {
		t.Error("Should return buffer with at least default capacity")
	}

	// Test with capacity larger than default
	buf = pool.GetWithCapacity(2048)
	if cap(buf) < 2048 {
		t.Error("Should return buffer with requested capacity")
	}

	// Test with zero capacity
	buf = pool.GetWithCapacity(0)
	if cap(buf) < 1024 {
		t.Error("Should return buffer with default capacity when minCapacity is 0")
	}

	// Test with negative capacity
	buf = pool.GetWithCapacity(-1)
	if cap(buf) < 1024 {
		t.Error("Should return buffer with default capacity when minCapacity is negative")
	}
}

func TestStringSlicePool(t *testing.T) {
	pool := cachex.NewStringSlicePool()
	if pool == nil {
		t.Fatal("NewStringSlicePool returned nil")
	}

	// Test normal Get/Put cycle
	slice := pool.Get()
	if slice == nil {
		t.Fatal("Get returned nil slice")
	}
	if len(slice) != 0 {
		t.Error("New slice should have zero length")
	}
	if cap(slice) < 16 {
		t.Error("New slice should have at least 16 capacity")
	}

	// Modify the slice
	slice = append(slice, "test1", "test2")

	// Put it back
	pool.Put(&slice)

	// Get it again and verify it was reset
	slice2 := pool.Get()
	if len(slice2) != 0 {
		t.Error("Slice should be reset after Put")
	}
}

func TestStringSlicePoolNilHandling(t *testing.T) {
	pool := cachex.NewStringSlicePool()

	// Test Get with nil pool (should not panic)
	var nilPool *cachex.StringSlicePool
	slice := nilPool.Get()
	if slice == nil {
		t.Fatal("Get should return a new slice even with nil pool")
	}

	// Test Put with nil pool (should not panic)
	nilPool.Put(nil)
	nilPool.Put(&[]string{})

	// Test Put with nil slice
	pool.Put(nil) // Should not panic
}

func TestMapPool(t *testing.T) {
	pool := cachex.NewMapPool()
	if pool == nil {
		t.Fatal("NewMapPool returned nil")
	}

	// Test normal Get/Put cycle
	m := pool.Get()
	if m == nil {
		t.Fatal("Get returned nil map")
	}
	if len(m) != 0 {
		t.Error("New map should be empty")
	}

	// Modify the map
	m["key1"] = []byte("value1")
	m["key2"] = []byte("value2")

	// Put it back
	pool.Put(m)

	// Get it again and verify it was reset
	m2 := pool.Get()
	if len(m2) != 0 {
		t.Error("Map should be reset after Put")
	}
}

func TestMapPoolNilHandling(t *testing.T) {
	pool := cachex.NewMapPool()

	// Test Get with nil pool (should not panic)
	var nilPool *cachex.MapPool
	m := nilPool.Get()
	if m == nil {
		t.Fatal("Get should return a new map even with nil pool")
	}

	// Test Put with nil pool (should not panic)
	nilPool.Put(nil)
	nilPool.Put(map[string][]byte{})

	// Test Put with nil map
	pool.Put(nil) // Should not panic
}

func TestGlobalPools(t *testing.T) {
	// Test that global pools are properly initialized
	if cachex.GlobalPools.CacheItem == nil {
		t.Error("GlobalPools.CacheItem should not be nil")
	}
	if cachex.GlobalPools.AsyncResult == nil {
		t.Error("GlobalPools.AsyncResult should not be nil")
	}
	if cachex.GlobalPools.Buffer == nil {
		t.Error("GlobalPools.Buffer should not be nil")
	}
	if cachex.GlobalPools.StringSlice == nil {
		t.Error("GlobalPools.StringSlice should not be nil")
	}
	if cachex.GlobalPools.Map == nil {
		t.Error("GlobalPools.Map should not be nil")
	}
}

func TestValidateGlobalPools(t *testing.T) {
	// Test validation with properly initialized pools
	err := cachex.ValidateGlobalPools()
	if err != nil {
		t.Errorf("ValidateGlobalPools should not return error: %v", err)
	}

	// Test validation with nil pools (this would require modifying the global pools,
	// which we don't want to do in tests as it could affect other tests)
	// This test verifies the validation function exists and works with valid pools
}

func TestPoolConcurrency(t *testing.T) {
	// Test that pools work correctly under concurrent access
	pool := cachex.NewCacheItemPool()
	const numGoroutines = 100
	const numOperations = 1000

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				item := pool.Get()
				if item == nil {
					t.Error("Get returned nil item in concurrent test")
					return
				}
				item.Value = []byte("test")
				pool.Put(item)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestPoolTypeSafety(t *testing.T) {
	// Test that pools handle type assertion failures gracefully
	// Note: We can't directly test type assertion failures with the current API
	// as the pool field is unexported, but we can test the general behavior

	pool := cachex.NewCacheItemPool()
	if pool == nil {
		t.Fatal("NewCacheItemPool returned nil")
	}

	// Test that Get always returns a valid item
	item := pool.Get()
	if item == nil {
		t.Fatal("Get should return a new item")
	}

	// Test that the item has the expected zero values
	if item.Value != nil {
		t.Error("New item should have nil Value")
	}
}
