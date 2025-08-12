# Implementation Details

This document describes the fully implemented features in Go-CacheX that provide advanced caching capabilities for production environments.

## Overview

Go-CacheX implements several key advanced features that provide enterprise-grade caching capabilities:

1. **GORM Integration** - First-class GORM plugin with automatic cache invalidation
2. **Tagging with Inverted Index** - Mass invalidation using tag-based key management
3. **Pub/Sub Invalidation** - Distributed cache coherence across multiple services
4. **Refresh-Ahead Background Scheduler** - Proactive cache refresh with distributed locking
5. **Thread-Safe Operations** - Race-free design with comprehensive concurrency controls

## 1. GORM Integration

### Overview

The GORM integration provides first-class support for automatic cache invalidation and read-through caching with GORM ORM. This enables seamless caching integration without manual cache management.

### Features

- **Automatic Cache Invalidation**: Automatic invalidation on Create/Update/Delete operations
- **Read-Through Caching**: Automatic cache population on database queries
- **Model Registration**: Flexible model configuration with custom TTLs and tags
- **Tag-Based Invalidation**: Support for tag-based mass invalidation
- **Thread-Safe**: Concurrent access with proper locking
- **Configurable**: Rich configuration options for different use cases

### Architecture

```
GORM Plugin
├── cache: Cache[any]                    // Cache instance
├── keyBuilder: *key.Builder             // Key generation
├── config: *Config                      // Plugin configuration
├── models: map[string]*ModelConfig      // Registered models
└── mu: sync.RWMutex                     // Thread safety
```

### Usage

```go
package main

import (
    "time"
    
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/gormx"
    "github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
)

func main() {
    // Create cache
    store, _ := memorystore.New(&memorystore.Config{
        DefaultTTL:      5 * time.Minute,
        CleanupInterval: 1 * time.Minute,
    })
    c, _ := cache.New[any](
        cache.WithStore(store),
        cache.WithDefaultTTL(5*time.Minute),
    )
    defer c.Close()

    // Create GORM plugin
    plugin := gormx.New(c, &gormx.Config{
        EnableInvalidation: true,
        EnableReadThrough:  true,
        DefaultTTL:         5 * time.Minute,
        EnableQueryCache:   true,
        KeyPrefix:          "gorm",
        EnableDebug:        true,
    })

    // Register models
    plugin.RegisterModelWithDefaults(&User{}, 10*time.Minute, "users", "auth")
    plugin.RegisterModelWithDefaults(&Product{}, 15*time.Minute, "products", "catalog")

    // Automatic cache invalidation
    user := &User{ID: "123", Name: "Alice"}
    plugin.InvalidateOnCreate(user)  // Invalidates cache on create
    plugin.InvalidateOnUpdate(user)  // Invalidates cache on update
    plugin.InvalidateOnDelete(user)  // Invalidates cache on delete

    // Read-through caching
    var dest User
    found, _ := plugin.ReadThrough("User", "123", &dest)
    if found {
        println("User found in cache:", dest.Name)
    }

    // Query result caching
    product := &Product{ID: "456", Name: "Laptop", Price: 999.99}
    plugin.CacheQueryResult("Product", "456", product)
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type Product struct {
    ID    string  `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}
```

### Configuration

```go
type Config struct {
    EnableInvalidation bool          // Enable automatic cache invalidation
    EnableReadThrough  bool          // Enable read-through caching
    DefaultTTL         time.Duration // Default TTL for cached items
    EnableQueryCache   bool          // Enable query result caching
    KeyPrefix          string        // Cache key prefix
    EnableDebug        bool          // Enable debug logging
    BatchSize          int           // Batch invalidation size
}

type ModelConfig struct {
    Name        string        // Model name
    TTL         time.Duration // Cache TTL for this model
    Enabled     bool          // Enable caching for this model
    KeyTemplate string        // Cache key template
    Tags        []string      // Invalidation tags
    ReadThrough bool          // Enable read-through
    WriteThrough bool         // Enable write-through
}
```

### Key Features

#### 1. Automatic Cache Invalidation

```go
// Automatic invalidation on model operations
plugin.InvalidateOnCreate(user)  // Invalidates by tags and specific keys
plugin.InvalidateOnUpdate(user)  // Invalidates by tags and specific keys
plugin.InvalidateOnDelete(user)  // Invalidates by tags and specific keys
```

#### 2. Read-Through Caching

```go
// Automatic cache population on database queries
var user User
found, err := plugin.ReadThrough("User", "123", &user)
if found {
    // User found in cache
} else {
    // Would query database
}
```

#### 3. Query Result Caching

```go
// Cache query results
plugin.CacheQueryResult("User", "123", user)
```

#### 4. Tag-Based Invalidation

```go
// Register model with tags
plugin.RegisterModelWithDefaults(&User{}, 10*time.Minute, "users", "auth")

