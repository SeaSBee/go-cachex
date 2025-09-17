package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/seasbee/go-cachex"
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
	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Fatal("Failed to create memory store for observability demo",
			logx.String("store_type", "memory"),
			logx.String("demo_type", "observability"),
			logx.ErrorField(err))
	}

	// Note: Observability is configured through cache options
	// The observability package is used internally by the cache

	// Create cache with observability enabled
	c, err := cachex.New[User](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithObservability(cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		}),
	)
	if err != nil {
		logx.Fatal("Failed to create cache with observability",
			logx.String("cache_type", "User"),
			logx.String("demo_type", "observability"),
			logx.ErrorField(err))
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

	fmt.Println("\n=== Observability Demo Complete ===")
	fmt.Println("• View metrics: http://localhost:8080/metrics")
	fmt.Println("• Check console for OpenTelemetry traces")
	fmt.Println("• Check logs for structured logging output")

	logx.Info("Observability demo completed successfully",
		logx.String("demo_type", "observability"),
		logx.String("metrics_url", "http://localhost:8080/metrics"),
		logx.Int("examples_completed", 4))
}

func initTracing() {
	// Create stdout exporter for demo
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		logx.Fatal("Failed to create trace exporter for observability demo",
			logx.String("exporter_type", "stdout"),
			logx.ErrorField(err))
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
		logx.Fatal("Failed to create resource for observability demo",
			logx.String("service_name", "cachex-demo"),
			logx.String("service_version", "1.0.0"),
			logx.ErrorField(err))
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	logx.Info("OpenTelemetry tracing initialized successfully",
		logx.String("service_name", "cachex-demo"),
		logx.String("service_version", "1.0.0"),
		logx.String("environment", "development"))
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("📊 Starting metrics server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logx.Fatal("Failed to start metrics server for observability demo",
			logx.String("port", "8080"),
			logx.ErrorField(err))
	}
}

func demoTracing(c cachex.Cache[User]) {
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

	ctx := context.Background()

	// Set operation with tracing
	setResult := <-c.Set(ctx, "user:123", user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user with tracing",
			logx.String("key", "user:123"),
			logx.String("user_id", user.ID),
			logx.ErrorField(setResult.Error))
	} else {
		logx.Info("Successfully set user with tracing",
			logx.String("key", "user:123"),
			logx.String("user_id", user.ID),
			logx.String("user_name", user.Name))
	}

	// Get operation with tracing
	getResult := <-c.Get(ctx, "user:123")
	if getResult.Error != nil {
		logx.Error("Failed to get user with tracing",
			logx.String("key", "user:123"),
			logx.ErrorField(getResult.Error))
	} else {
		fmt.Printf("✅ User found: %t\n", getResult.Found)
		logx.Info("Successfully retrieved user with tracing",
			logx.String("key", "user:123"),
			logx.Bool("found", getResult.Found))
	}

	// MGet operation with tracing
	mgetResult := <-c.MGet(ctx, "user:123", "user:456")
	if mgetResult.Error != nil {
		logx.Error("Failed to MGet users with tracing",
			logx.String("keys", fmt.Sprintf("%v", []string{"user:123", "user:456"})),
			logx.ErrorField(mgetResult.Error))
	} else {
		fmt.Printf("✅ Retrieved %d users\n", len(mgetResult.Values))
		logx.Info("Successfully retrieved multiple users with tracing",
			logx.Int("users_retrieved", len(mgetResult.Values)),
			logx.String("keys", fmt.Sprintf("%v", []string{"user:123", "user:456"})))
	}

	fmt.Println()
}

