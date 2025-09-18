package cachex

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seasbee/go-logx"
)

// LayeredStore implements a layered cache with memory as L1 and Redis as L2
type LayeredStore struct {
	// Stores
	l1Store Store // Memory store (fast)
	l2Store Store // Redis store (persistent)

	// Configuration
	config *LayeredConfig

	// Statistics
	stats *LayeredStats

	// Background sync management
	syncCtx    context.Context
	syncCancel context.CancelFunc
	syncWg     sync.WaitGroup
	syncOnce   sync.Once // Ensure background sync is started only once

	// Concurrency control
	asyncSemaphore chan struct{}
	asyncMu        sync.RWMutex // Protect asyncSemaphore access

	// State management
	closed int32        // Atomic flag for closed state
	mu     sync.RWMutex // Protect state changes
}

// LayeredConfig defines layered store configuration
type LayeredConfig struct {
	// Memory store configuration
	MemoryConfig *MemoryConfig `validate:"required"`

	// L2 store configuration (optional - if not provided, will use default Redis)
	L2StoreConfig *L2StoreConfig `validate:"omitempty"`

	// Layering behavior
	WritePolicy  WritePolicy   `validate:"omitempty,oneof:through behind around"` // Write-through, write-behind, or write-around
	ReadPolicy   ReadPolicy    `validate:"omitempty,oneof:through aside around"`  // Read-through, read-aside, or read-around
	SyncInterval time.Duration `validate:"min=1s,max=1h"`                         // How often to sync L1 with L2 (1s to 1h)
	EnableStats  bool          // Enable detailed statistics

	// Performance tuning
	MaxConcurrentSync int `validate:"min:1,max:100"` // Maximum concurrent sync operations

	// Timeout configurations
	AsyncOperationTimeout time.Duration `validate:"min=1s"` // Timeout for async operations (1s minimum)
	BackgroundSyncTimeout time.Duration `validate:"min=1s"` // Timeout for background sync operations
}

// L2StoreConfig defines L2 store configuration for layered cache
type L2StoreConfig struct {
	Type      string           `yaml:"type" json:"type"` // "redis", "ristretto", or "memory"
	Redis     *RedisConfig     `yaml:"redis" json:"redis"`
	Ristretto *RistrettoConfig `yaml:"ristretto" json:"ristretto"`
	Memory    *MemoryConfig    `yaml:"memory" json:"memory"`
}

// WritePolicy defines how writes are handled across layers
type WritePolicy string

const (
	WritePolicyThrough WritePolicy = "through" // Write to both L1 and L2
	WritePolicyBehind  WritePolicy = "behind"  // Write to L1, async to L2
	WritePolicyAround  WritePolicy = "around"  // Write to L2, invalidate L1
)

// ReadPolicy defines how reads are handled across layers
type ReadPolicy string

const (
	ReadPolicyThrough ReadPolicy = "through" // Read from L1, fallback to L2
	ReadPolicyAside   ReadPolicy = "aside"   // Read from L1 only, manual L2 sync
	ReadPolicyAround  ReadPolicy = "around"  // Read from L2 only, populate L1
)

// DefaultLayeredConfig returns a default configuration
func DefaultLayeredConfig() *LayeredConfig {
	return &LayeredConfig{
		MemoryConfig:          DefaultMemoryConfig(),
		WritePolicy:           WritePolicyThrough,
		ReadPolicy:            ReadPolicyThrough,
		SyncInterval:          5 * time.Minute,
		EnableStats:           true,
		MaxConcurrentSync:     10,
		AsyncOperationTimeout: 30 * time.Second,
		BackgroundSyncTimeout: 60 * time.Second,
	}
}

// HighPerfLayeredConfig returns a configuration optimized for high throughput
func HighPerfLayeredConfig() *LayeredConfig {
	return &LayeredConfig{
		MemoryConfig:          HighPerformanceMemoryConfig(),
		WritePolicy:           WritePolicyBehind,
		ReadPolicy:            ReadPolicyThrough,
		SyncInterval:          1 * time.Minute,
		EnableStats:           true,
		MaxConcurrentSync:     50,
		AsyncOperationTimeout: 30 * time.Second,
		BackgroundSyncTimeout: 60 * time.Second,
	}
}

