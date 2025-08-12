package redisstore

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis configuration
type Config struct {
	Addr            string
	Password        string
	DB              int
	TLSConfig       *TLSConfig
	PoolSize        int
	MinIdleConns    int
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolTimeout     time.Duration
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
}

// Store implements the cache.Store interface using Redis
type Store struct {
	client redis.Cmdable
	config *Config
}

// New creates a new Redis store
func New(config *Config) (*Store, error) {
	if config == nil {
		config = &Config{
			Addr:         "localhost:6379",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
		}
	}

	// Set default timeouts if not provided
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = 4 * time.Second
	}

	// Create Redis options
	opts := &redis.Options{
		Addr:            config.Addr,
		Password:        config.Password,
		DB:              config.DB,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
		DialTimeout:     config.DialTimeout,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		PoolTimeout:     config.PoolTimeout,
	}

	// Configure TLS if enabled
	if config.TLSConfig != nil && config.TLSConfig.Enabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: config.TLSConfig.InsecureSkipVerify,
		}
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Store{
		client: client,
		config: config,
	}, nil
}

// NewCluster creates a new Redis cluster store
func NewCluster(addrs []string, password string) (*Store, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
		PoolSize: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Store{
		client: client,
		config: &Config{},
	}, nil
}

// NewSentinel creates a new Redis sentinel store
func NewSentinel(masterName string, sentinelAddrs []string, password string) (*Store, error) {
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: sentinelAddrs,
		Password:      password,
		PoolSize:      10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Store{
		client: client,
		config: &Config{},
	}, nil
}

// Get retrieves a value from Redis
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	result := s.client.Get(ctx, key)
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

// Set stores a value in Redis with TTL
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

// MGet retrieves multiple values from Redis
func (s *Store) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	result := s.client.MGet(ctx, keys...)
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

// MSet stores multiple values in Redis with TTL
func (s *Store) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Use pipeline for better performance
	pipe := s.client.Pipeline()

	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del removes keys from Redis
func (s *Store) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	return s.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	result := s.client.Exists(ctx, key)
	if result.Err() != nil {
		return false, result.Err()
	}

	return result.Val() > 0, nil
}

// TTL gets the time to live of a key
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	result := s.client.TTL(ctx, key)
	if result.Err() != nil {
		return 0, result.Err()
	}

	return result.Val(), nil
}

// IncrBy increments a key by the given delta
func (s *Store) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
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

	result := s.client.Eval(ctx, script, []string{key}, delta, int(ttlIfCreate.Seconds()))
	if result.Err() != nil {
		return 0, result.Err()
	}

	return result.Int64()
}

// Close closes the Redis connection
func (s *Store) Close() error {
	if client, ok := s.client.(*redis.Client); ok {
		return client.Close()
	}
	if client, ok := s.client.(*redis.ClusterClient); ok {
		return client.Close()
	}

	return nil
}

// Client returns the underlying Redis client
func (s *Store) Client() redis.Cmdable {
	return s.client
}