func demoMetrics(c cachex.Cache[User]) {
	fmt.Println("📊 Demonstrating Prometheus Metrics...")

	// Perform operations to generate metrics
	user := User{
		ID:        "456",
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Set operation
	setResult := <-c.Set(ctx, "user:456", user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user for metrics demo",
			logx.String("key", "user:456"),
			logx.String("user_id", user.ID),
			logx.ErrorField(setResult.Error))
	} else {
		fmt.Printf("✅ Set user: %s\n", user.Name)
		logx.Info("Successfully set user for metrics demo",
			logx.String("key", "user:456"),
			logx.String("user_id", user.ID),
			logx.String("user_name", user.Name))
	}

	// Get operation (cache hit)
	getResult := <-c.Get(ctx, "user:456")
	if getResult.Error != nil {
		logx.Error("Failed to get user for metrics demo",
			logx.String("key", "user:456"),
			logx.ErrorField(getResult.Error))
	} else {
		fmt.Printf("✅ Get user (found: %t)\n", getResult.Found)
		logx.Info("Successfully retrieved user for metrics demo",
			logx.String("key", "user:456"),
			logx.Bool("found", getResult.Found))
	}

	// Get non-existent user (cache miss)
	getResult = <-c.Get(ctx, "user:999")
	if getResult.Error != nil {
		logx.Error("Failed to get non-existent user for metrics demo",
			logx.String("key", "user:999"),
			logx.ErrorField(getResult.Error))
	} else {
		fmt.Printf("✅ Get non-existent user (found: %t)\n", getResult.Found)
		logx.Info("Successfully handled cache miss for metrics demo",
			logx.String("key", "user:999"),
			logx.Bool("found", getResult.Found))
	}

	// Delete operation
	delResult := <-c.Del(ctx, "user:456")
	if delResult.Error != nil {
		logx.Error("Failed to delete user for metrics demo",
			logx.String("key", "user:456"),
			logx.ErrorField(delResult.Error))
	} else {
		fmt.Printf("✅ Deleted user\n")
		logx.Info("Successfully deleted user for metrics demo",
			logx.String("key", "user:456"))
	}

	fmt.Println("📋 Check http://localhost:8080/metrics for Prometheus metrics")
	fmt.Println()
}

func demoLogging(c cachex.Cache[User]) {
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

	ctx := context.Background()

	for _, tc := range testCases {
		// Set operation
		setResult := <-c.Set(ctx, tc.key, tc.user, 10*time.Minute)
		if setResult.Error != nil {
			logx.Error("Failed to set item for logging demo",
				logx.String("key", tc.key),
				logx.String("user_id", tc.user.ID),
				logx.ErrorField(setResult.Error))
		} else {
			fmt.Printf("✅ Set %s\n", tc.key)
			logx.Info("Successfully set item for logging demo",
				logx.String("key", tc.key),
				logx.String("user_id", tc.user.ID),
				logx.String("user_name", tc.user.Name))
		}

		// Get operation
		getResult := <-c.Get(ctx, tc.key)
		if getResult.Error != nil {
			logx.Error("Failed to get item for logging demo",
				logx.String("key", tc.key),
				logx.ErrorField(getResult.Error))
		} else {
			fmt.Printf("✅ Get %s (found: %t)\n", tc.key, getResult.Found)
			logx.Info("Successfully retrieved item for logging demo",
				logx.String("key", tc.key),
				logx.Bool("found", getResult.Found))
		}
	}

	fmt.Println("📋 Check console output for structured logs with enhanced fields")
	fmt.Println()
}

func demoErrorTracking(c cachex.Cache[User]) {
	fmt.Println("🚨 Demonstrating Error Tracking...")

	ctx := context.Background()

	// Test with invalid operations to generate errors
	testCases := []struct {
		description string
		operation   func() error
	}{
		{
			"Get non-existent key",
			func() error {
				result := <-c.Get(ctx, "non:existent:key")
				return result.Error
			},
		},
		{
			"Delete non-existent key",
			func() error {
				result := <-c.Del(ctx, "non:existent:key")
				return result.Error
			},
		},
		{
			"MGet with empty keys",
			func() error {
				result := <-c.MGet(ctx)
				return result.Error
			},
		},
	}

	for _, tc := range testCases {
		if err := tc.operation(); err != nil {
			fmt.Printf("❌ %s: %v\n", tc.description, err)
			logx.Error("Error tracking demo operation failed",
				logx.String("operation", tc.description),
				logx.ErrorField(err))
		} else {
			fmt.Printf("✅ %s: No error\n", tc.description)
			logx.Info("Error tracking demo operation completed successfully",
				logx.String("operation", tc.description))
		}
	}

	fmt.Println("📊 Error metrics will be recorded in Prometheus")
	fmt.Println()
}
