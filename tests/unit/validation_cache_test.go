package unit

import (
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// TestNewValidationCache tests cache creation with different configurations
func TestNewValidationCache(t *testing.T) {
	tests := []struct {
		name   string
		config *cachex.ValidationCacheConfig
		want   *cachex.ValidationCacheConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want:   cachex.DefaultValidationCacheConfig(),
		},
		{
			name: "custom config",
			config: &cachex.ValidationCacheConfig{
				MaxSize:     500,
				TTL:         2 * time.Minute,
				EnableStats: false,
			},
			want: &cachex.ValidationCacheConfig{
				MaxSize:     500,
				TTL:         2 * time.Minute,
				EnableStats: false,
			},
		},
		{
			name:   "high performance config",
			config: cachex.HighPerformanceValidationCacheConfig(),
			want:   cachex.HighPerformanceValidationCacheConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cachex.NewValidationCache(tt.config)
			defer cache.Close()

			if cache == nil {
				t.Fatal("Expected non-nil cache")
			}

			// Test that cache is functional by validating a config
			testConfig := &cachex.CacheConfig{
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

			valid, err := cache.IsValid(testConfig)
			if !valid || err != nil {
				t.Errorf("Cache validation failed: %v", err)
			}

			// Test stats functionality
			stats := cache.GetStats()
			if tt.want.EnableStats && stats == nil {
				t.Error("Expected stats to be available when EnableStats is true")
			}
		})
	}
}

