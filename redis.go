package cachex

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"sync/atomic"
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

	// Statistics - using atomic operations for better performance
	stats *RedisStats

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Health check state - using proper mutex field
	lastHealthCheck  time.Time
	healthCheckMutex sync.RWMutex

	// Shutdown synchronization
	shutdownWg   sync.WaitGroup
	shutdownMu   sync.RWMutex
	shutdownFlag int32
}

// Config holds Redis store configuration
type RedisConfig struct {
	// Redis connection settings
	Addr     string `yaml:"addr" json:"addr" validate:"required,max:256"`
	Password string `yaml:"password" json:"password" validate:"omitempty"`
	DB       int    `yaml:"db" json:"db" validate:"gte:0,lte:15"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls" json:"tls" validate:"omitempty"`

	// Connection pool settings
	PoolSize     int `yaml:"pool_size" json:"pool_size" validate:"min:1,max:1000"`
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns" validate:"gte:0"`
	MaxRetries   int `yaml:"max_retries" json:"max_retries" validate:"gte:0,lte:10"`

	// Timeout settings
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout" validate:"min=100ms,max=5m"`   // 100ms to 5min
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout" validate:"min=100ms,max=5m"`   // 100ms to 5min
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout" validate:"min=100ms,max=5m"` // 100ms to 5min

	// Performance settings
	EnablePipelining bool `yaml:"enable_pipelining" json:"enable_pipelining"`
	EnableMetrics    bool `yaml:"enable_metrics" json:"enable_metrics"`

	// Health check settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" validate:"min=1s,max=2m"` // 1s to 2min
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" validate:"min=1s,max=2m"`   // 1s to 2min
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool `yaml:"enabled" json:"enabled"`
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// DefaultRedisConfig returns a default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:                "localhost:6379",
		Password:            "",
		DB:                  0,
		TLS:                 &TLSConfig{Enabled: false},
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}
}

// HighPerformanceRedisConfig returns a high-performance configuration
func HighPerformanceRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:                "localhost:6379",
		Password:            "",
		DB:                  0,
		TLS:                 &TLSConfig{Enabled: false},
		PoolSize:            50,
		MinIdleConns:        20,
		MaxRetries:          5,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         1 * time.Second,
		WriteTimeout:        1 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 15 * time.Second,
		HealthCheckTimeout:  2 * time.Second,
	}
}

// ProductionRedisConfig returns a production configuration
func ProductionRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:                "localhost:6379",
		Password:            "",
		DB:                  0,
		TLS:                 &TLSConfig{Enabled: true, InsecureSkipVerify: false},
		PoolSize:            100,
		MinIdleConns:        50,
		MaxRetries:          3,
		DialTimeout:         10 * time.Second,
		ReadTimeout:         5 * time.Second,
		WriteTimeout:        5 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 60 * time.Second,
		HealthCheckTimeout:  10 * time.Second,
	}
}

// Stats holds Redis store statistics using atomic operations
type RedisStats struct {
	Hits           int64
	Misses         int64
	Sets           int64
	Dels           int64
	Errors         int64
	BytesIn        int64
	BytesOut       int64
	HealthChecks   int64
	HealthFailures int64
}

// NewRedisStore creates a new Redis store
func NewRedisStore(config *RedisConfig) (*RedisStore, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	// Validate configuration
	if err := validateRedisConfig(config); err != nil {
		return nil, fmt.Errorf("invalid Redis configuration: %w", err)
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

	// Configure TLS if enabled - improved nil check
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
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", config.Addr, err)
	}

	store := &RedisStore{
		client: client,
		config: config,
		stats:  &RedisStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize health check state
	store.lastHealthCheck = time.Now()

	// Safe nil check for TLS in logging
	tlsEnabled := false
	if config.TLS != nil {
		tlsEnabled = config.TLS.Enabled
	}

	logx.Info("Redis store created successfully",
		logx.String("addr", config.Addr),
		logx.Int("db", config.DB),
		logx.Int("pool_size", config.PoolSize),
		logx.Bool("tls_enabled", tlsEnabled),
		logx.Bool("pipelining_enabled", config.EnablePipelining),
		logx.Bool("metrics_enabled", config.EnableMetrics))

	return store, nil
}

// validateRedisConfig validates Redis configuration
func validateRedisConfig(config *RedisConfig) error {
	if config.Addr == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if config.PoolSize <= 0 {
		return fmt.Errorf("pool size must be positive")
	}
	if config.MinIdleConns < 0 {
		return fmt.Errorf("min idle connections cannot be negative")
	}
	if config.MinIdleConns > config.PoolSize {
		return fmt.Errorf("min idle connections cannot exceed pool size")
	}
	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if config.DialTimeout <= 0 {
		return fmt.Errorf("dial timeout must be positive")
	}
	if config.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if config.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if config.HealthCheckInterval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}
	if config.HealthCheckTimeout <= 0 {
		return fmt.Errorf("health check timeout must be positive")
	}

	// Validate TLS configuration
	if config.TLS != nil && config.TLS.Enabled {
		// TLS is enabled, configuration is valid
	} else if config.TLS != nil && !config.TLS.Enabled {
		// TLS is explicitly disabled, this is fine
	} else if config.TLS == nil {
		// TLS is nil, create default disabled configuration
		config.TLS = &TLSConfig{Enabled: false}
	}

	return nil
}

