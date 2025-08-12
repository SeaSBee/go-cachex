package cb

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "default values",
			config: Config{},
			expected: Config{
				Threshold:   5,
				Timeout:     30 * time.Second,
				HalfOpenMax: 3,
			},
		},
		{
			name: "custom values",
			config: Config{
				Threshold:   10,
				Timeout:     60 * time.Second,
				HalfOpenMax: 5,
			},
			expected: Config{
				Threshold:   10,
				Timeout:     60 * time.Second,
				HalfOpenMax: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New(tt.config)
			stats := cb.GetStats()
			assert.Equal(t, tt.expected, stats.Config)
			assert.Equal(t, StateClosed, stats.State)
		})
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Execute successful operation
	err := cb.Execute(func() error {
		return nil
	})

	assert.NoError(t, err)
	stats := cb.GetStats()
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.TotalSuccesses)
	assert.Equal(t, int64(0), stats.TotalFailures)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Execute failing operation
	err := cb.Execute(func() error {
		return fmt.Errorf("test error")
	})

	assert.Error(t, err)
	stats := cb.GetStats()
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.TotalSuccesses)
	assert.Equal(t, int64(1), stats.TotalFailures)
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := New(Config{
		Threshold:   2,
		Timeout:     100 * time.Millisecond,
		HalfOpenMax: 2,
	})

	// Start in CLOSED state
	assert.Equal(t, StateClosed, cb.GetState())

	// Fail twice to open circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return fmt.Errorf("failure %d", i+1)
		})
		assert.Error(t, err)
	}

	// Circuit should be OPEN
	assert.Equal(t, StateOpen, cb.GetState())

	// Try to execute - should fail immediately
	err := cb.Execute(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Try again - should transition to HALF_OPEN
	err = cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// Succeed again - should transition to CLOSED
	err = cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_ExecuteWithContext(t *testing.T) {
	cb := New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Test successful operation with context
	ctx := context.Background()
	err := cb.ExecuteWithContext(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)

	// Test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = cb.ExecuteWithContext(ctx, func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation timeout")
}

func TestCircuitBreaker_ExecuteWithTimeout(t *testing.T) {
	cb := New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Test successful operation with timeout
	err := cb.ExecuteWithTimeout(100*time.Millisecond, func() error {
		return nil
	})
	assert.NoError(t, err)

	// Test timeout
	err = cb.ExecuteWithTimeout(10*time.Millisecond, func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation timeout")
}

func TestCircuitBreaker_ForceOpen(t *testing.T) {
	cb := New(Config{
		Threshold:   10,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Force open
	cb.ForceOpen()
	assert.Equal(t, StateOpen, cb.GetState())

	// Try to execute - should fail
	err := cb.Execute(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCircuitBreaker_ForceClose(t *testing.T) {
	cb := New(Config{
		Threshold:   2,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Fail to open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Force close
	cb.ForceClose()
	assert.Equal(t, StateClosed, cb.GetState())

	// Should work again
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := New(Config{
		Threshold:   5,
		Timeout:     100 * time.Millisecond,
		HalfOpenMax: 3,
	})

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Start multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cb.Execute(func() error {
					if j%3 == 0 { // 33% failure rate
						return fmt.Errorf("failure from goroutine %d", id)
					}
					return nil
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	stats := cb.GetStats()
	assert.True(t, stats.TotalRequests > 0)
	assert.True(t, stats.TotalFailures >= 0)
	assert.True(t, stats.TotalSuccesses >= 0)
	assert.True(t, stats.TotalRequests == stats.TotalFailures+stats.TotalSuccesses)
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Execute some operations
	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			cb.Execute(func() error {
				return fmt.Errorf("failure")
			})
		} else {
			cb.Execute(func() error {
				return nil
			})
		}
	}

	stats := cb.GetStats()
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, int64(3), stats.TotalFailures)  // 0, 2, 4
	assert.Equal(t, int64(2), stats.TotalSuccesses) // 1, 3
	assert.Equal(t, 60.0, stats.FailureRate())
	assert.Equal(t, 40.0, stats.SuccessRate())
}

func TestCircuitBreaker_IsHealthy(t *testing.T) {
	cb := New(Config{
		Threshold:   2,
		Timeout:     100 * time.Millisecond, // Shorter timeout for testing
		HalfOpenMax: 2,
	})

	// CLOSED state is healthy
	stats := cb.GetStats()
	assert.True(t, stats.IsHealthy())

	// Fail to open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}

	// OPEN state is not healthy
	stats = cb.GetStats()
	assert.False(t, stats.IsHealthy())

	// Wait and try to transition to HALF_OPEN
	time.Sleep(150 * time.Millisecond) // Wait longer than timeout
	cb.Execute(func() error {
		return nil
	})

	// HALF_OPEN state is healthy
	stats = cb.GetStats()
	assert.True(t, stats.IsHealthy())

	// Succeed again to transition to CLOSED
	cb.Execute(func() error {
		return nil
	})

	// CLOSED state is healthy
	stats = cb.GetStats()
	assert.True(t, stats.IsHealthy())

	// Verify we're back to CLOSED state
	assert.Equal(t, StateClosed, stats.State)
}

func TestCircuitBreaker_String(t *testing.T) {
	assert.Equal(t, "CLOSED", StateClosed.String())
	assert.Equal(t, "OPEN", StateOpen.String())
	assert.Equal(t, "HALF_OPEN", StateHalfOpen.String())
	assert.Equal(t, "UNKNOWN", State(99).String())
}

func TestCircuitBreaker_EdgeCases(t *testing.T) {
	// Test with zero threshold
	cb := New(Config{
		Threshold:   0,
		Timeout:     1 * time.Second,
		HalfOpenMax: 2,
	})

	// Should use default threshold (5)
	stats := cb.GetStats()
	assert.Equal(t, 5, stats.Config.Threshold)

	// Test with negative timeout
	cb = New(Config{
		Threshold:   3,
		Timeout:     -1 * time.Second,
		HalfOpenMax: 2,
	})

	// Should use default timeout (30s)
	stats = cb.GetStats()
	assert.Equal(t, 30*time.Second, stats.Config.Timeout)

	// Test with zero half-open max
	cb = New(Config{
		Threshold:   3,
		Timeout:     1 * time.Second,
		HalfOpenMax: 0,
	})

	// Should use default half-open max (3)
	stats = cb.GetStats()
	assert.Equal(t, 3, stats.Config.HalfOpenMax)
}

func TestCircuitBreaker_Performance(t *testing.T) {
	cb := New(Config{
		Threshold:   10,
		Timeout:     1 * time.Second,
		HalfOpenMax: 5,
	})

	// Benchmark circuit breaker overhead
	start := time.Now()
	for i := 0; i < 1000; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
	duration := time.Since(start)

	// Should complete 1000 operations in reasonable time
	assert.True(t, duration < 1*time.Second, "Circuit breaker overhead too high: %v", duration)

	stats := cb.GetStats()
	assert.Equal(t, int64(1000), stats.TotalRequests)
	assert.Equal(t, int64(1000), stats.TotalSuccesses)
}
