package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Policy defines retry behavior
type Policy struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Exponential backoff multiplier
	Jitter          bool          // Add jitter to delays
	RetryableErrors []error       // Specific errors that should trigger retries
}

// DefaultPolicy returns a sensible default retry policy
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// AggressivePolicy returns an aggressive retry policy for critical operations
func AggressivePolicy() Policy {
	return Policy{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
	}
}

// ConservativePolicy returns a conservative retry policy for non-critical operations
func ConservativePolicy() Policy {
	return Policy{
		MaxAttempts:  2,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Retry executes an operation with retry logic
func Retry(policy Policy, operation func() error) error {
	return RetryWithContext(context.Background(), policy, operation)
}

// RetryWithContext executes an operation with retry logic and context
func RetryWithContext(ctx context.Context, policy Policy, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, policy.RetryableErrors) {
			return err
		}

		// Don't retry on last attempt
		if attempt == policy.MaxAttempts {
			break
		}

		// Calculate delay
		delay := calculateDelay(policy, attempt)

		// Wait with context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during delay: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", policy.MaxAttempts+1, lastErr)
}

// RetryWithResult executes an operation that returns a result with retry logic
func RetryWithResult[T any](policy Policy, operation func() (T, error)) (T, error) {
	return RetryWithResultAndContext(context.Background(), policy, operation)
}

// RetryWithResultAndContext executes an operation that returns a result with retry logic and context
func RetryWithResultAndContext[T any](ctx context.Context, policy Policy, operation func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= policy.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute operation
		result, err := operation()
		if err == nil {
			return result, nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, policy.RetryableErrors) {
			return zero, err
		}

		// Don't retry on last attempt
		if attempt == policy.MaxAttempts {
			break
		}

		// Calculate delay
		delay := calculateDelay(policy, attempt)

		// Wait with context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return zero, fmt.Errorf("retry cancelled during delay: %w", ctx.Err())
		}
	}

	return zero, fmt.Errorf("operation failed after %d attempts: %w", policy.MaxAttempts+1, lastErr)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff
func calculateDelay(policy Policy, attempt int) time.Duration {
	// Ensure multiplier is positive to avoid negative delays
	multiplier := policy.Multiplier
	if multiplier <= 0 {
		multiplier = 1.0 // Default to no exponential backoff
	}

	// Calculate exponential backoff
	delay := float64(policy.InitialDelay) * math.Pow(multiplier, float64(attempt))

	// Ensure delay is non-negative
	if delay < 0 {
		delay = 0
	}

	// Apply maximum delay cap
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}

	// Add jitter if enabled
	if policy.Jitter {
		jitter := delay * 0.1 // 10% jitter
		delay += (rand.Float64() * 2 * jitter) - jitter
	}

	// Final safety check to ensure non-negative delay
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error, retryableErrors []error) bool {
	// If no specific errors are defined, retry on all errors
	if len(retryableErrors) == 0 {
		return true
	}

	// Check if error matches any retryable error
	for _, retryableErr := range retryableErrors {
		if err == retryableErr {
			return true
		}
	}

	return false
}

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) RetryableError {
	return RetryableError{Err: err}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	_, ok := err.(RetryableError)
	return ok
}

// RetryStats holds retry statistics
type RetryStats struct {
	Attempts   int
	TotalDelay time.Duration
	LastError  error
	Success    bool
}

// RetryWithStats executes an operation with retry logic and returns statistics
func RetryWithStats(policy Policy, operation func() error) (RetryStats, error) {
	return RetryWithStatsAndContext(context.Background(), policy, operation)
}

// RetryWithStatsAndContext executes an operation with retry logic and context, returning statistics
func RetryWithStatsAndContext(ctx context.Context, policy Policy, operation func() error) (RetryStats, error) {
	stats := RetryStats{}
	var lastErr error

	for attempt := 0; attempt <= policy.MaxAttempts; attempt++ {
		stats.Attempts = attempt + 1

		// Check context cancellation
		select {
		case <-ctx.Done():
			return stats, fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute operation
		err := operation()
		if err == nil {
			stats.Success = true
			stats.LastError = nil // Clear last error on success
			return stats, nil     // Success
		}

		lastErr = err
		stats.LastError = err

		// Check if error is retryable
		if !isRetryableError(err, policy.RetryableErrors) {
			return stats, err
		}

		// Don't retry on last attempt
		if attempt == policy.MaxAttempts {
			break
		}

		// Calculate delay
		delay := calculateDelay(policy, attempt)
		stats.TotalDelay += delay

		// Wait with context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return stats, fmt.Errorf("retry cancelled during delay: %w", ctx.Err())
		}
	}

	return stats, fmt.Errorf("operation failed after %d attempts: %w", policy.MaxAttempts+1, lastErr)
}
