package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/config"
	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Configuration Demo ===")

	// Example 1: Basic Configuration Loading
	demoBasicConfiguration()

	// Example 2: Environment Variable Override
	demoEnvironmentOverride()

	// Example 3: YAML Configuration File
	demoYAMLConfiguration()

	// Example 4: Hot-Reload Configuration
	demoHotReload()

	// Example 5: Functional Options
	demoFunctionalOptions()

	// Example 6: Namespace-Specific Configuration
	demoNamespaceConfiguration()

	fmt.Println("\n=== Configuration Demo Complete ===")
}

func demoBasicConfiguration() {
	fmt.Println("\n1. Basic Configuration Loading")
	fmt.Println("==============================")

	// Load default configuration
	cfg := config.DefaultConfig()
	fmt.Printf("✓ Default Redis address: %s\n", cfg.Redis.Addr)
	fmt.Printf("✓ Default TTL: %v\n", cfg.Cache.DefaultTTL)
	fmt.Printf("✓ Circuit breaker threshold: %d\n", cfg.Cache.CircuitBreaker.Threshold)
	fmt.Printf("✓ Rate limit: %.0f req/sec\n", cfg.Cache.RateLimit.RequestsPerSecond)
}

func demoEnvironmentOverride() {
	fmt.Println("\n2. Environment Variable Override")
	fmt.Println("=================================")

	// Set environment variables
	os.Setenv("CACHEX_REDIS_ADDR", "redis.example.com:6379")
	os.Setenv("CACHEX_DEFAULT_TTL", "10m")
	os.Setenv("CACHEX_CIRCUIT_BREAKER_THRESHOLD", "10")
	os.Setenv("CACHEX_ENABLE_METRICS", "true")
	os.Setenv("CACHEX_REDACT_LOGS", "true")

	// Load configuration with environment overrides
	cfg := config.DefaultConfig()
	config.LoadFromEnvironment(cfg)

	fmt.Printf("✓ Redis address (from env): %s\n", cfg.Redis.Addr)
	fmt.Printf("✓ TTL (from env): %v\n", cfg.Cache.DefaultTTL)
	fmt.Printf("✓ Circuit breaker threshold (from env): %d\n", cfg.Cache.CircuitBreaker.Threshold)
	fmt.Printf("✓ Metrics enabled (from env): %t\n", cfg.Observability.EnableMetrics)
	fmt.Printf("✓ Log redaction (from env): %t\n", cfg.Security.RedactLogs)

	// Clean up environment variables
	os.Unsetenv("CACHEX_REDIS_ADDR")
	os.Unsetenv("CACHEX_DEFAULT_TTL")
	os.Unsetenv("CACHEX_CIRCUIT_BREAKER_THRESHOLD")
	os.Unsetenv("CACHEX_ENABLE_METRICS")
	os.Unsetenv("CACHEX_REDACT_LOGS")
}

func demoYAMLConfiguration() {
	fmt.Println("\n3. YAML Configuration File")
	fmt.Println("===========================")

	// Create a sample YAML configuration file
	yamlConfig := `redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 20
  min_idle_conns: 10
  tls:
    enabled: false
  bulkhead:
    enabled: true
    read_pool_size: 30
    write_pool_size: 15

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

namespaces:
  user:
    default_ttl: "30m"
    max_size: 10000
    max_memory: 1073741824
  product:
    default_ttl: "1h"
    max_size: 5000
    max_memory: 536870912
  session:
    default_ttl: "24h"
    max_size: 1000
    max_memory: 134217728
`

	// Write YAML file
	if err := os.WriteFile("config.yaml", []byte(yamlConfig), 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
		return
	}
	defer os.Remove("config.yaml")

	// Load configuration from YAML file
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	fmt.Printf("✓ Redis pool size (from YAML): %d\n", cfg.Redis.PoolSize)
	fmt.Printf("✓ Cache TTL (from YAML): %v\n", cfg.Cache.DefaultTTL)
	fmt.Printf("✓ Circuit breaker threshold (from YAML): %d\n", cfg.Cache.CircuitBreaker.Threshold)
	fmt.Printf("✓ Rate limit (from YAML): %.0f req/sec\n", cfg.Cache.RateLimit.RequestsPerSecond)
	fmt.Printf("✓ Worker pool max workers (from YAML): %d\n", cfg.Pool.WorkerPool.MaxWorkers)
	fmt.Printf("✓ Pipeline batch size (from YAML): %d\n", cfg.Pool.Pipeline.BatchSize)
	fmt.Printf("✓ Service name (from YAML): %s\n", cfg.Observability.ServiceName)
	fmt.Printf("✓ Namespaces configured: %d\n", len(cfg.Namespaces))

	// Show namespace configurations
	for name, nsConfig := range cfg.Namespaces {
		fmt.Printf("  - %s: TTL=%v, MaxSize=%d, MaxMemory=%d bytes\n",
			name, nsConfig.DefaultTTL, nsConfig.MaxSize, nsConfig.MaxMemory)
	}
}