// checkHealth performs a health check if needed with improved synchronization
func (s *RedisStore) checkHealth(ctx context.Context) error {
	// Check if we need a health check
	s.healthCheckMutex.RLock()
	lastCheck := s.lastHealthCheck
	needsCheck := time.Since(lastCheck) >= s.config.HealthCheckInterval
	s.healthCheckMutex.RUnlock()

	if !needsCheck {
		return nil // Health check not needed yet
	}

	// Use mutex to prevent concurrent health checks
	s.healthCheckMutex.Lock()
	defer s.healthCheckMutex.Unlock()

	// Double-check after acquiring lock
	lastCheck = s.lastHealthCheck
	if time.Since(lastCheck) < s.config.HealthCheckInterval {
		return nil
	}

	// Perform health check
	healthCtx, cancel := context.WithTimeout(ctx, s.config.HealthCheckTimeout)
	defer cancel()

	err := s.client.Ping(healthCtx).Err()
	if err != nil {
		atomic.AddInt64(&s.stats.HealthFailures, 1)
		logx.Error("Redis health check failed", logx.ErrorField(err))
		return fmt.Errorf("health check failed: %w", err)
	}

	atomic.AddInt64(&s.stats.HealthChecks, 1)
	s.lastHealthCheck = time.Now()
	return nil
}

// isShutdown checks if the store is shutting down
func (s *RedisStore) isShutdown() bool {
	return atomic.LoadInt32(&s.shutdownFlag) == 1
}

