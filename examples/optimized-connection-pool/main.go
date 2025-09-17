package main

import (
	"context"
	"fmt"
	"time"

	"github.com/seasbee/go-cachex"
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
	fmt.Println("=== Go-CacheX Optimized Connection Pool Demo ===")

	// Example 1: Basic Optimized Connection Pool
	demoBasicOptimizedPool()

	// Example 2: High Performance Optimized Connection Pool
	demoHighPerformanceOptimizedPool()

	// Example 3: Production Optimized Connection Pool
	demoProductionOptimizedPool()

	fmt.Println("=== Optimized Connection Pool Demo Complete ===")
}

func demoBasicOptimizedPool() {
	fmt.Println("\n1. Basic Optimized Connection Pool")
	fmt.Println("==================================")

	// Create optimized connection pool with basic configuration
	config := cachex.DefaultOptimizedConnectionPoolConfig()
	config.Addr = "localhost:6379" // Ensure Redis is running

	optimizedPool, err := cachex.NewOptimizedConnectionPool(config)
	if err != nil {
		logx.Error("Failed to create optimized connection pool",
			logx.ErrorField(err))
		return
	}
	defer optimizedPool.Close()

	fmt.Println("✓ Optimized connection pool created successfully")
	fmt.Printf("✓ Pool size: %d\n", config.PoolSize)
	fmt.Printf("✓ Min idle connections: %d\n", config.MinIdleConns)
	fmt.Printf("✓ Connection reuse: %t\n", config.EnableConnectionReuse)
	fmt.Printf("✓ Connection warming: %t\n", config.EnableConnectionWarming)

	// Test basic operations
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	// Encode user to JSON
	userData, err := cachex.NewJSONCodec().Encode(user)
	if err != nil {
		logx.Error("Failed to encode user",
			logx.ErrorField(err))
		return
	}

	ctx := context.Background()
	key := "user:1"

	// Set user
	setResult := <-optimizedPool.Set(ctx, key, userData, 5*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ User stored via optimized pool: %s\n", key)

	// Get user
	getResult := <-optimizedPool.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Exists {
		// Decode user from JSON
		var retrievedUser User
		err := cachex.NewJSONCodec().Decode(getResult.Value, &retrievedUser)
		if err != nil {
			logx.Error("Failed to decode user",
				logx.ErrorField(err))
			return
		}
		fmt.Printf("✓ User retrieved via optimized pool: %s (%s)\n", retrievedUser.Name, retrievedUser.Email)
	} else {
		fmt.Println("✗ User not found")
	}

	// Get connection pool statistics
	stats := optimizedPool.GetStats()
	fmt.Printf("\n✓ Connection Pool Statistics:\n")
	fmt.Printf("  - Total Connections: %d\n", stats.TotalConnections)
	fmt.Printf("  - Active Connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  - Idle Connections: %d\n", stats.IdleConnections)
	fmt.Printf("  - Reused Connections: %d\n", stats.ReusedConnections)
	fmt.Printf("  - New Connections: %d\n", stats.NewConnections)
	fmt.Printf("  - Warmed Connections: %d\n", stats.WarmedConnections)
	fmt.Printf("  - Average Wait Time: %dms\n", stats.AverageWaitTime)
	fmt.Printf("  - Average Response Time: %dms\n", stats.AverageResponseTime)
}

func demoHighPerformanceOptimizedPool() {
	fmt.Println("\n2. High Performance Optimized Connection Pool")
	fmt.Println("=============================================")

	// Create optimized connection pool with high performance configuration
	config := &cachex.OptimizedConnectionPoolConfig{
		Addr:                     "localhost:6379",
		Password:                 "",
		DB:                       0,
		TLS:                      &cachex.TLSConfig{Enabled: false},
		PoolSize:                 100,
		MinIdleConns:             50,
		MaxRetries:               5,
		PoolTimeout:              30 * time.Second,
		IdleTimeout:              5 * time.Minute,
		IdleCheckFrequency:       1 * time.Minute,
		DialTimeout:              5 * time.Second,
		ReadTimeout:              1 * time.Second,
		WriteTimeout:             1 * time.Second,
		EnablePipelining:         true,
		EnableMetrics:            true,
		EnableConnectionPool:     true,
		HealthCheckInterval:      15 * time.Second,
		HealthCheckTimeout:       2 * time.Second,
		EnableConnectionReuse:    true,
		EnableConnectionWarming:  true,
		ConnectionWarmingTimeout: 30 * time.Second,
		EnableLoadBalancing:      true,
		EnableCircuitBreaker:     true,
		EnablePoolMonitoring:     true,
		MonitoringInterval:       1 * time.Minute,
	}

	optimizedPool, err := cachex.NewOptimizedConnectionPool(config)
	if err != nil {
		logx.Error("Failed to create high performance optimized connection pool",
			logx.ErrorField(err))
		return
	}
	defer optimizedPool.Close()

	fmt.Println("✓ High performance optimized connection pool created")
	fmt.Printf("✓ Pool size: %d\n", config.PoolSize)
	fmt.Printf("✓ Min idle connections: %d\n", config.MinIdleConns)
	fmt.Printf("✓ Load balancing: %t\n", config.EnableLoadBalancing)
	fmt.Printf("✓ Circuit breaker: %t\n", config.EnableCircuitBreaker)
	fmt.Printf("✓ Pool monitoring: %t\n", config.EnablePoolMonitoring)

	// Test high-performance operations
	users := []User{
		{ID: "user:2", Name: "Jane Smith", Email: "jane@example.com", CreatedAt: time.Now()},
		{ID: "user:3", Name: "Bob Wilson", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "user:4", Name: "Alice Brown", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "user:5", Name: "Charlie Davis", Email: "charlie@example.com", CreatedAt: time.Now()},
		{ID: "user:6", Name: "Diana Evans", Email: "diana@example.com", CreatedAt: time.Now()},
	}

	ctx := context.Background()
	codec := cachex.NewJSONCodec()

	// Set multiple users individually
	for _, u := range users {
		userData, err := codec.Encode(u)
		if err != nil {
			logx.Error("Failed to encode user",
				logx.String("id", u.ID),
				logx.ErrorField(err))
			continue
		}

		setResult := <-optimizedPool.Set(ctx, u.ID, userData, 10*time.Minute)
		if setResult.Error != nil {
			logx.Error("Failed to set user",
				logx.String("id", u.ID),
				logx.ErrorField(setResult.Error))
			continue
		}
	}
	fmt.Printf("✓ Stored %d users via optimized pool\n", len(users))

	// Get multiple users individually
	keys := []string{"user:2", "user:3", "user:4", "user:5", "user:6"}
	retrievedUsers := make(map[string]User)

	for _, key := range keys {
		getResult := <-optimizedPool.Get(ctx, key)
		if getResult.Error != nil {
			logx.Error("Failed to get user",
				logx.String("key", key),
				logx.ErrorField(getResult.Error))
			continue
		}

		if getResult.Exists {
			var user User
			err := codec.Decode(getResult.Value, &user)
			if err != nil {
				logx.Error("Failed to decode user",
					logx.String("key", key),
					logx.ErrorField(err))
				continue
			}
			retrievedUsers[key] = user
		}
	}

	fmt.Printf("✓ Retrieved %d users via optimized pool\n", len(retrievedUsers))
	for key, user := range retrievedUsers {
		fmt.Printf("  - %s: %s (%s)\n", key, user.Name, user.Email)
	}

	// Get high-performance statistics
	stats := optimizedPool.GetStats()
	fmt.Printf("\n✓ High Performance Statistics:\n")
	fmt.Printf("  - Load Balanced Requests: %d\n", stats.LoadBalancedRequests)
	fmt.Printf("  - Circuit Breaker Trips: %d\n", stats.CircuitBreakerTrips)
	fmt.Printf("  - Circuit Breaker Resets: %d\n", stats.CircuitBreakerResets)
	fmt.Printf("  - P95 Response Time: %dms\n", stats.P95ResponseTime)
	fmt.Printf("  - P99 Response Time: %dms\n", stats.P99ResponseTime)
}

func demoProductionOptimizedPool() {
	fmt.Println("\n3. Production Optimized Connection Pool")
	fmt.Println("======================================")

	// Create optimized connection pool with production configuration
	config := &cachex.OptimizedConnectionPoolConfig{
		Addr:                     "redis-cluster:6379",
		Password:                 "your-secure-password",
		DB:                       0,
		TLS:                      &cachex.TLSConfig{Enabled: true, InsecureSkipVerify: false},
		PoolSize:                 200,
		MinIdleConns:             100,
		MaxRetries:               3,
		PoolTimeout:              60 * time.Second,
		IdleTimeout:              10 * time.Minute,
		IdleCheckFrequency:       2 * time.Minute,
		DialTimeout:              10 * time.Second,
		ReadTimeout:              5 * time.Second,
		WriteTimeout:             5 * time.Second,
		EnablePipelining:         true,
		EnableMetrics:            true,
		EnableConnectionPool:     true,
		HealthCheckInterval:      60 * time.Second,
		HealthCheckTimeout:       10 * time.Second,
		EnableConnectionReuse:    true,
		EnableConnectionWarming:  true,
		ConnectionWarmingTimeout: 60 * time.Second,
		EnableLoadBalancing:      true,
		EnableCircuitBreaker:     true,
		EnablePoolMonitoring:     true,
		MonitoringInterval:       30 * time.Second,
	}

	optimizedPool, err := cachex.NewOptimizedConnectionPool(config)
	if err != nil {
		logx.Error("Failed to create production optimized connection pool",
			logx.ErrorField(err))
		return
	}
	defer optimizedPool.Close()

	fmt.Println("✓ Production optimized connection pool created")
	fmt.Printf("✓ Pool size: %d\n", config.PoolSize)
	fmt.Printf("✓ Min idle connections: %d\n", config.MinIdleConns)
	fmt.Printf("✓ TLS enabled: %t\n", config.TLS.Enabled)
	fmt.Printf("✓ Health check interval: %s\n", config.HealthCheckInterval)
	fmt.Printf("✓ Monitoring interval: %s\n", config.MonitoringInterval)

	// Test production operations
	user := User{
		ID:        "user:7",
		Name:      "Production User",
		Email:     "prod@example.com",
		CreatedAt: time.Now(),
	}

	// Encode user to JSON
	userData, err := cachex.NewJSONCodec().Encode(user)
	if err != nil {
		logx.Error("Failed to encode production user",
			logx.ErrorField(err))
		return
	}

	ctx := context.Background()
	key := "user:7"

	// Set production user
	setResult := <-optimizedPool.Set(ctx, key, userData, 15*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set production user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ Production user stored: %s\n", key)

	// Get production user
	getResult := <-optimizedPool.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get production user",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Exists {
		// Decode user from JSON
		var retrievedUser User
		err := cachex.NewJSONCodec().Decode(getResult.Value, &retrievedUser)
		if err != nil {
			logx.Error("Failed to decode production user",
				logx.ErrorField(err))
			return
		}
		fmt.Printf("✓ Production user retrieved: %s (%s)\n", retrievedUser.Name, retrievedUser.Email)
	} else {
		fmt.Println("✗ Production user not found")
	}

	// Get production statistics
	stats := optimizedPool.GetStats()
	fmt.Printf("\n✓ Production Statistics:\n")
	fmt.Printf("  - Total Connections: %d\n", stats.TotalConnections)
	fmt.Printf("  - Active Connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  - Idle Connections: %d\n", stats.IdleConnections)
	fmt.Printf("  - Health Checks: %d\n", stats.HealthChecks)
	fmt.Printf("  - Health Failures: %d\n", stats.HealthFailures)
	fmt.Printf("  - Timeouts: %d\n", stats.Timeouts)
	fmt.Printf("  - Errors: %d\n", stats.Errors)
	fmt.Printf("  - Average Wait Time: %dms\n", stats.AverageWaitTime)
	fmt.Printf("  - Average Response Time: %dms\n", stats.AverageResponseTime)
}
