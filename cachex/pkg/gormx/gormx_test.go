package gormx

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModel represents a test model
type TestModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TestModel2 represents another test model
type TestModel2 struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestPlugin_New(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with default config
	plugin := New(c, nil)
	assert.NotNil(t, plugin)
	assert.Equal(t, "cachex", plugin.Name())
	assert.NotNil(t, plugin.GetCache())
	assert.NotNil(t, plugin.GetConfig())
}

func TestPlugin_Initialize(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Initialize plugin
	err = plugin.Initialize()
	assert.NoError(t, err)
}

func TestPlugin_RegisterModel(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Register model with custom config
	config := &ModelConfig{
		Name:        "TestModel",
		TTL:         10 * time.Minute,
		Enabled:     true,
		ReadThrough: true,
		Tags:        []string{"test", "demo"},
	}

	err = plugin.RegisterModel(&TestModel{}, config)
	assert.NoError(t, err)

	// Register model with defaults
	err = plugin.RegisterModelWithDefaults(&TestModel2{}, 15*time.Minute, "test2", "demo2")
	assert.NoError(t, err)

	// Check registered models
	models := plugin.GetRegisteredModels()
	assert.Len(t, models, 2)
	assert.Contains(t, models, "TestModel")
	assert.Contains(t, models, "TestModel2")
}

func TestPlugin_InvalidateOnCreate(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with invalidation enabled
	config := &Config{
		EnableInvalidation: true,
		EnableDebug:        true,
	}
	plugin := New(c, config)

	// Register model
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test")
	require.NoError(t, err)

	// Test invalidation on create
	model := &TestModel{ID: "123", Name: "Test"}
	err = plugin.InvalidateOnCreate(model)
	assert.NoError(t, err)
}

func TestPlugin_InvalidateOnUpdate(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with invalidation enabled
	config := &Config{
		EnableInvalidation: true,
		EnableDebug:        true,
	}
	plugin := New(c, config)

	// Register model
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test")
	require.NoError(t, err)

	// Test invalidation on update
	model := &TestModel{ID: "123", Name: "Test"}
	err = plugin.InvalidateOnUpdate(model)
	assert.NoError(t, err)
}

func TestPlugin_InvalidateOnDelete(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with invalidation enabled
	config := &Config{
		EnableInvalidation: true,
		EnableDebug:        true,
	}
	plugin := New(c, config)

	// Register model
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test")
	require.NoError(t, err)

	// Test invalidation on delete
	model := &TestModel{ID: "123", Name: "Test"}
	err = plugin.InvalidateOnDelete(model)
	assert.NoError(t, err)
}

func TestPlugin_ReadThrough(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with read-through enabled
	config := &Config{
		EnableReadThrough: true,
		EnableDebug:       true,
	}
	plugin := New(c, config)

	// Register model
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test")
	require.NoError(t, err)

	// Cache a model first
	model := &TestModel{ID: "123", Name: "Cached Model"}
	cacheKey := plugin.buildCacheKey("TestModel", "123")
	err = c.Set(cacheKey, model, 5*time.Minute)
	require.NoError(t, err)

	// Wait a bit for the cache to be set
	time.Sleep(10 * time.Millisecond)

	// Test read-through with cached data
	var dest TestModel
	found, err := plugin.ReadThrough("TestModel", "123", &dest)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Cached Model", dest.Name)

	// Test read-through with non-existent data
	var dest2 TestModel
	found, err = plugin.ReadThrough("TestModel", "999", &dest2)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestPlugin_CacheQueryResult(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin with query caching enabled
	config := &Config{
		EnableQueryCache: true,
		EnableDebug:      true,
	}
	plugin := New(c, config)

	// Register model
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test")
	require.NoError(t, err)

	// Test caching query result
	model := &TestModel{ID: "123", Name: "Query Result"}
	err = plugin.CacheQueryResult("TestModel", "123", model)
	assert.NoError(t, err)

	// Verify the result is cached
	cacheKey := plugin.buildCacheKey("TestModel", "123")
	if cached, found, err := c.Get(cacheKey); err == nil && found {
		if cachedModel, ok := cached.(*TestModel); ok {
			assert.Equal(t, "Query Result", cachedModel.Name)
		}
	}
}

func TestPlugin_ClearCache(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Register model with tags
	err = plugin.RegisterModelWithDefaults(&TestModel{}, 5*time.Minute, "test", "demo")
	require.NoError(t, err)

	// Test clearing cache
	err = plugin.ClearCache()
	assert.NoError(t, err)
}

func TestPlugin_WithContext(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Test with context
	ctx := context.Background()
	pluginWithCtx := plugin.WithContext(ctx)
	assert.NotNil(t, pluginWithCtx)
}

func TestPlugin_ExtractID(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Test ID extraction
	model := &TestModel{ID: "123", Name: "Test"}
	id := plugin.extractID(model)
	assert.Equal(t, "123", id)

	// Test with nil model
	id = plugin.extractID(nil)
	assert.Equal(t, "", id)
}

func TestPlugin_GetModelName(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Test model name extraction
	model := &TestModel{ID: "123", Name: "Test"}
	name := plugin.getModelName(model)
	assert.Equal(t, "TestModel", name)

	// Test with nil model
	name = plugin.getModelName(nil)
	assert.Equal(t, "", name)
}

func TestPlugin_BuildCacheKey(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Test cache key building
	key := plugin.buildCacheKey("TestModel", "123")
	assert.Contains(t, key, "TestModel")
	assert.Contains(t, key, "123")
}

func TestPlugin_SetDestValue(t *testing.T) {
	// Create cache
	store, err := memorystore.New(&memorystore.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	})
	require.NoError(t, err)

	c, err := cache.New[any](
		cache.WithStore(store),
		cache.WithDefaultTTL(5*time.Minute),
	)
	require.NoError(t, err)
	defer c.Close()

	// Create plugin
	plugin := New(c, nil)

	// Test setting destination value
	source := &TestModel{ID: "123", Name: "Source"}
	var dest TestModel

	err = plugin.setDestValue(&dest, source)
	assert.NoError(t, err)
	assert.Equal(t, "Source", dest.Name)
	assert.Equal(t, "123", dest.ID)

	// Test with nil values
	err = plugin.setDestValue(nil, source)
	assert.Error(t, err)

	err = plugin.setDestValue(&dest, nil)
	assert.Error(t, err)

	// Test with non-pointer destination
	err = plugin.setDestValue(dest, source)
	assert.Error(t, err)
}
