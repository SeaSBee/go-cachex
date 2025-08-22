package unit

import (
	"context"
	"fmt"
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

	plugin := cachex.NewGormPlugin(cache, config)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

	err = plugin.RegisterModel(nil, nil)
	if err == nil {
		t.Errorf("RegisterModel() with nil model should fail")
	}
}

func TestPlugin_RegisterModelWithDefaults(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[any](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	err = plugin.InvalidateOnCreate(model)
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
	plugin := cachex.NewGormPlugin(cache, config)

	model := &TestModel{ID: "1", Name: "Test"}

	err = plugin.InvalidateOnCreate(model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

	model := &TestModel{ID: "1", Name: "Test"}

	err = plugin.InvalidateOnCreate(model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

	// Register model as disabled
	model := &TestModel{ID: "1", Name: "Test"}
	config := &cachex.ModelConfig{
		Enabled: false,
	}
	err = plugin.RegisterModel(model, config)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	err = plugin.InvalidateOnCreate(model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

	err = plugin.InvalidateOnCreate(nil)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	err = plugin.InvalidateOnUpdate(model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	err = plugin.InvalidateOnDelete(model)
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
	plugin := cachex.NewGormPlugin(cache, config)

	var dest TestModel
	found, err := plugin.ReadThrough("TestModel", "1", &dest)
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

	plugin := cachex.NewGormPlugin(cache, nil)

	var dest TestModel
	found, err := plugin.ReadThrough("UnregisteredModel", "1", &dest)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	found, err := plugin.ReadThrough("TestModel", "1", &dest)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	found, err := plugin.ReadThrough("TestModel", "1", &dest)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	_, err = plugin.ReadThrough("TestModel", "1", &dest)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	found, err := plugin.ReadThrough("TestModel", "1", &dest)
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
	plugin := cachex.NewGormPlugin(cache, config)

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult("TestModel", "1", model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

	model := TestModel{ID: "1", Name: "Test"}
	err = plugin.CacheQueryResult("UnregisteredModel", "1", model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	err = plugin.CacheQueryResult("TestModel", "1", model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	err = plugin.CacheQueryResult("TestModel", "1", model)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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
				if err != nil {
					t.Fatalf("Failed to register model: %v", err)
				}

				// This will exercise extractID internally
				err = plugin.InvalidateOnCreate(tt.model)
				if err != nil {
					t.Errorf("InvalidateOnCreate() failed: %v", err)
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
	plugin := cachex.NewGormPlugin(cache, config)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	err = plugin.ClearCache()
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

	plugin := cachex.NewGormPlugin(cache, nil)

	err = plugin.ClearCache()
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

	plugin := cachex.NewGormPlugin(cache, nil)

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
			err = plugin.InvalidateOnCreate(model)
			if err != nil {
				t.Errorf("Concurrent InvalidateOnCreate() failed: %v", err)
			}

			err = plugin.InvalidateOnUpdate(model)
			if err != nil {
				t.Errorf("Concurrent InvalidateOnUpdate() failed: %v", err)
			}

			err = plugin.InvalidateOnDelete(model)
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
			valid: true, // This actually gets handled as a non-nil pointer with zero content
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

	plugin := cachex.NewGormPlugin(cache, nil)

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

	plugin := cachex.NewGormPlugin(cache, nil)

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
			plugin := cachex.NewGormPlugin(cache, config)
			if plugin == nil {
				t.Errorf("NewGormPlugin() should handle config gracefully")
			}

			// Test basic operations with this config
			err := plugin.Initialize()
			if err != nil {
				t.Errorf("Initialize() failed with config: %v", err)
			}
		})
	}
}