// ResourceLayeredConfig returns a configuration for resource-constrained environments
func ResourceLayeredConfig() *LayeredConfig {
	return &LayeredConfig{
		MemoryConfig:          ResourceConstrainedMemoryConfig(),
		WritePolicy:           WritePolicyAround,
		ReadPolicy:            ReadPolicyAround,
		SyncInterval:          10 * time.Minute,
		EnableStats:           false,
		MaxConcurrentSync:     5,
		AsyncOperationTimeout: 15 * time.Second,
		BackgroundSyncTimeout: 30 * time.Second,
	}
}

// LayeredStats holds layered store statistics
type LayeredStats struct {
	L1Hits     int64
	L1Misses   int64
	L2Hits     int64
	L2Misses   int64
	SyncCount  int64
	SyncErrors int64
	mu         sync.RWMutex
}

// NewLayeredStore creates a new layered store
func NewLayeredStore(l2Store Store, config *LayeredConfig) (*LayeredStore, error) {
	// Validate inputs
	if l2Store == nil {
		return nil, fmt.Errorf("L2 store cannot be nil")
	}

	// Validate that l2Store implements all required methods
	if err := validateStore(l2Store); err != nil {
		return nil, fmt.Errorf("invalid L2 store: %w", err)
	}

	if config == nil {
		config = DefaultLayeredConfig()
	}

	// Validate memory config
	if config.MemoryConfig == nil {
		return nil, fmt.Errorf("memory config cannot be nil")
	}

	// Create L1 (memory) store
	l1Store, err := NewMemoryStore(config.MemoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	// Validate that l1Store implements all required methods
	if err := validateStore(l1Store); err != nil {
		return nil, fmt.Errorf("invalid L1 store: %w", err)
	}

	store := &LayeredStore{
		l1Store:        l1Store,
		l2Store:        l2Store,
		config:         config,
		stats:          &LayeredStats{},
		asyncSemaphore: make(chan struct{}, config.MaxConcurrentSync),
	}

	// Start background sync if needed
	if config.WritePolicy == WritePolicyBehind {
		store.startBackgroundSync()
	}

	return store, nil
}

// validateStore validates that a store implements all required methods
func validateStore(store Store) error {
	if store == nil {
		return fmt.Errorf("store is nil")
	}

	// Test basic operations to ensure store is functional
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test Get method
	result := <-store.Get(ctx, "test-validation")
	if result.Error != nil && result.Error.Error() != "key cannot be empty" {
		// Only fail if it's not the expected validation error
		return fmt.Errorf("store Get method validation failed: %w", result.Error)
	}

	return nil
}

// isClosed checks if the store is closed
func (s *LayeredStore) isClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// Get retrieves a value from the layered store (non-blocking)
func (s *LayeredStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Try L1 first
		l1Result := <-s.l1Store.Get(ctx, key)
		if l1Result.Error != nil {
			result <- AsyncResult{Error: l1Result.Error}
			return
		}

		if l1Result.Exists && l1Result.Value != nil {
			s.recordL1Hit()
			result <- AsyncResult{Value: l1Result.Value, Exists: true}
			return
		}

		s.recordL1Miss()

		// L1 miss, try L2 based on read policy
		switch s.config.ReadPolicy {
		case ReadPolicyThrough:
			s.readThroughAsync(ctx, key, result)
		case ReadPolicyAside:
			result <- AsyncResult{Exists: false} // L1 miss, no fallback
		case ReadPolicyAround:
			s.readAroundAsync(ctx, key, result)
		default:
			s.readThroughAsync(ctx, key, result)
		}
	}()

	return result
}

