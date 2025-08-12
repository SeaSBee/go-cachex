# Resilience & Reliability Features

Go-CacheX provides comprehensive resilience and reliability features to ensure robust cache operations in production environments.

## 🔄 Resilience Overview

The library implements enterprise-grade resilience features including:

- **Circuit Breaker Pattern** - Automatic failure detection and recovery
- **Retry Mechanisms** - Configurable retry policies with exponential backoff
- **Timeout Management** - Per-operation and global timeout configuration
- **Dead Letter Queue (DLQ)** - Failed operation handling with retry mechanisms
- **Distributed Locks** - Redlock-style distributed locking for coordination
- **Pub/Sub Invalidation** - Multi-service cache coherence
- **Bloom Filter** - Probabilistic data structure to reduce backing store misses

## 🔌 Circuit Breaker Pattern

### Overview

The circuit breaker pattern prevents cascading failures by monitoring operation success rates and temporarily stopping operations when failure thresholds are exceeded.

### Configuration

```go
// Create cache with circuit breaker
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
        Threshold:   5,              // Number of failures before opening
        Timeout:     30 * time.Second, // Time to wait before half-open
        HalfOpenMax: 3,              // Max operations in half-open state
    }),
)
```

### States

1. **Closed**: Normal operation, all requests pass through
2. **Open**: Circuit is open, all requests fail fast
3. **Half-Open**: Limited requests allowed to test recovery

### Usage

```go
// Circuit breaker is automatically applied to all cache operations
user, found, err := c.Get("user:123")
if err != nil {
    // Error may be due to circuit breaker being open
    log.Printf("Cache operation failed: %v", err)
}

// Circuit breaker state is tracked in metrics
// Check metrics for circuit_breaker_state
```

## 🔄 Retry Mechanisms

### Exponential Backoff Retry

```go
// Create cache with retry policy
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithMaxRetries(3),
    cache.WithRetryDelay(100 * time.Millisecond),
)
```

### Retry Configuration

```go
type RetryConfig struct {
    MaxRetries    int           // Maximum retry attempts
    InitialDelay  time.Duration // Initial delay between retries
    MaxDelay      time.Duration // Maximum delay between retries
    Multiplier    float64       // Exponential backoff multiplier
    Jitter        bool          // Add jitter to prevent thundering herd
}
```

### Retry Behavior

- **Exponential Backoff**: Delay increases exponentially with each retry
- **Jitter**: Random variation in delay to prevent thundering herd
- **Max Delay Cap**: Prevents excessive delays
- **Context Support**: Respects context cancellation and timeouts

## ⏱️ Timeout Management

### Per-Operation Timeouts

```go
// Set operation with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := c.Set("user:123", user, 10*time.Minute)
if err != nil {
    log.Printf("Set operation failed: %v", err)
}
```

### Global Timeout Configuration

```go
// Create cache with timeout configuration
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithTimeout(5*time.Second), // Global timeout
)
```

### Timeout Features

- **Context Support**: Full context cancellation and timeout support
- **Per-Operation**: Individual operation timeouts
- **Global Defaults**: Default timeouts for all operations
- **Graceful Handling**: Proper timeout error handling

## 📬 Dead Letter Queue (DLQ)

### Overview

The Dead Letter Queue handles failed asynchronous operations (e.g., write-behind) with retry mechanisms and monitoring.

### Configuration

```go
// Create cache with DLQ
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithDeadLetterQueue(),
)
```

### DLQ Features

- **Failed Operation Tracking**: Records failed operations
- **Retry Mechanisms**: Automatic retry with exponential backoff
- **Metrics**: Comprehensive DLQ metrics
- **Manual Processing**: Manual retry of failed operations
- **Queue Management**: Configurable queue size and worker count

### Usage

```go
// Write-behind operations automatically use DLQ on failure
err := c.WriteBehind("user:123", user, 10*time.Minute, func() error {
    return saveUserToDB(user)
})
if err != nil {
    // Operation failed and was queued in DLQ
    log.Printf("Write-behind failed: %v", err)
}

// DLQ metrics are available
// Check dlq_failed_operations_total, dlq_retried_operations_total, etc.
```

## 🔒 Distributed Locks

### Redlock-Style Distributed Locking

```go
// Try to acquire distributed lock
unlock, ok, err := c.TryLock("user:123", 30*time.Second)
if err != nil {
    log.Printf("Lock acquisition failed: %v", err)
    return
}

if ok {
    defer unlock() // Always release lock
    
    // Critical section - only one process can execute this
    user, err := loadUserFromDB("123")
    if err != nil {
        return err
    }
    
    err = c.Set("user:123", user, 10*time.Minute)
    return err
} else {
    log.Printf("Lock not acquired")
}
```

