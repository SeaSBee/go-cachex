package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/SeaSBee/go-logx"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Redis Store Demo ===")

	// Example 1: Basic Redis Configuration
	demoBasicRedis()

	// Example 2: High Performance Redis Configuration
	demoHighPerformanceRedis()

	// Example 3: Production Redis Configuration
	demoProductionRedis()

	fmt.Println("=== Redis Store Demo Complete ===")
}

func demoBasicRedis() {
	fmt.Println("\n1. Basic Redis Configuration")
	fmt.Println("=============================")

	// Create Redis store with basic configuration
	redisStore, err := cachex.NewRedisStore(cachex.DefaultRedisConfig())
	if err != nil {
		logx.Error("Failed to create Redis store",
			logx.ErrorField(err))
		return
	}
	defer redisStore.Close()

	// Create cache with Redis store
	c, err := cachex.New[User](
		cachex.WithStore(redisStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	fmt.Println("✓ Redis store created successfully")
	fmt.Println("✓ Cache created with Redis backend")

	// Test basic operations
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	key := "user:1"
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ User stored in Redis: %s\n", key)
	logx.Info("User stored in Redis",
		logx.String("key", key))

	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved from Redis: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
		logx.Info("User retrieved from Redis",
			logx.String("key", key),
			logx.String("name", getResult.Value.Name),
			logx.String("email", getResult.Value.Email))
	} else {
		fmt.Println("✗ User not found in Redis")
		logx.Warn("User not found in Redis",
			logx.String("key", key))
	}

	// Test multiple operations
	fmt.Println("\nTesting multiple operations...")
	logx.Info("Testing multiple operations...")
	users := []User{
		{ID: "user:2", Name: "Jane Smith", Email: "jane@example.com", CreatedAt: time.Now()},
		{ID: "user:3", Name: "Bob Wilson", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "user:4", Name: "Alice Brown", Email: "alice@example.com", CreatedAt: time.Now()},
	}

	// MSet multiple users
	items := make(map[string]User)
	for _, u := range users {
		items[u.ID] = u
	}

	msetResult := <-c.MSet(ctx, items, 10*time.Minute)
	if msetResult.Error != nil {
		logx.Error("Failed to set multiple users",
			logx.Int("count", len(users)),
			logx.ErrorField(msetResult.Error))
		return
	}
	fmt.Printf("✓ Stored %d users in Redis\n", len(users))
	logx.Info("Stored users in Redis",
		logx.Int("count", len(users)))

	// MGet multiple users
	keys := []string{"user:2", "user:3", "user:4"}
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to get multiple users",
			logx.String("keys", fmt.Sprintf("%v", keys)),
			logx.ErrorField(mgetResult.Error))
		return
	}

	fmt.Printf("✓ Retrieved %d users from Redis\n", len(mgetResult.Values))
	logx.Info("Retrieved users from Redis",
		logx.Int("count", len(mgetResult.Values)))
	for key, user := range mgetResult.Values {
		fmt.Printf("  - %s: %s (%s)\n", key, user.Name, user.Email)
		logx.Debug("Retrieved user",
			logx.String("key", key),
			logx.String("name", user.Name),
			logx.String("email", user.Email))
	}

	// Get Redis statistics
	stats := redisStore.GetStats()
	fmt.Printf("\n✓ Redis Statistics:\n")
	fmt.Printf("  - Hits: %d\n", stats.Hits)
	fmt.Printf("  - Misses: %d\n", stats.Misses)
	fmt.Printf("  - Sets: %d\n", stats.Sets)
	fmt.Printf("  - Dels: %d\n", stats.Dels)
	fmt.Printf("  - Errors: %d\n", stats.Errors)
	fmt.Printf("  - Bytes In: %d\n", stats.BytesIn)
	fmt.Printf("  - Bytes Out: %d\n", stats.BytesOut)
	logx.Info("Redis Statistics",
		logx.Int64("hits", stats.Hits),
		logx.Int64("misses", stats.Misses),
		logx.Int64("sets", stats.Sets),
		logx.Int64("dels", stats.Dels),
		logx.Int64("errors", stats.Errors),
		logx.Int64("bytesIn", stats.BytesIn),
		logx.Int64("bytesOut", stats.BytesOut))
}

