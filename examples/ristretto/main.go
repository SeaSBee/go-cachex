package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	logx.Info("=== Go-CacheX Ristretto Store Demo ===")

	// Example 1: Basic Ristretto Store Usage
	exampleBasicRistrettoStore()

	// Example 2: Ristretto Store Performance
	exampleRistrettoStorePerformance()

	// Example 3: Ristretto Store Configuration
	exampleRistrettoStoreConfiguration()

	logx.Info("=== Ristretto Store Demo Complete ===")
}

func exampleBasicRistrettoStore() {
	logx.Info("1. Basic Ristretto Store Usage")
	logx.Info("==============================")

	// Create ristretto store
	ristrettoStore, err := cachex.NewRistrettoStore(cachex.DefaultRistrettoConfig())
	if err != nil {
		logx.Error("Failed to create ristretto store", logx.ErrorField(err))
		return
	}
	defer ristrettoStore.Close()

	// Create codec
	jsonCodec := cachex.NewJSONCodec()

	// Create key builder
	keyBuilder, err := cachex.NewBuilder("myapp", "dev", "secret123")
	if err != nil {
		log.Fatalf("Failed to create key builder: %v", err)
	}

	// Create cache with ristretto store
	c, err := cachex.New[User](
		cachex.WithStore(ristrettoStore),
		cachex.WithCodec(jsonCodec),
		cachex.WithKeyBuilder(keyBuilder),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Set user in cache
	userKey := keyBuilder.BuildUser("123")
	setResult := <-c.Set(ctx, userKey, user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", userKey), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User stored in ristretto cache", logx.String("key", userKey))

	// Get user from cache
	getResult := <-c.Get(ctx, userKey)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", userKey), logx.ErrorField(getResult.Error))
		return
	}
	if getResult.Found {
		logx.Info("User retrieved from ristretto cache",
			logx.String("key", userKey),
			logx.String("user_id", getResult.Value.ID),
			logx.String("user_name", getResult.Value.Name),
			logx.String("user_email", getResult.Value.Email))
	} else {
		logx.Warn("User not found in ristretto cache", logx.String("key", userKey))
	}
}

func exampleRistrettoStorePerformance() {
	logx.Info("2. Ristretto Store Performance")
	logx.Info("===============================")

	// Create ristretto store
	ristrettoStore, err := cachex.NewRistrettoStore(cachex.DefaultRistrettoConfig())
	if err != nil {
		logx.Error("Failed to create ristretto store", logx.ErrorField(err))
		return
	}
	defer ristrettoStore.Close()

	// Create cache
	c, err := cachex.New[User](
		cachex.WithStore(ristrettoStore),
		cachex.WithCodec(cachex.NewJSONCodec()),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	ctx := context.Background()

	// Performance test: Set operations
	logx.Info("Testing set performance...")
	start := time.Now()

	for i := 0; i < 1000; i++ {
		user := User{
			ID:        fmt.Sprintf("user:%d", i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now(),
		}

		key := fmt.Sprintf("user:%d", i)
		setResult := <-c.Set(ctx, key, user, 0)
		if setResult.Error != nil {
			logx.Error("Failed to set user", logx.Int("user_index", i), logx.String("key", key), logx.ErrorField(setResult.Error))
			return
		}
	}

	setDuration := time.Since(start)
	setOpsPerSec := 1000.0 / setDuration.Seconds()
	logx.Info("Set performance test completed",
		logx.String("duration", setDuration.String()),
		logx.Int("operations", 1000),
		logx.String("ops_per_sec", fmt.Sprintf("%.2f", setOpsPerSec)))

	// Performance test: Get operations
	logx.Info("Testing get performance...")
	start = time.Now()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:%d", i)
		getResult := <-c.Get(ctx, key)
		if getResult.Error != nil {
			logx.Error("Failed to get user", logx.Int("user_index", i), logx.String("key", key), logx.ErrorField(getResult.Error))
			return
		}
		if !getResult.Found {
			logx.Error("User not found", logx.Int("user_index", i), logx.String("key", key))
			return
		}
	}

	getDuration := time.Since(start)
	getOpsPerSec := 1000.0 / getDuration.Seconds()
	logx.Info("Get performance test completed",
		logx.String("duration", getDuration.String()),
		logx.Int("operations", 1000),
		logx.String("ops_per_sec", fmt.Sprintf("%.2f", getOpsPerSec)))

	// Performance test: Multi-get operations
	logx.Info("Testing multi-get performance...")
	start = time.Now()

	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("user:%d", i)
	}

	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to multi-get users", logx.ErrorField(mgetResult.Error))
		return
	}

	multiGetDuration := time.Since(start)
	multiGetOpsPerSec := 100.0 / multiGetDuration.Seconds()
	logx.Info("Multi-get performance test completed",
		logx.String("duration", multiGetDuration.String()),
		logx.Int("operations", 100),
		logx.String("ops_per_sec", fmt.Sprintf("%.2f", multiGetOpsPerSec)))

	logx.Info("Performance test completed!")
}

