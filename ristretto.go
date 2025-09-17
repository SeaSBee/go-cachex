package cachex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/seasbee/go-logx"
)

// Store implements a Ristretto-based local hot cache
type RistrettoStore struct {
	// Ristretto cache instance
	cache *ristretto.Cache

	// Configuration
	config *RistrettoConfig

	// Statistics
	stats *RistrettoStats

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Once for initialization
	initOnce sync.Once
}

// Config holds Ristretto configuration
type RistrettoConfig struct {
	// Maximum number of items in cache
	MaxItems int64 `validate:"min:1,max:100000000"`
	// Maximum memory usage in bytes
	MaxMemoryBytes int64 `validate:"min:1048576,max:34359738368"` // 1MB to 32GB
	// Default TTL for items
	DefaultTTL time.Duration `validate:"gte:0,lte:86400000000000"` // 0 to 24h in nanoseconds
	// Number of counters (should be 10x the number of items)
	NumCounters int64 `validate:"min:1,max:1000000000"`
	// Buffer size for items
	BufferItems int64 `validate:"min:1,max:10000"`
	// Cost function for memory calculation
	CostFunction func(value interface{}) int64 `validate:"required"`
	// Enable metrics
	EnableMetrics bool
	// Enable statistics
	EnableStats bool
	// Batch size for concurrent operations (MGet, MSet)
	BatchSize int `validate:"min:1,max:1000"`
}

// DefaultRistrettoConfig returns a default Ristretto configuration
func DefaultRistrettoConfig() *RistrettoConfig {
	return &RistrettoConfig{
		MaxItems:       10000,
		MaxMemoryBytes: 100 * 1024 * 1024, // 100MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    100000, // 10x MaxItems
		BufferItems:    64,
		CostFunction:   defaultCostFunction,
		EnableMetrics:  true,
		EnableStats:    true,
		BatchSize:      10, // Default batch size for concurrent operations
	}
}

// HighPerformanceConfig returns a high-performance configuration
func HighPerformanceConfig() *RistrettoConfig {
	return &RistrettoConfig{
		MaxItems:       100000,
		MaxMemoryBytes: 1 * 1024 * 1024 * 1024, // 1GB
		DefaultTTL:     10 * time.Minute,
		NumCounters:    1000000, // 10x MaxItems
		BufferItems:    128,
		CostFunction:   defaultCostFunction,
		EnableMetrics:  true,
		EnableStats:    true,
		BatchSize:      20, // Larger batch size for high-performance scenarios
	}
}

// ResourceConstrainedConfig returns a configuration for resource-constrained environments
func ResourceConstrainedConfig() *RistrettoConfig {
	return &RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     2 * time.Minute,
		NumCounters:    10000, // 10x MaxItems
		BufferItems:    32,
		CostFunction:   defaultCostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      5, // Smaller batch size for resource-constrained environments
	}
}

// Stats holds Ristretto cache statistics
type RistrettoStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
	Size        int64
	MemoryUsage int64
	mu          sync.RWMutex
}

// validateRistrettoConfig validates the Ristretto configuration
func validateRistrettoConfig(config *RistrettoConfig) error {
	if config.MaxItems <= 0 {
		return fmt.Errorf("MaxItems must be positive, got %d", config.MaxItems)
	}
	if config.MaxMemoryBytes <= 0 {
		return fmt.Errorf("MaxMemoryBytes must be positive, got %d", config.MaxMemoryBytes)
	}
	if config.DefaultTTL < 0 {
		return fmt.Errorf("DefaultTTL cannot be negative, got %v", config.DefaultTTL)
	}
	if config.NumCounters <= 0 {
		return fmt.Errorf("NumCounters must be positive, got %d", config.NumCounters)
	}
	if config.BufferItems <= 0 {
		return fmt.Errorf("BufferItems must be positive, got %d", config.BufferItems)
	}
	if config.CostFunction == nil {
		return fmt.Errorf("CostFunction cannot be nil")
	}
	if config.BatchSize <= 0 {
		return fmt.Errorf("BatchSize must be positive, got %d", config.BatchSize)
	}
	if config.BatchSize > 1000 {
		return fmt.Errorf("BatchSize cannot exceed 1000, got %d", config.BatchSize)
	}
	return nil
}

