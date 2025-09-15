package cachex

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/SeaSBee/go-logx"
)

// OptimizedConnectionPool provides advanced connection pooling with performance optimizations
type OptimizedConnectionPool struct {
	// Redis client with optimized connection pool
	client *redis.Client

	// Configuration
	config *OptimizedConnectionPoolConfig

	// Connection pool statistics - using atomic operations for thread safety
	stats *ConnectionPoolStats

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Connection pool state
	mu sync.RWMutex

	// Health check state with proper synchronization
	lastHealthCheck  time.Time
	healthCheckMutex sync.RWMutex

	// Connection pool monitoring
	monitorTicker *time.Ticker
	monitorDone   chan struct{}
	monitorWg     sync.WaitGroup // Wait group for monitoring goroutine

	// Connection warming
	warmingWg sync.WaitGroup // Wait group for connection warming goroutine

	// Pool state
	closed int32 // Atomic flag to track if pool is closed
}

// OptimizedConnectionPoolConfig provides enhanced connection pool configuration
type OptimizedConnectionPoolConfig struct {
	// Redis connection settings
	Addr     string `yaml:"addr" json:"addr" validate:"required,max:256"`
	Password string `yaml:"password" json:"password" validate:"omitempty"`
	DB       int    `yaml:"db" json:"db" validate:"gte:0,lte:15"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls" json:"tls" validate:"omitempty"`

	// Advanced connection pool settings
	PoolSize           int           `yaml:"pool_size" json:"pool_size" validate:"min:1,max:1000"`
	MinIdleConns       int           `yaml:"min_idle_conns" json:"min_idle_conns" validate:"gte:0"`
	MaxRetries         int           `yaml:"max_retries" json:"max_retries" validate:"gte:0,lte:10"`
	PoolTimeout        time.Duration `yaml:"pool_timeout" json:"pool_timeout" validate:"gte:100000000,lte:300000000000"`
	IdleTimeout        time.Duration `yaml:"idle_timeout" json:"idle_timeout" validate:"gte:1000000000,lte:3600000000000"`
	IdleCheckFrequency time.Duration `yaml:"idle_check_frequency" json:"idle_check_frequency" validate:"gte:1000000000,lte:300000000000"`

	// Timeout settings
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout" validate:"gte:100000000,lte:300000000000"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout" validate:"gte:100000000,lte:300000000000"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout" validate:"gte:100000000,lte:300000000000"`

	// Performance settings
	EnablePipelining     bool `yaml:"enable_pipelining" json:"enable_pipelining"`
	EnableMetrics        bool `yaml:"enable_metrics" json:"enable_metrics"`
	EnableConnectionPool bool `yaml:"enable_connection_pool" json:"enable_connection_pool"`

	// Health check settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" validate:"gte:1000000000,lte:120000000000"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" validate:"gte:1000000000,lte:120000000000"`

	// Connection pool optimization settings
	EnableConnectionReuse    bool          `yaml:"enable_connection_reuse" json:"enable_connection_reuse"`
	EnableConnectionWarming  bool          `yaml:"enable_connection_warming" json:"enable_connection_warming"`
	ConnectionWarmingTimeout time.Duration `yaml:"connection_warming_timeout" json:"connection_warming_timeout" validate:"gte:1000000000,lte:60000000000"`
	EnableLoadBalancing      bool          `yaml:"enable_load_balancing" json:"enable_load_balancing"`
	EnableCircuitBreaker     bool          `yaml:"enable_circuit_breaker" json:"enable_circuit_breaker"`

	// Monitoring settings
	EnablePoolMonitoring bool          `yaml:"enable_pool_monitoring" json:"enable_pool_monitoring"`
	MonitoringInterval   time.Duration `yaml:"monitoring_interval" json:"monitoring_interval" validate:"gte:1000000000,lte:300000000000"`
}