func demoHotReload() {
	fmt.Println("\n4. Hot-Reload Configuration")
	fmt.Println("============================")

	// Create a simple configuration file
	initialConfig := `redis:
  addr: "localhost:6379"
cache:
  default_ttl: "5m"
observability:
  service_name: "hot-reload-demo"
hot_reload:
  enabled: true
  check_interval: "5s"
  signal_reload: true
`

	if err := os.WriteFile("hot-reload.yaml", []byte(initialConfig), 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
		return
	}
	defer os.Remove("hot-reload.yaml")

	// Create configuration manager
	manager := config.NewManager("hot-reload.yaml")
	cfg, err := manager.Load()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	fmt.Printf("✓ Initial TTL: %v\n", cfg.Cache.DefaultTTL)
	fmt.Printf("✓ Initial service name: %s\n", cfg.Observability.ServiceName)

	// Register reload callback
	manager.OnReload(func(newConfig *config.Config) {
		fmt.Printf("🔄 Configuration reloaded! New TTL: %v, Service: %s\n",
			newConfig.Cache.DefaultTTL, newConfig.Observability.ServiceName)
	})

	// Start hot-reload monitoring
	if err := manager.StartHotReload(); err != nil {
		log.Printf("Failed to start hot-reload: %v", err)
		return
	}

	// Simulate configuration change after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		updatedConfig := `redis:
  addr: "localhost:6379"
cache:
  default_ttl: "10m"
observability:
  service_name: "hot-reload-demo-updated"
hot_reload:
  enabled: true
  check_interval: "5s"
  signal_reload: true
`
		if err := os.WriteFile("hot-reload.yaml", []byte(updatedConfig), 0644); err != nil {
			log.Printf("Failed to update config file: %v", err)
		}
	}()

	// Wait for reload
	time.Sleep(4 * time.Second)

	// Simulate SIGHUP signal
	go func() {
		time.Sleep(1 * time.Second)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP)
		sigChan <- syscall.SIGHUP
	}()

	// Wait for signal reload
	time.Sleep(2 * time.Second)

	manager.Stop()
	fmt.Println("✓ Hot-reload demo completed")
}

func demoFunctionalOptions() {
	fmt.Println("\n5. Functional Options")
	fmt.Println("=====================")

	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
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
			RedactLogs:         true,
			EnableTLS:          false,
			InsecureSkipVerify: false,
		}),
		cache.WithObservability(cache.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		}),
		cache.WithDeadLetterQueue(),
		cache.WithBloomFilter(),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	fmt.Println("✓ Cache created with functional options")
	fmt.Println("✓ All features enabled: circuit breaker, rate limiting, security, observability, DLQ, bloom filter")

	// Test cache operations
	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	if err := c.Set("user:123", user, 5*time.Minute); err != nil {
		log.Printf("Failed to set user: %v", err)
	} else {
		fmt.Println("✓ User set successfully")
	}

	if retrievedUser, found, err := c.Get("user:123"); err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Printf("✓ User retrieved: %s\n", retrievedUser.Name)
	}
}

func demoNamespaceConfiguration() {
	fmt.Println("\n6. Namespace-Specific Configuration")
	fmt.Println("====================================")

	// Load configuration with namespace-specific settings
	cfg := config.DefaultConfig()
	cfg.Namespaces = map[string]config.NamespaceConfig{
		"user": {
			DefaultTTL: 30 * time.Minute,
			MaxSize:    10000,
			MaxMemory:  1024 * 1024 * 1024, // 1GB
		},
		"product": {
			DefaultTTL: 2 * time.Hour,
			MaxSize:    5000,
			MaxMemory:  512 * 1024 * 1024, // 512MB
		},
		"session": {
			DefaultTTL: 24 * time.Hour,
			MaxSize:    1000,
			MaxMemory:  128 * 1024 * 1024, // 128MB
		},
		"cache": {
			DefaultTTL: 5 * time.Minute,
			MaxSize:    100,
			MaxMemory:  64 * 1024 * 1024, // 64MB
		},
	}

	fmt.Println("✓ Namespace-specific configurations:")
	for name, nsConfig := range cfg.Namespaces {
		fmt.Printf("  - %s: TTL=%v, MaxSize=%d, MaxMemory=%d MB\n",
			name, nsConfig.DefaultTTL, nsConfig.MaxSize, nsConfig.MaxMemory/(1024*1024))
	}

	// Demonstrate how to use namespace-specific TTLs
	fmt.Println("\n✓ Using namespace-specific TTLs:")
	userTTL := cfg.Namespaces["user"].DefaultTTL
	productTTL := cfg.Namespaces["product"].DefaultTTL
	sessionTTL := cfg.Namespaces["session"].DefaultTTL

	fmt.Printf("  - User data TTL: %v\n", userTTL)
	fmt.Printf("  - Product data TTL: %v\n", productTTL)
	fmt.Printf("  - Session data TTL: %v\n", sessionTTL)

	// Create cache with namespace-aware key building
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr: "localhost:6379",
		DB:   0,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
	}

	c, err := cache.New[User](
		cache.WithStore(redisStore),
		cache.WithDefaultTTL(cfg.Cache.DefaultTTL),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Demonstrate namespace-specific operations
	user := User{ID: "123", Name: "John Doe", Email: "john@example.com", CreatedAt: time.Now()}
	product := User{ID: "456", Name: "Product A", Email: "product@example.com", CreatedAt: time.Now()}

	// Use namespace-specific TTLs
	if err := c.Set("user:123", user, userTTL); err != nil {
		log.Printf("Failed to set user: %v", err)
	} else {
		fmt.Println("✓ User set with namespace-specific TTL")
	}

	if err := c.Set("product:456", product, productTTL); err != nil {
		log.Printf("Failed to set product: %v", err)
	} else {
		fmt.Println("✓ Product set with namespace-specific TTL")
	}
}