// NewRistrettoStore creates a new Ristretto store
func NewRistrettoStore(config *RistrettoConfig) (*RistrettoStore, error) {
	if config == nil {
		config = DefaultRistrettoConfig()
	}

	// Validate configuration
	if err := validateRistrettoConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxMemoryBytes,
		BufferItems: config.BufferItems,
		Cost:        config.CostFunction,
		OnEvict: func(item *ristretto.Item) {
			// Handle eviction - track evictions for statistics
			if config.EnableStats {
				// Note: This callback runs in a separate goroutine
				// We'll track evictions in the main operations for better accuracy
			}
		},
		OnReject: func(item *ristretto.Item) {
			// Handle rejection
			logx.Warn("Ristretto cache item rejected",
				logx.String("key", fmt.Sprintf("%d", item.Key)),
				logx.Int64("cost", item.Cost))
		},
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Ristretto cache: %w", err)
	}

	store := &RistrettoStore{
		cache:  cache,
		config: config,
		stats:  &RistrettoStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	// Start background cleanup if needed
	if config.EnableStats {
		go store.startStatsCollection()
	}

	return store, nil
}

// ensureInitialized ensures the store is properly initialized
func (s *RistrettoStore) ensureInitialized() error {
	var initErr error
	s.initOnce.Do(func() {
		// This is a no-op since initialization is done in NewRistrettoStore
		// but provides a hook for future initialization logic
	})
	return initErr
}

// Get retrieves a value from the Ristretto cache (non-blocking)
func (s *RistrettoStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Get from Ristretto cache
		s.mu.RLock()
		value, found := s.cache.Get(key)
		s.mu.RUnlock()

		// Check context cancellation after operation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		if !found || value == nil {
			s.recordMiss()
			result <- AsyncResult{Exists: false}
			return
		}

		// Convert value to bytes
		bytes, ok := value.([]byte)
		if !ok {
			s.recordMiss()
			result <- AsyncResult{Error: fmt.Errorf("invalid value type for key: %s", key)}
			return
		}

		s.recordHit()
		result <- AsyncResult{Value: bytes, Exists: true}
	}()

	return result
}

// Set stores a value in the Ristretto cache (non-blocking)
func (s *RistrettoStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

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

		// Use default TTL if not specified
		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		// Calculate cost
		cost := s.config.CostFunction(value)

		// Set in Ristretto cache with TTL
		s.mu.Lock()
		success := s.cache.SetWithTTL(key, value, cost, ttl)
		s.mu.Unlock()

		if !success {
			result <- AsyncResult{Error: fmt.Errorf("failed to set item in Ristretto cache: %s", key)}
			return
		}

		// Update statistics
		s.updateSize(1)
		result <- AsyncResult{}
	}()

	return result
}

// MGet retrieves multiple values from the Ristretto cache (non-blocking)
func (s *RistrettoStore) MGet(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		batchSize := s.config.BatchSize
		s.mu.RUnlock()

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		values := make(map[string][]byte)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstError error

		// Process all keys concurrently with batched operations
		for i := 0; i < len(keys); i += batchSize {
			end := i + batchSize
			if end > len(keys) {
				end = len(keys)
			}

			batchKeys := make([]string, end-i)
			copy(batchKeys, keys[i:end])

			wg.Add(1)
			go func(batchKeys []string) {
				defer wg.Done()

				// Check context cancellation
				select {
				case <-ctx.Done():
					mu.Lock()
					if firstError == nil {
						firstError = ctx.Err()
					}
					mu.Unlock()
					return
				default:
				}

				// Process batch with single lock acquisition
				s.mu.RLock()
				batchValues := make(map[string][]byte)
				var batchError error

				for _, k := range batchKeys {
					value, found := s.cache.Get(k)
					if found && value != nil {
						if bytes, ok := value.([]byte); ok {
							batchValues[k] = bytes
						} else {
							batchError = fmt.Errorf("invalid value type for key: %s", k)
							break
						}
					}
				}
				s.mu.RUnlock()

				// Handle batch error
				if batchError != nil {
					mu.Lock()
					if firstError == nil {
						firstError = batchError
					}
					mu.Unlock()
					return
				}

				// Update shared results
				if len(batchValues) > 0 {
					mu.Lock()
					for k, v := range batchValues {
						values[k] = v
					}
					mu.Unlock()
				}
			}(batchKeys)
		}

		wg.Wait()

		if firstError != nil {
			result <- AsyncResult{Error: firstError}
			return
		}

		result <- AsyncResult{Values: values}
	}()

	return result
}

