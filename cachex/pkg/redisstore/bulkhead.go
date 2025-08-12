package redisstore

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BulkheadConfig holds configuration for bulkhead isolation
type BulkheadConfig struct {
	// Read pool configuration
	ReadPoolSize     int
	ReadMinIdleConns int
	ReadMaxRetries   int
	ReadDialTimeout  time.Duration
	ReadTimeout      time.Duration
	ReadPoolTimeout  time.Duration

	// Write pool configuration
	WritePoolSize     int
	WriteMinIdleConns int
	WriteMaxRetries   int
	WriteDialTimeout  time.Duration
	WriteTimeout      time.Duration
	WritePoolTimeout  time.Duration

	// Common configuration
	Addr      string
	Password  string
	DB        int
	TLSConfig *TLSConfig
}

// BulkheadStore implements bulkhead isolation with separate pools for reads/writes
type BulkheadStore struct {
	readClient  redis.Cmdable
	writeClient redis.Cmdable
	config      *BulkheadConfig
	mu          sync.RWMutex
	closed      bool
}

// NewBulkhead creates a new bulkhead store with separate read/write pools
func NewBulkhead(config *BulkheadConfig) (*BulkheadStore, error) {
	if config == nil {
		config = &BulkheadConfig{
			Addr: "localhost:6379",
			DB:   0,
		}
	}

	// Set default values for read pool
	if config.ReadPoolSize == 0 {
		config.ReadPoolSize = 20
	}
	if config.ReadMinIdleConns == 0 {
		config.ReadMinIdleConns = 10
	}
	if config.ReadMaxRetries == 0 {
		config.ReadMaxRetries = 3
	}
	if config.ReadDialTimeout == 0 {
		config.ReadDialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.ReadPoolTimeout == 0 {
		config.ReadPoolTimeout = 4 * time.Second
	}

	// Set default values for write pool
	if config.WritePoolSize == 0 {
		config.WritePoolSize = 10
	}
	if config.WriteMinIdleConns == 0 {
		config.WriteMinIdleConns = 5
	}
	if config.WriteMaxRetries == 0 {
		config.WriteMaxRetries = 3
	}
	if config.WriteDialTimeout == 0 {
		config.WriteDialTimeout = 5 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.WritePoolTimeout == 0 {
		config.WritePoolTimeout = 4 * time.Second
	}

	// Create read client
	readOpts := &redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.ReadPoolSize,
		MinIdleConns: config.ReadMinIdleConns,
		MaxRetries:   config.ReadMaxRetries,
		DialTimeout:  config.ReadDialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.ReadPoolTimeout,
	}

	if config.TLSConfig != nil && config.TLSConfig.Enabled {
		readOpts.TLSConfig = &tls.Config{
			InsecureSkipVerify: config.TLSConfig.InsecureSkipVerify,
		}
	}

	readClient := redis.NewClient(readOpts)

	// Create write client
	writeOpts := &redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.WritePoolSize,
		MinIdleConns: config.WriteMinIdleConns,
		MaxRetries:   config.WriteMaxRetries,
		DialTimeout:  config.WriteDialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.WritePoolTimeout,
	}

	if config.TLSConfig != nil && config.TLSConfig.Enabled {
		writeOpts.TLSConfig = &tls.Config{
			InsecureSkipVerify: config.TLSConfig.InsecureSkipVerify,
		}
	}

	writeClient := redis.NewClient(writeOpts)

	// Test connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := readClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("read pool connection test failed: %w", err)
	}

	if err := writeClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("write pool connection test failed: %w", err)
	}

	return &BulkheadStore{
		readClient:  readClient,
		writeClient: writeClient,
		config:      config,
	}, nil
}

// NewBulkheadCluster creates a new bulkhead store with Redis cluster
func NewBulkheadCluster(addrs []string, password string, config *BulkheadConfig) (*BulkheadStore, error) {
	if config == nil {
		config = &BulkheadConfig{}
	}

	// Set defaults
	if config.ReadPoolSize == 0 {
		config.ReadPoolSize = 20
	}
	if config.WritePoolSize == 0 {
		config.WritePoolSize = 10
	}

	// Create read cluster client
	readClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		PoolSize:     config.ReadPoolSize,
		MinIdleConns: config.ReadMinIdleConns,
		MaxRetries:   config.ReadMaxRetries,
		DialTimeout:  config.ReadDialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.ReadPoolTimeout,
	})

	// Create write cluster client
	writeClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		PoolSize:     config.WritePoolSize,
		MinIdleConns: config.WriteMinIdleConns,
		MaxRetries:   config.WriteMaxRetries,
		DialTimeout:  config.WriteDialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.WritePoolTimeout,
	})

	// Test connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := readClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("read cluster connection test failed: %w", err)
	}

	if err := writeClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("write cluster connection test failed: %w", err)
	}

	return &BulkheadStore{
		readClient:  readClient,
		writeClient: writeClient,
		config:      config,
	}, nil
}

