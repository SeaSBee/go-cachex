package cachex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seasbee/go-validatorx"
)

// ValidationCache provides caching for configuration validation results
// to avoid repeated validation of identical configurations
type ValidationCache struct {
	cache map[string]*ValidationResult
	mu    sync.RWMutex
	// Configuration
	maxSize     int
	ttl         time.Duration
	enableStats bool
	// Statistics
	stats *ValidationCacheStats
	// Context for cleanup goroutine
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
}

// ValidationResult represents a cached validation result
type ValidationResult struct {
	Valid       bool
	Error       string
	Timestamp   time.Time
	ExpiresAt   time.Time
	AccessCount int64
}

// ValidationCacheStats holds statistics for the validation cache
type ValidationCacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
}

// ValidationCacheConfig defines configuration for the validation cache
type ValidationCacheConfig struct {
	MaxSize     int           // Maximum number of cached results
	TTL         time.Duration // Time to live for cached results
	EnableStats bool          // Enable statistics collection
}

// DefaultValidationCacheConfig returns a default validation cache configuration
func DefaultValidationCacheConfig() *ValidationCacheConfig {
	return &ValidationCacheConfig{
		MaxSize:     1000,            // Cache up to 1000 validation results
		TTL:         5 * time.Minute, // Cache results for 5 minutes
		EnableStats: true,            // Enable statistics by default
	}
}

// HighPerformanceValidationCacheConfig returns a high-performance configuration
func HighPerformanceValidationCacheConfig() *ValidationCacheConfig {
	return &ValidationCacheConfig{
		MaxSize:     5000,             // Cache up to 5000 validation results
		TTL:         10 * time.Minute, // Cache results for 10 minutes
		EnableStats: true,             // Enable statistics
	}
}

// NewValidationCache creates a new validation cache
func NewValidationCache(config *ValidationCacheConfig) *ValidationCache {
	if config == nil {
		config = DefaultValidationCacheConfig()
	}

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	cache := &ValidationCache{
		cache:         make(map[string]*ValidationResult),
		maxSize:       config.MaxSize,
		ttl:           config.TTL,
		enableStats:   config.EnableStats,
		stats:         &ValidationCacheStats{},
		cleanupCtx:    cleanupCtx,
		cleanupCancel: cleanupCancel,
	}

	// Start background cleanup if TTL is set
	if config.TTL > 0 {
		go cache.startCleanup()
	}

	return cache
}

// Close stops the validation cache and cleans up resources
func (vc *ValidationCache) Close() {
	if vc != nil && vc.cleanupCancel != nil {
		vc.cleanupCancel()
	}
}

// IsValid checks if a configuration is valid, using cache if available
func (vc *ValidationCache) IsValid(config *CacheConfig) (bool, error) {
	// Check for nil cache - return error instead of causing panic
	if vc == nil {
		return false, fmt.Errorf("validation cache is nil")
	}

	// Check for nil config
	if config == nil {
		return false, fmt.Errorf("configuration is nil")
	}

	// Generate hash for the configuration
	configHash := vc.generateConfigHash(config)

	// Check cache first
	vc.mu.RLock()
	result, exists := vc.cache[configHash]
	if exists {
		// Check if result is still valid (not expired)
		if time.Now().Before(result.ExpiresAt) {
			// Cache hit - update statistics atomically
			if vc.enableStats {
				atomic.AddInt64(&vc.stats.Hits, 1)
				atomic.AddInt64(&result.AccessCount, 1)
			}
			vc.mu.RUnlock()

			if result.Valid {
				return true, nil
			} else {
				return false, fmt.Errorf("%s", result.Error)
			}
		} else {
			// Result expired, remove it
			vc.mu.RUnlock()
			vc.mu.Lock()
			// Double-check that the item still exists and is still expired
			if result, stillExists := vc.cache[configHash]; stillExists && time.Now().After(result.ExpiresAt) {
				delete(vc.cache, configHash)
				if vc.enableStats {
					atomic.AddInt64(&vc.stats.Expirations, 1)
				}
			}
			vc.mu.Unlock()
		}
	} else {
		vc.mu.RUnlock()
	}

	// Cache miss - perform validation
	if vc.enableStats {
		atomic.AddInt64(&vc.stats.Misses, 1)
	}

	valid, err := vc.performValidation(config)

	// Cache the result
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Check if we need to evict items due to size limit
	if len(vc.cache) >= vc.maxSize {
		vc.evictOldest()
	}

	// Create validation result
	result = &ValidationResult{
		Valid:       valid,
		Error:       "",
		Timestamp:   time.Now(),
		ExpiresAt:   time.Now().Add(vc.ttl),
		AccessCount: 1,
	}

	if !valid && err != nil {
		result.Error = err.Error()
	}

	// Store in cache
	vc.cache[configHash] = result

	return valid, err
}