func demoHighPerformanceRedis() {
	fmt.Println("\n2. High Performance Redis Configuration")
	fmt.Println("======================================")

	// Create Redis store with high performance configuration
	config := cachex.HighPerformanceRedisConfig()
	config.Addr = "localhost:6379" // Ensure Redis is running

	redisStore, err := cachex.NewRedisStore(config)
	if err != nil {
		logx.Error("Failed to create high performance Redis store",
			logx.ErrorField(err))
		return
	}
	defer redisStore.Close()

	// Create cache with high performance Redis store
	c, err := cachex.New[User](
		cachex.WithStore(redisStore),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	fmt.Println("✓ High performance Redis store created")
	fmt.Printf("✓ Pool size: %d\n", config.PoolSize)
	fmt.Printf("✓ Min idle connections: %d\n", config.MinIdleConns)
	fmt.Printf("✓ Pipelining enabled: %t\n", config.EnablePipelining)
	logx.Info("High performance Redis store created",
		logx.Int("poolSize", config.PoolSize),
		logx.Int("minIdleConns", config.MinIdleConns),
		logx.Bool("enablePipelining", config.EnablePipelining))

	// Test high-performance operations
	fmt.Println("\nTesting high-performance operations...")
	logx.Info("Testing high-performance operations...")

	// Create many users for batch operations
	users := make(map[string]User)
	for i := 1; i <= 100; i++ {
		users[fmt.Sprintf("user:%d", i)] = User{
			ID:        fmt.Sprintf("user:%d", i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now(),
		}
	}

	ctx := context.Background()

	// Batch set operations
	start := time.Now()
	msetResult := <-c.MSet(ctx, users, 5*time.Minute)
	if msetResult.Error != nil {
		logx.Error("Failed to batch set users",
			logx.Int("count", len(users)),
			logx.ErrorField(msetResult.Error))
		return
	}
	setDuration := time.Since(start)
	fmt.Printf("✓ Batch set 100 users in %v\n", setDuration)
	logx.Info("Batch set users completed",
		logx.Int("count", len(users)),
		logx.String("duration", setDuration.String()))

	// Batch get operations
	keys := make([]string, 0, len(users))
	for key := range users {
		keys = append(keys, key)
	}

	start = time.Now()
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to batch get users",
			logx.Int("count", len(keys)),
			logx.ErrorField(mgetResult.Error))
		return
	}
	getDuration := time.Since(start)
	fmt.Printf("✓ Batch retrieved %d users in %v\n", len(mgetResult.Values), getDuration)
	logx.Info("Batch retrieved users",
		logx.Int("count", len(mgetResult.Values)),
		logx.String("duration", getDuration.String()))

	// Performance statistics
	stats := redisStore.GetStats()
	fmt.Printf("\n✓ Performance Statistics:\n")
	fmt.Printf("  - Total operations: %d\n", stats.Hits+stats.Misses+stats.Sets)
	fmt.Printf("  - Average set time: %v\n", setDuration/time.Duration(len(users)))
	fmt.Printf("  - Average get time: %v\n", getDuration/time.Duration(len(mgetResult.Values)))
	fmt.Printf("  - Total bytes transferred: %d\n", stats.BytesIn+stats.BytesOut)
	logx.Info("Performance Statistics",
		logx.Int64("totalOperations", stats.Hits+stats.Misses+stats.Sets),
		logx.String("averageSetTime", (setDuration/time.Duration(len(users))).String()),
		logx.String("averageGetTime", (getDuration/time.Duration(len(mgetResult.Values))).String()),
		logx.Int64("totalBytesTransferred", stats.BytesIn+stats.BytesOut))
}