// Get retrieves a value using the read pool
func (s *BulkheadStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	result := s.readClient.Get(ctx, key)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, nil // Key not found
		}
		return nil, result.Err()
	}

	val, err := result.Bytes()
	if err != nil {
		return nil, err
	}

	return val, nil
}

// Set stores a value using the write pool
func (s *BulkheadStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	return s.writeClient.Set(ctx, key, value, ttl).Err()
}

// MGet retrieves multiple values using the read pool
func (s *BulkheadStore) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	result := s.readClient.MGet(ctx, keys...)
	if result.Err() != nil {
		return nil, result.Err()
	}

	values := result.Val()
	resultMap := make(map[string][]byte)

	for i, key := range keys {
		if values[i] != nil {
			if str, ok := values[i].(string); ok {
				resultMap[key] = []byte(str)
			}
		}
	}

	return resultMap, nil
}

// MSet stores multiple values using the write pool
func (s *BulkheadStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	if len(items) == 0 {
		return nil
	}

	// Use pipeline for better performance
	pipe := s.writeClient.Pipeline()

	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del removes keys using the write pool
func (s *BulkheadStore) Del(ctx context.Context, keys ...string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	if len(keys) == 0 {
		return nil
	}

	return s.writeClient.Del(ctx, keys...).Err()
}

// Exists checks if a key exists using the read pool
func (s *BulkheadStore) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return false, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	result := s.readClient.Exists(ctx, key)
	if result.Err() != nil {
		return false, result.Err()
	}

	return result.Val() > 0, nil
}

// TTL gets the time to live of a key using the read pool
func (s *BulkheadStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return 0, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	result := s.readClient.TTL(ctx, key)
	if result.Err() != nil {
		return 0, result.Err()
	}

	return result.Val(), nil
}

// IncrBy increments a key using the write pool
func (s *BulkheadStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return 0, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	// Use Lua script to atomically increment and set TTL if key doesn't exist
	script := `
		local key = KEYS[1]
		local delta = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[2])
		
		local exists = redis.call('EXISTS', key)
		local result = redis.call('INCRBY', key, delta)
		
		if exists == 0 and ttl > 0 then
			redis.call('EXPIRE', key, ttl)
		end
		
		return result
	`

	result := s.writeClient.Eval(ctx, script, []string{key}, delta, int(ttlIfCreate.Seconds()))
	if result.Err() != nil {
		return 0, result.Err()
	}

	return result.Int64()
}

// Close closes both read and write pools
func (s *BulkheadStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Close read client
	if readClient, ok := s.readClient.(*redis.Client); ok {
		if err := readClient.Close(); err != nil {
			return fmt.Errorf("failed to close read client: %w", err)
		}
	}
	if readClient, ok := s.readClient.(*redis.ClusterClient); ok {
		if err := readClient.Close(); err != nil {
			return fmt.Errorf("failed to close read cluster client: %w", err)
		}
	}

	// Close write client
	if writeClient, ok := s.writeClient.(*redis.Client); ok {
		if err := writeClient.Close(); err != nil {
			return fmt.Errorf("failed to close write client: %w", err)
		}
	}
	if writeClient, ok := s.writeClient.(*redis.ClusterClient); ok {
		if err := writeClient.Close(); err != nil {
			return fmt.Errorf("failed to close write cluster client: %w", err)
		}
	}

	return nil
}

// ReadClient returns the read pool client
func (s *BulkheadStore) ReadClient() redis.Cmdable {
	return s.readClient
}

// WriteClient returns the write pool client
func (s *BulkheadStore) WriteClient() redis.Cmdable {
	return s.writeClient
}

// Config returns the bulkhead configuration
func (s *BulkheadStore) Config() *BulkheadConfig {
	return s.config
}
