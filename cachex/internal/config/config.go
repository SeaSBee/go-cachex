package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
	"gopkg.in/yaml.v3"
)

// Config holds the complete configuration for Go-CacheX
type Config struct {
	Redis         RedisConfig                `yaml:"redis"`
	Cache         CacheConfig                `yaml:"cache"`
	Pool          PoolConfig                 `yaml:"pool"`
	Security      SecurityConfig             `yaml:"security"`
	Observability ObservabilityConfig        `yaml:"observability"`
	PubSub        PubSubConfig               `yaml:"pubsub"`
	HotReload     HotReloadConfig            `yaml:"hot_reload"`
	Namespaces    map[string]NamespaceConfig `yaml:"namespaces"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr         string         `yaml:"addr"`
	Password     string         `yaml:"password"`
	DB           int            `yaml:"db"`
	ClusterAddrs []string       `yaml:"cluster_addrs"`
	TLS          TLSConfig      `yaml:"tls"`
	PoolSize     int            `yaml:"pool_size"`
	MinIdleConns int            `yaml:"min_idle_conns"`
	MaxRetries   int            `yaml:"max_retries"`
	DialTimeout  time.Duration  `yaml:"dial_timeout"`
	ReadTimeout  time.Duration  `yaml:"read_timeout"`
	WriteTimeout time.Duration  `yaml:"write_timeout"`
	PoolTimeout  time.Duration  `yaml:"pool_timeout"`
	Bulkhead     BulkheadConfig `yaml:"bulkhead"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool `yaml:"enabled"`
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

// BulkheadConfig holds bulkhead isolation configuration
type BulkheadConfig struct {
	Enabled           bool `yaml:"enabled"`
	ReadPoolSize      int  `yaml:"read_pool_size"`
	ReadMinIdleConns  int  `yaml:"read_min_idle_conns"`
	WritePoolSize     int  `yaml:"write_pool_size"`
	WriteMinIdleConns int  `yaml:"write_min_idle_conns"`
}

// CacheConfig holds cache-specific configuration
type CacheConfig struct {
	DefaultTTL            time.Duration        `yaml:"default_ttl"`
	CircuitBreaker        CircuitBreakerConfig `yaml:"circuit_breaker"`
	RateLimit             RateLimitConfig      `yaml:"rate_limit"`
	Retry                 RetryConfig          `yaml:"retry"`
	EnableDeadLetterQueue bool                 `yaml:"enable_dead_letter_queue"`
	EnableBloomFilter     bool                 `yaml:"enable_bloom_filter"`
}

// CircuitBreakerConfig holds circuit breaker settings
type CircuitBreakerConfig struct {
	Threshold   int           `yaml:"threshold"`
	Timeout     time.Duration `yaml:"timeout"`
	HalfOpenMax int           `yaml:"half_open_max"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

// RetryConfig holds retry settings
type RetryConfig struct {
	MaxRetries   int           `yaml:"max_retries"`
	InitialDelay time.Duration `yaml:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay"`
	Multiplier   float64       `yaml:"multiplier"`
	Jitter       bool          `yaml:"jitter"`
}

// PoolConfig holds worker pool and pipeline configuration
type PoolConfig struct {
	WorkerPool WorkerPoolConfig `yaml:"worker_pool"`
	Pipeline   PipelineConfig   `yaml:"pipeline"`
}

// WorkerPoolConfig holds worker pool settings
type WorkerPoolConfig struct {
	MinWorkers    int           `yaml:"min_workers"`
	MaxWorkers    int           `yaml:"max_workers"`
	QueueSize     int           `yaml:"queue_size"`
	IdleTimeout   time.Duration `yaml:"idle_timeout"`
	EnableMetrics bool          `yaml:"enable_metrics"`
}

// PipelineConfig holds pipeline settings
type PipelineConfig struct {
	BatchSize     int           `yaml:"batch_size"`
	MaxConcurrent int           `yaml:"max_concurrent"`
	BatchTimeout  time.Duration `yaml:"batch_timeout"`
	EnableMetrics bool          `yaml:"enable_metrics"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	EncryptionKey      string `yaml:"encryption_key"`
	EncryptionKeyFile  string `yaml:"encryption_key_file"`
	RedactLogs         bool   `yaml:"redact_logs"`
	EnableTLS          bool   `yaml:"enable_tls"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	MaxKeyLength       int    `yaml:"max_key_length"`
	MaxValueSize       int    `yaml:"max_value_size"`
	SecretsPrefix      string `yaml:"secrets_prefix"`
}

// ObservabilityConfig holds observability settings
type ObservabilityConfig struct {
	EnableMetrics  bool          `yaml:"enable_metrics"`
	EnableTracing  bool          `yaml:"enable_tracing"`
	EnableLogging  bool          `yaml:"enable_logging"`
	ServiceName    string        `yaml:"service_name"`
	ServiceVersion string        `yaml:"service_version"`
	Environment    string        `yaml:"environment"`
	Metrics        MetricsConfig `yaml:"metrics"`
}

// MetricsConfig holds metrics settings
type MetricsConfig struct {
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
	Enabled bool   `yaml:"enabled"`
}

// PubSubConfig holds pub/sub configuration
type PubSubConfig struct {
	Enabled             bool          `yaml:"enabled"`
	InvalidationChannel string        `yaml:"invalidation_channel"`
	HealthChannel       string        `yaml:"health_channel"`
	MaxSubscribers      int           `yaml:"max_subscribers"`
	MessageTimeout      time.Duration `yaml:"message_timeout"`
}

// HotReloadConfig holds hot-reload configuration
type HotReloadConfig struct {
	Enabled       bool          `yaml:"enabled"`
	ConfigFile    string        `yaml:"config_file"`
	CheckInterval time.Duration `yaml:"check_interval"`
	SignalReload  bool          `yaml:"signal_reload"`
}

// NamespaceConfig holds namespace-specific configuration
type NamespaceConfig struct {
	DefaultTTL time.Duration `yaml:"default_ttl"`
	MaxSize    int           `yaml:"max_size"`
	MaxMemory  int64         `yaml:"max_memory"`
}

// LoadFromFile loads configuration from YAML file
func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Override with environment variables
	LoadFromEnvironment(config)

	return config, nil
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment(config *Config) {
	// Redis configuration
	if addr := os.Getenv("CACHEX_REDIS_ADDR"); addr != "" {
		config.Redis.Addr = addr
	}
	if password := os.Getenv("CACHEX_REDIS_PASSWORD"); password != "" {
		config.Redis.Password = password
	}
	if db := os.Getenv("CACHEX_REDIS_DB"); db != "" {
		if dbInt, err := strconv.Atoi(db); err == nil {
			config.Redis.DB = dbInt
		}
	}
	if clusterAddrs := os.Getenv("CACHEX_REDIS_CLUSTER_ADDRS"); clusterAddrs != "" {
		config.Redis.ClusterAddrs = strings.Split(clusterAddrs, ",")
	}

	// TLS configuration
	if tlsEnabled := os.Getenv("CACHEX_REDIS_TLS_ENABLED"); tlsEnabled != "" {
		config.Redis.TLS.Enabled = tlsEnabled == "true"
	}
	if insecureSkipVerify := os.Getenv("CACHEX_REDIS_TLS_INSECURE_SKIP_VERIFY"); insecureSkipVerify != "" {
		config.Redis.TLS.InsecureSkipVerify = insecureSkipVerify == "true"
	}

	// Pool configuration
	if poolSize := os.Getenv("CACHEX_REDIS_POOL_SIZE"); poolSize != "" {
		if size, err := strconv.Atoi(poolSize); err == nil {
			config.Redis.PoolSize = size
		}
	}

	// Cache configuration
	if defaultTTL := os.Getenv("CACHEX_DEFAULT_TTL"); defaultTTL != "" {
		if ttl, err := time.ParseDuration(defaultTTL); err == nil {
			config.Cache.DefaultTTL = ttl
		}
	}
	if maxRetries := os.Getenv("CACHEX_MAX_RETRIES"); maxRetries != "" {
		if retries, err := strconv.Atoi(maxRetries); err == nil {
			config.Cache.Retry.MaxRetries = retries
		}
	}

	// Circuit breaker configuration
	if threshold := os.Getenv("CACHEX_CIRCUIT_BREAKER_THRESHOLD"); threshold != "" {
		if thresh, err := strconv.Atoi(threshold); err == nil {
			config.Cache.CircuitBreaker.Threshold = thresh
		}
	}

	// Security configuration
	if encryptionKey := os.Getenv("CACHEX_ENCRYPTION_KEY"); encryptionKey != "" {
		config.Security.EncryptionKey = encryptionKey
	}
	if redactLogs := os.Getenv("CACHEX_REDACT_LOGS"); redactLogs != "" {
		config.Security.RedactLogs = redactLogs == "true"
	}

	// Observability configuration
	if enableMetrics := os.Getenv("CACHEX_ENABLE_METRICS"); enableMetrics != "" {
		config.Observability.EnableMetrics = enableMetrics == "true"
	}
	if enableTracing := os.Getenv("CACHEX_ENABLE_TRACING"); enableTracing != "" {
		config.Observability.EnableTracing = enableTracing == "true"
	}
	if enableLogging := os.Getenv("CACHEX_ENABLE_LOGGING"); enableLogging != "" {
		config.Observability.EnableLogging = enableLogging == "true"
	}
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolTimeout:  4 * time.Second,
			Bulkhead: BulkheadConfig{
				Enabled:           false,
				ReadPoolSize:      20,
				ReadMinIdleConns:  10,
				WritePoolSize:     10,
				WriteMinIdleConns: 5,
			},
		},
		Cache: CacheConfig{
			DefaultTTL: 5 * time.Minute,
			CircuitBreaker: CircuitBreakerConfig{
				Threshold:   5,
				Timeout:     30 * time.Second,
				HalfOpenMax: 3,
			},
			RateLimit: RateLimitConfig{
				RequestsPerSecond: 1000,
				Burst:             100,
			},
			Retry: RetryConfig{
				MaxRetries:   3,
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				Jitter:       true,
			},
			EnableDeadLetterQueue: false,
			EnableBloomFilter:     false,
		},
		Pool: PoolConfig{
			WorkerPool: WorkerPoolConfig{
				MinWorkers:    5,
				MaxWorkers:    20,
				QueueSize:     1000,
				IdleTimeout:   30 * time.Second,
				EnableMetrics: true,
			},
			Pipeline: PipelineConfig{
				BatchSize:     100,
				MaxConcurrent: 10,
				BatchTimeout:  100 * time.Millisecond,
				EnableMetrics: true,
			},
		},
		Security: SecurityConfig{
			RedactLogs:    true,
			MaxKeyLength:  256,
			MaxValueSize:  1024 * 1024, // 1MB
			SecretsPrefix: "VAULT_",
		},
		Observability: ObservabilityConfig{
			EnableMetrics:  true,
			EnableTracing:  true,
			EnableLogging:  true,
			ServiceName:    "cachex",
			ServiceVersion: "1.0.0",
			Environment:    "production",
			Metrics: MetricsConfig{
				Port:    8080,
				Path:    "/metrics",
				Enabled: true,
			},
		},
		PubSub: PubSubConfig{
			Enabled:             false,
			InvalidationChannel: "cachex:invalidation",
			HealthChannel:       "cachex:health",
			MaxSubscribers:      100,
			MessageTimeout:      5 * time.Second,
		},
		HotReload: HotReloadConfig{
			Enabled:       false,
			CheckInterval: 30 * time.Second,
			SignalReload:  true,
		},
		Namespaces: make(map[string]NamespaceConfig),
	}
}

