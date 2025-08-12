package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/codec"
	"github.com/SeaSBee/go-cachex/cachex/pkg/key"
	"github.com/SeaSBee/go-cachex/cachex/pkg/layeredstore"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	fmt.Println("=== Go-CacheX Local In-Memory Cache Example ===")

	// Example 1: Pure Memory Store
	demoPureMemoryStore()

	// Example 2: Layered Cache (Memory + Redis)
	demoLayeredCache()

	// Example 3: Different Configurations
	demoConfigurations()

	// Example 4: Performance Comparison
	demoPerformanceComparison()

	fmt.Println("\n=== Local In-Memory Cache Example Complete ===")
}

func demoPureMemoryStore() {
	fmt.Println("1. Pure Memory Store")
	fmt.Println("====================")

	// Create memory store with default configuration
	memoryStore, err := memorystore.New(memorystore.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create memory store: %v", err)
	}
	defer memoryStore.Close()

	// Create cache with memory store
	c, err := cache.New[User](
		cache.WithStore(memoryStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithKeyBuilder(key.NewBuilder("demo", "local", "secret123")),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// Basic operations
	user := User{ID: 123, Name: "John Doe", Email: "john@example.com"}
	userKey := "user:123"

	fmt.Printf("Setting user: %+v\n", user)
	err = c.Set(userKey, user, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
	}

	fmt.Println("Getting user from memory cache...")
	cachedUser, found, err := c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Printf("Found user in memory: %+v\n", cachedUser)
	} else {
		fmt.Println("User not found in memory cache")
	}

	// Check statistics
	stats := memoryStore.GetStats()
	fmt.Printf("Memory store stats - Hits: %d, Misses: %d, Size: %d, Memory Usage: %d bytes\n",
		stats.Hits, stats.Misses, stats.Size, stats.MemoryUsage)

	fmt.Println()
}

func demoLayeredCache() {
	fmt.Println("2. Layered Cache (Memory + Redis)")
	fmt.Println("=================================")

	// Create Redis store (L2)
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		fmt.Printf("Warning: Redis not available, skipping layered cache demo: %v\n", err)
		fmt.Println()
		return
	}
	defer redisStore.Close()

	// Create layered store configuration
	layeredConfig := layeredstore.HighPerformanceConfig()
	layeredConfig.WritePolicy = layeredstore.WritePolicyBehind
	layeredConfig.ReadPolicy = layeredstore.ReadPolicyThrough

	// Create layered store
	layeredStore, err := layeredstore.New(redisStore, layeredConfig)
	if err != nil {
		log.Fatalf("Failed to create layered store: %v", err)
	}
	defer layeredStore.Close()

	// Create cache with layered store
	c, err := cache.New[User](
		cache.WithStore(layeredStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithKeyBuilder(key.NewBuilder("demo", "layered", "secret123")),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// Test layered caching
	user := User{ID: 456, Name: "Jane Smith", Email: "jane@example.com"}
	userKey := "user:456"

	fmt.Printf("Setting user in layered cache: %+v\n", user)
	err = c.Set(userKey, user, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
	}

	// First read (should populate L1 from L2)
	fmt.Println("First read (populating L1 from L2)...")
	cachedUser, found, err := c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Printf("Found user: %+v\n", cachedUser)
	}

	// Second read (should hit L1)
	fmt.Println("Second read (should hit L1)...")
	cachedUser, found, err = c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Printf("Found user: %+v\n", cachedUser)
	}

	// Check layered store statistics
	stats := layeredStore.GetStats()
	fmt.Printf("Layered store stats - L1 Hits: %d, L1 Misses: %d, L2 Hits: %d, L2 Misses: %d\n",
		stats.L1Hits, stats.L1Misses, stats.L2Hits, stats.L2Misses)

	fmt.Println()
}

func demoConfigurations() {
	fmt.Println("3. Different Memory Store Configurations")
	fmt.Println("========================================")

	// Test different configurations
	configs := map[string]*memorystore.Config{
		"Default":              memorystore.DefaultConfig(),
		"High Performance":     memorystore.HighPerformanceConfig(),
		"Resource Constrained": memorystore.ResourceConstrainedConfig(),
	}

	for name, config := range configs {
		fmt.Printf("\n%s Configuration:\n", name)
		fmt.Printf("  Max Size: %d items\n", config.MaxSize)
		fmt.Printf("  Max Memory: %d MB\n", config.MaxMemoryMB)
		fmt.Printf("  Default TTL: %v\n", config.DefaultTTL)
		fmt.Printf("  Eviction Policy: %s\n", config.EvictionPolicy)
		fmt.Printf("  Cleanup Interval: %v\n", config.CleanupInterval)

		// Create store with this configuration
		store, err := memorystore.New(config)
		if err != nil {
			log.Printf("Failed to create %s store: %v", name, err)
			continue
		}

		// Test basic operations
		ctx := context.Background()
		testKey := fmt.Sprintf("test:%s", name)
		testValue := []byte(fmt.Sprintf("value for %s", name))

		err = store.Set(ctx, testKey, testValue, time.Minute)
		if err != nil {
			log.Printf("Failed to set in %s store: %v", name, err)
		}

		result, err := store.Get(ctx, testKey)
		if err != nil {
			log.Printf("Failed to get from %s store: %v", name, err)
		} else {
			fmt.Printf("  Test result: %s\n", string(result))
		}

		store.Close()
	}

	fmt.Println()
}

func demoPerformanceComparison() {
	fmt.Println("4. Performance Comparison")
	fmt.Println("=========================")

	// Create memory store
	memoryStore, err := memorystore.New(memorystore.HighPerformanceConfig())
	if err != nil {
		log.Fatalf("Failed to create memory store: %v", err)
	}
	defer memoryStore.Close()

	ctx := context.Background()

	// Performance test parameters
	const numOperations = 10000
	const numKeys = 1000

	fmt.Printf("Running performance test with %d operations on %d keys...\n", numOperations, numKeys)

	// Test write performance
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf:key:%d", i%numKeys)
		value := []byte(fmt.Sprintf("value:%d", i))
		err := memoryStore.Set(ctx, key, value, time.Minute)
		if err != nil {
			log.Printf("Write error: %v", err)
		}
	}
	writeDuration := time.Since(start)

	// Test read performance
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf:key:%d", i%numKeys)
		_, err := memoryStore.Get(ctx, key)
		if err != nil {
			log.Printf("Read error: %v", err)
		}
	}
	readDuration := time.Since(start)

	// Calculate performance metrics
	writeOpsPerSec := float64(numOperations) / writeDuration.Seconds()
	readOpsPerSec := float64(numOperations) / readDuration.Seconds()

	fmt.Printf("Write Performance: %.2f ops/sec (%.2f ms avg)\n", writeOpsPerSec, float64(writeDuration.Milliseconds())/float64(numOperations))
	fmt.Printf("Read Performance:  %.2f ops/sec (%.2f ms avg)\n", readOpsPerSec, float64(readDuration.Milliseconds())/float64(numOperations))

	// Show final statistics
	stats := memoryStore.GetStats()
	fmt.Printf("Final Stats - Hits: %d, Misses: %d, Size: %d, Memory Usage: %d bytes\n",
		stats.Hits, stats.Misses, stats.Size, stats.MemoryUsage)

	fmt.Println()
}
