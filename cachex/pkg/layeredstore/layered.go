package layeredstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/seasbee/go-logx"
)

// Store implements a layered cache with memory as L1 and Redis as L2
type Store struct {
	// Stores
	l1Store cache.Store // Memory store (fast)
	l2Store cache.Store // Redis store (persistent)

	// Configuration
	config *Config

	// Statistics
	stats *Stats
}

// Config defines layered store configuration
type Config struct {
	// Memory store configuration
	MemoryConfig *memorystore.Config

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

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MemoryConfig:      memorystore.DefaultConfig(),
		WritePolicy:       WritePolicyThrough,
		ReadPolicy:        ReadPolicyThrough,
		SyncInterval:      5 * time.Minute,
		EnableStats:       true,
		MaxConcurrentSync: 10,
	}
}

// HighPerformanceConfig returns a configuration optimized for high throughput
func HighPerformanceConfig() *Config {
	return &Config{
		MemoryConfig:      memorystore.HighPerformanceConfig(),
		WritePolicy:       WritePolicyBehind,
		ReadPolicy:        ReadPolicyThrough,
		SyncInterval:      1 * time.Minute,
		EnableStats:       true,
		MaxConcurrentSync: 50,
	}
}

// ResourceConstrainedConfig returns a configuration for resource-constrained environments
func ResourceConstrainedConfig() *Config {
	return &Config{
		MemoryConfig:      memorystore.ResourceConstrainedConfig(),
		WritePolicy:       WritePolicyAround,
		ReadPolicy:        ReadPolicyAround,
		SyncInterval:      10 * time.Minute,
		EnableStats:       false,
		MaxConcurrentSync: 5,
	}
}

// Stats holds layered store statistics
type Stats struct {
	L1Hits     int64
	L1Misses   int64
	L2Hits     int64
	L2Misses   int64
	SyncCount  int64
	SyncErrors int64
	mu         sync.RWMutex
}

// New creates a new layered store
func New(l2Store cache.Store, config *Config) (*Store, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create L1 (memory) store
	l1Store, err := memorystore.New(config.MemoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	store := &Store{
		l1Store: l1Store,
		l2Store: l2Store,
		config:  config,
		stats:   &Stats{},
	}

	// Start background sync if needed
	if config.WritePolicy == WritePolicyBehind {
		store.startBackgroundSync()
	}

	return store, nil
}

// Get retrieves a value from the layered store
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	// Try L1 first
	value, err := s.l1Store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if value != nil {
		s.recordL1Hit()
		return value, nil
	}

	s.recordL1Miss()

	// L1 miss, try L2 based on read policy
	switch s.config.ReadPolicy {
	case ReadPolicyThrough:
		return s.readThrough(ctx, key)
	case ReadPolicyAside:
		return nil, nil // L1 miss, no fallback
	case ReadPolicyAround:
		return s.readAround(ctx, key)
	default:
		return s.readThrough(ctx, key)
	}
}

// Set stores a value in the layered store
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	switch s.config.WritePolicy {
	case WritePolicyThrough:
		return s.writeThrough(ctx, key, value, ttl)
	case WritePolicyBehind:
		return s.writeBehind(ctx, key, value, ttl)
	case WritePolicyAround:
		return s.writeAround(ctx, key, value, ttl)
	default:
		return s.writeThrough(ctx, key, value, ttl)
	}
}

// MGet retrieves multiple values from the layered store
func (s *Store) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Try L1 first
	l1Results, err := s.l1Store.MGet(ctx, keys...)
	if err != nil {
		return nil, err
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
		return l1Results, nil
	}

	s.recordL1Hits(int64(len(keys) - len(missingKeys)))
	s.recordL1Misses(int64(len(missingKeys)))

	// Get missing keys from L2
	l2Results, err := s.l2Store.MGet(ctx, missingKeys...)
	if err != nil {
		return nil, err
	}

	// Merge results
	for key, value := range l2Results {
		if value != nil {
			l1Results[key] = value
			// Populate L1 for future reads
			go func(k, v string) {
				s.l1Store.Set(context.Background(), k, []byte(v), s.config.MemoryConfig.DefaultTTL)
			}(key, string(value))
		}
	}

	s.recordL2Hits(int64(len(l2Results)))
	s.recordL2Misses(int64(len(missingKeys) - len(l2Results)))

	return l1Results, nil
}

