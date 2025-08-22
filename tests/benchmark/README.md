# go-cachex Benchmark Tests

This directory contains comprehensive benchmark tests for all core functionalities of the `go-cachex` library.

## Overview

The benchmark tests cover the following areas:

### Store Benchmarks
- **Redis Store**: Set, Get, MSet, MGet, IncrBy operations
- **Memory Store**: Set, Get, MSet, MGet operations  
- **Ristretto Store**: Set, Get operations (with async handling)
- **Layered Store**: Set, Get operations (L1 Memory + L2 Redis)

### Feature Benchmarks
- **Tagging**: AddTags, GetKeysByTag operations
- **Codecs**: MessagePack vs JSON Encode/Decode performance
- **Key Builder**: Key generation performance
- **Observability**: Metrics-enabled cache operations
- **Configuration**: Cache creation from config

### Data Type Benchmarks
- **Simple Data**: Basic User struct operations
- **Complex Data**: Product struct with nested fields
- **Large Data**: Large map operations
- **TTL Operations**: Time-to-live operations
- **Concurrent Operations**: Mixed operation types

## Prerequisites

1. **Redis Server**: Must be running at `localhost:6379`
   ```bash
   redis-server
   ```

2. **Go Environment**: Go 1.24.5 or later

## Running Benchmarks

### Quick Run
```bash
# Run all benchmarks
go test -bench=. -benchmem ./tests/benchmark/

# Run with verbose output
go test -v -bench=. -benchmem ./tests/benchmark/

# Run specific benchmark
go test -bench=BenchmarkRedisSet -benchmem ./tests/benchmark/
```

### Using the Script
```bash
# Make script executable
chmod +x tests/benchmark/run_benchmarks.sh

# Run benchmarks with summary
./tests/benchmark/run_benchmarks.sh
```

### Benchmark Options

- `-bench=.`: Run all benchmarks
- `-benchmem`: Include memory allocation statistics
- `-benchtime=10s`: Set benchmark duration (default: 1s)
- `-count=5`: Run each benchmark 5 times for averaging
- `-cpu=1,2,4`: Run benchmarks with different CPU counts

## Example Output

```
BenchmarkRedisSet-8             10000            123456 ns/op          2048 B/op         12 allocs/op
BenchmarkRedisGet-8             20000             67890 ns/op          1024 B/op          8 allocs/op
BenchmarkMemorySet-8            50000             23456 ns/op           512 B/op          4 allocs/op
BenchmarkMemoryGet-8           100000             12345 ns/op           256 B/op          2 allocs/op
```

## Benchmark Categories

### Performance Comparison
- **Redis vs Memory**: Network vs in-memory performance
- **MessagePack vs JSON**: Serialization performance
- **Layered vs Single**: Multi-tier cache performance
- **Ristretto vs Memory**: High-performance cache comparison

### Scalability Tests
- **Concurrent Operations**: Multi-goroutine performance
- **Large Data**: Memory usage with large objects
- **Batch Operations**: MSet/MGet vs individual operations

### Feature Overhead
- **Observability**: Metrics/tracing overhead
- **Tagging**: Tag management overhead
- **TTL**: Time-to-live overhead

## Interpreting Results

### Key Metrics
- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)

### Performance Patterns
- **Redis**: Higher latency, network overhead, persistent storage
- **Memory**: Lowest latency, no network, volatile storage
- **Ristretto**: High performance, automatic eviction, memory efficient
- **Layered**: Best of both worlds, L1 fast access, L2 persistence

## Troubleshooting

### Common Issues

1. **Redis Connection Failed**
   ```
   Error: Failed to create Redis store: dial tcp localhost:6379: connect: connection refused
   ```
   **Solution**: Start Redis server with `redis-server`

2. **Benchmark Timeout**
   ```
   Error: timeout waiting for benchmark to complete
   ```
   **Solution**: Increase timeout with `-timeout=300s`

3. **Memory Issues**
   ```
   Error: out of memory
   ```
   **Solution**: Reduce benchmark data size or increase system memory

### Performance Tips

1. **Warm-up**: First few benchmark runs may be slower due to JIT compilation
2. **System Load**: Ensure minimal background processes during benchmarking
3. **Network**: For Redis benchmarks, ensure stable network connection
4. **Memory**: Monitor system memory usage during large data benchmarks

## Contributing

When adding new benchmarks:

1. Follow the naming convention: `Benchmark<Component><Operation>`
2. Include both time and memory metrics
3. Use realistic data sizes and operation patterns
4. Add appropriate error handling and cleanup
5. Document any special setup requirements

## Notes

- Benchmarks use parallel execution (`b.RunParallel`) for realistic concurrent load testing
- Ristretto benchmarks include sleep delays to account for asynchronous operations
- Layered cache benchmarks test both L1 and L2 store interactions
- All benchmarks include proper cleanup to prevent resource leaks
