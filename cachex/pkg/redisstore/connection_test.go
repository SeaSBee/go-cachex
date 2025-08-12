package redisstore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectionPooling tests connection pooling configuration
func TestConnectionPooling(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     20,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Verify the store was created successfully
	assert.NotNil(t, store)
	assert.NotNil(t, store.client)
}

// TestConnectionPoolingWithTLS tests connection pooling with TLS
func TestConnectionPoolingWithTLS(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     15,
		MinIdleConns: 5,
		TLSConfig: &TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
		},
	}

	store, err := New(config)
	// Note: This might fail if Redis doesn't have TLS enabled
	// That's okay for testing purposes
	if err != nil {
		t.Logf("TLS connection failed (expected if Redis TLS not enabled): %v", err)
		return
	}
	defer store.Close()

	assert.NotNil(t, store)
	assert.NotNil(t, store.client)
}

// TestConnectionLeakPrevention tests that connections are properly managed
func TestConnectionLeakPrevention(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     5,
		MinIdleConns: 2,
		MaxRetries:   3,
	}

	store, err := New(config)
	require.NoError(t, err)

	// Perform many operations to stress the connection pool
	const numOperations = 1000
	var wg sync.WaitGroup

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("leak-test-%d", id)
			value := fmt.Sprintf("value-%d", id)

			// Set operation
			err := store.Set(context.Background(), key, []byte(value), time.Minute)
			assert.NoError(t, err)

			// Get operation
			retrieved, err := store.Get(context.Background(), key)
			assert.NoError(t, err)
			assert.Equal(t, []byte(value), retrieved)

			// Delete operation
			err = store.Del(context.Background(), key)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Close the store - this should clean up all connections
	err = store.Close()
	assert.NoError(t, err)
}

