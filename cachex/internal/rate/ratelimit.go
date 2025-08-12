package rate

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Limiter implements token bucket rate limiting
type Limiter struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	rate       float64
	lastRefill time.Time
}

// Config holds rate limiting configuration
type Config struct {
	RequestsPerSecond float64 // Tokens per second
	Burst             int     // Maximum burst capacity
}

// NewLimiter creates a new rate limiter
func NewLimiter(config *Config) (*Limiter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.RequestsPerSecond <= 0 {
		return nil, fmt.Errorf("requests per second must be positive")
	}

	if config.Burst <= 0 {
		return nil, fmt.Errorf("burst capacity must be positive")
	}

	return &Limiter{
		tokens:     float64(config.Burst),
		capacity:   float64(config.Burst),
		rate:       config.RequestsPerSecond,
		lastRefill: time.Now(),
	}, nil
}

// Allow checks if a request is allowed and consumes a token if so
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= 1 {
		l.tokens--
		return true
	}

	return false
}

// AllowN checks if N requests are allowed and consumes tokens if so
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		return true
	}

	return false
}

// Wait blocks until a token is available
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN blocks until N tokens are available
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if l.AllowN(n) {
				return nil
			}
			time.Sleep(10 * time.Millisecond) // Small delay to avoid busy waiting
		}
	}
}

// Reserve reserves a token and returns when it will be available
func (l *Limiter) Reserve() *Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= 1 {
		l.tokens--
		return &Reservation{
			OK:     true,
			Delay:  0,
			Tokens: 1,
		}
	}

	// Calculate delay needed
	tokensNeeded := 1.0 - l.tokens
	delay := time.Duration(tokensNeeded / l.rate * float64(time.Second))

	return &Reservation{
		OK:     false,
		Delay:  delay,
		Tokens: 1,
	}
}

// ReserveN reserves N tokens and returns when they will be available
func (l *Limiter) ReserveN(n int) *Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		return &Reservation{
			OK:     true,
			Delay:  0,
			Tokens: n,
		}
	}

	// Calculate delay needed
	tokensNeeded := float64(n) - l.tokens
	delay := time.Duration(tokensNeeded / l.rate * float64(time.Second))

	return &Reservation{
		OK:     false,
		Delay:  delay,
		Tokens: n,
	}
}

// refill adds tokens based on time elapsed
func (l *Limiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill)

	// Calculate tokens to add
	tokensToAdd := elapsed.Seconds() * l.rate

	// Add tokens, but don't exceed capacity
	l.tokens = min(l.capacity, l.tokens+tokensToAdd)
	l.lastRefill = now
}

// GetStats returns current rate limiter statistics
func (l *Limiter) GetStats() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	return map[string]interface{}{
		"tokens":           l.tokens,
		"capacity":         l.capacity,
		"rate":             l.rate,
		"last_refill":      l.lastRefill,
		"available_tokens": int(l.tokens),
	}
}

// Reservation represents a rate limit reservation
type Reservation struct {
	OK     bool          // Whether the reservation was successful
	Delay  time.Duration // How long to wait if not OK
	Tokens int           // Number of tokens reserved
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// DefaultConfig returns a default rate limiting configuration
func DefaultConfig() *Config {
	return &Config{
		RequestsPerSecond: 1000,
		Burst:             100,
	}
}

// ConservativeConfig returns a conservative rate limiting configuration
func ConservativeConfig() *Config {
	return &Config{
		RequestsPerSecond: 100,
		Burst:             10,
	}
}

// AggressiveConfig returns an aggressive rate limiting configuration
func AggressiveConfig() *Config {
	return &Config{
		RequestsPerSecond: 10000,
		Burst:             1000,
	}
}
