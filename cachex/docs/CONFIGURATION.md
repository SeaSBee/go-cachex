# Configuration Management

Go-CacheX provides comprehensive configuration management with support for functional options, YAML files, environment variables, and hot-reload capabilities.

## 📋 Overview

The configuration system supports:

- **✅ Functional Options** - Type-safe configuration via builder pattern
- **✅ YAML Configuration Files** - Structured configuration files
- **✅ Environment Variables** - Runtime configuration overrides
- **✅ Hot-Reload** - Dynamic configuration updates via SIGHUP or file monitoring
- **✅ Validation** - Configuration validation and error handling

## 🔧 Configuration Sources

### 1. Functional Options (Recommended)

```go
import (
    "time"
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// Create Redis store
redisStore, err := redisstore.New(&redisstore.Config{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
if err != nil {
    panic(err)
}

// Create cache with functional options
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithDefaultTTL(10*time.Minute),
    cache.WithMaxRetries(5),
    cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
        Threshold:   10,
        Timeout:     60 * time.Second,
        HalfOpenMax: 5,
    }),
    cache.WithRateLimit(cache.RateLimitConfig{
        RequestsPerSecond: 1500,
        Burst:             150,
    }),
    cache.WithSecurity(cache.SecurityConfig{
        RedactLogs: true,
    }),
    cache.WithObservability(cache.ObservabilityConfig{
        EnableMetrics: true,
        EnableTracing: true,
        EnableLogging: true,
    }),
    cache.WithDeadLetterQueue(),
    cache.WithBloomFilter(),
)
```

### 2. YAML Configuration File

```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 20
  min_idle_conns: 10
  tls:
    enabled: false

cache:
  default_ttl: "15m"
  circuit_breaker:
    threshold: 8
    timeout: "45s"
    half_open_max: 5
  rate_limit:
    requests_per_second: 2000
    burst: 200
  retry:
    max_retries: 5
    initial_delay: "200ms"
    max_delay: "2s"
    multiplier: 2.5
    jitter: true
  enable_dead_letter_queue: true
  enable_bloom_filter: true

pool:
  worker_pool:
    min_workers: 10
    max_workers: 50
    queue_size: 2000
    idle_timeout: "60s"
    enable_metrics: true
  pipeline:
    batch_size: 200
    max_concurrent: 20
    batch_timeout: "200ms"
    enable_metrics: true

security:
  encryption_key: "your-secret-key-here"
  redact_logs: true
  enable_tls: false
  max_key_length: 512
  max_value_size: 2097152
  secrets_prefix: "APP_"

observability:
  enable_metrics: true
  enable_tracing: true
  enable_logging: true
  service_name: "my-service"
  service_version: "2.0.0"
  environment: "staging"
  metrics:
    port: 9090
    path: "/metrics"
    enabled: true

pubsub:
  enabled: true
  invalidation_channel: "cache:invalidation"
  health_channel: "cache:health"
  max_subscribers: 200
  message_timeout: "10s"

hot_reload:
  enabled: true
  config_file: "config.yaml"
  check_interval: "30s"
  signal_reload: true
```

### 3. Environment Variables

```bash
# Redis Configuration
export CACHEX_REDIS_ADDR="localhost:6379"
export CACHEX_REDIS_PASSWORD="your_password"
export CACHEX_REDIS_DB="0"
export CACHEX_REDIS_TLS_ENABLED="true"
export CACHEX_REDIS_TLS_INSECURE_SKIP_VERIFY="false"
export CACHEX_REDIS_POOL_SIZE="20"

# Cache Configuration
export CACHEX_DEFAULT_TTL="10m"
export CACHEX_MAX_RETRIES="5"
export CACHEX_CIRCUIT_BREAKER_THRESHOLD="10"
export CACHEX_CIRCUIT_BREAKER_TIMEOUT="60s"
export CACHEX_RATE_LIMIT_REQUESTS_PER_SECOND="1500"
export CACHEX_RATE_LIMIT_BURST="150"

# Security Configuration
export CACHEX_ENCRYPTION_KEY="your-secret-key"
export CACHEX_REDACT_LOGS="true"
export CACHEX_ENABLE_TLS="true"
export CACHEX_MAX_KEY_LENGTH="512"
export CACHEX_MAX_VALUE_SIZE="2097152"

# Observability Configuration
export CACHEX_ENABLE_METRICS="true"
export CACHEX_ENABLE_TRACING="true"
export CACHEX_ENABLE_LOGGING="true"
export CACHEX_SERVICE_NAME="my-service"
export CACHEX_SERVICE_VERSION="2.0.0"
export CACHEX_ENVIRONMENT="production"

# Feature Flags
export CACHEX_ENABLE_DEAD_LETTER_QUEUE="true"
export CACHEX_ENABLE_BLOOM_FILTER="true"
```

