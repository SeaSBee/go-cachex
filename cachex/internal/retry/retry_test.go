package retry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy()
	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, policy.InitialDelay)
	assert.Equal(t, 5*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.Multiplier)
	assert.True(t, policy.Jitter)
}

func TestAggressivePolicy(t *testing.T) {
	policy := AggressivePolicy()
	assert.Equal(t, 5, policy.MaxAttempts)
	assert.Equal(t, 50*time.Millisecond, policy.InitialDelay)
	assert.Equal(t, 10*time.Second, policy.MaxDelay)
	assert.Equal(t, 1.5, policy.Multiplier)
	assert.True(t, policy.Jitter)
}

func TestConservativePolicy(t *testing.T) {
	policy := ConservativePolicy()
	assert.Equal(t, 2, policy.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, policy.InitialDelay)
	assert.Equal(t, 2*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.Multiplier)
	assert.True(t, policy.Jitter)
}

func TestRetry_Success(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	err := Retry(policy, func() error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("first attempt failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetry_MaxAttemptsExceeded(t *testing.T) {
	policy := Policy{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	err := Retry(policy, func() error {
		attempts++
		return fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
	assert.Equal(t, 3, attempts) // MaxAttempts + 1
}

func TestRetryWithContext_Success(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	ctx := context.Background()
	attempts := 0
	err := RetryWithContext(ctx, policy, func() error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("first attempt failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryWithContext_Cancelled(t *testing.T) {
	policy := Policy{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	err := RetryWithContext(ctx, policy, func() error {
		attempts++
		return fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
	assert.True(t, attempts >= 1)
}

func TestRetryWithResult_Success(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	result, err := RetryWithResult(policy, func() (string, error) {
		attempts++
		if attempts == 1 {
			return "", fmt.Errorf("first attempt failed")
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, attempts)
}

func TestRetryWithResultAndContext_Success(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	ctx := context.Background()
	attempts := 0
	result, err := RetryWithResultAndContext(ctx, policy, func() (int, error) {
		attempts++
		if attempts == 1 {
			return 0, fmt.Errorf("first attempt failed")
		}
		return 42, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 2, attempts)
}

func TestRetryWithResultAndContext_Cancelled(t *testing.T) {
	policy := Policy{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	result, err := RetryWithResultAndContext(ctx, policy, func() (string, error) {
		attempts++
		return "", fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
	assert.Equal(t, "", result)
	assert.True(t, attempts >= 1)
}

func TestRetryWithStats_Success(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	stats, err := RetryWithStats(policy, func() error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("first attempt failed")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, stats.Success)
	assert.Equal(t, 2, stats.Attempts)
	assert.True(t, stats.TotalDelay > 0)
	assert.Nil(t, stats.LastError)
}

func TestRetryWithStats_Failure(t *testing.T) {
	policy := Policy{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	stats, err := RetryWithStats(policy, func() error {
		attempts++
		return fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.False(t, stats.Success)
	assert.Equal(t, 3, stats.Attempts) // MaxAttempts + 1
	assert.True(t, stats.TotalDelay > 0)
	assert.NotNil(t, stats.LastError)
}

func TestRetryWithStatsAndContext_Cancelled(t *testing.T) {
	policy := Policy{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	stats, err := RetryWithStatsAndContext(ctx, policy, func() error {
		attempts++
		return fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
	assert.False(t, stats.Success)
	assert.True(t, stats.Attempts >= 1)
}

func TestCalculateDelay(t *testing.T) {
	policy := Policy{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	// Test exponential backoff
	delays := []time.Duration{
		calculateDelay(policy, 0), // First retry
		calculateDelay(policy, 1), // Second retry
		calculateDelay(policy, 2), // Third retry
		calculateDelay(policy, 3), // Fourth retry
	}

	// Delays should increase exponentially
	assert.Equal(t, 100*time.Millisecond, delays[0]) // 100ms
	assert.Equal(t, 200*time.Millisecond, delays[1]) // 200ms
	assert.Equal(t, 400*time.Millisecond, delays[2]) // 400ms
	assert.Equal(t, 800*time.Millisecond, delays[3]) // 800ms

	// Test max delay cap
	policy.MaxDelay = 300 * time.Millisecond
	delays = []time.Duration{
		calculateDelay(policy, 0), // 100ms
		calculateDelay(policy, 1), // 200ms
		calculateDelay(policy, 2), // 300ms (capped)
		calculateDelay(policy, 3), // 300ms (capped)
	}

	assert.Equal(t, 100*time.Millisecond, delays[0])
	assert.Equal(t, 200*time.Millisecond, delays[1])
	assert.Equal(t, 300*time.Millisecond, delays[2])
	assert.Equal(t, 300*time.Millisecond, delays[3])
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}

	// Test that jitter adds some randomness
	baseDelay := calculateDelay(Policy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}, 0)

	jitteredDelay := calculateDelay(policy, 0)

	// Jittered delay should be close to base delay but not exactly the same
	assert.True(t, jitteredDelay >= baseDelay*90/100)  // At least 90% of base
	assert.True(t, jitteredDelay <= baseDelay*110/100) // At most 110% of base
}

func TestIsRetryableError(t *testing.T) {
	// Test with no specific retryable errors (should retry all)
	policy := Policy{
		MaxAttempts:     1,
		RetryableErrors: []error{},
	}

	assert.True(t, isRetryableError(fmt.Errorf("any error"), policy.RetryableErrors))

	// Test with specific retryable errors
	retryableErr := fmt.Errorf("retryable error")
	nonRetryableErr := fmt.Errorf("non-retryable error")

	policy.RetryableErrors = []error{retryableErr}

	assert.True(t, isRetryableError(retryableErr, policy.RetryableErrors))
	assert.False(t, isRetryableError(nonRetryableErr, policy.RetryableErrors))
}

func TestRetryableError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	retryableErr := NewRetryableError(originalErr)

	assert.Error(t, retryableErr)
	assert.Contains(t, retryableErr.Error(), "retryable error")
	assert.Equal(t, originalErr, retryableErr.Unwrap())

	assert.True(t, IsRetryable(retryableErr))
	assert.False(t, IsRetryable(originalErr))
}

func TestRetry_Performance(t *testing.T) {
	policy := Policy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	// Benchmark retry overhead
	start := time.Now()
	for i := 0; i < 100; i++ {
		Retry(policy, func() error {
			return nil
		})
	}
	duration := time.Since(start)

	// Should complete 100 operations in reasonable time
	assert.True(t, duration < 1*time.Second, "Retry overhead too high: %v", duration)
}

func TestRetry_EdgeCases(t *testing.T) {
	// Test with zero max attempts
	policy := Policy{
		MaxAttempts:  0,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	err := Retry(policy, func() error {
		attempts++
		return fmt.Errorf("attempt %d failed", attempts)
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should only try once

	// Test with negative multiplier
	policy = Policy{
		MaxAttempts:  2,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   -1.0,
		Jitter:       false,
	}

	delays := []time.Duration{
		calculateDelay(policy, 0),
		calculateDelay(policy, 1),
	}

	// Should handle negative multiplier gracefully
	assert.True(t, delays[0] >= 0)
	assert.True(t, delays[1] >= 0)

	// Test with zero multiplier
	policy.Multiplier = 0.0
	delays = []time.Duration{
		calculateDelay(policy, 0),
		calculateDelay(policy, 1),
	}

	// Should handle zero multiplier gracefully
	assert.True(t, delays[0] >= 0)
	assert.True(t, delays[1] >= 0)
}