// Set stores a value in the layered store (non-blocking)
func (s *LayeredStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}
		if value == nil {
			result <- AsyncResult{Error: fmt.Errorf("value cannot be nil")}
			return
		}
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		switch s.config.WritePolicy {
		case WritePolicyThrough:
			s.writeThroughAsync(ctx, key, value, ttl, result)
		case WritePolicyBehind:
			s.writeBehindAsync(ctx, key, value, ttl, result)
		case WritePolicyAround:
			s.writeAroundAsync(ctx, key, value, ttl, result)
		default:
			s.writeThroughAsync(ctx, key, value, ttl, result)
		}
	}()

	return result
}

// MGet retrieves multiple values from the layered store (non-blocking)
func (s *LayeredStore) MGet(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{Values: make(map[string][]byte)}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Try L1 first
		l1Result := <-s.l1Store.MGet(ctx, keys...)
		if l1Result.Error != nil {
			result <- AsyncResult{Error: l1Result.Error}
			return
		}

		l1Results := l1Result.Values
		if l1Results == nil {
			l1Results = make(map[string][]byte)
		}

		// Find missing keys
		missingKeys := make([]string, 0)
		for _, key := range keys {
			if _, exists := l1Results[key]; !exists {
				missingKeys = append(missingKeys, key)
			}
		}

		// If all keys found in L1, return immediately
		if len(missingKeys) == 0 {
			s.recordL1Hits(int64(len(keys)))
			result <- AsyncResult{Values: l1Results}
			return
		}

		s.recordL1Hits(int64(len(keys) - len(missingKeys)))
		s.recordL1Misses(int64(len(missingKeys)))

		// Get missing keys from L2
		l2Result := <-s.l2Store.MGet(ctx, missingKeys...)
		if l2Result.Error != nil {
			result <- AsyncResult{Error: l2Result.Error}
			return
		}

		l2Results := l2Result.Values
		if l2Results == nil {
			l2Results = make(map[string][]byte)
		}

		// Merge results and populate L1 safely
		for key, value := range l2Results {
			if value != nil {
				l1Results[key] = value
				// Populate L1 for future reads with proper context and nil checks
				s.populateL1Async(key, value)
			}
		}

		s.recordL2Hits(int64(len(l2Results)))
		s.recordL2Misses(int64(len(missingKeys) - len(l2Results)))

		result <- AsyncResult{Values: l1Results}
	}()

	return result
}

// MSet stores multiple values in the layered store (non-blocking)
func (s *LayeredStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Validate all keys and values
		for key, value := range items {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
			if value == nil {
				result <- AsyncResult{Error: fmt.Errorf("value cannot be nil for key: %s", key)}
				return
			}
		}

		if len(items) == 0 {
			result <- AsyncResult{}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		switch s.config.WritePolicy {
		case WritePolicyThrough:
			s.msetThroughAsync(ctx, items, ttl, result)
		case WritePolicyBehind:
			s.msetBehindAsync(ctx, items, ttl, result)
		case WritePolicyAround:
			s.msetAroundAsync(ctx, items, ttl, result)
		default:
			s.msetThroughAsync(ctx, items, ttl, result)
		}
	}()

	return result
}

// Del removes keys from the layered store (non-blocking)
func (s *LayeredStore) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Delete from both layers
		l1Result := <-s.l1Store.Del(ctx, keys...)
		l2Result := <-s.l2Store.Del(ctx, keys...)

		// Return error if both failed (more lenient error handling)
		if l1Result.Error != nil && l2Result.Error != nil {
			result <- AsyncResult{Error: fmt.Errorf("both L1 and L2 delete failed: L1=%w, L2=%w", l1Result.Error, l2Result.Error)}
			return
		}

		// Log individual errors but don't fail the operation
		if l1Result.Error != nil {
			logx.Warn("L1 delete failed", logx.ErrorField(l1Result.Error))
		}
		if l2Result.Error != nil {
			logx.Warn("L2 delete failed", logx.ErrorField(l2Result.Error))
		}

		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in the layered store (non-blocking)
