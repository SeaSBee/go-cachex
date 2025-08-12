# Bulkhead Isolation

Go-CacheX provides **bulkhead isolation** with separate connection pools for read and write operations, ensuring operational isolation and better resource utilization.

## Overview

Bulkhead isolation separates read and write operations into dedicated connection pools, providing:

- **Operational Isolation**: Read and write operations don't compete for resources
- **Independent Scaling**: Each pool can be optimized for its specific workload
- **Failure Isolation**: Issues in one pool don't affect the other
- **Better Performance**: Predictable latency for each operation type

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Go-CacheX Application                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │   Read Pool     │    │  Write Pool     │                │
│  │   (30 conns)    │    │   (15 conns)    │                │
│  └─────────────────┘    └─────────────────┘                │
│           │                       │                        │
│           ▼                       ▼                        │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │  Read Client    │    │ Write Client    │                │
│  │   (go-redis)    │    │   (go-redis)    │                │
│  └─────────────────┘    └─────────────────┘                │
│           │                       │                        │
└───────────┼───────────────────────┼────────────────────────┘
            │                       │
            ▼                       ▼
    ┌─────────────────────────────────────────┐
    │              Redis Server               │
    └─────────────────────────────────────────┘
```

## Configuration

### Default Configuration

```go
config := cache.DefaultBulkheadConfig()
// Read pool: 30 connections (15 idle)
// Write pool: 15 connections (8 idle)
```

### High-Performance Configuration

```go
config := cache.HighPerformanceBulkheadConfig()
// Read pool: 50 connections (25 idle)
// Write pool: 25 connections (12 idle)
```

### Resource-Constrained Configuration

```go
config := cache.ResourceConstrainedBulkheadConfig()
// Read pool: 10 connections (5 idle)
// Write pool: 5 connections (2 idle)
```

### Custom Configuration

```go
config := redisstore.BulkheadConfig{
    // Read pool configuration
    ReadPoolSize:     40,
    ReadMinIdleConns: 20,
    ReadMaxRetries:   4,
    ReadDialTimeout:  8 * time.Second,
    ReadTimeout:      4 * time.Second,
    ReadPoolTimeout:  6 * time.Second,

    // Write pool configuration
    WritePoolSize:     20,
    WriteMinIdleConns: 10,
    WriteMaxRetries:   4,
    WriteDialTimeout:  8 * time.Second,
    WriteTimeout:      4 * time.Second,
    WritePoolTimeout:  6 * time.Second,

    // Common settings
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
}
```

## Usage

### Basic Usage

```go
// Create cache with bulkhead isolation
c, err := cache.New[string](
    cache.WithBulkheadStore(cache.DefaultBulkheadConfig()),
    cache.WithDefaultTTL(10*time.Minute),
)
if err != nil {
    log.Fatal(err)
}
defer c.Close()

// Read operations use the read pool
value, found, err := c.Get("user:123")
if err != nil {
    log.Printf("Get failed: %v", err)
}

// Write operations use the write pool
err = c.Set("user:123", "John Doe", 5*time.Minute)
if err != nil {
    log.Printf("Set failed: %v", err)
}
```

### Redis Cluster Support

```go
// Create bulkhead configuration for cluster
config := redisstore.BulkheadConfig{
    ReadPoolSize:     30,
    WritePoolSize:    15,
    // ... other settings
}

// Redis cluster addresses
addrs := []string{
    "redis-cluster-1:6379",
    "redis-cluster-2:6379",
    "redis-cluster-3:6379",
}

c, err := cache.New[string](
    cache.WithBulkheadClusterStore(addrs, "password", config),
    cache.WithDefaultTTL(30*time.Minute),
)
if err != nil {
    log.Fatal(err)
}
defer c.Close()
```

## Operation Routing

### Read Operations (Read Pool)
- `Get(key)` - Retrieve single value
- `MGet(keys...)` - Retrieve multiple values
- `Exists(key)` - Check if key exists
- `TTL(key)` - Get time to live

### Write Operations (Write Pool)
- `Set(key, value, ttl)` - Store single value
- `MSet(items, ttl)` - Store multiple values
- `Del(keys...)` - Delete keys
- `IncrBy(key, delta, ttlIfCreate)` - Increment value

## Benefits

### 1. Operational Isolation

Read and write operations are completely isolated:

```go
// These operations run on separate pools
go func() {
    // Uses read pool
    value, _, _ := c.Get("key1")
    fmt.Println("Read:", value)
}()

go func() {
    // Uses write pool
    c.Set("key2", "value2", time.Minute)
    fmt.Println("Write completed")
}()
```

### 2. Independent Scaling

Each pool can be sized optimally:

```go
// For read-heavy workloads
config := redisstore.BulkheadConfig{
    ReadPoolSize:     50,  // Large read pool
    WritePoolSize:    10,  // Small write pool
}

// For write-heavy workloads
config := redisstore.BulkheadConfig{
    ReadPoolSize:     20,  // Small read pool
    WritePoolSize:    40,  // Large write pool
}
```

### 3. Failure Isolation

Issues in one pool don't affect the other:

```go
// If write pool is exhausted, reads still work
for i := 0; i < 1000; i++ {
    go func() {
        c.Set("key", "value", time.Minute) // May block on write pool
    }()
}