## 🔄 Hot-Reload Configuration

### File-Based Hot-Reload

```go
import (
    "github.com/SeaSBee/go-cachex/cachex/internal/config"
)

// Create configuration manager
manager := config.NewManager("config.yaml")
cfg, err := manager.Load()
if err != nil {
    log.Fatal(err)
}

// Register reload callback
manager.OnReload(func(newConfig *config.Config) {
    log.Printf("Configuration reloaded! New TTL: %v", newConfig.Cache.DefaultTTL)
    // Update your application with new configuration
})

// Start hot-reload monitoring
if err := manager.StartHotReload(); err != nil {
    log.Fatal(err)
}

// Stop monitoring when done
defer manager.Stop()
```

### Signal-Based Hot-Reload

```bash
# Send SIGHUP to reload configuration
kill -HUP <pid>
```

### Hot-Reload Configuration

```yaml
hot_reload:
  enabled: true
  config_file: "config.yaml"
  check_interval: "30s"  # How often to check for file changes
  signal_reload: true    # Enable SIGHUP reload
```

## 🔧 Configuration Loading

### Basic Loading

```go
import (
    "github.com/SeaSBee/go-cachex/cachex/internal/config"
)

// Load from file
cfg, err := config.LoadFromFile("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Load with environment overrides
config.LoadFromEnvironment(cfg)

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}
```

### Manager-Based Loading

```go
// Create manager
manager := config.NewManager("config.yaml")

// Load configuration
cfg, err := manager.Load()
if err != nil {
    log.Fatal(err)
}

// Get current configuration
currentConfig := manager.Get()

// Register reload callbacks
manager.OnReload(func(newConfig *config.Config) {
    // Handle configuration changes
})

// Start hot-reload
if err := manager.StartHotReload(); err != nil {
    log.Fatal(err)
}

// Stop when done
defer manager.Stop()
```

## 📊 Configuration Structure

### Redis Configuration

```go
type RedisConfig struct {
    Addr         string        `yaml:"addr"`
    Password     string        `yaml:"password"`
    DB           int           `yaml:"db"`
    TLS          TLSConfig     `yaml:"tls"`
    PoolSize     int           `yaml:"pool_size"`
    MinIdleConns int           `yaml:"min_idle_conns"`
    MaxRetries   int           `yaml:"max_retries"`
    DialTimeout  time.Duration `yaml:"dial_timeout"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    PoolTimeout  time.Duration `yaml:"pool_timeout"`
}
```

### Cache Configuration

```go
type CacheConfig struct {
    DefaultTTL            time.Duration        `yaml:"default_ttl"`
    CircuitBreaker        CircuitBreakerConfig `yaml:"circuit_breaker"`
    RateLimit             RateLimitConfig      `yaml:"rate_limit"`
    Retry                 RetryConfig          `yaml:"retry"`
    EnableDeadLetterQueue bool                 `yaml:"enable_dead_letter_queue"`
    EnableBloomFilter     bool                 `yaml:"enable_bloom_filter"`
}
```

### Pool Configuration

```go
type PoolConfig struct {
    WorkerPool WorkerPoolConfig `yaml:"worker_pool"`
    Pipeline   PipelineConfig   `yaml:"pipeline"`
}

type WorkerPoolConfig struct {
    MinWorkers    int           `yaml:"min_workers"`
    MaxWorkers    int           `yaml:"max_workers"`
    QueueSize     int           `yaml:"queue_size"`
    IdleTimeout   time.Duration `yaml:"idle_timeout"`
    EnableMetrics bool          `yaml:"enable_metrics"`
}

type PipelineConfig struct {
    BatchSize     int           `yaml:"batch_size"`
    MaxConcurrent int           `yaml:"max_concurrent"`
    BatchTimeout  time.Duration `yaml:"batch_timeout"`
    EnableMetrics bool          `yaml:"enable_metrics"`
}
```

### Security Configuration

```go
type SecurityConfig struct {
    EncryptionKey     string `yaml:"encryption_key"`
    EncryptionKeyFile string `yaml:"encryption_key_file"`
    RedactLogs        bool   `yaml:"redact_logs"`
    EnableTLS         bool   `yaml:"enable_tls"`
    InsecureSkipVerify bool  `yaml:"insecure_skip_verify"`
    MaxKeyLength      int    `yaml:"max_key_length"`
    MaxValueSize      int    `yaml:"max_value_size"`
    SecretsPrefix     string `yaml:"secrets_prefix"`
}
```

### Observability Configuration

```go
type ObservabilityConfig struct {
    EnableMetrics  bool          `yaml:"enable_metrics"`
    EnableTracing  bool          `yaml:"enable_tracing"`
    EnableLogging  bool          `yaml:"enable_logging"`
    ServiceName    string        `yaml:"service_name"`
    ServiceVersion string        `yaml:"service_version"`
    Environment    string        `yaml:"environment"`
    Metrics        MetricsConfig `yaml:"metrics"`
}
```

## 🎯 Best Practices

### 1. **Use Functional Options for Development**

```go
// Development - use functional options for quick setup
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithDefaultTTL(5*time.Minute),
    cache.WithObservability(cache.ObservabilityConfig{
        EnableMetrics: true,
        EnableTracing: true,
        EnableLogging: true,
    }),
)
```

### 2. **Use YAML Files for Production**

```yaml
# production.yaml
redis:
  addr: "redis.production.com:6379"
  password: "${REDIS_PASSWORD}"
  tls:
    enabled: true

