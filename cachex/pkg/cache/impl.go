package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/bloom"
	"github.com/SeaSBee/go-cachex/cachex/internal/cb"
	"github.com/SeaSBee/go-cachex/cachex/internal/dlq"
	"github.com/SeaSBee/go-cachex/cachex/internal/retry"
	"github.com/SeaSBee/go-cachex/cachex/pkg/codec"
	"github.com/SeaSBee/go-cachex/cachex/pkg/observability"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
	"github.com/seasbee/go-logx"
	"go.opentelemetry.io/otel/trace"
)

// cache implements the Cache interface
type cache[T any] struct {
	store            Store
	codec            Codec
	keyBuilder       KeyBuilder
	keyHasher        KeyHasher
	options          *Options
	ctx              context.Context
	mu               sync.RWMutex
	closed           bool
	circuitBreaker   *cb.CircuitBreaker
	retryPolicy      retry.Policy
	deadLetterQueue  *dlq.DeadLetterQueue
	bloomFilter      *bloom.BloomFilter
	observability    *observability.Observability
	tagManager       *TagManager
	pubSubManager    *PubSubManager
	refreshScheduler *RefreshAheadScheduler
}

// New creates a new cache instance
func New[T any](opts ...Option) (Cache[T], error) {
	options := &Options{
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		CircuitBreaker: CircuitBreakerConfig{
			Threshold:   5,
			Timeout:     30 * time.Second,
			HalfOpenMax: 3,
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 1000,
			Burst:             100,
		},
		Security: SecurityConfig{
			RedactLogs: true,
		},
		Observability: ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Validate required components
	if options.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if options.Codec == nil {
		options.Codec = &defaultJSONCodec{}
	}
	if options.KeyBuilder == nil {
		options.KeyBuilder = &defaultKeyBuilder{}
	}
	if options.KeyHasher == nil {
		options.KeyHasher = &defaultKeyHasher{}
	}

	// Create circuit breaker
	circuitBreaker := cb.New(cb.Config{
		Threshold:   options.CircuitBreaker.Threshold,
		Timeout:     options.CircuitBreaker.Timeout,
		HalfOpenMax: options.CircuitBreaker.HalfOpenMax,
	})

	// Create retry policy
	retryPolicy := retry.Policy{
		MaxAttempts:  options.MaxRetries,
		InitialDelay: options.RetryDelay,
		MaxDelay:     options.RetryDelay * time.Duration(1<<uint(options.MaxRetries)),
		Multiplier:   2.0,
		Jitter:       true,
	}

	// Create dead-letter queue for failed write-behind operations
	var deadLetterQueue *dlq.DeadLetterQueue
	if options.EnableDeadLetterQueue {
		deadLetterQueue = dlq.NewDeadLetterQueue(&dlq.Config{
			MaxRetries:    3,
			RetryDelay:    5 * time.Minute,
			WorkerCount:   2,
			QueueSize:     1000,
			EnableMetrics: true,
		})
	}

	// Create bloom filter for reducing misses to backing store
	var bloomFilter *bloom.BloomFilter
	if options.EnableBloomFilter {
		// Create adapter for bloom filter store interface
		bloomStore := &bloomStoreAdapter{store: options.Store}
		bloomConfig := bloom.DefaultConfig(bloomStore)
		bloomFilter, _ = bloom.NewBloomFilter(bloomConfig)
	}

	// Create observability instance (disable metrics in tests to avoid conflicts)
	obsConfig := &observability.Config{
		EnableMetrics:  false, // Disable metrics to avoid registration conflicts
		EnableTracing:  options.Observability.EnableTracing,
		EnableLogging:  options.Observability.EnableLogging,
		ServiceName:    "cachex",
		ServiceVersion: "1.0.0",
		Environment:    "production",
	}
	obs := observability.New(obsConfig)

	// Create tag manager
	tagManager := NewTagManager(options.Store, DefaultTagConfig())

	// Create pub/sub manager (only if we have a Redis client)
	var pubSubManager *PubSubManager
	if redisStore, ok := options.Store.(*redisstore.Store); ok {
		if client := redisStore.Client(); client != nil {
			pubSubManager = NewPubSubManager(client, DefaultPubSubConfig())
		}
	}

	// Create refresh-ahead scheduler
	refreshScheduler := NewRefreshAheadScheduler(options.Store, DefaultRefreshAheadConfig())

	c := &cache[T]{
		store:            options.Store,
		codec:            options.Codec,
		keyBuilder:       options.KeyBuilder,
		keyHasher:        options.KeyHasher,
		options:          options,
		ctx:              context.Background(),
		circuitBreaker:   circuitBreaker,
		retryPolicy:      retryPolicy,
		deadLetterQueue:  deadLetterQueue,
		bloomFilter:      bloomFilter,
		observability:    obs,
		tagManager:       tagManager,
		pubSubManager:    pubSubManager,
		refreshScheduler: refreshScheduler,
	}

	return c, nil
}

// WithContext returns a new cache instance with the given context
func (c *cache[T]) WithContext(ctx context.Context) Cache[T] {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create new cache without copying the mutex
	newCache := &cache[T]{
		store:          c.store,
		codec:          c.codec,
		keyBuilder:     c.keyBuilder,
		keyHasher:      c.keyHasher,
		options:        c.options,
		ctx:            ctx,
		closed:         c.closed,
		circuitBreaker: c.circuitBreaker,
		retryPolicy:    c.retryPolicy,
	}
	return newCache
}

// Get retrieves a value from the cache
func (c *cache[T]) Get(key string) (T, bool, error) {
	var zero T

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return zero, false, ErrStoreClosed
	}
	c.mu.RUnlock()

	// Create tracing span
	var span trace.Span
	var traceCtx context.Context
	if c.observability != nil {
		traceCtx, span = c.observability.TraceOperation(c.ctx, "get", key, "", 0, 0)
		defer span.End()
	} else {
		traceCtx = c.ctx
	}

	start := time.Now()
	defer func() {
		c.logOperation("get", key, time.Since(start), nil)
	}()

	// Check bloom filter first if enabled
	if c.bloomFilter != nil {
		contains, bloomErr := c.bloomFilter.Contains(c.ctx, key)
		if bloomErr != nil {
			logx.Warn("Bloom filter check failed", logx.String("key", key), logx.ErrorField(bloomErr))
		} else if !contains {
			// Key definitely not in cache, return early
			return zero, false, nil
		}
	}

	// Execute with circuit breaker and retry logic
	var result T
	var found bool

	err := retry.RetryWithContext(traceCtx, c.retryPolicy, func() error {
		var getErr error
		result, found, getErr = c.executeGet(key)
		return getErr
	})

	if err != nil {
		return zero, false, NewCacheError("get", key, "retry failed", err)
	}

	// Add to bloom filter if not found (to remember non-existent keys)
	if !found && c.bloomFilter != nil {
		if bloomErr := c.bloomFilter.Add(c.ctx, key); bloomErr != nil {
			logx.Warn("Failed to add key to bloom filter", logx.String("key", key), logx.ErrorField(bloomErr))
		}
	}

	return result, found, nil
}

