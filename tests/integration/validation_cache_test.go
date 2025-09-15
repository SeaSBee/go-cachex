package integration

import (
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/SeaSBee/go-logx"
)

// TestValidationCacheIntegration tests validation cache integration with cache creation
func TestValidationCacheIntegration(t *testing.T) {
	// Test that validation cache works with cache creation
	config := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
		Observability: &cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: false,
			EnableLogging: false,
		},
	}

	// Test validation cache directly
	valid, err := cachex.GlobalValidationCache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Create cache - this should use validation cache internally
	cache, err := cachex.NewFromConfig[string](config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Verify that cache was created successfully
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}

	logx.Info("Validation cache integration test completed")
}

// TestValidationCacheReuse tests that validation cache is reused for identical configs
func TestValidationCacheReuse(t *testing.T) {
	// Clear the global validation cache to start fresh
	cachex.GlobalValidationCache.Clear()

	config := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
	}

	// Get initial stats
	initialStats := cachex.GlobalValidationCache.GetStats()
	if initialStats == nil {
		t.Fatal("Expected stats but got nil")
	}
	initialMisses := initialStats.Misses
	initialHits := initialStats.Hits

	// First validation - should be a cache miss
	valid1, err1 := cachex.GlobalValidationCache.IsValid(config)
	if !valid1 || err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}

	// Second validation with identical config - should be a cache hit
	valid2, err2 := cachex.GlobalValidationCache.IsValid(config)
	if !valid2 || err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Get final stats
	finalStats := cachex.GlobalValidationCache.GetStats()
	if finalStats == nil {
		t.Fatal("Expected stats but got nil")
	}

	// Should have 1 additional miss and 1 additional hit
	additionalMisses := finalStats.Misses - initialMisses
	additionalHits := finalStats.Hits - initialHits

	if additionalMisses != 1 {
		t.Errorf("Expected 1 additional miss, got %d", additionalMisses)
	}
	if additionalHits != 1 {
		t.Errorf("Expected 1 additional hit, got %d", additionalHits)
	}

	logx.Info("Validation cache reuse test completed")
}

// TestValidationCacheDifferentConfigs tests that different configs are cached separately
func TestValidationCacheDifferentConfigs(t *testing.T) {
	// Clear the global validation cache to start fresh
	cachex.GlobalValidationCache.Clear()

	config1 := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
	}

	config2 := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         2000, // Different value
			MaxMemoryMB:     100,  // Different value
			DefaultTTL:      10 * time.Minute,
			CleanupInterval: 2 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLFU, // Different policy
			EnableStats:     false,                    // Different value
		},
		DefaultTTL: 10 * time.Minute,
		MaxRetries: 5, // Different value
		RetryDelay: 200 * time.Millisecond,
		Codec:      "msgpack", // Different codec
	}

	// Get initial stats
	initialStats := cachex.GlobalValidationCache.GetStats()
	if initialStats == nil {
		t.Fatal("Expected stats but got nil")
	}
	initialMisses := initialStats.Misses
	initialHits := initialStats.Hits

	// Validate first config
	valid1, err1 := cachex.GlobalValidationCache.IsValid(config1)
	if !valid1 || err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}

	// Validate second config
	valid2, err2 := cachex.GlobalValidationCache.IsValid(config2)
	if !valid2 || err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Get final stats
	finalStats := cachex.GlobalValidationCache.GetStats()
	if finalStats == nil {
		t.Fatal("Expected stats but got nil")
	}

	// Should have 2 additional misses (no additional hits since configs are different)
	additionalMisses := finalStats.Misses - initialMisses
	additionalHits := finalStats.Hits - initialHits

	if additionalMisses != 2 {
		t.Errorf("Expected 2 additional misses, got %d", additionalMisses)
	}
	if additionalHits != 0 {
		t.Errorf("Expected 0 additional hits, got %d", additionalHits)
	}

	logx.Info("Validation cache different configs test completed")
}

// TestValidationCacheInvalidConfig tests that invalid configs are properly handled
func TestValidationCacheInvalidConfig(t *testing.T) {
	// Test with invalid config (multiple stores)
	invalidConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		Redis: &cachex.RedisConfig{
			Addr:         "localhost:6379",
			PoolSize:     10,
			MinIdleConns: 5,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
	}

	// Test validation cache directly
	valid, err := cachex.GlobalValidationCache.IsValid(invalidConfig)
	if valid {
		t.Fatal("Expected invalid config but got valid")
	}
	if err == nil {
		t.Fatal("Expected error for invalid config but got nil")
	}

	// Check that the error message is correct
	if !contains(err.Error(), "only one cache store can be configured per instance") {
		t.Errorf("Expected error about multiple stores, got: %v", err)
	}

	logx.Info("Validation cache invalid config test completed")
}

// TestValidationCachePerformance tests performance impact of validation caching
func TestValidationCachePerformance(t *testing.T) {
	// Clear the global validation cache to start fresh
	cachex.GlobalValidationCache.Clear()

	config := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
	}

	// Get initial stats
	initialStats := cachex.GlobalValidationCache.GetStats()
	if initialStats == nil {
		t.Fatal("Expected stats but got nil")
	}
	initialMisses := initialStats.Misses
	initialHits := initialStats.Hits

	// Measure time for first validation (cache miss)
	start := time.Now()
	valid1, err1 := cachex.GlobalValidationCache.IsValid(config)
	firstValidationTime := time.Since(start)
	if !valid1 || err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}

	// Measure time for second validation (cache hit)
	start = time.Now()
	valid2, err2 := cachex.GlobalValidationCache.IsValid(config)
	secondValidationTime := time.Since(start)
	if !valid2 || err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Second validation should be faster (cache hit)
	if secondValidationTime >= firstValidationTime {
		t.Logf("Warning: Second validation (%v) was not faster than first (%v)",
			secondValidationTime, firstValidationTime)
	}

	// Verify cache hit occurred
	finalStats := cachex.GlobalValidationCache.GetStats()
	if finalStats == nil {
		t.Fatal("Expected stats but got nil")
	}

	additionalHits := finalStats.Hits - initialHits
	additionalMisses := finalStats.Misses - initialMisses

	if additionalHits != 1 {
		t.Errorf("Expected 1 additional hit, got %d", additionalHits)
	}
	if additionalMisses != 1 {
		t.Errorf("Expected 1 additional miss, got %d", additionalMisses)
	}

	logx.Info("Validation cache performance test completed")
}

// TestValidationCacheWithObservability tests validation cache with observability enabled
func TestValidationCacheWithObservability(t *testing.T) {
	config := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     50,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		Codec:      "json",
		Observability: &cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		},
	}

	// Test validation cache with observability config
	valid, err := cachex.GlobalValidationCache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Create cache with observability
	cache, err := cachex.NewFromConfig[string](config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Verify that cache was created successfully
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}

	logx.Info("Validation cache with observability test completed")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
