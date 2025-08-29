package cachex

/*
Package cachex provides a comprehensive in-memory cache implementation with the following recent improvements:

CRITICAL BUG FIXES:
1. Fixed critical race condition in MSet method where capacity calculation was incorrect
2. Fixed memory leak in MGet method by properly managing pooled map resources
3. Added comprehensive nil checks for GlobalPools to prevent panic during initialization
4. Fixed memory calculation bug in MSet where eviction condition was incorrect
5. Added proper error handling in IncrBy method for non-integer values
6. Standardized channel creation patterns across all methods
7. Added nil checks before using memory pools to prevent panics
8. Fixed potential index out of bounds issues in access order management
9. Improved Close method with proper timeout handling
10. Standardized nil handling patterns across all methods

MEMORY SAFETY IMPROVEMENTS:
- All memory pool operations now have nil safety checks
- Proper cleanup of pooled resources to prevent memory leaks
- Consistent error handling and resource management
- Bounds checking for all slice and map operations

PERFORMANCE IMPROVEMENTS:
- Memory pool integration with nil safety
- Optimized value copying to reduce allocations
- Efficient eviction algorithms (LRU, LFU, TTL)
- Context-aware operations with proper cancellation support
- Comprehensive bounds checking and error handling

THREAD SAFETY:
- All operations are properly synchronized with mutexes
- Context cancellation support in all async operations
- Proper cleanup of background goroutines
*/

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// Store implements the cache.Store interface using in-memory storage
type MemoryStore struct {
	// Configuration
	config *MemoryConfig

	// Data storage
	data map[string]*cacheItem
	mu   sync.RWMutex

	// Eviction tracking
	accessOrder []string       // LRU order
	accessMap   map[string]int // key -> position in accessOrder

	// Statistics
	stats *MemoryStats

	// Background cleanup
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup

	// Memory pools for performance optimization
	cacheItemPool   *CacheItemPool
	asyncResultPool *AsyncResultPool
	bufferPool      *BufferPool
	stringSlicePool *StringSlicePool
	mapPool         *MapPool
}

// Config defines memory store configuration
type MemoryConfig struct {
	// Capacity limits
	MaxSize     int `yaml:"max_size" json:"max_size" validate:"min:1,max:10000000"`        // Maximum number of items
	MaxMemoryMB int `yaml:"max_memory_mb" json:"max_memory_mb" validate:"min:1,max:32768"` // Maximum memory usage in MB (approximate)

	// TTL settings
	DefaultTTL      time.Duration `yaml:"default_ttl" json:"default_ttl" validate:"gte:0,lte:86400000000000"`                   // Default TTL for items (0 to 24h in nanoseconds)
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" validate:"gte:1000000000,lte:3600000000000"` // How often to run cleanup (1s to 1h in nanoseconds)

	// Eviction policy
	EvictionPolicy EvictionPolicy `yaml:"eviction_policy" json:"eviction_policy" validate:"omitempty,oneof:lru lfu ttl"` // LRU, LFU, or TTL-based

	// Performance tuning
	EnableStats bool `yaml:"enable_stats" json:"enable_stats"` // Enable detailed statistics
}

// EvictionPolicy defines how items are evicted when capacity is reached
type EvictionPolicy string

const (
	EvictionPolicyLRU EvictionPolicy = "lru" // Least Recently Used
	EvictionPolicyLFU EvictionPolicy = "lfu" // Least Frequently Used
	EvictionPolicyTTL EvictionPolicy = "ttl" // Time To Live (oldest first)
)

// DefaultMemoryConfig returns a default configuration
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  EvictionPolicyLRU,
		EnableStats:     true,
	}
}

// HighPerformanceMemoryConfig returns a configuration optimized for high throughput
func HighPerformanceMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxSize:         50000,
		MaxMemoryMB:     500,
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 30 * time.Second,
		EvictionPolicy:  EvictionPolicyLRU,
		EnableStats:     true,
	}
}