// ToCacheOptions converts configuration to cache options
func (c *Config) ToCacheOptions() []cache.Option {
	var options []cache.Option

	// Set default TTL
	options = append(options, cache.WithDefaultTTL(c.Cache.DefaultTTL))

	// Set circuit breaker
	options = append(options, cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
		Threshold:   c.Cache.CircuitBreaker.Threshold,
		Timeout:     c.Cache.CircuitBreaker.Timeout,
		HalfOpenMax: c.Cache.CircuitBreaker.HalfOpenMax,
	}))

	// Set rate limit
	options = append(options, cache.WithRateLimit(cache.RateLimitConfig{
		RequestsPerSecond: c.Cache.RateLimit.RequestsPerSecond,
		Burst:             c.Cache.RateLimit.Burst,
	}))

	// Set security
	options = append(options, cache.WithSecurity(cache.SecurityConfig{
		EncryptionKey:      c.Security.EncryptionKey,
		RedactLogs:         c.Security.RedactLogs,
		EnableTLS:          c.Security.EnableTLS,
		InsecureSkipVerify: c.Security.InsecureSkipVerify,
	}))

	// Set observability
	options = append(options, cache.WithObservability(cache.ObservabilityConfig{
		EnableMetrics: c.Observability.EnableMetrics,
		EnableTracing: c.Observability.EnableTracing,
		EnableLogging: c.Observability.EnableLogging,
	}))

	// Set retry policy
	options = append(options, cache.WithMaxRetries(c.Cache.Retry.MaxRetries))

	// Enable features
	if c.Cache.EnableDeadLetterQueue {
		options = append(options, cache.WithDeadLetterQueue())
	}
	if c.Cache.EnableBloomFilter {
		options = append(options, cache.WithBloomFilter())
	}

	return options
}

