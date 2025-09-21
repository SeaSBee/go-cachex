package cachex

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seasbee/go-validatorx"
)

// Config holds Redis store configuration
type RedisConfig struct {
	// Redis connection settings
	Addr     string `yaml:"addr" json:"addr" validate:"required,max:256" default:"localhost:6379"` // max 256 chars
	Password string `yaml:"password" json:"password" validate:"omitempty" default:""`              // optional
	DB       int    `yaml:"db" json:"db" validate:"gte:0,lte:15" default:"0"`                      // 0 to 15

	// Connection pool settings
	PoolSize     int `yaml:"pool_size" json:"pool_size" validate:"min:1,max:1000" default:"10"`  // 1 to 1000
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns" validate:"gte:0" default:"5"`  // 0 or more
	MaxRetries   int `yaml:"max_retries" json:"max_retries" validate:"gte:0,lte:10" default:"3"` // 0 to 10

	// Timeout settings
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout" validate:"min=100ms,max=5m" default:"5s"`   // 100ms to 5min
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout" validate:"min=100ms,max=5m" default:"3s"`   // 100ms to 5min
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout" validate:"min=100ms,max=5m" default:"3s"` // 100ms to 5min

	// Performance settings
	EnablePipelining bool `yaml:"enable_pipelining" json:"enable_pipelining"`
	EnableMetrics    bool `yaml:"enable_metrics" json:"enable_metrics"`

	// Health check settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" validate:"min=1s,max=2m" default:"30s"` // 1s to 2min
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" validate:"min=1s,max=2m" default:"5s"`    // 1s to 2min
}

// NewRedisConfigWithParams creates a new RedisConfig with all parameters as input
func NewRedisConfig(
	addr string,
	password string,
	db int,
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

// Validate validates the RedisConfig using go-validatorx
func (r *RedisConfig) Validate() *validatorx.ValidationResult {
	return validatorx.ValidateStruct(r)
}

// CreateRedisClient creates a Redis client from the RedisConfig
func (r *RedisConfig) CreateRedisClient() *redis.Client {
	options := &redis.Options{
		Addr:         r.Addr,
		Password:     r.Password,
		DB:           r.DB,
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
	if result := r.Validate(); !result.Valid {
		return nil, errors.New(result.Errors[0].Message)
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

	// Validate configuration
	if result := config.Validate(); !result.Valid {
		return nil, errors.New(result.Errors[0].Message)
	}

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