// performValidation performs the actual validation logic
func (vc *ValidationCache) performValidation(config *CacheConfig) (bool, error) {
	// Check for nil config
	if config == nil {
		return false, fmt.Errorf("configuration is nil")
	}

	// This is the original validation logic from validateConfig
	// Create validator instance
	validator := validatorx.NewValidator()

	// Check that only one cache store is configured
	storeCount := 0
	if config.Memory != nil {
		storeCount++
	}
	if config.Redis != nil {
		storeCount++
	}
	if config.Ristretto != nil {
		storeCount++
	}
	if config.Layered != nil {
		storeCount++
	}

	if storeCount == 0 {
		return false, fmt.Errorf("at least one cache store must be configured (memory, redis, ristretto, or layered)")
	}
	if storeCount > 1 {
		return false, fmt.Errorf("only one cache store can be configured per instance, found %d stores", storeCount)
	}

	// Validate the entire config structure using go-validatorx
	result := validator.ValidateStruct(config)
	if !result.Valid {
		var errors []string
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s: %s", err.Field, err.Message))
		}
		return false, fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	// Additional custom validations that can't be expressed with struct tags
	if config.Redis != nil {
		// Validate MinIdleConns <= PoolSize
		if config.Redis.MinIdleConns > config.Redis.PoolSize {
			return false, fmt.Errorf("redis store min_idle_conns cannot exceed pool_size")
		}
	}

	// Validate refresh ahead settings
	if config.RefreshAhead != nil {
		if config.RefreshAhead.Enabled && config.RefreshAhead.RefreshInterval <= 0 {
			return false, fmt.Errorf("refresh_ahead refresh_interval must be greater than 0 when enabled")
		}
	}

	return true, nil
}

// generateConfigHash generates a hash for the configuration
func (vc *ValidationCache) generateConfigHash(config *CacheConfig) string {
	// Check for nil config
	if config == nil {
		return "nil-config"
	}

	// Create a simplified representation of the config for hashing
	// We exclude fields that don't affect validation (like function pointers)
	configForHash := struct {
		Memory        *MemoryConfig        `json:"memory,omitempty"`
		Redis         *RedisConfig         `json:"redis,omitempty"`
		Ristretto     *RistrettoConfig     `json:"ristretto,omitempty"`
		Layered       *LayeredConfig       `json:"layered,omitempty"`
		DefaultTTL    time.Duration        `json:"default_ttl"`
		MaxRetries    int                  `json:"max_retries"`
		RetryDelay    time.Duration        `json:"retry_delay"`
		Codec         string               `json:"codec"`
		Observability *ObservabilityConfig `json:"observability,omitempty"`
		Tagging       *TagConfig           `json:"tagging,omitempty"`
		RefreshAhead  *RefreshAheadConfig  `json:"refresh_ahead,omitempty"`
		GORM          *GormConfig          `json:"gorm,omitempty"`
	}{
		Memory:        config.Memory,
		Redis:         config.Redis,
		Ristretto:     config.Ristretto,
		Layered:       config.Layered,
		DefaultTTL:    config.DefaultTTL,
		MaxRetries:    config.MaxRetries,
		RetryDelay:    config.RetryDelay,
		Codec:         config.Codec,
		Observability: config.Observability,
		Tagging:       config.Tagging,
		RefreshAhead:  config.RefreshAhead,
		GORM:          config.GORM,
	}

	// Marshal to JSON for consistent hashing
	data, err := json.Marshal(configForHash)
	if err != nil {
		// Fallback to a simple string representation
		return fmt.Sprintf("config-%p", config)
	}

	// Generate SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// evictOldest removes the oldest (least recently accessed) item from the cache
func (vc *ValidationCache) evictOldest() {
	if len(vc.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestAccess int64 = 1<<63 - 1 // Max int64

	for key, result := range vc.cache {
		accessCount := atomic.LoadInt64(&result.AccessCount)
		if accessCount < oldestAccess {
			oldestAccess = accessCount
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(vc.cache, oldestKey)
		if vc.enableStats {
			atomic.AddInt64(&vc.stats.Evictions, 1)
		}
	}
}

// startCleanup starts the background cleanup goroutine
func (vc *ValidationCache) startCleanup() {
	ticker := time.NewTicker(vc.ttl / 2) // Run cleanup every half TTL
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			vc.cleanup()
		case <-vc.cleanupCtx.Done():
			return
		}
	}
}

// cleanup removes expired items from the cache
func (vc *ValidationCache) cleanup() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	vc.mu.RLock()
	for key, result := range vc.cache {
		if now.After(result.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	vc.mu.RUnlock()

	if len(expiredKeys) > 0 {
		vc.mu.Lock()
		for _, key := range expiredKeys {
			delete(vc.cache, key)
		}
		if vc.enableStats {
			atomic.AddInt64(&vc.stats.Expirations, int64(len(expiredKeys)))
		}
		vc.mu.Unlock()
	}
}

// GetStats returns cache statistics
func (vc *ValidationCache) GetStats() *ValidationCacheStats {
	if vc == nil || !vc.enableStats {
		return nil
	}

	return &ValidationCacheStats{
		Hits:        atomic.LoadInt64(&vc.stats.Hits),
		Misses:      atomic.LoadInt64(&vc.stats.Misses),
		Evictions:   atomic.LoadInt64(&vc.stats.Evictions),
		Expirations: atomic.LoadInt64(&vc.stats.Expirations),
	}
}

// Clear clears all cached validation results
func (vc *ValidationCache) Clear() {
	if vc == nil {
		return
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.cache = make(map[string]*ValidationResult)
}

// Size returns the current number of cached items
func (vc *ValidationCache) Size() int {
	if vc == nil {
		return 0
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	return len(vc.cache)
}

// Global validation cache instance
var GlobalValidationCache *ValidationCache

// init initializes the global validation cache
func init() {
	GlobalValidationCache = NewValidationCache(DefaultValidationCacheConfig())
}