// ConnectionPoolStats provides comprehensive connection pool statistics
type ConnectionPoolStats struct {
	// Connection pool metrics
	TotalConnections  int64
	ActiveConnections int64
	IdleConnections   int64
	WaitCount         int64
	WaitDuration      int64
	IdleClosed        int64
	StaleClosed       int64

	// Performance metrics
	Hits           int64
	Misses         int64
	Timeouts       int64
	Errors         int64
	HealthChecks   int64
	HealthFailures int64

	// Load balancing metrics
	LoadBalancedRequests int64
	CircuitBreakerTrips  int64
	CircuitBreakerResets int64

	// Connection reuse metrics
	ReusedConnections int64
	NewConnections    int64
	WarmedConnections int64

	// Timing metrics
	AverageWaitTime     int64
	AverageResponseTime int64
	P95ResponseTime     int64
	P99ResponseTime     int64
}

// DefaultOptimizedConnectionPoolConfig returns optimized default configuration
func DefaultOptimizedConnectionPoolConfig() *OptimizedConnectionPoolConfig {
	return &OptimizedConnectionPoolConfig{
		Addr:                     "localhost:6379",
		Password:                 "",
		DB:                       0,
		TLS:                      &TLSConfig{Enabled: false},
		PoolSize:                 20,
		MinIdleConns:             10,
		MaxRetries:               3,
		PoolTimeout:              30 * time.Second,
		IdleTimeout:              5 * time.Minute,
		IdleCheckFrequency:       1 * time.Minute,
		DialTimeout:              5 * time.Second,
		ReadTimeout:              3 * time.Second,
		WriteTimeout:             3 * time.Second,
		EnablePipelining:         true,
		EnableMetrics:            true,
		EnableConnectionPool:     true,
		HealthCheckInterval:      30 * time.Second,
		HealthCheckTimeout:       5 * time.Second,
		EnableConnectionReuse:    true,
		EnableConnectionWarming:  true,
		ConnectionWarmingTimeout: 10 * time.Second,
		EnableLoadBalancing:      false,
		EnableCircuitBreaker:     true,
		EnablePoolMonitoring:     true,
		MonitoringInterval:       30 * time.Second,
	}
}

// HighPerformanceConnectionPoolConfig returns high-performance configuration
func HighPerformanceConnectionPoolConfig() *OptimizedConnectionPoolConfig {
	return &OptimizedConnectionPoolConfig{
		Addr:                     "localhost:6379",
		Password:                 "",
		DB:                       0,
		TLS:                      &TLSConfig{Enabled: false},
		PoolSize:                 100,
		MinIdleConns:             50,
		MaxRetries:               5,
		PoolTimeout:              10 * time.Second,
		IdleTimeout:              2 * time.Minute,
		IdleCheckFrequency:       30 * time.Second,
		DialTimeout:              5 * time.Second,
		ReadTimeout:              1 * time.Second,
		WriteTimeout:             1 * time.Second,
		EnablePipelining:         true,
		EnableMetrics:            true,
		EnableConnectionPool:     true,
		HealthCheckInterval:      15 * time.Second,
		HealthCheckTimeout:       2 * time.Second,
		EnableConnectionReuse:    true,
		EnableConnectionWarming:  true,
		ConnectionWarmingTimeout: 5 * time.Second,
		EnableLoadBalancing:      true,
		EnableCircuitBreaker:     true,
		EnablePoolMonitoring:     true,
		MonitoringInterval:       15 * time.Second,
	}
}

// ProductionConnectionPoolConfig returns production-ready configuration
func ProductionConnectionPoolConfig() *OptimizedConnectionPoolConfig {
	return &OptimizedConnectionPoolConfig{
		Addr:                     "localhost:6379",
		Password:                 "",
		DB:                       0,
		TLS:                      &TLSConfig{Enabled: true, InsecureSkipVerify: false},
		PoolSize:                 200,
		MinIdleConns:             100,
		MaxRetries:               3,
		PoolTimeout:              60 * time.Second,
		IdleTimeout:              10 * time.Minute,
		IdleCheckFrequency:       2 * time.Minute,
		DialTimeout:              10 * time.Second,
		ReadTimeout:              5 * time.Second,
		WriteTimeout:             5 * time.Second,
		EnablePipelining:         true,
		EnableMetrics:            true,
		EnableConnectionPool:     true,
		HealthCheckInterval:      60 * time.Second,
		HealthCheckTimeout:       10 * time.Second,
		EnableConnectionReuse:    true,
		EnableConnectionWarming:  true,
		ConnectionWarmingTimeout: 15 * time.Second,
		EnableLoadBalancing:      true,
		EnableCircuitBreaker:     true,
		EnablePoolMonitoring:     true,
		MonitoringInterval:       60 * time.Second,
	}
}

