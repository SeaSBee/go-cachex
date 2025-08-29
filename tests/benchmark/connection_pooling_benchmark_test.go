package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// BenchmarkConnectionPooling tests the performance improvements of optimized connection pooling
func BenchmarkConnectionPooling(b *testing.B) {
	ctx := context.Background()

	b.Run("StandardRedisStore_Get", func(b *testing.B) {
		// Create standard Redis store
		config := &cachex.RedisConfig{
			Addr:                "localhost:6379",
			Password:            "",
			DB:                  0,
			PoolSize:            10,
			MinIdleConns:        5,
			MaxRetries:          3,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			EnablePipelining:    true,
			EnableMetrics:       true,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		store, err := cachex.NewRedisStore(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer store.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-connection-pooling-benchmark")
		setResult := <-store.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := <-store.Get(ctx, "test-key")
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			if !result.Exists {
				b.Fatalf("Expected key to exist")
			}
		}
	})

	b.Run("OptimizedConnectionPool_Get", func(b *testing.B) {
		// Create optimized connection pool
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                     "localhost:6379",
			Password:                 "",
			DB:                       0,
			PoolSize:                 20,
			MinIdleConns:             10,
			MaxRetries:               3,
			PoolTimeout:              30 * time.Second,
			DialTimeout:              5 * time.Second,
			ReadTimeout:              3 * time.Second,
			WriteTimeout:             3 * time.Second,
			EnablePipelining:         true,
			EnableMetrics:            true,
			EnableConnectionPool:     true,
			HealthCheckInterval:      30 * time.Second,
			HealthCheckTimeout:       5 * time.Second,
			EnableConnectionReuse:    true,
			EnableConnectionWarming:  true,
			ConnectionWarmingTimeout: 10 * time.Second,
			EnableLoadBalancing:      false,
			EnableCircuitBreaker:     true,
			EnablePoolMonitoring:     false, // Disable for benchmarks
			MonitoringInterval:       30 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer pool.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-optimized-connection-pooling-benchmark")
		setResult := <-pool.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := <-pool.Get(ctx, "test-key")
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			if !result.Exists {
				b.Fatalf("Expected key to exist")
			}
		}
	})

	b.Run("StandardRedisStore_Set", func(b *testing.B) {
		// Create standard Redis store
		config := &cachex.RedisConfig{
			Addr:                "localhost:6379",
			Password:            "",
			DB:                  0,
			PoolSize:            10,
			MinIdleConns:        5,
			MaxRetries:          3,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			EnablePipelining:    true,
			EnableMetrics:       true,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		store, err := cachex.NewRedisStore(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer store.Close()

		testData := []byte("test-value-for-set-benchmark")

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "test-key-" + string(rune(i%1000))
			result := <-store.Set(ctx, key, testData, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
		}
	})

	b.Run("OptimizedConnectionPool_Set", func(b *testing.B) {
		// Create optimized connection pool
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                     "localhost:6379",
			Password:                 "",
			DB:                       0,
			PoolSize:                 20,
			MinIdleConns:             10,
			MaxRetries:               3,
			PoolTimeout:              30 * time.Second,
			DialTimeout:              5 * time.Second,
			ReadTimeout:              3 * time.Second,
			WriteTimeout:             3 * time.Second,
			EnablePipelining:         true,
			EnableMetrics:            true,
			EnableConnectionPool:     true,
			HealthCheckInterval:      30 * time.Second,
			HealthCheckTimeout:       5 * time.Second,
			EnableConnectionReuse:    true,
			EnableConnectionWarming:  true,
			ConnectionWarmingTimeout: 10 * time.Second,
			EnableLoadBalancing:      false,
			EnableCircuitBreaker:     true,
			EnablePoolMonitoring:     false, // Disable for benchmarks
			MonitoringInterval:       30 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer pool.Close()

		testData := []byte("test-value-for-optimized-set-benchmark")

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "test-key-" + string(rune(i%1000))
			result := <-pool.Set(ctx, key, testData, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
		}
	})
}

// BenchmarkConnectionPoolingConcurrent tests concurrent performance
func BenchmarkConnectionPoolingConcurrent(b *testing.B) {
	ctx := context.Background()

	b.Run("StandardRedisStore_ConcurrentGet", func(b *testing.B) {
		// Create standard Redis store
		config := &cachex.RedisConfig{
			Addr:                "localhost:6379",
			Password:            "",
			DB:                  0,
			PoolSize:            10,
			MinIdleConns:        5,
			MaxRetries:          3,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			EnablePipelining:    true,
			EnableMetrics:       true,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		store, err := cachex.NewRedisStore(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer store.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-concurrent-benchmark")
		setResult := <-store.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				result := <-store.Get(ctx, "test-key")
				if result.Error != nil {
					b.Fatalf("Get failed: %v", result.Error)
				}
				if !result.Exists {
					b.Fatalf("Expected key to exist")
				}
			}
		})
	})

	b.Run("OptimizedConnectionPool_ConcurrentGet", func(b *testing.B) {
		// Create optimized connection pool
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                     "localhost:6379",
			Password:                 "",
			DB:                       0,
			PoolSize:                 20,
			MinIdleConns:             10,
			MaxRetries:               3,
			PoolTimeout:              30 * time.Second,
			DialTimeout:              5 * time.Second,
			ReadTimeout:              3 * time.Second,
			WriteTimeout:             3 * time.Second,
			EnablePipelining:         true,
			EnableMetrics:            true,
			EnableConnectionPool:     true,
			HealthCheckInterval:      30 * time.Second,
			HealthCheckTimeout:       5 * time.Second,
			EnableConnectionReuse:    true,
			EnableConnectionWarming:  true,
			ConnectionWarmingTimeout: 10 * time.Second,
			EnableLoadBalancing:      false,
			EnableCircuitBreaker:     true,
			EnablePoolMonitoring:     false, // Disable for benchmarks
			MonitoringInterval:       30 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer pool.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-optimized-concurrent-benchmark")
		setResult := <-pool.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				result := <-pool.Get(ctx, "test-key")
				if result.Error != nil {
					b.Fatalf("Get failed: %v", result.Error)
				}
				if !result.Exists {
					b.Fatalf("Expected key to exist")
				}
			}
		})
	})
}

