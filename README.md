# Go-CacheX 🚀

A high-performance, feature-rich caching library for Go applications with support for multiple cache backends, advanced optimizations, and comprehensive observability.

[![Go Report Card](https://goreportcard.com/badge/github.com/SeaSBee/go-cachex)](https://goreportcard.com/report/github.com/SeaSBee/go-cachex)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](https://github.com/SeaSBee/go-cachex)

## 📊 **Quality Metrics**

- ✅ **100% Test Pass Rate** (914 tests)
- 🚀 **Sub-millisecond Latency** for in-memory operations
- 💾 **81% Performance Improvement** with memory pools
- 🔧 **Production-Ready** with comprehensive observability
- 🛡️ **Thread-Safe** with optimized concurrency

## 🏗️ **Architecture Overview**

Go-CacheX is built with a modular, extensible architecture that supports multiple cache backends and advanced features:

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
├─────────────────────────────────────────────────────────────┤
│                    Cache Interface                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │   Memory    │ │  Ristretto  │ │    Redis    │           │
│  │    Store    │ │    Store    │ │    Store    │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │   Layered   │ │ Optimized   │ │ Connection  │           │
│  │   Cache     │ │   Tagging   │ │   Pooling   │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │ MessagePack │ │     JSON    │ │Observability│           │
│  │   Codec     │ │   Codec     │ │   Manager   │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## ✨ **Key Features**

### 🚀 **High Performance**
- **Sub-millisecond latency** for in-memory operations
- **Memory pools** for 81% performance improvement
- **Allocation optimization** for 33-41% reduction in allocations
- **Optimized serialization** with MessagePack and JSON support

### 🔧 **Multiple Cache Backends**
- **Memory Store**: Fast in-memory caching with LRU/LFU/TTL eviction
- **Ristretto Store**: High-performance concurrent cache
- **Redis Store**: Distributed caching with connection pooling
- **Layered Cache**: Multi-tier caching strategies

### 🏷️ **Advanced Tagging System**
- **Optimized Tag Manager**: Three-tier processing for high throughput
- **Batch operations**: Efficient bulk tag management
- **Memory optimization**: Configurable memory limits
- **Persistence**: Tag mapping persistence and recovery

### 🔍 **Comprehensive Observability**
- **Metrics collection**: Operation counts, latencies, error rates
- **Distributed tracing**: OpenTelemetry integration
- **Structured logging**: Configurable log levels and formats
- **Performance monitoring**: Real-time performance insights

### 🛡️ **Production Features**
- **Connection pooling**: Optimized Redis connection management
- **Error handling**: Robust error recovery and reporting
- **Context support**: Full context cancellation support
- **Graceful shutdown**: Clean resource cleanup

## 📦 **Installation**

```bash
go get github.com/SeaSBee/go-cachex
```

## 🚀 **Quick Start**

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/SeaSBee/go-cachex"
)

func main() {
    // Create a memory store
    store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
        MaxSize:         10000,
        MaxMemoryMB:     100,
        DefaultTTL:      5 * time.Minute,
        EvictionPolicy:  cachex.EvictionPolicyLRU,
        EnableStats:     true,
    })
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Create a cache with JSON serialization
    cache, err := cachex.New[User](cachex.WithStore(store))
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    ctx := context.Background()
    user := User{ID: 1, Name: "John Doe", Email: "john@example.com"}

    // Set a value
    result := <-cache.Set(ctx, "user:1", user, 10*time.Minute)
    if result.Error != nil {
        panic(result.Error)
    }

    // Get a value
    result = <-cache.Get(ctx, "user:1")
    if result.Error != nil {
        panic(result.Error)
    }
    
    fmt.Printf("User: %+v\n", result.Value)
}

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### Advanced Configuration

```go
// Create a layered cache with Redis and Memory
redisStore, _ := cachex.NewRedisStore(&cachex.RedisConfig{
    Addr:     "localhost:6379",
    PoolSize: 10,
})

memoryStore, _ := cachex.NewMemoryStore(&cachex.MemoryConfig{
    MaxSize: 1000,
})

layeredStore, _ := cachex.NewLayeredStore(&cachex.LayeredConfig{
    L1: memoryStore, // Fast local cache
    L2: redisStore,  // Distributed cache
    Strategy: cachex.StrategyReadThrough,
})

// Create cache with observability
observabilityConfig := &cachex.ObservabilityConfig{
    EnableMetrics: true,
    EnableTracing: true,
    EnableLogging: true,
    LogLevel:      "info",
}

cache, _ := cachex.New[User](
    cachex.WithStore(layeredStore),
    cachex.WithObservability(*observabilityConfig),
    cachex.WithCodec(cachex.NewMessagePackCodec()),
)
```

### Tagging System