func (s *LayeredStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Check L1 first
		l1Result := <-s.l1Store.Exists(ctx, key)
		if l1Result.Error != nil {
			result <- AsyncResult{Error: l1Result.Error}
			return
		}

		if l1Result.Exists {
			result <- AsyncResult{Exists: true}
			return
		}

		// Check L2
		l2Result := <-s.l2Store.Exists(ctx, key)
		result <- AsyncResult{Exists: l2Result.Exists, Error: l2Result.Error}
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking)
func (s *LayeredStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Check L1 first
		l1Result := <-s.l1Store.TTL(ctx, key)
		if l1Result.Error != nil {
			result <- AsyncResult{Error: l1Result.Error}
			return
		}

		if l1Result.Exists && l1Result.TTL > 0 {
			result <- AsyncResult{TTL: l1Result.TTL, Exists: true}
			return
		}

		// Check L2
		l2Result := <-s.l2Store.TTL(ctx, key)
		result <- AsyncResult{TTL: l2Result.TTL, Exists: l2Result.Exists, Error: l2Result.Error}
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (s *LayeredStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is closed
		if s.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("store is closed")}
			return
		}

		// Validate stores
		if s.l1Store == nil || s.l2Store == nil {
			result <- AsyncResult{Error: fmt.Errorf("store not properly initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}
		if ttlIfCreate < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Try L1 first
		l1Result := <-s.l1Store.IncrBy(ctx, key, delta, ttlIfCreate)
		if l1Result.Error == nil {
			// Success in L1, sync to L2 based on write policy
			s.syncIncrByToL2Async(key, delta, ttlIfCreate)
			result <- AsyncResult{Result: l1Result.Result}
			return
		}

		// L1 failed, try L2
		l2Result := <-s.l2Store.IncrBy(ctx, key, delta, ttlIfCreate)
		if l2Result.Error != nil {
			result <- AsyncResult{Error: l2Result.Error}
			return
		}

		// Populate L1 for future reads with timeout
		s.populateL1AfterIncrByAsync(key, l2Result.Result, ttlIfCreate)

		result <- AsyncResult{Result: l2Result.Result}
	}()

	return result
}

// Close closes the layered store and releases resources
func (s *LayeredStore) Close() error {
	// Set closed flag
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil // Already closed
	}

	var l1Err, l2Err error

	// Stop background sync if running
	if s.syncCancel != nil {
		s.syncCancel()
		s.syncWg.Wait()
	}

	// Close L1 store
	if closer, ok := s.l1Store.(interface{ Close() error }); ok {
		l1Err = closer.Close()
	}

	// Close L2 store
	if s.l2Store != nil {
		l2Err = s.l2Store.Close()
	}

	// Return error if both failed
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("L1 close error: %w, L2 close error: %w", l1Err, l2Err)
	}

	return nil
}

// GetStats returns current statistics
func (s *LayeredStore) GetStats() *LayeredStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats == nil {
		s.stats = &LayeredStats{}
	}

	return s.stats
}

// SyncL1ToL2 synchronizes L1 cache to L2 (for write-behind mode)
func (s *LayeredStore) SyncL1ToL2(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, you would track pending writes and sync them
	s.recordSync()
	return nil
}

// startBackgroundSync starts the background sync goroutine
func (s *LayeredStore) startBackgroundSync() {
	s.syncOnce.Do(func() {
		s.syncCtx, s.syncCancel = context.WithCancel(context.Background())

		s.syncWg.Add(1)
		go func() {
			defer func() {
				// Ensure cleanup happens even on panic
				if r := recover(); r != nil {
					logx.Error("Background sync goroutine panicked", logx.Any("panic", r))
				}
				s.syncWg.Done()
			}()

			// Check if config is available
			var syncInterval time.Duration
			if s.config != nil {
				syncInterval = s.config.SyncInterval
			} else {
				syncInterval = 5 * time.Minute // Default fallback
			}

			ticker := time.NewTicker(syncInterval)
			defer ticker.Stop()

			for {
				select {
				case <-s.syncCtx.Done():
					return
				case <-ticker.C:
					// Perform sync operation with timeout
					syncCtx, cancel := context.WithTimeout(s.syncCtx, s.getBackgroundSyncTimeout())
					if err := s.SyncL1ToL2(syncCtx); err != nil {
						logx.Error("Background sync failed", logx.ErrorField(err))
						s.recordSyncError()
					}
					cancel()
				}
			}
		}()
	})
}

