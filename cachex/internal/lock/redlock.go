package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// Redlock implements a distributed lock using the Redlock algorithm
type Redlock struct {
	clients []LockClient
	quorum  int
}

// LockClient defines the interface for lock operations
type LockClient interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
}

// Lock represents a distributed lock
type Lock struct {
	resource string
	value    string
	ttl      time.Duration
	redlock  *Redlock
	acquired bool
	mu       sync.RWMutex
}

// LockResult represents the result of a lock acquisition attempt
type LockResult struct {
	Acquired bool
	Lock     *Lock
	Error    error
}

// Config holds Redlock configuration
type Config struct {
	Clients     []LockClient
	Quorum      int           // Minimum number of clients that must acquire the lock
	RetryDelay  time.Duration // Delay between retry attempts
	MaxRetries  int           // Maximum number of retry attempts
	DriftFactor float64       // Clock drift factor (default: 0.01)
}

// NewRedlock creates a new Redlock instance
func NewRedlock(config *Config) (*Redlock, error) {
	if len(config.Clients) == 0 {
		return nil, fmt.Errorf("at least one client is required")
	}

	quorum := config.Quorum
	if quorum == 0 {
		// Default quorum: majority of clients
		quorum = len(config.Clients)/2 + 1
	}

	if quorum > len(config.Clients) {
		return nil, fmt.Errorf("quorum cannot be greater than number of clients")
	}

	return &Redlock{
		clients: config.Clients,
		quorum:  quorum,
	}, nil
}

// Lock attempts to acquire a distributed lock
func (rl *Redlock) Lock(ctx context.Context, resource string, ttl time.Duration) (*Lock, error) {
	return rl.LockWithRetry(ctx, resource, ttl, 0, 0)
}

// LockWithRetry attempts to acquire a distributed lock with retry logic
func (rl *Redlock) LockWithRetry(ctx context.Context, resource string, ttl time.Duration, retryDelay time.Duration, maxRetries int) (*Lock, error) {
	value := generateLockValue()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Attempt to acquire lock
		lock, err := rl.tryAcquireLock(ctx, resource, value, ttl)
		if err == nil && lock != nil {
			return lock, nil
		}

		// If this is the last attempt, return error
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to acquire lock after %d attempts: %w", maxRetries+1, err)
		}

		// Wait before retry
		if retryDelay > 0 {
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to acquire lock")
}

// tryAcquireLock attempts to acquire the lock on all clients
func (rl *Redlock) tryAcquireLock(ctx context.Context, resource, value string, ttl time.Duration) (*Lock, error) {
	startTime := time.Now()
	successCount := 0
	var lastError error

	// Try to acquire lock on all clients
	for _, client := range rl.clients {
		err := client.Set(ctx, resource, []byte(value), ttl)
		if err == nil {
			successCount++
		} else {
			lastError = err
		}
	}

	// Check if we have quorum
	if successCount < rl.quorum {
		// Release locks on clients where we succeeded
		rl.releaseLocks(ctx, resource, value, successCount)
		return nil, fmt.Errorf("failed to acquire lock on quorum of clients: %d/%d, last error: %w", successCount, rl.quorum, lastError)
	}

	// Calculate validity time
	validityTime := ttl - time.Since(startTime) - time.Duration(float64(ttl)*0.01) // 1% drift factor
	if validityTime <= 0 {
		// Release locks if validity time is negative
		rl.releaseLocks(ctx, resource, value, successCount)
		return nil, fmt.Errorf("lock validity time is negative")
	}

	lock := &Lock{
		resource: resource,
		value:    value,
		ttl:      validityTime,
		redlock:  rl,
		acquired: true,
	}

	logx.Info("Lock acquired",
		logx.String("resource", resource),
		logx.String("value", value),
		logx.String("validity", validityTime.String()),
		logx.Int("success_count", successCount))

	return lock, nil
}

