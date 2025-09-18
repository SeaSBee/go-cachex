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
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl" validate:"min=0s,max=24h"`
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
func NewGormPlugin(cache Cache[any], config *GormConfig) (*Plugin, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}

	if config == nil {
		config = DefaultGormConfig()
	}

	keyBuilder, err := NewBuilder(config.KeyPrefix, "gorm", "default-secret")
	if err != nil {
		// Log the error and use a default key builder
		logx.Error("Failed to create key builder, using default", logx.ErrorField(err))
		keyBuilder = &Builder{
			appName: config.KeyPrefix,
			env:     "gorm",
			secret:  "default-secret",
		}
	}

	return &Plugin{
		cache:      cache,
		keyBuilder: keyBuilder,
		config:     config,
		models:     make(map[string]*ModelConfig),
	}, nil
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "cachex"
}

// Initialize initializes the plugin (placeholder for GORM integration)
func (p *Plugin) Initialize() error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	logx.Info("GORM cachex plugin initialized",
		logx.Bool("invalidation_enabled", p.config.EnableInvalidation),
		logx.Bool("read_through_enabled", p.config.EnableReadThrough),
		logx.String("key_prefix", p.config.KeyPrefix))

	return nil
}

// RegisterModel registers a model for caching
func (p *Plugin) RegisterModel(model any, config *ModelConfig) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	if model == nil {
		return fmt.Errorf("model cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get model name
	modelName := p.getModelName(model)
	if modelName == "" {
		return fmt.Errorf("failed to get model name for %T", model)
	}

	// Validate that the model has an ID field
	if !p.hasIDField(model) {
		return fmt.Errorf("model %T must have an ID field (ID, Id, or id)", model)
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
	if config.TTL < 0 {
		return fmt.Errorf("TTL cannot be negative for model %s", modelName)
	}
	if config.TTL > 24*time.Hour {
		logx.Warn("TTL is very long, consider using a shorter duration",
			logx.String("model", modelName),
			logx.String("ttl", config.TTL.String()))
	}
	if config.KeyTemplate == "" {
		config.KeyTemplate = fmt.Sprintf("%s:{{.ID}}", modelName)
	}

	// Validate key template
	if !p.isValidKeyTemplate(config.KeyTemplate) {
		return fmt.Errorf("invalid key template '%s' for model %s", config.KeyTemplate, modelName)
	}

	p.models[modelName] = config

	logx.Info("Registered model for caching",
		logx.String("model", modelName),
		logx.String("key_template", config.KeyTemplate),
		logx.Bool("enabled", config.Enabled))

	return nil
}

// hasIDField checks if the model has an ID field
func (p *Plugin) hasIDField(model any) bool {
	if model == nil {
		return false
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	// Check for common ID field names
	idFields := []string{"ID", "Id", "id"}
	for _, fieldName := range idFields {
		if field := v.FieldByName(fieldName); field.IsValid() {
			return true
		}
	}

	return false
}

// isValidKeyTemplate validates the key template format
func (p *Plugin) isValidKeyTemplate(template string) bool {
	if template == "" {
		return false
	}

	// Simple check: template should contain {{.ID}} placeholder
	// This is a basic validation that ensures the required placeholder is present
	// We'll use a simple string search approach instead of complex slicing
	for i := 0; i <= len(template)-7; i++ {
		if template[i:i+7] == "{{.ID}}" {
			return true
		}
	}
	return false
}

// InvalidateOnCreate invalidates cache on model creation
func (p *Plugin) InvalidateOnCreate(ctx context.Context, model any) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

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

	// Invalidate by model tag with retry
	if len(config.Tags) > 0 {
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.InvalidateByTag(ctx, config.Tags...)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to invalidate cache on create after retries",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID with retry
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.Del(ctx, cacheKey)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to delete cache key on create after retries",
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
func (p *Plugin) InvalidateOnUpdate(ctx context.Context, model any) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

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

	// Invalidate by model tag with retry
	if len(config.Tags) > 0 {
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.InvalidateByTag(ctx, config.Tags...)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to invalidate cache on update after retries",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID with retry
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.Del(ctx, cacheKey)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to delete cache key on update after retries",
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
func (p *Plugin) InvalidateOnDelete(ctx context.Context, model any) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

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

	// Invalidate by model tag with retry
	if len(config.Tags) > 0 {
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.InvalidateByTag(ctx, config.Tags...)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to invalidate cache on delete after retries",
				logx.String("model", modelName),
				logx.ErrorField(err))
			return err
		}
	}

	// Invalidate specific keys if we have the ID with retry
	if id := p.extractID(model); id != "" {
		cacheKey := p.buildCacheKey(modelName, id)
		err := p.retryCacheOperation(ctx, func() error {
			result := <-p.cache.Del(ctx, cacheKey)
			return result.Error
		}, 2)
		if err != nil {
			logx.Error("Failed to delete cache key on delete after retries",
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
func (p *Plugin) ReadThrough(ctx context.Context, modelName, id string, dest any) (bool, error) {
	if p == nil {
		return false, fmt.Errorf("plugin is nil")
	}

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
	result := <-p.cache.Get(ctx, cacheKey)
	if result.Error == nil && result.Found {
		// Set the cached result
		if err := p.setDestValue(dest, result.Value); err == nil {
			if p.config.EnableDebug {
				logx.Debug("Cache hit on read-through",
					logx.String("key", cacheKey),
					logx.String("model", modelName))
			}
			return true, nil
		} else {
			// Log the error but don't fail the operation
			logx.Warn("Failed to set destination value from cache",
				logx.String("key", cacheKey),
				logx.String("model", modelName),
				logx.ErrorField(err))
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
func (p *Plugin) CacheQueryResult(ctx context.Context, modelName, id string, result any) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	if !p.config.EnableQueryCache {
		return nil
	}

	// Get model config
	config := p.getModelConfig(modelName)
	if config == nil || !config.Enabled {
		return nil
	}

	// Cache the result with retry
	cacheKey := p.buildCacheKey(modelName, id)
	err := p.retryCacheOperation(ctx, func() error {
		setResult := <-p.cache.Set(ctx, cacheKey, result, config.TTL)
		return setResult.Error
	}, 2)
	if err != nil {
		logx.Error("Failed to cache query result after retries",
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
func (p *Plugin) getModelName(model any) string {
	if p == nil || model == nil {
		return ""
	}

	// Get the type name
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	if name == "" {
		// Handle anonymous types or interfaces
		return t.String()
	}

	return name
}

// getModelConfig gets the configuration for a model
func (p *Plugin) getModelConfig(modelName string) *ModelConfig {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.models[modelName]
}

// buildCacheKey builds a cache key for a model and ID
func (p *Plugin) buildCacheKey(modelName, id string) string {
	if p == nil || p.keyBuilder == nil {
		return fmt.Sprintf("%s:%s", modelName, id)
	}

	key := p.keyBuilder.Build(modelName, id)
	return key
}

// extractID extracts the ID field from a model
func (p *Plugin) extractID(model any) string {
	if p == nil || model == nil {
		return ""
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Check if it's a struct
	if v.Kind() != reflect.Struct {
		return ""
	}

	// Try common ID field names
	idFields := []string{"ID", "Id", "id"}
	for _, fieldName := range idFields {
		if field := v.FieldByName(fieldName); field.IsValid() && field.CanInterface() {
			// Add additional safety check for nil pointers
			if field.Kind() == reflect.Ptr && field.IsNil() {
				continue
			}
			return fmt.Sprintf("%v", field.Interface())
		}
	}

	return ""
}

// setDestValue sets the destination value from cached data
func (p *Plugin) setDestValue(dest any, value any) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	if dest == nil || value == nil {
		return fmt.Errorf("destination or value is nil")
	}

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	// Check if destination is nil pointer
	if destVal.IsNil() {
		return fmt.Errorf("destination is a nil pointer")
	}

	destElem := destVal.Elem()

	// Check if destination is settable
	if !destElem.CanSet() {
		return fmt.Errorf("destination is not settable")
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
		if valueVal.IsNil() {
			return fmt.Errorf("value is a nil pointer")
		}
		valueVal = valueVal.Elem()
	}

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
	if p == nil {
		return nil
	}

	// Create a new plugin instance with the context-aware cache
	// Deep copy fields to avoid sharing mutable state
	newPlugin := &Plugin{
		cache:      p.cache.WithContext(ctx),
		keyBuilder: p.deepCopyKeyBuilder(),
		config:     p.deepCopyConfig(),
		models:     make(map[string]*ModelConfig),
		mu:         sync.RWMutex{}, // Add mutex to prevent race conditions
	}

	// Safely copy the models map
	p.mu.RLock()
	for k, v := range p.models {
		newPlugin.models[k] = p.deepCopyModelConfig(v)
	}
	p.mu.RUnlock()

	return newPlugin
}

// deepCopyKeyBuilder creates a deep copy of the key builder
func (p *Plugin) deepCopyKeyBuilder() *Builder {
	if p.keyBuilder == nil {
		return nil
	}

	// Create a new builder with the same configuration
	newBuilder, err := NewBuilder(p.keyBuilder.appName, p.keyBuilder.env, p.keyBuilder.secret)
	if err != nil {
		// Fallback to manual copy if builder creation fails
		return &Builder{
			appName: p.keyBuilder.appName,
			env:     p.keyBuilder.env,
			secret:  p.keyBuilder.secret,
		}
	}
	return newBuilder
}

// deepCopyConfig creates a deep copy of the configuration
func (p *Plugin) deepCopyConfig() *GormConfig {
	if p.config == nil {
		return nil
	}

	return &GormConfig{
		EnableInvalidation: p.config.EnableInvalidation,
		EnableReadThrough:  p.config.EnableReadThrough,
		DefaultTTL:         p.config.DefaultTTL,
		EnableQueryCache:   p.config.EnableQueryCache,
		KeyPrefix:          p.config.KeyPrefix,
		EnableDebug:        p.config.EnableDebug,
		BatchSize:          p.config.BatchSize,
	}
}

// deepCopyModelConfig creates a deep copy of the model configuration
func (p *Plugin) deepCopyModelConfig(config *ModelConfig) *ModelConfig {
	if config == nil {
		return nil
	}

	tags := make([]string, len(config.Tags))
	copy(tags, config.Tags)

	return &ModelConfig{
		Name:         config.Name,
		TTL:          config.TTL,
		Enabled:      config.Enabled,
		KeyTemplate:  config.KeyTemplate,
		Tags:         tags,
		ReadThrough:  config.ReadThrough,
		WriteThrough: config.WriteThrough,
	}
}

// GetCache returns the underlying cache instance
func (p *Plugin) GetCache() Cache[any] {
	if p == nil {
		return nil
	}
	return p.cache
}

// GetConfig returns the plugin configuration
func (p *Plugin) GetConfig() *GormConfig {
	if p == nil {
		return nil
	}
	return p.config
}

// GetRegisteredModels returns the list of registered models
func (p *Plugin) GetRegisteredModels() []string {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	models := make([]string, 0, len(p.models))
	for model := range p.models {
		models = append(models, model)
	}
	return models
}

// ClearCache clears all cached data for registered models
func (p *Plugin) ClearCache(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	// Get a copy of models to avoid holding lock during cache operations
	p.mu.RLock()
	modelsCopy := make(map[string]*ModelConfig, len(p.models))
	for k, v := range p.models {
		modelsCopy[k] = v
	}
	p.mu.RUnlock()

	// Process cache operations without holding the lock
	for modelName, config := range modelsCopy {
		// Clear by tags if available with retry
		if len(config.Tags) > 0 {
			err := p.retryCacheOperation(ctx, func() error {
				result := <-p.cache.InvalidateByTag(ctx, config.Tags...)
				return result.Error
			}, 2)
			if err != nil {
				logx.Error("Failed to clear cache for model by tags after retries",
					logx.String("model", modelName),
					logx.ErrorField(err))
				return err
			}
		}

		// Note: For models without tags, we rely on the application to manage
		// cache invalidation through explicit calls to InvalidateOnCreate/Update/Delete
		// or by using the model's ID-based cache keys directly
	}

	logx.Info("Cleared cache for all registered models")
	return nil
}

// Helper functions for common operations

// RegisterModel is a helper function to register a model with default configuration
func (p *Plugin) RegisterModelWithDefaults(model any, ttl time.Duration, tags ...string) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	config := &ModelConfig{
		TTL:         ttl,
		Enabled:     true,
		ReadThrough: true,
		Tags:        tags,
	}

	return p.RegisterModel(model, config)
}

// retryCacheOperation retries a cache operation with exponential backoff
func (p *Plugin) retryCacheOperation(ctx context.Context, operation func() error, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 3 // Default retry count
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context cancellation before each operation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < maxRetries {
				// Exponential backoff: 100ms, 200ms, 400ms, etc.
				backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
					timer.Stop()
					continue
				}
			}
		}
	}

	return fmt.Errorf("cache operation failed after %d retries: %w", maxRetries, lastErr)
}