// MSet stores multiple values in the Ristretto cache (non-blocking)
func (s *RistrettoStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		batchSize := s.config.BatchSize
		s.mu.RUnlock()

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

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Use default TTL if not specified
		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstError error
		successCount := 0

		// Process all items concurrently with batched operations
		itemEntries := make([]struct {
			key   string
			value []byte
		}, 0, len(items))

		for key, value := range items {
			itemEntries = append(itemEntries, struct {
				key   string
				value []byte
			}{key, value})
		}

		for i := 0; i < len(itemEntries); i += batchSize {
			end := i + batchSize
			if end > len(itemEntries) {
				end = len(itemEntries)
			}

			batchItems := make([]struct {
				key   string
				value []byte
			}, end-i)
			copy(batchItems, itemEntries[i:end])

			wg.Add(1)
			go func(batchItems []struct {
				key   string
				value []byte
			}) {
				defer wg.Done()

				// Check context cancellation
				select {
				case <-ctx.Done():
					mu.Lock()
					if firstError == nil {
						firstError = ctx.Err()
					}
					mu.Unlock()
					return
				default:
				}

				// Process batch with single lock acquisition
				s.mu.Lock()
				batchSuccessCount := 0
				var batchError error

				for _, item := range batchItems {
					cost := s.config.CostFunction(item.value)
					if s.cache.SetWithTTL(item.key, item.value, cost, ttl) {
						batchSuccessCount++
					} else {
						batchError = fmt.Errorf("failed to set item in Ristretto cache: %s", item.key)
						break
					}
				}
				s.mu.Unlock()

				// Handle batch error
				if batchError != nil {
					mu.Lock()
					if firstError == nil {
						firstError = batchError
					}
					mu.Unlock()
					return
				}

				// Update shared success count
				if batchSuccessCount > 0 {
					mu.Lock()
					successCount += batchSuccessCount
					mu.Unlock()
				}
			}(batchItems)
		}

		wg.Wait()

		if firstError != nil {
			result <- AsyncResult{Error: firstError}
			return
		}

		// Update statistics
		s.updateSize(int64(successCount))
		result <- AsyncResult{}
	}()

	return result
}

// Del removes keys from the Ristretto cache (non-blocking)
func (s *RistrettoStore) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Check which keys actually exist before deleting
		existingKeys := make([]string, 0, len(keys))
		s.mu.RLock()
		for _, key := range keys {
			if value, found := s.cache.Get(key); found && value != nil {
				existingKeys = append(existingKeys, key)
			}
		}
		s.mu.RUnlock()

		// Delete only existing keys
		for _, key := range existingKeys {
			s.mu.Lock()
			s.cache.Del(key)
			s.mu.Unlock()
		}

		// Update statistics only for actually deleted keys
		if len(existingKeys) > 0 {
			s.updateSize(-int64(len(existingKeys)))
		}
		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in the Ristretto cache (non-blocking)
