package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStressConcurrentLoad tests the cache under heavy concurrent load
func TestStressConcurrentLoad(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 1000
	const operationsPerGoroutine = 100
	var wg sync.WaitGroup

	// Start time for performance measurement
	start := time.Now()

	// Launch many goroutines performing various operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("stress-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Mix of operations
				switch j % 4 {
				case 0:
					// Set operation
					err := c.Set(key, value, time.Minute)
					assert.NoError(t, err)
				case 1:
					// Get operation
					_, _, err := c.Get(key)
					assert.NoError(t, err)
					// found might be false if key doesn't exist yet
				case 2:
					// Delete operation
					err := c.Del(key)
					assert.NoError(t, err)
				case 3:
					// Exists operation
					_, err := c.Exists(key)
					assert.NoError(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Log performance metrics
	t.Logf("Stress test completed: %d goroutines, %d operations each, total %d operations in %v",
		numGoroutines, operationsPerGoroutine, numGoroutines*operationsPerGoroutine, duration)
	t.Logf("Average operation time: %v", duration/time.Duration(numGoroutines*operationsPerGoroutine))
}

// TestStressMixedOperations tests mixed operations under load
func TestStressMixedOperations(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 500
	const operationsPerGoroutine = 50
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				baseKey := fmt.Sprintf("mixed-%d-%d", id, j)

				// Test MGet/MSet operations
				items := map[string]string{
					fmt.Sprintf("%s-1", baseKey): fmt.Sprintf("value1-%d-%d", id, j),
					fmt.Sprintf("%s-2", baseKey): fmt.Sprintf("value2-%d-%d", id, j),
					fmt.Sprintf("%s-3", baseKey): fmt.Sprintf("value3-%d-%d", id, j),
				}

				// MSet
				err := c.MSet(items, time.Minute)
				assert.NoError(t, err)

				// MGet
				keys := []string{
					fmt.Sprintf("%s-1", baseKey),
					fmt.Sprintf("%s-2", baseKey),
					fmt.Sprintf("%s-3", baseKey),
				}
				result, err := c.MGet(keys...)
				assert.NoError(t, err)
				assert.Len(t, result, 3)

				// Test ReadThrough
				loader := func(ctx context.Context) (string, error) {
					return fmt.Sprintf("loaded-%d-%d", id, j), nil
				}
				value, err := c.ReadThrough(fmt.Sprintf("%s-readthrough", baseKey), time.Minute, loader)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("loaded-%d-%d", id, j), value)

				// Test WriteBehind
				writer := func(ctx context.Context) error {
					time.Sleep(1 * time.Millisecond) // Simulate DB write
					return nil
				}
				err = c.WriteBehind(fmt.Sprintf("%s-writebehind", baseKey), fmt.Sprintf("async-%d-%d", id, j), time.Minute, writer)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Mixed operations test completed: %d goroutines, %d operations each, total %d operations in %v",
		numGoroutines, operationsPerGoroutine, numGoroutines*operationsPerGoroutine, duration)
	t.Logf("Average operation time: %v", duration/time.Duration(numGoroutines*operationsPerGoroutine))
}

// TestStressContextCancellation tests behavior under context cancellation
func TestStressContextCancellation(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cacheWithCtx := c.WithContext(ctx)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("ctx-%d-%d", id, j)

				// These operations should complete before context cancellation
				err := cacheWithCtx.Set(key, fmt.Sprintf("value-%d-%d", id, j), time.Minute)
				if err != nil {
					// It's okay if we get context cancellation errors
					t.Logf("Expected context cancellation: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStressMemoryUsage tests memory usage under load
func TestStressMemoryUsage(t *testing.T) {
	c, err := New[string](
		WithStore(newMockStore()),
		WithCodec(&mockCodec{}),
	)
	require.NoError(t, err)
	defer c.Close()

	const numOperations = 10000
	var wg sync.WaitGroup

	// Perform many operations to test memory usage
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("memory-%d", id)
			value := fmt.Sprintf("value-%d", id)

			err := c.Set(key, value, time.Minute)
			assert.NoError(t, err)

			// Retrieve the value
			retrieved, found, err := c.Get(key)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, value, retrieved)
		}(i)
	}

	wg.Wait()

	// Verify we can still retrieve values
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("memory-%d", i)
		expectedValue := fmt.Sprintf("value-%d", i)

		retrieved, found, err := c.Get(key)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expectedValue, retrieved)
	}
}
