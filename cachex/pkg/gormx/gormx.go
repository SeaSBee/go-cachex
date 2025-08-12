package gormx

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/key"
	"github.com/seasbee/go-logx"
)

// Plugin provides GORM integration for automatic cache invalidation and read-through
type Plugin struct {
	// Cache instance
	cache cache.Cache[any]

	// Key builder for generating cache keys
	keyBuilder *key.Builder

	// Configuration
	config *Config

	// Mutex for thread safety
	mu sync.RWMutex

	// Registered models and their cache configurations
	models map[string]*ModelConfig

	// Context for operations
	ctx context.Context
}

// Config holds GORM plugin configuration
type Config struct {
	// Enable automatic cache invalidation
	EnableInvalidation bool
	// Enable read-through caching
	EnableReadThrough bool
	// Default TTL for cached items
	DefaultTTL time.Duration
	// Enable query result caching
	EnableQueryCache bool
	// Cache key prefix
	KeyPrefix string
	// Enable debug logging
	EnableDebug bool
	// Batch invalidation size
	BatchSize int
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableInvalidation: true,
		EnableReadThrough:  true,
		DefaultTTL:         5 * time.Minute,
		EnableQueryCache:   true,
		KeyPrefix:          "gorm",
		EnableDebug:        false,
		BatchSize:          100,
	}
}

// ModelConfig holds configuration for a specific model
type ModelConfig struct {
	// Model name
	Name string
	// Cache TTL for this model
	TTL time.Duration
	// Enable caching for this model
	Enabled bool
	// Cache key template
	KeyTemplate string
	// Invalidation tags
	Tags []string
	// Enable read-through
	ReadThrough bool
	// Enable write-through
	WriteThrough bool
}

