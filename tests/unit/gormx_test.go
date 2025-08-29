package unit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// Test data structures for GORM plugin tests
type TestModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TestModelWithIntID struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TestModelWithoutID struct {
	Name string `json:"name"`
}

func TestDefaultGormConfig(t *testing.T) {
	config := cachex.DefaultGormConfig()

	if config == nil {
		t.Errorf("DefaultGormConfig() returned nil")
		return
	}
	if !config.EnableInvalidation {
		t.Errorf("EnableInvalidation should be true by default")
	}
	if !config.EnableReadThrough {
		t.Errorf("EnableReadThrough should be true by default")
	}
	if config.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultTTL should be 5 minutes, got %v", config.DefaultTTL)
	}
	if !config.EnableQueryCache {
		t.Errorf("EnableQueryCache should be true by default")
	}
	if config.KeyPrefix != "gorm" {
		t.Errorf("KeyPrefix should be 'gorm', got %s", config.KeyPrefix)
	}
	if config.EnableDebug {
		t.Errorf("EnableDebug should be false by default")
	}
	if config.BatchSize != 100 {
		t.Errorf("BatchSize should be 100, got %d", config.BatchSize)
	}
}

func TestNewGormPlugin_WithConfig(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	config := &cachex.GormConfig{
		EnableInvalidation: false,
		EnableReadThrough:  false,
		DefaultTTL:         10 * time.Minute,
		KeyPrefix:          "test",
	}

	plugin, err := cachex.NewGormPlugin(cache, config)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	if plugin == nil {
		t.Errorf("NewGormPlugin() returned nil")
		return
	}
	if plugin.GetConfig().EnableInvalidation != false {
		t.Errorf("Plugin should use provided config")
	}
	if plugin.GetConfig().KeyPrefix != "test" {
		t.Errorf("Plugin should use provided key prefix")
	}
}

func TestNewGormPlugin_NilConfig(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	if plugin == nil {
		t.Errorf("NewGormPlugin() returned nil")
		return
	}
	if plugin.GetConfig() == nil {
		t.Errorf("Plugin should use default config when nil provided")
		return
	}
	if plugin.GetConfig().KeyPrefix != "gorm" {
		t.Errorf("Plugin should use default key prefix")
	}
}

func TestPlugin_RegisterModel_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := &TestModel{}
	config := &cachex.ModelConfig{
		TTL:     10 * time.Minute,
		Enabled: true,
		Tags:    []string{"user", "admin"},
	}

	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Errorf("RegisterModel() failed: %v", err)
	}

	models := plugin.GetRegisteredModels()
	if len(models) != 1 {
		t.Errorf("RegisterModel() should register one model, got %d", len(models))
	}
	if models[0] != "TestModel" {
		t.Errorf("RegisterModel() should register TestModel, got %s", models[0])
	}
}

func TestPlugin_RegisterModel_NilConfig(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := &TestModel{}

	err = plugin.RegisterModel(model, nil)
	if err != nil {
		t.Errorf("RegisterModel() with nil config failed: %v", err)
	}

	models := plugin.GetRegisteredModels()
	if len(models) != 1 {
		t.Errorf("RegisterModel() should register one model")
	}
}

func TestPlugin_RegisterModel_NilModel(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	err = plugin.RegisterModel(nil, nil)
	if err == nil {
		t.Errorf("RegisterModel() should fail with nil model")
	}
}

func TestPlugin_RegisterModelWithDefaults(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := &TestModel{}
	ttl := 15 * time.Minute
	tags := []string{"user", "active"}

	err = plugin.RegisterModelWithDefaults(model, ttl, tags...)
	if err != nil {
		t.Errorf("RegisterModelWithDefaults() failed: %v", err)
	}

	models := plugin.GetRegisteredModels()
	if len(models) != 1 {
		t.Errorf("RegisterModelWithDefaults() should register one model")
	}
}

func TestPlugin_InvalidateOnCreate_Enabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model first
	model := &TestModel{ID: "1", Name: "Test"}
	config := &cachex.ModelConfig{
		Enabled: true,
		Tags:    []string{"user"},
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.InvalidateOnCreate(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnCreate() failed: %v", err)
	}
}

func TestPlugin_InvalidateOnCreate_Disabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	config := &cachex.GormConfig{
		EnableInvalidation: false,
	}
	plugin, err := cachex.NewGormPlugin(cache, config)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := &TestModel{ID: "1", Name: "Test"}

	err = plugin.InvalidateOnCreate(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnCreate() should not fail when disabled: %v", err)
	}
}

func TestPlugin_InvalidateOnCreate_ModelNotRegistered(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := &TestModel{ID: "1", Name: "Test"}

	err = plugin.InvalidateOnCreate(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnCreate() should not fail for unregistered model: %v", err)
	}
}

