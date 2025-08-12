package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/seasbee/go-logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Observability Demo ===")

	// Initialize OpenTelemetry tracing
	initTracing()

	// Create memory store for demo
	store, err := memorystore.New(memorystore.DefaultConfig())
	if err != nil {
		log.Fatal("Failed to create memory store:", err)
	}

	// Note: Observability is configured through cache options
	// The observability package is used internally by the cache

	// Create cache with observability enabled
	c, err := cache.New[User](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
		cache.WithObservability(cache.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		}),
		cache.WithSecurity(cache.SecurityConfig{
			RedactLogs: false, // Set to true to redact keys in logs
		}),
	)
	if err != nil {
		log.Fatal("Failed to create cache:", err)
	}

	// Start Prometheus metrics server
	go startMetricsServer()

	// Demonstrate tracing
	demoTracing(c)

	// Demonstrate metrics
	demoMetrics(c)

	// Demonstrate enhanced logging
	demoLogging(c)

	// Demonstrate error tracking
	demoErrorTracking(c)

	// Demonstrate circuit breaker metrics
	demoCircuitBreakerMetrics(c)

	fmt.Println("\n=== Observability Demo Complete ===")
	fmt.Println("• View metrics: http://localhost:8080/metrics")
	fmt.Println("• Check console for OpenTelemetry traces")
	fmt.Println("• Check logs for structured logging output")
}

