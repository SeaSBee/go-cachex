package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/codec"
	"github.com/SeaSBee/go-cachex/cachex/pkg/key"
	"github.com/SeaSBee/go-cachex/cachex/pkg/ristretto"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Ristretto Local Hot Cache Example ===")

	// Example 1: Basic Ristretto Usage
	demoBasicRistretto()

	// Example 2: Different Configurations
	demoConfigurations()

	// Example 3: Performance Comparison
	demoPerformanceComparison()

	// Example 4: Statistics and Monitoring
	demoStatistics()

	fmt.Println("\n=== Ristretto Local Hot Cache Example Complete ===")
}

func demoBasicRistretto() {
	fmt.Println("1. Basic Ristretto Usage")
	fmt.Println("=========================")

	// Create Ristretto store with default configuration
	ristrettoStore, err := ristretto.New(ristretto.DefaultConfig())
	if err != nil {
		log.Printf("Failed to create Ristretto store: %v", err)
		return
	}
	defer ristrettoStore.Close()

	// Create key builder
	keyBuilder := key.NewBuilder("myapp", "dev", "secret123")

	// Create cache with Ristretto store
	c, err := cache.New[User](
		cache.WithStore(ristrettoStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithKeyBuilder(keyBuilder),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        "ristretto:1",
		Name:      "Ristretto User",
		Email:     "ristretto@example.com",
		CreatedAt: time.Now(),
	}

	// Set user in cache
	userKey := keyBuilder.BuildUser("1")
	fmt.Printf("Setting user in Ristretto cache: %+v\n", user)
	err = c.Set(userKey, user, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
		return
	}

	// Get user from cache
	fmt.Println("Retrieving user from Ristretto cache...")
	cachedUser, found, err := c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	if found {
		fmt.Printf("Found user: %+v\n", cachedUser)
	} else {
		fmt.Println("User not found in cache")
	}

	// Test multiple operations
	fmt.Println("\nTesting multiple operations...")
	for i := 2; i <= 5; i++ {
		user := User{
			ID:        fmt.Sprintf("ristretto:%d", i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now(),
		}
		key := keyBuilder.BuildUser(fmt.Sprintf("%d", i))

		if err := c.Set(key, user, 5*time.Minute); err != nil {
			log.Printf("Failed to set user %d: %v", i, err)
			continue
		}
		fmt.Printf("  - Set user %d\n", i)
	}

	// Test MGet
	fmt.Println("\nTesting MGet operation...")
	keys := []string{
		keyBuilder.BuildUser("1"),
		keyBuilder.BuildUser("2"),
		keyBuilder.BuildUser("3"),
		keyBuilder.BuildUser("4"),
		keyBuilder.BuildUser("5"),
	}

	users, err := c.MGet(keys...)
	if err != nil {
		log.Printf("Failed to MGet users: %v", err)
	} else {
		fmt.Printf("Retrieved %d users via MGet\n", len(users))
		for key, user := range users {
			fmt.Printf("  - %s: %s\n", key, user.Name)
		}
	}
}

func demoConfigurations() {
	fmt.Println("\n2. Different Configurations")
	fmt.Println("============================")

	// High Performance Configuration
	fmt.Println("High Performance Configuration:")
	highPerfStore, err := ristretto.New(ristretto.HighPerformanceConfig())
	if err != nil {
		log.Printf("Failed to create high performance store: %v", err)
	} else {
		defer highPerfStore.Close()
		fmt.Println("  ✓ High performance Ristretto store created")
		fmt.Printf("  - Max Items: %d\n", ristretto.HighPerformanceConfig().MaxItems)
		fmt.Printf("  - Max Memory: %d MB\n", ristretto.HighPerformanceConfig().MaxMemoryBytes/(1024*1024))
		fmt.Printf("  - Default TTL: %v\n", ristretto.HighPerformanceConfig().DefaultTTL)
	}

	// Resource Constrained Configuration
	fmt.Println("\nResource Constrained Configuration:")
	resourceStore, err := ristretto.New(ristretto.ResourceConstrainedConfig())
	if err != nil {
		log.Printf("Failed to create resource constrained store: %v", err)
	} else {
		defer resourceStore.Close()
		fmt.Println("  ✓ Resource constrained Ristretto store created")
		fmt.Printf("  - Max Items: %d\n", ristretto.ResourceConstrainedConfig().MaxItems)
		fmt.Printf("  - Max Memory: %d MB\n", ristretto.ResourceConstrainedConfig().MaxMemoryBytes/(1024*1024))
		fmt.Printf("  - Default TTL: %v\n", ristretto.ResourceConstrainedConfig().DefaultTTL)
	}

	// Custom Configuration
	fmt.Println("\nCustom Configuration:")
	customConfig := &ristretto.Config{
		MaxItems:       5000,
		MaxMemoryBytes: 50 * 1024 * 1024, // 50MB
		DefaultTTL:     3 * time.Minute,
		NumCounters:    50000,
		BufferItems:    64,
		EnableMetrics:  true,
		EnableStats:    true,
	}

	customStore, err := ristretto.New(customConfig)
	if err != nil {
		log.Printf("Failed to create custom store: %v", err)
	} else {
		defer customStore.Close()
		fmt.Println("  ✓ Custom Ristretto store created")
		fmt.Printf("  - Max Items: %d\n", customConfig.MaxItems)
		fmt.Printf("  - Max Memory: %d MB\n", customConfig.MaxMemoryBytes/(1024*1024))
		fmt.Printf("  - Default TTL: %v\n", customConfig.DefaultTTL)
	}
}

func demoPerformanceComparison() {
	fmt.Println("\n3. Performance Comparison")
	fmt.Println("==========================")

	// Create stores for comparison
	ristrettoStore, err := ristretto.New(ristretto.HighPerformanceConfig())
	if err != nil {
		log.Printf("Failed to create Ristretto store: %v", err)
		return
	}
	defer ristrettoStore.Close()

	// Create cache with Ristretto
	c, err := cache.New[User](
		cache.WithStore(ristrettoStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Performance test with 1000 users
	fmt.Println("Testing performance with 1000 users...")

	// Set operations
	start := time.Now()
	for i := 0; i < 1000; i++ {
		user := User{
			ID:        fmt.Sprintf("perf:%d", i),
			Name:      fmt.Sprintf("Performance User %d", i),
			Email:     fmt.Sprintf("perf%d@example.com", i),
			CreatedAt: time.Now(),
		}
		key := fmt.Sprintf("user:perf:%d", i)

		if err := c.Set(key, user, 5*time.Minute); err != nil {
			log.Printf("Failed to set user %d: %v", i, err)
		}
	}
	setDuration := time.Since(start)
	fmt.Printf("Set 1000 users: %v (%.2f ops/sec)\n", setDuration, 1000.0/setDuration.Seconds())

	// Get operations
	start = time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:perf:%d", i)
		_, found, err := c.Get(key)
		if err != nil {
			log.Printf("Failed to get user %d: %v", i, err)
		} else if !found {
			log.Printf("User %d not found", i)
		}
	}
	getDuration := time.Since(start)
	fmt.Printf("Get 1000 users: %v (%.2f ops/sec)\n", getDuration, 1000.0/getDuration.Seconds())

	// MGet operations
	start = time.Now()
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("user:perf:%d", i)
	}

	users, err := c.MGet(keys...)
	if err != nil {
		log.Printf("Failed to MGet users: %v", err)
	} else {
		mgetDuration := time.Since(start)
		fmt.Printf("MGet 100 users: %v (%.2f ops/sec)\n", mgetDuration, 100.0/mgetDuration.Seconds())
		fmt.Printf("Retrieved %d users via MGet\n", len(users))
	}
}

func demoStatistics() {
	fmt.Println("\n4. Statistics and Monitoring")
	fmt.Println("=============================")

	// Create Ristretto store with statistics enabled
	config := ristretto.DefaultConfig()
	config.EnableStats = true
	config.EnableMetrics = true

	ristrettoStore, err := ristretto.New(config)
	if err != nil {
		log.Printf("Failed to create Ristretto store: %v", err)
		return
	}
	defer ristrettoStore.Close()

	// Create cache
	c, err := cache.New[User](
		cache.WithStore(ristrettoStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Perform some operations
	fmt.Println("Performing operations to generate statistics...")
	for i := 0; i < 100; i++ {
		user := User{
			ID:        fmt.Sprintf("stats:%d", i),
			Name:      fmt.Sprintf("Stats User %d", i),
			Email:     fmt.Sprintf("stats%d@example.com", i),
			CreatedAt: time.Now(),
		}
		key := fmt.Sprintf("user:stats:%d", i)

		// Set user
		if err := c.Set(key, user, 5*time.Minute); err != nil {
			log.Printf("Failed to set user %d: %v", i, err)
			continue
		}

		// Get user (hit)
		if _, found, err := c.Get(key); err != nil {
			log.Printf("Failed to get user %d: %v", i, err)
		} else if !found {
			log.Printf("User %d not found after set", i)
		}

		// Try to get non-existent user (miss)
		nonExistentKey := fmt.Sprintf("user:stats:non-existent:%d", i)
		if _, found, err := c.Get(nonExistentKey); err != nil {
			log.Printf("Failed to get non-existent user %d: %v", i, err)
		} else if found {
			log.Printf("Non-existent user %d was found (unexpected)", i)
		}
	}

	// Get and display statistics
	time.Sleep(1 * time.Second) // Allow time for stats to update
	stats := ristrettoStore.GetStats()

	fmt.Println("\nRistretto Cache Statistics:")
	fmt.Printf("  - Hits: %d\n", stats.Hits)
	fmt.Printf("  - Misses: %d\n", stats.Misses)
	fmt.Printf("  - Evictions: %d\n", stats.Evictions)
	fmt.Printf("  - Expirations: %d\n", stats.Expirations)
	fmt.Printf("  - Size: %d\n", stats.Size)
	fmt.Printf("  - Memory Usage: %d bytes (%.2f MB)\n", stats.MemoryUsage, float64(stats.MemoryUsage)/(1024*1024))

	if stats.Hits+stats.Misses > 0 {
		hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
		fmt.Printf("  - Hit Rate: %.2f%%\n", hitRate)
	}

	// Test eviction by filling the cache
	fmt.Println("\nTesting eviction by filling cache...")
	largeUser := User{
		ID:        "large",
		Name:      "Large User",
		Email:     "large@example.com",
		CreatedAt: time.Now(),
	}

	// Add large data to trigger evictions
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("large:user:%d", i)
		if err := c.Set(key, largeUser, 1*time.Minute); err != nil {
			break // Cache is full
		}
	}

	// Get updated statistics
	time.Sleep(1 * time.Second)
	updatedStats := ristrettoStore.GetStats()
	fmt.Printf("Updated Statistics - Evictions: %d\n", updatedStats.Evictions)
}