// MSet stores multiple values in the layered store
func (s *Store) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	switch s.config.WritePolicy {
	case WritePolicyThrough:
		return s.msetThrough(ctx, items, ttl)
	case WritePolicyBehind:
		return s.msetBehind(ctx, items, ttl)
	case WritePolicyAround:
		return s.msetAround(ctx, items, ttl)
	default:
		return s.msetThrough(ctx, items, ttl)
	}
}

// Del removes keys from the layered store
func (s *Store) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// Delete from both layers
	var l1Err, l2Err error

	// Delete from L1
	l1Err = s.l1Store.Del(ctx, keys...)

	// Delete from L2
	l2Err = s.l2Store.Del(ctx, keys...)

	// Return error if both failed
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("L1 error: %w, L2 error: %w", l1Err, l2Err)
	}

	return nil
}

// Exists checks if a key exists in the layered store
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	// Check L1 first
	exists, err := s.l1Store.Exists(ctx, key)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	// Check L2
	return s.l2Store.Exists(ctx, key)
}

// TTL gets the time to live of a key
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	// Check L1 first
	ttl, err := s.l1Store.TTL(ctx, key)
	if err != nil {
		return 0, err
	}

	if ttl > 0 {
		return ttl, nil
	}

	// Check L2
	return s.l2Store.TTL(ctx, key)
}

// IncrBy increments a key by the given delta
func (s *Store) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
	// Try L1 first
	result, err := s.l1Store.IncrBy(ctx, key, delta, ttlIfCreate)
	if err == nil {
		// Success in L1, sync to L2 based on write policy
		switch s.config.WritePolicy {
		case WritePolicyThrough:
			// Already synced in write-through mode
		case WritePolicyBehind:
			// Async sync to L2
			go func() {
				s.l2Store.IncrBy(context.Background(), key, delta, ttlIfCreate)
			}()
		case WritePolicyAround:
			// Invalidate L1, write to L2
			s.l1Store.Del(context.Background(), key)
			s.l2Store.IncrBy(context.Background(), key, delta, ttlIfCreate)
		}
		return result, nil
	}

	// L1 failed, try L2
	result, err = s.l2Store.IncrBy(ctx, key, delta, ttlIfCreate)
	if err != nil {
		return 0, err
	}

	// Populate L1 for future reads
	go func() {
		s.l1Store.Set(context.Background(), key, []byte(fmt.Sprintf("%d", result)), ttlIfCreate)
	}()

	return result, nil
}

// Close closes the layered store and releases resources
func (s *Store) Close() error {
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
func (s *Store) GetStats() *Stats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return s.stats
}

// SyncL1ToL2 synchronizes L1 cache to L2 (for write-behind mode)
func (s *Store) SyncL1ToL2(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, you would track pending writes and sync them
	s.recordSync()
	return nil
}

// readThrough implements read-through policy
func (s *Store) readThrough(ctx context.Context, key string) ([]byte, error) {
	// Get from L2
	value, err := s.l2Store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if value != nil {
		s.recordL2Hit()
		// Populate L1 for future reads
		go func() {
			s.l1Store.Set(context.Background(), key, value, s.config.MemoryConfig.DefaultTTL)
		}()
		return value, nil
	}

	s.recordL2Miss()
	return nil, nil
}

// readAround implements read-around policy
func (s *Store) readAround(ctx context.Context, key string) ([]byte, error) {
	// Get from L2 only
	value, err := s.l2Store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if value != nil {
		s.recordL2Hit()
	} else {
		s.recordL2Miss()
	}

	return value, nil
}

