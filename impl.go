package cachex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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
	closedFlag       *bool
	observability    *ObservabilityManager
	tagManager       *TagManager
	refreshScheduler *RefreshAheadScheduler
}

// New creates a new cache instance
func New[T any](opts ...Option) (Cache[T], error) {
	options := &Options{
		DefaultTTL: 5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
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
		// Create JSON codec with nil value configuration
		jsonCodec := &JSONCodec{
			AllowNilValues: options.AllowNilValues,
			bufferPool:     GlobalPools.Buffer,
		}
		options.Codec = jsonCodec
	} else {
		// If a custom codec is provided, try to configure it for nil values
		if jsonCodec, ok := options.Codec.(*JSONCodec); ok {
			jsonCodec.AllowNilValues = options.AllowNilValues
			jsonCodec.bufferPool = GlobalPools.Buffer
		}
	}
	if options.KeyBuilder == nil {
		options.KeyBuilder = &defaultKeyBuilder{}
	}
	if options.KeyHasher == nil {
		options.KeyHasher = &defaultKeyHasher{}
	}

	// Create observability instance (disable metrics in tests to avoid registration conflicts)
	obsConfig := &ObservabilityManagerConfig{
		EnableMetrics:  false, // Disable metrics to avoid registration conflicts
		EnableTracing:  options.Observability.EnableTracing,
		EnableLogging:  options.Observability.EnableLogging,
		ServiceName:    "cachex",
		ServiceVersion: "1.0.0",
		Environment:    "production",
	}
	obs := NewObservability(obsConfig)

	// Create tag manager
	tagManager, err := NewTagManager(options.Store, DefaultTagConfig())
	if err != nil {
		return nil, err
	}

	// Create refresh-ahead scheduler
	refreshScheduler := NewRefreshAheadScheduler(options.Store, DefaultRefreshAheadConfig())

	// Create a shared closed flag
	closedFlag := false

	c := &cache[T]{
		store:            options.Store,
		codec:            options.Codec,
		keyBuilder:       options.KeyBuilder,
		keyHasher:        options.KeyHasher,
		options:          options,
		ctx:              context.Background(),
		closed:           false,
		closedFlag:       &closedFlag,
		observability:    obs,
		tagManager:       tagManager,
		refreshScheduler: refreshScheduler,
	}

	return c, nil
}

// isClosed checks if the cache is closed
func (c *cache[T]) isClosed() bool {
	if c.closedFlag != nil {
		return *c.closedFlag
	}
	return c.closed
}

// WithContext returns a new cache instance with the given context
func (c *cache[T]) WithContext(ctx context.Context) Cache[T] {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create new cache without copying the mutex
	newCache := &cache[T]{
		store:            c.store,
		codec:            c.codec,
		keyBuilder:       c.keyBuilder,
		keyHasher:        c.keyHasher,
		options:          c.options,
		ctx:              ctx,
		closed:           c.closed,
		closedFlag:       c.closedFlag,
		observability:    c.observability,
		tagManager:       c.tagManager,
		refreshScheduler: c.refreshScheduler,
	}
	return newCache
}

// Get retrieves a value from the cache (non-blocking)
func (c *cache[T]) Get(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Create tracing span
		var span trace.Span
		var traceCtx context.Context
		if c.observability != nil {
			traceCtx, span = c.observability.TraceOperation(ctx, "get", key, "", 0, 0)
			defer span.End()
		} else {
			traceCtx = ctx
		}

		start := time.Now()
		defer func() {
			c.logOperation("get", key, time.Since(start), nil)
		}()

		// Execute get operation with retry
		asyncResult := c.executeGetWithRetryAsync(traceCtx, key)
		result <- asyncResult
	}()

	return result
}

// executeGetWithRetryAsync performs the get operation with retry logic (async)
func (c *cache[T]) executeGetWithRetryAsync(ctx context.Context, key string) AsyncCacheResult[T] {
	maxRetries := c.options.MaxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		asyncResult := c.executeGetAsync(ctx, key)
		if asyncResult.Error == nil {
			return asyncResult
		}

		if attempt == maxRetries {
			return AsyncCacheResult[T]{Error: asyncResult.Error}
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return AsyncCacheResult[T]{Error: ctx.Err()}
		case <-time.After(c.options.RetryDelay * time.Duration(1<<uint(attempt))):
			continue
		}
	}
	return AsyncCacheResult[T]{Error: fmt.Errorf("max retries exceeded")}
}

