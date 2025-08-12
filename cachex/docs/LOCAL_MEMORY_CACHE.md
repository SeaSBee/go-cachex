# Local In-Memory Cache

Go-CacheX provides **local in-memory cache** with TTL + size cap for ultra-low latency operations. This feature implements a "hot" shard that caches frequently accessed data in memory with automatic eviction policies.

## Overview

The local in-memory cache provides:

- **Ultra-Low Latency**: Sub-millisecond access times for cached data
- **Configurable Capacity**: Size and memory limits with automatic eviction
- **Multiple Eviction Policies**: LRU, LFU, and TTL-based eviction
- **Background Cleanup**: Automatic expiration of expired items
- **Thread-Safe**: Race-free implementation for concurrent access
- **Statistics**: Detailed performance and usage metrics
- **Layered Architecture**: Can be combined with Redis for L1/L2 caching

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Go-CacheX Application                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Memory Store (L1)                         │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   LRU List  │  │ Access Map  │  │   Stats     │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │              Data Map                           │    │ │
│  │  │  key1 → {value, expiresAt, accessCount, size}  │    │ │
│  │  │  key2 → {value, expiresAt, accessCount, size}  │    │ │
│  │  │  key3 → {value, expiresAt, accessCount, size}  │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                              │
│                              ▼                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Redis Store (L2)                          │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │              Redis Cluster                      │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Features

### Core Capabilities

- **High Performance**: Optimized for read-heavy workloads
- **Memory Efficient**: Configurable memory limits with automatic eviction
- **TTL Support**: Automatic expiration with background cleanup
- **Bulk Operations**: Efficient MGet/MSet for batch processing
- **Atomic Operations**: Thread-safe increment/decrement operations
- **Statistics**: Detailed hit/miss ratios and memory usage tracking

### Eviction Policies

#### 1. LRU (Least Recently Used) - Default
```go
config := memorystore.DefaultConfig()
config.EvictionPolicy = memorystore.EvictionPolicyLRU
```
- Evicts least recently accessed items first
- Best for general-purpose caching
- Maintains access order for optimal performance

#### 2. LFU (Least Frequently Used)
```go
config := memorystore.DefaultConfig()
config.EvictionPolicy = memorystore.EvictionPolicyLFU
```
- Evicts least frequently accessed items first
- Best for long-term caching patterns
- Tracks access frequency for each item

#### 3. TTL (Time To Live)
```go
config := memorystore.DefaultConfig()
config.EvictionPolicy = memorystore.EvictionPolicyTTL
```
- Evicts items with shortest remaining TTL first
- Best for time-sensitive data
- Prioritizes freshness over access patterns

## Configuration

### Default Configuration
```go
config := memorystore.DefaultConfig()
// MaxSize: 10000 items
// MaxMemoryMB: 100 MB
// DefaultTTL: 5 minutes
// CleanupInterval: 1 minute
// EvictionPolicy: LRU
// EnableStats: true
```

### High Performance Configuration
```go
config := memorystore.HighPerformanceConfig()
// MaxSize: 50000 items
// MaxMemoryMB: 500 MB
// DefaultTTL: 10 minutes
// CleanupInterval: 30 seconds
// EvictionPolicy: LRU
// EnableStats: true
```

### Resource Constrained Configuration
```go
config := memorystore.ResourceConstrainedConfig()
// MaxSize: 1000 items
// MaxMemoryMB: 10 MB
// DefaultTTL: 2 minutes
// CleanupInterval: 2 minutes
// EvictionPolicy: TTL
// EnableStats: false
```

### Custom Configuration
```go
config := &memorystore.Config{
    MaxSize:         5000,
    MaxMemoryMB:     50,
    DefaultTTL:      3 * time.Minute,
    CleanupInterval: 30 * time.Second,
    EvictionPolicy:  memorystore.EvictionPolicyLRU,
    EnableStats:     true,
}
```

## Usage Examples

