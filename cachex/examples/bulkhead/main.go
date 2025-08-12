package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
	// Example 1: Using default bulkhead configuration
	fmt.Println("=== Example 1: Default Bulkhead Configuration ===")
	exampleDefaultBulkhead()

	// Example 2: Using high-performance bulkhead configuration
	fmt.Println("\n=== Example 2: High-Performance Bulkhead Configuration ===")
	exampleHighPerformanceBulkhead()

	// Example 3: Using resource-constrained bulkhead configuration
	fmt.Println("\n=== Example 3: Resource-Constrained Bulkhead Configuration ===")
	exampleResourceConstrainedBulkhead()

	// Example 4: Custom bulkhead configuration
	fmt.Println("\n=== Example 4: Custom Bulkhead Configuration ===")
	exampleCustomBulkhead()

	// Example 5: Bulkhead with Redis cluster
	fmt.Println("\n=== Example 5: Bulkhead with Redis Cluster ===")
	exampleBulkheadCluster()
}

func exampleDefaultBulkhead() {
	// Create cache with default bulkhead configuration
	c, err := cache.New[string](
		cache.WithBulkheadStore(cache.DefaultBulkheadConfig()),
		cache.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Perform operations - reads and writes use separate pools

	// Write operations use the write pool
	err = c.Set("user:123", "John Doe", 5*time.Minute)
	if err != nil {
		log.Printf("Set failed: %v", err)
		return
	}
	fmt.Println("✓ Set operation completed using write pool")

	// Read operations use the read pool
	value, found, err := c.Get("user:123")
	if err != nil {
		log.Printf("Get failed: %v", err)
		return
	}
	if found {
		fmt.Printf("✓ Get operation completed using read pool: %s\n", value)
	}

	// Batch operations also use appropriate pools
	items := map[string]string{
		"user:456": "Jane Smith",
		"user:789": "Bob Johnson",
	}
	err = c.MSet(items, 5*time.Minute)
	if err != nil {
		log.Printf("MSet failed: %v", err)
		return
	}
	fmt.Println("✓ MSet operation completed using write pool")

	values, err := c.MGet("user:456", "user:789")
	if err != nil {
		log.Printf("MGet failed: %v", err)
		return
	}
	fmt.Printf("✓ MGet operation completed using read pool: %v\n", values)
}

func exampleHighPerformanceBulkhead() {
	// Create cache with high-performance bulkhead configuration
	c, err := cache.New[string](
		cache.WithBulkheadStore(cache.HighPerformanceBulkheadConfig()),
		cache.WithDefaultTTL(15*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// High-performance configuration is optimized for:
	// - Read pool: 50 connections, 25 idle (for high read throughput)
	// - Write pool: 25 connections, 12 idle (for write operations)

	fmt.Println("✓ High-performance bulkhead cache created")
	fmt.Println("  - Read pool: 50 connections (25 idle)")
	fmt.Println("  - Write pool: 25 connections (12 idle)")

	// Perform high-throughput operations
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("high-perf:%d", i)
		value := fmt.Sprintf("value-%d", i)

		// Write operation
		err := c.Set(key, value, 5*time.Minute)
		if err != nil {
			log.Printf("Set failed: %v", err)
			continue
		}

		// Read operation
		retrieved, found, err := c.Get(key)
		if err != nil {
			log.Printf("Get failed: %v", err)
			continue
		}
		if found {
			fmt.Printf("✓ High-perf operation %d: %s\n", i, retrieved)
		}
	}
}

func exampleResourceConstrainedBulkhead() {
	// Create cache with resource-constrained bulkhead configuration
	c, err := cache.New[string](
		cache.WithBulkheadStore(cache.ResourceConstrainedBulkheadConfig()),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Resource-constrained configuration is optimized for:
	// - Read pool: 10 connections, 5 idle (minimal resources)
	// - Write pool: 5 connections, 2 idle (minimal resources)

	fmt.Println("✓ Resource-constrained bulkhead cache created")
	fmt.Println("  - Read pool: 10 connections (5 idle)")
	fmt.Println("  - Write pool: 5 connections (2 idle)")

	// Perform operations with minimal resource usage
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("constrained:%d", i)
		value := fmt.Sprintf("value-%d", i)

		// Write operation
		err := c.Set(key, value, 2*time.Minute)
		if err != nil {
			log.Printf("Set failed: %v", err)
			continue
		}

		// Read operation
		retrieved, found, err := c.Get(key)
		if err != nil {
			log.Printf("Get failed: %v", err)
			continue
		}
		if found {
			fmt.Printf("✓ Constrained operation %d: %s\n", i, retrieved)
		}
	}
}

func exampleCustomBulkhead() {
	// Create custom bulkhead configuration
	config := redisstore.BulkheadConfig{
		// Read pool - optimized for read-heavy workloads
		ReadPoolSize:     40,
		ReadMinIdleConns: 20,
		ReadMaxRetries:   4,
		ReadDialTimeout:  8 * time.Second,
		ReadTimeout:      4 * time.Second,
		ReadPoolTimeout:  6 * time.Second,

		// Write pool - optimized for write operations
		WritePoolSize:     20,
		WriteMinIdleConns: 10,
		WriteMaxRetries:   4,
		WriteDialTimeout:  8 * time.Second,
		WriteTimeout:      4 * time.Second,
		WritePoolTimeout:  6 * time.Second,

		// Common settings
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	c, err := cache.New[string](
		cache.WithBulkheadStore(config),
		cache.WithDefaultTTL(20*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	fmt.Println("✓ Custom bulkhead cache created")
	fmt.Printf("  - Read pool: %d connections (%d idle)\n", config.ReadPoolSize, config.ReadMinIdleConns)
	fmt.Printf("  - Write pool: %d connections (%d idle)\n", config.WritePoolSize, config.WriteMinIdleConns)

	// Demonstrate custom configuration benefits
	// This configuration is optimized for read-heavy workloads with moderate writes

	// Perform read-heavy operations
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("custom:read:%d", i)
		value := fmt.Sprintf("read-value-%d", i)

		// Write operation (uses write pool)
		err := c.Set(key, value, 10*time.Minute)
		if err != nil {
			log.Printf("Set failed: %v", err)
			continue
		}

		// Multiple read operations (use read pool)
		for j := 0; j < 3; j++ {
			retrieved, found, err := c.Get(key)
			if err != nil {
				log.Printf("Get failed: %v", err)
				continue
			}
			if found {
				fmt.Printf("✓ Custom read operation %d-%d: %s\n", i, j, retrieved)
			}
		}
	}
}

func exampleBulkheadCluster() {
	// Example with Redis cluster (commented out since we don't have a cluster running)
	fmt.Println("Note: This example shows how to use bulkhead with Redis cluster")
	fmt.Println("      Uncomment the code below when you have a Redis cluster available")

	/*
		// Create bulkhead configuration for cluster
		config := redisstore.BulkheadConfig{
			// Read pool for cluster
			ReadPoolSize:     30,
			ReadMinIdleConns: 15,
			ReadMaxRetries:   3,
			ReadDialTimeout:  5 * time.Second,
			ReadTimeout:      3 * time.Second,
			ReadPoolTimeout:  4 * time.Second,

			// Write pool for cluster
			WritePoolSize:     15,
			WriteMinIdleConns: 8,
			WriteMaxRetries:   3,
			WriteDialTimeout:  5 * time.Second,
			WriteTimeout:      3 * time.Second,
			WritePoolTimeout:  4 * time.Second,
		}

		// Redis cluster addresses
		addrs := []string{
			"redis-cluster-1:6379",
			"redis-cluster-2:6379",
			"redis-cluster-3:6379",
		}

		c, err := cache.New[string](
			cache.WithBulkheadClusterStore(addrs, "password", config),
			cache.WithDefaultTTL(30*time.Minute),
		)
		if err != nil {
			log.Printf("Failed to create cluster cache: %v", err)
			return
		}
		defer c.Close()

		ctx := context.Background()

		fmt.Println("✓ Bulkhead cluster cache created")
		fmt.Printf("  - Cluster nodes: %v\n", addrs)
		fmt.Printf("  - Read pool: %d connections\n", config.ReadPoolSize)
		fmt.Printf("  - Write pool: %d connections\n", config.WritePoolSize)

		// Perform operations on cluster
		err = c.Set(ctx, "cluster:key", "cluster-value", 10*time.Minute)
		if err != nil {
			log.Printf("Cluster Set failed: %v", err)
			return
		}

		value, found, err := c.Get(ctx, "cluster:key")
		if err != nil {
			log.Printf("Cluster Get failed: %v", err)
			return
		}
		if found {
			fmt.Printf("✓ Cluster operation completed: %s\n", value)
		}
	*/
}