func (s *RistrettoStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

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

		s.mu.RLock()
		value, found := s.cache.Get(key)
		s.mu.RUnlock()

		// Consider a key as existing only if it's found and has a non-nil value
		// This is consistent with the Get method's handling of nil values
		exists := found && value != nil
		result <- AsyncResult{Exists: exists}
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking)
// Note: Ristretto doesn't expose TTL information, so this always returns 0
// This is a limitation of the underlying Ristretto library, not this implementation
// Use IsTTLSupported() to check if TTL information is available
func (s *RistrettoStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

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

		// Ristretto doesn't expose TTL information - this is a library limitation
		// We can only check if the key exists, but cannot retrieve its TTL
		s.mu.RLock()
		_, found := s.cache.Get(key)
		s.mu.RUnlock()

		if !found {
			result <- AsyncResult{Exists: false}
			return
		}

		// Key exists but TTL information is not available in Ristretto
		// Return 0 TTL to indicate TTL information is not supported
		result <- AsyncResult{TTL: 0, Exists: true}
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (s *RistrettoStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if store is properly initialized
		s.mu.RLock()
		if s.cache == nil {
			s.mu.RUnlock()
			result <- AsyncResult{Error: fmt.Errorf("cache not initialized")}
			return
		}
		s.mu.RUnlock()

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

		// Get current value
		s.mu.RLock()
		currentValue, found := s.cache.Get(key)
		s.mu.RUnlock()

		var current int64

		if found {
			if currentValue == nil {
				result <- AsyncResult{Error: fmt.Errorf("nil value found for key: %s", key)}
				return
			}
			if bytes, ok := currentValue.([]byte); ok {
				// Try to parse as int64 - validate length first
				if len(bytes) == 8 {
					current = int64(bytes[0])<<56 | int64(bytes[1])<<48 | int64(bytes[2])<<40 | int64(bytes[3])<<32 |
						int64(bytes[4])<<24 | int64(bytes[5])<<16 | int64(bytes[6])<<8 | int64(bytes[7])
				} else {
					result <- AsyncResult{Error: fmt.Errorf("invalid int64 value for key: %s", key)}
					return
				}
			} else {
				result <- AsyncResult{Error: fmt.Errorf("invalid value type for key: %s", key)}
				return
			}
		}

		// Check for integer overflow/underflow
		// For positive delta: ensure current + delta <= maxInt64
		// For negative delta: ensure current + delta >= minInt64
		if delta > 0 {
			maxInt64 := int64(1<<63 - 1)
			if current > maxInt64-delta {
				result <- AsyncResult{Error: fmt.Errorf("integer overflow for key: %s (current: %d, delta: %d)", key, current, delta)}
				return
			}
		} else if delta < 0 {
			minInt64 := int64(-(1 << 63))
			if current < minInt64-delta {
				result <- AsyncResult{Error: fmt.Errorf("integer underflow for key: %s (current: %d, delta: %d)", key, current, delta)}
				return
			}
		}

		// Calculate new value
		newValue := current + delta

		// Convert to bytes
		bytes := make([]byte, 8)
		bytes[0] = byte(newValue >> 56)
		bytes[1] = byte(newValue >> 48)
		bytes[2] = byte(newValue >> 40)
		bytes[3] = byte(newValue >> 32)
		bytes[4] = byte(newValue >> 24)
		bytes[5] = byte(newValue >> 16)
		bytes[6] = byte(newValue >> 8)
		bytes[7] = byte(newValue)

		// Set new value
		ttl := s.config.DefaultTTL
		if !found && ttlIfCreate > 0 {
			ttl = ttlIfCreate
		}

		setResult := <-s.Set(ctx, key, bytes, ttl)
		if setResult.Error != nil {
			result <- AsyncResult{Error: setResult.Error}
			return
		}

		result <- AsyncResult{Result: newValue}
	}()

	return result
}

// Close closes the Ristretto store
func (s *RistrettoStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already closed
	if s.cache == nil && s.cancel == nil {
		return nil
	}

	// Cancel context first to stop background goroutines
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	// Close cache
	if s.cache != nil {
		s.cache.Close()
		s.cache = nil
	}

	logx.Info("Closed Ristretto store")
	return nil
}

