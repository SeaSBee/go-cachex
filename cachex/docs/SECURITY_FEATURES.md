# Security Features

Go-CacheX provides comprehensive security features to protect your cache operations and data.

## 🔒 Security Overview

The library implements enterprise-grade security features including:

- **TLS Encryption** for Redis connections
- **ACL/Auth Support** for Redis authentication
- **Input Validation** and safe deserialization
- **Secrets Management** with environment variable integration
- **Rate Limiting** with token bucket algorithm
- **Log Redaction** for sensitive data
- **Security Scanning** with govulncheck and staticcheck

## 🛡️ TLS Configuration

### Basic TLS Setup

```go
import "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"

// Create Redis store with TLS
redisStore, err := redisstore.New(&redisstore.Config{
    Addr:     "localhost:6379",
    Password: "your-password",
    DB:       0,
    TLSConfig: &redisstore.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: false, // ✅ Off by default for security
    },
})
```

### Security Features

- ✅ **TLS 1.2+ Support**: Secure encrypted connections
- ✅ **Certificate Validation**: InsecureSkipVerify off by default
- ✅ **Connection Pooling**: Secure connection management
- ✅ **Timeout Configuration**: Configurable connection timeouts

## 🔐 ACL/Auth Support

### Redis Authentication

```go
// Create Redis store with authentication
redisStore, err := redisstore.New(&redisstore.Config{
    Addr:     "localhost:6379",
    Password: "your-secure-password", // ✅ Username/password support
    DB:       0,
})
```

### Features

- ✅ **Password Authentication**: Secure Redis password
- ✅ **Database Selection**: Multi-database support
- ✅ **Connection Pooling**: Authenticated connection pools
- ✅ **Connection Management**: Automatic connection handling

## 🛡️ Input Validation

### Key and Value Validation

```go
// Create cache with security configuration
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithSecurity(cache.SecurityConfig{
        RedactLogs:    true,
        MaxKeyLength:  256,
        MaxValueSize:  1024 * 1024, // 1MB
    }),
)
```

### Security Protections

- ✅ **Size Limits**: Configurable key and value size limits
- ✅ **Input Sanitization**: Safe deserialization and validation
- ✅ **Pattern Validation**: Key format validation
- ✅ **Memory Protection**: Prevents memory exhaustion attacks

## 🔑 Secrets Management

### Environment Variable Integration

```bash
# Set environment variables for secrets
export CACHEX_REDIS_PASSWORD="secure-redis-password"
export CACHEX_ENCRYPTION_KEY="your-encryption-key"
```

### Features

- ✅ **Environment Variables**: Secrets loaded from environment
- ✅ **No Hardcoding**: Secrets never hardcoded in code
- ✅ **Error Handling**: Proper error handling for missing secrets
- ✅ **Security Best Practices**: Follows 12-factor app principles

## 🚦 Rate Limiting

### Token Bucket Rate Limiting

```go
// Create cache with rate limiting
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithRateLimit(cache.RateLimitConfig{
        RequestsPerSecond: 1000,
        Burst:             100,
    }),
)
```

### Features

- ✅ **Token Bucket Algorithm**: Efficient rate limiting
- ✅ **Burst Support**: Configurable burst capacity
- ✅ **Context Support**: Context-aware waiting
- ✅ **Thread-Safe**: Concurrent access support
- ✅ **Configurable**: Flexible rate limiting configuration

## 📝 Log Redaction

### Sensitive Data Redaction

```go
// Create cache with log redaction
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithSecurity(cache.SecurityConfig{
        RedactLogs: true, // Enable log redaction
    }),
)
```

### Features

- ✅ **Automatic Redaction**: Keys redacted as "[REDACTED]" when enabled
- ✅ **Configurable**: Can be enabled/disabled via SecurityConfig
- ✅ **Structured Logging**: Works with go-logx structured logging
- ✅ **Security Compliant**: No sensitive data in logs when enabled

## 🔍 Security Scanning

### CI/CD Integration

The library includes comprehensive security scanning in CI/CD:

```yaml
# .github/workflows/tests.yml
security:
  - name: Install govulncheck
    run: go install golang.org/x/vuln/cmd/govulncheck@latest
  - name: Run govulncheck
    run: govulncheck ./...

lint:
  - name: Install golangci-lint
    uses: golangci/golangci-lint-action@v3
  - name: Run golangci-lint
    run: golangci-lint run
```

### Security Tools

- ✅ **govulncheck**: Vulnerability scanning
- ✅ **golangci-lint**: Code quality and security
- ✅ **staticcheck**: Static analysis
- ✅ **gosec**: Security-focused linting
- ✅ **Race Detection**: Concurrent access detection

## 🚀 Usage Examples

### Complete Security Setup

