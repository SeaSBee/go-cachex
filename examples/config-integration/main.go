package main

import (
	"context"
	"fmt"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// User struct for demonstration
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	fmt.Println("=== CacheX Configuration Integration Demo ===")

	// Example 1: Loading from YAML file
	fmt.Println("\n=== Example 1: Loading from YAML file ===")
	config, err := cachex.LoadConfig("examples/config-examples/simple-memory-config.yaml")
	if err != nil {
		logx.Fatal("Failed to load configuration from YAML file",
			logx.String("config_file", "examples/config-examples/simple-memory-config.yaml"),
			logx.ErrorField(err))
	}

	cache, err := cachex.NewFromConfig[User](config)
	if err != nil {
		logx.Fatal("Failed to create cache from YAML configuration",
			logx.String("config_source", "yaml"),
			logx.ErrorField(err))
	}
	defer cache.Close()

	user := User{ID: "1", Name: "John Doe", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user in cache from YAML config",
			logx.String("key", "user:1"),
			logx.String("user_id", user.ID),
			logx.ErrorField(setResult.Error))
	} else {
		fmt.Printf("Retrieved user: %+v\n", user)
	}

	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		logx.Error("Failed to get user from cache with YAML config",
			logx.String("key", "user:1"),
			logx.ErrorField(getResult.Error))
	} else if getResult.Found {
		fmt.Printf("Retrieved user: %+v\n", getResult.Value)
		logx.Info("Successfully retrieved user from cache with YAML configuration",
			logx.String("key", "user:1"),
			logx.String("user_id", getResult.Value.ID),
			logx.Bool("found", getResult.Found))
	}

	// Example 2: Environment variable overrides
	fmt.Println("\n=== Example 2: Environment variable overrides ===")

	// Set environment variables to override YAML settings
	// Note: In a real application, these would be set in the environment
	// For demo purposes, we'll simulate the behavior

	config2, err := cachex.LoadConfig("examples/config-examples/simple-memory-config.yaml")
	if err != nil {
		logx.Fatal("Failed to load configuration with environment overrides",
			logx.String("config_file", "examples/config-examples/simple-memory-config.yaml"),
			logx.ErrorField(err))
	}

	anyCache, err := cachex.NewAnyFromConfig(config2)
	if err != nil {
		logx.Fatal("Failed to create any cache with environment overrides",
			logx.String("config_source", "yaml_with_env"),
			logx.ErrorField(err))
	}
	defer anyCache.Close()

	anySetResult := <-anyCache.Set(ctx, "test-key", "test-value", 0)
	if anySetResult.Error != nil {
		logx.Error("Failed to set test value with environment overrides",
			logx.String("key", "test-key"),
			logx.String("value", "test-value"),
			logx.ErrorField(anySetResult.Error))
	} else {
		fmt.Printf("Retrieved value: test-value\n")
	}

	anyGetResult := <-anyCache.Get(ctx, "test-key")
	if anyGetResult.Error != nil {
		logx.Error("Failed to get test value with environment overrides",
			logx.String("key", "test-key"),
			logx.ErrorField(anyGetResult.Error))
	} else if anyGetResult.Found {
		fmt.Printf("Retrieved value: %v\n", anyGetResult.Value)
		logx.Info("Successfully retrieved value with environment overrides",
			logx.String("key", "test-key"),
			logx.String("value", fmt.Sprintf("%v", anyGetResult.Value)),
			logx.Bool("found", anyGetResult.Found))
	}

	// Example 3: Environment-only configuration
	fmt.Println("\n=== Example 3: Environment-only configuration ===")

	// Load configuration without YAML file (environment variables only)
	config3, err := cachex.LoadConfig("")
	if err != nil {
		logx.Fatal("Failed to load environment-only configuration",
			logx.String("config_source", "environment_only"),
			logx.ErrorField(err))
	}

	envCache, err := cachex.NewFromConfig[string](config3)
	if err != nil {
		logx.Fatal("Failed to create cache from environment-only configuration",
			logx.String("config_source", "environment_only"),
			logx.ErrorField(err))
	}
	defer envCache.Close()

	envSetResult := <-envCache.Set(ctx, "env-key", "environment configured", 0)
	if envSetResult.Error != nil {
		logx.Error("Failed to set environment test value",
			logx.String("key", "env-key"),
			logx.String("value", "environment configured"),
			logx.ErrorField(envSetResult.Error))
	} else {
		fmt.Printf("Retrieved env value: environment configured\n")
	}

	envGetResult := <-envCache.Get(ctx, "env-key")
	if envGetResult.Error != nil {
		logx.Error("Failed to get environment test value",
			logx.String("key", "env-key"),
			logx.ErrorField(envGetResult.Error))
	} else if envGetResult.Found {
		fmt.Printf("Retrieved env value: %s\n", envGetResult.Value)
		logx.Info("Successfully retrieved environment-only configured value",
			logx.String("key", "env-key"),
			logx.String("value", envGetResult.Value),
			logx.Bool("found", envGetResult.Found))
	}

	// Example 4: Configuration validation
	fmt.Println("\n=== Example 4: Configuration validation ===")

	// Test validation by trying to create an invalid configuration
	invalidConfig := &cachex.CacheConfig{
		Memory: &cachex.MemoryConfig{MaxSize: 1000},
		Redis:  &cachex.RedisConfig{Addr: "localhost:6379"}, // This should fail validation
	}

	_, err = cachex.NewFromConfig[string](invalidConfig)
	if err != nil {
		fmt.Printf("✅ Validation correctly caught error: %v\n", err)
		logx.Info("Configuration validation correctly prevented multiple store configuration",
			logx.String("validation_error", err.Error()),
			logx.String("stores_configured", "memory_and_redis"))
	} else {
		logx.Error("Configuration validation failed to catch multiple store configuration")
	}

	fmt.Println("\n=== Configuration examples completed ===")
	logx.Info("Configuration integration demo completed successfully",
		logx.String("demo_type", "config_integration"),
		logx.Int("examples_completed", 4))
}