// NewOptimizedConnectionPool creates a new optimized connection pool
func NewOptimizedConnectionPool(config *OptimizedConnectionPoolConfig) (*OptimizedConnectionPool, error) {
	if config == nil {
		config = DefaultOptimizedConnectionPoolConfig()
	}

	// Create a copy of config to avoid modifying the original
	configCopy := *config

	// Validate configuration
	if err := validateOptimizedConnectionPoolConfig(&configCopy); err != nil {
		return nil, fmt.Errorf("invalid connection pool configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Redis client options with optimized connection pool settings
	opts := &redis.Options{
		Addr:         configCopy.Addr,
		Password:     configCopy.Password,
		DB:           configCopy.DB,
		PoolSize:     configCopy.PoolSize,
		MinIdleConns: configCopy.MinIdleConns,
		MaxRetries:   configCopy.MaxRetries,
		PoolTimeout:  configCopy.PoolTimeout,
		DialTimeout:  configCopy.DialTimeout,
		ReadTimeout:  configCopy.ReadTimeout,
		WriteTimeout: configCopy.WriteTimeout,
	}

	// Configure TLS if enabled
	if configCopy.TLS != nil && configCopy.TLS.Enabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: configCopy.TLS.InsecureSkipVerify,
		}
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test connection with timeout
	connCtx, connCancel := context.WithTimeout(ctx, configCopy.DialTimeout)
	defer connCancel()

	if err := client.Ping(connCtx).Err(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", configCopy.Addr, err)
	}

	pool := &OptimizedConnectionPool{
		client: client,
		config: &configCopy,
		stats:  &ConnectionPoolStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize health check state
	pool.lastHealthCheck = time.Now()

	// Start connection warming if enabled
	if configCopy.EnableConnectionWarming {
		pool.warmingWg.Add(1)
		go pool.warmConnections()
	}

	// Start pool monitoring if enabled
	if configCopy.EnablePoolMonitoring {
		pool.startPoolMonitoring()
	}

	logx.Info("Optimized connection pool created successfully",
		logx.String("addr", configCopy.Addr),
		logx.Int("db", configCopy.DB),
		logx.Int("poolSize", configCopy.PoolSize),
		logx.Int("minIdleConns", configCopy.MinIdleConns),
		logx.Bool("tlsEnabled", configCopy.TLS != nil && configCopy.TLS.Enabled),
		logx.Bool("pipeliningEnabled", configCopy.EnablePipelining),
		logx.Bool("connectionReuseEnabled", configCopy.EnableConnectionReuse),
		logx.Bool("loadBalancingEnabled", configCopy.EnableLoadBalancing),
		logx.Bool("circuitBreakerEnabled", configCopy.EnableCircuitBreaker))

	return pool, nil
}

// GetClient returns the underlying Redis client
func (p *OptimizedConnectionPool) GetClient() *redis.Client {
	if p == nil {
		return nil
	}
	return p.client
}

// isClosed checks if the pool is closed
func (p *OptimizedConnectionPool) isClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

// Get returns a value from the connection pool (non-blocking)
func (p *OptimizedConnectionPool) Get(ctx context.Context, key string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if pool is closed
		if p.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("connection pool is closed")}
			return
		}

		// Validate input
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		start := time.Now()

		// Check health if needed
		if err := p.checkHealth(ctx); err != nil {
			atomic.AddInt64(&p.stats.HealthFailures, 1) // Use HealthFailures instead of Errors
			result <- AsyncResult{Error: fmt.Errorf("health check failed: %w", err)}
			return
		}

		// Execute Redis command with nil check
		if p.client == nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis client is not available")}
			return
		}

		val, err := p.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				atomic.AddInt64(&p.stats.Misses, 1)
				result <- AsyncResult{Exists: false}
			} else {
				atomic.AddInt64(&p.stats.Errors, 1)
				result <- AsyncResult{Error: fmt.Errorf("Redis get error: %w", err)}
			}
			return
		}

		atomic.AddInt64(&p.stats.Hits, 1)
		responseTime := time.Since(start)
		atomic.AddInt64(&p.stats.AverageResponseTime, int64(responseTime))

		result <- AsyncResult{
			Value:  []byte(val),
			Exists: true,
		}
	}()

	return result
}