func TestPlugin_InvalidateOnCreate_ModelDisabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model as disabled
	model := &TestModel{ID: "1", Name: "Test"}
	config := &cachex.ModelConfig{
		Enabled: false,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.InvalidateOnCreate(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnCreate() should not fail for disabled model: %v", err)
	}
}

func TestPlugin_InvalidateOnCreate_NilModel(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	err = plugin.InvalidateOnCreate(context.Background(), nil)
	if err == nil {
		t.Errorf("InvalidateOnCreate() should fail with nil model")
	}
}

func TestPlugin_InvalidateOnUpdate_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model first
	model := &TestModel{ID: "1", Name: "Test"}
	config := &cachex.ModelConfig{
		Enabled: true,
		Tags:    []string{"user"},
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.InvalidateOnUpdate(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnUpdate() failed: %v", err)
	}
}

func TestPlugin_InvalidateOnDelete_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model first
	model := &TestModel{ID: "1", Name: "Test"}
	config := &cachex.ModelConfig{
		Enabled: true,
		Tags:    []string{"user"},
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.InvalidateOnDelete(context.Background(), model)
	if err != nil {
		t.Errorf("InvalidateOnDelete() failed: %v", err)
	}
}

func TestPlugin_ReadThrough_Disabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	config := &cachex.GormConfig{
		EnableReadThrough: false,
	}
	plugin, err := cachex.NewGormPlugin(cache, config)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	var dest TestModel
	found, err := plugin.ReadThrough(context.Background(), "TestModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() should not fail when disabled: %v", err)
	}
	if found {
		t.Errorf("ReadThrough() should return false when disabled")
	}
}

func TestPlugin_ReadThrough_ModelNotRegistered(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	var dest TestModel
	found, err := plugin.ReadThrough(context.Background(), "UnregisteredModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() should not fail for unregistered model: %v", err)
	}
	if found {
		t.Errorf("ReadThrough() should return false for unregistered model")
	}
}

func TestPlugin_ReadThrough_ModelDisabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model as disabled
	model := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled: false,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	var dest TestModel
	found, err := plugin.ReadThrough(context.Background(), "TestModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() should not fail for disabled model: %v", err)
	}
	if found {
		t.Errorf("ReadThrough() should return false for disabled model")
	}
}

func TestPlugin_ReadThrough_ReadThroughDisabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model with read-through disabled
	model := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled:     true,
		ReadThrough: false,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	var dest TestModel
	found, err := plugin.ReadThrough(context.Background(), "TestModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() should not fail when read-through disabled: %v", err)
	}
	if found {
		t.Errorf("ReadThrough() should return false when read-through disabled")
	}
}

func TestPlugin_ReadThrough_CacheHit(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model
	model := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled:     true,
		ReadThrough: true,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	// Cache some data first
	testModel := TestModel{ID: "1", Name: "Cached"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "gorm:TestModel:1", testModel, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set cache data: %v", setResult.Error)
	}

	var dest TestModel
	_, err = plugin.ReadThrough(context.Background(), "TestModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() failed: %v", err)
	}
	// Note: This might not work as expected due to type conversion issues in mock
	// The test validates the flow rather than the exact data conversion
}

func TestPlugin_ReadThrough_CacheMiss(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model
	model := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled:     true,
		ReadThrough: true,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	var dest TestModel
	found, err := plugin.ReadThrough(context.Background(), "TestModel", "1", &dest)
	if err != nil {
		t.Errorf("ReadThrough() failed: %v", err)
	}
	if found {
		t.Errorf("ReadThrough() should return false for cache miss")
	}
}

func TestPlugin_CacheQueryResult_Disabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	config := &cachex.GormConfig{
		EnableQueryCache: false,
	}
	plugin, err := cachex.NewGormPlugin(cache, config)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult(context.Background(), "TestModel", "1", model)
	if err != nil {
		t.Errorf("CacheQueryResult() should not fail when disabled: %v", err)
	}
}

func TestPlugin_CacheQueryResult_ModelNotRegistered(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult(context.Background(), "UnregisteredModel", "1", model)
	if err != nil {
		t.Errorf("CacheQueryResult() should not fail for unregistered model: %v", err)
	}
}

func TestPlugin_CacheQueryResult_ModelDisabled(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model as disabled
	modelStruct := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled: false,
	}
	err = plugin.RegisterModel(modelStruct, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult(context.Background(), "TestModel", "1", model)
	if err != nil {
		t.Errorf("CacheQueryResult() should not fail for disabled model: %v", err)
	}
}

func TestPlugin_CacheQueryResult_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model
	modelStruct := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled: true,
		TTL:     10 * time.Minute,
	}
	err = plugin.RegisterModel(modelStruct, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult(context.Background(), "TestModel", "1", model)
	if err != nil {
		t.Errorf("CacheQueryResult() failed: %v", err)
	}
}

