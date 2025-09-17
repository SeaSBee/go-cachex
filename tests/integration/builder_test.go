package integration

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/seasbee/go-logx"
)

// IntegrationTestUser represents a test user for integration tests
type IntegrationTestUser struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Age      int       `json:"age"`
	Created  time.Time `json:"created"`
	IsActive bool      `json:"is_active"`
}

// Test cache builder basic functionality
func TestCacheBuilderBasic(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test building cache
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

	logx.Info("Cache builder basic test completed")
}

// Test cache builder with custom codec
func TestCacheBuilderWithCustomCodec(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test building cache with custom codec
	cache, err := cachex.New[IntegrationTestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewJSONCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with custom codec: %v", err)
	}
	defer cache.Close()

	// Test with IntegrationTestUser struct
	user := IntegrationTestUser{
		ID:       "user:1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		Created:  time.Now(),
		IsActive: true,
	}

	ctx := context.Background()

	// Set user
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set user: %v", setResult.Error)
	}

	// Get user
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

	logx.Info("Cache builder with custom codec test completed")
}

// Test cache builder with key builder
func TestCacheBuilderWithKeyBuilder(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test building cache with key builder
	cache, err := cachex.New[IntegrationTestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with key builder: %v", err)
	}
	defer cache.Close()

	// Test key building using the Builder
	keyBuilder, err := cachex.NewBuilder("test", "env", "secret")
	if err != nil {
		t.Fatalf("Failed to create key builder: %v", err)
	}

	// Test Build method
	key := keyBuilder.Build("user", "123")
	if key == "" {
		t.Error("Expected non-empty key")
	}

	// Test BuildList method
	listKey := keyBuilder.BuildList("user", map[string]any{
		"age":    30,
		"active": true,
	})
	if listKey == "" {
		t.Error("Expected non-empty list key")
	}

	// Test BuildComposite method
	compositeKey := keyBuilder.BuildComposite("user", "123", "order", "456")
	if compositeKey == "" {
		t.Error("Expected non-empty composite key")
	}

	// Test BuildSession method
	sessionKey := keyBuilder.BuildSession("session123")
	if sessionKey == "" {
		t.Error("Expected non-empty session key")
	}

	// Test cache operations with built keys
	user := IntegrationTestUser{
		ID:       "123",
		Name:     "Test User",
		Email:    "test@example.com",
		Age:      25,
		Created:  time.Now(),
		IsActive: true,
	}

	ctx := context.Background()

	// Set user using built key
	setResult := <-cache.Set(ctx, key, user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set user: %v", setResult.Error)
	}

	// Get user using built key
	getResult := <-cache.Get(ctx, key)
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

	logx.Info("Cache builder with key builder test completed")
}

// Test cache builder with observability
func TestCacheBuilderWithObservability(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test building cache with observability
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithObservability(cachex.ObservabilityConfig{
			EnableMetrics: true,
			EnableTracing: true,
			EnableLogging: true,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with observability: %v", err)
	}
	defer cache.Close()

	// Test basic operations
	ctx := context.Background()
	key := "test-obs-key"
	value := "test-obs-value"

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

	logx.Info("Cache builder with observability test completed")
}
