package benchmark

import (
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// BenchmarkValidationCachePerformance compares performance with and without validation caching
func BenchmarkValidationCachePerformance(b *testing.B) {
	// Create a valid configuration for testing
	validConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
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

	// Create an invalid configuration for testing (multiple stores)
	invalidConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
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
	}

	b.Run("ValidConfig_WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		cachex.GlobalValidationCache.Clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use direct validation instead of creating cache
			valid, err := cachex.GlobalValidationCache.IsValid(validConfig)
			if !valid || err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})

	b.Run("ValidConfig_WithoutCache", func(b *testing.B) {
		// Disable cache for this benchmark
		originalCache := cachex.GlobalValidationCache
		cachex.GlobalValidationCache = nil
		defer func() { cachex.GlobalValidationCache = originalCache }()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a temporary cache for validation
			tempCache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
			valid, err := tempCache.IsValid(validConfig)
			if !valid || err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})

	b.Run("InvalidConfig_WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		cachex.GlobalValidationCache.Clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use direct validation instead of creating cache
			valid, err := cachex.GlobalValidationCache.IsValid(invalidConfig)
			if valid {
				b.Fatal("Expected validation error but got none")
			}
			if err == nil {
				b.Fatal("Expected validation error but got nil")
			}
		}
	})

	b.Run("InvalidConfig_WithoutCache", func(b *testing.B) {
		// Disable cache for this benchmark
		originalCache := cachex.GlobalValidationCache
		cachex.GlobalValidationCache = nil
		defer func() { cachex.GlobalValidationCache = originalCache }()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a temporary cache for validation
			tempCache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
			valid, err := tempCache.IsValid(invalidConfig)
			if valid {
				b.Fatal("Expected validation error but got none")
			}
			if err == nil {
				b.Fatal("Expected validation error but got nil")
			}
		}
	})
}

// BenchmarkValidationCacheHitRate tests cache hit rates with repeated configurations
func BenchmarkValidationCacheHitRate(b *testing.B) {
	// Create multiple different configurations (memory only to avoid Redis)
	configs := []*cachex.CacheConfig{
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     50,
				DefaultTTL:      2 * time.Minute,
				CleanupInterval: 30 * time.Second,
				EvictionPolicy:  cachex.EvictionPolicyLRU,
				EnableStats:     true,
			},
			DefaultTTL: 2 * time.Minute,
			MaxRetries: 2,
			RetryDelay: 50 * time.Millisecond,
			Codec:      "json",
		},
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         5000,
				MaxMemoryMB:     200,
				DefaultTTL:      10 * time.Minute,
				CleanupInterval: 2 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLFU,
				EnableStats:     false,
			},
			DefaultTTL: 10 * time.Minute,
			MaxRetries: 5,
			RetryDelay: 200 * time.Millisecond,
			Codec:      "msgpack",
		},
		{
			Memory: &cachex.MemoryConfig{
				MaxSize:         2000,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyTTL,
				EnableStats:     true,
			},
			DefaultTTL: 5 * time.Minute,
			MaxRetries: 3,
			RetryDelay: 100 * time.Millisecond,
			Codec:      "json",
		},
	}

	b.Run("MixedConfigs_WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		cachex.GlobalValidationCache.Clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			config := configs[i%len(configs)]
			valid, err := cachex.GlobalValidationCache.IsValid(config)
			if !valid || err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})

	b.Run("MixedConfigs_WithoutCache", func(b *testing.B) {
		// Disable cache for this benchmark
		originalCache := cachex.GlobalValidationCache
		cachex.GlobalValidationCache = nil
		defer func() { cachex.GlobalValidationCache = originalCache }()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			config := configs[i%len(configs)]
			// Create a temporary cache for validation
			tempCache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
			valid, err := tempCache.IsValid(config)
			if !valid || err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})
}

// BenchmarkValidationCacheConcurrent tests concurrent access to validation cache
func BenchmarkValidationCacheConcurrent(b *testing.B) {
	validConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
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

	b.Run("Concurrent_WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		cachex.GlobalValidationCache.Clear()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				valid, err := cachex.GlobalValidationCache.IsValid(validConfig)
				if !valid || err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
			}
		})
	})

	b.Run("Concurrent_WithoutCache", func(b *testing.B) {
		// Disable cache for this benchmark
		originalCache := cachex.GlobalValidationCache
		cachex.GlobalValidationCache = nil
		defer func() { cachex.GlobalValidationCache = originalCache }()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Create a temporary cache for validation
				tempCache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())
				valid, err := tempCache.IsValid(validConfig)
				if !valid || err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkValidationCacheStats tests cache statistics collection
func BenchmarkValidationCacheStats(b *testing.B) {
	validConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
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

	b.Run("StatsCollection", func(b *testing.B) {
		// Clear cache before benchmark
		cachex.GlobalValidationCache.Clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Perform validation
			valid, err := cachex.GlobalValidationCache.IsValid(validConfig)
			if !valid || err != nil {
				b.Fatalf("Validation failed: %v", err)
			}

			// Get stats (this should be fast due to caching)
			stats := cachex.GlobalValidationCache.GetStats()
			if stats == nil {
				b.Fatal("Expected stats but got nil")
			}
		}
	})
}

// BenchmarkValidationCacheMemoryUsage tests memory usage of validation cache
func BenchmarkValidationCacheMemoryUsage(b *testing.B) {
	b.Run("MemoryUsage", func(b *testing.B) {
		b.ReportAllocs()

		// Create a new validation cache
		cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())

		// Create multiple configurations
		for i := 0; i < 100; i++ {
			config := &cachex.CacheConfig{
				Memory: &cachex.MemoryConfig{
					MaxSize:         i + 1000,
					MaxMemoryMB:     50 + i,
					DefaultTTL:      time.Duration(i+1) * time.Minute,
					CleanupInterval: time.Duration(i+1) * 30 * time.Second,
					EvictionPolicy:  cachex.EvictionPolicyLRU,
					EnableStats:     true,
				},
				DefaultTTL: time.Duration(i+1) * time.Minute,
				MaxRetries: i % 5,
				RetryDelay: time.Duration(i+1) * 10 * time.Millisecond,
				Codec:      "json",
			}

			// Validate and cache
			_, err := cache.IsValid(config)
			if err != nil {
				b.Fatalf("Failed to validate config: %v", err)
			}
		}

		// Check cache size
		size := cache.Size()
		if size == 0 {
			b.Fatal("Expected cached items but got 0")
		}
	})
}

// BenchmarkValidationCacheHashGeneration tests the performance of config hash generation
func BenchmarkValidationCacheHashGeneration(b *testing.B) {
	config := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
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

	b.Run("HashGeneration", func(b *testing.B) {
		b.ReportAllocs()

		cache := cachex.NewValidationCache(cachex.DefaultValidationCacheConfig())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This will trigger hash generation
			_, err := cache.IsValid(config)
			if err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})
}