// ToRedisConfig converts configuration to Redis store configuration
func (c *Config) ToRedisConfig() *redisstore.Config {
	config := &redisstore.Config{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		PoolSize:     c.Redis.PoolSize,
		MinIdleConns: c.Redis.MinIdleConns,
		MaxRetries:   c.Redis.MaxRetries,
		DialTimeout:  c.Redis.DialTimeout,
		ReadTimeout:  c.Redis.ReadTimeout,
		WriteTimeout: c.Redis.WriteTimeout,
		PoolTimeout:  c.Redis.PoolTimeout,
	}

	if c.Redis.TLS.Enabled {
		config.TLSConfig = &redisstore.TLSConfig{
			Enabled:            c.Redis.TLS.Enabled,
			InsecureSkipVerify: c.Redis.TLS.InsecureSkipVerify,
		}
	}

	return config
}

// ToBulkheadConfig converts configuration to bulkhead configuration
func (c *Config) ToBulkheadConfig() redisstore.BulkheadConfig {
	return redisstore.BulkheadConfig{
		Addr:              c.Redis.Addr,
		Password:          c.Redis.Password,
		DB:                c.Redis.DB,
		ReadPoolSize:      c.Redis.Bulkhead.ReadPoolSize,
		ReadMinIdleConns:  c.Redis.Bulkhead.ReadMinIdleConns,
		WritePoolSize:     c.Redis.Bulkhead.WritePoolSize,
		WriteMinIdleConns: c.Redis.Bulkhead.WriteMinIdleConns,
		TLSConfig: &redisstore.TLSConfig{
			Enabled:            c.Redis.TLS.Enabled,
			InsecureSkipVerify: c.Redis.TLS.InsecureSkipVerify,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Redis configuration
	if len(c.Redis.ClusterAddrs) == 0 && c.Redis.Addr == "" {
		return fmt.Errorf("either Redis addr or cluster_addrs must be specified")
	}

	// Validate cache configuration
	if c.Cache.DefaultTTL <= 0 {
		return fmt.Errorf("default_ttl must be positive")
	}

	// Validate circuit breaker configuration
	if c.Cache.CircuitBreaker.Threshold <= 0 {
		return fmt.Errorf("circuit_breaker.threshold must be positive")
	}

	// Validate rate limit configuration
	if c.Cache.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("rate_limit.requests_per_second must be positive")
	}

	return nil
}
