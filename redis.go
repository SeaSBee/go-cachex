package cachex

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seasbee/go-logx"
)

// Store implements the cache.Store interface using Redis
type RedisStore struct {
	// Redis client
	client *redis.Client

	// Configuration
	config *RedisConfig

	// Statistics
	stats *RedisStats

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds Redis store configuration
type RedisConfig struct {
	// Redis connection settings
	Addr     string `yaml:"addr" json:"addr"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db" json:"db"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls" json:"tls"`

	// Connection pool settings
	PoolSize     int `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int `yaml:"max_retries" json:"max_retries"`

	// Timeout settings
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// Performance settings
	EnablePipelining bool `yaml:"enable_pipelining" json:"enable_pipelining"`
	EnableMetrics    bool `yaml:"enable_metrics" json:"enable_metrics"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool `yaml:"enabled" json:"enabled"`
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// DefaultRedisConfig returns a default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:             "localhost:6379",
		Password:         "",
		DB:               0,
		TLS:              &TLSConfig{Enabled: false},
		PoolSize:         10,
		MinIdleConns:     5,
		MaxRetries:       3,
		DialTimeout:      5 * time.Second,
		ReadTimeout:      3 * time.Second,
		WriteTimeout:     3 * time.Second,
		EnablePipelining: true,
		EnableMetrics:    true,
	}
}

// HighPerformanceRedisConfig returns a high-performance configuration
func HighPerformanceRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:             "localhost:6379",
		Password:         "",
		DB:               0,
		TLS:              &TLSConfig{Enabled: false},
		PoolSize:         50,
		MinIdleConns:     20,
		MaxRetries:       5,
		DialTimeout:      5 * time.Second,
		ReadTimeout:      1 * time.Second,
		WriteTimeout:     1 * time.Second,
		EnablePipelining: true,
		EnableMetrics:    true,
	}
}

// ProductionRedisConfig returns a production configuration
func ProductionRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:             "localhost:6379",
		Password:         "",
		DB:               0,
		TLS:              &TLSConfig{Enabled: true, InsecureSkipVerify: false},
		PoolSize:         100,
		MinIdleConns:     50,
		MaxRetries:       3,
		DialTimeout:      10 * time.Second,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
		EnablePipelining: true,
		EnableMetrics:    true,
	}
}

// Stats holds Redis store statistics
type RedisStats struct {
	Hits     int64
	Misses   int64
	Sets     int64
	Dels     int64
	Errors   int64
	BytesIn  int64
	BytesOut int64
	mu       sync.RWMutex
}

// NewRedisStore creates a new Redis store
func NewRedisStore(config *RedisConfig) (*RedisStore, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Redis client options
	opts := &redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	// Configure TLS if enabled
	if config.TLS != nil && config.TLS.Enabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test connection with timeout to avoid blocking
	connCtx, connCancel := context.WithTimeout(ctx, config.DialTimeout)
	defer connCancel()

	if err := client.Ping(connCtx).Err(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	store := &RedisStore{
		client: client,
		config: config,
		stats:  &RedisStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	logx.Info("Redis store created successfully",
		logx.String("addr", config.Addr),
		logx.Int("db", config.DB),
		logx.Int("pool_size", config.PoolSize))

	return store, nil
}

// Get retrieves a value from Redis (non-blocking)
func (s *RedisStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("get", time.Since(start), nil)
		}()

		redisResult, err := s.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				s.recordMiss()
				result <- AsyncResult{Exists: false}
				return
			}
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to get key %s: %w", key, err)}
			return
		}

		s.recordHit()
		s.recordBytesIn(int64(len(redisResult)))
		result <- AsyncResult{Value: []byte(redisResult), Exists: true}
	}()

	return result
}

// Set stores a value in Redis (non-blocking)
func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}
		if value == nil {
			result <- AsyncResult{Error: fmt.Errorf("value cannot be nil")}
			return
		}
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("set", time.Since(start), nil)
		}()

		err := s.client.Set(ctx, key, value, ttl).Err()
		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to set key %s: %w", key, err)}
			return
		}

		s.recordSet()
		s.recordBytesOut(int64(len(value)))
		result <- AsyncResult{}
	}()

	return result
}