// Read operations continue to work
value, found, err := c.Get("existing-key") // Uses read pool
```

### 4. Predictable Performance

Each operation type has consistent latency:

```go
// Read operations have predictable performance
start := time.Now()
value, _, _ := c.Get("key")
readLatency := time.Since(start)

// Write operations have predictable performance
start = time.Now()
c.Set("key", "value", time.Minute)
writeLatency := time.Since(start)
```

## Configuration Guidelines

### Read-Heavy Workloads

```go
config := redisstore.BulkheadConfig{
    ReadPoolSize:     50,
    ReadMinIdleConns: 25,
    WritePoolSize:    10,
    WriteMinIdleConns: 5,
}
```

### Write-Heavy Workloads

```go
config := redisstore.BulkheadConfig{
    ReadPoolSize:     20,
    ReadMinIdleConns: 10,
    WritePoolSize:    40,
    WriteMinIdleConns: 20,
}
```

### Balanced Workloads

```go
config := redisstore.BulkheadConfig{
    ReadPoolSize:     30,
    ReadMinIdleConns: 15,
    WritePoolSize:    20,
    WriteMinIdleConns: 10,
}
```

### Resource-Constrained Environments

```go
config := redisstore.BulkheadConfig{
    ReadPoolSize:     10,
    ReadMinIdleConns: 5,
    WritePoolSize:    5,
    WriteMinIdleConns: 2,
}
```

## Monitoring and Observability

### Pool Metrics

Monitor pool utilization and performance:

```go
// Access pool clients for metrics
bulkheadStore := c.(*redisstore.BulkheadStore)
readClient := bulkheadStore.ReadClient()
writeClient := bulkheadStore.WriteClient()

// Monitor pool stats
// (Implementation depends on your monitoring system)
```

### Health Checks

```go
// Check read pool health
readClient := bulkheadStore.ReadClient()
err := readClient.Ping(ctx).Err()
if err != nil {
    log.Printf("Read pool health check failed: %v", err)
}

// Check write pool health
writeClient := bulkheadStore.WriteClient()
err = writeClient.Ping(ctx).Err()
if err != nil {
    log.Printf("Write pool health check failed: %v", err)
}
```

## Best Practices

### 1. Pool Sizing

- **Read Pool**: Size based on expected read throughput
- **Write Pool**: Size based on expected write throughput
- **Monitor**: Track pool utilization and adjust accordingly

### 2. Timeout Configuration

```go
config := redisstore.BulkheadConfig{
    // Shorter timeouts for reads (usually faster)
    ReadTimeout:      2 * time.Second,
    ReadPoolTimeout:  3 * time.Second,
    
    // Longer timeouts for writes (may take longer)
    WriteTimeout:     5 * time.Second,
    WritePoolTimeout: 8 * time.Second,
}
```

### 3. Retry Configuration

```go
config := redisstore.BulkheadConfig{
    // More retries for reads (idempotent)
    ReadMaxRetries:   5,
    
    // Fewer retries for writes (may not be idempotent)
    WriteMaxRetries:  2,
}
```

### 4. TLS Configuration

```go
config := redisstore.BulkheadConfig{
    // ... other settings
    TLSConfig: &redisstore.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: false,
    },
}
```

## Migration from Single Pool

### Before (Single Pool)

```go
// Old single pool configuration
redisConfig := &redisstore.Config{
    Addr:     "localhost:6379",
    PoolSize: 20,
}

c, err := cache.New[string](
    cache.WithStore(redisstore.New(redisConfig)),
)
```

### After (Bulkhead Isolation)

```go
// New bulkhead configuration
bulkheadConfig := cache.DefaultBulkheadConfig()
bulkheadConfig.Addr = "localhost:6379"

c, err := cache.New[string](
    cache.WithBulkheadStore(bulkheadConfig),
)
```

## Troubleshooting

### Common Issues

1. **Pool Exhaustion**
   - Monitor pool utilization
   - Increase pool size if needed
   - Check for connection leaks

2. **Timeout Errors**
   - Adjust timeout settings
   - Check network connectivity
   - Monitor Redis server performance

3. **Performance Issues**
   - Profile pool utilization
   - Adjust pool sizes based on workload
   - Consider workload-specific configurations

### Debug Information

```go
// Get bulkhead configuration
config := bulkheadStore.Config()
fmt.Printf("Read pool size: %d\n", config.ReadPoolSize)
fmt.Printf("Write pool size: %d\n", config.WritePoolSize)

// Access underlying clients
readClient := bulkheadStore.ReadClient()
writeClient := bulkheadStore.WriteClient()
```

## Examples

See the [bulkhead examples](../examples/bulkhead/main.go) for complete working examples demonstrating:

- Default bulkhead configuration
- High-performance configuration
- Resource-constrained configuration
- Custom configuration
- Redis cluster support
- Bulkhead isolation benefits

## Performance Considerations

### Memory Usage

- Each pool maintains its own connection pool
- Memory usage is proportional to pool sizes
- Monitor memory usage in resource-constrained environments

### Network Connections

- Total connections = ReadPoolSize + WritePoolSize
- Ensure Redis server can handle the total connection count
- Consider Redis server connection limits

### Latency Impact

- Read operations: Minimal latency impact
- Write operations: Minimal latency impact
- Pool acquisition: Very low overhead
- Overall: Better performance due to reduced contention
