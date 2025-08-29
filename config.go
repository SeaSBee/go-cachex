package cachex

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheConfig represents the main configuration structure
type CacheConfig struct {
	// Cache store configuration - only one can be active
	Memory    *MemoryConfig    `yaml:"memory" json:"memory" validate:"omitempty"`
	Redis     *RedisConfig     `yaml:"redis" json:"redis" validate:"omitempty"`
	Ristretto *RistrettoConfig `yaml:"ristretto" json:"ristretto" validate:"omitempty"`
	Layered   *LayeredConfig   `yaml:"layered" json:"layered" validate:"omitempty"`

	// General cache settings
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl" validate:"gte:0,lte:86400000000000"` // 0 to 24h in nanoseconds
	MaxRetries int           `yaml:"max_retries" json:"max_retries" validate:"gte:0,lte:10"`
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay" validate:"gte:0,lte:60000000000"` // 0 to 1min in nanoseconds

	// Codec settings
	Codec string `yaml:"codec" json:"codec" validate:"omitempty,oneof:json msgpack"` // "json" or "msgpack"

	// Observability settings
	Observability *ObservabilityConfig `yaml:"observability" json:"observability" validate:"omitempty"`

	// Tagging settings
	Tagging *TagConfig `yaml:"tagging" json:"tagging" validate:"omitempty"`

	// Refresh ahead settings
	RefreshAhead *RefreshAheadConfig `yaml:"refresh_ahead" json:"refresh_ahead" validate:"omitempty"`

	// GORM integration settings
	GORM *GormConfig `yaml:"gorm" json:"gorm" validate:"omitempty"`
}

// LoadConfig loads configuration from environment variables and YAML file
// Environment variables take precedence over YAML file
func LoadConfig(configPath string) (*CacheConfig, error) {
	config := &CacheConfig{}

	// Load from YAML file if provided
	if configPath != "" {
		if err := loadFromYAML(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to load YAML config: %w", err)
		}
	}

	// Override with environment variables
	if err := loadFromEnvironment(config); err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Final validation to ensure configuration is complete
	if err := validateConfigurationCompleteness(config); err != nil {
		return nil, fmt.Errorf("configuration completeness check failed: %w", err)
	}

	return config, nil
}

// loadFromYAML loads configuration from a YAML file
func loadFromYAML(configPath string, config *CacheConfig) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	return nil
}

// parseEnvInt safely parses an environment variable as an integer
func parseEnvInt(key string) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return 0, nil
	}
	return strconv.Atoi(val)
}

// parseEnvInt64 safely parses an environment variable as an int64
func parseEnvInt64(key string) (int64, error) {
	val := os.Getenv(key)
	if val == "" {
		return 0, nil
	}
	return strconv.ParseInt(val, 10, 64)
}

// parseEnvDuration safely parses an environment variable as a duration
func parseEnvDuration(key string) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return 0, nil
	}
	return time.ParseDuration(val)
}

// parseEnvBool safely parses an environment variable as a boolean
func parseEnvBool(key string) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return false, nil
	}
	return strconv.ParseBool(val)
}