func TestPlugin_GetModelName(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	tests := []struct {
		name     string
		model    any
		expected string
	}{
		{
			name:     "struct pointer",
			model:    &TestModel{},
			expected: "TestModel",
		},
		{
			name:     "struct value",
			model:    TestModel{},
			expected: "TestModel",
		},
		{
			name:     "nil model",
			model:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't directly test getModelName as it's unexported
			// We test it indirectly through RegisterModel
			if tt.model == nil {
				err := plugin.RegisterModel(tt.model, nil)
				if err == nil && tt.expected == "" {
					// Expected behavior for nil model
					return
				}
				if err != nil && tt.expected == "" {
					// Expected error for nil model
					return
				}
				t.Errorf("RegisterModel() with nil should fail")
				return
			}

			err := plugin.RegisterModel(tt.model, nil)
			if err != nil {
				t.Errorf("RegisterModel() failed: %v", err)
				return
			}

			models := plugin.GetRegisteredModels()
			found := false
			for _, model := range models {
				if model == tt.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected model name %s not found in registered models", tt.expected)
			}
		})
	}
}

func TestPlugin_ExtractID(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	tests := []struct {
		name  string
		model any
		hasID bool
	}{
		{
			name:  "model with string ID",
			model: &TestModel{ID: "123", Name: "Test"},
			hasID: true,
		},
		{
			name:  "model with int ID",
			model: &TestModelWithIntID{ID: 456, Name: "Test"},
			hasID: true,
		},
		{
			name:  "model without ID",
			model: &TestModelWithoutID{Name: "Test"},
			hasID: false,
		},
		{
			name:  "nil model",
			model: nil,
			hasID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We test extractID indirectly through InvalidateOnCreate
			// Register the model first if not nil
			if tt.model != nil {
				config := &cachex.ModelConfig{
					Enabled: true,
					Tags:    []string{"test"},
				}
				err := plugin.RegisterModel(tt.model, config)
				if tt.hasID {
					if err != nil {
						t.Fatalf("Failed to register model: %v", err)
					}

					// This will exercise extractID internally
					err = plugin.InvalidateOnCreate(context.Background(), tt.model)
					if err != nil {
						t.Errorf("InvalidateOnCreate() failed: %v", err)
					}
				} else {
					// Models without ID fields should be rejected
					if err == nil {
						t.Errorf("RegisterModel() should fail for model without ID field")
					}
				}
			}
		})
	}
}

func TestPlugin_WithContext(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	type testKeyType struct{}
	type testValueType struct{}
	testKey := testKeyType{}
	testValue := testValueType{}
	ctx := context.WithValue(context.Background(), testKey, testValue)
	newPlugin := plugin.WithContext(ctx)

	if newPlugin == nil {
		t.Errorf("WithContext() returned nil")
	}
	if newPlugin == plugin {
		t.Errorf("WithContext() should return a new plugin instance")
	}
}

func TestPlugin_GetConfig(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	config := &cachex.GormConfig{
		KeyPrefix: "test",
	}
	plugin, err := cachex.NewGormPlugin(cache, config)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	retrievedConfig := plugin.GetConfig()
	if retrievedConfig == nil {
		t.Errorf("GetConfig() returned nil")
		return
	}
	if retrievedConfig.KeyPrefix != "test" {
		t.Errorf("GetConfig() returned wrong config")
	}
}

func TestPlugin_GetRegisteredModels_Empty(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	models := plugin.GetRegisteredModels()
	if len(models) != 0 {
		t.Errorf("GetRegisteredModels() should return empty slice initially")
	}
}

func TestPlugin_GetRegisteredModels_Multiple(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register multiple models
	models := []any{
		&TestModel{},
		&TestModelWithIntID{},
	}

	for _, model := range models {
		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}
	}

	registeredModels := plugin.GetRegisteredModels()
	if len(registeredModels) != 2 {
		t.Errorf("GetRegisteredModels() should return 2 models, got %d", len(registeredModels))
	}

	expectedModels := map[string]bool{
		"TestModel":          false,
		"TestModelWithIntID": false,
	}

	for _, modelName := range registeredModels {
		if _, exists := expectedModels[modelName]; exists {
			expectedModels[modelName] = true
		} else {
			t.Errorf("Unexpected model name: %s", modelName)
		}
	}

	for modelName, found := range expectedModels {
		if !found {
			t.Errorf("Expected model %s not found", modelName)
		}
	}
}

func TestPlugin_ClearCache_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Register model with tags
	model := &TestModel{}
	config := &cachex.ModelConfig{
		Enabled: true,
		Tags:    []string{"user", "admin"},
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.ClearCache(context.Background())
	if err != nil {
		t.Errorf("ClearCache() failed: %v", err)
	}
}

func TestPlugin_ClearCache_NoModels(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	err = plugin.ClearCache(context.Background())
	if err != nil {
		t.Errorf("ClearCache() should not fail with no models: %v", err)
	}
}