// ResourceConstrainedMemoryConfig returns a configuration optimized for resource-constrained environments
func ResourceConstrainedMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxSize:         1000,
		MaxMemoryMB:     10,
		DefaultTTL:      2 * time.Minute,
		CleanupInterval: 2 * time.Minute,
		EvictionPolicy:  EvictionPolicyTTL,
		EnableStats:     false,
	}
}

// cacheItem represents a cached item with metadata
type cacheItem struct {
	Value       []byte
	ExpiresAt   time.Time
	AccessCount int64
	LastAccess  time.Time
	Size        int // Size in bytes (including struct overhead)
}

// calculateItemSize calculates the total size of a cache item including struct overhead
func calculateItemSize(value []byte) int {
	// Base struct size (approximate)
	structOverhead := 64 // time.Time (24 bytes) + int64 (8 bytes) + int64 (8 bytes) + time.Time (24 bytes) = ~64 bytes
	// Add value size
	return structOverhead + len(value)
}

// Stats holds memory store statistics
type MemoryStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
	Size        int64
	MemoryUsage int64 // Approximate memory usage in bytes
	mu          sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(config *MemoryConfig) (*MemoryStore, error) {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	// Validate configuration
	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("MaxSize must be positive, got %d", config.MaxSize)
	}
	if config.MaxMemoryMB < 0 {
		return nil, fmt.Errorf("MaxMemoryMB cannot be negative, got %d", config.MaxMemoryMB)
	}
	if config.DefaultTTL < 0 {
		return nil, fmt.Errorf("DefaultTTL cannot be negative, got %v", config.DefaultTTL)
	}
	if config.CleanupInterval <= 0 {
		return nil, fmt.Errorf("CleanupInterval must be positive, got %v", config.CleanupInterval)
	}
	if config.EvictionPolicy != EvictionPolicyLRU && config.EvictionPolicy != EvictionPolicyLFU && config.EvictionPolicy != EvictionPolicyTTL {
		return nil, fmt.Errorf("EvictionPolicy must be one of: %s, %s, %s, got %s", EvictionPolicyLRU, EvictionPolicyLFU, EvictionPolicyTTL, config.EvictionPolicy)
	}

	store := &MemoryStore{
		config:      config,
		data:        make(map[string]*cacheItem),
		accessOrder: make([]string, 0),
		accessMap:   make(map[string]int),
		stats:       &MemoryStats{},
		stopChan:    make(chan struct{}),
	}

	// Initialize memory pools with nil checks to prevent panics
	if GlobalPools.CacheItem != nil {
		store.cacheItemPool = GlobalPools.CacheItem
	}
	if GlobalPools.AsyncResult != nil {
		store.asyncResultPool = GlobalPools.AsyncResult
	}
	if GlobalPools.Buffer != nil {
		store.bufferPool = GlobalPools.Buffer
	}
	if GlobalPools.StringSlice != nil {
		store.stringSlicePool = GlobalPools.StringSlice
	}
	if GlobalPools.Map != nil {
		store.mapPool = GlobalPools.Map
	}

	// Ensure stats is properly initialized
	if store.stats == nil {
		store.stats = &MemoryStats{}
	}

	// Start background cleanup
	store.startCleanup()

	return store, nil
}

// Get retrieves a value from the memory store (non-blocking)
func (s *MemoryStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		s.mu.Lock()
		item, exists := s.data[key]
		if !exists {
			s.mu.Unlock()
			s.recordMiss()
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired
		now := time.Now()
		expired := now.After(item.ExpiresAt)

		if expired {
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.mu.Unlock()
			s.recordExpiration()
			result <- AsyncResult{Exists: false}
			return
		}

		// Use optimized value copy to reduce allocations
		value := OptimizedValueCopy(item.Value)

		// Update access statistics
		item.AccessCount++
		item.LastAccess = now
		s.updateAccessOrder(key)

		s.mu.Unlock()

		s.recordHit()
		result <- AsyncResult{Value: value, Exists: true}
	}()

	return result
}