// BenchmarkConnectionPoolingHighPerformance tests high-performance configurations
func BenchmarkConnectionPoolingHighPerformance(b *testing.B) {
	ctx := context.Background()

	b.Run("HighPerformanceRedisStore_Get", func(b *testing.B) {
		// Create high-performance Redis store
		config := cachex.HighPerformanceRedisConfig()

		store, err := cachex.NewRedisStore(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer store.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-high-performance-benchmark")
		setResult := <-store.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := <-store.Get(ctx, "test-key")
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			if !result.Exists {
				b.Fatalf("Expected key to exist")
			}
		}
	})

	b.Run("HighPerformanceConnectionPool_Get", func(b *testing.B) {
		// Create high-performance connection pool
		config := cachex.HighPerformanceConnectionPoolConfig()
		config.EnablePoolMonitoring = false // Disable for benchmarks

		pool, err := cachex.NewOptimizedConnectionPool(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer pool.Close()

		// Pre-populate with test data
		testData := []byte("test-value-for-high-performance-pool-benchmark")
		setResult := <-pool.Set(ctx, "test-key", testData, 5*time.Minute)
		if setResult.Error != nil {
			b.Fatalf("Failed to set test data: %v", setResult.Error)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := <-pool.Get(ctx, "test-key")
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			if !result.Exists {
				b.Fatalf("Expected key to exist")
			}
		}
	})
}

// BenchmarkConnectionPoolingStats tests statistics collection performance
func BenchmarkConnectionPoolingStats(b *testing.B) {
	b.Run("StandardRedisStore_Stats", func(b *testing.B) {
		// Create standard Redis store
		config := &cachex.RedisConfig{
			Addr:                "localhost:6379",
			Password:            "",
			DB:                  0,
			PoolSize:            10,
			MinIdleConns:        5,
			MaxRetries:          3,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			EnablePipelining:    true,
			EnableMetrics:       true,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		store, err := cachex.NewRedisStore(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer store.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			stats := store.GetStats()
			if stats == nil {
				b.Fatalf("Expected stats to be non-nil")
			}
		}
	})

	b.Run("OptimizedConnectionPool_Stats", func(b *testing.B) {
		// Create optimized connection pool
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                     "localhost:6379",
			Password:                 "",
			DB:                       0,
			PoolSize:                 20,
			MinIdleConns:             10,
			MaxRetries:               3,
			PoolTimeout:              30 * time.Second,
			DialTimeout:              5 * time.Second,
			ReadTimeout:              3 * time.Second,
			WriteTimeout:             3 * time.Second,
			EnablePipelining:         true,
			EnableMetrics:            true,
			EnableConnectionPool:     true,
			HealthCheckInterval:      30 * time.Second,
			HealthCheckTimeout:       5 * time.Second,
			EnableConnectionReuse:    true,
			EnableConnectionWarming:  true,
			ConnectionWarmingTimeout: 10 * time.Second,
			EnableLoadBalancing:      false,
			EnableCircuitBreaker:     true,
			EnablePoolMonitoring:     false, // Disable for benchmarks
			MonitoringInterval:       30 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		if err != nil {
			b.Skipf("Skipping benchmark - Redis not available: %v", err)
		}
		defer pool.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			stats := pool.GetStats()
			if stats == nil {
				b.Fatalf("Expected stats to be non-nil")
			}
		}
	})
}