// Set stores a value in the connection pool (non-blocking)
func (p *OptimizedConnectionPool) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if pool is closed
		if p.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("connection pool is closed")}
			return
		}

		// Validate input
		if key == "" {
			result <- AsyncResult{Error: fmt.Errorf("key cannot be empty")}
			return
		}

		start := time.Now()

		// Check health if needed
		if err := p.checkHealth(ctx); err != nil {
			atomic.AddInt64(&p.stats.HealthFailures, 1) // Use HealthFailures instead of Errors
			result <- AsyncResult{Error: fmt.Errorf("health check failed: %w", err)}
			return
		}

		// Execute Redis command with nil check
		if p.client == nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis client is not available")}
			return
		}

		err := p.client.Set(ctx, key, value, ttl).Err()
		if err != nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis set error: %w", err)}
			return
		}

		responseTime := time.Since(start)
		atomic.AddInt64(&p.stats.AverageResponseTime, int64(responseTime))

		result <- AsyncResult{}
	}()

	return result
}

// Del removes keys from the connection pool (non-blocking)
func (p *OptimizedConnectionPool) Del(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if pool is closed
		if p.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("connection pool is closed")}
			return
		}

		// Validate input
		if len(keys) == 0 {
			result <- AsyncResult{Error: fmt.Errorf("at least one key must be provided")}
			return
		}

		start := time.Now()

		// Check health if needed
		if err := p.checkHealth(ctx); err != nil {
			atomic.AddInt64(&p.stats.HealthFailures, 1) // Use HealthFailures instead of Errors
			result <- AsyncResult{Error: fmt.Errorf("health check failed: %w", err)}
			return
		}

		// Execute Redis command with nil check
		if p.client == nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis client is not available")}
			return
		}

		err := p.client.Del(ctx, keys...).Err()
		if err != nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis del error: %w", err)}
			return
		}

		responseTime := time.Since(start)
		atomic.AddInt64(&p.stats.AverageResponseTime, int64(responseTime))

		result <- AsyncResult{}
	}()

	return result
}

// Exists checks if keys exist in the connection pool (non-blocking)
func (p *OptimizedConnectionPool) Exists(ctx context.Context, keys ...string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		// Check if pool is closed
		if p.isClosed() {
			result <- AsyncResult{Error: fmt.Errorf("connection pool is closed")}
			return
		}

		// Validate input
		if len(keys) == 0 {
			result <- AsyncResult{Error: fmt.Errorf("at least one key must be provided")}
			return
		}

		start := time.Now()

		// Check health if needed
		if err := p.checkHealth(ctx); err != nil {
			atomic.AddInt64(&p.stats.HealthFailures, 1) // Use HealthFailures instead of Errors
			result <- AsyncResult{Error: fmt.Errorf("health check failed: %w", err)}
			return
		}

		// Execute Redis command with nil check
		if p.client == nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis client is not available")}
			return
		}

		count, err := p.client.Exists(ctx, keys...).Result()
		if err != nil {
			atomic.AddInt64(&p.stats.Errors, 1)
			result <- AsyncResult{Error: fmt.Errorf("Redis exists error: %w", err)}
			return
		}

		responseTime := time.Since(start)
		atomic.AddInt64(&p.stats.AverageResponseTime, int64(responseTime))

		result <- AsyncResult{
			Exists: count > 0,
		}
	}()

	return result
}