// writeThrough implements write-through policy
func (s *Store) writeThrough(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Write to both L1 and L2
	var l1Err, l2Err error

	// Write to L1
	l1Err = s.l1Store.Set(ctx, key, value, ttl)

	// Write to L2
	l2Err = s.l2Store.Set(ctx, key, value, ttl)

	// Return error if both failed
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("L1 error: %w, L2 error: %w", l1Err, l2Err)
	}

	return nil
}

// writeBehind implements write-behind policy
func (s *Store) writeBehind(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Write to L1 immediately
	err := s.l1Store.Set(ctx, key, value, ttl)
	if err != nil {
		return err
	}

	// Write to L2 asynchronously
	go func() {
		if err := s.l2Store.Set(context.Background(), key, value, ttl); err != nil {
			logx.Error("Failed to write behind to L2", logx.String("key", key), logx.ErrorField(err))
		}
	}()

	return nil
}

// writeAround implements write-around policy
func (s *Store) writeAround(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Write to L2 only
	err := s.l2Store.Set(ctx, key, value, ttl)
	if err != nil {
		return err
	}

	// Invalidate L1
	s.l1Store.Del(ctx, key)

	return nil
}

// msetThrough implements MSet with write-through policy
func (s *Store) msetThrough(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	// Write to both L1 and L2
	var l1Err, l2Err error

	// Write to L1
	l1Err = s.l1Store.MSet(ctx, items, ttl)

	// Write to L2
	l2Err = s.l2Store.MSet(ctx, items, ttl)

	// Return error if both failed
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("L1 error: %w, L2 error: %w", l1Err, l2Err)
	}

	return nil
}

// msetBehind implements MSet with write-behind policy
func (s *Store) msetBehind(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	// Write to L1 immediately
	err := s.l1Store.MSet(ctx, items, ttl)
	if err != nil {
		return err
	}

	// Write to L2 asynchronously
	go func() {
		if err := s.l2Store.MSet(context.Background(), items, ttl); err != nil {
			logx.Error("Failed to MSet behind to L2", logx.ErrorField(err))
		}
	}()

	return nil
}

// msetAround implements MSet with write-around policy
func (s *Store) msetAround(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	// Write to L2 only
	err := s.l2Store.MSet(ctx, items, ttl)
	if err != nil {
		return err
	}

	// Invalidate L1 for all keys
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	s.l1Store.Del(ctx, keys...)

	return nil
}

// startBackgroundSync starts the background sync goroutine
func (s *Store) startBackgroundSync() {
	// This is a placeholder for background sync implementation
	// In a real implementation, you would track pending writes and sync them periodically
}

// recordL1Hit records an L1 cache hit
func (s *Store) recordL1Hit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Hits++
		s.stats.mu.Unlock()
	}
}

// recordL1Hits records multiple L1 cache hits
func (s *Store) recordL1Hits(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Hits += count
		s.stats.mu.Unlock()
	}
}

// recordL1Miss records an L1 cache miss
func (s *Store) recordL1Miss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Misses++
		s.stats.mu.Unlock()
	}
}

// recordL1Misses records multiple L1 cache misses
func (s *Store) recordL1Misses(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L1Misses += count
		s.stats.mu.Unlock()
	}
}

// recordL2Hit records an L2 cache hit
func (s *Store) recordL2Hit() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Hits++
		s.stats.mu.Unlock()
	}
}

// recordL2Hits records multiple L2 cache hits
func (s *Store) recordL2Hits(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Hits += count
		s.stats.mu.Unlock()
	}
}

// recordL2Miss records an L2 cache miss
func (s *Store) recordL2Miss() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Misses++
		s.stats.mu.Unlock()
	}
}

// recordL2Misses records multiple L2 cache misses
func (s *Store) recordL2Misses(count int64) {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.L2Misses += count
		s.stats.mu.Unlock()
	}
}

// recordSync records a sync operation
func (s *Store) recordSync() {
	if s.config.EnableStats {
		s.stats.mu.Lock()
		s.stats.SyncCount++
		s.stats.mu.Unlock()
	}
}
