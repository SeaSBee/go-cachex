package cachex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seasbee/go-logx"
)

// redisCache implements the Cache interface using go-redis client
type redisCache struct {
	client     *redis.Client
	codec      Codec
	keyBuilder KeyBuilder
	keyHasher  KeyHasher
	ctx        context.Context
	mu         sync.RWMutex
	closed     bool
	closedFlag *bool
}

// NewRedisCache creates a new cache instance using go-redis client
func NewRedisCache(client *redis.Client, codec Codec, keyBuilder KeyBuilder, keyHasher KeyHasher) Cache {
	if codec == nil {
		codec = &JSONCodec{}
	}
	if keyBuilder == nil {
		keyBuilder = &DefaultKeyBuilder{}
	}
	if keyHasher == nil {
		keyHasher = &DefaultKeyHasher{}
	}

	// Create a shared closed flag
	closedFlag := false

	return &redisCache{
		client:     client,
		codec:      codec,
		keyBuilder: keyBuilder,
		keyHasher:  keyHasher,
		ctx:        context.Background(),
		closed:     false,
		closedFlag: &closedFlag,
	}
}

// isClosed checks if the cache is closed
func (c *redisCache) isClosed() bool {
	if c.closedFlag != nil {
		return *c.closedFlag
	}
	return c.closed
}

// WithContext returns a new cache instance with the given context
func (c *redisCache) WithContext(ctx context.Context) Cache {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create new cache without copying the mutex
	newCache := &redisCache{
		client:     c.client,
		codec:      c.codec,
		keyBuilder: c.keyBuilder,
		keyHasher:  c.keyHasher,
		ctx:        ctx,
		closed:     c.closed,
		closedFlag: c.closedFlag,
	}
	return newCache
}

// Get retrieves a value from the cache (non-blocking)
func (c *redisCache) Get(ctx context.Context, key string) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Execute get operation asynchronously
		asyncResult := c.executeGetAsync(ctx, key)
		result <- asyncResult
	}()

	return result
}

// executeGetAsync performs the actual get operation (async)
func (c *redisCache) executeGetAsync(ctx context.Context, key string) AsyncCacheResult[any] {
	// Get from Redis
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return AsyncCacheResult[any]{Found: false} // Key not found
		}
		return AsyncCacheResult[any]{Error: NewCacheError("get", key, "redis error", err)}
	}

	// Decode value - we'll return the raw decoded value as any
	var value any
	if err := c.codec.Decode([]byte(val), &value); err != nil {
		return AsyncCacheResult[any]{Error: NewCacheError("get", key, "decode error", err)}
	}

	return AsyncCacheResult[any]{Value: value, Found: true}
}

// Set stores a value in the cache (non-blocking)
func (c *redisCache) Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Execute set operation asynchronously
		asyncResult := c.executeSetAsync(ctx, key, val, ttl)
		result <- asyncResult
	}()

	return result
}

// executeSetAsync performs the actual set operation (async)
func (c *redisCache) executeSetAsync(ctx context.Context, key string, val any, ttl time.Duration) AsyncCacheResult[any] {
	// Basic boundary condition validations
	if key == "" {
		return AsyncCacheResult[any]{Error: NewCacheError("set", key, "key validation error", fmt.Errorf("key cannot be empty"))}
	}

	// Encode value
	data, err := c.codec.Encode(val)
	if err != nil {
		return AsyncCacheResult[any]{Error: NewCacheError("set", key, "encode error", err)}
	}

	// Check if encoded data is nil
	if data == nil {
		return AsyncCacheResult[any]{Error: NewCacheError("set", key, "value validation error", fmt.Errorf("value cannot be nil"))}
	}

	// Store in Redis
	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return AsyncCacheResult[any]{Error: NewCacheError("set", key, "redis error", err)}
	}

	return AsyncCacheResult[any]{Value: val}
}

// MGet retrieves multiple values from the cache (non-blocking)
func (c *redisCache) MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(keys) == 0 {
			result <- AsyncCacheResult[any]{Values: make(map[string]any)}
			return
		}

		// Get from Redis
		vals, err := c.client.MGet(ctx, keys...).Result()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("mget", "multiple", "redis error", err)}
			return
		}

		// Decode values
		values := make(map[string]any)
		for i, val := range vals {
			if val != nil {
				if strVal, ok := val.(string); ok && strVal != "" {
					var value any
					if err := c.codec.Decode([]byte(strVal), &value); err != nil {
						logx.Warn("Failed to decode value in MGet", logx.String("key", keys[i]), logx.ErrorField(err))
						continue
					}
					values[keys[i]] = value
				}
			}
		}

		result <- AsyncCacheResult[any]{Values: values}
	}()

	return result
}