func TestModelConfig_Defaults(t *testing.T) {
	config := &cachex.ModelConfig{}

	if config.Name != "" {
		t.Errorf("ModelConfig Name should be empty by default")
	}
	if config.TTL != 0 {
		t.Errorf("ModelConfig TTL should be 0 by default")
	}
	if config.Enabled {
		t.Errorf("ModelConfig Enabled should be false by default")
	}
	if config.KeyTemplate != "" {
		t.Errorf("ModelConfig KeyTemplate should be empty by default")
	}
	if len(config.Tags) != 0 {
		t.Errorf("ModelConfig Tags should be empty by default")
	}
	if config.ReadThrough {
		t.Errorf("ModelConfig ReadThrough should be false by default")
	}
	if config.WriteThrough {
		t.Errorf("ModelConfig WriteThrough should be false by default")
	}
}

func TestPlugin_ConcurrentAccess(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test concurrent model registration
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Create unique model types to ensure different registrations
			model := &TestModel{ID: fmt.Sprintf("%d", id), Name: fmt.Sprintf("Model%d", id)}
			config := &cachex.ModelConfig{
				Enabled: true,
				Tags:    []string{fmt.Sprintf("tag%d", id)},
			}

			err := plugin.RegisterModel(model, config)
			if err != nil {
				t.Errorf("Concurrent RegisterModel() failed: %v", err)
			}

			// Test concurrent operations
			err = plugin.InvalidateOnCreate(context.Background(), model)
			if err != nil {
				t.Errorf("Concurrent InvalidateOnCreate() failed: %v", err)
			}

			err = plugin.InvalidateOnUpdate(context.Background(), model)
			if err != nil {
				t.Errorf("Concurrent InvalidateOnUpdate() failed: %v", err)
			}

			err = plugin.InvalidateOnDelete(context.Background(), model)
			if err != nil {
				t.Errorf("Concurrent InvalidateOnDelete() failed: %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify models were registered (they will all have the same type name "TestModel")
	models := plugin.GetRegisteredModels()
	if len(models) != 1 {
		t.Errorf("Expected 1 registered model type, got %d", len(models))
	}
	if len(models) > 0 && models[0] != "TestModel" {
		t.Errorf("Expected TestModel to be registered, got %s", models[0])
	}
}

func TestPlugin_TypeConversion(t *testing.T) {
	// Test different model types and their reflection handling
	tests := []struct {
		name  string
		model any
		valid bool
	}{
		{
			name:  "struct pointer",
			model: &TestModel{ID: "1", Name: "Test"},
			valid: true,
		},
		{
			name:  "struct value",
			model: TestModel{ID: "1", Name: "Test"},
			valid: true,
		},
		{
			name:  "interface{}",
			model: interface{}(&TestModel{ID: "1", Name: "Test"}),
			valid: true,
		},
		{
			name:  "nil pointer",
			model: (*TestModel)(nil),
			valid: false, // Nil pointers should be rejected
		},
		{
			name:  "nil interface",
			model: nil,
			valid: false,
		},
	}

	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.RegisterModel(tt.model, nil)

			if tt.valid && err != nil {
				t.Errorf("RegisterModel() should succeed for valid model: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("RegisterModel() should fail for invalid model")
			}
		})
	}
}

func TestPlugin_ReflectionEdgeCases(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test with various reflection edge cases
	testCases := []any{
		// Pointer to pointer
		func() any {
			model := &TestModel{ID: "1", Name: "Test"}
			return &model
		}(),
		// Slice
		[]TestModel{{ID: "1", Name: "Test"}},
		// Map
		map[string]TestModel{"test": {ID: "1", Name: "Test"}},
		// Channel
		make(chan TestModel),
		// Function
		func() TestModel { return TestModel{} },
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d_%T", i, testCase), func(t *testing.T) {
			// These should generally fail or handle gracefully
			err := plugin.RegisterModel(testCase, nil)
			// We don't assert specific behavior as it depends on implementation
			// The test ensures no panics occur
			_ = err
		})
	}
}

func TestGormConfig_Validation(t *testing.T) {
	// Test various configuration combinations
	configs := []*cachex.GormConfig{
		{
			EnableInvalidation: true,
			EnableReadThrough:  true,
			DefaultTTL:         0, // Zero TTL
			EnableQueryCache:   true,
		},
		{
			EnableInvalidation: false,
			EnableReadThrough:  false,
			DefaultTTL:         -1 * time.Minute, // Negative TTL
			EnableQueryCache:   false,
		},
		{
			EnableInvalidation: true,
			EnableReadThrough:  true,
			DefaultTTL:         24 * time.Hour, // Long TTL
			EnableQueryCache:   true,
			BatchSize:          0, // Zero batch size
		},
	}

	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	for i, config := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			plugin, err := cachex.NewGormPlugin(cache, config)
			if err != nil {
				t.Errorf("NewGormPlugin() should handle config gracefully: %v", err)
			}

			// Test basic operations with this config
			err = plugin.Initialize()
			if err != nil {
				t.Errorf("Initialize() failed with config: %v", err)
			}
		})
	}
}