cache:
  default_ttl: "30m"
  circuit_breaker:
    threshold: 10
    timeout: "60s"

observability:
  enable_metrics: true
  enable_tracing: true
  service_name: "production-service"
  environment: "production"
```

### 3. **Use Environment Variables for Secrets**

```bash
# Never put secrets in YAML files
export CACHEX_REDIS_PASSWORD="your-secret-password"
export CACHEX_ENCRYPTION_KEY="your-secret-key"
```

### 4. **Enable Hot-Reload in Production**

```yaml
hot_reload:
  enabled: true
  check_interval: "60s"
  signal_reload: true
```

### 5. **Validate Configuration**

```go
// Always validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

## 🔍 Configuration Validation

The configuration system validates:

- **Redis Configuration**: Address must be specified
- **Cache Configuration**: TTL must be positive
- **Circuit Breaker**: Threshold must be positive
- **Rate Limit**: Requests per second must be positive
- **Pool Configuration**: Worker counts must be valid
- **Security Configuration**: Key lengths and sizes must be reasonable

## 🚀 Production Deployment

### Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.24-alpine

WORKDIR /app
COPY . .

RUN go build -o main .

# Copy configuration
COPY config.yaml /app/config.yaml

# Set environment variables
ENV CACHEX_REDIS_PASSWORD=${REDIS_PASSWORD}
ENV CACHEX_ENCRYPTION_KEY=${ENCRYPTION_KEY}

CMD ["./main"]
```

### Kubernetes Configuration

```yaml
# k8s-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cachex-config
data:
  config.yaml: |
    redis:
      addr: "redis-service:6379"
    cache:
      default_ttl: "30m"
    observability:
      enable_metrics: true
      service_name: "cachex-service"
---
apiVersion: v1
kind: Secret
metadata:
  name: cachex-secrets
data:
  redis-password: <base64-encoded-password>
  encryption-key: <base64-encoded-key>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cachex-app
spec:
  template:
    spec:
      containers:
      - name: cachex
        image: cachex:latest
        env:
        - name: CACHEX_REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: cachex-secrets
              key: redis-password
        - name: CACHEX_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: cachex-secrets
              key: encryption-key
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: cachex-config
```

## 🔧 Troubleshooting

### Common Issues

1. **Configuration File Not Found**
   ```bash
   # Check file path
   ls -la config.yaml
   
   # Use absolute path
   export CACHEX_CONFIG_FILE="/absolute/path/to/config.yaml"
   ```

2. **Environment Variables Not Loading**
   ```bash
   # Check environment variables
   env | grep CACHEX
   
   # Ensure proper prefix
   export CACHEX_REDIS_ADDR="localhost:6379"
   ```

3. **Hot-Reload Not Working**
   ```bash
   # Check file permissions
   chmod 644 config.yaml
   
   # Check hot-reload configuration
   hot_reload:
     enabled: true
     check_interval: "30s"
   ```

4. **Validation Errors**
   ```bash
   # Check configuration values
   # Ensure all required fields are set
   # Verify TTL values are positive
   ```

### Debug Configuration

```go
// Enable debug logging
log.SetLevel(log.DebugLevel)

// Print configuration
cfg, _ := config.LoadFromFile("config.yaml")
fmt.Printf("Configuration: %+v\n", cfg)
```

## ✅ Implementation Status

- ✅ **Functional Options**: Fully implemented with type-safe configuration
- ✅ **YAML Configuration**: Complete YAML file support
- ✅ **Environment Variables**: Full environment variable override support
- ✅ **Hot-Reload**: File monitoring and SIGHUP support
- ✅ **Configuration Validation**: Comprehensive validation rules
- ✅ **Production Ready**: Kubernetes and Docker deployment support

This configuration system provides comprehensive, flexible, and production-ready configuration management for Go-CacheX applications.