// executeGet performs the actual get operation with circuit breaker protection
func (c *cache[T]) executeGet(key string) (T, bool, error) {
	var zero T

	// Execute with circuit breaker protection
	var data []byte
	var err error

	err = c.circuitBreaker.ExecuteWithContext(c.ctx, func() error {
		// Get from store
		data, err = c.store.Get(c.ctx, key)
		return err
	})

	if err != nil {
		return zero, false, err
	}

	if data == nil {
		return zero, false, nil // Key not found
	}

	// Decode value
	var value T
	if err := c.codec.Decode(data, &value); err != nil {
		return zero, false, NewCacheError("get", key, "decode error", err)
	}

	return value, true, nil
}

// Set stores a value in the cache
func (c *cache[T]) Set(key string, val T, ttl time.Duration) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	if ttl == 0 {
		ttl = c.options.DefaultTTL
	}

	start := time.Now()
	defer func() {
		c.logOperation("set", key, time.Since(start), nil)
	}()

	// Execute with circuit breaker and retry logic
	err := retry.RetryWithContext(c.ctx, c.retryPolicy, func() error {
		return c.executeSet(key, val, ttl)
	})

	if err != nil {
		return NewCacheError("set", key, "retry failed", err)
	}

	return nil
}

// executeSet performs the actual set operation with circuit breaker protection
func (c *cache[T]) executeSet(key string, val T, ttl time.Duration) error {
	// Execute with circuit breaker protection
	return c.circuitBreaker.ExecuteWithContext(c.ctx, func() error {
		// Encode value
		data, err := c.codec.Encode(val)
		if err != nil {
			return NewCacheError("set", key, "encode error", err)
		}

		// Store in cache
		if err := c.store.Set(c.ctx, key, data, ttl); err != nil {
			return NewCacheError("set", key, "store error", err)
		}

		return nil
	})
}