// getBackgroundSyncTimeout returns the background sync timeout
func (s *LayeredStore) getBackgroundSyncTimeout() time.Duration {
	if s.config != nil && s.config.BackgroundSyncTimeout > 0 {
		return s.config.BackgroundSyncTimeout
	}
	return 60 * time.Second // Default fallback
}

// getAsyncOperationTimeout returns the async operation timeout
func (s *LayeredStore) getAsyncOperationTimeout() time.Duration {
	if s.config != nil && s.config.AsyncOperationTimeout > 0 {
		return s.config.AsyncOperationTimeout
	}
	return 30 * time.Second // Default fallback
}

// recordL1Hit records an L1 cache hit
func (s *LayeredStore) recordL1Hit() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L1Hits++
	}
}

// recordL1Hits records multiple L1 cache hits
func (s *LayeredStore) recordL1Hits(count int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L1Hits += count
	}
}

// recordL1Miss records an L1 cache miss
func (s *LayeredStore) recordL1Miss() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L1Misses++
	}
}

// recordL1Misses records multiple L1 cache misses
func (s *LayeredStore) recordL1Misses(count int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L1Misses += count
	}
}

// recordL2Hit records an L2 cache hit
func (s *LayeredStore) recordL2Hit() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L2Hits++
	}
}

// recordL2Hits records multiple L2 cache hits
func (s *LayeredStore) recordL2Hits(count int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L2Hits += count
	}
}

// recordL2Miss records an L2 cache miss
func (s *LayeredStore) recordL2Miss() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L2Misses++
	}
}

// recordL2Misses records multiple L2 cache misses
func (s *LayeredStore) recordL2Misses(count int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.L2Misses += count
	}
}

// recordSync records a sync operation
func (s *LayeredStore) recordSync() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.SyncCount++
	}
}

// recordSyncError records a sync error
func (s *LayeredStore) recordSyncError() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats != nil && s.config != nil && s.config.EnableStats {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		s.stats.SyncErrors++
	}
}

// runAsyncOperation runs an async operation with concurrency control
func (s *LayeredStore) runAsyncOperation(operation func()) {
	// Check if store is closed
	if s.isClosed() {
		logx.Warn("Async operation skipped - store is closed")
		return
	}

	s.asyncMu.RLock()
	semaphore := s.asyncSemaphore
	s.asyncMu.RUnlock()

	if semaphore == nil {
		logx.Warn("Async operation skipped - semaphore not initialized")
		return
	}

	select {
	case semaphore <- struct{}{}:
		go func() {
			defer func() {
				// Ensure semaphore is always released, even on panic
				if r := recover(); r != nil {
					logx.Error("Async operation panicked", logx.Any("panic", r))
				}
				<-semaphore
			}()
			operation()
		}()
	default:
		// Semaphore is full, skip the operation
		logx.Warn("Async operation skipped due to concurrency limit")
	}
}

// populateL1Async populates L1 cache asynchronously
func (s *LayeredStore) populateL1Async(key string, value []byte) {
	s.runAsyncOperation(func() {
		// Use a timeout context to prevent hanging operations
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()

		// Check if memory config is available
		var ttl time.Duration
		if s.config != nil && s.config.MemoryConfig != nil {
			ttl = s.config.MemoryConfig.DefaultTTL
		} else {
			ttl = 5 * time.Minute // Default fallback TTL
		}

		setResult := <-s.l1Store.Set(asyncCtx, key, value, ttl)
		if setResult.Error != nil {
			logx.Error("Failed to populate L1 cache", logx.String("key", key), logx.ErrorField(setResult.Error))
		}
	})
}

