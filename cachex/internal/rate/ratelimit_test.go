package rate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLimiter(t *testing.T) {
	// Test with valid config
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)
	assert.NotNil(t, limiter)
}

func TestNewLimiter_InvalidConfig(t *testing.T) {
	// Test with nil config
	_, err := NewLimiter(nil)
	assert.Error(t, err)

	// Test with zero requests per second
	_, err = NewLimiter(&Config{RequestsPerSecond: 0, Burst: 5})
	assert.Error(t, err)

	// Test with negative requests per second
	_, err = NewLimiter(&Config{RequestsPerSecond: -1, Burst: 5})
	assert.Error(t, err)

	// Test with zero burst
	_, err = NewLimiter(&Config{RequestsPerSecond: 10, Burst: 0})
	assert.Error(t, err)

	// Test with negative burst
	_, err = NewLimiter(&Config{RequestsPerSecond: 10, Burst: -1})
	assert.Error(t, err)
}

func TestLimiter_Allow(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test initial burst
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(), "Should allow initial burst")
	}

	// Test after burst is exhausted
	assert.False(t, limiter.Allow(), "Should not allow after burst is exhausted")
}

func TestLimiter_AllowN(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test allowing multiple tokens
	assert.True(t, limiter.AllowN(3), "Should allow 3 tokens")
	assert.True(t, limiter.AllowN(2), "Should allow 2 more tokens")
	assert.False(t, limiter.AllowN(1), "Should not allow more tokens")
}

func TestLimiter_Wait(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             1,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Consume the burst
	limiter.Allow()

	// Test waiting with context - give more time for token refill
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx)
	assert.NoError(t, err, "Should wait and get token")
}

func TestLimiter_WaitN(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             2,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test waiting for multiple tokens
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err = limiter.WaitN(ctx, 2)
	assert.NoError(t, err, "Should wait and get 2 tokens")
}

func TestLimiter_Wait_ContextCancellation(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 1, // Very slow rate
		Burst:             1, // Need at least 1 for burst
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Consume the burst
	limiter.Allow()

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx)
	assert.Error(t, err, "Should return context error")
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestLimiter_Reserve(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             1,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test reservation when tokens are available
	reservation := limiter.Reserve()
	assert.True(t, reservation.OK, "Should reserve token")
	assert.Equal(t, 0, int(reservation.Delay), "Should have no delay")
	assert.Equal(t, 1, reservation.Tokens, "Should reserve 1 token")

	// Test reservation when no tokens are available
	reservation = limiter.Reserve()
	assert.False(t, reservation.OK, "Should not reserve token")
	assert.True(t, reservation.Delay > 0, "Should have delay")
	assert.Equal(t, 1, reservation.Tokens, "Should request 1 token")
}

func TestLimiter_ReserveN(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test reservation for multiple tokens
	reservation := limiter.ReserveN(3)
	assert.True(t, reservation.OK, "Should reserve 3 tokens")
	assert.Equal(t, 0, int(reservation.Delay), "Should have no delay")
	assert.Equal(t, 3, reservation.Tokens, "Should reserve 3 tokens")

	// Test reservation for more tokens than available
	reservation = limiter.ReserveN(5)
	assert.False(t, reservation.OK, "Should not reserve 5 tokens")
	assert.True(t, reservation.Delay > 0, "Should have delay")
	assert.Equal(t, 5, reservation.Tokens, "Should request 5 tokens")
}

func TestLimiter_GetStats(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	stats := limiter.GetStats()

	// Check required fields
	assert.Equal(t, float64(5), stats["tokens"])
	assert.Equal(t, float64(5), stats["capacity"])
	assert.Equal(t, float64(10), stats["rate"])
	assert.NotNil(t, stats["last_refill"])
	assert.Equal(t, 5, stats["available_tokens"])
}

func TestLimiter_TokenRefill(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		Burst:             5,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Consume all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Wait for token refill
	time.Sleep(100 * time.Millisecond)

	// Should have some tokens available
	assert.True(t, limiter.Allow(), "Should have tokens after refill")
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 100,
		Burst:             10,
	}

	limiter, err := NewLimiter(config)
	require.NoError(t, err)

	// Test concurrent access
	const numGoroutines = 10
	const operationsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				limiter.Allow()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that limiter is still functional
	stats := limiter.GetStats()
	assert.NotNil(t, stats)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, float64(1000), config.RequestsPerSecond)
	assert.Equal(t, 100, config.Burst)
}

func TestConservativeConfig(t *testing.T) {
	config := ConservativeConfig()
	assert.Equal(t, float64(100), config.RequestsPerSecond)
	assert.Equal(t, 10, config.Burst)
}

func TestAggressiveConfig(t *testing.T) {
	config := AggressiveConfig()
	assert.Equal(t, float64(10000), config.RequestsPerSecond)
	assert.Equal(t, 1000, config.Burst)
}