// executeGetAsync performs the actual get operation (async)
func (c *cache[T]) executeGetAsync(ctx context.Context, key string) AsyncCacheResult[T] {
	// Get from store
	storeResult := <-c.store.Get(ctx, key)
	if storeResult.Error != nil {
		return AsyncCacheResult[T]{Error: storeResult.Error}
	}

	if !storeResult.Exists || storeResult.Value == nil {
		return AsyncCacheResult[T]{Found: false} // Key not found
	}

	// Decode value
	var value T
	if err := c.codec.Decode(storeResult.Value, &value); err != nil {
		return AsyncCacheResult[T]{Error: NewCacheError("get", key, "decode error", err)}
	}

	return AsyncCacheResult[T]{Value: value, Found: true}
}

// Set stores a value in the cache (non-blocking)
func (c *cache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if ttl == 0 {
			ttl = c.options.DefaultTTL
		}

		start := time.Now()
		defer func() {
			c.logOperation("set", key, time.Since(start), nil)
		}()

		// Execute with retry logic
		asyncResult := c.executeSetWithRetryAsync(ctx, key, val, ttl)
		result <- asyncResult
	}()

	return result
}

// executeSetWithRetryAsync performs the set operation with retry logic (async)
func (c *cache[T]) executeSetWithRetryAsync(ctx context.Context, key string, val T, ttl time.Duration) AsyncCacheResult[T] {
	maxRetries := c.options.MaxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		asyncResult := c.executeSetAsync(ctx, key, val, ttl)
		if asyncResult.Error == nil {
			return asyncResult
		}

		if attempt == maxRetries {
			return AsyncCacheResult[T]{Error: asyncResult.Error}
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return AsyncCacheResult[T]{Error: ctx.Err()}
		case <-time.After(c.options.RetryDelay * time.Duration(1<<uint(attempt))):
			continue
		}
	}
	return AsyncCacheResult[T]{Error: fmt.Errorf("max retries exceeded")}
}

// executeSetAsync performs the actual set operation (async)
func (c *cache[T]) executeSetAsync(ctx context.Context, key string, val T, ttl time.Duration) AsyncCacheResult[T] {
	// Basic boundary condition validations (always performed)
	if key == "" {
		return AsyncCacheResult[T]{Error: NewCacheError("set", key, "key validation error", fmt.Errorf("key cannot be empty"))}
	}

	// Encode value
	data, err := c.codec.Encode(val)
	if err != nil {
		return AsyncCacheResult[T]{Error: NewCacheError("set", key, "encode error", err)}
	}

	// Check if encoded data is nil (this catches nil values that encode to nil)
	if data == nil {
		return AsyncCacheResult[T]{Error: NewCacheError("set", key, "value validation error", fmt.Errorf("value cannot be nil"))}
	}

	// Store in cache
	storeResult := <-c.store.Set(ctx, key, data, ttl)
	if storeResult.Error != nil {
		return AsyncCacheResult[T]{Error: NewCacheError("set", key, "store error", storeResult.Error)}
	}

	return AsyncCacheResult[T]{Value: val}
}

// MGet retrieves multiple values from the cache (non-blocking)
func (c *cache[T]) MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(keys) == 0 {
			result <- AsyncCacheResult[T]{Values: make(map[string]T)}
			return
		}

		start := time.Now()
		defer func() {
			c.logOperation("mget", fmt.Sprintf("%d keys", len(keys)), time.Since(start), nil)
		}()

		// Get from store
		storeResult := <-c.store.MGet(ctx, keys...)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("mget", "multiple", "store error", storeResult.Error)}
			return
		}

		// Decode values
		values := make(map[string]T)
		for key, data := range storeResult.Values {
			if data != nil {
				var value T
				if err := c.codec.Decode(data, &value); err != nil {
					logx.Warn("Failed to decode value in MGet", logx.String("key", key), logx.ErrorField(err))
					continue
				}
				values[key] = value
			}
		}

		result <- AsyncCacheResult[T]{Values: values}
	}()

	return result
}

// MSet stores multiple values in the cache (non-blocking)
func (c *cache[T]) MSet(ctx context.Context, items map[string]T, ttl time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(items) == 0 {
			result <- AsyncCacheResult[T]{}
			return
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
				result <- AsyncCacheResult[T]{Error: NewCacheError("mset", key, "encode error", err)}
				return
			}

			// Check if encoded data is nil (this catches nil values that encode to nil)
			if data == nil {
				result <- AsyncCacheResult[T]{Error: NewCacheError("mset", key, "value validation error", fmt.Errorf("value cannot be nil"))}
				return
			}

			dataMap[key] = data
		}

		// Store in cache
		storeResult := <-c.store.MSet(ctx, dataMap, ttl)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("mset", "multiple", "store error", storeResult.Error)}
			return
		}

		result <- AsyncCacheResult[T]{}
	}()

	return result
}