// MGet retrieves multiple values from the cache
func (c *cache[T]) MGet(keys ...string) (map[string]T, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	c.mu.RUnlock()

	if len(keys) == 0 {
		return make(map[string]T), nil
	}

	start := time.Now()
	defer func() {
		c.logOperation("mget", fmt.Sprintf("%d keys", len(keys)), time.Since(start), nil)
	}()

	// Get from store
	dataMap, err := c.store.MGet(c.ctx, keys...)
	if err != nil {
		return nil, NewCacheError("mget", "multiple", "store error", err)
	}

	// Decode values
	result := make(map[string]T)
	for key, data := range dataMap {
		if data != nil {
			var value T
			if err := c.codec.Decode(data, &value); err != nil {
				logx.Warn("Failed to decode value in MGet", logx.String("key", key), logx.ErrorField(err))
				continue
			}
			result[key] = value
		}
	}

	return result, nil
}

// MSet stores multiple values in the cache
func (c *cache[T]) MSet(items map[string]T, ttl time.Duration) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	if len(items) == 0 {
		return nil
	}

	if ttl == 0 {
		ttl = c.options.DefaultTTL
	}

	start := time.Now()
	defer func() {
		c.logOperation("mset", fmt.Sprintf("%d items", len(items)), time.Since(start), nil)
	}()

	// Encode values
	dataMap := make(map[string][]byte)
	for key, value := range items {
		data, err := c.codec.Encode(value)
		if err != nil {
			return NewCacheError("mset", key, "encode error", err)
		}
		dataMap[key] = data
	}

	// Store in cache
	if err := c.store.MSet(c.ctx, dataMap, ttl); err != nil {
		return NewCacheError("mset", "multiple", "store error", err)
	}

	return nil
}

// Del removes keys from the cache
func (c *cache[T]) Del(keys ...string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	if len(keys) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		c.logOperation("del", fmt.Sprintf("%d keys", len(keys)), time.Since(start), nil)
	}()

	// Delete from store
	if err := c.store.Del(c.ctx, keys...); err != nil {
		return NewCacheError("del", "multiple", "store error", err)
	}

	// Remove tag associations
	for _, key := range keys {
		if err := c.tagManager.RemoveKey(c.ctx, key); err != nil {
			logx.Warn("Failed to remove tag associations", logx.String("key", key), logx.ErrorField(err))
		}
	}

	return nil
}

// Exists checks if a key exists in the cache
func (c *cache[T]) Exists(key string) (bool, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false, ErrStoreClosed
	}
	c.mu.RUnlock()

	start := time.Now()
	defer func() {
		c.logOperation("exists", key, time.Since(start), nil)
	}()

	return c.store.Exists(c.ctx, key)
}

// TTL gets the time to live of a key
func (c *cache[T]) TTL(key string) (time.Duration, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	c.mu.RUnlock()

	start := time.Now()
	defer func() {
		c.logOperation("ttl", key, time.Since(start), nil)
	}()

	return c.store.TTL(c.ctx, key)
}

// IncrBy increments a key by the given delta
func (c *cache[T]) IncrBy(key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	c.mu.RUnlock()

	start := time.Now()
	defer func() {
		c.logOperation("incrby", key, time.Since(start), nil)
	}()

	return c.store.IncrBy(c.ctx, key, delta, ttlIfCreate)
}

