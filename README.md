# Go-CacheX

A production-grade, highly concurrent, async cache library for Go with multiple backend support and comprehensive observability features.

[![Go Report Card](https://goreportcard.com/badge/github.com/SeaSBee/go-cachex)](https://goreportcard.com/report/github.com/SeaSBee/go-cachex)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SeaSBee/go-cachex)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://github.com/SeaSBee/go-cachex/workflows/Tests/badge.svg)](https://github.com/SeaSBee/go-cachex/actions)

## 🚀 Features

- **High Performance**: Async I/O with non-blocking operations returning channels
- **Multiple Backends**: Memory, Redis, Ristretto, and Layered store support
- **Thread-Safe**: Comprehensive concurrency controls with race-free design
- **Type-Safe**: Generic API with compile-time type safety
- **Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging
- **Caching Patterns**: Read-Through, Write-Through, Write-Behind, Refresh-Ahead
- **Production Ready**: Circuit breakers, retries, distributed locks
- **Flexible Configuration**: YAML config files and environment variables

## 📦 Installation

```bash
go get github.com/SeaSBee/go-cachex
```

## 🎯 Quick Start

### Configuration Files

The library supports YAML configuration files for easy setup. Two configuration files are provided:

- **`cachex-config-reference.yaml`**: Complete reference with all available options
- **`cachex-config-example.yaml`**: Practical example you can copy and modify

```go
// Load configuration from YAML file
cache, err := cachex.NewFromConfig[User]("cachex-config-example.yaml")
if err != nil {
    panic(err)
}
defer cache.Close()
```

### Basic Usage with Memory Store

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/SeaSBee/go-cachex"
)

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Create memory store
    store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Create cache instance
    cache, err := cachex.New[User](
        cachex.WithStore(store),
        cachex.WithDefaultTTL(5*time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    ctx := context.Background()
    user := User{ID: "123", Name: "John Doe", Email: "john@example.com"}

    // Set value (async operation)
    setResult := <-cache.Set(ctx, "user:123", user, 10*time.Minute)
    if setResult.Error != nil {
        panic(setResult.Error)
    }

    // Get value (async operation)
    getResult := <-cache.Get(ctx, "user:123")
    if getResult.Error != nil {
        panic(getResult.Error)
    }
    
    if getResult.Found {
        fmt.Printf("Found user: %+v\n", getResult.Value)
    } else {
        fmt.Println("User not found")
    }
}
```

### Redis Store Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/SeaSBee/go-cachex"
)

func main() {
    // Create Redis store
    redisConfig := cachex.DefaultRedisConfig()
    redisConfig.Addr = "localhost:6379"
    
    store, err := cachex.NewRedisStore(redisConfig)
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Create cache with JSON codec
    cache, err := cachex.New[map[string]interface{}](
        cachex.WithStore(store),
        cachex.WithCodec(cachex.NewJSONCodec()),
        cachex.WithDefaultTTL(10*time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    ctx := context.Background()
    data := map[string]interface{}{
        "id":   "456",
        "name": "Jane Smith",
        "age":  30,
    }

    // Async operations
    setResult := <-cache.Set(ctx, "user:456", data, 15*time.Minute)
    if setResult.Error != nil {
        panic(setResult.Error)
    }

    getResult := <-cache.Get(ctx, "user:456")
    if getResult.Error != nil {
        panic(getResult.Error)
    }

    if getResult.Found {
        fmt.Printf("Retrieved: %+v\n", getResult.Value)
    }
}
```

### Configuration-Based Setup

```go
package main

import (
    "context"
    "fmt"

    "github.com/SeaSBee/go-cachex"
)

func main() {
    // Load from YAML configuration
    config, err := cachex.LoadConfig("config.yaml")
    if err != nil {
        panic(err)
    }

    // Create cache from configuration
    cache, err := cachex.NewFromConfig[string](config)
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    ctx := context.Background()
    
    // Use the configured cache
    setResult := <-cache.Set(ctx, "key1", "value1", 0) // Uses default TTL from config
    if setResult.Error != nil {
        panic(setResult.Error)
    }

    getResult := <-cache.Get(ctx, "key1")
    if getResult.Error != nil {
        panic(getResult.Error)
    }

    if getResult.Found {
        fmt.Printf("Value: %s\n", getResult.Value)
    }
}
```

## 🏗️ Architecture

### Core Interface (Async API)

```go
type Cache[T any] interface {
    // Context management
    WithContext(ctx context.Context) Cache[T]

    // Basic operations (all non-blocking, return channels)
    Get(ctx context.Context, key string) <-chan AsyncCacheResult[T]
    Set(ctx context.Context, key string, val T, ttl time.Duration) <-chan AsyncCacheResult[T]
    MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T]
    MSet(ctx context.Context, items map[string]T, ttl time.Duration) <-chan AsyncCacheResult[T]
    Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T]
    Exists(ctx context.Context, key string) <-chan AsyncCacheResult[T]
    TTL(ctx context.Context, key string) <-chan AsyncCacheResult[T]
    IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[T]

    // Caching patterns
    ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T]
    WriteThrough(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T]
    WriteBehind(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T]
    RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T]

    // Tagging
    AddTags(ctx context.Context, key string, tags ...string) <-chan AsyncCacheResult[T]
    InvalidateByTag(ctx context.Context, tags ...string) <-chan AsyncCacheResult[T]

    // Distributed locks
    TryLock(ctx context.Context, key string, ttl time.Duration) <-chan AsyncLockResult

    // Statistics and lifecycle
    GetStats() map[string]any
    Close() error
}
```

### Available Store Types

#### 1. Memory Store (In-Memory)
```go
// Default configuration
store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())

// Custom configuration
config := &cachex.MemoryConfig{
    MaxSize:         10000,
    MaxMemoryMB:     100,
    DefaultTTL:      5 * time.Minute,
    CleanupInterval: 1 * time.Minute,
    EvictionPolicy:  cachex.EvictionPolicyLRU, // LRU, LFU, TTL
    EnableStats:     true,
}
store, err := cachex.NewMemoryStore(config)
```

#### 2. Redis Store (Distributed)
```go
// Default configuration
store, err := cachex.NewRedisStore(cachex.DefaultRedisConfig())

// Custom configuration
config := &cachex.RedisConfig{
    Addr:             "localhost:6379",
    Password:         "",
    DB:               0,
    PoolSize:         10,
    MinIdleConns:     5,
    MaxRetries:       3,
    DialTimeout:      5 * time.Second,
    ReadTimeout:      3 * time.Second,
    WriteTimeout:     3 * time.Second,
    EnablePipelining: true,
    EnableMetrics:    true,
}
store, err := cachex.NewRedisStore(config)
```

#### 3. Ristretto Store (High-Performance In-Memory)
```go
// Default configuration
store, err := cachex.NewRistrettoStore(cachex.DefaultRistrettoConfig())

// Custom configuration
config := &cachex.RistrettoConfig{
    MaxItems:       10000,
    MaxMemoryBytes: 512 * 1024 * 1024, // 512MB
    DefaultTTL:     30 * time.Minute,
    NumCounters:    100000,
    BufferItems:    64,
    EnableMetrics:  true,
}
store, err := cachex.NewRistrettoStore(config)
```

#### 4. Layered Store (Multi-Level Caching)
```go
// Create L2 store (typically Redis)
l2Store, err := cachex.NewRedisStore(cachex.DefaultRedisConfig())

// Create layered store
config := cachex.DefaultLayeredConfig()
config.WritePolicy = cachex.WritePolicyThrough
config.ReadPolicy = cachex.ReadPolicyThrough

layeredStore, err := cachex.NewLayeredStore(l2Store, config)
```

## ⚙️ Configuration

### YAML Configuration Example

```yaml
# config.yaml
memory:
  max_size: 10000
  max_memory_mb: 100
  default_ttl: 5m
  cleanup_interval: 1m
  eviction_policy: "lru"
  enable_stats: true

# OR use Redis (only one store type allowed)
# redis:
#   addr: "localhost:6379"
#   password: ""
#   db: 0
#   pool_size: 10
#   dial_timeout: 5s

default_ttl: 10m
max_retries: 3
retry_delay: 100ms
codec: "json"  # or "msgpack"

observability:
  enable_metrics: true
  enable_tracing: true
  enable_logging: true
```

### Environment Variables

```bash
# Store configuration
CACHEX_STORE_TYPE=memory  # memory, redis, ristretto
CACHEX_REDIS_ADDR=localhost:6379
CACHEX_REDIS_PASSWORD=your_password
CACHEX_REDIS_DB=0

# Cache settings
CACHEX_DEFAULT_TTL=5m
CACHEX_MAX_RETRIES=3
CACHEX_CODEC=json

# Observability
CACHEX_ENABLE_METRICS=true
CACHEX_ENABLE_TRACING=true
CACHEX_ENABLE_LOGGING=true
```

## 🔄 Caching Patterns

### 1. Cache-Aside (Manual)
```go
ctx := context.Background()

// Try to get from cache first
getResult := <-cache.Get(ctx, "user:123")
if getResult.Error != nil {
    // Handle error
    return
}

var user User
if getResult.Found {
    user = getResult.Value
} else {
    // Load from database
    user = loadUserFromDB("123")
    
    // Store in cache
    setResult := <-cache.Set(ctx, "user:123", user, 10*time.Minute)
    if setResult.Error != nil {
        // Handle cache set error (optional)
        log.Printf("Failed to cache user: %v", setResult.Error)
    }
}
```

### 2. Read-Through (Automatic Loading)
```go
ctx := context.Background()

// Automatically loads from source if not in cache
result := <-cache.ReadThrough(ctx, "user:123", 10*time.Minute, func(ctx context.Context) (User, error) {
    return loadUserFromDB("123")
})

if result.Error != nil {
    // Handle error
    return
}

user := result.Value
```

### 3. Write-Behind (Async Persistence)
```go
ctx := context.Background()

// Cache immediately, persist asynchronously
result := <-cache.WriteBehind(ctx, "user:123", user, 10*time.Minute, func(ctx context.Context) error {
    return saveUserToDB(user)
})

if result.Error != nil {
    // Handle error
    return
}
```

### 4. Refresh-Ahead (Proactive Refresh)
```go
ctx := context.Background()

// Refresh cache before expiration
result := <-cache.RefreshAhead(ctx, "user:123", 1*time.Minute, func(ctx context.Context) (User, error) {
    return loadUserFromDB("123")
})

if result.Error != nil {
    // Handle error
    return
}
```

## 🔒 Advanced Features

### Batch Operations
```go
ctx := context.Background()

// Batch set
items := map[string]User{
    "user:1": {ID: "1", Name: "Alice"},
    "user:2": {ID: "2", Name: "Bob"},
}
msetResult := <-cache.MSet(ctx, items, 10*time.Minute)

// Batch get
keys := []string{"user:1", "user:2"}
mgetResult := <-cache.MGet(ctx, keys...)
if mgetResult.Error == nil {
    for key, user := range mgetResult.Values {
        fmt.Printf("%s: %+v\n", key, user)
    }
}
```

### Distributed Locking
```go
ctx := context.Background()

// Try to acquire distributed lock
lockResult := <-cache.TryLock(ctx, "lock:critical-section", 30*time.Second)
if lockResult.Error != nil {
    // Handle error
    return
}

if lockResult.Acquired {
    defer func() {
        if err := lockResult.Unlock(); err != nil {
            log.Printf("Failed to unlock: %v", err)
        }
    }()
    
    // Critical section code here
    fmt.Println("Lock acquired, executing critical section")
} else {
    fmt.Println("Failed to acquire lock")
}
```

### Tag-Based Invalidation
```go
ctx := context.Background()

// Add tags to cached items
tagResult := <-cache.AddTags(ctx, "user:123", "users", "active", "premium")
if tagResult.Error != nil {
    // Handle error
}

// Invalidate all items with specific tags
invalidateResult := <-cache.InvalidateByTag(ctx, "premium")
if invalidateResult.Error != nil {
    // Handle error
}
```

## 📊 Observability

### Built-in Metrics
- `cache_operations_total{operation, status}` - Total cache operations
- `cache_operation_duration_seconds{operation}` - Operation latencies
- `cache_hit_ratio` - Cache hit ratio
- `cache_memory_usage_bytes` - Memory usage (memory stores)
- `cache_evictions_total` - Number of evictions

### Structured Logging
```go
// Enable logging in cache configuration
cache, err := cachex.New[string](
    cachex.WithStore(store),
    cachex.WithObservability(cachex.ObservabilityConfig{
        EnableLogging: true,
        EnableMetrics: true,
        EnableTracing: true,
    }),
)
```

### OpenTelemetry Tracing
Automatic span creation for all cache operations with attributes:
- `cache.operation` - Operation type (get, set, etc.)
- `cache.key` - Cache key (redacted if configured)
- `cache.hit` - Whether it was a cache hit
- `cache.ttl` - TTL value
- `cache.size` - Value size in bytes

## 🧪 Testing

```bash
# Run all tests
go test ./... -race -v

# Run integration tests
go test ./tests/integration/ -v

# Run benchmarks
go test -bench=. ./tests/benchmark/

# Run with Redis (requires Redis running)
docker run -d -p 6379:6379 redis:alpine
go test ./tests/integration/ -v

# Linting
golangci-lint run

# Security checks
govulncheck ./...
```

## 📊 Performance & Test Results

### 🚀 Performance Benchmarks

The library is designed for high performance with async operations and efficient memory usage:

**Memory Store Performance (Apple M3 Pro, 12 cores):**
- **Set Operations**: ~1.5μs per operation (784,498 ops/sec)
- **Get Operations**: ~3.2μs per operation (378,142 ops/sec)
- **Batch Set (MSet)**: ~9.2μs per operation (239,446 ops/sec)
- **Batch Get (MGet)**: ~11.3μs per operation (105,052 ops/sec)

**Memory Usage:**
- Set: 1,427 B/op (21 allocations)
- Get: 2,713 B/op (32 allocations)
- MSet: 6,796 B/op (68 allocations)
- MGet: 7,582 B/op (107 allocations)

### 🧪 Test Coverage & Quality

**Comprehensive Test Suite:**
- **Total Tests**: 400+ tests across unit and integration suites
- **Test Results**: 396 PASS, 4 SKIP (Redis-dependent tests)
- **Race Condition Testing**: All tests run with `-race` flag
- **Concurrency Testing**: Extensive concurrent access testing

**Test Categories:**
- **Unit Tests**: 396 tests covering all components
- **Integration Tests**: End-to-end functionality testing
- **Benchmark Tests**: Performance measurement and regression testing
- **Security Tests**: Validation, redaction, and authorization testing

**Test Coverage Areas:**
- ✅ Core cache operations (Get, Set, MGet, MSet, Del, Exists, TTL, IncrBy)
- ✅ Store implementations (Memory, Redis, Ristretto, Layered)
- ✅ Codec implementations (JSON, MessagePack)
- ✅ Security features (Validation, Redaction, Authorization)
- ✅ Observability (Metrics, Tracing, Logging)
- ✅ Tagging and invalidation
- ✅ GORM integration
- ✅ Refresh-ahead scheduling
- ✅ Error handling and edge cases
- ✅ Concurrency and race conditions

### 🔒 Security Features

**Built-in Security Components:**
- **Key Validation**: Pattern-based key validation with allow/block lists
- **Value Validation**: Size limits and content validation
- **Data Redaction**: Sensitive data masking in logs and traces
- **RBAC Authorization**: Role-based access control for cache operations
- **Secrets Management**: Secure secret retrieval and management
- **TLS Support**: Encrypted connections for Redis and other stores

**Security Best Practices:**
- Input validation and sanitization
- Secure key generation with HMAC
- Environment isolation
- Audit logging and monitoring
- Principle of least privilege
```

## 📚 Examples

Complete examples are available in the `examples/` directory:

- **[Single Service](examples/single-service/)** - Basic REST API integration
- **[Memory Store](examples/local-memory/)** - In-memory caching configurations  
- **[Redis Store](examples/redis/)** - Redis backend setup and usage
- **[Ristretto Store](examples/ristretto/)** - High-performance in-memory caching
- **[Configuration](examples/config-examples/)** - YAML and environment-based config
- **[GORM Integration](examples/gorm-integration/)** - Database integration patterns
- **[MessagePack](examples/msgpack/)** - MessagePack serialization
- **[Observability](examples/observability/)** - Metrics, tracing, and logging

## ⚙️ Configuration

### Configuration Options

The library provides extensive configuration options through YAML files and environment variables:

#### Store Types
- **Memory Store**: In-memory caching with configurable size limits and eviction policies
- **Redis Store**: Distributed caching with connection pooling and TLS support
- **Ristretto Store**: High-performance in-memory caching with automatic eviction
- **Layered Store**: Multi-tier caching combining memory and persistent stores

#### Security Features
- **Input Validation**: Key pattern validation, value size limits, content filtering
- **Data Redaction**: Automatic masking of sensitive data in logs and traces
- **RBAC Authorization**: Role-based access control for cache operations
- **Secrets Management**: Secure retrieval of configuration secrets

#### Observability
- **Prometheus Metrics**: Cache hit/miss ratios, operation latencies, error rates
- **OpenTelemetry Tracing**: Distributed tracing for cache operations
- **Structured Logging**: Detailed operation logs with correlation IDs

#### Advanced Features
- **Tagging**: Logical grouping and bulk invalidation of cached items
- **Refresh-Ahead**: Automatic cache refresh before expiration
- **GORM Integration**: Automatic cache invalidation for database operations

### Environment Variables

All configuration options can be overridden using environment variables:

```bash
# Store type and basic settings
export CACHEX_STORE_TYPE=redis
export CACHEX_DEFAULT_TTL=10m
export CACHEX_CODEC=json

# Redis configuration
export CACHEX_REDIS_ADDR=localhost:6379
export CACHEX_REDIS_PASSWORD=your-password
export CACHEX_REDIS_POOL_SIZE=20

# Security settings
export CACHEX_SECURITY_MAX_KEY_LENGTH=256
export CACHEX_SECURITY_MAX_VALUE_SIZE=1048576

# Observability
export CACHEX_OBSERVABILITY_ENABLE_METRICS=true
export CACHEX_OBSERVABILITY_SERVICE_NAME=my-app
```

## 🚀 Best Practices

### 1. Choose the Right Store
- **Memory Store**: Ultra-low latency, single-node applications
- **Redis Store**: Distributed applications, data persistence
- **Ristretto Store**: High-performance, memory-efficient caching
- **Layered Store**: Multi-level caching for optimal performance

### 2. Handle Async Operations
```go
// Always handle errors from async operations
result := <-cache.Get(ctx, key)
if result.Error != nil {
    // Log error and fallback to source
    log.Printf("Cache error: %v", result.Error)
    return loadFromSource(key)
}

if !result.Found {
    // Cache miss - load from source
    return loadFromSource(key)
}

return result.Value
```

### 3. Use Context for Cancellation
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result := <-cache.Get(ctx, key)
// Operation will respect context timeout
```

### 4. Configure Appropriate TTLs
```go
// Short TTL for frequently changing data
userProfile := <-cache.Set(ctx, "profile:123", profile, 5*time.Minute)

// Longer TTL for static data
config := <-cache.Set(ctx, "config:app", appConfig, 1*time.Hour)

// No TTL for permanent data (use 0)
constants := <-cache.Set(ctx, "constants", data, 0)
```

## ⚠️ Important Notes

- **Async API**: All operations return channels and are non-blocking
- **Context**: Always pass context for cancellation and timeouts
- **Error Handling**: Check `AsyncCacheResult.Error` for operation errors
- **Resource Management**: Always call `Close()` on caches and stores
- **Type Safety**: Use generic types for compile-time safety

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 [Documentation](https://pkg.go.dev/github.com/SeaSBee/go-cachex)
- 🐛 [Issues](https://github.com/SeaSBee/go-cachex/issues)
- 💬 [Discussions](https://github.com/SeaSBee/go-cachex/discussions)