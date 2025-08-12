package cb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// Configuration
	threshold   int
	timeout     time.Duration
	halfOpenMax int

	// State management
	state        int32 // atomic state
	failureCount int64
	lastFailure  int64 // atomic timestamp
	successCount int64
	mu           sync.RWMutex

	// Metrics
	totalRequests  int64
	totalFailures  int64
	totalSuccesses int64
	totalTimeouts  int64
}

// Config holds circuit breaker configuration
type Config struct {
	Threshold   int           // Number of failures before opening circuit
	Timeout     time.Duration // How long to keep circuit open
	HalfOpenMax int           // Max requests in half-open state
}

// New creates a new circuit breaker
func New(config Config) *CircuitBreaker {
	if config.Threshold <= 0 {
		config.Threshold = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMax <= 0 {
		config.HalfOpenMax = 3
	}

	return &CircuitBreaker{
		threshold:   config.Threshold,
		timeout:     config.Timeout,
		halfOpenMax: config.HalfOpenMax,
		state:       int32(StateClosed),
	}
}

// Execute runs the operation with circuit breaker protection
func (cb *CircuitBreaker) Execute(operation func() error) error {
	atomic.AddInt64(&cb.totalRequests, 1)

	// Check if circuit is open
	if cb.getState() == StateOpen {
		if cb.shouldAttemptReset() {
			cb.transitionToHalfOpen()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute operation
	err := operation()

	// Update state based on result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// ExecuteWithContext runs the operation with context and circuit breaker protection
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, operation func() error) error {
	atomic.AddInt64(&cb.totalRequests, 1)

	// Check if circuit is open
	if cb.getState() == StateOpen {
		if cb.shouldAttemptReset() {
			cb.transitionToHalfOpen()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute operation with context
	done := make(chan error, 1)
	go func() {
		done <- operation()
	}()

	select {
	case err := <-done:
		if err != nil {
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}
		return err
	case <-ctx.Done():
		atomic.AddInt64(&cb.totalTimeouts, 1)
		cb.recordFailure()
		return fmt.Errorf("operation timeout: %w", ctx.Err())
	}
}

// ExecuteWithTimeout runs the operation with timeout and circuit breaker protection
func (cb *CircuitBreaker) ExecuteWithTimeout(timeout time.Duration, operation func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return cb.ExecuteWithContext(ctx, operation)
}

// getState returns the current state atomically
func (cb *CircuitBreaker) getState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// setState sets the state atomically
func (cb *CircuitBreaker) setState(state State) {
	atomic.StoreInt32(&cb.state, int32(state))
}

// shouldAttemptReset checks if enough time has passed to attempt reset
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	lastFailure := atomic.LoadInt64(&cb.lastFailure)
	if lastFailure == 0 {
		return false
	}
	return time.Since(time.Unix(0, lastFailure)) >= cb.timeout
}

// transitionToHalfOpen transitions the circuit to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.getState() == StateOpen {
		cb.setState(StateHalfOpen)
		atomic.StoreInt64(&cb.successCount, 0)
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt64(&cb.totalFailures, 1)
	atomic.StoreInt64(&cb.lastFailure, time.Now().UnixNano())

	state := cb.getState()
	switch state {
	case StateClosed:
		failures := atomic.AddInt64(&cb.failureCount, 1)
		if failures >= int64(cb.threshold) {
			cb.mu.Lock()
			if cb.getState() == StateClosed {
				cb.setState(StateOpen)
			}
			cb.mu.Unlock()
		}
	case StateHalfOpen:
		cb.mu.Lock()
		if cb.getState() == StateHalfOpen {
			cb.setState(StateOpen)
		}
		cb.mu.Unlock()
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	atomic.AddInt64(&cb.totalSuccesses, 1)

	state := cb.getState()
	switch state {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt64(&cb.failureCount, 0)
	case StateHalfOpen:
		successes := atomic.AddInt64(&cb.successCount, 1)
		if successes >= int64(cb.halfOpenMax) {
			cb.mu.Lock()
			if cb.getState() == StateHalfOpen {
				cb.setState(StateClosed)
				atomic.StoreInt64(&cb.failureCount, 0)
			}
			cb.mu.Unlock()
		}
	}
}

// ForceOpen forces the circuit breaker to open state
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateOpen)
}

// ForceClose forces the circuit breaker to closed state
func (cb *CircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateClosed)
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)
	atomic.StoreInt64(&cb.lastFailure, 0)
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	return cb.getState()
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() Stats {
	return Stats{
		State:          cb.getState(),
		FailureCount:   atomic.LoadInt64(&cb.failureCount),
		SuccessCount:   atomic.LoadInt64(&cb.successCount),
		LastFailure:    time.Unix(0, atomic.LoadInt64(&cb.lastFailure)),
		TotalRequests:  atomic.LoadInt64(&cb.totalRequests),
		TotalFailures:  atomic.LoadInt64(&cb.totalFailures),
		TotalSuccesses: atomic.LoadInt64(&cb.totalSuccesses),
		TotalTimeouts:  atomic.LoadInt64(&cb.totalTimeouts),
		Config:         cb.getConfig(),
	}
}

// getConfig returns the current configuration
func (cb *CircuitBreaker) getConfig() Config {
	return Config{
		Threshold:   cb.threshold,
		Timeout:     cb.timeout,
		HalfOpenMax: cb.halfOpenMax,
	}
}

// Stats holds circuit breaker statistics
type Stats struct {
	State          State
	FailureCount   int64
	SuccessCount   int64
	LastFailure    time.Time
	TotalRequests  int64
	TotalFailures  int64
	TotalSuccesses int64
	TotalTimeouts  int64
	Config         Config
}

// IsHealthy returns true if the circuit breaker is in a healthy state
func (s Stats) IsHealthy() bool {
	return s.State == StateClosed || s.State == StateHalfOpen
}

// FailureRate returns the failure rate as a percentage
func (s Stats) FailureRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.TotalFailures) / float64(s.TotalRequests) * 100
}

// SuccessRate returns the success rate as a percentage
func (s Stats) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.TotalSuccesses) / float64(s.TotalRequests) * 100
}
