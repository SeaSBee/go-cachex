package cachex

import (
	"context"
	"time"
)

// Cache defines the main interface for the cache layer - Non-blocking Only
//
// IMPORTANT: Resource Management
//   - The WithContext method returns a new cache instance that shares the same underlying
//     resources (store, codec, etc.) but with a different context. This is NOT a deep copy.
//   - All cache instances created from the same original cache share the same resources.
//   - Only call Close() on the original cache instance to avoid resource leaks.
//   - Multiple calls to Close() are safe and will only close resources once.
type Cache[T any] interface {
	// WithContext returns a new cache instance with the given context.
	// This method creates a lightweight wrapper that shares the same underlying
	// resources (store, codec, observability, etc.) but uses the provided context
	// for all operations. The returned cache should NOT be closed independently.
	WithContext(ctx context.Context) Cache[T]

	// Non-blocking operations only
	Get(ctx context.Context, key string) <-chan AsyncCacheResult[T]
	Set(ctx context.Context, key string, val T, ttl time.Duration) <-chan AsyncCacheResult[T]
	MGet(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T]
	MSet(ctx context.Context, items map[string]T, ttl time.Duration) <-chan AsyncCacheResult[T]
	Del(ctx context.Context, keys ...string) <-chan AsyncCacheResult[T]
	Exists(ctx context.Context, key string) <-chan AsyncCacheResult[T]
	TTL(ctx context.Context, key string) <-chan AsyncCacheResult[T]
	IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncCacheResult[T]

	// Non-blocking caching patterns
	ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T]
	WriteThrough(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T]
	WriteBehind(ctx context.Context, key string, val T, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncCacheResult[T]
	RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (T, error)) <-chan AsyncCacheResult[T]

	// Non-blocking Namespaces/Tags
	InvalidateByTag(ctx context.Context, tags ...string) <-chan AsyncCacheResult[T]
	AddTags(ctx context.Context, key string, tags ...string) <-chan AsyncCacheResult[T]

	// Non-blocking Distributed Locks
	TryLock(ctx context.Context, key string, ttl time.Duration) <-chan AsyncLockResult

	// Statistics
	GetStats() map[string]any

	// Close releases resources. This method is safe to call multiple times.
	// Only call Close() on the original cache instance, not on instances
	// returned by WithContext().
	Close() error
}

// AnyCache defines a cache that can work with any type using type assertions - Non-blocking Only
//
// IMPORTANT: Resource Management
//   - The WithContext method returns a new cache instance that shares the same underlying
//     resources (store, codec, etc.) but with a different context. This is NOT a deep copy.
//   - All cache instances created from the same original cache share the same resources.
//   - Only call Close() on the original cache instance to avoid resource leaks.
//   - Multiple calls to Close() are safe and will only close resources once.
type AnyCache interface {
	// WithContext returns a new cache instance with the given context.
	// This method creates a lightweight wrapper that shares the same underlying
	// resources (store, codec, observability, etc.) but uses the provided context
	// for all operations. The returned cache should NOT be closed independently.
	WithContext(ctx context.Context) AnyCache

	// Non-blocking operations only
	Get(ctx context.Context, key string) <-chan AsyncAnyCacheResult
	Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan AsyncAnyCacheResult
	MGet(ctx context.Context, keys ...string) <-chan AsyncAnyCacheResult
	MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan AsyncAnyCacheResult
	Del(ctx context.Context, keys ...string) <-chan AsyncAnyCacheResult
	Exists(ctx context.Context, key string) <-chan AsyncAnyCacheResult
	TTL(ctx context.Context, key string) <-chan AsyncAnyCacheResult
	IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncAnyCacheResult

	// Non-blocking caching patterns
	ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (any, error)) <-chan AsyncAnyCacheResult
	WriteThrough(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncAnyCacheResult
	WriteBehind(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncAnyCacheResult
	RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (any, error)) <-chan AsyncAnyCacheResult

	// Non-blocking Namespaces/Tags
	InvalidateByTag(ctx context.Context, tags ...string) <-chan AsyncAnyCacheResult
	AddTags(ctx context.Context, key string, tags ...string) <-chan AsyncAnyCacheResult

	// Non-blocking Distributed Locks
	TryLock(ctx context.Context, key string, ttl time.Duration) <-chan AsyncLockResult

	// Statistics
	GetStats() map[string]any

	// Close releases resources. This method is safe to call multiple times.
	// Only call Close() on the original cache instance, not on instances
	// returned by WithContext().
	Close() error
}

// Store defines the interface for cache storage backends - Non-blocking Only
type Store interface {
	Get(ctx context.Context, key string) <-chan AsyncResult
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan AsyncResult
	MGet(ctx context.Context, keys ...string) <-chan AsyncResult
	MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan AsyncResult
	Del(ctx context.Context, keys ...string) <-chan AsyncResult
	Exists(ctx context.Context, key string) <-chan AsyncResult
	TTL(ctx context.Context, key string) <-chan AsyncResult
	IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncResult
	Close() error
}

// AsyncCacheResult represents the result of an async cache operation
type AsyncCacheResult[T any] struct {
	Value  T
	Values map[string]T
	Int    int64
	Found  bool
	TTL    time.Duration
	Error  error
}

// AsyncAnyCacheResult represents the result of an async any cache operation
type AsyncAnyCacheResult struct {
	Value  any
	Values map[string]any
	Found  bool
	TTL    time.Duration
	Error  error
}

// AsyncLockResult represents the result of an async lock operation
type AsyncLockResult struct {
	Unlock func() error
	OK     bool
	Error  error
}

// AsyncResult represents the result of an async store operation
type AsyncResult struct {
	Value  []byte
	Values map[string][]byte
	Result int64
	Exists bool
	TTL    time.Duration
	Error  error
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

// Options defines configuration options for the cache
type Options struct {
	Store         Store
	Codec         Codec
	KeyBuilder    KeyBuilder
	KeyHasher     KeyHasher
	DefaultTTL    time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	Observability ObservabilityConfig

	AllowNilValues bool // Whether to allow nil values in the cache
}

// ObservabilityConfig defines observability settings
type ObservabilityConfig struct {
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing bool `yaml:"enable_tracing" json:"enable_tracing"`
	EnableLogging bool `yaml:"enable_logging" json:"enable_logging"`
}

// Option is a functional option for configuring the cache
type Option func(*Options)

// WithStore sets the storage backend
func WithStore(store Store) Option {
	return func(o *Options) {
		o.Store = store
	}
}

// WithCodec sets the serialization codec
func WithCodec(codec Codec) Option {
	return func(o *Options) {
		o.Codec = codec
	}
}

// WithKeyBuilder sets the key builder
func WithKeyBuilder(builder KeyBuilder) Option {
	return func(o *Options) {
		o.KeyBuilder = builder
	}
}

// WithKeyHasher sets the key hasher
func WithKeyHasher(hasher KeyHasher) Option {
	return func(o *Options) {
		o.KeyHasher = hasher
	}
}

// WithDefaultTTL sets the default TTL
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(max int) Option {
	return func(o *Options) {
		o.MaxRetries = max
	}
}

// WithRetryDelay sets the retry delay
func WithRetryDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = delay
	}
}

// WithObservability sets observability configuration
func WithObservability(config ObservabilityConfig) Option {
	return func(o *Options) {
		o.Observability = config
	}
}

// WithNilValues allows nil values to be stored in the cache
func WithNilValues(allow bool) Option {
	return func(o *Options) {
		o.AllowNilValues = allow
	}
}