// GetStats returns Ristretto cache statistics
// Note: Some statistics (like evictions) are not directly exposed by Ristretto
// and are tracked separately where possible. Memory usage is approximated
// based on cost metrics from Ristretto.
func (s *RistrettoStore) GetStats() *RistrettoStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if cache is initialized
	if s.cache == nil {
		return &RistrettoStats{}
	}

	// Get Ristretto metrics
	metrics := s.cache.Metrics

	stats := &RistrettoStats{
		Hits:        int64(metrics.Hits()),
		Misses:      int64(metrics.Misses()),
		Evictions:   s.stats.Evictions, // Ristretto doesn't expose evictions count
		Expirations: s.stats.Expirations,
		Size:        s.stats.Size,
		MemoryUsage: s.stats.MemoryUsage,
	}

	return stats
}

// GetConfig returns a copy of the current configuration
// This is useful for monitoring and debugging purposes
func (s *RistrettoStore) GetConfig() *RistrettoConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	config := *s.config
	return &config
}

// IsTTLSupported returns false since Ristretto doesn't expose TTL information
// This method helps clients understand the limitations of this implementation
// Ristretto manages TTL internally but doesn't provide access to TTL values
func (s *RistrettoStore) IsTTLSupported() bool {
	return false
}

// GetTTLLimitationInfo returns information about TTL limitations
// This helps clients understand why TTL operations return 0
func (s *RistrettoStore) GetTTLLimitationInfo() string {
	return "Ristretto manages TTL internally but doesn't expose TTL information. TTL values are always returned as 0."
}

// recordHit records a cache hit
func (s *RistrettoStore) recordHit() {
	s.stats.mu.Lock()
	s.mu.RLock()
	enableStats := s.config.EnableStats
	s.mu.RUnlock()

	if enableStats {
		s.stats.Hits++
	}
	s.stats.mu.Unlock()
}

// recordMiss records a cache miss
func (s *RistrettoStore) recordMiss() {
	s.stats.mu.Lock()
	s.mu.RLock()
	enableStats := s.config.EnableStats
	s.mu.RUnlock()

	if enableStats {
		s.stats.Misses++
	}
	s.stats.mu.Unlock()
}

// updateSize updates the cache size statistics
func (s *RistrettoStore) updateSize(delta int64) {
	s.stats.mu.Lock()
	s.mu.RLock()
	enableStats := s.config.EnableStats
	s.mu.RUnlock()

	if enableStats {
		s.stats.Size += delta
		if s.stats.Size < 0 {
			s.stats.Size = 0
		}
	}
	s.stats.mu.Unlock()
}

// startStatsCollection starts background statistics collection
func (s *RistrettoStore) startStatsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logx.Debug("Starting Ristretto stats collection goroutine")
	defer logx.Debug("Ristretto stats collection goroutine stopped")

	for {
		select {
		case <-s.ctx.Done():
			logx.Debug("Stats collection goroutine shutting down")
			return
		case <-ticker.C:
			// Update memory usage statistics
			// Use consistent lock ordering: stats.mu first, then s.mu
			s.stats.mu.Lock()
			s.mu.RLock()
			if s.cache != nil {
				metrics := s.cache.Metrics
				// Calculate memory usage as cost added minus cost evicted
				// This gives us an approximation of current memory usage
				s.stats.MemoryUsage = int64(metrics.CostAdded() - metrics.CostEvicted())
				if s.stats.MemoryUsage < 0 {
					s.stats.MemoryUsage = 0 // Ensure non-negative
				}
			}
			s.mu.RUnlock()
			s.stats.mu.Unlock()
		}
	}
}

// defaultCostFunction calculates the cost of a value in bytes
// This function is used by Ristretto to determine memory usage for eviction decisions
func defaultCostFunction(value interface{}) int64 {
	if value == nil {
		return 1 // Default cost for nil values (minimum cost)
	}
	if bytes, ok := value.([]byte); ok {
		return int64(len(bytes)) // Cost equals the actual byte length
	}
	return 1 // Default cost for non-byte values (minimum cost)
}