// runWithContextCancellation runs a cache operation with proper context cancellation handling
func runWithContextCancellation[T any](ctx context.Context, operation func() <-chan T) <-chan T {
	result := make(chan T, 1)
	go func() {
		defer close(result)

		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			// Return a zero value with context error
			var zero T
			result <- zero
			return
		default:
		}

		// Create a channel to receive the operation result
		operationChan := operation()

		// Wait for either the operation result or context cancellation
		select {
		case operationResult := <-operationChan:
			result <- operationResult
		case <-ctx.Done():
			// Return a zero value with context error
			var zero T
			result <- zero
		}
	}()
	return result
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *CacheConfig) error {
	// Cache store type
	if storeType := os.Getenv("CACHEX_STORE_TYPE"); storeType != "" {
		switch strings.ToLower(storeType) {
		case "memory":
			if config.Memory == nil {
				config.Memory = &MemoryConfig{}
			}
		case "redis":
			if config.Redis == nil {
				config.Redis = &RedisConfig{}
			}
		case "ristretto":
			if config.Ristretto == nil {
				config.Ristretto = &RistrettoConfig{}
			}
		case "layered":
			if config.Layered == nil {
				config.Layered = &LayeredConfig{}
			}
		}
	}

	// Memory store configuration
	if config.Memory != nil {
		if maxSize, err := parseEnvInt("CACHEX_MEMORY_MAX_SIZE"); err != nil {
			return fmt.Errorf("invalid CACHEX_MEMORY_MAX_SIZE: %w", err)
		} else if maxSize > 0 {
			config.Memory.MaxSize = maxSize
		}

		if maxMemoryMB, err := parseEnvInt("CACHEX_MEMORY_MAX_MEMORY_MB"); err != nil {
			return fmt.Errorf("invalid CACHEX_MEMORY_MAX_MEMORY_MB: %w", err)
		} else if maxMemoryMB > 0 {
			config.Memory.MaxMemoryMB = maxMemoryMB
		}

		if ttl, err := parseEnvDuration("CACHEX_MEMORY_DEFAULT_TTL"); err != nil {
			return fmt.Errorf("invalid CACHEX_MEMORY_DEFAULT_TTL: %w", err)
		} else if ttl > 0 {
			config.Memory.DefaultTTL = ttl
		}

		if interval, err := parseEnvDuration("CACHEX_MEMORY_CLEANUP_INTERVAL"); err != nil {
			return fmt.Errorf("invalid CACHEX_MEMORY_CLEANUP_INTERVAL: %w", err)
		} else if interval > 0 {
			config.Memory.CleanupInterval = interval
		}

		if val := os.Getenv("CACHEX_MEMORY_EVICTION_POLICY"); val != "" {
			config.Memory.EvictionPolicy = EvictionPolicy(val)
		}
		// Note: Default values are now set centrally in setDefaultValues function
	}

	// Redis store configuration
	if config.Redis != nil {
		if val := os.Getenv("CACHEX_REDIS_ADDR"); val != "" {
			config.Redis.Addr = val
		}
		if val := os.Getenv("CACHEX_REDIS_PASSWORD"); val != "" {
			config.Redis.Password = val
		}
		if db, err := parseEnvInt("CACHEX_REDIS_DB"); err != nil {
			return fmt.Errorf("invalid CACHEX_REDIS_DB: %w", err)
		} else if db >= 0 {
			config.Redis.DB = db
		}
		if poolSize, err := parseEnvInt("CACHEX_REDIS_POOL_SIZE"); err != nil {
			return fmt.Errorf("invalid CACHEX_REDIS_POOL_SIZE: %w", err)
		} else if poolSize > 0 {
			config.Redis.PoolSize = poolSize
		}
		if timeout, err := parseEnvDuration("CACHEX_REDIS_DIAL_TIMEOUT"); err != nil {
			return fmt.Errorf("invalid CACHEX_REDIS_DIAL_TIMEOUT: %w", err)
		} else if timeout > 0 {
			config.Redis.DialTimeout = timeout
		}
		if timeout, err := parseEnvDuration("CACHEX_REDIS_READ_TIMEOUT"); err != nil {
			return fmt.Errorf("invalid CACHEX_REDIS_READ_TIMEOUT: %w", err)
		} else if timeout > 0 {
			config.Redis.ReadTimeout = timeout
		}
		if timeout, err := parseEnvDuration("CACHEX_REDIS_WRITE_TIMEOUT"); err != nil {
			return fmt.Errorf("invalid CACHEX_REDIS_WRITE_TIMEOUT: %w", err)
		} else if timeout > 0 {
			config.Redis.WriteTimeout = timeout
		}
		if val := os.Getenv("CACHEX_REDIS_TLS_ENABLED"); val != "" {
			if config.Redis.TLS == nil {
				config.Redis.TLS = &TLSConfig{}
			}
			if enabled, err := parseEnvBool("CACHEX_REDIS_TLS_ENABLED"); err != nil {
				return fmt.Errorf("invalid CACHEX_REDIS_TLS_ENABLED: %w", err)
			} else {
				config.Redis.TLS.Enabled = enabled
			}
		}
	}

	// Ristretto store configuration
	if config.Ristretto != nil {
		if maxItems, err := parseEnvInt("CACHEX_RISTRETTO_MAX_ITEMS"); err != nil {
			return fmt.Errorf("invalid CACHEX_RISTRETTO_MAX_ITEMS: %w", err)
		} else if maxItems > 0 {
			config.Ristretto.MaxItems = int64(maxItems)
		}
		if maxMemory, err := parseEnvInt64("CACHEX_RISTRETTO_MAX_MEMORY_BYTES"); err != nil {
			return fmt.Errorf("invalid CACHEX_RISTRETTO_MAX_MEMORY_BYTES: %w", err)
		} else if maxMemory > 0 {
			config.Ristretto.MaxMemoryBytes = maxMemory
		}
		if ttl, err := parseEnvDuration("CACHEX_RISTRETTO_DEFAULT_TTL"); err != nil {
			return fmt.Errorf("invalid CACHEX_RISTRETTO_DEFAULT_TTL: %w", err)
		} else if ttl > 0 {
			config.Ristretto.DefaultTTL = ttl
		}
	}

	// Layered store configuration
	if config.Layered != nil {
		if val := os.Getenv("CACHEX_LAYERED_READ_POLICY"); val != "" {
			config.Layered.ReadPolicy = ReadPolicy(val)
		}
		if val := os.Getenv("CACHEX_LAYERED_WRITE_POLICY"); val != "" {
			config.Layered.WritePolicy = WritePolicy(val)
		}
	}

	// General cache settings
	if ttl, err := parseEnvDuration("CACHEX_DEFAULT_TTL"); err != nil {
		return fmt.Errorf("invalid CACHEX_DEFAULT_TTL: %w", err)
	} else if ttl > 0 {
		config.DefaultTTL = ttl
	}
	if maxRetries, err := parseEnvInt("CACHEX_MAX_RETRIES"); err != nil {
		return fmt.Errorf("invalid CACHEX_MAX_RETRIES: %w", err)
	} else if maxRetries >= 0 {
		config.MaxRetries = maxRetries
	}
	if retryDelay, err := parseEnvDuration("CACHEX_RETRY_DELAY"); err != nil {
		return fmt.Errorf("invalid CACHEX_RETRY_DELAY: %w", err)
	} else if retryDelay > 0 {
		config.RetryDelay = retryDelay
	}

	// Codec settings
	if val := os.Getenv("CACHEX_CODEC"); val != "" {
		config.Codec = val
	}

	// Observability settings
	if config.Observability == nil {
		config.Observability = &ObservabilityConfig{}
	}
	if enabled, err := parseEnvBool("CACHEX_OBSERVABILITY_ENABLE_METRICS"); err != nil {
		return fmt.Errorf("invalid CACHEX_OBSERVABILITY_ENABLE_METRICS: %w", err)
	} else {
		config.Observability.EnableMetrics = enabled
	}
	if enabled, err := parseEnvBool("CACHEX_OBSERVABILITY_ENABLE_TRACING"); err != nil {
		return fmt.Errorf("invalid CACHEX_OBSERVABILITY_ENABLE_TRACING: %w", err)
	} else {
		config.Observability.EnableTracing = enabled
	}
	if enabled, err := parseEnvBool("CACHEX_OBSERVABILITY_ENABLE_LOGGING"); err != nil {
		return fmt.Errorf("invalid CACHEX_OBSERVABILITY_ENABLE_LOGGING: %w", err)
	} else {
		config.Observability.EnableLogging = enabled
	}

	// Tagging settings
	if config.Tagging == nil {
		config.Tagging = &TagConfig{}
	}
	if enabled, err := parseEnvBool("CACHEX_TAGGING_ENABLE_PERSISTENCE"); err != nil {
		return fmt.Errorf("invalid CACHEX_TAGGING_ENABLE_PERSISTENCE: %w", err)
	} else {
		config.Tagging.EnablePersistence = enabled
	}
	if ttl, err := parseEnvDuration("CACHEX_TAGGING_TAG_MAPPING_TTL"); err != nil {
		return fmt.Errorf("invalid CACHEX_TAGGING_TAG_MAPPING_TTL: %w", err)
	} else if ttl > 0 {
		config.Tagging.TagMappingTTL = ttl
	}

	// Refresh ahead settings
	if config.RefreshAhead == nil {
		config.RefreshAhead = &RefreshAheadConfig{}
	}
	if enabled, err := parseEnvBool("CACHEX_REFRESH_AHEAD_ENABLED"); err != nil {
		return fmt.Errorf("invalid CACHEX_REFRESH_AHEAD_ENABLED: %w", err)
	} else {
		config.RefreshAhead.Enabled = enabled
	}
	if interval, err := parseEnvDuration("CACHEX_REFRESH_AHEAD_REFRESH_INTERVAL"); err != nil {
		return fmt.Errorf("invalid CACHEX_REFRESH_AHEAD_REFRESH_INTERVAL: %w", err)
	} else if interval > 0 {
		config.RefreshAhead.RefreshInterval = interval
	}

	// GORM settings
	if config.GORM == nil {
		config.GORM = &GormConfig{}
	}
	if enabled, err := parseEnvBool("CACHEX_GORM_ENABLE_READ_THROUGH"); err != nil {
		return fmt.Errorf("invalid CACHEX_GORM_ENABLE_READ_THROUGH: %w", err)
	} else {
		config.GORM.EnableReadThrough = enabled
	}
	if enabled, err := parseEnvBool("CACHEX_GORM_ENABLE_INVALIDATION"); err != nil {
		return fmt.Errorf("invalid CACHEX_GORM_ENABLE_INVALIDATION: %w", err)
	} else {
		config.GORM.EnableInvalidation = enabled
	}

	return nil
}

