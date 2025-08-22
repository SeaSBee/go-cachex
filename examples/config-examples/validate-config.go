package main

import (
	"context"
	"fmt"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

func main() {
	fmt.Println("=== CacheX Configuration Validation ===")

	// Test comprehensive configuration
	config, err := cachex.LoadConfig("examples/config-examples/comprehensive-config.yaml")
	if err != nil {
		logx.Fatal("Failed to load comprehensive config", logx.ErrorField(err))
	}

	fmt.Println("✅ Configuration loaded successfully!")

	// Display configuration summary
	fmt.Println("\n=== Configuration Summary ===")

	// Check which store is configured
	if config.Memory != nil {
		fmt.Printf("📦 Store Type: Memory\n")
		fmt.Printf("   Max Size: %d items\n", config.Memory.MaxSize)
		fmt.Printf("   Max Memory: %d MB\n", config.Memory.MaxMemoryMB)
		fmt.Printf("   Default TTL: %v\n", config.Memory.DefaultTTL)
		fmt.Printf("   Eviction Policy: %s\n", config.Memory.EvictionPolicy)
	} else if config.Redis != nil {
		fmt.Printf("📦 Store Type: Redis\n")
		fmt.Printf("   Address: %s\n", config.Redis.Addr)
		fmt.Printf("   Database: %d\n", config.Redis.DB)
		fmt.Printf("   Pool Size: %d\n", config.Redis.PoolSize)
		fmt.Printf("   TLS Enabled: %v\n", config.Redis.TLS.Enabled)
	} else if config.Ristretto != nil {
		fmt.Printf("📦 Store Type: Ristretto\n")
		fmt.Printf("   Max Items: %d\n", config.Ristretto.MaxItems)
		fmt.Printf("   Max Memory: %d bytes\n", config.Ristretto.MaxMemoryBytes)
		fmt.Printf("   Default TTL: %v\n", config.Ristretto.DefaultTTL)
	} else if config.Layered != nil {
		fmt.Printf("📦 Store Type: Layered\n")
		fmt.Printf("   Read Policy: %s\n", config.Layered.ReadPolicy)
		fmt.Printf("   Write Policy: %s\n", config.Layered.WritePolicy)
		fmt.Printf("   Sync Interval: %v\n", config.Layered.SyncInterval)
	}

	// General settings
	fmt.Printf("\n⚙️  General Settings:\n")
	fmt.Printf("   Default TTL: %v\n", config.DefaultTTL)
	fmt.Printf("   Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("   Retry Delay: %v\n", config.RetryDelay)
	fmt.Printf("   Codec: %s\n", config.Codec)

	// Observability settings
	if config.Observability != nil {
		fmt.Printf("\n📊 Observability:\n")
		fmt.Printf("   Metrics: %v\n", config.Observability.EnableMetrics)
		fmt.Printf("   Tracing: %v\n", config.Observability.EnableTracing)
		fmt.Printf("   Logging: %v\n", config.Observability.EnableLogging)
	}

	// Security settings
	if config.Security != nil {
		fmt.Printf("\n🔒 Security:\n")
		if config.Security.Validation != nil {
			fmt.Printf("   Max Key Length: %d\n", config.Security.Validation.MaxKeyLength)
			fmt.Printf("   Max Value Size: %d bytes\n", config.Security.Validation.MaxValueSize)
			fmt.Printf("   Blocked Patterns: %d\n", len(config.Security.Validation.BlockedPatterns))
		}
		fmt.Printf("   Redaction Patterns: %d\n", len(config.Security.RedactionPatterns))
		fmt.Printf("   Secrets Prefix: %s\n", config.Security.SecretsPrefix)
	}

	// Tagging settings
	if config.Tagging != nil {
		fmt.Printf("\n🏷️  Tagging:\n")
		fmt.Printf("   Persistence: %v\n", config.Tagging.EnablePersistence)
		fmt.Printf("   Tag Mapping TTL: %v\n", config.Tagging.TagMappingTTL)
		fmt.Printf("   Batch Size: %d\n", config.Tagging.BatchSize)
		fmt.Printf("   Stats: %v\n", config.Tagging.EnableStats)
	}

	// Refresh ahead settings
	if config.RefreshAhead != nil {
		fmt.Printf("\n🔄 Refresh Ahead:\n")
		fmt.Printf("   Enabled: %v\n", config.RefreshAhead.Enabled)
		fmt.Printf("   Refresh Interval: %v\n", config.RefreshAhead.RefreshInterval)
		fmt.Printf("   Default Refresh Before: %v\n", config.RefreshAhead.DefaultRefreshBefore)
		fmt.Printf("   Metrics: %v\n", config.RefreshAhead.EnableMetrics)
	}

	// GORM settings
	if config.GORM != nil {
		fmt.Printf("\n🗄️  GORM Integration:\n")
		fmt.Printf("   Invalidation: %v\n", config.GORM.EnableInvalidation)
		fmt.Printf("   Read Through: %v\n", config.GORM.EnableReadThrough)
		fmt.Printf("   Default TTL: %v\n", config.GORM.DefaultTTL)
		fmt.Printf("   Query Cache: %v\n", config.GORM.EnableQueryCache)
		fmt.Printf("   Key Prefix: %s\n", config.GORM.KeyPrefix)
		fmt.Printf("   Debug: %v\n", config.GORM.EnableDebug)
		fmt.Printf("   Batch Size: %d\n", config.GORM.BatchSize)
	}

	// Test creating cache from config
	fmt.Println("\n=== Testing Cache Creation ===")

	cache, err := cachex.NewFromConfig[string](config)
	if err != nil {
		logx.Fatal("Failed to create cache from configuration",
			logx.String("config_type", "comprehensive"),
			logx.ErrorField(err))
	}
	defer cache.Close()

	fmt.Println("✅ Cache created successfully!")

	// Test basic operations
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "test-key", "test-value", 0)
	if setResult.Error != nil {
		logx.Error("Failed to set test value in cache",
			logx.String("key", "test-key"),
			logx.String("value", "test-value"),
			logx.ErrorField(setResult.Error))
	} else {
		fmt.Println("✅ Set operation successful")
	}

	getResult := <-cache.Get(ctx, "test-key")
	if getResult.Error != nil {
		logx.Error("Failed to get test value from cache",
			logx.String("key", "test-key"),
			logx.ErrorField(getResult.Error))
	} else if getResult.Found {
		fmt.Printf("✅ Get operation successful: %s\n", getResult.Value)
		logx.Info("Cache get operation completed successfully",
			logx.String("key", "test-key"),
			logx.String("value", getResult.Value),
			logx.Bool("found", getResult.Found))
	}

	fmt.Println("\n=== Configuration Validation Complete ===")
	logx.Info("Configuration validation completed successfully",
		logx.String("config_file", "examples/config-examples/comprehensive-config.yaml"),
		logx.String("store_type", getStoreType(config)))
}

// getStoreType returns the configured store type for logging
func getStoreType(config *cachex.CacheConfig) string {
	if config.Memory != nil {
		return "memory"
	} else if config.Redis != nil {
		return "redis"
	} else if config.Ristretto != nil {
		return "ristretto"
	} else if config.Layered != nil {
		return "layered"
	}
	return "unknown"
}
