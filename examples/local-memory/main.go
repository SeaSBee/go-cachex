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
	fmt.Println("=== Go-CacheX Local Memory Cache Demo ===")

	// Example 1: Basic memory cache
	fmt.Println("\n=== Example 1: Basic Memory Cache ===")

	// Create memory store with default configuration
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store with default configuration",
			logx.String("config_type", "default"),
			logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache
	cache, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache with default memory store",
			logx.String("store_type", "memory"),
			logx.ErrorField(err))
		return
	}
	defer cache.Close()

	// Create and store a user
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	// Store user in cache
	key := "user:1"
	ctx := context.Background()
	setResult := <-cache.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user in basic memory cache",
			logx.String("key", key),
			logx.String("user_id", user.ID),
			logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✅ User stored in cache with key: %s\n", key)
	logx.Info("Successfully stored user in basic memory cache",
		logx.String("key", key),
		logx.String("user_id", user.ID),
		logx.String("user_name", user.Name))

	// Retrieve user from cache
	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user from basic memory cache",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		fmt.Printf("✅ User retrieved from cache: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
		logx.Info("Successfully retrieved user from basic memory cache",
			logx.String("key", key),
			logx.String("user_id", getResult.Value.ID),
			logx.String("user_name", getResult.Value.Name),
			logx.Bool("found", getResult.Found))
	} else {
		fmt.Println("❌ User not found in cache")
		logx.Warn("User not found in basic memory cache",
			logx.String("key", key))
	}

	// Check if user exists
	existsResult := <-cache.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check user existence in basic memory cache",
			logx.String("key", key),
			logx.ErrorField(existsResult.Error))
		return
	}
	fmt.Printf("✅ User exists in cache: %t\n", existsResult.Found)

	// Get TTL
	ttlResult := <-cache.TTL(ctx, key)
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL for user in basic memory cache",
			logx.String("key", key),
			logx.ErrorField(ttlResult.Error))
		return
	}
	fmt.Printf("✅ User TTL: %v\n", ttlResult.TTL)

	// Delete user
	delResult := <-cache.Del(ctx, key)
	if delResult.Error != nil {
		logx.Error("Failed to delete user from basic memory cache",
			logx.String("key", key),
			logx.ErrorField(delResult.Error))
		return
	}
	fmt.Printf("✅ User deleted from cache\n")
	logx.Info("Successfully deleted user from basic memory cache",
		logx.String("key", key))

	// Verify deletion
	existsResult = <-cache.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check user existence after deletion",
			logx.String("key", key),
			logx.ErrorField(existsResult.Error))
		return
	}
	fmt.Printf("✅ User exists after deletion: %t\n", existsResult.Found)

	// Example 2: Custom memory configuration
	fmt.Println("\n=== Example 2: Custom Memory Configuration ===")

	// Create memory store with custom configuration
	customConfig := &cachex.MemoryConfig{
		MaxSize:         1000,
		MaxMemoryMB:     50,
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 2 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLFU,
		EnableStats:     true,
	}

	customMemoryStore, err := cachex.NewMemoryStore(customConfig)
	if err != nil {
		logx.Error("Failed to create memory store with custom configuration",
			logx.String("config_type", "custom"),
			logx.Int("max_size", customConfig.MaxSize),
			logx.Int("max_memory_mb", customConfig.MaxMemoryMB),
			logx.ErrorField(err))
		return
	}
	defer customMemoryStore.Close()

	// Create cache with custom configuration
	customCache, err := cachex.New[User](
		cachex.WithStore(customMemoryStore),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache with custom memory store",
			logx.String("store_type", "memory_custom"),
			logx.ErrorField(err))
		return
	}
	defer customCache.Close()

	// Store multiple products
	products := map[string]User{
		"user:1": {ID: "user:1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		"user:2": {ID: "user:2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		"user:3": {ID: "user:3", Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	msetResult := <-customCache.MSet(ctx, products, 0)
	if msetResult.Error != nil {
		logx.Error("Failed to set multiple users in custom memory cache",
			logx.Int("user_count", len(products)),
			logx.ErrorField(msetResult.Error))
		return
	}
	fmt.Printf("✅ %d users stored in custom cache\n", len(products))
	logx.Info("Successfully stored multiple users in custom memory cache",
		logx.Int("user_count", len(products)))

	// Retrieve multiple users
	userKeys := []string{"user:1", "user:2", "user:3"}
	mgetResult := <-customCache.MGet(ctx, userKeys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to get multiple users from custom memory cache",
			logx.String("keys", fmt.Sprintf("%v", userKeys)),
			logx.ErrorField(mgetResult.Error))
		return
	}
	fmt.Printf("✅ Retrieved %d users from custom cache\n", len(mgetResult.Values))
	logx.Info("Successfully retrieved multiple users from custom memory cache",
		logx.Int("users_retrieved", len(mgetResult.Values)),
		logx.String("user_keys", fmt.Sprintf("%v", userKeys)))

	// Display retrieved data
	fmt.Println("\nRetrieved users:")
	for key, user := range mgetResult.Values {
		fmt.Printf("  %s: %s (%s)\n", key, user.Name, user.Email)
	}

	// Example 3: Layered caching (L1: Memory, L2: Memory)
	fmt.Println("\n=== Example 3: Layered Caching ===")

	// Create level 1 store (fast, small)
	l1Store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         100,
		MaxMemoryMB:     10,
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 30 * time.Second,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		logx.Error("Failed to create level 1 memory store",
			logx.String("store_type", "memory_l1"),
			logx.ErrorField(err))
		return
	}
	defer l1Store.Close()

	// Create level 2 store (slower, larger)
	l2Store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         1000,
		MaxMemoryMB:     100,
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 2 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLFU,
		EnableStats:     true,
	})
	if err != nil {
		logx.Error("Failed to create level 2 memory store",
			logx.String("store_type", "memory_l2"),
			logx.ErrorField(err))
		return
	}
	defer l2Store.Close()

	// Create layered cache
	l1Cache, err := cachex.New[User](
		cachex.WithStore(l1Store),
		cachex.WithDefaultTTL(1*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create level 1 cache",
			logx.String("cache_type", "User"),
			logx.String("level", "l1"),
			logx.ErrorField(err))
		return
	}
	defer l1Cache.Close()

	l2Cache, err := cachex.New[User](
		cachex.WithStore(l2Store),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create level 2 cache",
			logx.String("cache_type", "User"),
			logx.String("level", "l2"),
			logx.ErrorField(err))
		return
	}
	defer l2Cache.Close()

	// Simulate layered caching
	layeredUser := User{
		ID:        "user:layered",
		Name:      "Layered User",
		Email:     "layered@example.com",
		CreatedAt: time.Now(),
	}

	// Store in L1
	l1SetResult := <-l1Cache.Set(ctx, "user:layered", layeredUser, 0)
	if l1SetResult.Error != nil {
		logx.Error("Failed to set user in level 1 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l1"),
			logx.ErrorField(l1SetResult.Error))
		return
	}
	fmt.Printf("✅ User stored in L1 cache\n")
	logx.Info("Successfully stored user in level 1 cache",
		logx.String("key", "user:layered"),
		logx.String("level", "l1"))

	// Store in L2
	l2SetResult := <-l2Cache.Set(ctx, "user:layered", layeredUser, 0)
	if l2SetResult.Error != nil {
		logx.Error("Failed to set user in level 2 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l2"),
			logx.ErrorField(l2SetResult.Error))
		return
	}
	fmt.Printf("✅ User stored in L2 cache\n")
	logx.Info("Successfully stored user in level 2 cache",
		logx.String("key", "user:layered"),
		logx.String("level", "l2"))

	// Retrieve from L1 (should be fast)
	start := time.Now()
	l1GetResult := <-l1Cache.Get(ctx, "user:layered")
	l1Duration := time.Since(start)
	if l1GetResult.Error != nil {
		logx.Error("Failed to get user from level 1 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l1"),
			logx.ErrorField(l1GetResult.Error))
		return
	}
	if l1GetResult.Found {
		fmt.Printf("✅ Retrieved from L1 in %v: %s\n", l1Duration, l1GetResult.Value.Name)
		logx.Info("Successfully retrieved user from level 1 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l1"),
			logx.String("duration", l1Duration.String()),
			logx.String("user_name", l1GetResult.Value.Name))
	}

	// Retrieve from L2 (should be slower)
	start = time.Now()
	l2GetResult := <-l2Cache.Get(ctx, "user:layered")
	l2Duration := time.Since(start)
	if l2GetResult.Error != nil {
		logx.Error("Failed to get user from level 2 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l2"),
			logx.ErrorField(l2GetResult.Error))
		return
	}
	if l2GetResult.Found {
		fmt.Printf("✅ Retrieved from L2 in %v: %s\n", l2Duration, l2GetResult.Value.Name)
		logx.Info("Successfully retrieved user from level 2 cache",
			logx.String("key", "user:layered"),
			logx.String("level", "l2"),
			logx.String("duration", l2Duration.String()),
			logx.String("user_name", l2GetResult.Value.Name))
	}

	// Example 4: High-performance memory cache
	fmt.Println("\n=== Example 4: High-Performance Memory Cache ===")

	// Create high-performance memory store
	hpConfig := cachex.HighPerformanceMemoryConfig()
	hpMemoryStore, err := cachex.NewMemoryStore(hpConfig)
	if err != nil {
		logx.Error("Failed to create high-performance memory store",
			logx.String("config_type", "high_performance"),
			logx.ErrorField(err))
		return
	}
	defer hpMemoryStore.Close()

	// Create high-performance cache
	hpCache, err := cachex.New[User](
		cachex.WithStore(hpMemoryStore),
		cachex.WithDefaultTTL(30*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create high-performance cache",
			logx.String("cache_type", "User"),
			logx.String("performance_level", "high"),
			logx.ErrorField(err))
		return
	}
	defer hpCache.Close()

	// Simulate high-performance operations
	fmt.Println("\nSimulating high-performance operations...")

	// Store many users quickly
	for i := 0; i < 100; i++ {
		user := User{
			ID:        fmt.Sprintf("user:%d", i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now(),
		}

		key := fmt.Sprintf("user:%d", i)
		setResult := <-hpCache.Set(ctx, key, user, 0)
		if setResult.Error != nil {
			logx.Error("Failed to set user in high-performance cache",
				logx.String("key", key),
				logx.Int("user_index", i),
				logx.ErrorField(setResult.Error))
		} else {
			fmt.Printf("  ✅ Set user %d\n", i)
		}
	}

	// Retrieve users quickly
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("user:%d", i)
		getResult := <-hpCache.Get(ctx, key)
		if getResult.Error != nil {
			logx.Error("Failed to get user from high-performance cache",
				logx.String("key", key),
				logx.Int("user_index", i),
				logx.ErrorField(getResult.Error))
		} else if getResult.Found {
			fmt.Printf("  ✅ Get user %d\n", i)
		} else {
			fmt.Printf("  ❌ User %d not found\n", i)
		}
	}

	// Multi-get operation
	multiKeys := make([]string, 50)
	for i := 0; i < 50; i++ {
		multiKeys[i] = fmt.Sprintf("user:%d", i)
	}

	mgetResult = <-hpCache.MGet(ctx, multiKeys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to multi-get users from high-performance cache",
			logx.Int("key_count", len(multiKeys)),
			logx.ErrorField(mgetResult.Error))
	} else {
		fmt.Printf("✅ Multi-get retrieved %d users\n", len(mgetResult.Values))
		logx.Info("Successfully performed multi-get operation on high-performance cache",
			logx.Int("keys_requested", len(multiKeys)),
			logx.Int("users_retrieved", len(mgetResult.Values)))
	}

	// Check existence for many keys
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("user:%d", i)
		existsResult := <-hpCache.Exists(ctx, key)
		if existsResult.Error != nil {
			logx.Error("Failed to check existence in high-performance cache",
				logx.String("key", key),
				logx.Int("user_index", i),
				logx.ErrorField(existsResult.Error))
		} else if existsResult.Found {
			fmt.Printf("  ✅ Exists user %d\n", i)
		} else {
			fmt.Printf("  ❌ User %d doesn't exist\n", i)
		}
	}

	fmt.Println("\n=== Local Memory Cache Demo Complete ===")
	logx.Info("Local memory cache demo completed successfully",
		logx.String("demo_type", "local_memory"),
		logx.Int("examples_completed", 4))
}
