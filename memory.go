package cachex

import (
	"context"
	"fmt"
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
}

// Config defines memory store configuration
type MemoryConfig struct {
	// Capacity limits
	MaxSize     int `yaml:"max_size" json:"max_size"`           // Maximum number of items
	MaxMemoryMB int `yaml:"max_memory_mb" json:"max_memory_mb"` // Maximum memory usage in MB (approximate)

	// TTL settings
	DefaultTTL      time.Duration `yaml:"default_ttl" json:"default_ttl"`           // Default TTL for items
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"` // How often to run cleanup

	// Eviction policy
	EvictionPolicy EvictionPolicy `yaml:"eviction_policy" json:"eviction_policy"` // LRU, LFU, or TTL-based

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
	Size        int // Size in bytes
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

	store := &MemoryStore{
		config:      config,
		data:        make(map[string]*cacheItem),
		accessOrder: make([]string, 0),
		accessMap:   make(map[string]int),
		stats:       &MemoryStats{},
		stopChan:    make(chan struct{}),
	}

	// Start background cleanup
	store.startCleanup()

	return store, nil
}

// Get retrieves a value from the memory store (non-blocking)
func (s *MemoryStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		s.mu.RLock()
		item, exists := s.data[key]
		if !exists {
			s.mu.RUnlock()
			s.recordMiss()
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired while still holding read lock
		now := time.Now()
		expired := now.After(item.ExpiresAt)

		if expired {
			s.mu.RUnlock()
			// Acquire write lock to delete expired item
			s.mu.Lock()
			// Check again in case another goroutine already deleted it
			if item, exists := s.data[key]; exists && now.After(item.ExpiresAt) {
				delete(s.data, key)
				s.removeFromAccessOrder(key)
			}
			s.mu.Unlock()
			s.recordExpiration()
			result <- AsyncResult{Exists: false}
			return
		}

		// Copy the value while holding read lock
		value := make([]byte, len(item.Value))
		copy(value, item.Value)
		s.mu.RUnlock()

		// Update access statistics
		s.mu.Lock()
		// Check again that item still exists
		if currentItem, exists := s.data[key]; exists && !now.After(currentItem.ExpiresAt) {
			currentItem.AccessCount++
			currentItem.LastAccess = now
			s.updateAccessOrder(key)
		}
		s.mu.Unlock()

		s.recordHit()
		result <- AsyncResult{Value: value, Exists: true}
	}()

	return result
}

// Set stores a value in the memory store (non-blocking)
func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
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

		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		expiresAt := time.Now().Add(ttl)
		item := &cacheItem{
			Value:       value,
			ExpiresAt:   expiresAt,
			AccessCount: 1,
			LastAccess:  time.Now(),
			Size:        len(value),
		}

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

		values := make(map[string][]byte)
		now := time.Now()

		s.mu.RLock()
		for _, key := range keys {
			if item, exists := s.data[key]; exists {
				if now.After(item.ExpiresAt) {
					// Item expired, will be cleaned up later
					continue
				}
				values[key] = item.Value
			}
		}
		s.mu.RUnlock()

		// Update access statistics for found items
		s.mu.Lock()
		for _, key := range keys {
			if item, exists := s.data[key]; exists {
				if now.After(item.ExpiresAt) {
					continue
				}
				item.AccessCount++
				item.LastAccess = now
				s.updateAccessOrder(key)
			}
		}
		s.mu.Unlock()

		result <- AsyncResult{Values: values}
	}()

	return result
}