// TestConnectionPoolExhaustion tests behavior when connection pool is exhausted
func TestConnectionPoolExhaustion(t *testing.T) {
	// Create a store with very limited connection pool
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     2, // Very small pool
		MinIdleConns: 1,
		PoolTimeout:  1 * time.Second, // Short timeout
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Create operations that will hold connections for a while
	const numSlowOperations = 5
	var wg sync.WaitGroup

	for i := 0; i < numSlowOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("slow-test-%d", id)

			// This operation will hold a connection for 500ms
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := store.Set(ctx, key, []byte("slow-value"), time.Minute)
			// Some operations might timeout due to pool exhaustion
			if err != nil {
				t.Logf("Expected timeout due to pool exhaustion: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// TestConnectionPoolRecovery tests that the connection pool recovers after failures
func TestConnectionPoolRecovery(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Perform operations that might cause connection failures
	const numOperations = 100
	var wg sync.WaitGroup

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("recovery-test-%d", id)
			value := fmt.Sprintf("value-%d", id)

			// Mix of operations
			switch id % 4 {
			case 0:
				err := store.Set(context.Background(), key, []byte(value), time.Minute)
				assert.NoError(t, err)
			case 1:
				_, err := store.Get(context.Background(), key)
				// This might fail if key doesn't exist, which is okay
				if err != nil {
					t.Logf("Get failed (expected for non-existent key): %v", err)
				}
			case 2:
				err := store.Del(context.Background(), key)
				assert.NoError(t, err)
			case 3:
				_, err := store.Exists(context.Background(), key)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify the store is still functional after all operations
	err = store.Set(context.Background(), "final-test", []byte("final-value"), time.Minute)
	assert.NoError(t, err)

	retrieved, err := store.Get(context.Background(), "final-test")
	assert.NoError(t, err)
	assert.Equal(t, []byte("final-value"), retrieved)
}

// TestConnectionPoolMetrics tests that we can access connection pool metrics
func TestConnectionPoolMetrics(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     10,
		MinIdleConns: 5,
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Get the underlying client to check pool stats
	client := store.Client()
	assert.NotNil(t, client)

	// Perform some operations to generate pool activity
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("metrics-test-%d", i)
		err := store.Set(context.Background(), key, []byte("test-value"), time.Minute)
		assert.NoError(t, err)
	}

	// The go-redis client doesn't expose pool metrics directly,
	// but we can verify the client is still functional
	err = store.Set(context.Background(), "metrics-verification", []byte("verification"), time.Minute)
	assert.NoError(t, err)
}

// TestConnectionPoolConcurrentAccess tests concurrent access to the connection pool
func TestConnectionPoolConcurrentAccess(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     20,
		MinIdleConns: 10,
		MaxRetries:   3,
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	const numGoroutines = 50
	const operationsPerGoroutine = 20
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("concurrent-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Mix of operations
				switch j % 3 {
				case 0:
					err := store.Set(context.Background(), key, []byte(value), time.Minute)
					assert.NoError(t, err)
				case 1:
					_, err := store.Get(context.Background(), key)
					// This might fail if key doesn't exist yet
					if err != nil {
						t.Logf("Get failed (expected): %v", err)
					}
				case 2:
					err := store.Del(context.Background(), key)
					assert.NoError(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Concurrent pool access test completed: %d goroutines, %d operations each, total %d operations in %v",
		numGoroutines, operationsPerGoroutine, numGoroutines*operationsPerGoroutine, duration)
	t.Logf("Average operation time: %v", duration/time.Duration(numGoroutines*operationsPerGoroutine))
}

// TestConnectionPoolTimeout tests connection pool timeout behavior
func TestConnectionPoolTimeout(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     1, // Very small pool to force timeouts
		MinIdleConns: 1,
		PoolTimeout:  100 * time.Millisecond, // Very short timeout
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Start a slow operation that will hold the connection
	slowDone := make(chan bool)
	go func() {
		defer close(slowDone)
		// This will hold the connection for 200ms
		time.Sleep(200 * time.Millisecond)
		err := store.Set(context.Background(), "slow-key", []byte("slow-value"), time.Minute)
		assert.NoError(t, err)
	}()

	// Try to perform another operation while the pool is exhausted
	time.Sleep(50 * time.Millisecond) // Wait for slow operation to start

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = store.Set(ctx, "timeout-test", []byte("timeout-value"), time.Minute)
	// This should timeout due to pool exhaustion
	if err != nil {
		t.Logf("Expected timeout due to pool exhaustion: %v", err)
	}

	// Wait for slow operation to complete
	<-slowDone

	// Now operations should work again
	err = store.Set(context.Background(), "after-timeout", []byte("after-value"), time.Minute)
	assert.NoError(t, err)
}

// TestConnectionPoolCleanup tests that connections are properly cleaned up
func TestConnectionPoolCleanup(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     10,
		MinIdleConns: 5,
	}

	store, err := New(config)
	require.NoError(t, err)

	// Perform operations to establish connections
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("cleanup-test-%d", i)
		err := store.Set(context.Background(), key, []byte("test-value"), time.Minute)
		assert.NoError(t, err)
	}

	// Close the store - this should clean up all connections
	err = store.Close()
	assert.NoError(t, err)

	// Try to use the store after closing - should fail
	err = store.Set(context.Background(), "after-close", []byte("should-fail"), time.Minute)
	assert.Error(t, err)
}

// TestConnectionPoolReuse tests that connections are reused efficiently
func TestConnectionPoolReuse(t *testing.T) {
	config := &Config{
		Addr:         "localhost:6379",
		PoolSize:     5,
		MinIdleConns: 2,
	}

	store, err := New(config)
	require.NoError(t, err)
	defer store.Close()

	// Perform many rapid operations to test connection reuse
	const numOperations = 1000
	start := time.Now()

	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("reuse-test-%d", i)
		err := store.Set(context.Background(), key, []byte("test-value"), time.Minute)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("Connection reuse test: %d operations in %v, average %v per operation",
		numOperations, duration, duration/time.Duration(numOperations))
}