// Set stores a value in the memory store (non-blocking)
func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
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

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		expiresAt := time.Now().Add(ttl)
		// Use memory pool for cacheItem allocation with nil check
		var item *cacheItem
		if s.cacheItemPool != nil {
			item = s.cacheItemPool.Get()
		} else {
			item = &cacheItem{}
		}
		item.Value = value
		item.ExpiresAt = expiresAt
		item.AccessCount = 1
		item.LastAccess = time.Now()
		item.Size = calculateItemSize(value)

		s.mu.Lock()
		defer s.mu.Unlock()

		// Check if key already exists
		if existing, exists := s.data[key]; exists {
			// Update existing item
			s.removeFromAccessOrder(key)
			s.data[key] = item
			s.addToAccessOrder(key)
			s.updateStats(existing.Size, item.Size)
			result <- AsyncResult{}
			return
		}

		// Check capacity before adding new item
		if len(s.data) >= s.config.MaxSize {
			s.evictItems(1)
		}

		// Check memory limit if configured
		if s.config.MaxMemoryMB > 0 {
			requiredMemoryBytes := s.stats.MemoryUsage + int64(item.Size)
			maxMemoryBytes := int64(s.config.MaxMemoryMB) * 1024 * 1024

			if requiredMemoryBytes > maxMemoryBytes {
				// Evict items until we have enough space
				for s.stats.MemoryUsage+int64(item.Size) > maxMemoryBytes && len(s.data) > 0 {
					s.evictItems(1)
					// Break if we can't free more memory
					if s.stats.MemoryUsage+int64(item.Size) <= maxMemoryBytes {
						break
					}
				}
			}
		}

		// Add new item
		s.data[key] = item
		s.addToAccessOrder(key)
		s.updateStats(0, item.Size)

		result <- AsyncResult{}
	}()

	return result
}

// MGet retrieves multiple values from the memory store (non-blocking)
func (s *MemoryStore) MGet(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
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

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		// Use memory pool for map allocation with nil check
		var values map[string][]byte
		if s.mapPool != nil {
			values = s.mapPool.Get()
		} else {
			values = make(map[string][]byte)
		}
		now := time.Now()

		s.mu.Lock()
		for _, key := range keys {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				s.mu.Unlock()
				result <- AsyncResult{Error: ctx.Err()}
				return
			default:
			}

			if item, exists := s.data[key]; exists {
				if now.After(item.ExpiresAt) {
					// Item expired, remove it
					delete(s.data, key)
					s.removeFromAccessOrder(key)
					s.updateStats(item.Size, 0)
					s.recordExpiration()
					continue
				}
				// Use optimized value copy to reduce allocations
				value := OptimizedValueCopy(item.Value)
				values[key] = value

				// Update access statistics
				item.AccessCount++
				item.LastAccess = now
				s.updateAccessOrder(key)
			}
		}
		s.mu.Unlock()

		// Create a copy of the map to avoid returning pooled memory to caller
		resultMap := make(map[string][]byte, len(values))
		for k, v := range values {
			resultMap[k] = v
		}

		// Return the map to the pool since we're not using it anymore
		if s.mapPool != nil {
			s.mapPool.Put(values)
		}

		result <- AsyncResult{Values: resultMap}
	}()

	return result
}

