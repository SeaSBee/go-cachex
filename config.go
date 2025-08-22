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
	Memory    *MemoryConfig    `yaml:"memory" json:"memory"`
	Redis     *RedisConfig     `yaml:"redis" json:"redis"`
	Ristretto *RistrettoConfig `yaml:"ristretto" json:"ristretto"`
	Layered   *LayeredConfig   `yaml:"layered" json:"layered"`

	// General cache settings
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"`
	MaxRetries int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay"`

	// Codec settings
	Codec string `yaml:"codec" json:"codec"` // "json" or "msgpack"

	// Observability settings
	Observability *ObservabilityConfig `yaml:"observability" json:"observability"`

	// Security settings
	Security *SecurityConfig `yaml:"security" json:"security"`

	// Tagging settings
	Tagging *TagConfig `yaml:"tagging" json:"tagging"`

	// Refresh ahead settings
	RefreshAhead *RefreshAheadConfig `yaml:"refresh_ahead" json:"refresh_ahead"`

	// GORM integration settings
	GORM *GormConfig `yaml:"gorm" json:"gorm"`
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
		if val := os.Getenv("CACHEX_MEMORY_MAX_SIZE"); val != "" {
			if maxSize, err := strconv.Atoi(val); err == nil {
				config.Memory.MaxSize = maxSize
			}
		}
		if val := os.Getenv("CACHEX_MEMORY_MAX_MEMORY_MB"); val != "" {
			if maxMemoryMB, err := strconv.Atoi(val); err == nil {
				config.Memory.MaxMemoryMB = maxMemoryMB
			}
		}
		if val := os.Getenv("CACHEX_MEMORY_DEFAULT_TTL"); val != "" {
			if ttl, err := time.ParseDuration(val); err == nil {
				config.Memory.DefaultTTL = ttl
			}
		}
		if val := os.Getenv("CACHEX_MEMORY_CLEANUP_INTERVAL"); val != "" {
			if interval, err := time.ParseDuration(val); err == nil {
				config.Memory.CleanupInterval = interval
			}
		}
		if val := os.Getenv("CACHEX_MEMORY_EVICTION_POLICY"); val != "" {
			config.Memory.EvictionPolicy = EvictionPolicy(val)
		}
		// Set defaults if not provided
		if config.Memory.MaxSize == 0 {
			config.Memory.MaxSize = 10000
		}
		if config.Memory.MaxMemoryMB == 0 {
			config.Memory.MaxMemoryMB = 100
		}
		if config.Memory.DefaultTTL == 0 {
			config.Memory.DefaultTTL = 5 * time.Minute
		}
		if config.Memory.CleanupInterval == 0 {
			config.Memory.CleanupInterval = 1 * time.Minute
		}
		if config.Memory.EvictionPolicy == "" {
			config.Memory.EvictionPolicy = EvictionPolicyLRU
		}
	}

	// Redis store configuration
	if config.Redis != nil {
		if val := os.Getenv("CACHEX_REDIS_ADDR"); val != "" {
			config.Redis.Addr = val
		}
		if val := os.Getenv("CACHEX_REDIS_PASSWORD"); val != "" {
			config.Redis.Password = val
		}
		if val := os.Getenv("CACHEX_REDIS_DB"); val != "" {
			if db, err := strconv.Atoi(val); err == nil {
				config.Redis.DB = db
			}
		}
		if val := os.Getenv("CACHEX_REDIS_POOL_SIZE"); val != "" {
			if poolSize, err := strconv.Atoi(val); err == nil {
				config.Redis.PoolSize = poolSize
			}
		}
		if val := os.Getenv("CACHEX_REDIS_DIAL_TIMEOUT"); val != "" {
			if timeout, err := time.ParseDuration(val); err == nil {
				config.Redis.DialTimeout = timeout
			}
		}
		if val := os.Getenv("CACHEX_REDIS_READ_TIMEOUT"); val != "" {
			if timeout, err := time.ParseDuration(val); err == nil {
				config.Redis.ReadTimeout = timeout
			}
		}
		if val := os.Getenv("CACHEX_REDIS_WRITE_TIMEOUT"); val != "" {
			if timeout, err := time.ParseDuration(val); err == nil {
				config.Redis.WriteTimeout = timeout
			}
		}
		if val := os.Getenv("CACHEX_REDIS_TLS_ENABLED"); val != "" {
			if config.Redis.TLS == nil {
				config.Redis.TLS = &TLSConfig{}
			}
			config.Redis.TLS.Enabled = strings.ToLower(val) == "true"
		}
	}

	// Ristretto store configuration
	if config.Ristretto != nil {
		if val := os.Getenv("CACHEX_RISTRETTO_MAX_ITEMS"); val != "" {
			if maxItems, err := strconv.Atoi(val); err == nil {
				config.Ristretto.MaxItems = int64(maxItems)
			}
		}
		if val := os.Getenv("CACHEX_RISTRETTO_MAX_MEMORY_BYTES"); val != "" {
			if maxMemory, err := strconv.ParseInt(val, 10, 64); err == nil {
				config.Ristretto.MaxMemoryBytes = maxMemory
			}
		}
		if val := os.Getenv("CACHEX_RISTRETTO_DEFAULT_TTL"); val != "" {
			if ttl, err := time.ParseDuration(val); err == nil {
				config.Ristretto.DefaultTTL = ttl
			}
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
	if val := os.Getenv("CACHEX_DEFAULT_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			config.DefaultTTL = ttl
		}
	}
	if val := os.Getenv("CACHEX_MAX_RETRIES"); val != "" {
		if maxRetries, err := strconv.Atoi(val); err == nil {
			config.MaxRetries = maxRetries
		}
	}
	if val := os.Getenv("CACHEX_RETRY_DELAY"); val != "" {
		if retryDelay, err := time.ParseDuration(val); err == nil {
			config.RetryDelay = retryDelay
		}
	}

	// Codec settings
	if val := os.Getenv("CACHEX_CODEC"); val != "" {
		config.Codec = val
	}

	// Observability settings
	if config.Observability == nil {
		config.Observability = &ObservabilityConfig{}
	}
	if val := os.Getenv("CACHEX_OBSERVABILITY_ENABLE_METRICS"); val != "" {
		config.Observability.EnableMetrics = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CACHEX_OBSERVABILITY_ENABLE_TRACING"); val != "" {
		config.Observability.EnableTracing = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CACHEX_OBSERVABILITY_ENABLE_LOGGING"); val != "" {
		config.Observability.EnableLogging = strings.ToLower(val) == "true"
	}

	// Security settings
	if config.Security == nil {
		config.Security = &SecurityConfig{}
	}
	if val := os.Getenv("CACHEX_SECURITY_MAX_KEY_LENGTH"); val != "" {
		if maxKeyLength, err := strconv.Atoi(val); err == nil {
			if config.Security.Validation == nil {
				config.Security.Validation = &Config{}
			}
			config.Security.Validation.MaxKeyLength = maxKeyLength
		}
	}
	if val := os.Getenv("CACHEX_SECURITY_MAX_VALUE_SIZE"); val != "" {
		if maxValueSize, err := strconv.Atoi(val); err == nil {
			if config.Security.Validation == nil {
				config.Security.Validation = &Config{}
			}
			config.Security.Validation.MaxValueSize = maxValueSize
		}
	}

	// Tagging settings
	if config.Tagging == nil {
		config.Tagging = &TagConfig{}
	}
	if val := os.Getenv("CACHEX_TAGGING_ENABLE_PERSISTENCE"); val != "" {
		config.Tagging.EnablePersistence = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CACHEX_TAGGING_TAG_MAPPING_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			config.Tagging.TagMappingTTL = ttl
		}
	}

	// Refresh ahead settings
	if config.RefreshAhead == nil {
		config.RefreshAhead = &RefreshAheadConfig{}
	}
	if val := os.Getenv("CACHEX_REFRESH_AHEAD_ENABLED"); val != "" {
		config.RefreshAhead.Enabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CACHEX_REFRESH_AHEAD_REFRESH_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			config.RefreshAhead.RefreshInterval = interval
		}
	}

	// GORM settings
	if config.GORM == nil {
		config.GORM = &GormConfig{}
	}
	if val := os.Getenv("CACHEX_GORM_ENABLE_READ_THROUGH"); val != "" {
		config.GORM.EnableReadThrough = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CACHEX_GORM_ENABLE_INVALIDATION"); val != "" {
		config.GORM.EnableInvalidation = strings.ToLower(val) == "true"
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *CacheConfig) error {
	// Check that only one cache store is configured
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
		return fmt.Errorf("at least one cache store must be configured (memory, redis, ristretto, or layered)")
	}
	if storeCount > 1 {
		return fmt.Errorf("only one cache store can be configured per instance, found %d stores", storeCount)
	}

	// Validate memory store configuration
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
		if config.Memory.EvictionPolicy != "" {
			validPolicies := []EvictionPolicy{EvictionPolicyLRU, EvictionPolicyLFU, EvictionPolicyTTL}
			valid := false
			for _, policy := range validPolicies {
				if config.Memory.EvictionPolicy == policy {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("memory store eviction_policy must be one of: %v", validPolicies)
			}
		}
	}

	// Validate Redis store configuration
	if config.Redis != nil {
		if config.Redis.Addr == "" {
			return fmt.Errorf("redis store addr is required")
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

	// Validate Ristretto store configuration
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

	// Validate layered store configuration
	if config.Layered != nil {
		if config.Layered.ReadPolicy != "" {
			validPolicies := []ReadPolicy{ReadPolicyThrough, ReadPolicyAside, ReadPolicyAround}
			valid := false
			for _, policy := range validPolicies {
				if config.Layered.ReadPolicy == policy {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("layered store read_policy must be one of: %v", validPolicies)
			}
		}
		if config.Layered.WritePolicy != "" {
			validPolicies := []WritePolicy{WritePolicyThrough, WritePolicyBehind, WritePolicyAround}
			valid := false
			for _, policy := range validPolicies {
				if config.Layered.WritePolicy == policy {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("layered store write_policy must be one of: %v", validPolicies)
			}
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
	if config.Codec != "" {
		validCodecs := []string{"json", "msgpack"}
		valid := false
		for _, codec := range validCodecs {
			if config.Codec == codec {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("codec must be one of: %v", validCodecs)
		}
	}

	// Validate security settings
	if config.Security != nil && config.Security.Validation != nil {
		if config.Security.Validation.MaxKeyLength <= 0 {
			return fmt.Errorf("security validation max_key_length must be greater than 0")
		}
		if config.Security.Validation.MaxValueSize <= 0 {
			return fmt.Errorf("security validation max_value_size must be greater than 0")
		}
	}

	// Validate refresh ahead settings
	if config.RefreshAhead != nil {
		if config.RefreshAhead.Enabled && config.RefreshAhead.RefreshInterval <= 0 {
			return fmt.Errorf("refresh_ahead refresh_interval must be greater than 0 when enabled")
		}
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
		// This is a simplified implementation - in practice, you'd want to configure L2 store separately
		l2Store, err := NewRedisStore(DefaultRedisConfig())
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
		cacheResult := <-w.cache.Get(ctx, key)
		result <- AsyncAnyCacheResult{
			Value: cacheResult.Value,
			Found: cacheResult.Found,
			Error: cacheResult.Error,
		}
	}()
	return result
}

func (w *anyCacheWrapper) Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan AsyncAnyCacheResult {
	result := make(chan AsyncAnyCacheResult, 1)
	go func() {
		defer close(result)
		cacheResult := <-w.cache.Set(ctx, key, val, ttl)
		result <- AsyncAnyCacheResult{Error: cacheResult.Error}
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
		// Note: AsyncAnyCacheResult doesn't have TTL field, using Found to indicate existence
		result <- AsyncAnyCacheResult{
			Found: cacheResult.Found,
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
		// Note: AsyncAnyCacheResult doesn't have Result field, using Value
		result <- AsyncAnyCacheResult{
			Value: cacheResult.Value,
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
