package integration

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/SeaSBee/go-logx"
)

// Helper function to create Redis store for tagging testing
func createRedisStoreForTagging(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	return store
}

// Test tagging with basic operations
func TestTaggingBasicOperations(t *testing.T) {
	store := createRedisStoreForTagging(t)
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

	// Test basic operations with tagging
	ctx := context.Background()
	key1 := "tagging-key-1"
	key2 := "tagging-key-2"
	value1 := "tagging-value-1"
	value2 := "tagging-value-2"

	// Set values
	setResult1 := <-cache.Set(ctx, key1, value1, 5*time.Minute)
	if setResult1.Error != nil {
		t.Fatalf("Failed to set value1: %v", setResult1.Error)
	}

	setResult2 := <-cache.Set(ctx, key2, value2, 5*time.Minute)
	if setResult2.Error != nil {
		t.Fatalf("Failed to set value2: %v", setResult2.Error)
	}

	// Get values
	getResult1 := <-cache.Get(ctx, key1)
	if getResult1.Error != nil {
		t.Fatalf("Failed to get value1: %v", getResult1.Error)
	}
	if !getResult1.Found {
		t.Fatal("Value1 not found")
	}

	getResult2 := <-cache.Get(ctx, key2)
	if getResult2.Error != nil {
		t.Fatalf("Failed to get value2: %v", getResult2.Error)
	}
	if !getResult2.Found {
		t.Fatal("Value2 not found")
	}

	if getResult1.Value != value1 {
		t.Errorf("Expected value1 %s, got %s", value1, getResult1.Value)
	}
	if getResult2.Value != value2 {
		t.Errorf("Expected value2 %s, got %s", value2, getResult2.Value)
	}

	logx.Info("Tagging basic operations test completed")
}

// Test tagging with different data types
func TestTaggingDifferentDataTypes(t *testing.T) {
	store := createRedisStoreForTagging(t)
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
	intKey := "tagging-int-key"

	intSetResult := <-intCache.Set(ctx, intKey, intValue, 5*time.Minute)
	if intSetResult.Error != nil {
		t.Fatalf("Failed to set int value: %v", intSetResult.Error)
	}

	intGetResult := <-intCache.Get(ctx, intKey)
	if intGetResult.Error != nil {
		t.Fatalf("Failed to get int value: %v", intGetResult.Error)
	}
	if !intGetResult.Found {
		t.Fatal("Int value not found")
	}
	if intGetResult.Value != intValue {
		t.Errorf("Expected int value %d, got %d", intValue, intGetResult.Value)
	}

	// Test with struct
	structCache, err := cachex.New[IntegrationTestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create struct cache: %v", err)
	}
	defer structCache.Close()

	user := IntegrationTestUser{
		ID:       "user:1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		Created:  time.Now(),
		IsActive: true,
	}

	structSetResult := <-structCache.Set(ctx, "user:1", user, 5*time.Minute)
	if structSetResult.Error != nil {
		t.Fatalf("Failed to set user: %v", structSetResult.Error)
	}

	structGetResult := <-structCache.Get(ctx, "user:1")
	if structGetResult.Error != nil {
		t.Fatalf("Failed to get user: %v", structGetResult.Error)
	}
	if !structGetResult.Found {
		t.Fatal("User not found")
	}

	if structGetResult.Value.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, structGetResult.Value.ID)
	}
	if structGetResult.Value.Name != user.Name {
		t.Errorf("Expected user name %s, got %s", user.Name, structGetResult.Value.Name)
	}

	logx.Info("Tagging different data types test completed")
}

// Test tagging with context
func TestTaggingWithContext(t *testing.T) {
	store := createRedisStoreForTagging(t)
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
	key := "tagging-context-key"
	value := "tagging-context-value"

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

	logx.Info("Tagging with context test completed")
}