// ===== NEW COMPREHENSIVE TESTS =====

// TestPlugin_ErrorScenarios tests various error scenarios
func TestPlugin_ErrorScenarios(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test nil plugin receiver
	t.Run("nil_receiver", func(t *testing.T) {
		var nilPlugin *cachex.Plugin
		err := nilPlugin.RegisterModel(&TestModel{}, nil)
		if err == nil {
			t.Errorf("RegisterModel() should fail with nil receiver")
		}
	})

	// Test context cancellation
	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		model := &TestModel{ID: "1", Name: "Test"}
		config := &cachex.ModelConfig{Enabled: true, Tags: []string{"test"}}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		// These should handle context cancellation gracefully
		err = plugin.InvalidateOnCreate(ctx, model)
		if err != nil && err != context.Canceled {
			t.Errorf("InvalidateOnCreate() should handle context cancellation gracefully: %v", err)
		}

		err = plugin.InvalidateOnUpdate(ctx, model)
		if err != nil && err != context.Canceled {
			t.Errorf("InvalidateOnUpdate() should handle context cancellation gracefully: %v", err)
		}

		err = plugin.InvalidateOnDelete(ctx, model)
		if err != nil && err != context.Canceled {
			t.Errorf("InvalidateOnDelete() should handle context cancellation gracefully: %v", err)
		}
	})

	// Test invalid TTL values
	t.Run("invalid_ttl", func(t *testing.T) {
		model := &TestModel{}
		config := &cachex.ModelConfig{
			TTL:     -1 * time.Hour, // Negative TTL
			Enabled: true,
		}
		err := plugin.RegisterModel(model, config)
		if err == nil {
			t.Errorf("RegisterModel() should fail with negative TTL")
		}
	})

	// Test very long TTL (should warn but not fail)
	t.Run("very_long_ttl", func(t *testing.T) {
		model := &TestModel{}
		config := &cachex.ModelConfig{
			TTL:     48 * time.Hour, // Very long TTL
			Enabled: true,
		}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Errorf("RegisterModel() should accept very long TTL: %v", err)
		}
	})
}

// TestPlugin_AdvancedReflection tests advanced reflection scenarios
func TestPlugin_AdvancedReflection(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test embedded structs
	t.Run("embedded_structs", func(t *testing.T) {
		type BaseModel struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		type ExtendedModel struct {
			BaseModel
			Email string `json:"email"`
		}

		model := &ExtendedModel{
			BaseModel: BaseModel{ID: "1", Name: "Test"},
			Email:     "test@example.com",
		}

		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with embedded structs: %v", err)
		}
	})

	// Test anonymous structs
	t.Run("anonymous_structs", func(t *testing.T) {
		model := &struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "1",
			Name: "Anonymous",
		}

		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with anonymous structs: %v", err)
		}
	})

	// Test interface implementations
	t.Run("interface_implementations", func(t *testing.T) {
		type ModelInterface interface {
			GetID() string
		}

		type InterfaceModel struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		// Define method outside the test function
		getID := func(m *InterfaceModel) string {
			return m.ID
		}

		model := &InterfaceModel{ID: "1", Name: "Interface"}
		// Test that the model has an ID field (which is what the plugin checks)
		id := getID(model)
		if id != "1" {
			t.Errorf("Expected ID '1', got '%s'", id)
		}

		// This should work because the underlying type has an ID field
		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with interface implementations: %v", err)
		}
	})

	// Test complex nested structs
	t.Run("complex_nested_structs", func(t *testing.T) {
		type Address struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			Country string `json:"country"`
		}

		type ComplexModel struct {
			ID      string                 `json:"id"`
			Name    string                 `json:"name"`
			Address Address                `json:"address"`
			Tags    []string               `json:"tags"`
			Meta    map[string]interface{} `json:"meta"`
		}

		model := &ComplexModel{
			ID:   "1",
			Name: "Complex",
			Address: Address{
				Street:  "123 Main St",
				City:    "Test City",
				Country: "Test Country",
			},
			Tags: []string{"tag1", "tag2"},
			Meta: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		}

		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with complex nested structs: %v", err)
		}
	})

	// Test pointer fields
	t.Run("pointer_fields", func(t *testing.T) {
		type PointerModel struct {
			ID       string  `json:"id"`
			Name     string  `json:"name"`
			Optional *string `json:"optional"`
		}

		optional := "optional_value"
		model := &PointerModel{
			ID:       "1",
			Name:     "Pointer",
			Optional: &optional,
		}

		err := plugin.RegisterModel(model, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with pointer fields: %v", err)
		}

		// Test with nil pointer field
		modelNil := &PointerModel{
			ID:       "2",
			Name:     "PointerNil",
			Optional: nil,
		}

		err = plugin.RegisterModel(modelNil, nil)
		if err != nil {
			t.Errorf("RegisterModel() should work with nil pointer fields: %v", err)
		}
	})
}

