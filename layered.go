package cachex

import (
	"context"
	"fmt"
	"sync"
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
}

// LayeredConfig defines layered store configuration
type LayeredConfig struct {
	// Memory store configuration
	MemoryConfig *MemoryConfig

	// Layering behavior
	WritePolicy  WritePolicy   // Write-through, write-behind, or write-around
	ReadPolicy   ReadPolicy    // Read-through, read-aside, or read-around
	SyncInterval time.Duration // How often to sync L1 with L2
	EnableStats  bool          // Enable detailed statistics

	// Performance tuning
	MaxConcurrentSync int // Maximum concurrent sync operations
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
		MemoryConfig:      DefaultMemoryConfig(),
		WritePolicy:       WritePolicyThrough,
		ReadPolicy:        ReadPolicyThrough,
		SyncInterval:      5 * time.Minute,
		EnableStats:       true,
		MaxConcurrentSync: 10,
	}
}

// HighPerfLayeredConfig returns a configuration optimized for high throughput
func HighPerfLayeredConfig() *LayeredConfig {
	return &LayeredConfig{
		MemoryConfig:      HighPerformanceMemoryConfig(),
		WritePolicy:       WritePolicyBehind,
		ReadPolicy:        ReadPolicyThrough,
		SyncInterval:      1 * time.Minute,
		EnableStats:       true,
		MaxConcurrentSync: 50,
	}
}

// ResourceLayeredConfig returns a configuration for resource-constrained environments
func ResourceLayeredConfig() *LayeredConfig {
	return &LayeredConfig{
		MemoryConfig:      ResourceConstrainedMemoryConfig(),
		WritePolicy:       WritePolicyAround,
		ReadPolicy:        ReadPolicyAround,
		SyncInterval:      10 * time.Minute,
		EnableStats:       false,
		MaxConcurrentSync: 5,
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
	if config == nil {
		config = DefaultLayeredConfig()
	}

	// Create L1 (memory) store
	l1Store, err := NewMemoryStore(config.MemoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	store := &LayeredStore{
		l1Store: l1Store,
		l2Store: l2Store,
		config:  config,
		stats:   &LayeredStats{},
	}

	// Start background sync if needed
	if config.WritePolicy == WritePolicyBehind {
		store.startBackgroundSync()
	}

	return store, nil
}

// Get retrieves a value from the layered store (non-blocking)
func (s *LayeredStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
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

		// Merge results
		for key, value := range l2Results {
			if value != nil {
				l1Results[key] = value
				// Populate L1 for future reads
				go func(k string, v []byte) {
					setResult := <-s.l1Store.Set(context.Background(), k, v, s.config.MemoryConfig.DefaultTTL)
					if setResult.Error != nil {
						logx.Error("Failed to populate L1 cache", logx.String("key", k), logx.ErrorField(setResult.Error))
					}
				}(key, value)
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

		// Delete from both layers
		l1Result := <-s.l1Store.Del(ctx, keys...)
		l2Result := <-s.l2Store.Del(ctx, keys...)

		// Return error if both failed
		if l1Result.Error != nil && l2Result.Error != nil {
			result <- AsyncResult{Error: fmt.Errorf("L1 error: %w, L2 error: %w", l1Result.Error, l2Result.Error)}
			return
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

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
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

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
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

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}
		if ttlIfCreate < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Try L1 first
		l1Result := <-s.l1Store.IncrBy(ctx, key, delta, ttlIfCreate)
		if l1Result.Error == nil {
			// Success in L1, sync to L2 based on write policy
			switch s.config.WritePolicy {
			case WritePolicyThrough:
				// Sync to L2 in write-through mode
				go func(k string, d int64, ttl time.Duration) {
					l2Result := <-s.l2Store.IncrBy(context.Background(), k, d, ttl)
					if l2Result.Error != nil {
						logx.Error("Failed to sync IncrBy to L2 in write-through mode", logx.String("key", k), logx.ErrorField(l2Result.Error))
					}
				}(key, delta, ttlIfCreate)
			case WritePolicyBehind:
				// Async sync to L2
				go func(k string, d int64, ttl time.Duration) {
					l2Result := <-s.l2Store.IncrBy(context.Background(), k, d, ttl)
					if l2Result.Error != nil {
						logx.Error("Failed to sync IncrBy to L2", logx.String("key", k), logx.ErrorField(l2Result.Error))
					}
				}(key, delta, ttlIfCreate)
			case WritePolicyAround:
				// Invalidate L1, write to L2
				go func(k string, d int64, ttl time.Duration) {
					delResult := <-s.l1Store.Del(context.Background(), k)
					if delResult.Error != nil {
						logx.Error("Failed to invalidate L1 cache in write-around mode", logx.String("key", k), logx.ErrorField(delResult.Error))
					}
					l2Result := <-s.l2Store.IncrBy(context.Background(), k, d, ttl)
					if l2Result.Error != nil {
						logx.Error("Failed to write to L2 in write-around mode", logx.String("key", k), logx.ErrorField(l2Result.Error))
					}
				}(key, delta, ttlIfCreate)
			}
			result <- AsyncResult{Result: l1Result.Result}
			return
		}

		// L1 failed, try L2
		l2Result := <-s.l2Store.IncrBy(ctx, key, delta, ttlIfCreate)
		if l2Result.Error != nil {
			result <- AsyncResult{Error: l2Result.Error}
			return
		}

		// Populate L1 for future reads
		go func(k string, r int64, ttl time.Duration) {
			setResult := <-s.l1Store.Set(context.Background(), k, []byte(fmt.Sprintf("%d", r)), ttl)
			if setResult.Error != nil {
				logx.Error("Failed to populate L1 cache after IncrBy", logx.String("key", k), logx.ErrorField(setResult.Error))
			}
		}(key, l2Result.Result, ttlIfCreate)

		result <- AsyncResult{Result: l2Result.Result}
	}()

	return result
}

// Close closes the layered store and releases resources
func (s *LayeredStore) Close() error {
	var l1Err, l2Err error

	// Close L1 store
	if closer, ok := s.l1Store.(interface{ Close() error }); ok {
		l1Err = closer.Close()
	}

	// Close L2 store
	l2Err = s.l2Store.Close()

	// Return error if both failed
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("L1 close error: %w, L2 close error: %w", l1Err, l2Err)
	}

	return nil
}

// GetStats returns current statistics
func (s *LayeredStore) GetStats() *LayeredStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

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
	// This is a placeholder for background sync implementation
	// In a real implementation, you would track pending writes and sync them periodically
}

// recordL1Hit records an L1 cache hit
func (s *LayeredStore) recordL1Hit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Hits++
		s.stats.mu.Unlock()
	}
}