```go
// Create optimized tag manager
tagConfig := &cachex.OptimizedTagConfig{
    EnablePersistence:           true,
    BatchSize:                   100,
    EnableMemoryOptimization:    true,
    MaxMemoryUsage:              100 * 1024 * 1024, // 100MB
    BatchFlushInterval:          1 * time.Second,
}

tagManager, _ := cachex.NewOptimizedTagManager(store, tagConfig)
defer tagManager.Close()

ctx := context.Background()

// Add tags to keys
err := tagManager.AddTags(ctx, "user:1", "admin", "premium", "verified")
err = tagManager.AddTags(ctx, "user:2", "user", "verified")

// Get keys by tag
keys, err := tagManager.GetKeysByTag(ctx, "verified")
// Returns: ["user:1", "user:2"]

// Invalidate by tag
err = tagManager.InvalidateByTag(ctx, "admin")
// Removes all keys tagged with "admin"
```

## 📈 **Performance Benchmarks**

### Core Operations (Apple M3 Pro)

| Operation | Memory Store | Ristretto Store | Redis Store |
|-----------|-------------|-----------------|-------------|
| **Set**   | 1,830 ns/op | 2,159 ns/op     | 9,721 ns/op |
| **Get**   | 3,271 ns/op | 961.9 ns/op     | 9,400 ns/op |

### Serialization Performance

| Codec     | Encode      | Decode      | Memory Usage |
|-----------|-------------|-------------|--------------|
| **JSON**  | 42.33 ns/op | 178.7 ns/op | 150 B/op     |
| **MsgPack**| 50.43 ns/op | 66.15 ns/op | 176 B/op     |

### Optimization Benefits

- **Memory Pools**: 81% performance improvement for map operations
- **Allocation Optimization**: 33-41% reduction in allocations
- **Connection Pooling**: 2% performance improvement for Redis operations

## 🔧 **Configuration Examples**

### Memory Store Configuration

```yaml
memory:
  max_size: 10000
  max_memory_mb: 100
  default_ttl: 5m
  cleanup_interval: 1m
  eviction_policy: lru
  enable_stats: true
```

### Redis Store Configuration

```yaml
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10
  min_idle_conns: 5
  max_retries: 3
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  tls:
    enabled: false
```

### Layered Cache Configuration

```yaml
layered:
  strategy: read_through
  sync_interval: 30s
  concurrency_limit: 100
  timeout: 5s
  enable_stats: true
```

## 🧪 **Testing**

### Run All Tests

```bash
# Run unit tests
go test ./tests/unit/ -v

# Run integration tests
go test ./tests/integration/ -v

# Run benchmarks
go test ./tests/benchmark/ -bench=. -benchmem
```

### Test Results

- **Unit Tests**: 842 tests (100% pass rate)
- **Integration Tests**: 72 tests (100% pass rate)
- **Benchmarks**: Comprehensive performance testing
- **Coverage**: Complete edge case coverage

## 📚 **Documentation**

### Core Concepts

- [Cache Stores](./docs/stores.md) - Memory, Redis, Ristretto implementations
- [Layered Caching](./docs/layered.md) - Multi-tier caching strategies
- [Tagging System](./docs/tagging.md) - Advanced tag management
- [Observability](./docs/observability.md) - Metrics, tracing, and logging
- [Configuration](./docs/configuration.md) - YAML and programmatic configuration

### Optimization Guides

- [Memory Optimization](./MEMORY_POOL_OPTIMIZATION.md) - Memory pool benefits
- [Allocation Optimization](./ALLOCATION_OPTIMIZATION.md) - Performance improvements
- [Connection Pooling](./CONNECTION_POOLING_OPTIMIZATION.md) - Redis optimization
- [Tagging Optimization](./TAGGING_OPTIMIZATION.md) - High-throughput tagging

### Performance Reports

- [Comprehensive Test Report](./COMPREHENSIVE_TEST_REPORT.md) - Complete test analysis
- [Benchmark Results](./tests/benchmark/README.md) - Performance benchmarks

## 🤝 **Contributing**

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/SeaSBee/go-cachex.git
cd go-cachex

# Install dependencies
go mod download

# Run tests
go test ./...

# Run benchmarks
go test ./tests/benchmark/ -bench=. -benchmem
```

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 **Acknowledgments**

- [Ristretto](https://github.com/dgraph-io/ristretto) - High-performance concurrent cache
- [Redis](https://redis.io/) - Distributed cache backend
- [MessagePack](https://msgpack.org/) - Efficient serialization format
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework

## 📞 **Support**

- 📧 **Email**: support@seasbee.com
- 🐛 **Issues**: [GitHub Issues](https://github.com/SeaSBee/go-cachex/issues)
- 📖 **Documentation**: [GitHub Wiki](https://github.com/SeaSBee/go-cachex/wiki)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/SeaSBee/go-cachex/discussions)

---

**Made with ❤️ by the SeaSBee Team**