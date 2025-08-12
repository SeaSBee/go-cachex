package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/seasbee/go-logx"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"` // Sensitive field
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Logging Demo ===")

	// 1. Initialize logger with app/env/service fields
	initLogger()

	// 2. Demonstrate structured logging with different security levels
	demoStructuredLogging()

	// 3. Demonstrate log redaction
	demoLogRedaction()

	// 4. Demonstrate cache operations with logging
	demoCacheOperations()

	fmt.Println("\n=== Logging Demo Complete ===")
	fmt.Println("Check the console output for structured logging examples")
}

// initLogger initializes the logger with app/env/service fields
func initLogger() {
	fmt.Println("\n1. Logger Initialization")
	fmt.Println("========================")

	// Note: go-logx is used directly without explicit initialization
	// The logger is configured through environment variables or defaults

	logx.Info("Logger initialized successfully",
		logx.String("component", "cachex"),
		logx.String("app", "cachex-demo"),
		logx.String("env", "development"),
		logx.String("service", "user-service"),
		logx.String("version", "1.0.0"))

	fmt.Println("✓ Logger initialized with structured fields")
}

// demoStructuredLogging demonstrates structured logging with required fields
func demoStructuredLogging() {
	fmt.Println("\n2. Structured Logging Fields")
	fmt.Println("============================")

	// Demonstrate all required log fields
	logx.Info("Cache operation started",
		logx.String("component", "cachex"),
		logx.String("op", "get"),
		logx.String("key_ns", "users"),
		logx.String("key_type", "user"),
		logx.Int("ttl_ms", 300000), // 5 minutes
		logx.Int("attempt", 1),
		logx.Int("duration_ms", 45),
		logx.String("key", "user:123"))

	logx.Warn("Cache operation warning",
		logx.String("component", "cachex"),
		logx.String("op", "set"),
		logx.String("key_ns", "products"),
		logx.String("key_type", "product"),
		logx.Int("ttl_ms", 600000), // 10 minutes
		logx.Int("attempt", 2),
		logx.Int("duration_ms", 120),
		logx.String("key", "product:456"),
		logx.ErrorField(fmt.Errorf("temporary network issue")))

	logx.Error("Cache operation failed",
		logx.String("component", "cachex"),
		logx.String("op", "del"),
		logx.String("key_ns", "orders"),
		logx.String("key_type", "order"),
		logx.Int("ttl_ms", 0),
		logx.Int("attempt", 3),
		logx.Int("duration_ms", 500),
		logx.String("key", "order:789"),
		logx.ErrorField(fmt.Errorf("connection timeout")))

	fmt.Println("✓ Structured logging with all required fields demonstrated")
}

// demoLogRedaction demonstrates log redaction when Security.RedactLogs=true
func demoLogRedaction() {
	fmt.Println("\n3. Log Redaction (Security.RedactLogs=true)")
	fmt.Println("============================================")

	// Create cache with redaction enabled
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	if err != nil {
		log.Fatal("Failed to create memory store:", err)
	}

	// Create cache with redaction enabled
	c, err := cache.New[*User](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
		cache.WithSecurity(cache.SecurityConfig{
			RedactLogs: true, // Enable log redaction
		}),
		cache.WithObservability(cache.ObservabilityConfig{
			EnableLogging: true,
		}),
	)
	if err != nil {
		log.Fatal("Failed to create cache:", err)
	}
	defer c.Close()

	// Create a user with sensitive data
	user := &User{
		ID:        "user:123",
		Name:      "John Doe",
		Email:     "john@example.com",
		Password:  "secret-password-123", // This should be redacted
		CreatedAt: time.Now(),
	}

	// Perform operations that will trigger logging
	fmt.Println("✓ Setting user with sensitive data...")
	if err := c.Set("user:123", user, 5*time.Minute); err != nil {
		log.Printf("Failed to set user: %v", err)
	}

	fmt.Println("✓ Getting user (should show redacted key)...")
	if _, found, err := c.Get("user:123"); err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Println("  - User found in cache")
	}

	fmt.Println("✓ Deleting user...")
	if err := c.Del("user:123"); err != nil {
		log.Printf("Failed to delete user: %v", err)
	}

	fmt.Println("✓ Log redaction demonstrated - check logs for [REDACTED] entries")
}

// demoCacheOperations demonstrates cache operations with comprehensive logging
func demoCacheOperations() {
	fmt.Println("\n4. Cache Operations with Logging")
	fmt.Println("=================================")

	// Create cache with logging enabled
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	if err != nil {
		log.Fatal("Failed to create memory store:", err)
	}

	c, err := cache.New[*User](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
		cache.WithSecurity(cache.SecurityConfig{
			RedactLogs: false, // Disable redaction to see full keys
		}),
		cache.WithObservability(cache.ObservabilityConfig{
			EnableLogging: true,
		}),
	)
	if err != nil {
		log.Fatal("Failed to create cache:", err)
	}
	defer c.Close()

	// Demonstrate various cache operations with logging
	users := []*User{
		{ID: "user:1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "user:2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "user:3", Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	// Set operations
	fmt.Println("✓ Setting multiple users...")
	for _, user := range users {
		if err := c.Set(user.ID, user, 5*time.Minute); err != nil {
			log.Printf("Failed to set user %s: %v", user.ID, err)
		}
	}

	// Get operations
	fmt.Println("✓ Getting users...")
	for _, user := range users {
		if _, found, err := c.Get(user.ID); err != nil {
			log.Printf("Failed to get user %s: %v", user.ID, err)
		} else if found {
			fmt.Printf("  - Found user: %s\n", user.Name)
		}
	}

	// MGet operations
	fmt.Println("✓ Getting multiple users at once...")
	keys := []string{"user:1", "user:2", "user:3"}
	if results, err := c.MGet(keys...); err != nil {
		log.Printf("Failed to MGet users: %v", err)
	} else {
		fmt.Printf("  - Retrieved %d users\n", len(results))
	}

	// Tag-based operations
	fmt.Println("✓ Adding tags to users...")
	if err := c.AddTags("user:1", "users", "active"); err != nil {
		log.Printf("Failed to add tags: %v", err)
	}

	// Invalidate by tag
	fmt.Println("✓ Invalidating by tag...")
	if err := c.InvalidateByTag("users"); err != nil {
		log.Printf("Failed to invalidate by tag: %v", err)
	}

	// Delete operations
	fmt.Println("✓ Deleting users...")
	for _, user := range users {
		if err := c.Del(user.ID); err != nil {
			log.Printf("Failed to delete user %s: %v", user.ID, err)
		}
	}

	fmt.Println("✓ Cache operations with logging demonstrated")
}

// demoErrorLogging demonstrates error logging scenarios
func demoErrorLogging() {
	fmt.Println("\n5. Error Logging Scenarios")
	fmt.Println("===========================")

	// Simulate various error scenarios
	logx.Error("Cache connection failed",
		logx.String("component", "cachex"),
		logx.String("op", "connect"),
		logx.String("key_ns", "redis"),
		logx.String("key_type", "connection"),
		logx.Int("attempt", 3),
		logx.Int("duration_ms", 5000),
		logx.ErrorField(fmt.Errorf("connection timeout after 5 seconds")))

	logx.Warn("Cache miss on frequently accessed key",
		logx.String("component", "cachex"),
		logx.String("op", "get"),
		logx.String("key_ns", "users"),
		logx.String("key_type", "user"),
		logx.Int("ttl_ms", 300000),
		logx.Int("attempt", 1),
		logx.Int("duration_ms", 25),
		logx.String("key", "user:hot-key"))

	logx.Error("Circuit breaker opened",
		logx.String("component", "cachex"),
		logx.String("op", "circuit_breaker"),
		logx.String("key_ns", "system"),
		logx.String("key_type", "circuit_breaker"),
		logx.Int("attempt", 5),
		logx.Int("duration_ms", 1000),
		logx.ErrorField(fmt.Errorf("circuit breaker opened due to high failure rate")))

	fmt.Println("✓ Error logging scenarios demonstrated")
}

// demoPerformanceLogging demonstrates performance-related logging
func demoPerformanceLogging() {
	fmt.Println("\n6. Performance Logging")
	fmt.Println("======================")

	// Simulate performance metrics
	logx.Info("Cache performance metrics",
		logx.String("component", "cachex"),
		logx.String("op", "metrics"),
		logx.String("key_ns", "performance"),
		logx.String("key_type", "metrics"),
		logx.Int("ttl_ms", 0),
		logx.Int("attempt", 1),
		logx.Int("duration_ms", 0),
		logx.Float64("hit_rate", 0.85),
		logx.Float64("avg_latency_ms", 12.5),
		logx.Int64("total_operations", 10000),
		logx.Int64("cache_hits", 8500),
		logx.Int64("cache_misses", 1500))

	logx.Warn("High latency detected",
		logx.String("component", "cachex"),
		logx.String("op", "get"),
		logx.String("key_ns", "users"),
		logx.String("key_type", "user"),
		logx.Int("ttl_ms", 300000),
		logx.Int("attempt", 1),
		logx.Int("duration_ms", 250), // High latency
		logx.String("key", "user:complex-query"),
		logx.ErrorField(fmt.Errorf("operation took longer than expected")))

	fmt.Println("✓ Performance logging demonstrated")
}