// recordL1Hits records multiple L1 cache hits
func (s *LayeredStore) recordL1Hits(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Hits += count
		s.stats.mu.Unlock()
	}
}

// recordL1Miss records an L1 cache miss
func (s *LayeredStore) recordL1Miss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Misses++
		s.stats.mu.Unlock()
	}
}

// recordL1Misses records multiple L1 cache misses
func (s *LayeredStore) recordL1Misses(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Misses += count
		s.stats.mu.Unlock()
	}
}

// recordL2Hit records an L2 cache hit
func (s *LayeredStore) recordL2Hit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Hits++
		s.stats.mu.Unlock()
	}
}

// recordL2Hits records multiple L2 cache hits
func (s *LayeredStore) recordL2Hits(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Hits += count
		s.stats.mu.Unlock()
	}
}

// recordL2Miss records an L2 cache miss
func (s *LayeredStore) recordL2Miss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Misses++
		s.stats.mu.Unlock()
	}
}

// recordL2Misses records multiple L2 cache misses
func (s *LayeredStore) recordL2Misses(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Misses += count
		s.stats.mu.Unlock()
	}
}

// recordSync records a sync operation
func (s *LayeredStore) recordSync() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.SyncCount++
		s.stats.mu.Unlock()
	}
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
		// Populate L1 for future reads
		go func(k string, v []byte) {
			setResult := <-s.l1Store.Set(context.Background(), k, v, s.config.MemoryConfig.DefaultTTL)
			if setResult.Error != nil {
				logx.Error("Failed to populate L1 cache in readThroughAsync", logx.String("key", k), logx.ErrorField(setResult.Error))
			}
		}(key, l2Result.Value)
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

	// Return error if both failed
	if l1Result.Error != nil && l2Result.Error != nil {
		result <- AsyncResult{Error: fmt.Errorf("L1 error: %w, L2 error: %w", l1Result.Error, l2Result.Error)}
		return
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
	go func(k string, v []byte, t time.Duration) {
		// Use a timeout context to prevent hanging operations
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		l2Result := <-s.l2Store.Set(asyncCtx, k, v, t)
		if l2Result.Error != nil {
			logx.Error("Failed to write behind to L2", logx.String("key", k), logx.ErrorField(l2Result.Error))
		}
	}(key, value, ttl)

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

	// Invalidate L1
	go func(k string) {
		delResult := <-s.l1Store.Del(context.Background(), k)
		if delResult.Error != nil {
			logx.Error("Failed to invalidate L1 cache in writeAroundAsync", logx.String("key", k), logx.ErrorField(delResult.Error))
		}
	}(key)

	result <- AsyncResult{}
}

// msetThroughAsync implements MSet with write-through policy (async)
func (s *LayeredStore) msetThroughAsync(ctx context.Context, items map[string][]byte, ttl time.Duration, result chan<- AsyncResult) {
	// Write to both L1 and L2
	l1Result := <-s.l1Store.MSet(ctx, items, ttl)
	l2Result := <-s.l2Store.MSet(ctx, items, ttl)

	// Return error if both failed
	if l1Result.Error != nil && l2Result.Error != nil {
		result <- AsyncResult{Error: fmt.Errorf("L1 error: %w, L2 error: %w", l1Result.Error, l2Result.Error)}
		return
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
	go func(itemsMap map[string][]byte, t time.Duration) {
		// Use a timeout context to prevent hanging operations
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		l2Result := <-s.l2Store.MSet(asyncCtx, itemsMap, t)
		if l2Result.Error != nil {
			logx.Error("Failed to MSet behind to L2", logx.ErrorField(l2Result.Error))
		}
	}(items, ttl)

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

	// Invalidate L1 for all keys
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	go func(keyList []string) {
		delResult := <-s.l1Store.Del(context.Background(), keyList...)
		if delResult.Error != nil {
			logx.Error("Failed to invalidate L1 cache in msetAroundAsync", logx.ErrorField(delResult.Error))
		}
	}(keys)

	result <- AsyncResult{}
}