// MSet stores multiple values in the memory store (non-blocking)
func (s *MemoryStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		if len(items) == 0 {
			result <- AsyncResult{}
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

		if ttl == 0 {
			ttl = s.config.DefaultTTL
		}

		expiresAt := time.Now().Add(ttl)
		now := time.Now()

		s.mu.Lock()
		defer s.mu.Unlock()

		// Check capacity and evict if necessary
		needed := len(items)
		for key := range items {
			if _, exists := s.data[key]; !exists {
				needed++
			}
		}

		if len(s.data)+needed > s.config.MaxSize {
			s.evictItems(needed)
		}

		// Add/update items
		for key, value := range items {
			item := &cacheItem{
				Value:       value,
				ExpiresAt:   expiresAt,
				AccessCount: 1,
				LastAccess:  now,
				Size:        len(value),
			}

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

		s.mu.Lock()
		defer s.mu.Unlock()

		for _, key := range keys {
			if item, exists := s.data[key]; exists {
				delete(s.data, key)
				s.removeFromAccessOrder(key)
				s.updateStats(item.Size, 0)
			}
		}

		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in the memory store (non-blocking)
func (s *MemoryStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		s.mu.RLock()
		item, exists := s.data[key]
		s.mu.RUnlock()

		if !exists {
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired
		if time.Now().After(item.ExpiresAt) {
			s.mu.Lock()
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.mu.Unlock()
			s.recordExpiration()
			result <- AsyncResult{Exists: false}
			return
		}

		result <- AsyncResult{Exists: true}
	}()

	return result
}

// TTL returns the time to live for a key (non-blocking)
func (s *MemoryStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		s.mu.RLock()
		item, exists := s.data[key]
		s.mu.RUnlock()

		if !exists {
			result <- AsyncResult{Exists: false}
			return
		}

		// Check if item has expired
		if time.Now().After(item.ExpiresAt) {
			s.mu.Lock()
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.mu.Unlock()
			s.recordExpiration()
			result <- AsyncResult{Exists: false}
			return
		}

		ttl := time.Until(item.ExpiresAt)
		result <- AsyncResult{TTL: ttl, Exists: true}
	}()

	return result
}

// IncrBy increments a counter by the specified delta (non-blocking)
func (s *MemoryStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
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
					}
				}
			}
		}

		// Calculate new value
		newValue := currentValue + delta

		// Create or update item
		expiresAt := time.Now().Add(ttlIfCreate)
		if exists {
			expiresAt = item.ExpiresAt
		}

		newItem := &cacheItem{
			Value:       []byte(strconv.FormatInt(newValue, 10)),
			ExpiresAt:   expiresAt,
			AccessCount: 1,
			LastAccess:  time.Now(),
			Size:        len(strconv.FormatInt(newValue, 10)),
		}

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
	select {
	case <-s.stopChan:
		// Already closed
		return nil
	default:
		close(s.stopChan)
	}
	s.wg.Wait()
	return nil
}

// GetStats returns current statistics
func (s *MemoryStore) GetStats() *MemoryStats {
	// First get the data size while holding the data mutex
	s.mu.RLock()
	dataSize := int64(len(s.data))
	s.mu.RUnlock()

	// Then update the stats while holding the stats mutex
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	s.stats.Size = dataSize
	return s.stats
}

// Clear removes all items from the store
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*cacheItem)
	s.accessOrder = make([]string, 0)
	s.accessMap = make(map[string]int)
	s.stats.mu.Lock()
	s.stats.Size = 0
	s.stats.MemoryUsage = 0
	s.stats.mu.Unlock()
}

// startCleanup starts the background cleanup goroutine
func (s *MemoryStore) startCleanup() {
	s.cleanupTicker = time.NewTicker(s.config.CleanupInterval)
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		defer s.cleanupTicker.Stop()

		for {
			select {
			case <-s.cleanupTicker.C:
				s.cleanup()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// cleanup removes expired items from the store
func (s *MemoryStore) cleanup() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	s.mu.RLock()
	for key, item := range s.data {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.mu.RUnlock()

	if len(expiredKeys) > 0 {
		s.mu.Lock()
		for _, key := range expiredKeys {
			if item, exists := s.data[key]; exists {
				delete(s.data, key)
				s.removeFromAccessOrder(key)
				s.updateStats(item.Size, 0)
				s.recordExpiration()
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
		// Evict least frequently used
		minAccess := int64(^uint64(0) >> 1) // Max int64
		for key, item := range s.data {
			if item.AccessCount < minAccess {
				minAccess = item.AccessCount
				keysToEvict = []string{key}
			}
		}

	case EvictionPolicyTTL:
		// Evict items with shortest TTL remaining
		earliestExpiry := time.Now().Add(time.Hour * 24 * 365) // Far future
		for key, item := range s.data {
			if item.ExpiresAt.Before(earliestExpiry) {
				earliestExpiry = item.ExpiresAt
				keysToEvict = []string{key}
			}
		}
	}

	// Evict the selected keys
	for _, key := range keysToEvict {
		if item, exists := s.data[key]; exists {
			delete(s.data, key)
			s.removeFromAccessOrder(key)
			s.updateStats(item.Size, 0)
			s.recordEviction()
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
		// Remove from slice
		s.accessOrder = append(s.accessOrder[:pos], s.accessOrder[pos+1:]...)
		delete(s.accessMap, key)

		// Update positions for keys after the removed one
		for i := pos; i < len(s.accessOrder); i++ {
			s.accessMap[s.accessOrder[i]] = i
		}
	}
}

// updateAccessOrder moves a key to the end of the access order (most recently used)
func (s *MemoryStore) updateAccessOrder(key string) {
	s.removeFromAccessOrder(key)
	s.addToAccessOrder(key)
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