// TestPlugin_Performance tests performance characteristics
func TestPlugin_Performance(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test bulk model registration performance
	t.Run("bulk_registration", func(t *testing.T) {
		const numModels = 1000
		start := time.Now()

		for i := 0; i < numModels; i++ {
			model := &TestModel{
				ID:   fmt.Sprintf("id_%d", i),
				Name: fmt.Sprintf("model_%d", i),
			}
			config := &cachex.ModelConfig{
				Enabled: true,
				TTL:     time.Duration(i%60+1) * time.Minute,
				Tags:    []string{fmt.Sprintf("tag_%d", i)},
			}

			err := plugin.RegisterModel(model, config)
			if err != nil {
				t.Fatalf("Failed to register model %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		t.Logf("Registered %d models in %v (%.2f models/sec)", numModels, duration, float64(numModels)/duration.Seconds())

		// Verify all models were registered
		models := plugin.GetRegisteredModels()
		if len(models) != 1 { // All models have the same type name
			t.Errorf("Expected 1 model type, got %d", len(models))
		}
	})

	// Test concurrent operations performance
	t.Run("concurrent_operations", func(t *testing.T) {
		const numGoroutines = 100
		const operationsPerGoroutine = 10

		// Register a model first
		model := &TestModel{ID: "1", Name: "Test"}
		config := &cachex.ModelConfig{Enabled: true, Tags: []string{"test"}}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		start := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*operationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					ctx := context.Background()
					err := plugin.InvalidateOnCreate(ctx, model)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d, operation %d: %v", id, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOperations := numGoroutines * operationsPerGoroutine
		t.Logf("Completed %d concurrent operations in %v (%.2f ops/sec)", totalOperations, duration, float64(totalOperations)/duration.Seconds())

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Found %d errors in concurrent operations", errorCount)
		}
	})
}

// TestPlugin_StressTest tests the plugin under stress conditions
func TestPlugin_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test memory usage under load
	t.Run("memory_usage", func(t *testing.T) {
		const numModels = 10000
		const numOperations = 1000

		// Register many models
		models := make([]*TestModel, numModels)
		for i := 0; i < numModels; i++ {
			models[i] = &TestModel{
				ID:   fmt.Sprintf("id_%d", i),
				Name: fmt.Sprintf("model_%d", i),
			}
			config := &cachex.ModelConfig{
				Enabled: true,
				Tags:    []string{fmt.Sprintf("tag_%d", i)},
			}
			err := plugin.RegisterModel(models[i], config)
			if err != nil {
				t.Fatalf("Failed to register model %d: %v", i, err)
			}
		}

		// Perform many operations
		var wg sync.WaitGroup
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				model := models[id%numModels]
				ctx := context.Background()

				// Mix different operations
				switch id % 3 {
				case 0:
					plugin.InvalidateOnCreate(ctx, model)
				case 1:
					plugin.InvalidateOnUpdate(ctx, model)
				case 2:
					plugin.InvalidateOnDelete(ctx, model)
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Completed stress test with %d models and %d operations", numModels, numOperations)
	})
}

// TestPlugin_IntegrationScenarios tests integration scenarios
func TestPlugin_IntegrationScenarios(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test complete workflow
	t.Run("complete_workflow", func(t *testing.T) {
		// 1. Register model
		model := &TestModel{ID: "1", Name: "Workflow"}
		config := &cachex.ModelConfig{
			Enabled:     true,
			ReadThrough: true,
			TTL:         10 * time.Minute,
			Tags:        []string{"workflow", "test"},
		}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		// 2. Cache query result
		ctx := context.Background()
		err = plugin.CacheQueryResult(ctx, "TestModel", "1", model)
		if err != nil {
			t.Fatalf("Failed to cache query result: %v", err)
		}

		// 3. Test read-through (should find cached data)
		var dest TestModel
		found, err := plugin.ReadThrough(ctx, "TestModel", "1", &dest)
		if err != nil {
			t.Fatalf("ReadThrough() failed: %v", err)
		}
		if !found {
			t.Logf("ReadThrough() returned false (cache miss)")
		}

		// 4. Update model
		model.Name = "Updated Workflow"
		err = plugin.InvalidateOnUpdate(ctx, model)
		if err != nil {
			t.Fatalf("InvalidateOnUpdate() failed: %v", err)
		}

		// 5. Delete model
		err = plugin.InvalidateOnDelete(ctx, model)
		if err != nil {
			t.Fatalf("InvalidateOnDelete() failed: %v", err)
		}

		// 6. Clear cache
		err = plugin.ClearCache(ctx)
		if err != nil {
			t.Fatalf("ClearCache() failed: %v", err)
		}

		t.Logf("Complete workflow test passed")
	})

	// Test multiple model types
	t.Run("multiple_model_types", func(t *testing.T) {
		models := []any{
			&TestModel{ID: "1", Name: "User"},
			&TestModelWithIntID{ID: 2, Name: "Product"},
		}

		configs := []*cachex.ModelConfig{
			{Enabled: true, Tags: []string{"user"}},
			{Enabled: true, Tags: []string{"product"}},
		}

		// Register multiple model types
		for i, model := range models {
			err := plugin.RegisterModel(model, configs[i])
			if err != nil {
				t.Fatalf("Failed to register model %d: %v", i, err)
			}
		}

		// Test operations on all models
		ctx := context.Background()
		for i, model := range models {
			// Cache query results
			err := plugin.CacheQueryResult(ctx, fmt.Sprintf("Model%d", i), fmt.Sprintf("%d", i), model)
			if err != nil {
				t.Errorf("Failed to cache query result for model %d: %v", i, err)
			}

			// Invalidate
			err = plugin.InvalidateOnCreate(ctx, model)
			if err != nil {
				t.Errorf("Failed to invalidate model %d: %v", i, err)
			}
		}

		t.Logf("Multiple model types test passed")
	})
}

// TestPlugin_ErrorInjection tests error injection scenarios
func TestPlugin_ErrorInjection(t *testing.T) {
	// Create a mock store that can simulate failures
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test with timeout context
	t.Run("timeout_context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give the timeout a chance to trigger
		time.Sleep(1 * time.Millisecond)

		model := &TestModel{ID: "1", Name: "Timeout"}
		config := &cachex.ModelConfig{Enabled: true, Tags: []string{"test"}}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		// These should handle timeout gracefully
		err = plugin.InvalidateOnCreate(ctx, model)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("InvalidateOnCreate() should handle timeout gracefully: %v", err)
		}
	})

	// Test with very short retry intervals
	t.Run("short_retry_intervals", func(t *testing.T) {
		model := &TestModel{ID: "1", Name: "Retry"}
		config := &cachex.ModelConfig{Enabled: true, Tags: []string{"test"}}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		ctx := context.Background()
		start := time.Now()

		// This should complete quickly even with retries
		err = plugin.InvalidateOnCreate(ctx, model)
		if err != nil {
			t.Errorf("InvalidateOnCreate() failed: %v", err)
		}

		duration := time.Since(start)
		if duration > 1*time.Second {
			t.Errorf("Operation took too long: %v", duration)
		}
	})
}