### Lock Features

- **Redis-Based**: Uses Redis for distributed coordination
- **TTL Support**: Automatic lock expiration
- **Auto-Release**: Automatic cleanup on completion
- **Thread-Safe**: Safe concurrent lock operations
- **Error Handling**: Proper error handling for lock failures

## 📡 Pub/Sub Invalidation

### Multi-Service Cache Coherence

```go
// Service A: Publish invalidation
err := c.PublishInvalidation("user:123", "user:456")
if err != nil {
    log.Printf("Failed to publish invalidation: %v", err)
}

// Service B: Subscribe to invalidations
closeSub, err := c.SubscribeInvalidations(func(keys ...string) {
    for _, key := range keys {
        // Invalidate local cache
        localCache.Delete(key)
        log.Printf("Invalidated key: %s", key)
    }
})
if err != nil {
    log.Printf("Failed to subscribe: %v", err)
}
defer closeSub()
```

### Pub/Sub Features

- **Redis Pub/Sub**: Uses Redis pub/sub for message broadcasting
- **Message Serialization**: JSON-based message format
- **Health Monitoring**: Built-in health check messages
- **Subscriber Management**: Automatic subscriber lifecycle
- **Message Filtering**: Stale message detection

## 🌸 Bloom Filter

### Probabilistic Data Structure

```go
// Create cache with bloom filter
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithBloomFilter(),
)
```

### Bloom Filter Features

- **False Positive Reduction**: Reduces unnecessary backing store queries
- **Memory Efficient**: Minimal memory overhead
- **Configurable**: Adjustable false positive rate
- **Metrics**: Bloom filter statistics
- **Automatic Management**: Automatic key addition and cleanup

### Usage

```go
// Bloom filter is automatically used for cache misses
user, found, err := c.Get("user:123")
if !found {
    // Bloom filter checks if key might exist in backing store
    // If not, avoids unnecessary database query
    log.Printf("User not found in cache or backing store")
}
```

## 📊 Monitoring & Metrics

### Circuit Breaker Metrics

```prometheus
# Circuit breaker state
circuit_breaker_state{component="cachex"} # 0=closed, 1=half_open, 2=open

# Circuit breaker counters
circuit_breaker_failures_total
circuit_breaker_successes_total
```

### Retry Metrics

```prometheus
# Retry attempts
cache_operations_total{operation="get", status="retry"}
cache_operation_duration_seconds{operation="get"}
```

### DLQ Metrics

```prometheus
# DLQ statistics
dlq_failed_operations_total
dlq_retried_operations_total
dlq_succeeded_operations_total
dlq_dropped_operations_total
dlq_current_queue_size
```

### Bloom Filter Metrics

```prometheus
# Bloom filter statistics
bloom_filter_items
bloom_filter_false_positives_total
bloom_filter_capacity
```

## 🚀 Usage Examples

### Complete Resilience Setup

```go
package main

import (
    "context"
    "log"
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
        log.Fatal(err)
    }

    // Create cache with comprehensive resilience
    c, err := cache.New[User](
        cache.WithStore(redisStore),
        cache.WithDefaultTTL(5*time.Minute),
        cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
            Threshold:   5,
            Timeout:     30 * time.Second,
            HalfOpenMax: 3,
        }),
        cache.WithMaxRetries(3),
        cache.WithRetryDelay(100*time.Millisecond),
        cache.WithDeadLetterQueue(),
        cache.WithBloomFilter(),
        cache.WithObservability(cache.ObservabilityConfig{
            EnableMetrics: true,
            EnableTracing: true,
            EnableLogging: true,
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Subscribe to invalidations
    closeSub, err := c.SubscribeInvalidations(func(keys ...string) {
        log.Printf("Received invalidation for keys: %v", keys)
    })
    if err != nil {
        log.Printf("Failed to subscribe: %v", err)
    }
    defer closeSub()

    // Demonstrate resilient operations
    user := User{ID: "123", Name: "John Doe"}

    // Set with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err = c.Set("user:123", user, 10*time.Minute)
    if err != nil {
        log.Printf("Set failed: %v", err)
    }

    // Get with circuit breaker and retry
    cachedUser, found, err := c.Get("user:123")
    if err != nil {
        log.Printf("Get failed: %v", err)
    } else if found {
        log.Printf("User found: %s", cachedUser.Name)
    }

    // Use distributed lock for critical operations
    unlock, ok, err := c.TryLock("user:123", 30*time.Second)
    if err != nil {
        log.Printf("Lock failed: %v", err)
    } else if ok {
        defer unlock()
        
        // Critical section
        log.Printf("Executing critical section")
        
        // Publish invalidation
        err = c.PublishInvalidation("user:123")
        if err != nil {
            log.Printf("Invalidation failed: %v", err)
        }
    }
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

## 🎯 Best Practices

### 1. **Configure Appropriate Circuit Breaker Settings**

```go
// ✅ Good - Conservative settings for critical systems
cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
    Threshold:   3,              // Lower threshold for faster failure detection
    Timeout:     60 * time.Second, // Longer timeout for recovery
    HalfOpenMax: 1,              // Conservative half-open testing
})

