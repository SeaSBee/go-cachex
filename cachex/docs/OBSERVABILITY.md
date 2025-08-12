# Observability Features

Go-CacheX provides comprehensive observability features including OpenTelemetry tracing, Prometheus metrics, and enhanced structured logging via SeaSBee/go-logx.

## 📊 Overview

The observability system is built around three core pillars:

1. **OpenTelemetry Tracing** - Distributed tracing for cache operations
2. **Prometheus Metrics** - Comprehensive metrics collection
3. **Structured Logging** - Enhanced logging with SeaSBee/go-logx

## 🔧 Configuration

### Basic Configuration

```go
import (
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "time"
)

// Create cache with observability enabled
c, err := cache.New[User](
    cache.WithStore(store),
    cache.WithObservability(cache.ObservabilityConfig{
        EnableMetrics: true,
        EnableTracing: true,
        EnableLogging: true,
    }),
    cache.WithSecurity(cache.SecurityConfig{
        RedactLogs: false, // Set to true to redact sensitive data
    }),
)
```

### Advanced Configuration

```go
// Custom observability configuration
obsConfig := &observability.Config{
    EnableMetrics:   true,
    EnableTracing:   true,
    EnableLogging:   true,
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    Environment:     "production",
}

obs := observability.New(obsConfig)
```

## 🔍 OpenTelemetry Tracing

### Automatic Span Creation

All cache operations automatically create OpenTelemetry spans with rich attributes:

```go
// Get operation with tracing
user, found, err := c.Get("user:123")
// Creates span: cachex.get
// Attributes: cache.operation=get, cache.key=user:123, cache.component=cachex
```

### Span Attributes

Each span includes the following attributes:

- `cache.operation` - Operation type (get, set, del, etc.)
- `cache.key` - Cache key (redacted if Security.RedactLogs=true)
- `cache.namespace` - Extracted namespace from key
- `cache.ttl_ms` - TTL in milliseconds
- `cache.attempt` - Retry attempt number
- `cache.component` - Component name (cachex)
- `cache.service.name` - Service name
- `cache.service.version` - Service version
- `cache.environment` - Environment

### Manual Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

// Create custom spans
tracer := otel.Tracer("my-app")
ctx, span := tracer.Start(ctx, "custom-cache-operation",
    trace.WithAttributes(
        attribute.String("cache.operation", "batch-get"),
        attribute.String("cache.keys", "key1,key2,key3"),
    ),
)
defer span.End()

// Perform cache operations
users, err := c.MGet("user:1", "user:2", "user:3")
```

### Tracing Setup

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracing() {
    // Create exporter
    exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
    if err != nil {
        log.Fatal(err)
    }

    // Create resource
    res, err := resource.New(context.Background(),
        resource.WithAttributes(
            semconv.ServiceName("my-service"),
            semconv.ServiceVersion("1.0.0"),
            semconv.DeploymentEnvironment("production"),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create trace provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )

    // Set global provider
    otel.SetTracerProvider(tp)
}
```

## 📈 Prometheus Metrics

### Available Metrics

#### Operation Metrics

```prometheus
# Operation counters
cache_operations_total{operation="get", status="hit", component="cachex"}
cache_operations_total{operation="get", status="miss", component="cachex"}
cache_operations_total{operation="set", status="success", component="cachex"}
cache_operations_total{operation="del", status="success", component="cachex"}

# Operation duration histograms
cache_operation_duration_seconds{operation="get", component="cachex"}
cache_operation_duration_seconds{operation="set", component="cachex"}
```

#### Cache Performance Metrics

```prometheus
# Cache hits and misses
cache_hits_total
cache_misses_total
cache_sets_total
cache_dels_total

# Data transfer
cache_bytes_in_total
cache_bytes_out_total
```

#### Error Metrics

```prometheus
# Error counters by type
cache_errors_total{type="timeout", component="cachex"}
cache_errors_total{type="connection_failed", component="cachex"}
cache_errors_total{type="operation_failed", component="cachex"}
```