// setDefaultValues sets default values for configuration fields
func setDefaultValues(config *CacheConfig) {
	// Set default TTL if not specified
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 5 * time.Minute
	}

	// Set default retry settings if not specified
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	// Set default codec if not specified
	if config.Codec == "" {
		config.Codec = "json"
	}

	// Set default observability settings if not specified
	if config.Observability == nil {
		config.Observability = &ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: false,
			EnableLogging: false,
		}
	}

	// Set default tagging settings if not specified
	if config.Tagging == nil {
		config.Tagging = &TagConfig{
			EnablePersistence: false,
			TagMappingTTL:     1 * time.Hour,
		}
	}

	// Set default refresh ahead settings if not specified
	if config.RefreshAhead == nil {
		config.RefreshAhead = &RefreshAheadConfig{
			Enabled:         false,
			RefreshInterval: 5 * time.Minute,
		}
	}

	// Set default GORM settings if not specified
	if config.GORM == nil {
		config.GORM = &GormConfig{
			EnableReadThrough:  false,
			EnableInvalidation: false,
		}
	}
}

// validateStoreConfiguration validates that exactly one store type is configured
func validateStoreConfiguration(config *CacheConfig) error {
	storeCount := 0
	if config.Memory != nil {
		storeCount++
	}
	if config.Redis != nil {
		storeCount++
	}
	if config.Ristretto != nil {
		storeCount++
	}
	if config.Layered != nil {
		storeCount++
	}

	if storeCount == 0 {
		return fmt.Errorf("no store configuration provided - must specify one of: memory, redis, ristretto, or layered")
	}
	if storeCount > 1 {
		return fmt.Errorf("multiple store configurations provided - only one store type can be active")
	}

	return nil
}