func demoProductionRedis() {
	fmt.Println("\n3. Production Redis Configuration")
	fmt.Println("=================================")

	// Create Redis store with production configuration
	config := cachex.ProductionRedisConfig()
	config.Addr = "localhost:6379" // Ensure Redis is running
	config.TLS.Enabled = false     // Disable TLS for local development

	redisStore, err := cachex.NewRedisStore(config)
	if err != nil {
		logx.Error("Failed to create production Redis store",
			logx.ErrorField(err))
		return
	}
	defer redisStore.Close()

	// Create cache with production Redis store
	c, err := cachex.New[User](
		cachex.WithStore(redisStore),
		cachex.WithDefaultTTL(15*time.Minute),
		cachex.WithMaxRetries(5),
		cachex.WithRetryDelay(100*time.Millisecond),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	fmt.Println("✓ Production Redis store created")
	fmt.Printf("✓ Pool size: %d\n", config.PoolSize)
	fmt.Printf("✓ Min idle connections: %d\n", config.MinIdleConns)
	fmt.Printf("✓ Max retries: %d\n", config.MaxRetries)
	fmt.Printf("✓ TLS enabled: %t\n", config.TLS.Enabled)
	logx.Info("Production Redis store created",
		logx.Int("poolSize", config.PoolSize),
		logx.Int("minIdleConns", config.MinIdleConns),
		logx.Int("maxRetries", config.MaxRetries),
		logx.Bool("tlsEnabled", config.TLS.Enabled))

	// Test production features
	fmt.Println("\nTesting production features...")
	logx.Info("Testing production features...")

	// Test with context
	ctx := context.Background()
	cWithCtx := c.WithContext(ctx)

	user := User{
		ID:        "prod:user:1",
		Name:      "Production User",
		Email:     "prod@example.com",
		CreatedAt: time.Now(),
	}

	key := "prod:user:1"
	setResult := <-cWithCtx.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user with context",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ User stored with context: %s\n", key)
	logx.Info("User stored with context",
		logx.String("key", key))

	// Test TTL
	ttlResult := <-c.TTL(ctx, key)
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL",
			logx.String("key", key),
			logx.ErrorField(ttlResult.Error))
		return
	}
	fmt.Printf("✓ TTL for key %s: %v\n", key, ttlResult.TTL)
	logx.Info("TTL retrieved",
		logx.String("key", key),
		logx.String("ttl", ttlResult.TTL.String()))

	// Test exists
	existsResult := <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence",
			logx.String("key", key),
			logx.ErrorField(existsResult.Error))
		return
	}
	fmt.Printf("✓ Key %s exists: %t\n", key, existsResult.Found)
	logx.Info("Key existence checked",
		logx.String("key", key),
		logx.Bool("exists", existsResult.Found))

	// Test increment
	incrResult := <-c.IncrBy(ctx, "counter:test", 1, 1*time.Hour)
	if incrResult.Error != nil {
		logx.Error("Failed to increment counter",
			logx.String("key", "counter:test"),
			logx.ErrorField(incrResult.Error))
		return
	}
	fmt.Printf("✓ Counter incremented to: %d\n", incrResult.Int)
	logx.Info("Counter incremented",
		logx.String("key", "counter:test"),
		logx.Int64("value", incrResult.Int))

	// Test delete
	delResult := <-c.Del(ctx, key)
	if delResult.Error != nil {
		logx.Error("Failed to delete key",
			logx.String("key", key),
			logx.ErrorField(delResult.Error))
		return
	}
	fmt.Printf("✓ Key %s deleted\n", key)
	logx.Info("Key deleted",
		logx.String("key", key))

	// Verify deletion
	existsResult = <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence after deletion",
			logx.String("key", key),
			logx.ErrorField(existsResult.Error))
		return
	}
	fmt.Printf("✓ Key %s exists after deletion: %t\n", key, existsResult.Found)
	logx.Info("Key existence checked after deletion",
		logx.String("key", key),
		logx.Bool("exists", existsResult.Found))

	// Final statistics
	stats := redisStore.GetStats()
	fmt.Printf("\n✓ Final Statistics:\n")
	fmt.Printf("  - Total hits: %d\n", stats.Hits)
	fmt.Printf("  - Total misses: %d\n", stats.Misses)
	fmt.Printf("  - Total sets: %d\n", stats.Sets)
	fmt.Printf("  - Total dels: %d\n", stats.Dels)
	fmt.Printf("  - Total errors: %d\n", stats.Errors)
	fmt.Printf("  - Total bytes in: %d\n", stats.BytesIn)
	fmt.Printf("  - Total bytes out: %d\n", stats.BytesOut)
	logx.Info("Final Statistics",
		logx.Int64("totalHits", stats.Hits),
		logx.Int64("totalMisses", stats.Misses),
		logx.Int64("totalSets", stats.Sets),
		logx.Int64("totalDels", stats.Dels),
		logx.Int64("totalErrors", stats.Errors),
		logx.Int64("totalBytesIn", stats.BytesIn),
		logx.Int64("totalBytesOut", stats.BytesOut))
}