#### Circuit Breaker Metrics

```prometheus
# Circuit breaker state
circuit_breaker_state{component="cachex"} # 0=closed, 1=half_open, 2=open

# Circuit breaker counters
circuit_breaker_failures_total
circuit_breaker_successes_total
```

#### Worker Pool Metrics

```prometheus
# Worker pool statistics
worker_pool_active_workers
worker_pool_queued_jobs
worker_pool_completed_jobs_total
worker_pool_failed_jobs_total
```

#### Dead Letter Queue Metrics

```prometheus
# DLQ statistics
dlq_failed_operations_total
dlq_retried_operations_total
dlq_succeeded_operations_total
dlq_dropped_operations_total
dlq_current_queue_size
```

#### Bloom Filter Metrics

```prometheus
# Bloom filter statistics
bloom_filter_items
bloom_filter_false_positives_total
bloom_filter_capacity
```

### Metrics Server

```go
import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Start metrics server
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":8080", nil)
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'cachex'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Grafana Dashboards

Example queries for Grafana dashboards:

```promql
# Cache hit rate
rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))

# Average operation duration
histogram_quantile(0.95, rate(cache_operation_duration_seconds_bucket[5m]))

# Error rate
rate(cache_errors_total[5m])

# Circuit breaker state
circuit_breaker_state
```

## 📝 Structured Logging with go-logx

### Enhanced Log Fields

All cache operations are logged with enhanced structured fields:

```json
{
  "level": "info",
  "component": "cachex",
  "operation": "get",
  "service_name": "cachex-demo",
  "service_version": "1.0.0",
  "environment": "development",
  "namespace": "user",
  "ttl": "10m0s",
  "attempt": 1,
  "duration_ms": 5,
  "key": "user:123",
  "time": "2024-01-15T10:30:00Z"
}
```

### Required Log Fields

The logging system ensures all required fields are present:

- **component**: Always set to "cachex"
- **op**: Operation type (get, set, del, etc.)
- **key_ns**: Key namespace (extracted from key)
- **key_type**: Key type (user, product, etc.)
- **ttl_ms**: TTL in milliseconds
- **attempt**: Retry attempt number
- **duration_ms**: Operation duration in milliseconds
- **err**: Error field when operations fail

### Log Levels

- **DEBUG** - Successful operations
- **INFO** - General operations
- **WARN** - Non-critical issues (bloom filter warnings, etc.)
- **ERROR** - Failed operations

### Log Redaction

When `Security.RedactLogs=true`, sensitive data is automatically redacted:

```json
{
  "level": "info",
  "component": "cachex",
  "operation": "get",
  "key": "[REDACTED]",
  "duration_ms": 5
}
```

### Custom Logging

```go
// Access observability instance for custom logging
if c.observability != nil {
    c.observability.LogOperation(
        "info",
        "custom-operation",
        "user:123",
        "user",
        10*time.Minute,
        1,
        5*time.Millisecond,
        nil,
        false,
    )
}
```

## 🔧 Integration Examples

### Complete Observability Setup

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
    // Initialize tracing
    initTracing()

    // Create cache with observability
    store, _ := memorystore.New(&memorystore.Config{
        DefaultTTL:      5 * time.Minute,
        CleanupInterval: 1 * time.Minute,
    })
    c, _ := cache.New[User](
        cache.WithStore(store),
        cache.WithObservability(cache.ObservabilityConfig{
            EnableMetrics: true,
            EnableTracing: true,
            EnableLogging: true,
        }),
    )

    // Start metrics server
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        log.Fatal(http.ListenAndServe(":8080", nil))
    }()

    // Use cache with full observability
    user := User{ID: "123", Name: "John"}
    c.Set("user:123", user, 10*time.Minute)
    
    if _, found, err := c.Get("user:123"); err != nil {
        log.Printf("Error: %v", err)
    } else {
        log.Printf("User found: %t", found)
    }
}

func initTracing() {
    exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
    res, _ := resource.New(context.Background(),
        resource.WithAttributes(semconv.ServiceName("my-app")),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### Production Configuration

```go
// Production-ready observability setup
func createProductionCache() (cache.Cache[User], error) {
    // Initialize Jaeger tracing
    initJaegerTracing()

    // Create Redis store
    redisStore, err := redisstore.New(&redisstore.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    if err != nil {
        return nil, err
    }

    // Create cache with production observability
    return cache.New[User](
        cache.WithStore(redisStore),
        cache.WithObservability(cache.ObservabilityConfig{
            EnableMetrics: true,
            EnableTracing: true,
            EnableLogging: true,
        }),
        cache.WithSecurity(cache.SecurityConfig{
            RedactLogs: true, // Redact sensitive data in production
        }),
        cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
            Threshold:   5,
            Timeout:     30 * time.Second,
            HalfOpenMax: 3,
        }),
        cache.WithDeadLetterQueue(),
        cache.WithBloomFilter(),
    )
}
```

## 📊 Monitoring & Alerting

### Key Metrics to Monitor

1. **Cache Hit Rate** - Should be > 80%
2. **Error Rate** - Should be < 1%
3. **Circuit Breaker State** - Monitor for open states
4. **Operation Latency** - P95 should be < 10ms
5. **DLQ Queue Size** - Should be minimal

### Example Alerting Rules

```yaml
# prometheus-alerts.yml
groups:
  - name: cachex
    rules:
      - alert: CacheHitRateLow
        expr: rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m])) < 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cache hit rate is low"

      - alert: CacheErrorRateHigh
        expr: rate(cache_errors_total[5m]) > 0.01
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Cache error rate is high"

      - alert: CircuitBreakerOpen
        expr: circuit_breaker_state > 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Circuit breaker is open"