func initTracing() {
	// Create stdout exporter for demo
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal("Failed to create trace exporter:", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("cachex-demo"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("development"),
		),
	)
	if err != nil {
		log.Fatal("Failed to create resource:", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("📊 Starting metrics server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Failed to start metrics server:", err)
	}
}

func demoTracing(c cache.Cache[User]) {
	fmt.Println("🔍 Demonstrating OpenTelemetry Tracing...")

	// Create a root span
	tracer := otel.Tracer("cachex-demo")
	_, span := tracer.Start(context.Background(), "demo-tracing")
	defer span.End()

	// Perform cache operations with tracing
	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	// Set operation with tracing
	if err := c.Set("user:123", user, 10*time.Minute); err != nil {
		logx.Error("Failed to set user", logx.ErrorField(err))
	}

	// Get operation with tracing
	if _, found, err := c.Get("user:123"); err != nil {
		logx.Error("Failed to get user", logx.ErrorField(err))
	} else {
		fmt.Printf("✅ User found: %t\n", found)
	}

	// MGet operation with tracing
	users, err := c.MGet("user:123", "user:456")
	if err != nil {
		logx.Error("Failed to MGet users", logx.ErrorField(err))
	} else {
		fmt.Printf("✅ Retrieved %d users\n", len(users))
	}

	fmt.Println()
}

func demoMetrics(c cache.Cache[User]) {
	fmt.Println("📊 Demonstrating Prometheus Metrics...")

	// Perform various operations to generate metrics
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	// Set operations
	for i, user := range users {
		key := fmt.Sprintf("user:%s", user.ID)
		if err := c.Set(key, user, 5*time.Minute); err != nil {
			logx.Error("Failed to set user", logx.String("key", key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Set user %d\n", i+1)
		}
	}

	// Get operations (hits)
	for i, user := range users {
		key := fmt.Sprintf("user:%s", user.ID)
		if _, found, err := c.Get(key); err != nil {
			logx.Error("Failed to get user", logx.String("key", key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Get user %d (hit: %t)\n", i+1, found)
		}
	}

	// Get operations (misses)
	for i := 4; i <= 6; i++ {
		key := fmt.Sprintf("user:%d", i)
		if _, found, err := c.Get(key); err != nil {
			logx.Error("Failed to get user", logx.String("key", key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Get user %d (hit: %t)\n", i, found)
		}
	}

	// Delete operations
	for i := 1; i <= 2; i++ {
		key := fmt.Sprintf("user:%d", i)
		if err := c.Del(key); err != nil {
			logx.Error("Failed to delete user", logx.String("key", key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Deleted user %d\n", i)
		}
	}

	fmt.Println("📈 Metrics available at http://localhost:8080/metrics")
	fmt.Println()
}

func demoLogging(c cache.Cache[User]) {
	fmt.Println("📝 Demonstrating Enhanced Logging...")

	// Test with different key patterns to show namespace extraction
	testCases := []struct {
		key  string
		user User
	}{
		{"user:123", User{ID: "123", Name: "User 123", Email: "user123@example.com", CreatedAt: time.Now()}},
		{"product:456", User{ID: "456", Name: "Product 456", Email: "product456@example.com", CreatedAt: time.Now()}},
		{"session:789", User{ID: "789", Name: "Session 789", Email: "session789@example.com", CreatedAt: time.Now()}},
	}

	for _, tc := range testCases {
		// Set operation
		if err := c.Set(tc.key, tc.user, 10*time.Minute); err != nil {
			logx.Error("Failed to set item", logx.String("key", tc.key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Set %s\n", tc.key)
		}

		// Get operation
		if _, found, err := c.Get(tc.key); err != nil {
			logx.Error("Failed to get item", logx.String("key", tc.key), logx.ErrorField(err))
		} else {
			fmt.Printf("✅ Get %s (found: %t)\n", tc.key, found)
		}
	}

	fmt.Println("📋 Check console output for structured logs with enhanced fields")
	fmt.Println()
}

func demoErrorTracking(c cache.Cache[User]) {
	fmt.Println("🚨 Demonstrating Error Tracking...")

	// Test with invalid operations to generate errors
	testCases := []struct {
		description string
		operation   func() error
	}{
		{
			"Get non-existent key",
			func() error {
				_, _, err := c.Get("non:existent:key")
				return err
			},
		},
		{
			"Delete non-existent key",
			func() error {
				return c.Del("non:existent:key")
			},
		},
		{
			"MGet with empty keys",
			func() error {
				_, err := c.MGet()
				return err
			},
		},
	}

	for _, tc := range testCases {
		if err := tc.operation(); err != nil {
			fmt.Printf("❌ %s: %v\n", tc.description, err)
		} else {
			fmt.Printf("✅ %s: No error\n", tc.description)
		}
	}

	fmt.Println("📊 Error metrics will be recorded in Prometheus")
	fmt.Println()
}

func demoCircuitBreakerMetrics(c cache.Cache[User]) {
	fmt.Println("⚡ Demonstrating Circuit Breaker Metrics...")

	// Get circuit breaker stats
	stats := c.GetCircuitBreakerStats()
	fmt.Printf("📊 Circuit Breaker State: %s\n", stats.State)
	fmt.Printf("📊 Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("📊 Total Failures: %d\n", stats.TotalFailures)
	fmt.Printf("📊 Total Successes: %d\n", stats.TotalSuccesses)
	fmt.Printf("📊 Failure Rate: %.2f%%\n", stats.FailureRate()*100)
	fmt.Printf("📊 Success Rate: %.2f%%\n", stats.SuccessRate()*100)
	fmt.Printf("📊 Is Healthy: %t\n", stats.IsHealthy())

	// Force circuit breaker to open for demonstration
	fmt.Println("🔧 Forcing circuit breaker to open...")
	c.ForceOpenCircuitBreaker()

	// Try an operation (should fail due to open circuit)
	if _, _, err := c.Get("test:key"); err != nil {
		fmt.Printf("❌ Expected error due to open circuit: %v\n", err)
	}

	// Force circuit breaker to close
	fmt.Println("🔧 Forcing circuit breaker to close...")
	c.ForceCloseCircuitBreaker()

	// Try an operation (should succeed)
	if _, _, err := c.Get("test:key"); err != nil {
		fmt.Printf("❌ Unexpected error: %v\n", err)
	} else {
		fmt.Println("✅ Operation succeeded after closing circuit")
	}

	fmt.Println("📈 Circuit breaker metrics available in Prometheus")
	fmt.Println()
}
