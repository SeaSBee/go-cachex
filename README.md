# Go-CacheX

A production-grade, highly concurrent, thread-safe cache library built on top of go-redis for multi-microservice environments.

[![Go Report Card](https://goreportcard.com/badge/github.com/SeaSBee/go-cachex)](https://goreportcard.com/report/github.com/SeaSBee/go-cachex)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SeaSBee/go-cachex)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://github.com/SeaSBee/go-cachex/workflows/Tests/badge.svg)](https://github.com/SeaSBee/go-cachex/actions)

## 🚀 Features

- **High Performance**: Async Redis I/O with pipelining and batching
- **Thread-Safe**: Race-free design with comprehensive concurrency controls
- **Security**: TLS, ACL support, encryption-at-rest, input validation
- **Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging via go-logx
- **Caching Patterns**: Cache-Aside, Read-Through, Write-Through, Write-Behind, Refresh-Ahead
- **GORM Integration**: First-class GORM plugin with automatic cache invalidation
- **Distributed**: Pub/Sub invalidation, distributed locks, multi-service coherence
- **Production Ready**: Circuit breakers, retries, rate limiting, bulkhead isolation

## 📦 Installation

```bash
go get github.com/SeaSBee/go-cachex
```

## 🎯 Quick Start

### Basic Usage

```go
package main

import (
    "time"
    
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
    // Create Redis store
    redisStore, err := redisstore.New(&redisstore.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    if err != nil {
        panic(err)
    }

    // Create cache instance
    c, err := cache.New[User](
        cache.WithStore(redisStore),
        cache.WithDefaultTTL(5 * time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // Basic operations
    user := &User{ID: "123", Name: "John Doe"}
    err = c.Set("user:123", user, 10*time.Minute)
    if err != nil {
        panic(err)
    }

    cachedUser, found, err := c.Get("user:123")
    if err != nil {
        panic(err)
    }
    if found {
        println("Found user:", cachedUser.Name)
    }
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### GORM Integration

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
    })

    // Register models
    plugin.RegisterModelWithDefaults(&User{}, 10*time.Minute, "users", "auth")

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
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

## 🏗️ Architecture

### Core Interfaces

```go
type Cache[T any] interface {
    // Basic operations
    Get(key string) (T, bool, error)
    Set(key string, val T, ttl time.Duration) error
    MGet(keys ...string) (map[string]T, error)
    MSet(items map[string]T, ttl time.Duration) error
    Del(keys ...string) error
    
    // Caching patterns
    ReadThrough(key string, ttl time.Duration, loader func() (T, error)) (T, error)
    WriteBehind(key string, val T, ttl time.Duration, writer func() error) error
    RefreshAhead(key string, refreshBefore time.Duration, loader func() (T, error)) error
    
    // Distributed features
    TryLock(key string, ttl time.Duration) (unlock func() error, ok bool, err error)
    PublishInvalidation(keys ...string) error
    SubscribeInvalidations(handler func(keys ...string)) (close func() error, err error)
    
    // Tagging
    AddTags(key string, tags ...string) error
    InvalidateByTag(tags ...string) error
    
    // Close
    Close() error
}
```

### Key Generation Scheme

- **Entity Keys**: `{prefix}:{namespace}:{entity}:{id}`
- **List Keys**: `{prefix}:{namespace}:list:{entity}:{hash(filters)}`
- **Composite Keys**: `{prefix}:{namespace}:{entityA}:{idA}:{entityB}:{idB}`
- **Session Keys**: `{prefix}:{namespace}:session:{sid}`

## ⚙️ Configuration

### Environment Variables

```bash
# Redis Configuration
CACHEX_REDIS_ADDR=localhost:6379
CACHEX_REDIS_PASSWORD=your_password
CACHEX_REDIS_DB=0
CACHEX_REDIS_TLS_ENABLED=true

# Cache Configuration
CACHEX_DEFAULT_TTL=5m
CACHEX_MAX_RETRIES=3
CACHEX_CIRCUIT_BREAKER_THRESHOLD=5

# Security
CACHEX_ENCRYPTION_KEY=your_encryption_key
CACHEX_REDACT_LOGS=true

# Observability
CACHEX_ENABLE_METRICS=true
CACHEX_ENABLE_TRACING=true
CACHEX_ENABLE_LOGGING=true
```

### YAML Configuration

```yaml
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  tls:
    enabled: true
    insecure_skip_verify: false

cache:
  default_ttl: "5m"
  max_retries: 3
  circuit_breaker:
    threshold: 5
    timeout: "30s"

security:
  encryption_key: "your_encryption_key"
  redact_logs: true
  rate_limit:
    requests_per_second: 1000

observability:
  enable_metrics: true
  enable_tracing: true
  enable_logging: true
  service_name: "cachex"
  service_version: "1.0.0"
  environment: "production"
```

## 🔄 Caching Patterns

### 1. Cache-Aside (Default)

```go
// Manual cache management
user, found, err := cache.Get("user:123")
if !found {
    user = loadUserFromDB("123")
    cache.Set("user:123", user, 10*time.Minute)
}
```

### 2. Read-Through

```go
// Automatic loading from source
user, err := cache.ReadThrough("user:123", 10*time.Minute, func() (User, error) {
    return loadUserFromDB("123")
})
```

### 3. Write-Behind

```go
// Asynchronous write to source
err := cache.WriteBehind("user:123", user, 10*time.Minute, func() error {
    return saveUserToDB(user)
})
```

### 4. Refresh-Ahead

```go
// Proactive refresh before expiry
err := cache.RefreshAhead("user:123", 1*time.Minute, func() (User, error) {
    return loadUserFromDB("123")
})
```

### 5. Local In-Memory Cache

```go
// Pure memory store for ultra-low latency
memoryStore, err := memorystore.New(&memorystore.Config{
    DefaultTTL:      5 * time.Minute,
    CleanupInterval: 1 * time.Minute,
})
c, err := cache.New[User](cache.WithStore(memoryStore))

// Ristretto store for high-performance caching
ristrettoStore, err := ristretto.New(ristretto.DefaultConfig())
c, err := cache.New[User](cache.WithStore(ristrettoStore))
```

## 📊 Observability

### Metrics (Prometheus)

```go
// Available metrics
cache_operations_total{operation="get", status="hit"}
cache_operations_total{operation="get", status="miss"}
cache_operation_duration_seconds{operation="get"}
cache_bytes_in_total
cache_bytes_out_total
cache_errors_total{type="timeout"}
circuit_breaker_state{state="closed"}
worker_pool_active_workers
dlq_failed_operations
```

### Tracing (OpenTelemetry)

```go
// Automatic span creation for all operations
// Span attributes: operation, key, ttl, bytes, attempt, duration
```

### Logging (go-logx)

```go
// Structured logging with fields
// component=cachex, operation=get, key=user:123, ttl_ms=600000, duration_ms=5, attempt=1
```

## 🔒 Security Features

- **TLS Encryption**: Secure Redis connections
- **ACL Support**: Username/password authentication
- **Encryption-at-Rest**: AES-GCM encryption for cached values
- **Input Validation**: Safe deserialization and validation
- **Rate Limiting**: Token bucket rate limiting
- **Log Redaction**: Sensitive data masking in logs when Security.RedactLogs=true

## 📚 Examples

See the `cachex/examples/` directory for complete working examples:

- `examples/single-service/`: REST API with cache integration
- `examples/local-memory/`: Local in-memory cache with different configurations
- `examples/multi-service-pubsub/`: Cross-service cache invalidation
- `examples/gorm-integration/`: GORM plugin usage with automatic C/U/D invalidation
- `examples/observability/`: OpenTelemetry tracing, Prometheus metrics, structured logging
- `examples/logging/`: Structured logging with go-logx and log redaction
- `examples/worker-pool/`: Worker pool with backpressure and batch operations
- `examples/pipeline/`: Redis pipelining and batching
- `examples/security/`: Encryption, rate limiting, and security features
- `examples/circuit-breaker-retry/`: Circuit breaker and retry mechanisms
- `examples/bulkhead/`: Bulkhead isolation patterns
- `examples/msgpack/`: MessagePack serialization
- `examples/ristretto/`: High-performance in-memory caching
- `examples/feature-flags/`: Dynamic feature toggles
- `examples/configuration/`: YAML and environment-based configuration

## 🧪 Testing

```bash
# Run all tests with race detection
go test ./... -race

# Run specific package tests
go test ./cachex/pkg/cache -v
go test ./cachex/pkg/gormx -v

# Run benchmarks
go test -bench=. ./cachex/pkg/cache/

# Run linting
golangci-lint run

# Run security checks
govulncheck ./...
```

## 🚀 Deployment

### Docker Compose

```bash
# Start Redis, demo app, Prometheus, and Grafana
cd cachex/deployments
docker-compose up -d
```

### Kubernetes

```bash
# Deploy to Kubernetes (Helm charts available)
kubectl apply -f cachex/deployments/k8s/
```

## 📖 Documentation

- [Configuration Guide](cachex/docs/CONFIGURATION.md)
- [Observability Features](cachex/docs/OBSERVABILITY.md)
- [Security Features](cachex/docs/SECURITY_FEATURES.md)
- [GORM Integration](cachex/docs/IMPLEMENTATION_DETAILS.md)
- [Worker Pool](cachex/docs/WORKER_POOL.md)
- [Circuit Breaker & Retry](cachex/docs/CIRCUIT_BREAKER_RETRY.md)
- [Local Memory Cache](cachex/docs/LOCAL_MEMORY_CACHE.md)
- [MessagePack Serialization](cachex/docs/MESSAGEPACK_SERIALIZATION.md)
- [Bulkhead Isolation](cachex/docs/BULKHEAD_ISOLATION.md)

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Limitations

- **Redlock Caveats**: Distributed locks are not 100% safe in all failure scenarios
- **Memory Usage**: Large cached objects may impact memory usage
- **Network Latency**: Cache performance depends on Redis network latency
- **Data Consistency**: Eventual consistency in distributed scenarios

## 🆘 Support

- 📖 [Documentation](https://pkg.go.dev/github.com/SeaSBee/go-cachex)
- 🐛 [Issues](https://github.com/SeaSBee/go-cachex/issues)
- 💬 [Discussions](https://github.com/SeaSBee/go-cachex/discussions)

## ✅ Status

- ✅ **Thread Safety**: All race conditions fixed
- ✅ **GORM Integration**: Complete implementation with automatic C/U/D invalidation
- ✅ **Observability**: Full OpenTelemetry, Prometheus, and structured logging
- ✅ **Security**: TLS, encryption, rate limiting, log redaction
- ✅ **Production Ready**: Circuit breakers, retries, bulkhead isolation
- ✅ **Documentation**: Comprehensive examples and guides
- ✅ **Testing**: Race detection, linting, security checks