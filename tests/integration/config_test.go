package integration

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Helper function to create Redis store for config testing
func createRedisStoreForConfig(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
		return nil
	}

	return store
}

// Test cache configuration loading
func TestCacheConfigLoading(t *testing.T) {
	store := createRedisStoreForConfig(t)
	if store != nil {
		defer store.Close()
	}

	// Create cache from config
	config := &cachex.CacheConfig{
		Redis: &cachex.RedisConfig{
			Addr: "localhost:6379",
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

	cache, err := cachex.NewFromConfig[string](config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}
	defer cache.Close()

	// Test basic operations
	ctx := context.Background()
	key := "config-test-key"
	value := "config-test-value"

	// Set value
	setResult := <-cache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	// Get value
	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	logx.Info("Cache config loading test completed")
}

// Test cache configuration with different stores
func TestCacheConfigWithDifferentStores(t *testing.T) {
	// Test with memory store
	memoryConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{
			MaxSize:         1000,
			MaxMemoryMB:     10,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		DefaultTTL: 5 * time.Minute,
		Codec:      "json",
	}

	memoryCache, err := cachex.NewFromConfig[string](memoryConfig)
	if err != nil {
		t.Fatalf("Failed to create memory cache from config: %v", err)
	}
	defer memoryCache.Close()

	// Test memory cache operations
	ctx := context.Background()
	key := "memory-config-key"
	value := "memory-config-value"

	setResult := <-memoryCache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in memory cache: %v", setResult.Error)
	}

	getResult := <-memoryCache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value from memory cache: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found in memory cache")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	// Test with Redis store (if available)
	store := createRedisStoreForConfig(t)
	if store != nil {
		defer store.Close()

		redisConfig := &cachex.CacheConfig{
			Redis: &cachex.RedisConfig{
				Addr: "localhost:6379",
			},
			DefaultTTL: 5 * time.Minute,
			Codec:      "json",
		}

		redisCache, err := cachex.NewFromConfig[string](redisConfig)
		if err != nil {
			t.Skipf("Skipping Redis test - Redis not available: %v", err)
		}
		defer redisCache.Close()

		// Test Redis cache operations
		redisKey := "redis-config-key"
		redisValue := "redis-config-value"

		setResult = <-redisCache.Set(ctx, redisKey, redisValue, 5*time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set value in Redis cache: %v", setResult.Error)
		}

		getResult = <-redisCache.Get(ctx, redisKey)
		if getResult.Error != nil {
			t.Fatalf("Failed to get value from Redis cache: %v", getResult.Error)
		}
		if !getResult.Found {
			t.Fatal("Value not found in Redis cache")
		}

		if getResult.Value != redisValue {
			t.Errorf("Expected value %s, got %s", redisValue, getResult.Value)
		}
	}

	logx.Info("Cache config with different stores test completed")
}

// Test cache configuration with observability
func TestCacheConfigWithObservability(t *testing.T) {
	store := createRedisStoreForConfig(t)
	if store != nil {
		defer store.Close()
	}

	// Create cache with observability config
	config := &cachex.CacheConfig{
		Redis: &cachex.RedisConfig{
			Addr: "localhost:6379",
		},
		DefaultTTL: 5 * time.Minute,
		Observability: &cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		},
	}

	cache, err := cachex.NewFromConfig[string](config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}
	defer cache.Close()

	// Test basic operations with observability
	ctx := context.Background()
	key := "obs-config-key"
	value := "obs-config-value"

	setResult := <-cache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	// Test Exists operation
	existsResult := <-cache.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence: %v", existsResult.Error)
	}
	if !existsResult.Found {
		t.Error("Key should exist")
	}

	logx.Info("Cache config with observability test completed")
}