// releaseLocks releases locks on clients where we succeeded
func (rl *Redlock) releaseLocks(ctx context.Context, resource, value string, successCount int) {
	released := 0
	for _, client := range rl.clients {
		err := rl.releaseLock(ctx, client, resource, value)
		if err == nil {
			released++
		}
	}

	logx.Info("Released locks",
		logx.String("resource", resource),
		logx.Int("released", released),
		logx.Int("success_count", successCount))
}

// releaseLock releases a lock using Lua script for atomicity
func (rl *Redlock) releaseLock(ctx context.Context, client LockClient, resource, value string) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := client.Eval(ctx, script, []string{resource}, value)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == nil || result == 0 {
		return fmt.Errorf("lock not found or value mismatch")
	}

	return nil
}

// Unlock releases the distributed lock
func (l *Lock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired {
		return fmt.Errorf("lock not acquired")
	}

	l.acquired = false

	// Release lock on all clients
	released := 0
	for _, client := range l.redlock.clients {
		err := l.redlock.releaseLock(ctx, client, l.resource, l.value)
		if err == nil {
			released++
		}
	}

	logx.Info("Lock released",
		logx.String("resource", l.resource),
		logx.String("value", l.value),
		logx.Int("released", released))

	return nil
}

// Extend extends the lock validity time
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.acquired {
		return fmt.Errorf("lock not acquired")
	}

	// Try to extend lock on all clients
	successCount := 0
	for _, client := range l.redlock.clients {
		script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("expire", KEYS[1], ARGV[2])
			else
				return 0
			end
		`

		result, err := client.Eval(ctx, script, []string{l.resource}, l.value, int(ttl.Seconds()))
		if err == nil && result == 1 {
			successCount++
		}
	}

	// Check if we have quorum
	if successCount < l.redlock.quorum {
		return fmt.Errorf("failed to extend lock on quorum of clients: %d/%d", successCount, l.redlock.quorum)
	}

	l.ttl = ttl

	logx.Info("Lock extended",
		logx.String("resource", l.resource),
		logx.String("value", l.value),
		logx.String("new_ttl", ttl.String()),
		logx.Int("success_count", successCount))

	return nil
}

// IsAcquired returns whether the lock is currently acquired
func (l *Lock) IsAcquired() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.acquired
}

// GetResource returns the lock resource name
func (l *Lock) GetResource() string {
	return l.resource
}

// GetValue returns the lock value
func (l *Lock) GetValue() string {
	return l.value
}

// GetTTL returns the lock TTL
func (l *Lock) GetTTL() time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.ttl
}

// generateLockValue generates a unique lock value
func generateLockValue() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp-based value if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// TryLock attempts to acquire a lock without blocking
func (rl *Redlock) TryLock(ctx context.Context, resource string, ttl time.Duration) (*Lock, error) {
	return rl.tryAcquireLock(ctx, resource, generateLockValue(), ttl)
}

// LockWithTimeout attempts to acquire a lock with a timeout
func (rl *Redlock) LockWithTimeout(ctx context.Context, resource string, ttl, timeout time.Duration) (*Lock, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return rl.Lock(timeoutCtx, resource, ttl)
}

// DefaultConfig returns a default Redlock configuration
func DefaultConfig(clients []LockClient) *Config {
	return &Config{
		Clients:     clients,
		Quorum:      0, // Will be calculated as majority
		RetryDelay:  100 * time.Millisecond,
		MaxRetries:  3,
		DriftFactor: 0.01,
	}
}

// ConservativeConfig returns a conservative Redlock configuration
func ConservativeConfig(clients []LockClient) *Config {
	return &Config{
		Clients:     clients,
		Quorum:      0, // Will be calculated as majority
		RetryDelay:  500 * time.Millisecond,
		MaxRetries:  5,
		DriftFactor: 0.005,
	}
}

// AggressiveConfig returns an aggressive Redlock configuration
func AggressiveConfig(clients []LockClient) *Config {
	return &Config{
		Clients:     clients,
		Quorum:      0, // Will be calculated as majority
		RetryDelay:  50 * time.Millisecond,
		MaxRetries:  10,
		DriftFactor: 0.02,
	}
}
