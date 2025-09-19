package cachex

import (
	"context"
	"fmt"
	"time"
)

// AsyncCacheResult represents the result of an asynchronous cache operation
type AsyncCacheResult[T any] struct {
	// Value contains the retrieved value for single-value operations
	Value T

	// Values contains multiple values for batch operations (MGet, MSet)
	Values map[string]T

	// Found indicates whether the key was found (for Get, Exists operations)
	Found bool

	// TTL contains the time-to-live for TTL operations
	TTL time.Duration

	// Int contains integer results for IncrBy operations
	Int int64

	// Count contains count results for operations like Del, Flush
	Count int64

	// Keys contains key lists for Keys operations
	Keys []string

	// Size contains size information for Size operations
	Size int64

	// Error contains any error that occurred during the operation
	Error error

	// Metadata contains additional operation metadata
	Metadata map[string]interface{}
}

type Cache interface {
	WithContext(ctx context.Context) Cache

	// Non-blocking operations only
	Get(ctx context.Context, key string) <-chan AsyncCacheResult[any]
	Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan AsyncCacheResult[any]
	MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[any]
	MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan AsyncCacheResult[any]
	Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[any]
	Exists(ctx context.Context, key string) <-chan AsyncCacheResult[any]
	TTL(ctx context.Context, key string) <-chan AsyncCacheResult[any]
	IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[any]
	Close() error
}

// TypedCache provides type-safe operations on top of the base Cache interface
type TypedCache[T any] struct {
	cache Cache
}

// NewTypedCache creates a type-safe wrapper around a Cache instance
func NewTypedCache[T any](cache Cache) *TypedCache[T] {
	return &TypedCache[T]{cache: cache}
}

// WithContext returns a new typed cache instance with the given context
func (tc *TypedCache[T]) WithContext(ctx context.Context) *TypedCache[T] {
	return &TypedCache[T]{cache: tc.cache.WithContext(ctx)}
}

// Get retrieves a value from the cache (non-blocking)
func (tc *TypedCache[T]) Get(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Get from base cache
		baseResult := <-tc.cache.Get(ctx, key)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		// Type assert the value if found
		if baseResult.Found && baseResult.Value != nil {
			if typedValue, ok := baseResult.Value.(T); ok {
				typedResult.Value = typedValue
			} else {
				typedResult.Error = fmt.Errorf("type assertion failed: expected %T, got %T", *new(T), baseResult.Value)
				typedResult.Found = false
			}
		}

		result <- typedResult
	}()

	return result
}

// Set stores a value in the cache (non-blocking)
func (tc *TypedCache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Set in base cache
		baseResult := <-tc.cache.Set(ctx, key, val, ttl)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Value:    val,
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// MGet retrieves multiple values from the cache (non-blocking)
func (tc *TypedCache[T]) MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Get from base cache
		baseResult := <-tc.cache.MGet(ctx, keys...)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		// Type assert the values if found
		if baseResult.Values != nil {
			typedValues := make(map[string]T)
			for key, value := range baseResult.Values {
				if typedValue, ok := value.(T); ok {
					typedValues[key] = typedValue
				}
				// Skip values that don't match the expected type
			}
			typedResult.Values = typedValues
		}

		result <- typedResult
	}()

	return result
}

// MSet stores multiple values in the cache (non-blocking)
func (tc *TypedCache[T]) MSet(ctx context.Context, items map[string]T, ttl time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Convert to any map for base cache
		anyItems := make(map[string]any)
		for key, value := range items {
			anyItems[key] = value
		}

		// Set in base cache
		baseResult := <-tc.cache.MSet(ctx, anyItems, ttl)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// Del removes keys from the cache (non-blocking)
func (tc *TypedCache[T]) Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Delete from base cache
		baseResult := <-tc.cache.Del(ctx, keys...)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// Exists checks if a key exists in the cache (non-blocking)
func (tc *TypedCache[T]) Exists(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Check in base cache
		baseResult := <-tc.cache.Exists(ctx, key)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// TTL gets the time to live of a key (non-blocking)
func (tc *TypedCache[T]) TTL(ctx context.Context, key string) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Get TTL from base cache
		baseResult := <-tc.cache.TTL(ctx, key)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// IncrBy increments a key by the given delta (non-blocking)
func (tc *TypedCache[T]) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[T] {
	result := make(chan AsyncCacheResult[T], 1)

	go func() {
		defer close(result)

		// Increment in base cache
		baseResult := <-tc.cache.IncrBy(ctx, key, delta, ttlIfCreate)

		// Convert to typed result
		typedResult := AsyncCacheResult[T]{
			Found:    baseResult.Found,
			TTL:      baseResult.TTL,
			Int:      baseResult.Int,
			Count:    baseResult.Count,
			Keys:     baseResult.Keys,
			Size:     baseResult.Size,
			Error:    baseResult.Error,
			Metadata: baseResult.Metadata,
		}

		result <- typedResult
	}()

	return result
}

// Close closes the cache and releases resources
func (tc *TypedCache[T]) Close() error {
	return tc.cache.Close()
}

// Codec defines the interface for serialization/deserialization
type Codec interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}

// KeyBuilder defines the interface for key generation
type KeyBuilder interface {
	Build(entity, id string) string
	BuildList(entity string, filters map[string]any) string
	BuildComposite(entityA, idA, entityB, idB string) string
	BuildSession(sid string) string
}

// KeyHasher defines the interface for key hashing
type KeyHasher interface {
	Hash(data string) string
}