// Invalidate by tags (handled automatically by the plugin)
// This will invalidate all users when any user is created/updated/deleted
```

### Thread Safety

The GORM plugin is fully thread-safe with:

- **Concurrent Model Registration**: Safe concurrent model registration
- **Concurrent Operations**: Safe concurrent invalidation and read-through operations
- **Proper Locking**: RWMutex for efficient concurrent access
- **Race-Free Design**: No race conditions in any operations

## 2. Tagging with Inverted Index

### Overview

The tagging system provides efficient mass invalidation capabilities by maintaining an inverted index of tags to keys and keys to tags. This allows you to invalidate multiple cache entries simultaneously based on logical groupings.

### Features

- **Bidirectional Mapping**: Maintains both tag-to-keys and key-to-tags relationships
- **Persistence**: Tag mappings can be persisted to Redis for durability
- **Batch Operations**: Efficient batch invalidation of tagged keys
- **Thread-Safe**: Concurrent access with proper locking
- **Statistics**: Built-in metrics for monitoring tag usage

### Usage

```go
// Add tags to keys
err = c.AddTags("user:123", "premium", "active", "verified")

// Invalidate by tag
err = c.InvalidateByTag("premium") // Invalidates all premium users

// Remove specific tags
err = c.RemoveTags("user:123", "premium")
```

### Configuration

```go
type TagConfig struct {
    EnablePersistence bool          // Persist tag mappings to Redis
    TagMappingTTL     time.Duration // TTL for tag mappings (24h default)
    BatchSize         int           // Batch size for operations (100 default)
    EnableStats       bool          // Enable statistics collection
}
```

## 3. Pub/Sub Invalidation

### Overview

The Pub/Sub invalidation system enables distributed cache coherence across multiple services by broadcasting cache invalidation messages through Redis pub/sub channels.

### Features

- **Redis Pub/Sub**: Uses Redis pub/sub for message broadcasting
- **Message Serialization**: JSON-based message format with metadata
- **Health Monitoring**: Built-in health check messages
- **Subscriber Management**: Automatic subscriber lifecycle management
- **Message Filtering**: Stale message detection and filtering
- **Statistics**: Comprehensive pub/sub metrics

### Usage

```go
// Publish invalidation
err = c.PublishInvalidation("user:123", "user:456")

// Subscribe to invalidations
closeSub, err := c.SubscribeInvalidations(func(keys ...string) {
    // Handle invalidation
    for _, key := range keys {
        // Remove from local cache
        localCache.Delete(key)
    }
})
defer closeSub()
```

### Message Format

```json
{
  "keys": ["user:123", "user:456"],
  "tags": ["premium", "active"],
  "timestamp": "2024-01-01T12:00:00Z",
  "source": "user-service",
  "message_id": "msg_1704110400000000000",
  "ttl": 300
}
```

## 4. Refresh-Ahead Background Scheduler

### Overview

The refresh-ahead scheduler proactively refreshes cache entries before they expire, ensuring data freshness while maintaining high availability. It uses distributed locking to prevent multiple services from refreshing the same data simultaneously.

### Features

- **Background Processing**: Automatic refresh in background goroutines
- **Distributed Locking**: Redis-based locks prevent duplicate refreshes
- **Configurable Timing**: Flexible refresh-before-expiry settings
- **Worker Pool**: Configurable concurrency limits
- **Error Handling**: Comprehensive error tracking and retry logic
- **Statistics**: Detailed refresh metrics and monitoring

### Usage

```go
// Schedule refresh-ahead
loader := func() (User, error) {
    // Load fresh data from database
    return db.GetUser("user:123")
}