// validateConfig validates the configuration using go-validatorx with caching
func validateConfig(config *CacheConfig) error {
	// Set default values first
	setDefaultValues(config)

	// Validate store configuration
	if err := validateStoreConfiguration(config); err != nil {
		return err
	}

	// Check if validation cache is initialized
	if GlobalValidationCache == nil {
		// Fallback to direct validation or return error
		return fmt.Errorf("validation cache not initialized")
	}

	// Use validation cache for improved performance
	valid, err := GlobalValidationCache.IsValid(config)
	if !valid {
		return err
	}
	return nil
}

// validateConfigurationCompleteness performs final validation to ensure the configuration is complete
func validateConfigurationCompleteness(config *CacheConfig) error {
	// Validate that required fields are set based on the store type
	if config.Memory != nil {
		if config.Memory.MaxSize <= 0 {
			return fmt.Errorf("memory store max_size must be greater than 0")
		}
		if config.Memory.MaxMemoryMB <= 0 {
			return fmt.Errorf("memory store max_memory_mb must be greater than 0")
		}
		if config.Memory.DefaultTTL < 0 {
			return fmt.Errorf("memory store default_ttl cannot be negative")
		}
		if config.Memory.CleanupInterval < 0 {
			return fmt.Errorf("memory store cleanup_interval cannot be negative")
		}
	}

	if config.Redis != nil {
		if config.Redis.Addr == "" {
			return fmt.Errorf("redis store address is required")
		}
		if config.Redis.PoolSize <= 0 {
			return fmt.Errorf("redis store pool_size must be greater than 0")
		}
		if config.Redis.DialTimeout < 0 {
			return fmt.Errorf("redis store dial_timeout cannot be negative")
		}
		if config.Redis.ReadTimeout < 0 {
			return fmt.Errorf("redis store read_timeout cannot be negative")
		}
		if config.Redis.WriteTimeout < 0 {
			return fmt.Errorf("redis store write_timeout cannot be negative")
		}
	}

	if config.Ristretto != nil {
		if config.Ristretto.MaxItems <= 0 {
			return fmt.Errorf("ristretto store max_items must be greater than 0")
		}
		if config.Ristretto.MaxMemoryBytes <= 0 {
			return fmt.Errorf("ristretto store max_memory_bytes must be greater than 0")
		}
		if config.Ristretto.DefaultTTL < 0 {
			return fmt.Errorf("ristretto store default_ttl cannot be negative")
		}
	}

	if config.Layered != nil {
		if config.Layered.MemoryConfig == nil {
			return fmt.Errorf("layered store memory_config is required")
		}
		if config.Layered.SyncInterval < 0 {
			return fmt.Errorf("layered store sync_interval cannot be negative")
		}
		if config.Layered.MaxConcurrentSync <= 0 {
			return fmt.Errorf("layered store max_concurrent_sync must be greater than 0")
		}
		if config.Layered.AsyncOperationTimeout < 0 {
			return fmt.Errorf("layered store async_operation_timeout cannot be negative")
		}
		if config.Layered.BackgroundSyncTimeout < 0 {
			return fmt.Errorf("layered store background_sync_timeout cannot be negative")
		}
	}

	// Validate general settings
	if config.DefaultTTL < 0 {
		return fmt.Errorf("default_ttl cannot be negative")
	}
	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	if config.RetryDelay < 0 {
		return fmt.Errorf("retry_delay cannot be negative")
	}

	// Validate codec
	if config.Codec != "" && config.Codec != "json" && config.Codec != "msgpack" {
		return fmt.Errorf("codec must be either 'json' or 'msgpack'")
	}

	return nil
}