// MSet stores multiple values in the memory store (non-blocking)
func (s *MemoryStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
			return
		}

		// Boundary condition validations
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		if len(items) == 0 {
			result <- AsyncResult{}
			return
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
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

		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		expiresAt := time.Now().Add(ttl)
		now := time.Now()

		s.mu.Lock()
		defer s.mu.Unlock()

		// Check capacity and evict if necessary
		needed := 0
		for key := range items {
			if _, exists := s.data[key]; !exists {
				needed++
			}
		}

		if len(s.data)+needed > s.config.MaxSize {
			s.evictItems(needed)
		}

		// Check memory limit if configured
		if s.config.MaxMemoryMB > 0 {
			totalNewSize := int64(0)
			for _, value := range items {
				totalNewSize += int64(calculateItemSize(value))
			}

			// Calculate required memory in bytes
			requiredMemoryBytes := s.stats.MemoryUsage + totalNewSize
			maxMemoryBytes := int64(s.config.MaxMemoryMB) * 1024 * 1024

			if requiredMemoryBytes > maxMemoryBytes {
				// Evict items until we have enough space
				for s.stats.MemoryUsage+totalNewSize > maxMemoryBytes && len(s.data) > 0 {
					s.evictItems(1)
					// Break if we can't free more memory
					if s.stats.MemoryUsage+totalNewSize <= maxMemoryBytes {
						break
					}
				}
			}
		}

		// Add/update items
		for key, value := range items {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				result <- AsyncResult{Error: ctx.Err()}
				return
			default:
			}

			// Use memory pool for cacheItem allocation with nil check
			var item *cacheItem
			if s.cacheItemPool != nil {
				item = s.cacheItemPool.Get()
			} else {
				item = &cacheItem{}
			}
			item.Value = value
			item.ExpiresAt = expiresAt
			item.AccessCount = 1
			item.LastAccess = now
			item.Size = calculateItemSize(value)

			if existing, exists := s.data[key]; exists {
				s.removeFromAccessOrder(key)
				s.updateStats(existing.Size, item.Size)
			} else {
				s.updateStats(0, item.Size)
			}

			s.data[key] = item
			s.addToAccessOrder(key)
		}

		result <- AsyncResult{}
	}()

	return result
}

// Del removes keys from the memory store (non-blocking)
func (s *MemoryStore) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
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

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		for _, key := range keys {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				result <- AsyncResult{Error: ctx.Err()}
				return
			default:
			}

			if item, exists := s.data[key]; exists {
				delete(s.data, key)
				s.removeFromAccessOrder(key)
				s.updateStats(item.Size, 0)

				// Return the item to the memory pool with nil check
				if s.cacheItemPool != nil {
					s.cacheItemPool.Put(item)
				}
			}
		}

		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in the memory store (non-blocking)
func (s *MemoryStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		s.mu.Lock()
		item, exists := s.data[key]
		if !exists {
			s.mu.Unlock()
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired
		if time.Now().After(item.ExpiresAt) {
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.mu.Unlock()
			s.recordExpiration()

			// Return the item to the memory pool with nil check
			if s.cacheItemPool != nil {
				s.cacheItemPool.Put(item)
			}

			result <- AsyncResult{Exists: false}
			return
		}

		s.mu.Unlock()
		result <- AsyncResult{Exists: true}
	}()

	return result
}

// TTL returns the time to live for a key (non-blocking)
func (s *MemoryStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
			return
		}

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		s.mu.Lock()
		item, exists := s.data[key]
		if !exists {
			s.mu.Unlock()
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired
		if time.Now().After(item.ExpiresAt) {
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.mu.Unlock()
			s.recordExpiration()

			// Return the item to the memory pool with nil check
			if s.cacheItemPool != nil {
				s.cacheItemPool.Put(item)
			}

			result <- AsyncResult{Exists: false}
			return
		}

		ttl := time.Until(item.ExpiresAt)
		s.mu.Unlock()
		result <- AsyncResult{TTL: ttl, Exists: true}
	}()

	return result
}

