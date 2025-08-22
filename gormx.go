package cachex

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// Plugin provides GORM integration for automatic cache invalidation and read-through
type Plugin struct {
	// Cache instance
	cache Cache[any]

	// Key builder for generating cache keys
	keyBuilder *Builder

	// Configuration
	config *GormConfig

	// Mutex for thread safety
	mu sync.RWMutex

	// Registered models and their cache configurations
	models map[string]*ModelConfig
}

// Config holds GORM plugin configuration
type GormConfig struct {
	// Enable automatic cache invalidation
	EnableInvalidation bool `yaml:"enable_invalidation" json:"enable_invalidation"`
	// Enable read-through caching
	EnableReadThrough bool `yaml:"enable_read_through" json:"enable_read_through"`
	// Default TTL for cached items
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"`
	// Enable query result caching
	EnableQueryCache bool `yaml:"enable_query_cache" json:"enable_query_cache"`
	// Cache key prefix
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
	// Enable debug logging
	EnableDebug bool `yaml:"enable_debug" json:"enable_debug"`
	// Batch invalidation size
	BatchSize int `yaml:"batch_size" json:"batch_size"`
}

// DefaultGormConfig returns a default configuration
func DefaultGormConfig() *GormConfig {
	return &GormConfig{
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

// NewGormPlugin creates a new GORM plugin
func NewGormPlugin(cache Cache[any], config *GormConfig) *Plugin {
	if config == nil {
		config = DefaultGormConfig()
	}

	return &Plugin{
		cache:      cache,
		keyBuilder: NewBuilder(config.KeyPrefix, "gorm", ""),
		config:     config,
		models:     make(map[string]*ModelConfig),
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
func (p *Plugin) RegisterModel(model any, config *ModelConfig) error {
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
func (p *Plugin) InvalidateOnCreate(model any) error {
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
		result := <-p.cache.InvalidateByTag(context.Background(), config.Tags...)
		if result.Error != nil {
			logx.Error("Failed to invalidate cache on create",
				logx.String("model", modelName),
				logx.ErrorField(result.Error))
			return result.Error
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		result := <-p.cache.Del(context.Background(), cacheKey)
		if result.Error != nil {
			logx.Error("Failed to delete cache key on create",
				logx.String("key", cacheKey),
				logx.ErrorField(result.Error))
			return result.Error
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
func (p *Plugin) InvalidateOnUpdate(model any) error {
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
		result := <-p.cache.InvalidateByTag(context.Background(), config.Tags...)
		if result.Error != nil {
			logx.Error("Failed to invalidate cache on update",
				logx.String("model", modelName),
				logx.ErrorField(result.Error))
			return result.Error
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		result := <-p.cache.Del(context.Background(), cacheKey)
		if result.Error != nil {
			logx.Error("Failed to delete cache key on update",
				logx.String("key", cacheKey),
				logx.ErrorField(result.Error))
			return result.Error
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
func (p *Plugin) InvalidateOnDelete(model any) error {
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
		result := <-p.cache.InvalidateByTag(context.Background(), config.Tags...)
		if result.Error != nil {
			logx.Error("Failed to invalidate cache on delete",
				logx.String("model", modelName),
				logx.ErrorField(result.Error))
			return result.Error
		}
	}

	// Invalidate specific keys if we have the ID
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		result := <-p.cache.Del(context.Background(), cacheKey)
		if result.Error != nil {
			logx.Error("Failed to delete cache key on delete",
				logx.String("key", cacheKey),
				logx.ErrorField(result.Error))
			return result.Error
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
func (p *Plugin) ReadThrough(modelName, id string, dest any) (bool, error) {
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
	result := <-p.cache.Get(context.Background(), cacheKey)
	if result.Error == nil && result.Found {
		// Set the cached result
		if err := p.setDestValue(dest, result.Value); err == nil {
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
func (p *Plugin) CacheQueryResult(modelName, id string, result any) error {
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
	setResult := <-p.cache.Set(context.Background(), cacheKey, result, config.TTL)
	if setResult.Error != nil {
		logx.Error("Failed to cache query result",
			logx.String("key", cacheKey),
			logx.ErrorField(setResult.Error))
		return setResult.Error
	}

	if p.config.EnableDebug {
		logx.Debug("Cached query result",
			logx.String("key", cacheKey),
			logx.String("model", modelName))
	}

	return nil
}

// getModelName extracts the model name from the model interface
func (p *Plugin) getModelName(model any) string {
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
func (p *Plugin) extractID(model any) string {
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
func (p *Plugin) setDestValue(dest any, value any) error {
	if dest == nil || value == nil {
		return fmt.Errorf("destination or value is nil")
	}

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	// Handle map[string]any (JSON decoded data)
	if mapData, ok := value.(map[string]any); ok {
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
	// Create a new plugin instance with the context-aware cache
	// Copy fields manually to avoid copying mutex
	newPlugin := &Plugin{
		cache:      p.cache.WithContext(ctx),
		keyBuilder: p.keyBuilder,
		config:     p.config,
		models:     p.models,
	}
	return newPlugin
}

// GetCache returns the underlying cache instance
func (p *Plugin) GetCache() Cache[any] {
	return p.cache
}

// GetConfig returns the plugin configuration
func (p *Plugin) GetConfig() *GormConfig {
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
			result := <-p.cache.InvalidateByTag(context.Background(), config.Tags...)
			if result.Error != nil {
				logx.Error("Failed to clear cache for model",
					logx.String("model", modelName),
					logx.ErrorField(result.Error))
				return result.Error
			}
		}
	}

	logx.Info("Cleared cache for all registered models")
	return nil
}

// Helper functions for common operations

// RegisterModel is a helper function to register a model with default configuration
func (p *Plugin) RegisterModelWithDefaults(model any, ttl time.Duration, tags ...string) error {
	config := &ModelConfig{
		TTL:         ttl,
		Enabled:     true,
		ReadThrough: true,
		Tags:        tags,
	}

	return p.RegisterModel(model, config)
}