// Get retrieves a value from Redis (non-blocking)
func (s *RedisStore) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.ReadTimeout)
			defer cancel()
		}

		// Boundary condition validations
		if err := validateKey(key); err != nil {
			result <- AsyncResult{Error: err}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for get operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("get", time.Since(start), operationErr)
		}()

		redisResult, err := s.client.Get(timeoutCtx, key).Result()
		if err != nil {
			operationErr = err
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.WriteTimeout)
			defer cancel()
		}

		// Boundary condition validations
		if err := validateKey(key); err != nil {
			result <- AsyncResult{Error: err}
			return
		}
		if err := validateValue(value); err != nil {
			result <- AsyncResult{Error: err}
			return
		}
		if ttl < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for set operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("set", time.Since(start), operationErr)
		}()

		err := s.client.Set(timeoutCtx, key, value, ttl).Err()
		if err != nil {
			operationErr = err
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.ReadTimeout)
			defer cancel()
		}

		// Boundary condition validations
		for _, key := range keys {
			if err := validateKey(key); err != nil {
				result <- AsyncResult{Error: err}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{Values: make(map[string][]byte)}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for mget operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("mget", time.Since(start), operationErr)
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
				cmds[i] = pipe.Get(timeoutCtx, key)
			}

			// Execute pipeline with improved error handling
			_, err = pipe.Exec(timeoutCtx)
			if err != nil {
				operationErr = err
				s.recordError()
				result <- AsyncResult{Error: fmt.Errorf("failed to execute pipeline for keys %v: %w", keys, err)}
				return
			}

			// Extract results with improved error handling - allow partial failures
			redisResult = make([]interface{}, len(keys))
			var partialErrors []string
			for i, cmd := range cmds {
				if cmd.Err() == redis.Nil {
					redisResult[i] = nil
				} else if cmd.Err() != nil {
					// Record individual command errors but continue processing
					partialErrors = append(partialErrors, fmt.Sprintf("key %s: %v", keys[i], cmd.Err()))
					redisResult[i] = nil
					s.recordError()
				} else {
					redisResult[i], _ = cmd.Result()
				}
			}

			// Log partial errors if any occurred
			if len(partialErrors) > 0 {
				logx.Warn("MGet completed with partial failures",
					logx.String("errors", fmt.Sprintf("%v", partialErrors)))
			}
		} else {
			redisResult, err = s.client.MGet(timeoutCtx, keys...).Result()
			if err != nil {
				operationErr = err
				s.recordError()
				result <- AsyncResult{Error: fmt.Errorf("failed to mget keys %v: %w", keys, err)}
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.WriteTimeout)
			defer cancel()
		}

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
			if err := validateKey(key); err != nil {
				result <- AsyncResult{Error: err}
				return
			}
			if err := validateValue(value); err != nil {
				result <- AsyncResult{Error: fmt.Errorf("invalid value for key %s: %w", key, err)}
				return
			}
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for mset operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("mset", time.Since(start), operationErr)
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
			// Use pipeline for better performance with improved error handling
			pipe.MSet(timeoutCtx, redisItems)
			if ttl > 0 {
				// Set TTL for each key
				for key := range items {
					pipe.Expire(timeoutCtx, key, ttl)
				}
			}
			_, err = pipe.Exec(timeoutCtx)
			if err != nil {
				operationErr = err
				s.recordError()
				result <- AsyncResult{Error: fmt.Errorf("failed to mset items with keys %v: %w", getKeys(items), err)}
				return
			}
		} else {
			// Use regular MSET with improved TTL handling
			err = s.client.MSet(timeoutCtx, redisItems).Err()
			if err == nil && ttl > 0 {
				// Set TTL for each key with proper error handling
				var ttlErrors []string
				for key := range items {
					if ttlErr := s.client.Expire(timeoutCtx, key, ttl).Err(); ttlErr != nil {
						ttlErrors = append(ttlErrors, fmt.Sprintf("key %s: %v", key, ttlErr))
						s.recordError()
					}
				}

				// Log TTL errors but don't fail the entire operation
				if len(ttlErrors) > 0 {
					logx.Warn("MSet completed but some TTL operations failed",
						logx.String("ttl_errors", fmt.Sprintf("%v", ttlErrors)))
				}
			}
		}

		if err != nil {
			operationErr = err
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to mset items with keys %v: %w", getKeys(items), err)}
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.WriteTimeout)
			defer cancel()
		}

		// Boundary condition validations
		for _, key := range keys {
			if err := validateKey(key); err != nil {
				result <- AsyncResult{Error: err}
				return
			}
		}

		if len(keys) == 0 {
			result <- AsyncResult{}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for del operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("del", time.Since(start), operationErr)
		}()

		err := s.client.Del(timeoutCtx, keys...).Err()
		if err != nil {
			operationErr = err
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to del keys %v: %w", keys, err)}
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.ReadTimeout)
			defer cancel()
		}

		// Boundary condition validations
		if err := validateKey(key); err != nil {
			result <- AsyncResult{Error: err}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for exists operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("exists", time.Since(start), operationErr)
		}()

		redisResult, err := s.client.Exists(timeoutCtx, key).Result()
		if err != nil {
			operationErr = err
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

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.ReadTimeout)
			defer cancel()
		}

		// Boundary condition validations
		if err := validateKey(key); err != nil {
			result <- AsyncResult{Error: err}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for ttl operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("ttl", time.Since(start), operationErr)
		}()

		redisResult, err := s.client.TTL(timeoutCtx, key).Result()
		if err != nil {
			operationErr = err
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

// IncrBy increments a counter by the specified delta (non-blocking) with improved atomicity
func (s *RedisStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	// Check shutdown state
	if s.isShutdown() {
		result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
		close(result)
		return result
	}

	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		defer close(result)

		// Check context cancellation first
		select {
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
			return
		case <-s.ctx.Done():
			result <- AsyncResult{Error: fmt.Errorf("store is shutting down")}
			return
		default:
		}

		// Enhanced context handling with proper timeout
		var timeoutCtx context.Context
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); ok {
			// Context already has deadline, use it
			timeoutCtx = ctx
		} else {
			// Add timeout to context
			timeoutCtx, cancel = context.WithTimeout(ctx, s.config.WriteTimeout)
			defer cancel()
		}

		// Boundary condition validations
		if err := validateKey(key); err != nil {
			result <- AsyncResult{Error: err}
			return
		}
		if ttlIfCreate < 0 {
			result <- AsyncResult{Error: fmt.Errorf("ttl cannot be negative")}
			return
		}

		// Perform health check if needed
		if err := s.checkHealth(timeoutCtx); err != nil {
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("health check failed for incrby operation: %w", err)}
			return
		}

		start := time.Now()
		var operationErr error
		defer func() {
			s.recordOperation("incrby", time.Since(start), operationErr)
		}()

		var redisResult int64
		var err error

		if ttlIfCreate > 0 {
			// Use transaction for atomic INCRBY + EXPIRE operation
			txf := func(tx *redis.Tx) error {
				// Check if key exists
				exists, err := tx.Exists(timeoutCtx, key).Result()
				if err != nil {
					return err
				}

				// Increment the key
				result, err := tx.IncrBy(timeoutCtx, key, delta).Result()
				if err != nil {
					return err
				}
				redisResult = result

				// Set TTL only if key was newly created
				if exists == 0 {
					return tx.Expire(timeoutCtx, key, ttlIfCreate).Err()
				}
				return nil
			}

			// Execute transaction with retry logic
			err = s.client.Watch(timeoutCtx, txf, key)
		} else {
			// Simple increment without TTL
			redisResult, err = s.client.IncrBy(timeoutCtx, key, delta).Result()
		}

		if err != nil {
			operationErr = err
			s.recordError()
			result <- AsyncResult{Error: fmt.Errorf("failed to increment key %s: %w", key, err)}
			return
		}

		result <- AsyncResult{Result: redisResult}
	}()

	return result
}