// ❌ Bad - Too aggressive
cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
    Threshold:   20,             // Too high - slow failure detection
    Timeout:     5 * time.Second, // Too short - may not allow recovery
    HalfOpenMax: 10,             // Too many - may overwhelm system
})
```

### 2. **Use Appropriate Retry Policies**

```go
// ✅ Good - Exponential backoff with jitter
cache.WithMaxRetries(3),
cache.WithRetryDelay(100*time.Millisecond),

// ❌ Bad - No retries or too many retries
cache.WithMaxRetries(0),  // No resilience
// or
cache.WithMaxRetries(10), // Too many retries
```

### 3. **Set Proper Timeouts**

```go
// ✅ Good - Context-aware timeouts
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// ❌ Bad - No timeouts
// Operations may hang indefinitely
```

### 4. **Monitor DLQ Health**

```go
// ✅ Good - Monitor DLQ metrics
// Check dlq_failed_operations_total, dlq_current_queue_size

// ❌ Bad - Ignore DLQ
// Failed operations may accumulate
```

### 5. **Use Distributed Locks Appropriately**

```go
// ✅ Good - Short critical sections
unlock, ok, err := c.TryLock("key", 30*time.Second)
if ok {
    defer unlock()
    // Quick critical operation
}

// ❌ Bad - Long critical sections
unlock, ok, err := c.TryLock("key", 30*time.Second)
if ok {
    defer unlock()
    // Long operation that may timeout
    time.Sleep(60 * time.Second)
}
```

## 🔧 Configuration Examples

### Production Configuration

```yaml
# resilience.yaml
cache:
  circuit_breaker:
    threshold: 5
    timeout: "30s"
    half_open_max: 3
  retry:
    max_retries: 3
    initial_delay: "100ms"
    max_delay: "1s"
    multiplier: 2.0
    jitter: true
  timeout: "5s"
  enable_dead_letter_queue: true
  enable_bloom_filter: true

pubsub:
  enabled: true
  invalidation_channel: "cache:invalidation"
  health_channel: "cache:health"
  max_subscribers: 50
  message_timeout: "10s"
```

### Development Configuration

```yaml
# development.yaml
cache:
  circuit_breaker:
    threshold: 3
    timeout: "10s"
    half_open_max: 1
  retry:
    max_retries: 2
    initial_delay: "50ms"
    max_delay: "500ms"
  timeout: "2s"
  enable_dead_letter_queue: false
  enable_bloom_filter: false

pubsub:
  enabled: false
```

## 🔍 Troubleshooting

### Common Issues

1. **Circuit Breaker Always Open**
   - Check failure thresholds and timeouts
   - Monitor underlying system health
   - Verify Redis connectivity

2. **High Retry Counts**
   - Check retry configuration
   - Monitor underlying system performance
   - Verify timeout settings

3. **DLQ Queue Growing**
   - Check DLQ worker configuration
   - Monitor failed operation patterns
   - Verify backing store health

4. **Lock Contention**
   - Reduce critical section duration
   - Use more granular locks
   - Monitor lock acquisition patterns

### Debug Configuration

```go
// Enable debug logging
log.SetLevel(log.DebugLevel)

// Check circuit breaker state
// Monitor circuit_breaker_state metric

// Check DLQ health
// Monitor dlq_current_queue_size metric

// Check bloom filter performance
// Monitor bloom_filter_false_positives_total metric
```

## ✅ Implementation Status

- ✅ **Circuit Breaker**: Full implementation with configurable thresholds
- ✅ **Retry Mechanisms**: Exponential backoff with jitter
- ✅ **Timeout Management**: Context-aware timeout handling
- ✅ **Dead Letter Queue**: Failed operation handling with retry
- ✅ **Distributed Locks**: Redlock-style distributed locking
- ✅ **Pub/Sub Invalidation**: Multi-service cache coherence
- ✅ **Bloom Filter**: Probabilistic data structure for efficiency
- ✅ **Metrics**: Comprehensive monitoring and alerting
- ✅ **Thread Safety**: Race-free resilience operations
- ✅ **Production Ready**: Enterprise-grade resilience features

This resilience system provides comprehensive failure handling, recovery mechanisms, and monitoring capabilities for production cache deployments.