// IncrBy increments a counter by the specified delta (non-blocking)
func (s *MemoryStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
	result := OptimizedResultChannel()

	go func() {
		defer close(result)

		// Check if store is properly initialized
		if s == nil {
			result <- AsyncResult{Error: fmt.Errorf("store is not initialized")}
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

		// Check for context cancellation
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		default:
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		// Try to get existing value
		item, exists := s.data[key]
		var currentValue int64

		if exists {
			// Check if item has expired
			if time.Now().After(item.ExpiresAt) {
				exists = false
			} else {
				// Parse existing value
				if len(item.Value) > 0 {
					if val, err := strconv.ParseInt(string(item.Value), 10, 64); err == nil {
						currentValue = val
					} else {
						// If parsing fails, treat as an error - the value is not a valid integer
						result <- AsyncResult{Error: fmt.Errorf("cannot increment non-integer value for key %s: %v", key, err)}
						return
					}
				}
			}
		}

		// Calculate new value with overflow check
		if delta > 0 && currentValue > (1<<63-1)-delta {
			// Overflow would occur
			result <- AsyncResult{Error: fmt.Errorf("integer overflow: %d + %d would exceed int64 maximum", currentValue, delta)}
			return
		}
		if delta < 0 && currentValue < -(1<<63)-delta {
			// Underflow would occur
			result <- AsyncResult{Error: fmt.Errorf("integer underflow: %d + %d would exceed int64 minimum", currentValue, delta)}
			return
		}
		newValue := currentValue + delta

		// Create or update item
		expiresAt := time.Now().Add(ttlIfCreate)
		if exists {
			expiresAt = item.ExpiresAt
		}

		// Use memory pool for cacheItem allocation with nil check
		var newItem *cacheItem
		if s.cacheItemPool != nil {
			newItem = s.cacheItemPool.Get()
		} else {
			newItem = &cacheItem{}
		}
		newItem.Value = []byte(strconv.FormatInt(newValue, 10))
		newItem.ExpiresAt = expiresAt
		newItem.AccessCount = 1
		newItem.LastAccess = time.Now()
		newItem.Size = calculateItemSize([]byte(strconv.FormatInt(newValue, 10)))

		if existing, exists := s.data[key]; exists {
			s.removeFromAccessOrder(key)
			s.updateStats(existing.Size, newItem.Size)
		} else {
			s.updateStats(0, newItem.Size)
		}

		s.data[key] = newItem
		s.addToAccessOrder(key)

		result <- AsyncResult{Result: newValue}
	}()

	return result
}

// Close closes the memory store and releases resources
func (s *MemoryStore) Close() error {
	// Check if already closed
	select {
	case <-s.stopChan:
		// Already closed
		return nil
	default:
		close(s.stopChan)
	}

	// Stop the cleanup ticker if it exists
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	// Wait for cleanup goroutine with timeout and better error handling
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Successfully closed
		return nil
	case <-time.After(5 * time.Second):
		// Log warning but don't fail - the goroutine might be stuck
		// but we've closed the stop channel so it should eventually terminate
		// We could add logging here if needed
		return fmt.Errorf("timeout waiting for cleanup goroutine to finish - resources may not be fully cleaned up")
	}
}

// GetStats returns current statistics
func (s *MemoryStore) GetStats() *MemoryStats {
	// Check if store is properly initialized
	if s == nil {
		return nil
	}

	// First get the data size while holding the data mutex
	s.mu.RLock()
	dataSize := int64(len(s.data))
	s.mu.RUnlock()

	// Check if stats is properly initialized
	if s.stats == nil {
		return &MemoryStats{Size: dataSize}
	}

	// Then get the stats while holding the stats mutex
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// Return a copy to prevent race conditions
	statsCopy := &MemoryStats{
		Hits:        s.stats.Hits,
		Misses:      s.stats.Misses,
		Evictions:   s.stats.Evictions,
		Expirations: s.stats.Expirations,
		Size:        dataSize,
		MemoryUsage: s.stats.MemoryUsage,
	}

	return statsCopy
}

// Clear removes all items from the store
func (s *MemoryStore) Clear() {
	// Check if store is properly initialized
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*cacheItem)
	s.accessOrder = make([]string, 0)
	s.accessMap = make(map[string]int)

	// Check if stats is properly initialized
	if s.stats != nil {
		s.stats.mu.Lock()
		s.stats.Size = 0
		s.stats.MemoryUsage = 0
		s.stats.mu.Unlock()
	}
}