// syncIncrByToL2Async syncs IncrBy operation to L2 asynchronously
func (s *LayeredStore) syncIncrByToL2Async(key string, delta int64, ttl time.Duration) {
	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()

		switch s.config.WritePolicy {
		case WritePolicyThrough:
			l2Result := <-s.l2Store.IncrBy(asyncCtx, key, delta, ttl)
			if l2Result.Error != nil {
				logx.Error("Failed to sync IncrBy to L2 in write-through mode", logx.String("key", key), logx.ErrorField(l2Result.Error))
			}
		case WritePolicyBehind:
			l2Result := <-s.l2Store.IncrBy(asyncCtx, key, delta, ttl)
			if l2Result.Error != nil {
				logx.Error("Failed to sync IncrBy to L2", logx.String("key", key), logx.ErrorField(l2Result.Error))
			}
		case WritePolicyAround:
			delResult := <-s.l1Store.Del(asyncCtx, key)
			if delResult.Error != nil {
				logx.Error("Failed to invalidate L1 cache in write-around mode", logx.String("key", key), logx.ErrorField(delResult.Error))
			}
			l2Result := <-s.l2Store.IncrBy(asyncCtx, key, delta, ttl)
			if l2Result.Error != nil {
				logx.Error("Failed to write to L2 in write-around mode", logx.String("key", key), logx.ErrorField(l2Result.Error))
			}
		}
	})
}

// populateL1AfterIncrByAsync populates L1 after IncrBy operation
func (s *LayeredStore) populateL1AfterIncrByAsync(key string, result int64, ttl time.Duration) {
	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()

		setResult := <-s.l1Store.Set(asyncCtx, key, []byte(fmt.Sprintf("%d", result)), ttl)
		if setResult.Error != nil {
			logx.Error("Failed to populate L1 cache after IncrBy", logx.String("key", key), logx.ErrorField(setResult.Error))
		}
	})
}

// readThroughAsync implements read-through policy (async)
func (s *LayeredStore) readThroughAsync(ctx context.Context, key string, result chan<- AsyncResult) {
	// Get from L2
	l2Result := <-s.l2Store.Get(ctx, key)
	if l2Result.Error != nil {
		result <- AsyncResult{Error: l2Result.Error}
		return
	}

	if l2Result.Exists && l2Result.Value != nil {
		s.recordL2Hit()
		// Populate L1 for future reads with proper context and nil checks
		s.populateL1Async(key, l2Result.Value)
		result <- AsyncResult{Value: l2Result.Value, Exists: true}
		return
	}

	s.recordL2Miss()
	result <- AsyncResult{Exists: false}
}

// readAroundAsync implements read-around policy (async)
func (s *LayeredStore) readAroundAsync(ctx context.Context, key string, result chan<- AsyncResult) {
	// Get from L2 only
	l2Result := <-s.l2Store.Get(ctx, key)
	if l2Result.Error != nil {
		result <- AsyncResult{Error: l2Result.Error}
		return
	}

	if l2Result.Exists && l2Result.Value != nil {
		s.recordL2Hit()
		result <- AsyncResult{Value: l2Result.Value, Exists: true}
	} else {
		s.recordL2Miss()
		result <- AsyncResult{Exists: false}
	}
}

// writeThroughAsync implements write-through policy (async)
func (s *LayeredStore) writeThroughAsync(ctx context.Context, key string, value []byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to both L1 and L2
	l1Result := <-s.l1Store.Set(ctx, key, value, ttl)
	l2Result := <-s.l2Store.Set(ctx, key, value, ttl)

	// Return error if both failed (more lenient error handling)
	if l1Result.Error != nil && l2Result.Error != nil {
		result <- AsyncResult{Error: fmt.Errorf("both L1 and L2 write failed: L1=%w, L2=%w", l1Result.Error, l2Result.Error)}
		return
	}

	// Log individual errors but don't fail the operation
	if l1Result.Error != nil {
		logx.Warn("L1 write failed", logx.String("key", key), logx.ErrorField(l1Result.Error))
	}
	if l2Result.Error != nil {
		logx.Warn("L2 write failed", logx.String("key", key), logx.ErrorField(l2Result.Error))
	}

	result <- AsyncResult{}
}