### Basic Memory Store
```go
package main

import (
    "context"
    "time"
    
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
    "github.com/SeaSBee/go-cachex/cachex/pkg/codec"
)

func main() {
    // Create memory store
    memoryStore, err := memorystore.New(memorystore.DefaultConfig())
    if err != nil {
        panic(err)
    }
    defer memoryStore.Close()

    // Create cache with memory store
    c, err := cache.New[User](
        cache.WithStore(memoryStore),
        cache.WithCodec(codec.NewJSONCodec()),
        cache.WithDefaultTTL(5*time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // Use cache
    user := User{ID: 123, Name: "John Doe"}
    err = c.Set("user:123", user, 10*time.Minute)
    if err != nil {
        panic(err)
    }

    cachedUser, found, err := c.Get("user:123")
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("Found user: %+v\n", cachedUser)
    }
}
```

### Layered Cache (Memory + Redis)
```go
package main

import (
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/layeredstore"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
    // Create Redis store (L2)
    redisStore, err := redisstore.New(&redisstore.Config{
        Addr: "localhost:6379",
    })
    if err != nil {
        panic(err)
    }
    defer redisStore.Close()

    // Create layered store
    layeredConfig := layeredstore.HighPerformanceConfig()
    layeredConfig.WritePolicy = layeredstore.WritePolicyBehind
    layeredConfig.ReadPolicy = layeredstore.ReadPolicyThrough

    layeredStore, err := layeredstore.New(redisStore, layeredConfig)
    if err != nil {
        panic(err)
    }
    defer layeredStore.Close()

    // Create cache with layered store
    c, err := cache.New[User](
        cache.WithStore(layeredStore),
        cache.WithCodec(codec.NewJSONCodec()),
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // Use layered cache
    user := User{ID: 456, Name: "Jane Smith"}
    err = c.Set("user:456", user, 10*time.Minute)
    if err != nil {
        panic(err)
    }

    // First read populates L1 from L2
    cachedUser, found, err := c.Get("user:456")
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("Found user: %+v\n", cachedUser)
    }
}
```

### Advanced Usage with Statistics
```go
func monitorCachePerformance(store *memorystore.Store) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := store.GetStats()
        
        hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
        memoryUsageMB := float64(stats.MemoryUsage) / 1024 / 1024
        
        fmt.Printf("Cache Stats:\n")
        fmt.Printf("  Hit Rate: %.2f%%\n", hitRate)
        fmt.Printf("  Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
        fmt.Printf("  Size: %d items\n", stats.Size)
        fmt.Printf("  Memory Usage: %.2f MB\n", memoryUsageMB)
        fmt.Printf("  Evictions: %d\n", stats.Evictions)
        fmt.Printf("  Expirations: %d\n", stats.Expirations)
    }
}
```

## Performance Characteristics

### Benchmarks
- **Read Operations**: ~100,000 ops/sec (single-threaded)
- **Write Operations**: ~80,000 ops/sec (single-threaded)
- **Memory Overhead**: ~50 bytes per item (excluding value size)
- **Concurrent Access**: Linear scaling with CPU cores

### Memory Usage
```go
// Approximate memory usage calculation
memoryUsage := len(data) * 50 + totalValueSize + len(accessOrder) * 8
```

### Eviction Performance
- **LRU Eviction**: O(1) average case
- **LFU Eviction**: O(n) worst case
- **TTL Eviction**: O(n) worst case
- **Background Cleanup**: O(n) per cleanup cycle

## Best Practices

### 1. Capacity Planning
```go
// Estimate memory usage
estimatedMemoryMB := (maxItems * 50) / 1024 / 1024 + totalValueSizeMB
config := &memorystore.Config{
    MaxSize:     maxItems,
    MaxMemoryMB: int(estimatedMemoryMB * 1.2), // 20% buffer
}
```

### 2. TTL Strategy
```go
// Use appropriate TTL based on data characteristics
config := memorystore.DefaultConfig()
config.DefaultTTL = 5 * time.Minute  // For frequently changing data
// config.DefaultTTL = 1 * time.Hour   // For stable data
// config.DefaultTTL = 24 * time.Hour  // For rarely changing data
```