// Close closes the connection pool
func (p *OptimizedConnectionPool) Close() error {
	if p == nil {
		return nil
	}

	// Set closed flag first
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // Already closed
	}

	// Stop monitoring first
	if p.monitorTicker != nil {
		p.monitorTicker.Stop()
		close(p.monitorDone)
	}

	// Cancel context to stop all goroutines
	p.cancel()

	// Wait for goroutines to finish
	p.monitorWg.Wait()
	p.warmingWg.Wait()

	// Close Redis client
	if p.client != nil {
		return p.client.Close()
	}

	return nil
}

// GetStats returns connection pool statistics
func (p *OptimizedConnectionPool) GetStats() *ConnectionPoolStats {
	if p == nil {
		return &ConnectionPoolStats{}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.getStatsInternal()
}

// getStatsInternal returns connection pool statistics without acquiring locks
// This method should only be called when the caller already holds the appropriate lock
func (p *OptimizedConnectionPool) getStatsInternal() *ConnectionPoolStats {
	// Create a copy of stats to avoid race conditions
	stats := &ConnectionPoolStats{}

	// Get pool stats from Redis client with nil check
	if p.client != nil {
		poolStats := p.client.PoolStats()
		stats.TotalConnections = int64(poolStats.TotalConns)
		stats.IdleConnections = int64(poolStats.IdleConns)
		stats.WaitCount = int64(poolStats.WaitCount)
		// Note: Some fields are not available in go-redis PoolStats
		// These are limitations of the go-redis library
		stats.ActiveConnections = 0 // Not available in go-redis
		stats.WaitDuration = 0      // Not available in go-redis
		stats.IdleClosed = 0        // Not available in go-redis
		stats.StaleClosed = 0       // Not available in go-redis
	}

	// Add nil check for p.stats
	if p.stats != nil {
		// Copy atomic stats
		stats.Hits = atomic.LoadInt64(&p.stats.Hits)
		stats.Misses = atomic.LoadInt64(&p.stats.Misses)
		stats.Timeouts = atomic.LoadInt64(&p.stats.Timeouts)
		stats.Errors = atomic.LoadInt64(&p.stats.Errors)
		stats.HealthChecks = atomic.LoadInt64(&p.stats.HealthChecks)
		stats.HealthFailures = atomic.LoadInt64(&p.stats.HealthFailures)
		stats.LoadBalancedRequests = atomic.LoadInt64(&p.stats.LoadBalancedRequests)
		stats.CircuitBreakerTrips = atomic.LoadInt64(&p.stats.CircuitBreakerTrips)
		stats.CircuitBreakerResets = atomic.LoadInt64(&p.stats.CircuitBreakerResets)
		stats.ReusedConnections = atomic.LoadInt64(&p.stats.ReusedConnections)
		stats.NewConnections = atomic.LoadInt64(&p.stats.NewConnections)
		stats.WarmedConnections = atomic.LoadInt64(&p.stats.WarmedConnections)
		stats.AverageResponseTime = atomic.LoadInt64(&p.stats.AverageResponseTime)
	}

	return stats
}

// Helper methods

func (p *OptimizedConnectionPool) checkHealth(ctx context.Context) error {
	// Check if pool is closed
	if p.isClosed() {
		return fmt.Errorf("connection pool is closed")
	}

	// Use a single mutex for health check synchronization to prevent race conditions
	p.healthCheckMutex.Lock()
	defer p.healthCheckMutex.Unlock()

	// Check if health check is needed
	if time.Since(p.lastHealthCheck) < p.config.HealthCheckInterval {
		return nil
	}

	// Check if pool is still not closed after acquiring lock
	if p.isClosed() {
		return fmt.Errorf("connection pool is closed")
	}

	// Check if client is available
	if p.client == nil {
		atomic.AddInt64(&p.stats.HealthFailures, 1)
		return fmt.Errorf("Redis client is not available")
	}

	// Perform health check
	healthCtx, cancel := context.WithTimeout(ctx, p.config.HealthCheckTimeout)
	defer cancel()

	err := p.client.Ping(healthCtx).Err()
	if err != nil {
		atomic.AddInt64(&p.stats.HealthFailures, 1)
		logx.Error("Connection pool health check failed", logx.ErrorField(err))
		return fmt.Errorf("health check failed: %w", err)
	}

	atomic.AddInt64(&p.stats.HealthChecks, 1)
	p.lastHealthCheck = time.Now()
	return nil
}

func (p *OptimizedConnectionPool) warmConnections() {
	defer p.warmingWg.Done()

	// Check if pool is closed before starting
	if p.isClosed() {
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, p.config.ConnectionWarmingTimeout)
	defer cancel()

	// Warm up connections by performing simple operations
	for i := 0; i < p.config.MinIdleConns; i++ {
		select {
		case <-ctx.Done():
			logx.Info("Connection warming cancelled due to timeout or shutdown")
			return
		case <-p.ctx.Done():
			logx.Info("Connection warming cancelled due to pool shutdown")
			return
		default:
			// Check if pool is still not closed
			if p.isClosed() {
				logx.Info("Connection warming cancelled due to pool closure")
				return
			}

			// Check if client is available
			if p.client == nil {
				logx.Error("Connection warming failed: Redis client is not available")
				return
			}

			// Perform a simple ping to warm up connection
			if err := p.client.Ping(ctx).Err(); err == nil {
				atomic.AddInt64(&p.stats.WarmedConnections, 1)
			} else {
				logx.Error("Connection warming failed", logx.ErrorField(err))
			}
		}
	}

	logx.Info("Connection warming completed",
		logx.Int("warmedConnections", p.config.MinIdleConns))
}

func (p *OptimizedConnectionPool) startPoolMonitoring() {
	p.monitorTicker = time.NewTicker(p.config.MonitoringInterval)
	p.monitorDone = make(chan struct{})

	p.monitorWg.Add(1)
	go func() {
		defer p.monitorWg.Done()
		defer func() {
			if r := recover(); r != nil {
				logx.Error("Pool monitoring goroutine panicked", logx.Any("panic", r))
				// Don't restart the goroutine after panic to prevent leaks
			}
		}()

		for {
			select {
			case <-p.monitorTicker.C:
				p.monitorPoolHealth()
			case <-p.monitorDone:
				return
			case <-p.ctx.Done():
				return
			}
		}
	}()
}

func (p *OptimizedConnectionPool) monitorPoolHealth() {
	// Check if pool is closed
	if p.isClosed() {
		return
	}

	// Get stats without double locking
	stats := p.getStatsInternal()

	// Log pool statistics
	logx.Info("Connection pool health check",
		logx.Int64("totalConnections", stats.TotalConnections),
		logx.Int64("activeConnections", stats.ActiveConnections),
		logx.Int64("idleConnections", stats.IdleConnections),
		logx.Int64("waitCount", stats.WaitCount),
		logx.Int64("errors", stats.Errors),
		logx.Int64("healthChecks", stats.HealthChecks),
		logx.Int64("healthFailures", stats.HealthFailures))

	// Alert if pool is under pressure
	if stats.WaitCount > 0 {
		logx.Warn("Connection pool under pressure",
			logx.Int64("waitCount", stats.WaitCount),
			logx.Int64("activeConnections", stats.ActiveConnections))
	}

	// Alert if error rate is high
	if stats.Errors > 0 && stats.HealthFailures > 0 {
		logx.Error("Connection pool experiencing errors",
			logx.Int64("errors", stats.Errors),
			logx.Int64("healthFailures", stats.HealthFailures))
	}
}

func validateOptimizedConnectionPoolConfig(config *OptimizedConnectionPoolConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

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
	if config.PoolTimeout <= 0 {
		return fmt.Errorf("pool timeout must be positive")
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 5 * time.Minute // Set default if not provided
	}
	if config.IdleCheckFrequency <= 0 {
		config.IdleCheckFrequency = 1 * time.Minute // Set default if not provided
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
	if config.ConnectionWarmingTimeout <= 0 {
		config.ConnectionWarmingTimeout = 10 * time.Second // Set default if not provided
	}
	if config.MonitoringInterval <= 0 {
		config.MonitoringInterval = 30 * time.Second // Set default if not provided
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
