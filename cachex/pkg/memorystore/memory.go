package memorystore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// Store implements the cache.Store interface using in-memory storage
type Store struct {
	// Configuration
	config *Config

	// Data storage
	data map[string]*cacheItem
	mu   sync.RWMutex

	// Eviction tracking
	accessOrder []string       // LRU order
	accessMap   map[string]int // key -> position in accessOrder

	// Statistics
	stats *Stats

	// Background cleanup
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// Config defines memory store configuration
type Config struct {
	// Capacity limits
	MaxSize     int // Maximum number of items
	MaxMemoryMB int // Maximum memory usage in MB (approximate)

	// TTL settings
	DefaultTTL      time.Duration // Default TTL for items
	CleanupInterval time.Duration // How often to run cleanup

	// Eviction policy
	EvictionPolicy EvictionPolicy // LRU, LFU, or TTL-based

	// Performance tuning
	EnableStats bool // Enable detailed statistics
}

// EvictionPolicy defines how items are evicted when capacity is reached
type EvictionPolicy string

const (
	EvictionPolicyLRU EvictionPolicy = "lru" // Least Recently Used
	EvictionPolicyLFU EvictionPolicy = "lfu" // Least Frequently Used
	EvictionPolicyTTL EvictionPolicy = "ttl" // Time To Live (oldest first)
)

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  EvictionPolicyLRU,
		EnableStats:     true,
	}
}

// HighPerformanceConfig returns a configuration optimized for high throughput
func HighPerformanceConfig() *Config {
	return &Config{
		MaxSize:         50000,
		MaxMemoryMB:     500,
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 30 * time.Second,
		EvictionPolicy:  EvictionPolicyLRU,
		EnableStats:     true,
	}
}

// ResourceConstrainedConfig returns a configuration for resource-constrained environments
func ResourceConstrainedConfig() *Config {
	return &Config{
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
type Stats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
	Size        int64
	MemoryUsage int64 // Approximate memory usage in bytes
	mu          sync.RWMutex
}

// New creates a new memory store
func New(config *Config) (*Store, error) {
	if config == nil {
		config = DefaultConfig()
	}

	store := &Store{
		config:      config,
		data:        make(map[string]*cacheItem),
		accessOrder: make([]string, 0),
		accessMap:   make(map[string]int),
		stats:       &Stats{},
		stopChan:    make(chan struct{}),
	}

	// Start background cleanup
	store.startCleanup()

	return store, nil
}

// Get retrieves a value from the memory store
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	item, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		s.recordMiss()
		return nil, nil // Key not found
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.removeFromAccessOrder(key)
		s.mu.Unlock()
		s.recordExpiration()
		return nil, nil // Item expired
	}

	// Update access statistics
	s.mu.Lock()
	item.AccessCount++
	item.LastAccess = time.Now()
	s.updateAccessOrder(key)
	s.mu.Unlock()

	s.recordHit()
	return item.Value, nil
}

// Set stores a value in the memory store
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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
		return nil
	}

	// Check capacity before adding new item
	if len(s.data) >= s.config.MaxSize {
		s.evictItems(1)
	}

	// Add new item
	s.data[key] = item
	s.addToAccessOrder(key)
	s.updateStats(0, item.Size)

	return nil
}

// MGet retrieves multiple values from the memory store
func (s *Store) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	result := make(map[string][]byte)
	now := time.Now()

	s.mu.RLock()
	for _, key := range keys {
		if item, exists := s.data[key]; exists {
			if now.After(item.ExpiresAt) {
				// Item expired, will be cleaned up later
				continue
			}
			result[key] = item.Value
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

	return result, nil
}

// MSet stores multiple values in the memory store
func (s *Store) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
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

	return nil
}

// Del removes keys from the memory store
func (s *Store) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
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

	return nil
}

// Exists checks if a key exists in the memory store
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	item, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.removeFromAccessOrder(key)
		s.mu.Unlock()
		s.recordExpiration()
		return false, nil
	}

	return true, nil
}

// TTL gets the time to live of a key
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	s.mu.RLock()
	item, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return 0, nil
	}

	ttl := time.Until(item.ExpiresAt)
	if ttl <= 0 {
		// Item has expired
		s.mu.Lock()
		delete(s.data, key)
		s.removeFromAccessOrder(key)
		s.mu.Unlock()
		s.recordExpiration()
		return 0, nil
	}

	return ttl, nil
}

// IncrBy increments a key by the given delta
func (s *Store) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.data[key]
	if !exists {
		// Create new item
		if ttlIfCreate == 0 {
			ttlIfCreate = s.config.DefaultTTL
		}

		expiresAt := time.Now().Add(ttlIfCreate)
		item = &cacheItem{
			Value:       []byte(fmt.Sprintf("%d", delta)),
			ExpiresAt:   expiresAt,
			AccessCount: 1,
			LastAccess:  time.Now(),
			Size:        len(fmt.Sprintf("%d", delta)),
		}

		// Check capacity
		if len(s.data) >= s.config.MaxSize {
			s.evictItems(1)
		}

		s.data[key] = item
		s.addToAccessOrder(key)
		s.updateStats(0, item.Size)

		return delta, nil
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		delete(s.data, key)
		s.removeFromAccessOrder(key)
		s.recordExpiration()
		return 0, fmt.Errorf("key has expired")
	}

	// Parse current value
	var current int64
	fmt.Sscanf(string(item.Value), "%d", &current)
	newValue := current + delta

	// Update item
	oldSize := item.Size
	item.Value = []byte(fmt.Sprintf("%d", newValue))
	item.Size = len(item.Value)
	item.AccessCount++
	item.LastAccess = time.Now()
	s.updateAccessOrder(key)
	s.updateStats(oldSize, item.Size)

	return newValue, nil
}

// Close closes the memory store and releases resources
func (s *Store) Close() error {
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// GetStats returns current statistics
func (s *Store) GetStats() *Stats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	s.stats.Size = int64(len(s.data))
	return s.stats
}

// Clear removes all items from the store
func (s *Store) Clear() {
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
func (s *Store) startCleanup() {
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
func (s *Store) cleanup() {
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
func (s *Store) evictItems(count int) {
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
func (s *Store) addToAccessOrder(key string) {
	s.accessOrder = append(s.accessOrder, key)
	s.accessMap[key] = len(s.accessOrder) - 1
}

// removeFromAccessOrder removes a key from the access order
func (s *Store) removeFromAccessOrder(key string) {
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
func (s *Store) updateAccessOrder(key string) {
	s.removeFromAccessOrder(key)
	s.addToAccessOrder(key)
}

// updateStats updates memory usage statistics
func (s *Store) updateStats(oldSize, newSize int) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.MemoryUsage += int64(newSize - oldSize)
		s.stats.mu.Unlock()
	}
}

// recordHit records a cache hit
func (s *Store) recordHit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Hits++
		s.stats.mu.Unlock()
	}
}

// recordMiss records a cache miss
func (s *Store) recordMiss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Misses++
		s.stats.mu.Unlock()
	}
}

// recordEviction records an eviction
func (s *Store) recordEviction() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Evictions++
		s.stats.mu.Unlock()
	}
}

// recordExpiration records an expiration
func (s *Store) recordExpiration() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.Expirations++
		s.stats.mu.Unlock()
	}
}