// startCleanup starts the background cleanup goroutine
func (s *MemoryStore) startCleanup() {
	// Validate cleanup interval
	if s.config.CleanupInterval <= 0 {
		// Use a reasonable default if invalid
		s.config.CleanupInterval = 1 * time.Minute
	}

	s.cleanupTicker = time.NewTicker(s.config.CleanupInterval)
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		defer s.cleanupTicker.Stop()

		for {
			select {
			case <-s.cleanupTicker.C:
				// Use a context with timeout for cleanup to prevent blocking
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				s.cleanupWithContext(ctx)
				cancel()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// cleanup removes expired items from the store
func (s *MemoryStore) cleanup() {
	s.cleanupWithContext(context.Background())
}

// cleanupWithContext removes expired items from the store with context support
func (s *MemoryStore) cleanupWithContext(ctx context.Context) {
	now := time.Now()
	expiredKeys := make([]string, 0)

	s.mu.RLock()
	for key, item := range s.data {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			s.mu.RUnlock()
			return
		default:
		}

		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.mu.RUnlock()

	if len(expiredKeys) > 0 {
		s.mu.Lock()
		for _, key := range expiredKeys {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				s.mu.Unlock()
				return
			default:
			}

			if item, exists := s.data[key]; exists {
				delete(s.data, key)
				s.removeFromAccessOrder(key)
				s.updateStats(item.Size, 0)
				s.recordExpiration()

				// Return the item to the memory pool with nil check
				if s.cacheItemPool != nil {
					s.cacheItemPool.Put(item)
				}
			}
		}
		s.mu.Unlock()

		if s.config.EnableStats {
			logx.Debug("Memory store cleanup", logx.Int("expired", len(expiredKeys)))
		}
	}
}

// evictItems evicts items based on the configured eviction policy
func (s *MemoryStore) evictItems(count int) {
	if len(s.data) == 0 {
		return
	}

	keysToEvict := make([]string, 0, count)

	switch s.config.EvictionPolicy {
	case EvictionPolicyLRU:
		// Evict least recently used (first in access order)
		for i := 0; i < count && i < len(s.accessOrder); i++ {
			keysToEvict = append(keysToEvict, s.accessOrder[i])
		}

	case EvictionPolicyLFU:
		// Evict least frequently used items using efficient approach
		if len(s.data) == 0 {
			break
		}

		// Use a more efficient approach: collect all items and sort by access count
		type itemInfo struct {
			key   string
			count int64
		}

		items := make([]itemInfo, 0, len(s.data))
		for key, item := range s.data {
			items = append(items, itemInfo{key: key, count: item.AccessCount})
		}

		// Sort by access count (ascending) using Go's built-in sort
		// This is much more efficient than bubble sort
		sort.Slice(items, func(i, j int) bool {
			return items[i].count < items[j].count
		})

		// Take the first 'count' items (lowest access counts)
		for i := 0; i < count && i < len(items); i++ {
			keysToEvict = append(keysToEvict, items[i].key)
		}

	case EvictionPolicyTTL:
		// Evict items with shortest TTL remaining using efficient approach
		if len(s.data) == 0 {
			break
		}

		// Find the earliest expiration time
		earliestExpiry := time.Now().Add(time.Hour * 24 * 365) // Far future
		for _, item := range s.data {
			if item.ExpiresAt.Before(earliestExpiry) {
				earliestExpiry = item.ExpiresAt
			}
		}

		// Collect all items with earliest expiration time
		earliestKeys := make([]string, 0)
		for key, item := range s.data {
			if item.ExpiresAt.Equal(earliestExpiry) {
				earliestKeys = append(earliestKeys, key)
			}
		}

		// Take up to 'count' items from the earliest expiration group
		for i := 0; i < count && i < len(earliestKeys); i++ {
			keysToEvict = append(keysToEvict, earliestKeys[i])
		}
	}

	// Evict the selected keys
	for _, key := range keysToEvict {
		if item, exists := s.data[key]; exists {
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.updateStats(item.Size, 0)
			s.recordEviction()

			// Return the item to the memory pool with nil check
			if s.cacheItemPool != nil {
				s.cacheItemPool.Put(item)
			}
		}
	}
}

// addToAccessOrder adds a key to the access order (LRU tracking)
func (s *MemoryStore) addToAccessOrder(key string) {
	s.accessOrder = append(s.accessOrder, key)
	s.accessMap[key] = len(s.accessOrder) - 1
}

// removeFromAccessOrder removes a key from the access order
func (s *MemoryStore) removeFromAccessOrder(key string) {
	if pos, exists := s.accessMap[key]; exists {
		// Bounds checking
		if pos < 0 || pos >= len(s.accessOrder) {
			// Invalid position, remove from map and clean up any orphaned entries
			delete(s.accessMap, key)
			// Clean up any orphaned entries in accessOrder
			s.cleanupOrphanedAccessOrder()
			return
		}

		// Remove from slice
		if pos == len(s.accessOrder)-1 {
			// Last element, just truncate
			s.accessOrder = s.accessOrder[:pos]
		} else {
			// Remove element and shift
			s.accessOrder = append(s.accessOrder[:pos], s.accessOrder[pos+1:]...)
		}
		delete(s.accessMap, key)

		// Update positions for keys after the removed one
		for i := pos; i < len(s.accessOrder); i++ {
			s.accessMap[s.accessOrder[i]] = i
		}
	}
}

// cleanupOrphanedAccessOrder removes orphaned entries from accessOrder that don't exist in accessMap
func (s *MemoryStore) cleanupOrphanedAccessOrder() {
	// Create a new slice with only valid entries
	validOrder := make([]string, 0, len(s.accessOrder))
	for _, key := range s.accessOrder {
		if _, exists := s.accessMap[key]; exists {
			validOrder = append(validOrder, key)
		}
	}
	s.accessOrder = validOrder

	// Rebuild the accessMap positions
	for i, key := range s.accessOrder {
		s.accessMap[key] = i
	}
}

// updateAccessOrder moves a key to the end of the access order (most recently used)
func (s *MemoryStore) updateAccessOrder(key string) {
	if pos, exists := s.accessMap[key]; exists {
		// Bounds checking
		if pos < 0 || pos >= len(s.accessOrder) {
			// Invalid position, just remove from map and add to end
			delete(s.accessMap, key)
			s.accessOrder = append(s.accessOrder, key)
			s.accessMap[key] = len(s.accessOrder) - 1
			return
		}

		// If it's already the last element, no need to move
		if pos == len(s.accessOrder)-1 {
			return
		}

		// Remove from current position
		if pos == 0 {
			// First element, just shift
			s.accessOrder = s.accessOrder[1:]
		} else {
			// Remove element and shift
			s.accessOrder = append(s.accessOrder[:pos], s.accessOrder[pos+1:]...)
		}

		// Update positions for keys after the removed one
		for i := pos; i < len(s.accessOrder); i++ {
			s.accessMap[s.accessOrder[i]] = i
		}

		// Add to end
		s.accessOrder = append(s.accessOrder, key)
		s.accessMap[key] = len(s.accessOrder) - 1
	} else {
		// Key doesn't exist in access order, just add it
		s.accessOrder = append(s.accessOrder, key)
		s.accessMap[key] = len(s.accessOrder) - 1
	}
}

// updateStats updates memory usage statistics
func (s *MemoryStore) updateStats(oldSize, newSize int) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.MemoryUsage += int64(newSize - oldSize)
		s.stats.mu.Unlock()
	}
}

// recordHit records a cache hit
func (s *MemoryStore) recordHit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Hits++
		s.stats.mu.Unlock()
	}
}

// recordMiss records a cache miss
func (s *MemoryStore) recordMiss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Misses++
		s.stats.mu.Unlock()
	}
}

// recordEviction records an eviction
func (s *MemoryStore) recordEviction() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Evictions++
		s.stats.mu.Unlock()
	}
}

// recordExpiration records an expiration
func (s *MemoryStore) recordExpiration() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Expirations++
		s.stats.mu.Unlock()
	}
}