### 3. Eviction Policy Selection
```go
// Choose eviction policy based on access patterns
switch accessPattern {
case "temporal":
    config.EvictionPolicy = memorystore.EvictionPolicyLRU
case "frequency":
    config.EvictionPolicy = memorystore.EvictionPolicyLFU
case "freshness":
    config.EvictionPolicy = memorystore.EvictionPolicyTTL
}
```

### 4. Monitoring and Alerting
```go
// Monitor cache performance
func checkCacheHealth(stats memorystore.Stats) bool {
    hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses)
    memoryUsagePercent := float64(stats.MemoryUsage) / float64(config.MaxMemoryMB*1024*1024) * 100
    
    return hitRate > 0.8 && memoryUsagePercent < 80
}
```

## Integration with Caching Patterns

### Cache-Aside with Memory Store
```go
func getUserWithCache(id int) (User, error) {
    key := fmt.Sprintf("user:%d", id)
    
    // Try memory cache first
    user, found, err := cache.Get(key)
    if err != nil {
        return User{}, err
    }
    if found {
        return user, nil
    }
    
    // Load from database
    user, err = loadUserFromDB(id)
    if err != nil {
        return User{}, err
    }
    
    // Store in memory cache
    err = cache.Set(key, user, 10*time.Minute)
    if err != nil {
        log.Printf("Failed to cache user: %v", err)
    }
    
    return user, nil
}
```

### Read-Through with Memory Store
```go
func getUserReadThrough(id int) (User, error) {
    key := fmt.Sprintf("user:%d", id)
    
    return cache.ReadThrough(key, 10*time.Minute, func(ctx context.Context) (User, error) {
        return loadUserFromDB(id)
    })
}
```

### Write-Through with Memory Store
```go
func saveUserWriteThrough(user User) error {
    key := fmt.Sprintf("user:%d", user.ID)
    
    return cache.WriteThrough(key, user, 10*time.Minute, func(ctx context.Context) error {
        return saveUserToDB(user)
    })
}
```

## Troubleshooting

### Common Issues

#### 1. High Memory Usage
```go
// Check if items are being evicted properly
stats := store.GetStats()
if stats.Evictions == 0 && stats.Size >= config.MaxSize {
    // Items not being evicted - check eviction policy
    fmt.Println("Warning: Cache at capacity but no evictions")
}
```

#### 2. Low Hit Rate
```go
// Monitor hit rate and adjust TTL
stats := store.GetStats()
hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses)
if hitRate < 0.5 {
    // Consider increasing TTL or cache size
    fmt.Println("Warning: Low cache hit rate")
}
```

#### 3. High Eviction Rate
```go
// Check if cache size is too small
stats := store.GetStats()
evictionRate := float64(stats.Evictions) / float64(stats.Hits+stats.Misses)
if evictionRate > 0.1 {
    // Consider increasing cache size
    fmt.Println("Warning: High eviction rate")
}
```

### Performance Tuning

#### 1. Optimize for Read-Heavy Workloads
```go
config := memorystore.HighPerformanceConfig()
config.EvictionPolicy = memorystore.EvictionPolicyLRU
config.MaxSize = 100000  // Large cache for read-heavy workloads
```

#### 2. Optimize for Write-Heavy Workloads
```go
config := memorystore.DefaultConfig()
config.EvictionPolicy = memorystore.EvictionPolicyTTL
config.CleanupInterval = 10 * time.Second  // More frequent cleanup
```

#### 3. Optimize for Memory-Constrained Environments
```go
config := memorystore.ResourceConstrainedConfig()
config.MaxSize = 1000
config.MaxMemoryMB = 10
config.EnableStats = false  // Disable stats to save memory
```

## Conclusion

The local in-memory cache provides ultra-low latency access to frequently used data with automatic management of capacity and expiration. When combined with Redis in a layered architecture, it creates a powerful caching solution that balances performance, persistence, and resource utilization.

Key benefits:
- **Ultra-low latency** for cached data
- **Automatic resource management** with configurable limits
- **Multiple eviction policies** for different use cases
- **Thread-safe** concurrent access
- **Rich statistics** for monitoring and optimization
- **Seamless integration** with existing caching patterns
