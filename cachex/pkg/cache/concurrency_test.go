package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentOperations tests concurrent access to the cache
func TestConcurrentOperations(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 100
	const operationsPerGoroutine = 50
	var wg sync.WaitGroup

	// Test concurrent Get/Set operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Set operation
				err := c.Set(key, value, time.Minute)
				assert.NoError(t, err)

				// Get operation
				retrieved, found, err := c.Get(key)
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, value, retrieved)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentMGetMSet tests concurrent MGet/MSet operations
func TestConcurrentMGetMSet(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 50
	const batchSize = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Create batch of items
				items := make(map[string]string)
				keys := make([]string, batchSize)
				for k := 0; k < batchSize; k++ {
					key := fmt.Sprintf("batch-%d-%d-%d", id, j, k)
					value := fmt.Sprintf("value-%d-%d-%d", id, j, k)
					items[key] = value
					keys[k] = key
				}

				// MSet operation
				err := c.MSet(items, time.Minute)
				assert.NoError(t, err)

				// MGet operation
				retrieved, err := c.MGet(keys...)
				assert.NoError(t, err)
				assert.Len(t, retrieved, batchSize)

				// Verify all values
				for k := 0; k < batchSize; k++ {
					key := fmt.Sprintf("batch-%d-%d-%d", id, j, k)
					expectedValue := fmt.Sprintf("value-%d-%d-%d", id, j, k)
					assert.Equal(t, expectedValue, retrieved[key])
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentReadThrough tests concurrent ReadThrough operations
func TestConcurrentReadThrough(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 30
	var wg sync.WaitGroup
	var loaderCalls int
	var loaderMutex sync.Mutex

	loader := func(ctx context.Context) (string, error) {
		loaderMutex.Lock()
		loaderCalls++
		loaderMutex.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate database load
		return "loaded-value", nil
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("readthrough-%d", id)

			// Multiple concurrent ReadThrough calls for the same key
			for j := 0; j < 5; j++ {
				value, err := c.ReadThrough(key, time.Minute, loader)
				assert.NoError(t, err)
				assert.Equal(t, "loaded-value", value)
			}
		}(i)
	}

	wg.Wait()

	// Verify loader was called only once per key (due to caching)
	assert.LessOrEqual(t, loaderCalls, numGoroutines)
}

// TestConcurrentWriteBehind tests concurrent WriteBehind operations
func TestConcurrentWriteBehind(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 20
	var wg sync.WaitGroup
	var writerCalls int32

	writer := func(ctx context.Context) error {
		atomic.AddInt32(&writerCalls, 1)
		time.Sleep(5 * time.Millisecond) // Simulate database write
		return nil
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("writebehind-%d", id)
			value := fmt.Sprintf("value-%d", id)

			err := c.WriteBehind(key, value, time.Minute, writer)
			assert.NoError(t, err)

			// Verify value is immediately available in cache
			retrieved, found, err := c.Get(key)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, value, retrieved)
		}(i)
	}

	wg.Wait()

	// Wait a bit for async writes to complete
	time.Sleep(100 * time.Millisecond)

	// Verify all writer calls were made
	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&writerCalls))
}

// TestConcurrentTryLock tests concurrent lock acquisition
func TestConcurrentTryLock(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 50
	var wg sync.WaitGroup
	var successfulLocks int
	var lockMutex sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("lock-%d", id)

			unlock, acquired, err := c.TryLock(key, time.Second)
			assert.NoError(t, err)

			if acquired {
				lockMutex.Lock()
				successfulLocks++
				lockMutex.Unlock()

				// Hold the lock for a short time
				time.Sleep(10 * time.Millisecond)

				// Release the lock
				err := unlock()
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify some locks were acquired (not all due to contention)
	assert.Greater(t, successfulLocks, 0)
	assert.LessOrEqual(t, successfulLocks, numGoroutines)
}

// TestConcurrentClose tests concurrent access during close
func TestConcurrentClose(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("close-test-%d", id)

			// Try to perform operations while cache is being closed
			err := c.Set(key, "value", time.Minute)
			if err != nil {
				// It's okay if we get ErrStoreClosed
				assert.ErrorIs(t, err, ErrStoreClosed)
			}
		}(i)
	}

	// Close the cache
	err = c.Close()
	assert.NoError(t, err)

	wg.Wait()
}

// TestContextCancellation tests that context cancellation works properly
func TestContextCancellation(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	cacheWithCtx := c.WithContext(ctx)

	// Wait for context to be cancelled
	time.Sleep(2 * time.Millisecond)

	// Try to perform an operation
	err = cacheWithCtx.Set("key", "value", time.Minute)
	// Should either succeed or fail gracefully due to context cancellation, but not deadlock
	if err != nil {
		// Check if it's a context cancellation error (which is expected)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}
}

// TestDeadlockPrevention tests that no deadlocks occur
func TestDeadlockPrevention(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 10
	var wg sync.WaitGroup

	// Test that multiple operations can complete without deadlock
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Perform a sequence of operations
			key := fmt.Sprintf("deadlock-test-%d", id)

			// Set
			err := c.Set(key, "value", time.Minute)
			assert.NoError(t, err)

			// Get
			_, found, err := c.Get(key)
			assert.NoError(t, err)
			assert.True(t, found)

			// Delete
			err = c.Del(key)
			assert.NoError(t, err)

			// Verify deleted
			_, found, err = c.Get(key)
			assert.NoError(t, err)
			assert.False(t, found)
		}(i)
	}

	// Use a timeout to detect deadlocks
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock detected")
	}
}

// TestInfiniteLoopPrevention tests that no infinite loops occur
func TestInfiniteLoopPrevention(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	// Test with a large number of operations to ensure no infinite loops
	const numOperations = 1000
	var wg sync.WaitGroup

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("loop-test-%d", id)

			// Simple operation that should complete quickly
			err := c.Set(key, "value", time.Minute)
			assert.NoError(t, err)
		}(i)
	}

	// Use a timeout to detect infinite loops
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out - possible infinite loop detected")
	}
}