// CreateStoreFromConfig creates a store instance based on the configuration
func CreateStoreFromConfig(config *CacheConfig) (Store, error) {
	if config.Memory != nil {
		return NewMemoryStore(config.Memory)
	}
	if config.Redis != nil {
		return NewRedisStore(config.Redis)
	}
	if config.Ristretto != nil {
		return NewRistrettoStore(config.Ristretto)
	}
	if config.Layered != nil {
		// For layered store, we need to create the L2 store first
		// Use the L2 store configuration from the layered config if available
		var l2Store Store
		var err error

		if config.Layered.L2StoreConfig != nil {
			// Use configured L2 store
			l2Store, err = CreateStoreFromConfig(&CacheConfig{
				Redis:     config.Layered.L2StoreConfig.Redis,
				Ristretto: config.Layered.L2StoreConfig.Ristretto,
				Memory:    config.Layered.L2StoreConfig.Memory,
			})
		} else {
			// Fallback to default Redis config
			l2Store, err = NewRedisStore(DefaultRedisConfig())
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create L2 store for layered cache: %w", err)
		}
		return NewLayeredStore(l2Store, config.Layered)
	}

	return nil, fmt.Errorf("no valid store configuration found")
}

// CreateCodecFromConfig creates a codec instance based on the configuration
func CreateCodecFromConfig(config *CacheConfig) Codec {
	switch config.Codec {
	case "msgpack":
		return NewMessagePackCodec()
	case "json":
		fallthrough
	default:
		return NewJSONCodec()
	}
}

// NewFromConfig creates a new cache instance from configuration
func NewFromConfig[T any](config *CacheConfig) (Cache[T], error) {
	// Create store
	store, err := CreateStoreFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Create codec
	codec := CreateCodecFromConfig(config)

	// Create observability config
	obsConfig := ObservabilityConfig{}
	if config.Observability != nil {
		obsConfig.EnableMetrics = config.Observability.EnableMetrics
		obsConfig.EnableTracing = config.Observability.EnableTracing
		obsConfig.EnableLogging = config.Observability.EnableLogging
	}

	// Create cache with options
	opts := []Option{
		WithStore(store),
		WithCodec(codec),
		WithDefaultTTL(config.DefaultTTL),
		WithMaxRetries(config.MaxRetries),
		WithRetryDelay(config.RetryDelay),
		WithObservability(obsConfig),
	}

	return New[T](opts...)
}

// NewAnyFromConfig creates a new AnyCache instance from configuration
func NewAnyFromConfig(config *CacheConfig) (AnyCache, error) {
	// Create store
	store, err := CreateStoreFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Create codec
	codec := CreateCodecFromConfig(config)

	// Create observability config
	obsConfig := ObservabilityConfig{}
	if config.Observability != nil {
		obsConfig.EnableMetrics = config.Observability.EnableMetrics
		obsConfig.EnableTracing = config.Observability.EnableTracing
		obsConfig.EnableLogging = config.Observability.EnableLogging
	}

	// Create cache with options
	opts := []Option{
		WithStore(store),
		WithCodec(codec),
		WithDefaultTTL(config.DefaultTTL),
		WithMaxRetries(config.MaxRetries),
		WithRetryDelay(config.RetryDelay),
		WithObservability(obsConfig),
	}

	cache, err := New[any](opts...)
	if err != nil {
		return nil, err
	}

	// Create an AnyCache wrapper
	return &anyCacheWrapper{cache: cache}, nil
}