// writeBehindAsync implements write-behind policy (async)
func (s *LayeredStore) writeBehindAsync(ctx context.Context, key string, value []byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to L1 immediately
	l1Result := <-s.l1Store.Set(ctx, key, value, ttl)
	if l1Result.Error != nil {
		result <- AsyncResult{Error: l1Result.Error}
		return
	}

	// Write to L2 asynchronously with context timeout
	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()

		l2Result := <-s.l2Store.Set(asyncCtx, key, value, ttl)
		if l2Result.Error != nil {
			logx.Error("Failed to write behind to L2", logx.String("key", key), logx.ErrorField(l2Result.Error))
		}
	})

	result <- AsyncResult{}
}

// writeAroundAsync implements write-around policy (async)
func (s *LayeredStore) writeAroundAsync(ctx context.Context, key string, value []byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to L2 only
	l2Result := <-s.l2Store.Set(ctx, key, value, ttl)
	if l2Result.Error != nil {
		result <- AsyncResult{Error: l2Result.Error}
		return
	}

	// Invalidate L1 with timeout
	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()
		delResult := <-s.l1Store.Del(asyncCtx, key)
		if delResult.Error != nil {
			logx.Error("Failed to invalidate L1 cache in writeAroundAsync", logx.String("key", key), logx.ErrorField(delResult.Error))
		}
	})

	result <- AsyncResult{}
}

// msetThroughAsync implements MSet with write-through policy (async)
func (s *LayeredStore) msetThroughAsync(ctx context.Context, items map[string][]byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to both L1 and L2
	l1Result := <-s.l1Store.MSet(ctx, items, ttl)
	l2Result := <-s.l2Store.MSet(ctx, items, ttl)

	// Return error if both failed (more lenient error handling)
	if l1Result.Error != nil && l2Result.Error != nil {
		result <- AsyncResult{Error: fmt.Errorf("both L1 and L2 MSet failed: L1=%w, L2=%w", l1Result.Error, l2Result.Error)}
		return
	}

	// Log individual errors but don't fail the operation
	if l1Result.Error != nil {
		logx.Warn("L1 MSet failed", logx.ErrorField(l1Result.Error))
	}
	if l2Result.Error != nil {
		logx.Warn("L2 MSet failed", logx.ErrorField(l2Result.Error))
	}

	result <- AsyncResult{}
}

// msetBehindAsync implements MSet with write-behind policy (async)
func (s *LayeredStore) msetBehindAsync(ctx context.Context, items map[string][]byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to L1 immediately
	l1Result := <-s.l1Store.MSet(ctx, items, ttl)
	if l1Result.Error != nil {
		result <- AsyncResult{Error: l1Result.Error}
		return
	}

	// Write to L2 asynchronously with context timeout
	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()

		l2Result := <-s.l2Store.MSet(asyncCtx, items, ttl)
		if l2Result.Error != nil {
			logx.Error("Failed to MSet behind to L2", logx.ErrorField(l2Result.Error))
		}
	})

	result <- AsyncResult{}
}

// msetAroundAsync implements MSet with write-around policy (async)
func (s *LayeredStore) msetAroundAsync(ctx context.Context, items map[string][]byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to L2 only
	l2Result := <-s.l2Store.MSet(ctx, items, ttl)
	if l2Result.Error != nil {
		result <- AsyncResult{Error: l2Result.Error}
		return
	}

	// Invalidate L1 for all keys with timeout
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}

	s.runAsyncOperation(func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), s.getAsyncOperationTimeout())
		defer cancel()
		delResult := <-s.l1Store.Del(asyncCtx, keys...)
		if delResult.Error != nil {
			logx.Error("Failed to invalidate L1 cache in msetAroundAsync", logx.ErrorField(delResult.Error))
		}
	})

	result <- AsyncResult{}
}
