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
}

// Config holds Ristretto configuration
type RistrettoConfig struct {
	// Maximum number of items in cache
	MaxItems int64
	// Maximum memory usage in bytes
	MaxMemoryBytes int64
	// Default TTL for items
	DefaultTTL time.Duration
	// Number of counters (should be 10x the number of items)
	NumCounters int64
	// Buffer size for items
	BufferItems int64
	// Cost function for memory calculation
	CostFunction func(value interface{}) int64
	// Enable metrics
	EnableMetrics bool
	// Enable statistics
	EnableStats bool
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

// NewRistrettoStore creates a new Ristretto store
func NewRistrettoStore(config *RistrettoConfig) (*RistrettoStore, error) {
	if config == nil {
		config = DefaultRistrettoConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxMemoryBytes,
		BufferItems: config.BufferItems,
		Cost:        config.CostFunction,
		OnEvict: func(item *ristretto.Item) {
			// Handle eviction
			if config.EnableStats {
				// Stats will be updated in the main operations
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

// Get retrieves a value from the Ristretto cache (non-blocking)
func (s *RistrettoStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

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

		// Get from Ristretto cache
		value, found := s.cache.Get(key)
		if !found {
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
		success := s.cache.SetWithTTL(key, value, cost, ttl)
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

		for _, key := range keys {
			getResult := <-s.Get(ctx, key)
			if getResult.Error != nil {
				result <- AsyncResult{Error: getResult.Error}
				return
			}
			if getResult.Exists && getResult.Value != nil {
				values[key] = getResult.Value
			}
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

		for key, value := range items {
			setResult := <-s.Set(ctx, key, value, ttl)
			if setResult.Error != nil {
				result <- AsyncResult{Error: fmt.Errorf("failed to set %s: %w", key, setResult.Error)}
				return
			}
		}

		result <- AsyncResult{}
	}()

	return result
}

// Del removes keys from the Ristretto cache (non-blocking)
func (s *RistrettoStore) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
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

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		for _, key := range keys {
			s.cache.Del(key)
		}

		// Update statistics
		s.updateSize(-int64(len(keys)))
		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in the Ristretto cache (non-blocking)
func (s *RistrettoStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

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

		value, found := s.cache.Get(key)
		result <- AsyncResult{Exists: found && value != nil}
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking) - Ristretto doesn't expose TTL
func (s *RistrettoStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

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

		// Ristretto doesn't expose TTL information
		// We return 0 to indicate TTL is not available
		_, found := s.cache.Get(key)
		if !found {
			result <- AsyncResult{Exists: false}
			return
		}

		result <- AsyncResult{TTL: 0, Exists: true} // TTL not available in Ristretto
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (s *RistrettoStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
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

		// Check context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Get current value
		currentValue, found := s.cache.Get(key)
		var current int64

		if found {
			if bytes, ok := currentValue.([]byte); ok {
				// Try to parse as int64
				if len(bytes) == 8 {
					current = int64(bytes[0])<<56 | int64(bytes[1])<<48 | int64(bytes[2])<<40 | int64(bytes[3])<<32 |
						int64(bytes[4])<<24 | int64(bytes[5])<<16 | int64(bytes[6])<<8 | int64(bytes[7])
				}
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
	s.cancel()
	s.cache.Close()
	logx.Info("Closed Ristretto store")
	return nil
}

// GetStats returns Ristretto cache statistics
func (s *RistrettoStore) GetStats() *RistrettoStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

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

// recordHit records a cache hit
func (s *RistrettoStore) recordHit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Hits++
		s.stats.mu.Unlock()
	}
}

// recordMiss records a cache miss
func (s *RistrettoStore) recordMiss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Misses++
		s.stats.mu.Unlock()
	}
}

// updateSize updates the cache size statistics
func (s *RistrettoStore) updateSize(delta int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Size += delta
		if s.stats.Size < 0 {
			s.stats.Size = 0
		}
		s.stats.mu.Unlock()
	}
}

// startStatsCollection starts background statistics collection
func (s *RistrettoStore) startStatsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Update memory usage statistics
			metrics := s.cache.Metrics
			s.stats.mu.Lock()
			s.stats.MemoryUsage = int64(metrics.CostAdded() - metrics.CostEvicted())
			s.stats.mu.Unlock()
		}
	}
}

// defaultCostFunction calculates the cost of a value in bytes
func defaultCostFunction(value interface{}) int64 {
	if bytes, ok := value.([]byte); ok {
		return int64(len(bytes))
	}
	return 1 // Default cost for non-byte values
}