// TestValidationCache_IsValid_ValidConfigs tests validation with valid configurations
func TestValidationCache_IsValid_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config *cachex.CacheConfig
	}{
		{
			name: "memory only config",
			config: &cachex.CacheConfig{
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
			},
		},
		{
			name: "redis only config",
			config: &cachex.CacheConfig{
				Redis: &cachex.RedisConfig{
					Addr:         "localhost:6379",
					Password:     "",
					DB:           0,
					PoolSize:     10,
					MinIdleConns: 5,
					MaxRetries:   3,
					DialTimeout:  5 * time.Second,
					ReadTimeout:  3 * time.Second,
					WriteTimeout: 3 * time.Second,
				},
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
		},
		{
			name: "ristretto only config",
			config: &cachex.CacheConfig{
				Ristretto: &cachex.RistrettoConfig{
					MaxItems:       100000,
					MaxMemoryBytes: 100 * 1024 * 1024,
					DefaultTTL:     5 * time.Minute,
					NumCounters:    1000000,
					BufferItems:    64,
					CostFunction:   func(value interface{}) int64 { return 1 },
					EnableMetrics:  true,
					EnableStats:    true,
					BatchSize:      20,
				},
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
		},
		{
			name: "layered only config",
			config: &cachex.CacheConfig{
				Layered: &cachex.LayeredConfig{
					MemoryConfig: &cachex.MemoryConfig{
						MaxSize:         1000,
						MaxMemoryMB:     50,
						DefaultTTL:      1 * time.Minute,
						CleanupInterval: 30 * time.Second,
						EvictionPolicy:  cachex.EvictionPolicyLRU,
						EnableStats:     true,
					},
					L2StoreConfig: &cachex.L2StoreConfig{
						Type: "redis",
						Redis: &cachex.RedisConfig{
							Addr:         "localhost:6379",
							Password:     "",
							DB:           0,
							PoolSize:     10,
							MinIdleConns: 5,
							MaxRetries:   3,
							DialTimeout:  5 * time.Second,
							ReadTimeout:  3 * time.Second,
							WriteTimeout: 3 * time.Second,
						},
					},
					SyncInterval: 5 * time.Second,
					EnableStats:  true,
				},
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
			defer cache.Close()

			valid, err := cache.IsValid(tt.config)
			if !valid {
				t.Errorf("Expected valid config, got invalid: %v", err)
			}
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestValidationCache_IsValid_InvalidConfigs tests validation with invalid configurations
func TestValidationCache_IsValid_InvalidConfigs(t *testing.T) {
	tests := []struct {
		name          string
		config        *cachex.CacheConfig
		expectedError string
		expectedValid bool
	}{
		{
			name:          "nil config",
			config:        nil,
			expectedError: "configuration is nil",
			expectedValid: false,
		},
		{
			name: "no store configured",
			config: &cachex.CacheConfig{
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
			expectedError: "at least one cache store must be configured",
			expectedValid: false,
		},
		{
			name: "multiple stores configured",
			config: &cachex.CacheConfig{
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
			},
			expectedError: "only one cache store can be configured per instance",
			expectedValid: false,
		},
		{
			name: "redis min_idle_conns exceeds pool_size",
			config: &cachex.CacheConfig{
				Redis: &cachex.RedisConfig{
					Addr:         "localhost:6379",
					PoolSize:     5,
					MinIdleConns: 10, // Greater than PoolSize
				},
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
			expectedError: "redis store min_idle_conns cannot exceed pool_size",
			expectedValid: false,
		},
		{
			name: "refresh_ahead enabled with invalid interval",
			config: &cachex.CacheConfig{
				Memory: &cachex.MemoryConfig{
					MaxSize:         1000,
					MaxMemoryMB:     50,
					DefaultTTL:      5 * time.Minute,
					CleanupInterval: 1 * time.Minute,
					EvictionPolicy:  cachex.EvictionPolicyLRU,
					EnableStats:     true,
				},
				RefreshAhead: &cachex.RefreshAheadConfig{
					Enabled:         true,
					RefreshInterval: 0, // Invalid: must be > 0
				},
				DefaultTTL: 5 * time.Minute,
				MaxRetries: 3,
				RetryDelay: 100 * time.Millisecond,
				Codec:      "json",
			},
			expectedError: "refresh_ahead refresh_interval must be greater than 0 when enabled",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
			defer cache.Close()

			valid, err := cache.IsValid(tt.config)
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}
			if err == nil && tt.expectedError != "" {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if err != nil && tt.expectedError != "" {
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

// TestValidationCache_CacheBehavior tests cache hit/miss behavior
func TestValidationCache_CacheBehavior(t *testing.T) {
	cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
		MaxSize:     10,
		TTL:         1 * time.Second,
		EnableStats: true,
	})
	defer cache.Close()

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

	// First call should be a cache miss
	valid, err := cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	stats := cache.GetStats()
	if stats == nil {
		t.Fatal("Expected stats but got nil")
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	// Second call with same config should be a cache hit
	valid, err = cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}

	stats = cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	// Cache size should be 1
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}
}

// TestValidationCache_CacheExpiration tests cache expiration behavior
func TestValidationCache_CacheExpiration(t *testing.T) {
	cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
		MaxSize:     10,
		TTL:         100 * time.Millisecond, // Very short TTL for testing
		EnableStats: true,
	})
	defer cache.Close()

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

	// First call - cache miss
	valid, err := cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	// Second call - cache hit
	valid, err = cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Third call - should be cache miss again due to expiration
	valid, err = cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Third validation failed: %v", err)
	}

	stats := cache.GetStats()
	if stats.Expirations == 0 {
		t.Error("Expected expirations to be > 0")
	}
}

// TestValidationCache_CacheEviction tests cache eviction when size limit is reached
func TestValidationCache_CacheEviction(t *testing.T) {
	cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
		MaxSize:     2, // Very small size for testing
		TTL:         1 * time.Minute,
		EnableStats: true,
	})
	defer cache.Close()

	// Create 3 different configs
	configs := []*cachex.CacheConfig{
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     50,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLRU,
				EnableStats:     true,
			},
			DefaultTTL: 5 * time.Minute,
			MaxRetries: 1,
			RetryDelay: 100 * time.Millisecond,
			Codec:      "json",
		},
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         2000,
				MaxMemoryMB:     100,
				DefaultTTL:      10 * time.Minute,
				CleanupInterval: 2 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLFU,
				EnableStats:     false,
			},
			DefaultTTL: 10 * time.Minute,
			MaxRetries: 2,
			RetryDelay: 200 * time.Millisecond,
			Codec:      "msgpack",
		},
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         3000,
				MaxMemoryMB:     150,
				DefaultTTL:      15 * time.Minute,
				CleanupInterval: 3 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyTTL,
				EnableStats:     true,
			},
			DefaultTTL: 15 * time.Minute,
			MaxRetries: 3,
			RetryDelay: 300 * time.Millisecond,
			Codec:      "json",
		},
	}

	// Add all configs to cache
	for i, config := range configs {
		valid, err := cache.IsValid(config)
		if !valid || err != nil {
			t.Fatalf("Validation %d failed: %v", i, err)
		}
	}

	// Cache size should not exceed maxSize
	if cache.Size() > 2 {
		t.Errorf("Expected cache size <= 2, got %d", cache.Size())
	}

	stats := cache.GetStats()
	if stats.Evictions == 0 {
		t.Error("Expected evictions to be > 0")
	}
}

// TestValidationCache_Statistics tests statistics collection
func TestValidationCache_Statistics(t *testing.T) {
	tests := []struct {
		name        string
		enableStats bool
	}{
		{
			name:        "stats enabled",
			enableStats: true,
		},
		{
			name:        "stats disabled",
			enableStats: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
				MaxSize:     10,
				TTL:         1 * time.Minute,
				EnableStats: tt.enableStats,
			})
			defer cache.Close()

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

			// Perform some operations
			valid, err := cache.IsValid(config)
			if !valid || err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			// Cache hit
			valid, err = cache.IsValid(config)
			if !valid || err != nil {
				t.Fatalf("Second validation failed: %v", err)
			}

			stats := cache.GetStats()
			if tt.enableStats {
				if stats == nil {
					t.Fatal("Expected stats but got nil")
				}
				if stats.Hits == 0 {
					t.Error("Expected hits > 0")
				}
				if stats.Misses == 0 {
					t.Error("Expected misses > 0")
				}
			} else {
				if stats != nil {
					t.Error("Expected nil stats when disabled")
				}
			}
		})
	}
}