// MGet retrieves multiple values from Redis (non-blocking)
func (s *RedisStore) MGet(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{Values: make(map[string][]byte)}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("mget", time.Since(start), nil)
		}()

		// Use pipelining for better performance
		var pipe redis.Pipeliner
		if s.config.EnablePipelining {
			pipe = s.client.Pipeline()
		}

		// Execute MGET
		var redisResult []interface{}
		var err error
		if pipe != nil {
			cmds := make([]*redis.StringCmd, len(keys))
			for i, key := range keys {
				cmds[i] = pipe.Get(ctx, key)
			}
			_, err = pipe.Exec(ctx)
			if err != nil && err != redis.Nil {
				s.recordError()
				result <- AsyncResult{Error: fmt.Errorf("failed to execute pipeline: %w", err)}
				return
			}

			// Extract results
			redisResult = make([]interface{}, len(keys))
			for i, cmd := range cmds {
				if cmd.Err() == redis.Nil {
					redisResult[i] = nil
				} else {
					redisResult[i], _ = cmd.Result()
				}
			}
		} else {
			redisResult, err = s.client.MGet(ctx, keys...).Result()
			if err != nil {
				s.recordError()
				result <- AsyncResult{Error: fmt.Errorf("failed to mget keys: %w", err)}
				return
			}
		}

		// Convert results to map
		results := make(map[string][]byte)
		for i, key := range keys {
			if redisResult[i] != nil {
				if str, ok := redisResult[i].(string); ok {
					results[key] = []byte(str)
					s.recordHit()
					s.recordBytesIn(int64(len(str)))
				}
			} else {
				s.recordMiss()
			}
		}

		result <- AsyncResult{Values: results}
	}()

	return result
}

// MSet stores multiple values in Redis (non-blocking)
func (s *RedisStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		if len(items) == 0 {
			result <- AsyncResult{}
			return
		}

		// Validate all keys and values
		for key, value := range items {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
			if value == nil {
				result <- AsyncResult{Error: fmt.Errorf("value cannot be nil for key: %s", key)}
				return
			}
		}

		start := time.Now()
		defer func() {
			s.recordOperation("mset", time.Since(start), nil)
		}()

		// Use pipelining for better performance
		var pipe redis.Pipeliner
		if s.config.EnablePipelining {
			pipe = s.client.Pipeline()
		}

		// Prepare items for Redis
		redisItems := make(map[string]interface{}, len(items))
		for key, value := range items {
			redisItems[key] = value
			s.recordBytesOut(int64(len(value)))
		}

		var err error
		if pipe != nil {
			// Use pipeline for better performance
			pipe.MSet(ctx, redisItems)
			if ttl > 0 {
				// Set TTL for each key
				for key := range items {
					pipe.Expire(ctx, key, ttl)
				}
			}
			_, err = pipe.Exec(ctx)
		} else {
			// Use regular MSET
			err = s.client.MSet(ctx, redisItems).Err()
			if err == nil && ttl > 0 {
				// Set TTL for each key
				for key := range items {
					s.client.Expire(ctx, key, ttl)
				}
			}
		}

		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to mset items: %w", err)}
			return
		}

		s.recordSet()
		result <- AsyncResult{}
	}()

	return result
}

// Del removes keys from Redis (non-blocking)
func (s *RedisStore) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		for _, key := range keys {
			if key == "" {
				result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("del", time.Since(start), nil)
		}()

		err := s.client.Del(ctx, keys...).Err()
		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to del keys: %w", err)}
			return
		}

		s.recordDel()
		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if a key exists in Redis (non-blocking)
func (s *RedisStore) Exists(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("exists", time.Since(start), nil)
		}()

		redisResult, err := s.client.Exists(ctx, key).Result()
		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to check existence of key %s: %w", key, err)}
			return
		}

		result <- AsyncResult{Exists: redisResult > 0}
	}()

	return result
}