func exampleRistrettoStoreConfiguration() {
	logx.Info("3. Ristretto Store Configuration")
	logx.Info("=================================")

	// Create custom ristretto store configuration
	config := cachex.DefaultRistrettoConfig()
	config.NumCounters = 1000000 // 1M counters
	config.MaxItems = 1000       // Maximum 1000 items
	config.BufferItems = 64      // Buffer size
	config.EnableMetrics = true  // Enable metrics

	ristrettoStore, err := cachex.NewRistrettoStore(config)
	if err != nil {
		logx.Error("Failed to create custom ristretto store", logx.ErrorField(err))
		return
	}
	defer ristrettoStore.Close()

	// Create cache with custom configuration
	c, err := cachex.New[User](
		cachex.WithStore(ristrettoStore),
		cachex.WithCodec(cachex.NewJSONCodec()),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Store multiple users
	users := map[string]User{
		"user:1": {ID: "user:1", Name: "User 1", Email: "user1@example.com", CreatedAt: time.Now()},
		"user:2": {ID: "user:2", Name: "User 2", Email: "user2@example.com", CreatedAt: time.Now()},
		"user:3": {ID: "user:3", Name: "User 3", Email: "user3@example.com", CreatedAt: time.Now()},
		"user:4": {ID: "user:4", Name: "User 4", Email: "user4@example.com", CreatedAt: time.Now()},
		"user:5": {ID: "user:5", Name: "User 5", Email: "user5@example.com", CreatedAt: time.Now()},
	}

	ctx := context.Background()

	msetResult := <-c.MSet(ctx, users, 0)
	if msetResult.Error != nil {
		logx.Error("Failed to set users", logx.ErrorField(msetResult.Error))
		return
	}
	logx.Info("Users stored in custom ristretto cache", logx.Int("count", len(users)))

	// Retrieve all users
	keys := []string{"user:1", "user:2", "user:3", "user:4", "user:5"}
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to get users", logx.ErrorField(mgetResult.Error))
		return
	}
	logx.Info("Users retrieved from custom ristretto cache", logx.Int("count", len(mgetResult.Values)))

	// Display retrieved users
	logx.Info("Retrieved users details")
	for key, user := range mgetResult.Values {
		logx.Info("User details",
			logx.String("key", key),
			logx.String("name", user.Name),
			logx.String("email", user.Email))
	}

	// Test TTL functionality
	logx.Info("Testing TTL functionality...")
	ttlUser := User{
		ID:        "ttl:user",
		Name:      "TTL User",
		Email:     "ttl@example.com",
		CreatedAt: time.Now(),
	}

	// Set with short TTL
	ttlKey := "user:ttl"
	setResult := <-c.Set(ctx, ttlKey, ttlUser, 2*time.Second)
	if setResult.Error != nil {
		logx.Error("Failed to set TTL user", logx.String("key", ttlKey), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User set with 2-second TTL", logx.String("key", ttlKey))

	// Check TTL
	ttlResult := <-c.TTL(ctx, ttlKey)
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL", logx.String("key", ttlKey), logx.ErrorField(ttlResult.Error))
		return
	}
	logx.Info("TTL remaining", logx.String("key", ttlKey), logx.String("ttl", ttlResult.TTL.String()))

	// Wait for expiration
	logx.Info("Waiting for TTL to expire...")
	time.Sleep(3 * time.Second)

	// Check if expired
	existsResult := <-c.Exists(ctx, ttlKey)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence", logx.String("key", ttlKey), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("User existence after TTL", logx.String("key", ttlKey), logx.Bool("exists", existsResult.Found))

	logx.Info("Ristretto store configuration demo completed!")
}