// anyCacheWrapper wraps Cache[any] to implement AnyCache interface
type anyCacheWrapper struct {
	cache Cache[any]
}

func (w *anyCacheWrapper) WithContext(ctx context.Context) AnyCache {
	return &anyCacheWrapper{cache: w.cache.WithContext(ctx)}
}

func (w *anyCacheWrapper) Get(ctx context.Context, key string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)

		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			result <- AsyncAnyCacheResult{Error: ctx.Err()}
			return
		default:
		}

		// Create a channel to receive the cache result
		cacheResultChan := w.cache.Get(ctx, key)

		// Wait for either the cache result or context cancellation
		select {
		case cacheResult := <-cacheResultChan:
			result <- AsyncAnyCacheResult{
				Value: cacheResult.Value,
				Found: cacheResult.Found,
				Error: cacheResult.Error,
			}
		case <-ctx.Done():
			result <- AsyncAnyCacheResult{Error: ctx.Err()}
		}
	}()
	return result
}

func (w *anyCacheWrapper) Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)

		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			result <- AsyncAnyCacheResult{Error: ctx.Err()}
			return
		default:
		}

		// Create a channel to receive the cache result
		cacheResultChan := w.cache.Set(ctx, key, val, ttl)

		// Wait for either the cache result or context cancellation
		select {
		case cacheResult := <-cacheResultChan:
			result <- AsyncAnyCacheResult{Error: cacheResult.Error}
		case <-ctx.Done():
			result <- AsyncAnyCacheResult{Error: ctx.Err()}
		}
	}()
	return result
}

func (w *anyCacheWrapper) MGet(ctx context.Context, keys ...string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.MGet(ctx, keys...)
		if cacheResult.Error != nil {
			result <- AsyncAnyCacheResult{Error: cacheResult.Error}
			return
		}

		// Convert map[string][]byte to map[string]any
		values := make(map[string]any)
		for k, v := range cacheResult.Values {
			values[k] = v
		}
		result <- AsyncAnyCacheResult{Values: values}
	}()
	return result
}

func (w *anyCacheWrapper) MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.MSet(ctx, items, ttl)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) Del(ctx context.Context, keys ...string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.Del(ctx, keys...)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) Exists(ctx context.Context, key string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.Exists(ctx, key)
		result <- AsyncAnyCacheResult{
			Found: cacheResult.Found,
			Error: cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) TTL(ctx context.Context, key string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.TTL(ctx, key)
		result <- AsyncAnyCacheResult{
			Found: cacheResult.Found,
			TTL:   cacheResult.TTL,
			Error: cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.IncrBy(ctx, key, delta, ttlIfCreate)
		// Convert int64 result to any for AsyncAnyCacheResult
		result <- AsyncAnyCacheResult{
			Value: cacheResult.Int,
			Error: cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (any, error)) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.ReadThrough(ctx, key, ttl, loader)
		result <- AsyncAnyCacheResult{
			Value: cacheResult.Value,
			Found: cacheResult.Found,
			Error: cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) WriteThrough(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.WriteThrough(ctx, key, val, ttl, writer)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) WriteBehind(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.WriteBehind(ctx, key, val, ttl, writer)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (any, error)) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.RefreshAhead(ctx, key, refreshBefore, loader)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) InvalidateByTag(ctx context.Context, tags ...string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.InvalidateByTag(ctx, tags...)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) AddTags(ctx context.Context, key string, tags ...string) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.AddTags(ctx, key, tags...)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
	}()
	return result
}

func (w *anyCacheWrapper) TryLock(ctx context.Context, key string, ttl time.Duration) <-chan AsyncLockResult {
	result := make(chan AsyncLockResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.TryLock(ctx, key, ttl)
		result <- AsyncLockResult{
			Unlock: cacheResult.Unlock,
			OK:     cacheResult.OK,
			Error:  cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) GetStats() map[string]any {
	return w.cache.GetStats()
}

func (w *anyCacheWrapper) Close() error {
	return w.cache.Close()
}