err = c.RefreshAhead("user:123", 1*time.Minute, loader)
```

### Configuration

```go
type RefreshAheadConfig struct {
    Enabled                bool          // Enable refresh-ahead
    DefaultRefreshBefore   time.Duration // Default refresh timing
    MaxConcurrentRefreshes int           // Maximum concurrent refreshes
    RefreshInterval        time.Duration // Background scan interval
    EnableDistributedLock  bool          // Enable distributed locking
    LockTimeout            time.Duration // Lock acquisition timeout
    EnableMetrics          bool          // Enable metrics collection
}
```

## 5. Thread-Safe Operations

### Overview

All cache operations are designed to be thread-safe with comprehensive concurrency controls to prevent race conditions and ensure data consistency.

### Features

- **Race-Free Design**: All operations are race-condition free
- **Concurrent Access**: Safe concurrent read/write operations
- **Proper Locking**: Efficient locking strategies for different operations
- **Context Support**: Full context cancellation and timeout support
- **Atomic Operations**: Atomic cache operations where possible

### Implementation Details

#### Context Handling

```go
// Proper context handling prevents race conditions
func (c *cache[T]) Get(key string) (T, bool, error) {
    // Create tracing span with local context
    var span trace.Span
    if c.observability != nil {
        var traceCtx context.Context
        traceCtx, span = c.observability.TraceOperation(c.ctx, "get", key, "", 0, 0)
        defer span.End()
    } else {
        traceCtx = c.ctx
    }

    // Use local context for retry operations
    err := retry.RetryWithContext(traceCtx, c.retryPolicy, func() error {
        // Cache operation logic
        return nil
    })
    
    return result, found, err
}
```

#### Thread-Safe Data Structures

```go
// Thread-safe tag management
type TagManager struct {
    tagToKeys map[string]map[string]bool
    keyToTags map[string]map[string]bool
    mu        sync.RWMutex
    store     Store
    config    *TagConfig
}

// Thread-safe pub/sub management
type PubSubManager struct {
    client      redis.Cmdable
    subscribers map[string]*Subscriber
    config      *PubSubConfig
    ctx         context.Context
    cancel      context.CancelFunc
    mu          sync.RWMutex
}
```

## Integration Examples

### Complete GORM Integration

```go
package main