// MSet stores multiple values in the cache (non-blocking)
func (c *redisCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(items) == 0 {
			result <- AsyncCacheResult[any]{}
			return
		}

		// Encode values
		dataMap := make(map[string]interface{})
		for key, value := range items {
			data, err := c.codec.Encode(value)
			if err != nil {
				result <- AsyncCacheResult[any]{Error: NewCacheError("mset", key, "encode error", err)}
				return
			}

			// Check if encoded data is nil
			if data == nil {
				result <- AsyncCacheResult[any]{Error: NewCacheError("mset", key, "value validation error", fmt.Errorf("value cannot be nil"))}
				return
			}

			dataMap[key] = data
		}

		// Store in Redis
		err := c.client.MSet(ctx, dataMap).Err()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("mset", "multiple", "redis error", err)}
			return
		}

		// Set TTL for all keys if specified
		if ttl > 0 {
			for key := range dataMap {
				c.client.Expire(ctx, key, ttl)
			}
		}

		result <- AsyncCacheResult[any]{}
	}()

	return result
}

// Del removes keys from the cache (non-blocking)
func (c *redisCache) Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(keys) == 0 {
			result <- AsyncCacheResult[any]{}
			return
		}

		// Delete from Redis
		count, err := c.client.Del(ctx, keys...).Result()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("del", "multiple", "redis error", err)}
			return
		}

		result <- AsyncCacheResult[any]{Count: count}
	}()

	return result
}

// Exists checks if a key exists in the cache (non-blocking)
func (c *redisCache) Exists(ctx context.Context, key string) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Check in Redis
		count, err := c.client.Exists(ctx, key).Result()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("exists", key, "redis error", err)}
			return
		}

		result <- AsyncCacheResult[any]{Found: count > 0}
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking)
func (c *redisCache) TTL(ctx context.Context, key string) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Get TTL from Redis
		ttl, err := c.client.TTL(ctx, key).Result()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("ttl", key, "redis error", err)}
			return
		}

		result <- AsyncCacheResult[any]{TTL: ttl}
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (c *redisCache) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[any] {
	result := make(chan AsyncCacheResult[any], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[any]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Increment in Redis
		newVal, err := c.client.IncrBy(ctx, key, delta).Result()
		if err != nil {
			result <- AsyncCacheResult[any]{Error: NewCacheError("incrby", key, "redis error", err)}
			return
		}

		// Set TTL if this is a new key
		if ttlIfCreate > 0 {
			exists, err := c.client.Exists(ctx, key).Result()
			if err == nil && exists == 1 {
				// Check if this was a new key by trying to set TTL
				c.client.Expire(ctx, key, ttlIfCreate)
			}
		}

		result <- AsyncCacheResult[any]{Int: newVal}
	}()

	return result
}

// Close closes the cache and releases resources
func (c *redisCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed() {
		return nil
	}

	c.closed = true
	if c.closedFlag != nil {
		*c.closedFlag = true
	}

	// Close Redis client
	if err := c.client.Close(); err != nil {
		logx.Error("Failed to close Redis client", logx.ErrorField(err))
		return err
	}

	return nil
}

// Default implementations
type DefaultKeyBuilder struct{}

func (b *DefaultKeyBuilder) Build(entity, id string) string {
	return fmt.Sprintf("%s:%s", entity, id)
}

func (b *DefaultKeyBuilder) BuildList(entity string, filters map[string]any) string {
	return fmt.Sprintf("list:%s", entity)
}

func (b *DefaultKeyBuilder) BuildComposite(entityA, idA, entityB, idB string) string {
	return fmt.Sprintf("%s:%s:%s:%s", entityA, idA, entityB, idB)
}

func (b *DefaultKeyBuilder) BuildSession(sid string) string {
	return fmt.Sprintf("session:%s", sid)
}

type DefaultKeyHasher struct{}

func (h *DefaultKeyHasher) Hash(data string) string {
	// Simple hash implementation
	return fmt.Sprintf("hash_%s", data)
}

// KeyBuilder helper methods for redisCache
func (c *redisCache) BuildKey(entity, id string) string {
	return c.keyBuilder.Build(entity, id)
}

func (c *redisCache) BuildListKey(entity string, filters map[string]any) string {
	return c.keyBuilder.BuildList(entity, filters)
}

func (c *redisCache) BuildCompositeKey(entityA, idA, entityB, idB string) string {
	return c.keyBuilder.BuildComposite(entityA, idA, entityB, idB)
}

func (c *redisCache) BuildSessionKey(sid string) string {
	return c.keyBuilder.BuildSession(sid)
}

func (c *redisCache) GetKeyBuilder() KeyBuilder {
	return c.keyBuilder
}