// TestValidationCache_Concurrency tests concurrent access
func TestValidationCache_Concurrency(t *testing.T) {
	cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
		MaxSize:     100,
		TTL:         1 * time.Minute,
		EnableStats: true,
	})
	defer cache.Close()

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

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				valid, err := cache.IsValid(config)
				if !valid || err != nil {
					t.Errorf("Concurrent validation failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Verify cache is still functional
	valid, err := cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Post-concurrency validation failed: %v", err)
	}

	stats := cache.GetStats()
	if stats == nil {
		t.Fatal("Expected stats but got nil")
	}
}

// TestValidationCache_ErrorHandling tests error handling scenarios
func TestValidationCache_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		cache         *cachex.ValidationCache
		config        *cachex.CacheConfig
		expectedError string
		expectedValid bool
	}{
		{
			name:          "nil cache",
			cache:         nil,
			config:        &cachex.CacheConfig{},
			expectedError: "validation cache is nil",
			expectedValid: false,
		},
		{
			name:          "nil config",
			cache:         cachex.NewValidationCache(nil),
			config:        nil,
			expectedError: "configuration is nil",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cache != nil {
				defer tt.cache.Close()
			}

			valid, err := tt.cache.IsValid(tt.config)
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}
			if err == nil && tt.expectedError != "" {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if err != nil && tt.expectedError != "" {
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

// TestValidationCache_HashGeneration tests hash generation functionality
func TestValidationCache_HashGeneration(t *testing.T) {
	cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
	defer cache.Close()

	// Test that identical configs generate the same hash
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

	// Both configs should be cached and result in cache hits
	valid1, err1 := cache.IsValid(config1)
	valid2, err2 := cache.IsValid(config2)

	if !valid1 || err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}
	if !valid2 || err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Should have 1 miss and 1 hit
	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	// Test that different configs generate different hashes
	config3 := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         2000, // Different value
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

	valid3, err3 := cache.IsValid(config3)
	if !valid3 || err3 != nil {
		t.Fatalf("Third validation failed: %v", err3)
	}

	// Should have 2 misses and 1 hit
	stats = cache.GetStats()
	if stats.Misses != 2 {
		t.Errorf("Expected 2 misses, got %d", stats.Misses)
	}
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

// TestValidationCache_Clear tests cache clearing functionality
func TestValidationCache_Clear(t *testing.T) {
	cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
	defer cache.Close()

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

	// Add item to cache
	valid, err := cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	// Next validation should be a cache miss
	valid, err = cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Post-clear validation failed: %v", err)
	}

	stats := cache.GetStats()
	if stats.Misses != 2 {
		t.Errorf("Expected 2 misses, got %d", stats.Misses)
	}
}

// TestValidationCache_Close tests cache closing functionality
func TestValidationCache_Close(t *testing.T) {
	cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())

	// Add some items to cache
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

	valid, err := cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Close cache
	cache.Close()

	// Cache should still be functional for reads
	valid, err = cache.IsValid(config)
	if !valid || err != nil {
		t.Fatalf("Post-close validation failed: %v", err)
	}
}

// TestValidationCache_Size tests cache size functionality
func TestValidationCache_Size(t *testing.T) {
	cache := cachex.NewValidationCache(&cachex.ValidationCacheConfig{
		MaxSize:     10,
		TTL:         1 * time.Minute,
		EnableStats: true,
	})
	defer cache.Close()

	// Initially empty
	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	// Add items
	configs := []*cachex.CacheConfig{
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     50,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLRU,
				EnableStats:     true,
			},
			DefaultTTL: 5 * time.Minute,
			MaxRetries: 1,
			RetryDelay: 100 * time.Millisecond,
			Codec:      "json",
		},
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         2000,
				MaxMemoryMB:     100,
				DefaultTTL:      10 * time.Minute,
				CleanupInterval: 2 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLFU,
				EnableStats:     false,
			},
			DefaultTTL: 10 * time.Minute,
			MaxRetries: 2,
			RetryDelay: 200 * time.Millisecond,
			Codec:      "msgpack",
		},
	}

	for i, config := range configs {
		valid, err := cache.IsValid(config)
		if !valid || err != nil {
			t.Fatalf("Validation %d failed: %v", i, err)
		}
	}

	// Size should reflect number of unique configs
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

// TestValidationCache_NilReceiver tests behavior with nil receiver
func TestValidationCache_NilReceiver(t *testing.T) {
	var cache *cachex.ValidationCache

	// Test methods on nil receiver
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

	valid, err := cache.IsValid(config)
	if valid {
		t.Error("Expected valid=false for nil receiver")
	}
	if err == nil {
		t.Error("Expected error for nil receiver")
	}

	stats := cache.GetStats()
	if stats != nil {
		t.Error("Expected nil stats for nil receiver")
	}

	size := cache.Size()
	if size != 0 {
		t.Errorf("Expected size 0 for nil receiver, got %d", size)
	}

	// These should not panic
	cache.Clear()
	cache.Close()
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
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