```go
package main

import (
    "log"
    "time"

    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
    // 1. Create Redis store with TLS and auth
    redisStore, err := redisstore.New(&redisstore.Config{
        Addr:     "localhost:6379",
        Password: "secure-password", // From environment variable
        TLSConfig: &redisstore.TLSConfig{
            Enabled:            true,
            InsecureSkipVerify: false,
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create cache with comprehensive security
    c, err := cache.New[User](
        cache.WithStore(redisStore),
        cache.WithDefaultTTL(5*time.Minute),
        cache.WithSecurity(cache.SecurityConfig{
            RedactLogs:    true,  // Redact sensitive data in logs
            MaxKeyLength:  256,   // Limit key length
            MaxValueSize:  1024 * 1024, // 1MB value limit
        }),
        cache.WithRateLimit(cache.RateLimitConfig{
            RequestsPerSecond: 1000,
            Burst:             100,
        }),
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

    // 3. Use cache with security features
    user := User{ID: "123", Name: "John Doe", Email: "john@example.com"}
    
    // Cache user data (logs will show redacted keys if RedactLogs=true)
    err = c.Set("user:123", user, 10*time.Minute)
    if err != nil {
        log.Printf("Cache error: %v", err)
    }

    // Retrieve user data
    cachedUser, found, err := c.Get("user:123")
    if err != nil {
        log.Printf("Cache error: %v", err)
    } else if found {
        log.Printf("User found: %s", cachedUser.Name)
    }
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### Environment-Based Configuration

```bash
#!/bin/bash
# Set up environment variables for production

# Redis configuration
export CACHEX_REDIS_ADDR="redis.production.com:6379"
export CACHEX_REDIS_PASSWORD="your-secure-production-password"
export CACHEX_REDIS_TLS_ENABLED="true"

# Security configuration
export CACHEX_REDACT_LOGS="true"
export CACHEX_MAX_KEY_LENGTH="256"
export CACHEX_MAX_VALUE_SIZE="1048576"

# Rate limiting
export CACHEX_RATE_LIMIT_REQUESTS_PER_SECOND="1000"
export CACHEX_RATE_LIMIT_BURST="100"

# Observability
export CACHEX_ENABLE_METRICS="true"
export CACHEX_ENABLE_TRACING="true"
export CACHEX_ENABLE_LOGGING="true"
export CACHEX_SERVICE_NAME="production-service"
export CACHEX_ENVIRONMENT="production"
```

## 🔒 Security Best Practices

### 1. Always Use TLS
```go
// ✅ Good
TLSConfig: &redisstore.TLSConfig{
    Enabled:            true,
    InsecureSkipVerify: false,
}

// ❌ Bad
TLSConfig: &redisstore.TLSConfig{
    Enabled:            true,
    InsecureSkipVerify: true, // Never skip verification
}
```

### 2. Use Environment Variables for Secrets
```go
// ✅ Good
password := os.Getenv("CACHEX_REDIS_PASSWORD")

// ❌ Bad
password := "hardcoded-password"
```

### 3. Enable Log Redaction
```go
// ✅ Good
cache.WithSecurity(cache.SecurityConfig{
    RedactLogs: true,
})

// ❌ Bad
// No log redaction - sensitive data may appear in logs
```

### 4. Configure Rate Limiting
```go
// ✅ Good
cache.WithRateLimit(cache.RateLimitConfig{
    RequestsPerSecond: 1000,
    Burst:             100,
})

// ❌ Bad
// No rate limiting - potential for abuse
```

### 5. Set Appropriate Size Limits
```go
// ✅ Good
cache.WithSecurity(cache.SecurityConfig{
    MaxKeyLength:  256,
    MaxValueSize:  1024 * 1024, // 1MB
})

// ❌ Bad
// No size limits - potential for memory exhaustion
```

### 6. Use Strong Authentication
```go
// ✅ Good
redisstore.New(&redisstore.Config{
    Addr:     "localhost:6379",
    Password: "strong-complex-password",
    TLSConfig: &redisstore.TLSConfig{
        Enabled: true,
    },
})

// ❌ Bad
redisstore.New(&redisstore.Config{
    Addr: "localhost:6379",
    // No password, no TLS
})
```

## 🎯 Security Checklist

- [ ] TLS enabled for Redis connections
- [ ] Strong authentication credentials
- [ ] Input validation implemented
- [ ] Rate limiting configured
- [ ] Log redaction enabled
- [ ] Secrets loaded from environment
- [ ] Security scanning in CI/CD
- [ ] Regular security updates
- [ ] Access logging enabled
- [ ] Error handling without information leakage
- [ ] Size limits configured
- [ ] Proper error handling

## 🔗 Related Documentation

- [Configuration Management](./CONFIGURATION.md)
- [Observability Features](./OBSERVABILITY.md)
- [GORM Integration](./IMPLEMENTATION_DETAILS.md)
- [Circuit Breaker & Retry](./CIRCUIT_BREAKER_RETRY.md)
- [Worker Pool](./WORKER_POOL.md)
- [Local Memory Cache](./LOCAL_MEMORY_CACHE.md)

## 🆘 Support

For security-related issues or questions:

1. **Security Issues**: Report via GitHub Security Advisories
2. **General Questions**: Open GitHub Issues
3. **Documentation**: Check the docs folder
4. **Examples**: See the examples folder

## ✅ Implementation Status

- ✅ **TLS Encryption**: Full TLS support for Redis connections
- ✅ **ACL/Auth Support**: Redis authentication and authorization
- ✅ **Input Validation**: Comprehensive input validation and sanitization
- ✅ **Secrets Management**: Environment variable integration
- ✅ **Rate Limiting**: Token bucket rate limiting implementation
- ✅ **Log Redaction**: Automatic redaction of sensitive data
- ✅ **Security Scanning**: CI/CD integration with security tools
- ✅ **Thread Safety**: Race-free security operations
- ✅ **Production Ready**: Enterprise-grade security features

---

**Note**: This library provides security tools and patterns, but proper implementation and configuration is the responsibility of the application developer. Always follow security best practices and conduct regular security audits.
