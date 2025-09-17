package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Helper function to create Redis store for security testing
func createRedisStoreForSecurity(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	return store
}

// Test basic cache operations (security context)
func TestSecurityBasicOperations(t *testing.T) {
	store := createRedisStoreForSecurity(t)
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

	// Test basic operations
	ctx := context.Background()
	key := "security-basic-key"
	value := "security-basic-value"

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

	logx.Info("Security basic operations test completed")
}

// Test cache with different data types (security context)
func TestSecurityDifferentDataTypes(t *testing.T) {
	store := createRedisStoreForSecurity(t)
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
	intKey := "security-int-key"

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

	logx.Info("Security different data types test completed")
}

// Test cache with context (security context)
func TestSecurityWithContext(t *testing.T) {
	store := createRedisStoreForSecurity(t)
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
	key := "security-context-key"
	value := "security-context-value"

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

	logx.Info("Security with context test completed")
}