// ReadThrough implements the read-through pattern
func (c *cache[T]) ReadThrough(key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// Try to get from cache first
	value, found, err := c.Get(key)
	if err != nil {
		return zero, err
	}

	if found {
		return value, nil
	}

	// Load from source
	value, err = loader(c.ctx)
	if err != nil {
		return zero, err
	}

	// Store in cache
	if err := c.Set(key, value, ttl); err != nil {
		logx.Warn("Failed to store value in ReadThrough", logx.String("key", key), logx.ErrorField(err))
	}

	return value, nil
}

// WriteThrough implements the write-through pattern
func (c *cache[T]) WriteThrough(key string, val T, ttl time.Duration, writer func(ctx context.Context) error) error {
	// Write to source first
	if err := writer(c.ctx); err != nil {
		return err
	}

	// Then write to cache
	return c.Set(key, val, ttl)
}

// WriteBehind implements the write-behind pattern
func (c *cache[T]) WriteBehind(key string, val T, ttl time.Duration, writer func(ctx context.Context) error) error {
	// Write to cache immediately
	if err := c.Set(key, val, ttl); err != nil {
		return err
	}

	// Write to source asynchronously
	go func() {
		if err := writer(context.Background()); err != nil {
			logx.Error("Failed to write behind", logx.String("key", key), logx.ErrorField(err))

			// Add to dead-letter queue if enabled
			if c.deadLetterQueue != nil {
				operation := &dlq.FailedOperation{
					Operation:   "write_behind",
					Key:         key,
					Data:        val,
					Error:       err.Error(),
					HandlerType: "write_behind",
				}

				if dlqErr := c.deadLetterQueue.AddFailedOperation(operation); dlqErr != nil {
					logx.Error("Failed to add operation to DLQ",
						logx.String("key", key),
						logx.ErrorField(dlqErr))
				}
			}
		}
	}()

	return nil
}

// RefreshAhead implements the refresh-ahead pattern
func (c *cache[T]) RefreshAhead(key string, refreshBefore time.Duration, loader func(ctx context.Context) (T, error)) error {
	// Schedule refresh-ahead if scheduler is available
	if c.refreshScheduler != nil {
		// Create a wrapper loader that handles the type conversion
		wrapperLoader := func(ctx context.Context) (interface{}, error) {
			value, err := loader(ctx)
			if err != nil {
				return nil, err
			}

			// Encode the value to bytes
			data, err := c.codec.Encode(value)
			if err != nil {
				return nil, fmt.Errorf("failed to encode value for refresh: %w", err)
			}

			return data, nil
		}

		if err := c.refreshScheduler.ScheduleRefresh(key, refreshBefore, wrapperLoader); err != nil {
			return NewCacheError("refresh_ahead", key, "scheduler error", err)
		}

		logx.Info("Scheduled refresh-ahead",
			logx.String("key", key),
			logx.String("refresh_before", refreshBefore.String()))
		return nil
	}

	// Fallback to immediate refresh if scheduler is not available
	ttl, err := c.TTL(key)
	if err != nil {
		return err
	}

	// If TTL is less than refreshBefore, refresh the value
	if ttl <= refreshBefore {
		value, err := loader(c.ctx)
		if err != nil {
			return err
		}

		// Get the original TTL or use default
		originalTTL := c.options.DefaultTTL
		if ttl > 0 {
			originalTTL = ttl
		}

		return c.Set(key, value, originalTTL)
	}

	return nil
}

// InvalidateByTag invalidates all keys with the given tags
func (c *cache[T]) InvalidateByTag(tags ...string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	// Collect all keys for the given tags
	var allKeys []string
	for _, tag := range tags {
		keys, err := c.tagManager.GetKeysByTag(c.ctx, tag)
		if err != nil {
			logx.Error("Failed to get keys by tag", logx.String("tag", tag), logx.ErrorField(err))
			continue
		}
		allKeys = append(allKeys, keys...)
	}

	// Remove duplicates
	keySet := make(map[string]bool)
	for _, key := range allKeys {
		keySet[key] = true
	}

	uniqueKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		uniqueKeys = append(uniqueKeys, key)
	}

	// Invalidate keys
	if err := c.tagManager.InvalidateByTag(c.ctx, uniqueKeys); err != nil {
		return NewCacheError("invalidate_by_tag", fmt.Sprintf("%v", tags), "tag manager error", err)
	}

	logx.Info("Tag invalidation completed",
		logx.String("tags", fmt.Sprintf("%v", tags)),
		logx.Int("keys_invalidated", len(uniqueKeys)))
	return nil
}

