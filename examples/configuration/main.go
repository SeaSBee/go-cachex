package main

import (
	"context"
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
	logx.Info("Starting Go-CacheX Configuration Demo")

	// Example 1: Basic Configuration
	demoBasicConfiguration()

	// Example 2: Advanced Configuration
	demoAdvancedConfiguration()

	// Example 3: Custom Configuration
	demoCustomConfiguration()

	logx.Info("Configuration Demo Complete")
}

func demoBasicConfiguration() {
	logx.Info("Starting Basic Configuration Demo")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}

	// Create cache with basic configuration
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	logx.Info("Cache created with basic configuration",
		logx.String("default_ttl", (5*time.Minute).String()))

	// Test basic operations
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:1"
	ctx := context.Background()
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}

	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("Successfully retrieved user",
			logx.String("key", key),
			logx.String("name", getResult.Value.Name))
	} else {
		logx.Warn("User not found",
			logx.String("key", key))
	}
}

func demoAdvancedConfiguration() {
	logx.Info("Starting Advanced Configuration Demo")

	ctx := context.Background()

	// Create memory store with custom configuration
	memoryConfig := cachex.DefaultMemoryConfig()
	memoryConfig.MaxSize = 1000 // 1000 items max
	memoryConfig.CleanupInterval = 30 * time.Second

	memoryStore, err := cachex.NewMemoryStore(memoryConfig)
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}

	// Create cache with advanced configuration
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(10*time.Minute),
		cachex.WithObservability(cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		}),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	logx.Info("Cache created with advanced configuration",
		logx.String("default_ttl", (10*time.Minute).String()),
		logx.Int("max_size", 1000),
		logx.String("cleanup_interval", (30*time.Second).String()),
		logx.Bool("observability_enabled", true))

	// Test advanced features
	user := User{
		ID:        "user:2",
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:2"
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}

	logx.Info("Successfully set user in advanced cache",
		logx.String("key", key),
		logx.String("name", user.Name))
}

func demoCustomConfiguration() {
	logx.Info("Starting Custom Configuration Demo")

	ctx := context.Background()

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}

	// Create cache with custom configuration
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(1*time.Hour),
		cachex.WithMaxRetries(5),
		cachex.WithRetryDelay(200*time.Millisecond),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	logx.Info("Cache created with custom configuration",
		logx.String("default_ttl", (1*time.Hour).String()),
		logx.Int("max_retries", 5),
		logx.String("retry_delay", (200*time.Millisecond).String()))

	// Test custom configuration
	user := User{
		ID:        "user:3",
		Name:      "Bob Wilson",
		Email:     "bob@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:3"
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user",
			logx.String("key", key),
			logx.ErrorField(setResult.Error))
		return
	}

	// Test with context
	ctx = context.Background()
	cWithCtx := c.WithContext(ctx)

	getResult := <-cWithCtx.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user",
			logx.String("key", key),
			logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("Successfully retrieved user with context",
			logx.String("key", key),
			logx.String("name", getResult.Value.Name))
	} else {
		logx.Warn("User not found",
			logx.String("key", key))
	}
}