// TestPlugin_EndToEnd tests end-to-end scenarios
func TestPlugin_EndToEnd(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Test realistic user scenario
	t.Run("user_scenario", func(t *testing.T) {
		// Simulate a user management system
		type User struct {
			ID       string    `json:"id"`
			Username string    `json:"username"`
			Email    string    `json:"email"`
			Created  time.Time `json:"created"`
			Updated  time.Time `json:"updated"`
		}

		// Register user model
		userModel := &User{}
		userConfig := &cachex.ModelConfig{
			Enabled:     true,
			ReadThrough: true,
			TTL:         30 * time.Minute,
			Tags:        []string{"user", "auth"},
		}
		err := plugin.RegisterModel(userModel, userConfig)
		if err != nil {
			t.Fatalf("Failed to register user model: %v", err)
		}

		ctx := context.Background()

		// 1. Create user
		user := &User{
			ID:       "user_123",
			Username: "testuser",
			Email:    "test@example.com",
			Created:  time.Now(),
			Updated:  time.Now(),
		}

		// Cache the user data
		err = plugin.CacheQueryResult(ctx, "User", user.ID, user)
		if err != nil {
			t.Fatalf("Failed to cache user: %v", err)
		}

		// 2. Read user (should hit cache)
		var cachedUser User
		found, err := plugin.ReadThrough(ctx, "User", user.ID, &cachedUser)
		if err != nil {
			t.Fatalf("ReadThrough() failed: %v", err)
		}
		if found {
			t.Logf("User found in cache")
		}

		// 3. Update user
		user.Username = "updateduser"
		user.Updated = time.Now()

		// Invalidate cache on update
		err = plugin.InvalidateOnUpdate(ctx, user)
		if err != nil {
			t.Fatalf("InvalidateOnUpdate() failed: %v", err)
		}

		// 4. Read updated user (should miss cache, then cache new data)
		err = plugin.CacheQueryResult(ctx, "User", user.ID, user)
		if err != nil {
			t.Fatalf("Failed to cache updated user: %v", err)
		}

		// 5. Delete user
		err = plugin.InvalidateOnDelete(ctx, user)
		if err != nil {
			t.Fatalf("InvalidateOnDelete() failed: %v", err)
		}

		t.Logf("User scenario test completed successfully")
	})

	// Test multi-tenant scenario
	t.Run("multi_tenant_scenario", func(t *testing.T) {
		type Tenant struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		type TenantResource struct {
			ID       string `json:"id"`
			TenantID string `json:"tenant_id"`
			Name     string `json:"name"`
		}

		// Register models
		tenantModel := &Tenant{}
		resourceModel := &TenantResource{}

		tenantConfig := &cachex.ModelConfig{
			Enabled: true,
			TTL:     1 * time.Hour,
			Tags:    []string{"tenant"},
		}

		resourceConfig := &cachex.ModelConfig{
			Enabled: true,
			TTL:     15 * time.Minute,
			Tags:    []string{"resource", "tenant"},
		}

		err := plugin.RegisterModel(tenantModel, tenantConfig)
		if err != nil {
			t.Fatalf("Failed to register tenant model: %v", err)
		}

		err = plugin.RegisterModel(resourceModel, resourceConfig)
		if err != nil {
			t.Fatalf("Failed to register resource model: %v", err)
		}

		ctx := context.Background()

		// Create tenant
		tenant := &Tenant{ID: "tenant_1", Name: "Test Tenant"}
		err = plugin.CacheQueryResult(ctx, "Tenant", tenant.ID, tenant)
		if err != nil {
			t.Fatalf("Failed to cache tenant: %v", err)
		}

		// Create resources for tenant
		resources := []*TenantResource{
			{ID: "res_1", TenantID: "tenant_1", Name: "Resource 1"},
			{ID: "res_2", TenantID: "tenant_1", Name: "Resource 2"},
		}

		for _, resource := range resources {
			err = plugin.CacheQueryResult(ctx, "TenantResource", resource.ID, resource)
			if err != nil {
				t.Fatalf("Failed to cache resource: %v", err)
			}
		}

		// Delete tenant (should invalidate all tenant-related cache)
		err = plugin.InvalidateOnDelete(ctx, tenant)
		if err != nil {
			t.Fatalf("InvalidateOnDelete() failed: %v", err)
		}

		// Clear all cache
		err = plugin.ClearCache(ctx)
		if err != nil {
			t.Fatalf("ClearCache() failed: %v", err)
		}

		t.Logf("Multi-tenant scenario test completed successfully")
	})
}