// TTL returns the time to live for a key (non-blocking)
func (s *RedisStore) TTL(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("ttl", time.Since(start), nil)
		}()

		redisResult, err := s.client.TTL(ctx, key).Result()
		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to get TTL for key %s: %w", key, err)}
			return
		}

		// Redis returns -1 for keys without TTL, -2 for non-existent keys
		if redisResult == -2 {
			result <- AsyncResult{Exists: false}
		} else if redisResult == -1 {
			result <- AsyncResult{TTL: 0, Exists: true} // No TTL set
		} else {
			result <- AsyncResult{TTL: redisResult, Exists: true}
		}
	}()

	return result
}

// IncrBy increments a counter by the specified delta (non-blocking)
func (s *RedisStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Boundary condition validations
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}
		if ttlIfCreate < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		start := time.Now()
		defer func() {
			s.recordOperation("incrby", time.Since(start), nil)
		}()

		// Use pipeline to set TTL if creating new key
		var pipe redis.Pipeliner
		if ttlIfCreate > 0 {
			pipe = s.client.Pipeline()
		}

		var redisResult int64
		var err error

		if pipe != nil {
			// Use pipeline to set TTL for new keys
			incrCmd := pipe.IncrBy(ctx, key, delta)
			pipe.Expire(ctx, key, ttlIfCreate)
			_, err = pipe.Exec(ctx)
			if err == nil {
				redisResult, err = incrCmd.Result()
			}
		} else {
			redisResult, err = s.client.IncrBy(ctx, key, delta).Result()
		}

		if err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to increment key %s: %w", key, err)}
			return
		}

		result <- AsyncResult{Result: redisResult}
	}()

	return result
}

// Close closes the Redis store and releases resources
func (s *RedisStore) Close() error {
	s.cancel()
	if err := s.client.Close(); err != nil {
		logx.Error("Failed to close Redis client", logx.ErrorField(err))
		return err
	}

	logx.Info("Redis store closed successfully")
	return nil
}

// GetStats returns current statistics
func (s *RedisStore) GetStats() *RedisStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return &RedisStats{
		Hits:     s.stats.Hits,
		Misses:   s.stats.Misses,
		Sets:     s.stats.Sets,
		Dels:     s.stats.Dels,
		Errors:   s.stats.Errors,
		BytesIn:  s.stats.BytesIn,
		BytesOut: s.stats.BytesOut,
	}
}

// recordOperation records operation metrics
func (s *RedisStore) recordOperation(op string, duration time.Duration, err error) {
	if !s.config.EnableMetrics {
		return
	}

	// This would integrate with observability metrics
	// For now, just log the operation
	if err != nil {
		logx.Error("Redis operation failed",
			logx.String("operation", op),
			logx.String("duration", duration.String()),
			logx.ErrorField(err))
	}
}

// recordHit records a cache hit
func (s *RedisStore) recordHit() {
	s.stats.mu.Lock()
	s.stats.Hits++
	s.stats.mu.Unlock()
}

// recordMiss records a cache miss
func (s *RedisStore) recordMiss() {
	s.stats.mu.Lock()
	s.stats.Misses++
	s.stats.mu.Unlock()
}

// recordSet records a set operation
func (s *RedisStore) recordSet() {
	s.stats.mu.Lock()
	s.stats.Sets++
	s.stats.mu.Unlock()
}

// recordDel records a delete operation
func (s *RedisStore) recordDel() {
	s.stats.mu.Lock()
	s.stats.Dels++
	s.stats.mu.Unlock()
}

// recordError records an error
func (s *RedisStore) recordError() {
	s.stats.mu.Lock()
	s.stats.Errors++
	s.stats.mu.Unlock()
}

// recordBytesIn records bytes received
func (s *RedisStore) recordBytesIn(bytes int64) {
	s.stats.mu.Lock()
	s.stats.BytesIn += bytes
	s.stats.mu.Unlock()
}

// recordBytesOut records bytes sent
func (s *RedisStore) recordBytesOut(bytes int64) {
	s.stats.mu.Lock()
	s.stats.BytesOut += bytes
	s.stats.mu.Unlock()
}