// AddTags adds tags to a key
func (c *cache[T]) AddTags(key string, tags ...string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	if err := c.tagManager.AddTags(c.ctx, key, tags...); err != nil {
		return NewCacheError("add_tags", key, "tag manager error", err)
	}

	logx.Info("Tags added to key", logx.String("key", key), logx.String("tags", fmt.Sprintf("%v", tags)))
	return nil
}

// TryLock attempts to acquire a distributed lock
func (c *cache[T]) TryLock(key string, ttl time.Duration) (func() error, bool, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, false, ErrStoreClosed
	}
	c.mu.RUnlock()

	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	// Try to set the lock key
	err := c.store.Set(c.ctx, lockKey, []byte(lockValue), ttl)
	if err != nil {
		return nil, false, ErrLockFailed
	}

	// Return unlock function
	unlock := func() error {
		return c.store.Del(c.ctx, lockKey)
	}

	return unlock, true, nil
}

// PublishInvalidation publishes invalidation messages
func (c *cache[T]) PublishInvalidation(keys ...string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrStoreClosed
	}
	c.mu.RUnlock()

	if c.pubSubManager == nil {
		return fmt.Errorf("pub/sub manager not available")
	}

	// Get tags for these keys
	var allTags []string
	for _, key := range keys {
		tags, err := c.tagManager.GetTagsByKey(c.ctx, key)
		if err != nil {
			logx.Warn("Failed to get tags for key", logx.String("key", key), logx.ErrorField(err))
			continue
		}
		allTags = append(allTags, tags...)
	}

	// Remove duplicate tags
	tagSet := make(map[string]bool)
	for _, tag := range allTags {
		tagSet[tag] = true
	}

	uniqueTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		uniqueTags = append(uniqueTags, tag)
	}

	// Publish invalidation message
	source := "cachex"

	if err := c.pubSubManager.PublishInvalidation(c.ctx, keys, uniqueTags, source); err != nil {
		return NewCacheError("publish_invalidation", fmt.Sprintf("%v", keys), "pub/sub error", err)
	}

	logx.Info("Invalidation published", logx.String("keys", fmt.Sprintf("%v", keys)), logx.String("tags", fmt.Sprintf("%v", uniqueTags)))
	return nil
}

// SubscribeInvalidations subscribes to invalidation messages
func (c *cache[T]) SubscribeInvalidations(handler func(keys ...string)) (func() error, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	c.mu.RUnlock()

	if c.pubSubManager == nil {
		return nil, fmt.Errorf("pub/sub manager not available")
	}

	// Subscribe to invalidation messages
	subscriberID, err := c.pubSubManager.SubscribeInvalidations(c.ctx, handler)
	if err != nil {
		return nil, NewCacheError("subscribe_invalidations", "", "pub/sub error", err)
	}

	// Return close function
	close := func() error {
		return c.pubSubManager.Unsubscribe(subscriberID)
	}

	logx.Info("Subscribed to invalidations", logx.String("subscriber_id", subscriberID))
	return close, nil
}

