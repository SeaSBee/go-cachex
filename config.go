package cachex

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis store configuration
type RedisConfig struct {
	// Redis connection settings
	Addr     string
	Password string
	DB       int
	Username string

	// Connection pool settings
	PoolSize     int
	MinIdleConns int
	MaxRetries   int

	// Timeout settings
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Performance settings
	EnablePipelining bool
	EnableMetrics    bool

	// Health check settings
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}

// NewRedisConfigWithParams creates a new RedisConfig with all parameters as input
func NewRedisConfig(
	addr string,
	password string,
	db int,
	username string,
	poolSize int,
	minIdleConns int,
	maxRetries int,
	dialTimeout time.Duration,
	readTimeout time.Duration,
	writeTimeout time.Duration,
	enablePipelining bool,
	enableMetrics bool,
	healthCheckInterval time.Duration,
	healthCheckTimeout time.Duration,
) *RedisConfig {
	return &RedisConfig{
		Addr:                addr,
		Password:            password,
		DB:                  db,
		Username:            username,
		PoolSize:            poolSize,
		MinIdleConns:        minIdleConns,
		MaxRetries:          maxRetries,
		DialTimeout:         dialTimeout,
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		EnablePipelining:    enablePipelining,
		EnableMetrics:       enableMetrics,
		HealthCheckInterval: healthCheckInterval,
		HealthCheckTimeout:  healthCheckTimeout,
	}
}

// Validate validates the RedisConfig
func (r *RedisConfig) Validate() error {
	if r.Addr == "" {
		return errors.New("address is required")
	}
	if r.DB < 0 {
		return errors.New("database number cannot be negative")
	}
	if r.DB > 15 {
		return errors.New("database number cannot exceed 15")
	}
	if r.PoolSize < 0 {
		return errors.New("pool size cannot be negative")
	}
	if r.PoolSize > 1000 {
		return errors.New("pool size cannot exceed 1000")
	}
	if r.MinIdleConns < 0 {
		return errors.New("min idle connections cannot be negative")
	}
	if r.MinIdleConns > r.PoolSize {
		return errors.New("min idle connections cannot exceed pool size")
	}
	if r.MaxRetries < 0 {
		return errors.New("max retries cannot be negative")
	}
	if r.MaxRetries > 10 {
		return errors.New("max retries cannot exceed 10")
	}
	if r.DialTimeout < 0 {
		return errors.New("dial timeout cannot be negative")
	}
	if r.ReadTimeout <= 0 {
		return errors.New("read timeout must be positive")
	}
	if r.WriteTimeout <= 0 {
		return errors.New("write timeout must be positive")
	}
	if r.HealthCheckInterval <= 0 {
		return errors.New("health check interval must be positive")
	}
	if r.HealthCheckTimeout <= 0 {
		return errors.New("health check timeout must be positive")
	}
	return nil
}

// CreateRedisClient creates a Redis client from the RedisConfig
func (r *RedisConfig) CreateRedisClient() *redis.Client {
	options := &redis.Options{
		Addr:         r.Addr,
		Password:     r.Password,
		DB:           r.DB,
		Username:     r.Username,
		PoolSize:     r.PoolSize,
		MinIdleConns: r.MinIdleConns,
		MaxRetries:   r.MaxRetries,
		DialTimeout:  r.DialTimeout,
		ReadTimeout:  r.ReadTimeout,
		WriteTimeout: r.WriteTimeout,
	}

	return redis.NewClient(options)
}

// ConnectRedisClient creates a Redis client and tests the connection
func (r *RedisConfig) ConnectRedisClient(ctx context.Context) (*redis.Client, error) {
	// Validate configuration first
	if err := r.Validate(); err != nil {
		return nil, err
	}

	// Create client
	client := r.CreateRedisClient()

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// CreateRedisCacheWithParams creates a Redis cache instance with all parameters as input
// This function takes all configuration parameters, creates a Redis client, and returns a cache instance
func CreateRedisCache(
	// Redis connection settings
	addr string,
	password string,
	db int,
	// Connection pool settings
	username string,
	poolSize int,
	minIdleConns int,
	maxRetries int,
	// Timeout settings
	dialTimeout time.Duration,
	readTimeout time.Duration,
	writeTimeout time.Duration,
	// Performance settings
	enablePipelining bool,
	enableMetrics bool,
	// Health check settings
	healthCheckInterval time.Duration,
	healthCheckTimeout time.Duration,
	// Optional components (can be nil for defaults)
	codec Codec,
	keyBuilder KeyBuilder,
	keyHasher KeyHasher,
) (Cache, error) {
	// Create Redis configuration
	config := NewRedisConfig(
		addr,
		password,
		db,
		username,
		poolSize,
		minIdleConns,
		maxRetries,
		dialTimeout,
		readTimeout,
		writeTimeout,
		enablePipelining,
		enableMetrics,
		healthCheckInterval,
		healthCheckTimeout,
	)

	// Create and test Redis client connection
	ctx := context.Background()
	client, err := config.ConnectRedisClient(ctx)
	if err != nil {
		return nil, err
	}

	// Create cache instance
	cache := NewRedisCache(client, codec, keyBuilder, keyHasher)
	return cache, nil
}