// New creates a new GORM plugin
func New(cache cache.Cache[any], config *Config) *Plugin {
	if config == nil {
		config = DefaultConfig()
	}

	return &Plugin{
		cache:      cache,
		keyBuilder: key.NewBuilder(config.KeyPrefix, "gorm", ""),
		config:     config,
		models:     make(map[string]*ModelConfig),
		ctx:        context.Background(),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "cachex"
}

// Initialize initializes the plugin (placeholder for GORM integration)
func (p *Plugin) Initialize() error {
	logx.Info("GORM cachex plugin initialized",
		logx.Bool("invalidation_enabled", p.config.EnableInvalidation),
		logx.Bool("read_through_enabled", p.config.EnableReadThrough),
		logx.String("key_prefix", p.config.KeyPrefix))

	return nil
}

// RegisterModel registers a model for caching
func (p *Plugin) RegisterModel(model interface{}, config *ModelConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get model name
	modelName := p.getModelName(model)
	if modelName == "" {
		return fmt.Errorf("failed to get model name for %T", model)
	}

	// Set default values
	if config == nil {
		config = &ModelConfig{}
	}
	if config.Name == "" {
		config.Name = modelName
	}
	if config.TTL == 0 {
		config.TTL = p.config.DefaultTTL
	}
	if config.KeyTemplate == "" {
		config.KeyTemplate = fmt.Sprintf("%s:{{.ID}}", modelName)
	}

	p.models[modelName] = config

	logx.Info("Registered model for caching",
		logx.String("model", modelName),
		logx.String("key_template", config.KeyTemplate),
		logx.Bool("enabled", config.Enabled))

	return nil
}

// InvalidateOnCreate invalidates cache on model creation
func (p *Plugin) InvalidateOnCreate(model interface{}) error {
	if !p.config.EnableInvalidation {
		return nil
	}

	// Get model name
	modelName := p.getModelName(model)
	if modelName == "" {
		return fmt.Errorf("failed to get model name")
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled {
		return nil
	}

	// Invalidate by model tag
	if len(config.Tags) > 0 {
		if err := p.cache.InvalidateByTag(config.Tags...); err != nil {
			logx.Error("Failed to invalidate cache on create",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		if err := p.cache.Del(cacheKey); err != nil {
			logx.Error("Failed to delete cache key on create",
				logx.String("key", cacheKey),
				logx.ErrorField(err))
			return err
		}
	}

	if p.config.EnableDebug {
		logx.Debug("Invalidated cache on create",
			logx.String("model", modelName),
			logx.String("tags", fmt.Sprintf("%v", config.Tags)))
	}

	return nil
}

// InvalidateOnUpdate invalidates cache on model update
func (p *Plugin) InvalidateOnUpdate(model interface{}) error {
	if !p.config.EnableInvalidation {
		return nil
	}

	// Get model name
	modelName := p.getModelName(model)
	if modelName == "" {
		return fmt.Errorf("failed to get model name")
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled {
		return nil
	}

	// Invalidate by model tag
	if len(config.Tags) > 0 {
		if err := p.cache.InvalidateByTag(config.Tags...); err != nil {
			logx.Error("Failed to invalidate cache on update",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		if err := p.cache.Del(cacheKey); err != nil {
			logx.Error("Failed to delete cache key on update",
				logx.String("key", cacheKey),
				logx.ErrorField(err))
			return err
		}
	}

	if p.config.EnableDebug {
		logx.Debug("Invalidated cache on update",
			logx.String("model", modelName),
			logx.String("tags", fmt.Sprintf("%v", config.Tags)))
	}

	return nil
}

// InvalidateOnDelete invalidates cache on model deletion
func (p *Plugin) InvalidateOnDelete(model interface{}) error {
	if !p.config.EnableInvalidation {
		return nil
	}

	// Get model name
	modelName := p.getModelName(model)
	if modelName == "" {
		return fmt.Errorf("failed to get model name")
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled {
		return nil
	}

	// Invalidate by model tag
	if len(config.Tags) > 0 {
		if err := p.cache.InvalidateByTag(config.Tags...); err != nil {
			logx.Error("Failed to invalidate cache on delete",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		if err := p.cache.Del(cacheKey); err != nil {
			logx.Error("Failed to delete cache key on delete",
				logx.String("key", cacheKey),
				logx.ErrorField(err))
			return err
		}
	}

	if p.config.EnableDebug {
		logx.Debug("Invalidated cache on delete",
			logx.String("model", modelName),
			logx.String("tags", fmt.Sprintf("%v", config.Tags)))
	}

	return nil
}

// ReadThrough implements read-through caching for a model by ID
func (p *Plugin) ReadThrough(modelName, id string, dest interface{}) (bool, error) {
	if !p.config.EnableReadThrough {
		return false, nil
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled || !config.ReadThrough {
		return false, nil
	}

	// Try to get from cache
	cacheKey := p.buildCacheKey(modelName, id)
	if cached, found, err := p.cache.Get(cacheKey); err == nil && found {
		// Set the cached result
		if err := p.setDestValue(dest, cached); err == nil {
			if p.config.EnableDebug {
				logx.Debug("Cache hit on read-through",
					logx.String("key", cacheKey),
					logx.String("model", modelName))
			}
			return true, nil
		}
	}

	if p.config.EnableDebug {
		logx.Debug("Cache miss on read-through",
			logx.String("key", cacheKey),
			logx.String("model", modelName))
	}

	return false, nil
}

// CacheQueryResult caches query results
func (p *Plugin) CacheQueryResult(modelName, id string, result interface{}) error {
	if !p.config.EnableQueryCache {
		return nil
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled {
		return nil
	}

	// Cache the result
	cacheKey := p.buildCacheKey(modelName, id)
	if err := p.cache.Set(cacheKey, result, config.TTL); err != nil {
		logx.Error("Failed to cache query result",
			logx.String("key", cacheKey),
			logx.ErrorField(err))
		return err
	}

	if p.config.EnableDebug {
		logx.Debug("Cached query result",
			logx.String("key", cacheKey),
			logx.String("model", modelName))
	}

	return nil
}

// getModelName extracts the model name from the model interface
func (p *Plugin) getModelName(model interface{}) string {
	if model == nil {
		return ""
	}

	// Get the type name
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name()
}

// getModelConfig gets the configuration for a model
func (p *Plugin) getModelConfig(modelName string) *ModelConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.models[modelName]
}

// buildCacheKey builds a cache key for a model and ID
func (p *Plugin) buildCacheKey(modelName, id string) string {
	return p.keyBuilder.Build(modelName, id)
}

// extractID extracts the ID field from a model
func (p *Plugin) extractID(model interface{}) string {
	if model == nil {
		return ""
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Try common ID field names
	idFields := []string{"ID", "Id", "id"}
	for _, fieldName := range idFields {
		if field := v.FieldByName(fieldName); field.IsValid() {
			return fmt.Sprintf("%v", field.Interface())
		}
	}

	return ""
}

// setDestValue sets the destination value from cached data
func (p *Plugin) setDestValue(dest interface{}, value interface{}) error {
	if dest == nil || value == nil {
		return fmt.Errorf("destination or value is nil")
	}

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	// Handle map[string]interface{} (JSON decoded data)
	if mapData, ok := value.(map[string]interface{}); ok {
		// Convert map back to JSON and then decode to destination
		jsonData, err := json.Marshal(mapData)
		if err != nil {
			return fmt.Errorf("failed to marshal map data: %w", err)
		}

		if err := json.Unmarshal(jsonData, dest); err != nil {
			return fmt.Errorf("failed to unmarshal to destination: %w", err)
		}
		return nil
	}

	// Try reflection for type conversion
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}

	destElem := destVal.Elem()

	// Check if types are compatible
	if destElem.Type() == valueVal.Type() {
		destElem.Set(valueVal)
		return nil
	}

	// Try to convert if possible
	if valueVal.CanConvert(destElem.Type()) {
		destElem.Set(valueVal.Convert(destElem.Type()))
		return nil
	}

	return fmt.Errorf("cannot set value of type %T to destination of type %T", value, dest)
}

// WithContext sets the context for the plugin
func (p *Plugin) WithContext(ctx context.Context) *Plugin {
	p.ctx = ctx
	return p
}

// GetCache returns the underlying cache instance
func (p *Plugin) GetCache() cache.Cache[any] {
	return p.cache
}

// GetConfig returns the plugin configuration
func (p *Plugin) GetConfig() *Config {
	return p.config
}

// GetRegisteredModels returns the list of registered models
func (p *Plugin) GetRegisteredModels() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	models := make([]string, 0, len(p.models))
	for model := range p.models {
		models = append(models, model)
	}
	return models
}

// ClearCache clears all cached data for registered models
func (p *Plugin) ClearCache() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for modelName, config := range p.models {
		if len(config.Tags) > 0 {
			if err := p.cache.InvalidateByTag(config.Tags...); err != nil {
				logx.Error("Failed to clear cache for model",
					logx.String("model", modelName),
					logx.ErrorField(err))
				return err
			}
		}
	}

	logx.Info("Cleared cache for all registered models")
	return nil
}

// Helper functions for common operations

// RegisterModel is a helper function to register a model with default configuration
func (p *Plugin) RegisterModelWithDefaults(model interface{}, ttl time.Duration, tags ...string) error {
	config := &ModelConfig{
		TTL:         ttl,
		Enabled:     true,
		ReadThrough: true,
		Tags:        tags,
	}

	return p.RegisterModel(model, config)
}