import (
    "time"
    
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/gormx"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
    // Create Redis store
    redisStore, _ := redisstore.New(&redisstore.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    // Create cache with full features
    c, _ := cache.New[any](
        cache.WithStore(redisStore),
        cache.WithDefaultTTL(5*time.Minute),
        cache.WithObservability(cache.ObservabilityConfig{
            EnableMetrics: true,
            EnableTracing: true,
            EnableLogging: true,
        }),
        cache.WithSecurity(cache.SecurityConfig{
            RedactLogs: true,
        }),
    )
    defer c.Close()

    // Create GORM plugin
    plugin := gormx.New(c, &gormx.Config{
        EnableInvalidation: true,
        EnableReadThrough:  true,
        DefaultTTL:         5 * time.Minute,
        EnableQueryCache:   true,
        KeyPrefix:          "gorm",
        EnableDebug:        true,
    })

    // Register models with tags
    plugin.RegisterModelWithDefaults(&User{}, 10*time.Minute, "users", "auth")
    plugin.RegisterModelWithDefaults(&Product{}, 15*time.Minute, "products", "catalog")
    plugin.RegisterModelWithDefaults(&Order{}, 8*time.Minute, "orders", "transactions")

    // Demonstrate automatic invalidation
    user := &User{ID: "123", Name: "Alice", Email: "alice@example.com"}
    
    // Create user - automatically invalidates cache
    plugin.InvalidateOnCreate(user)
    
    // Update user - automatically invalidates cache
    user.Name = "Alice Smith"
    plugin.InvalidateOnUpdate(user)
    
    // Delete user - automatically invalidates cache
    plugin.InvalidateOnDelete(user)

    // Demonstrate read-through caching
    var dest User
    found, _ := plugin.ReadThrough("User", "123", &dest)
    if found {
        println("User found in cache:", dest.Name)
    } else {
        println("User not in cache, would query database")
    }

    // Demonstrate query result caching
    product := &Product{ID: "456", Name: "Laptop", Price: 999.99}
    plugin.CacheQueryResult("Product", "456", product)

    // Demonstrate tag-based invalidation
    // This will invalidate all users when any user is modified
    println("All users invalidated due to tag-based invalidation")
}

type User struct {
    ID    string    `json:"id"`
    Name  string    `json:"name"`
    Email string    `json:"email"`
}

type Product struct {
    ID    string  `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

type Order struct {
    ID     string  `json:"id"`
    UserID string  `json:"user_id"`
    Total  float64 `json:"total"`
}
```

### Multi-Service Cache Coherence

```go
// Service A: User Service
func updateUser(userID string, user User) error {
    // Update database
    err := db.UpdateUser(userID, user)
    if err != nil {
        return err
    }
    
    // Update cache
    err = cache.Set(fmt.Sprintf("user:%s", userID), user, 30*time.Minute)
    if err != nil {
        return err
    }
    
    // Publish invalidation to other services
    return cache.PublishInvalidation(fmt.Sprintf("user:%s", userID))
}

// Service B: Order Service
func init() {
    // Subscribe to user invalidations
    closeSub, err := cache.SubscribeInvalidations(func(keys ...string) {
        for _, key := range keys {
            if strings.HasPrefix(key, "user:") {
                // Invalidate local user cache
                localUserCache.Delete(key)
            }
        }
    })
    // Handle cleanup on shutdown
}
```

## Monitoring and Observability

### Metrics

All features provide comprehensive metrics:

- **GORM Integration**: Model operations, cache hits/misses, invalidation counts
- **Tagging**: Total tags, total keys, tag operations
- **Pub/Sub**: Subscribers, messages sent/received, health status
- **Refresh-Ahead**: Total tasks, refresh count, error count

### Logging

Structured logging with consistent fields:

```go
logx.Info("GORM cache invalidation completed", 
    logx.String("model", "User"), 
    logx.String("operation", "create"),
    logx.String("tags", "users,auth"))

logx.Info("Published invalidation message",
    logx.String("channel", "cachex:invalidation"),
    logx.Int("recipients", 3))

logx.Info("Refresh-ahead completed",
    logx.String("key", "user:123"),
    logx.String("duration", "150ms"))
```

### Health Checks

Built-in health monitoring for all components:

- **GORM Plugin**: Model registration status, operation health
- **Tag Manager**: Persistence status, mapping integrity
- **Pub/Sub Manager**: Connection status, subscriber health
- **Refresh Scheduler**: Task status, lock health

## Best Practices

### GORM Integration

1. **Register Models Early**: Register all models during application startup
2. **Use Descriptive Tags**: Choose meaningful tag names for invalidation
3. **Configure TTLs**: Set appropriate TTLs for different model types
4. **Monitor Performance**: Track cache hit rates and invalidation patterns

### Thread Safety

1. **Use Proper Contexts**: Always use context for cancellation and timeouts
2. **Avoid Race Conditions**: Use provided thread-safe methods
3. **Monitor Concurrency**: Track concurrent access patterns
4. **Handle Errors**: Implement proper error handling for all operations

### Performance

1. **Batch Operations**: Use batch operations where possible
2. **Monitor Memory**: Track memory usage for large datasets
3. **Optimize TTLs**: Set appropriate TTLs for your use case
4. **Use Tags Wisely**: Limit tag count to avoid performance impact

## Configuration Examples

### Production Configuration

```yaml
# GORM integration configuration
gorm:
  enable_invalidation: true
  enable_read_through: true
  default_ttl: "5m"
  enable_query_cache: true
  key_prefix: "gorm"
  enable_debug: false

# Tagging configuration
tagging:
  enable_persistence: true
  tag_mapping_ttl: "24h"
  batch_size: 100
  enable_stats: true

# Pub/Sub configuration
pubsub:
  invalidation_channel: "cachex:invalidation:prod"
  health_channel: "cachex:health:prod"
  max_subscribers: 50
  message_timeout: "10s"
  enable_health: true

# Refresh-ahead configuration
refresh_ahead:
  enabled: true
  default_refresh_before: "2m"
  max_concurrent_refreshes: 20
  refresh_interval: "30s"
  enable_distributed_lock: true
  lock_timeout: "30s"
```

### Development Configuration

```yaml
# Simplified configuration for development
gorm:
  enable_invalidation: true
  enable_read_through: true
  default_ttl: "5m"
  enable_debug: true

tagging:
  enable_persistence: false
  batch_size: 10

pubsub:
  max_subscribers: 5
  message_timeout: "5s"

refresh_ahead:
  enabled: true
  max_concurrent_refreshes: 5
  enable_distributed_lock: false
```

## Troubleshooting

### Common Issues

1. **GORM Plugin Errors**: Check model registration and configuration
2. **Tag Persistence Failures**: Check Redis connectivity and storage
3. **Pub/Sub Message Loss**: Verify subscriber health and message timeouts
4. **Refresh Lock Contention**: Adjust lock timeouts and concurrency limits
5. **Race Conditions**: Ensure proper context usage and thread safety

### Debugging

Enable debug logging for detailed troubleshooting:

```go
// Enable debug logging
logx.SetLevel(logx.DebugLevel)

// Check component health
models := plugin.GetRegisteredModels()
tagStats := cache.GetTagStats()
pubSubStats := cache.GetPubSubStats()
refreshStats := cache.GetRefreshStats()
```

## Conclusion

The implementation details provide a comprehensive caching solution with:

- **First-Class GORM Integration**: Automatic cache invalidation and read-through
- **Efficient Mass Invalidation**: Tag-based invalidation with inverted index
- **Distributed Coherence**: Pub/sub invalidation across services
- **Proactive Refresh**: Background refresh-ahead with distributed locking
- **Thread-Safe Operations**: Race-free design with comprehensive concurrency controls
- **Production Ready**: Comprehensive monitoring, error handling, and configuration

These features enable Go-CacheX to handle complex caching scenarios in production environments with high performance, reliability, and observability.