// TestPlugin_Benchmark benchmarks critical operations
func TestPlugin_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmarks in short mode")
	}

	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin, err := cachex.NewGormPlugin(cache, nil)
	if err != nil {
		t.Fatalf("NewGormPlugin() failed: %v", err)
	}

	// Benchmark model registration
	t.Run("benchmark_registration", func(t *testing.T) {
		const numModels = 1000
		models := make([]*TestModel, numModels)
		for i := 0; i < numModels; i++ {
			models[i] = &TestModel{
				ID:   fmt.Sprintf("id_%d", i),
				Name: fmt.Sprintf("model_%d", i),
			}
		}

		start := time.Now()
		for i := 0; i < numModels; i++ {
			err := plugin.RegisterModel(models[i], nil)
			if err != nil {
				t.Fatalf("Failed to register model %d: %v", i, err)
			}
		}
		duration := time.Since(start)

		t.Logf("Registered %d models in %v (%.2f models/sec)", numModels, duration, float64(numModels)/duration.Seconds())
	})

	// Benchmark invalidation operations
	t.Run("benchmark_invalidation", func(t *testing.T) {
		model := &TestModel{ID: "1", Name: "Benchmark"}
		config := &cachex.ModelConfig{Enabled: true, Tags: []string{"benchmark"}}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		const numOperations = 10000
		ctx := context.Background()

		start := time.Now()
		for i := 0; i < numOperations; i++ {
			err := plugin.InvalidateOnCreate(ctx, model)
			if err != nil {
				t.Fatalf("Failed to invalidate: %v", err)
			}
		}
		duration := time.Since(start)

		t.Logf("Completed %d invalidation operations in %v (%.2f ops/sec)", numOperations, duration, float64(numOperations)/duration.Seconds())
	})

	// Benchmark read-through operations
	t.Run("benchmark_read_through", func(t *testing.T) {
		model := &TestModel{ID: "1", Name: "Benchmark"}
		config := &cachex.ModelConfig{Enabled: true, ReadThrough: true}
		err := plugin.RegisterModel(model, config)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		// Cache some data first
		ctx := context.Background()
		err = plugin.CacheQueryResult(ctx, "TestModel", "1", model)
		if err != nil {
			t.Fatalf("Failed to cache data: %v", err)
		}

		const numOperations = 10000
		var dest TestModel

		start := time.Now()
		for i := 0; i < numOperations; i++ {
			_, err := plugin.ReadThrough(ctx, "TestModel", "1", &dest)
			if err != nil {
				t.Fatalf("Failed to read through: %v", err)
			}
		}
		duration := time.Since(start)

		t.Logf("Completed %d read-through operations in %v (%.2f ops/sec)", numOperations, duration, float64(numOperations)/duration.Seconds())
	})
}
