package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
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
	fmt.Println("=== Go-CacheX Resilience & Reliability Demo ===")

	// Create Redis store with timeouts
	redisConfig := &redisstore.Config{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}

	store, err := redisstore.New(redisConfig)
	if err != nil {
		logx.Error("Failed to create Redis store", logx.ErrorField(err))
		return
	}

	// Create cache with all resilience features enabled
	c, err := cache.New[User](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
		cache.WithMaxRetries(3),
		cache.WithRetryDelay(100*time.Millisecond),
		cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
			Threshold:   5,
			Timeout:     30 * time.Second,
			HalfOpenMax: 3,
		}),
		cache.WithRateLimit(cache.RateLimitConfig{
			RequestsPerSecond: 1000,
			Burst:             100,
		}),
		cache.WithDeadLetterQueue(), // Enable dead-letter queue
		cache.WithBloomFilter(),     // Enable bloom filter
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}

	demoRetryPolicy(c)
	demoTimeouts(c)
	demoDeadLetterQueue(c)
	demoBloomFilter(c)
	demoCircuitBreaker(c)
	demoDistributedLocks(c)
	demoPubSubInvalidation(c)

	fmt.Println("\n=== Resilience Demo Complete ===")
}

func demoRetryPolicy(c cache.Cache[User]) {
	fmt.Println("1. Configurable Retry Policy Demo")
	fmt.Println("   - Max attempts: 3")
	fmt.Println("   - Initial delay: 100ms")
	fmt.Println("   - Exponential backoff with jitter")
	fmt.Println("   - Retryable errors handled automatically")
	fmt.Println()

	// Simulate a failing operation that eventually succeeds
	user := User{
		ID:        "12345",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	// This will use the retry policy internally
	err := c.Set("user:12345", user, 5*time.Minute)
	if err != nil {
		fmt.Printf("❌ Set operation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Set operation succeeded with retry policy\n")
	}

	// Retrieve with retry policy
	retrieved, found, err := c.Get("user:12345")
	if err != nil {
		fmt.Printf("❌ Get operation failed: %v\n", err)
	} else if found {
		fmt.Printf("✅ Get operation succeeded: %s\n", retrieved.Name)
	}
	fmt.Println()
}

func demoTimeouts(c cache.Cache[User]) {
	fmt.Println("2. Timeouts Demo")
	fmt.Println("   - Connection timeout: 5s")
	fmt.Println("   - Read timeout: 3s")
	fmt.Println("   - Write timeout: 3s")
	fmt.Println("   - Pool timeout: 4s")
	fmt.Println()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use context-aware cache
	ctxCache := c.WithContext(ctx)

	user := User{
		ID:        "timeout-test",
		Name:      "Timeout User",
		Email:     "timeout@example.com",
		CreatedAt: time.Now(),
	}

	err := ctxCache.Set("user:timeout", user, 1*time.Minute)
	if err != nil {
		fmt.Printf("❌ Set with timeout failed: %v\n", err)
	} else {
		fmt.Printf("✅ Set with timeout succeeded\n")
	}
	fmt.Println()
}

func demoDeadLetterQueue(c cache.Cache[User]) {
	fmt.Println("3. Dead-Letter Queue Demo")
	fmt.Println("   - Failed write-behind operations")
	fmt.Println("   - Configurable retry policy")
	fmt.Println("   - Metrics tracking")
	fmt.Println()

	// Simulate a write-behind operation that fails
	failingWriter := func(ctx context.Context) error {
		// Simulate database failure
		return fmt.Errorf("database connection failed")
	}

	user := User{
		ID:        "dlq-test",
		Name:      "DLQ User",
		Email:     "dlq@example.com",
		CreatedAt: time.Now(),
	}

	// This will succeed in cache but fail in background
	err := c.WriteBehind("user:dlq", user, 5*time.Minute, failingWriter)
	if err != nil {
		fmt.Printf("❌ Write-behind failed: %v\n", err)
	} else {
		fmt.Printf("✅ Write-behind queued (will fail in background)\n")
	}

	// Get DLQ statistics
	if dlqStats := c.GetDeadLetterQueueStats(); dlqStats != nil {
		fmt.Printf("📊 DLQ Stats - Failed: %d, Retried: %d, Succeeded: %d\n",
			dlqStats.TotalFailed, dlqStats.TotalRetried, dlqStats.TotalSucceeded)
	}
	fmt.Println()
}

func demoBloomFilter(c cache.Cache[User]) {
	fmt.Println("4. Bloom Filter Demo")
	fmt.Println("   - Reduces misses to backing store")
	fmt.Println("   - Configurable false positive rate")
	fmt.Println("   - Memory efficient")
	fmt.Println()

	// Add some users to cache
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	for _, user := range users {
		c.Set(fmt.Sprintf("user:%s", user.ID), user, 5*time.Minute)
	}

	// Try to get existing and non-existing users
	testKeys := []string{"user:1", "user:2", "user:999", "user:888"}

	for _, key := range testKeys {
		_, found, _ := c.Get(key)
		if found {
			fmt.Printf("✅ Found: %s\n", key)
		} else {
			fmt.Printf("❌ Not found: %s (bloom filter may have prevented cache miss)\n", key)
		}
	}

	// Get bloom filter statistics
	if bfStats := c.GetBloomFilterStats(); bfStats != nil {
		fmt.Printf("📊 Bloom Filter Stats - Items: %v, False Positive Rate: %v\n",
			bfStats["item_count"], bfStats["current_false_positive_rate"])
	}
	fmt.Println()
}

func demoCircuitBreaker(c cache.Cache[User]) {
	fmt.Println("5. Circuit Breaker Demo")
	fmt.Println("   - Failure threshold: 5")
	fmt.Println("   - Timeout: 30s")
	fmt.Println("   - Half-open max: 3")
	fmt.Println()

	// Get circuit breaker state
	state := c.GetCircuitBreakerState()
	fmt.Printf("🔌 Circuit Breaker State: %s\n", state)

	// Get circuit breaker statistics
	stats := c.GetCircuitBreakerStats()
	fmt.Printf("📊 Circuit Breaker Stats - Failures: %d, Successes: %d, IsHealthy: %t\n",
		stats.TotalFailures, stats.TotalSuccesses, stats.IsHealthy())

	// Force open circuit breaker for demonstration
	c.ForceOpenCircuitBreaker()
	fmt.Printf("🔌 Circuit Breaker forced open\n")

	// Try an operation (should fail fast)
	_, _, err := c.Get("circuit-breaker-test")
	if err != nil {
		fmt.Printf("❌ Operation blocked by circuit breaker: %v\n", err)
	}

	// Force close circuit breaker
	c.ForceCloseCircuitBreaker()
	fmt.Printf("🔌 Circuit Breaker forced closed\n")
	fmt.Println()
}

func demoDistributedLocks(c cache.Cache[User]) {
	fmt.Println("6. Distributed Locks Demo")
	fmt.Println("   - Redlock-style distributed locks")
	fmt.Println("   - Lock extension support")
	fmt.Println("   - Automatic cleanup")
	fmt.Println()

	// Try to acquire a lock
	unlock, acquired, err := c.TryLock("user:12345:lock", 10*time.Second)
	if err != nil {
		fmt.Printf("❌ Lock acquisition failed: %v\n", err)
		return
	}

	if acquired {
		fmt.Printf("🔒 Lock acquired successfully\n")

		// Simulate some work
		time.Sleep(1 * time.Second)

		// Release the lock
		if err := unlock(); err != nil {
			fmt.Printf("❌ Lock release failed: %v\n", err)
		} else {
			fmt.Printf("🔓 Lock released successfully\n")
		}
	} else {
		fmt.Printf("❌ Lock not acquired\n")
	}
	fmt.Println()
}

func demoPubSubInvalidation(c cache.Cache[User]) {
	fmt.Println("7. Pub/Sub Invalidation Demo")
	fmt.Println("   - Multi-service cache coherence")
	fmt.Println("   - Tag-based invalidation")
	fmt.Println()

	// Add tags to a key
	err := c.AddTags("user:12345", "users", "active", "premium")
	if err != nil {
		fmt.Printf("❌ Add tags failed: %v\n", err)
	} else {
		fmt.Printf("✅ Tags added to user:12345\n")
	}

	// Publish invalidation
	err = c.PublishInvalidation("user:12345", "user:67890")
	if err != nil {
		fmt.Printf("❌ Publish invalidation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Invalidation published\n")
	}

	// Subscribe to invalidations
	unsubscribe, err := c.SubscribeInvalidations(func(keys ...string) {
		fmt.Printf("🔄 Received invalidation for keys: %v\n", keys)
	})
	if err != nil {
		fmt.Printf("❌ Subscribe failed: %v\n", err)
	} else {
		fmt.Printf("✅ Subscribed to invalidations\n")
		defer unsubscribe()
	}

	// Invalidate by tag
	err = c.InvalidateByTag("users", "active")
	if err != nil {
		fmt.Printf("❌ Tag invalidation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Tag invalidation completed\n")
	}
	fmt.Println()
}