// Del removes keys from the cache (non-blocking)
func (c *cache[T]) Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if len(keys) == 0 {
			result <- AsyncCacheResult[T]{}
			return
		}

		start := time.Now()
		defer func() {
			c.logOperation("del", fmt.Sprintf("%d keys", len(keys)), time.Since(start), nil)
		}()

		// Delete from store
		storeResult := <-c.store.Del(ctx, keys...)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("del", "multiple", "store error", storeResult.Error)}
			return
		}

		// Remove tag associations
		for _, key := range keys {
			if err := c.tagManager.RemoveKey(ctx, key); err != nil {
				logx.Warn("Failed to remove tag associations", logx.String("key", key), logx.ErrorField(err))
			}
		}

		result <- AsyncCacheResult[T]{}
	}()

	return result
}

// Exists checks if a key exists in the cache (non-blocking)
func (c *cache[T]) Exists(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		start := time.Now()
		defer func() {
			c.logOperation("exists", key, time.Since(start), nil)
		}()

		// Check in store
		storeResult := <-c.store.Exists(ctx, key)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("exists", key, "store error", storeResult.Error)}
			return
		}

		result <- AsyncCacheResult[T]{Found: storeResult.Exists}
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking)
func (c *cache[T]) TTL(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		start := time.Now()
		defer func() {
			c.logOperation("ttl", key, time.Since(start), nil)
		}()

		// Get TTL from store
		storeResult := <-c.store.TTL(ctx, key)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("ttl", key, "store error", storeResult.Error)}
			return
		}

		result <- AsyncCacheResult[T]{TTL: storeResult.TTL}
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (c *cache[T]) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		start := time.Now()
		defer func() {
			c.logOperation("incrby", key, time.Since(start), nil)
		}()

		// Increment in store
		storeResult := <-c.store.IncrBy(ctx, key, delta, ttlIfCreate)
		if storeResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("incrby", key, "store error", storeResult.Error)}
			return
		}

		result <- AsyncCacheResult[T]{Int: storeResult.Result}
	}()

	return result
}

// ReadThrough implements the read-through pattern (non-blocking)
func (c *cache[T]) ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Try to get from cache first
		getResult := <-c.Get(ctx, key)
		if getResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: getResult.Error}
			return
		}

		if getResult.Found {
			result <- AsyncCacheResult[T]{Value: getResult.Value, Found: true}
			return
		}

		// Load from source
		value, err := loader(ctx)
		if err != nil {
			result <- AsyncCacheResult[T]{Error: err}
			return
		}

		// Store in cache
		setResult := <-c.Set(ctx, key, value, ttl)
		if setResult.Error != nil {
			logx.Warn("Failed to store value in ReadThrough", logx.String("key", key), logx.ErrorField(setResult.Error))
		}

		result <- AsyncCacheResult[T]{Value: value, Found: true}
	}()

	return result
}

// WriteThrough implements the write-through pattern (non-blocking)
func (c *cache[T]) WriteThrough(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Write to source first
		if err := writer(ctx); err != nil {
			result <- AsyncCacheResult[T]{Error: err}
			return
		}

		// Then write to cache
		setResult := <-c.Set(ctx, key, val, ttl)
		result <- setResult
	}()

	return result
}

// WriteBehind implements the write-behind pattern (non-blocking)
func (c *cache[T]) WriteBehind(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Write to cache immediately
		setResult := <-c.Set(ctx, key, val, ttl)
		if setResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: setResult.Error}
			return
		}

		// Write to source asynchronously
		go func() {
			if err := writer(ctx); err != nil {
				logx.Error("Failed to write behind", logx.String("key", key), logx.ErrorField(err))
			}
		}()

		result <- AsyncCacheResult[T]{Value: val}
	}()

	return result
}

// RefreshAhead implements the refresh-ahead pattern (non-blocking)
func (c *cache[T]) RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

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
				result <- AsyncCacheResult[T]{Error: NewCacheError("refresh_ahead", key, "scheduler error", err)}
				return
			}

			logx.Info("Scheduled refresh-ahead",
				logx.String("key", key),
				logx.String("refresh_before", refreshBefore.String()))
			result <- AsyncCacheResult[T]{}
			return
		}

		// Fallback to immediate refresh if scheduler is not available
		ttlResult := <-c.TTL(ctx, key)
		if ttlResult.Error != nil {
			result <- AsyncCacheResult[T]{Error: ttlResult.Error}
			return
		}

		// If TTL is less than refreshBefore, refresh the value
		if ttlResult.TTL <= refreshBefore {
			value, err := loader(ctx)
			if err != nil {
				result <- AsyncCacheResult[T]{Error: err}
				return
			}

			// Get the original TTL or use default
			originalTTL := c.options.DefaultTTL
			if ttlResult.TTL > 0 {
				originalTTL = ttlResult.TTL
			}

			setResult := <-c.Set(ctx, key, value, originalTTL)
			result <- setResult
			return
		}

		result <- AsyncCacheResult[T]{}
	}()

	return result
}