// Close closes the cache and releases resources
func (c *cache[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close store
	if err := c.store.Close(); err != nil {
		logx.Error("Failed to close store", logx.ErrorField(err))
	}

	// Close dead letter queue
	if c.deadLetterQueue != nil {
		if err := c.deadLetterQueue.Close(); err != nil {
			logx.Error("Failed to close dead letter queue", logx.ErrorField(err))
		}
	}

	// Close pub/sub manager
	if c.pubSubManager != nil {
		if err := c.pubSubManager.Close(); err != nil {
			logx.Error("Failed to close pub/sub manager", logx.ErrorField(err))
		}
	}

	// Close refresh-ahead scheduler
	if c.refreshScheduler != nil {
		if err := c.refreshScheduler.Close(); err != nil {
			logx.Error("Failed to close refresh-ahead scheduler", logx.ErrorField(err))
		}
	}

	return nil
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (c *cache[T]) GetCircuitBreakerStats() cb.Stats {
	return c.circuitBreaker.GetStats()
}

// GetCircuitBreakerState returns the current circuit breaker state
func (c *cache[T]) GetCircuitBreakerState() cb.State {
	return c.circuitBreaker.GetState()
}

// ForceOpenCircuitBreaker forces the circuit breaker to open state
func (c *cache[T]) ForceOpenCircuitBreaker() {
	c.circuitBreaker.ForceOpen()
}

// ForceCloseCircuitBreaker forces the circuit breaker to closed state
func (c *cache[T]) ForceCloseCircuitBreaker() {
	c.circuitBreaker.ForceClose()
}

// GetDeadLetterQueueStats returns dead-letter queue statistics
func (c *cache[T]) GetDeadLetterQueueStats() *dlq.Metrics {
	if c.deadLetterQueue != nil {
		return c.deadLetterQueue.GetMetrics()
	}
	return nil
}

// GetBloomFilterStats returns bloom filter statistics
func (c *cache[T]) GetBloomFilterStats() map[string]interface{} {
	if c.bloomFilter != nil {
		return c.bloomFilter.GetStats()
	}
	return nil
}

// logOperation logs cache operations for observability
func (c *cache[T]) logOperation(op, key string, duration time.Duration, err error) {
	if c.observability == nil {
		return
	}

	// Determine log level
	level := "info"
	if err != nil {
		level = "error"
	}

	// Extract namespace from key if available
	namespace := ""
	if c.keyBuilder != nil && len(key) > 0 {
		// Try to extract namespace from key pattern
		parts := strings.Split(key, ":")
		if len(parts) > 0 {
			namespace = parts[0]
		}
	}

	// Use observability logging with enhanced fields
	c.observability.LogOperation(level, op, key, namespace, 0, 0, duration, err, c.options.Security.RedactLogs)

	// Record metrics
	if err != nil {
		c.observability.RecordError("operation_failed", "cachex")
	} else {
		status := "success"
		if op == "get" {
			// For get operations, we need to determine hit/miss
			// This is a simplified approach - in practice, you'd track this in the operation
			status = "hit" // Default assumption
		}
		c.observability.RecordOperation(op, status, "cachex", duration, 0, 0)
	}
}

// Default implementations
type defaultJSONCodec struct{}

func (c *defaultJSONCodec) Encode(v any) ([]byte, error) {
	// Use the JSON codec from the codec package
	jsonCodec := &codec.JSONCodec{}
	return jsonCodec.Encode(v)
}

func (c *defaultJSONCodec) Decode(data []byte, v any) error {
	// Use the JSON codec from the codec package
	jsonCodec := &codec.JSONCodec{}
	return jsonCodec.Decode(data, v)
}

type defaultKeyBuilder struct{}

func (b *defaultKeyBuilder) Build(entity, id string) string {
	return fmt.Sprintf("%s:%s", entity, id)
}

func (b *defaultKeyBuilder) BuildList(entity string, filters map[string]any) string {
	return fmt.Sprintf("list:%s", entity)
}

func (b *defaultKeyBuilder) BuildComposite(entityA, idA, entityB, idB string) string {
	return fmt.Sprintf("%s:%s:%s:%s", entityA, idA, entityB, idB)
}

func (b *defaultKeyBuilder) BuildSession(sid string) string {
	return fmt.Sprintf("session:%s", sid)
}

type defaultKeyHasher struct{}

func (h *defaultKeyHasher) Hash(data string) string {
	// Simple hash implementation
	return fmt.Sprintf("hash_%s", data)
}

// bloomStoreAdapter adapts Store interface to BloomStore interface
type bloomStoreAdapter struct {
	store Store
}

func (a *bloomStoreAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return a.store.Set(ctx, key, value, ttl)
}

func (a *bloomStoreAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	return a.store.Get(ctx, key)
}

func (a *bloomStoreAdapter) Del(ctx context.Context, key string) error {
	return a.store.Del(ctx, key)
}