// Close closes the Redis store and releases resources with proper cleanup
func (s *RedisStore) Close() error {
	// Mark as shutting down
	atomic.StoreInt32(&s.shutdownFlag, 1)

	// Cancel context to stop new operations
	s.cancel()

	// Wait for ongoing operations to complete with timeout
	done := make(chan struct{})
	go func() {
		s.shutdownWg.Wait()
		close(done)
	}()

	// Wait with timeout to prevent hanging
	select {
	case <-done:
		logx.Info("All operations completed, closing Redis store")
	case <-time.After(30 * time.Second):
		logx.Warn("Timeout waiting for operations to complete, forcing close")
	}

	// Close Redis client
	if err := s.client.Close(); err != nil {
		logx.Error("Failed to close Redis client", logx.ErrorField(err))
		return err
	}

	logx.Info("Redis store closed successfully")
	return nil
}

// GetStats returns current statistics
func (s *RedisStore) GetStats() *RedisStats {
	return &RedisStats{
		Hits:           atomic.LoadInt64(&s.stats.Hits),
		Misses:         atomic.LoadInt64(&s.stats.Misses),
		Sets:           atomic.LoadInt64(&s.stats.Sets),
		Dels:           atomic.LoadInt64(&s.stats.Dels),
		Errors:         atomic.LoadInt64(&s.stats.Errors),
		BytesIn:        atomic.LoadInt64(&s.stats.BytesIn),
		BytesOut:       atomic.LoadInt64(&s.stats.BytesOut),
		HealthChecks:   atomic.LoadInt64(&s.stats.HealthChecks),
		HealthFailures: atomic.LoadInt64(&s.stats.HealthFailures),
	}
}

// recordOperation records operation metrics with proper error handling
func (s *RedisStore) recordOperation(op string, duration time.Duration, err error) {
	if !s.config.EnableMetrics {
		return
	}

	// Enhanced observability with structured logging
	if err != nil {
		logx.Error("Redis operation failed",
			logx.String("operation", op),
			logx.String("duration", duration.String()),
			logx.String("addr", s.config.Addr),
			logx.Int("db", s.config.DB),
			logx.ErrorField(err))
	} else {
		// Log successful operations for debugging/monitoring
		logx.Debug("Redis operation completed",
			logx.String("operation", op),
			logx.String("duration", duration.String()),
			logx.String("addr", s.config.Addr),
			logx.Int("db", s.config.DB))
	}
}

// recordHit records a cache hit using atomic operation
func (s *RedisStore) recordHit() {
	atomic.AddInt64(&s.stats.Hits, 1)
}

// recordMiss records a cache miss using atomic operation
func (s *RedisStore) recordMiss() {
	atomic.AddInt64(&s.stats.Misses, 1)
}

// recordSet records a set operation using atomic operation
func (s *RedisStore) recordSet() {
	atomic.AddInt64(&s.stats.Sets, 1)
}

// recordDel records a delete operation using atomic operation
func (s *RedisStore) recordDel() {
	atomic.AddInt64(&s.stats.Dels, 1)
}

// recordError records an error using atomic operation
func (s *RedisStore) recordError() {
	atomic.AddInt64(&s.stats.Errors, 1)
}

// recordBytesIn records bytes received using atomic operation
func (s *RedisStore) recordBytesIn(bytes int64) {
	atomic.AddInt64(&s.stats.BytesIn, bytes)
}

// recordBytesOut records bytes sent using atomic operation
func (s *RedisStore) recordBytesOut(bytes int64) {
	atomic.AddInt64(&s.stats.BytesOut, bytes)
}

// getKeys extracts keys from a map for error messages
func getKeys(items map[string][]byte) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

// validateKey validates a Redis key
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 512 {
		return fmt.Errorf("key length cannot exceed 512 bytes")
	}
	return nil
}

// validateValue validates a Redis value
func validateValue(value []byte) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	if len(value) > 512*1024*1024 { // 512MB limit
		return fmt.Errorf("value size cannot exceed 512MB")
	}
	return nil
}
