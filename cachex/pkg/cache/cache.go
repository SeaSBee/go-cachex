package cache

import (
	"context"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/cb"
	"github.com/SeaSBee/go-cachex/cachex/internal/dlq"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// Cache defines the main interface for the cache layer
type Cache[T any] interface {
	// WithContext returns a new cache instance with the given context
	WithContext(ctx context.Context) Cache[T]

	// Basic operations
	Get(key string) (T, bool, error)
	Set(key string, val T, ttl time.Duration) error
	MGet(keys ...string) (map[string]T, error)
	MSet(items map[string]T, ttl time.Duration) error
	Del(keys ...string) error
	Exists(key string) (bool, error)
	TTL(key string) (time.Duration, error)
	IncrBy(key string, delta int64, ttlIfCreate time.Duration) (int64, error)

	// Caching patterns
	ReadThrough(key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) (T, error)
	WriteThrough(key string, val T, ttl time.Duration, writer func(ctx context.Context) error) error
	WriteBehind(key string, val T, ttl time.Duration, writer func(ctx context.Context) error) error
	RefreshAhead(key string, refreshBefore time.Duration, loader func(ctx context.Context) (T, error)) error

	// Namespaces/Tags
	InvalidateByTag(tags ...string) error
	AddTags(key string, tags ...string) error

	// Distributed Locks
	TryLock(key string, ttl time.Duration) (unlock func() error, ok bool, err error)

	// Pub/Sub Invalidation
	PublishInvalidation(keys ...string) error
	SubscribeInvalidations(handler func(keys ...string)) (close func() error, err error)

	// Circuit breaker management
	GetCircuitBreakerStats() cb.Stats
	GetCircuitBreakerState() cb.State
	ForceOpenCircuitBreaker()
	ForceCloseCircuitBreaker()

	// Resilience features
	GetDeadLetterQueueStats() *dlq.Metrics
	GetBloomFilterStats() map[string]interface{}

	// Close releases resources
	Close() error
}

// AnyCache defines a cache that can work with any type using type assertions
type AnyCache interface {
	// WithContext returns a new cache instance with the given context
	WithContext(ctx context.Context) AnyCache

	// Basic operations
	Get(key string) (any, bool, error)
	Set(key string, val any, ttl time.Duration) error
	MGet(keys ...string) (map[string]any, error)
	MSet(items map[string]any, ttl time.Duration) error
	Del(keys ...string) error
	Exists(key string) (bool, error)
	TTL(key string) (time.Duration, error)
	IncrBy(key string, delta int64, ttlIfCreate time.Duration) (int64, error)

	// Caching patterns
	ReadThrough(key string, ttl time.Duration, loader func(ctx context.Context) (any, error)) (any, error)
	WriteThrough(key string, val any, ttl time.Duration, writer func(ctx context.Context) error) error
	WriteBehind(key string, val any, ttl time.Duration, writer func(ctx context.Context) error) error
	RefreshAhead(key string, refreshBefore time.Duration, loader func(ctx context.Context) (any, error)) error

	// Namespaces/Tags
	InvalidateByTag(tags ...string) error
	AddTags(key string, tags ...string) error

	// Distributed Locks
	TryLock(key string, ttl time.Duration) (unlock func() error, ok bool, err error)

	// Pub/Sub Invalidation
	PublishInvalidation(keys ...string) error
	SubscribeInvalidations(handler func(keys ...string)) (close func() error, err error)

	// Circuit breaker management
	GetCircuitBreakerStats() cb.Stats
	GetCircuitBreakerState() cb.State
	ForceOpenCircuitBreaker()
	ForceCloseCircuitBreaker()

	// Resilience features
	GetDeadLetterQueueStats() *dlq.Metrics
	GetBloomFilterStats() map[string]interface{}

	// Close releases resources
	Close() error
}

// Store defines the interface for cache storage backends
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	MGet(ctx context.Context, keys ...string) (map[string][]byte, error)
	MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error)
	Close() error
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
	Store                 Store
	Codec                 Codec
	KeyBuilder            KeyBuilder
	KeyHasher             KeyHasher
	DefaultTTL            time.Duration
	MaxRetries            int
	RetryDelay            time.Duration
	CircuitBreaker        CircuitBreakerConfig
	RateLimit             RateLimitConfig
	Security              SecurityConfig
	Observability         ObservabilityConfig
	Bulkhead              redisstore.BulkheadConfig
	EnableDeadLetterQueue bool
	EnableBloomFilter     bool
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	Threshold   int
	Timeout     time.Duration
	HalfOpenMax int
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	EncryptionKey      string
	RedactLogs         bool
	EnableTLS          bool
	InsecureSkipVerify bool
}

// ObservabilityConfig defines observability settings
type ObservabilityConfig struct {
	EnableMetrics bool
	EnableTracing bool
	EnableLogging bool
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

// WithCircuitBreaker sets circuit breaker configuration
func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(o *Options) {
		o.CircuitBreaker = config
	}
}

// WithRateLimit sets rate limiting configuration
func WithRateLimit(config RateLimitConfig) Option {
	return func(o *Options) {
		o.RateLimit = config
	}
}

// WithSecurity sets security configuration
func WithSecurity(config SecurityConfig) Option {
	return func(o *Options) {
		o.Security = config
	}
}

// WithDeadLetterQueue enables dead-letter queue for failed operations
func WithDeadLetterQueue() Option {
	return func(o *Options) {
		o.EnableDeadLetterQueue = true
	}
}

// WithBloomFilter enables bloom filter for reducing misses to backing store
func WithBloomFilter() Option {
	return func(o *Options) {
		o.EnableBloomFilter = true
	}
}

// WithObservability sets observability configuration
func WithObservability(config ObservabilityConfig) Option {
	return func(o *Options) {
		o.Observability = config
	}
}

func WithBulkhead(config redisstore.BulkheadConfig) Option {
	return func(o *Options) {
		o.Bulkhead = config
	}
}