```

## 🎯 Best Practices

### 1. **Enable All Observability Features in Production**

```go
cache.WithObservability(cache.ObservabilityConfig{
    EnableMetrics: true,
    EnableTracing: true,
    EnableLogging: true,
})
```

### 2. **Use Log Redaction for Sensitive Data**

```go
cache.WithSecurity(cache.SecurityConfig{
    RedactLogs: true,
})
```

### 3. **Monitor Key Metrics**

- Cache hit rate
- Error rates
- Circuit breaker state
- Operation latency
- DLQ queue size

### 4. **Set Up Proper Tracing**

- Use Jaeger or Zipkin for distributed tracing
- Configure proper sampling rates
- Set up trace correlation with application traces

### 5. **Configure Prometheus Scraping**

- Set appropriate scrape intervals
- Configure alerting rules
- Set up Grafana dashboards

### 6. **Use Structured Logging**

- Log all cache operations
- Include relevant context
- Use appropriate log levels

## 🔮 Future Enhancements

### Planned Features

- **Custom Metrics** - User-defined metrics
- **Trace Sampling** - Configurable sampling rates
- **Log Aggregation** - Integration with ELK stack
- **Performance Profiling** - CPU and memory profiling
- **Health Checks** - Built-in health check endpoints

### Integration Possibilities

- **Jaeger** - Distributed tracing
- **Zipkin** - Distributed tracing
- **ELK Stack** - Log aggregation
- **Grafana** - Metrics visualization
- **Datadog** - APM integration
- **New Relic** - APM integration

## ✅ Implementation Status

- ✅ **OpenTelemetry Tracing**: Fully implemented with automatic span creation
- ✅ **Prometheus Metrics**: Comprehensive metrics collection
- ✅ **Structured Logging**: Complete implementation with go-logx
- ✅ **Log Redaction**: Automatic redaction when Security.RedactLogs=true
- ✅ **Required Log Fields**: All 8 required fields implemented
- ✅ **Thread Safety**: Race-free observability operations
- ✅ **Production Ready**: Enterprise-grade observability features

This observability system provides comprehensive monitoring, tracing, and logging capabilities for production cache deployments.
