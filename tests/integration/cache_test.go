package integration

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Helper function to create Redis store for cache testing
func createRedisStoreForCache(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	return store
}

// Test basic cache operations
func TestCacheBasicOperations(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Create cache
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test Set and Get
	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value
	setResult := <-cache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	// Get value
	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	// Test Exists
	existsResult := <-cache.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence: %v", existsResult.Error)
	}
	if !existsResult.Found {
		t.Error("Key should exist")
	}

	// Test TTL
	ttlResult := <-cache.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Fatalf("Failed to get TTL: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttlResult.TTL)
	}

	// Test Delete
	delResult := <-cache.Del(ctx, key)
	if delResult.Error != nil {
		t.Fatalf("Failed to delete key: %v", delResult.Error)
	}

	// Verify deletion
	existsResult = <-cache.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence after deletion: %v", existsResult.Error)
	}
	if existsResult.Found {
		t.Error("Key should not exist after deletion")
	}

	logx.Info("Cache basic operations test completed")
}

// Test cache MGet and MSet operations
func TestCacheBatchOperations(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Create cache
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test MSet
	ctx := context.Background()
	items := map[string]string{
		"batch-key-1": "value-1",
		"batch-key-2": "value-2",
		"batch-key-3": "value-3",
		"batch-key-4": "value-4",
	}

	msetResult := <-cache.MSet(ctx, items, 5*time.Minute)
	if msetResult.Error != nil {
		t.Fatalf("Failed to MSet values: %v", msetResult.Error)
	}

	// Test MGet
	keys := []string{"batch-key-1", "batch-key-2", "batch-key-3", "batch-key-4"}
	mgetResult := <-cache.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Fatalf("Failed to MGet values: %v", mgetResult.Error)
	}

	if len(mgetResult.Values) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(mgetResult.Values))
	}

	for key, expectedValue := range items {
		if retrievedValue, exists := mgetResult.Values[key]; !exists {
			t.Errorf("Key %s not found in retrieved items", key)
		} else if retrievedValue != expectedValue {
			t.Errorf("Expected value %s for key %s, got %s", expectedValue, key, retrievedValue)
		}
	}

	logx.Info("Cache batch operations test completed")
}

// Test cache with different data types
func TestCacheWithDifferentDataTypes(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Test with int
	intCache, err := cachex.New[int](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create int cache: %v", err)
	}
	defer intCache.Close()

	ctx := context.Background()
	intValue := 42
	intKey := "int-key"

	setResult := <-intCache.Set(ctx, intKey, intValue, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set int value: %v", setResult.Error)
	}

	getResult := <-intCache.Get(ctx, intKey)
	if getResult.Error != nil {
		t.Fatalf("Failed to get int value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Int value not found")
	}
	if getResult.Value != intValue {
		t.Errorf("Expected int value %d, got %d", intValue, getResult.Value)
	}

	// Test with float64
	floatCache, err := cachex.New[float64](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create float cache: %v", err)
	}
	defer floatCache.Close()

	floatValue := 3.14159
	floatKey := "float-key"

	floatSetResult := <-floatCache.Set(ctx, floatKey, floatValue, 5*time.Minute)
	if floatSetResult.Error != nil {
		t.Fatalf("Failed to set float value: %v", floatSetResult.Error)
	}

	floatGetResult := <-floatCache.Get(ctx, floatKey)
	if floatGetResult.Error != nil {
		t.Fatalf("Failed to get float value: %v", floatGetResult.Error)
	}
	if !floatGetResult.Found {
		t.Fatal("Float value not found")
	}
	if floatGetResult.Value != floatValue {
		t.Errorf("Expected float value %f, got %f", floatValue, floatGetResult.Value)
	}

	// Test with bool
	boolCache, err := cachex.New[bool](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create bool cache: %v", err)
	}
	defer boolCache.Close()

	boolValue := true
	boolKey := "bool-key"

	boolSetResult := <-boolCache.Set(ctx, boolKey, boolValue, 5*time.Minute)
	if boolSetResult.Error != nil {
		t.Fatalf("Failed to set bool value: %v", boolSetResult.Error)
	}

	boolGetResult := <-boolCache.Get(ctx, boolKey)
	if boolGetResult.Error != nil {
		t.Fatalf("Failed to get bool value: %v", boolGetResult.Error)
	}
	if !boolGetResult.Found {
		t.Fatal("Bool value not found")
	}
	if boolGetResult.Value != boolValue {
		t.Errorf("Expected bool value %t, got %t", boolValue, boolGetResult.Value)
	}

	logx.Info("Cache with different data types test completed")
}

// Test cache with struct
func TestCacheWithStruct(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Create cache with struct type
	cache, err := cachex.New[IntegrationTestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test with struct
	ctx := context.Background()
	user := IntegrationTestUser{
		ID:       "user:1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		Created:  time.Now(),
		IsActive: true,
	}

	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set user: %v", setResult.Error)
	}

	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Fatalf("Failed to get user: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("User not found")
	}

	if getResult.Value.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, getResult.Value.ID)
	}
	if getResult.Value.Name != user.Name {
		t.Errorf("Expected user name %s, got %s", user.Name, getResult.Value.Name)
	}
	if getResult.Value.Email != user.Email {
		t.Errorf("Expected user email %s, got %s", user.Email, getResult.Value.Email)
	}
	if getResult.Value.Age != user.Age {
		t.Errorf("Expected user age %d, got %d", user.Age, getResult.Value.Age)
	}
	if getResult.Value.IsActive != user.IsActive {
		t.Errorf("Expected user is_active %t, got %t", user.IsActive, getResult.Value.IsActive)
	}

	logx.Info("Cache with struct test completed")
}

// Test cache with context
func TestCacheWithContext(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Create cache
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test with context
	ctx := context.Background()
	key := "context-key"
	value := "context-value"

	setResult := <-cache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	logx.Info("Cache with context test completed")
}

// Test cache with timeout context
func TestCacheWithTimeoutContext(t *testing.T) {
	store := createRedisStoreForCache(t)
	defer store.Close()

	// Create cache
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "timeout-key"
	value := "timeout-value"

	setResult := <-cache.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	getResult := <-cache.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Fatal("Value not found")
	}

	if getResult.Value != value {
		t.Errorf("Expected value %s, got %s", value, getResult.Value)
	}

	logx.Info("Cache with timeout context test completed")
}