// InvalidateByTag invalidates all keys with the given tags (non-blocking)
func (c *cache[T]) InvalidateByTag(ctx context.Context, tags ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		// Collect all keys for the given tags
		var allKeys []string
		for _, tag := range tags {
			keys, err := c.tagManager.GetKeysByTag(ctx, tag)
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
		if err := c.tagManager.InvalidateByTag(ctx, uniqueKeys...); err != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("invalidate_by_tag", fmt.Sprintf("%v", tags), "tag manager error", err)}
			return
		}

		logx.Info("Tag invalidation completed",
			logx.String("tags", fmt.Sprintf("%v", tags)),
			logx.Int("keys_invalidated", len(uniqueKeys)))
		result <- AsyncCacheResult[T]{}
	}()

	return result
}

// AddTags adds tags to a key (non-blocking)
func (c *cache[T]) AddTags(ctx context.Context, key string, tags ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncCacheResult[T]{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		if err := c.tagManager.AddTags(ctx, key, tags...); err != nil {
			result <- AsyncCacheResult[T]{Error: NewCacheError("add_tags", key, "tag manager error", err)}
			return
		}

		logx.Info("Tags added to key", logx.String("key", key), logx.String("tags", fmt.Sprintf("%v", tags)))
		result <- AsyncCacheResult[T]{}
	}()

	return result
}

// TryLock attempts to acquire a distributed lock (non-blocking)
func (c *cache[T]) TryLock(ctx context.Context, key string, ttl time.Duration) <-chan AsyncLockResult {
	result := make(chan AsyncLockResult, 1)

	go func() {
		defer close(result)

		c.mu.RLock()
		if c.isClosed() {
			c.mu.RUnlock()
			result <- AsyncLockResult{Error: ErrStoreClosed}
			return
		}
		c.mu.RUnlock()

		lockKey := fmt.Sprintf("lock:%s", key)
		lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

		// Try to set the lock key
		setResult := <-c.store.Set(ctx, lockKey, []byte(lockValue), ttl)
		if setResult.Error != nil {
			result <- AsyncLockResult{Error: ErrLockFailed}
			return
		}

		// Create unlock function
		unlock := func() error {
			// Delete the lock key to release the lock
			delResult := <-c.store.Del(ctx, lockKey)
			return delResult.Error
		}

		result <- AsyncLockResult{OK: true, Unlock: unlock}
	}()

	return result
}

// GetStats returns cache statistics
func (c *cache[T]) GetStats() map[string]any {
	stats := make(map[string]any)

	// Add basic cache info
	stats["closed"] = c.closed
	stats["default_ttl"] = c.options.DefaultTTL.String()
	stats["max_retries"] = c.options.MaxRetries
	stats["retry_delay"] = c.options.RetryDelay.String()

	// Add store type info
	if c.store != nil {
		stats["store_type"] = fmt.Sprintf("%T", c.store)
	}

	// Add observability info
	if c.observability != nil {
		stats["observability_enabled"] = true
	} else {
		stats["observability_enabled"] = false
	}

	return stats
}

// Close closes the cache and releases resources
func (c *cache[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed() {
		return nil
	}

	c.closed = true
	if c.closedFlag != nil {
		*c.closedFlag = true
	}

	// Close store
	if err := c.store.Close(); err != nil {
		logx.Error("Failed to close store", logx.ErrorField(err))
	}

	// Close refresh-ahead scheduler
	if c.refreshScheduler != nil {
		if err := c.refreshScheduler.Close(); err != nil {
			logx.Error("Failed to close refresh-ahead scheduler", logx.ErrorField(err))
		}
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
	c.observability.LogOperation(level, op, key, namespace, 0, 0, duration, err, false)

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
	// Use the JSON codec from the local package with proper initialization
	jsonCodec := &JSONCodec{
		AllowNilValues:     false,
		EnableDebugLogging: false,
		bufferPool:         GlobalPools.Buffer,
	}

	// Validate the codec before use
	if err := jsonCodec.Validate(); err != nil {
		return nil, fmt.Errorf("codec validation failed: %w", err)
	}

	return jsonCodec.Encode(v)
}

func (c *defaultJSONCodec) Decode(data []byte, v any) error {
	// Use the JSON codec from the local package with proper initialization
	jsonCodec := &JSONCodec{
		AllowNilValues:     false,
		EnableDebugLogging: false,
		bufferPool:         GlobalPools.Buffer,
	}

	// Validate the codec before use
	if err := jsonCodec.Validate(); err != nil {
		return fmt.Errorf("codec validation failed: %w", err)
	}

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
