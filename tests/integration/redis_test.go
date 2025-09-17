package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Test data structures
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// Helper function to create Redis store for testing
func createRedisStore(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	return store
}

// Test basic Redis operations
func TestRedisBasicOperations(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test Set and Get
	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set value
	setResult := <-store.Set(ctx, key, value, time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value: %v", setResult.Error)
	}

	// Get value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Failed to get value: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Fatal("Value not found")
	}

	if string(getResult.Value) != string(value) {
		t.Errorf("Expected value %s, got %s", string(value), string(getResult.Value))
	}

	// Test Exists
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence: %v", existsResult.Error)
	}
	if !existsResult.Exists {
		t.Error("Key should exist")
	}

	// Test TTL
	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Fatalf("Failed to get TTL: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 {
		t.Errorf("Expected positive TTL, got %v", ttlResult.TTL)
	}

	// Test Delete
	delResult := <-store.Del(ctx, key)
	if delResult.Error != nil {
		t.Fatalf("Failed to delete key: %v", delResult.Error)
	}

	// Verify deletion
	existsResult = <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Failed to check existence after deletion: %v", existsResult.Error)
	}
	if existsResult.Exists {
		t.Error("Key should not exist after deletion")
	}

	logx.Info("Redis basic operations test completed")
}

// Test Redis batch operations
func TestRedisBatchOperations(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Test MSet and MGet
	ctx := context.Background()
	items := map[string][]byte{
		"batch-key-1": []byte("value-1"),
		"batch-key-2": []byte("value-2"),
		"batch-key-3": []byte("value-3"),
		"batch-key-4": []byte("value-4"),
	}

	// Set batch values
	msetResult := <-store.MSet(ctx, items, time.Minute)
	if msetResult.Error != nil {
		t.Fatalf("Failed to MSet values: %v", msetResult.Error)
	}

	// Get batch values
	keys := []string{"batch-key-1", "batch-key-2", "batch-key-3", "batch-key-4"}
	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Fatalf("Failed to MGet values: %v", mgetResult.Error)
	}

	if len(mgetResult.Values) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(mgetResult.Values))
	}

	for key, expectedValue := range items {
		if retrievedValue, exists := mgetResult.Values[key]; !exists {
			t.Errorf("Key %s not found in retrieved items", key)
		} else if string(retrievedValue) != string(expectedValue) {
			t.Errorf("Expected value %s for key %s, got %s", string(expectedValue), key, string(retrievedValue))
		}
	}

	logx.Info("Redis batch operations test completed")
}

// Test Redis increment operations
func TestRedisIncrementOperations(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	ctx := context.Background()
	key := fmt.Sprintf("counter-%d", time.Now().UnixNano())

	// Test initial increment
	incrResult := <-store.IncrBy(ctx, key, 5, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to increment: %v", incrResult.Error)
	}
	if incrResult.Result != 5 {
		t.Errorf("Expected value 5, got %d", incrResult.Result)
	}

	// Test subsequent increment
	incrResult = <-store.IncrBy(ctx, key, 3, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to increment: %v", incrResult.Error)
	}
	if incrResult.Result != 8 {
		t.Errorf("Expected value 8, got %d", incrResult.Result)
	}

	// Test negative increment
	incrResult = <-store.IncrBy(ctx, key, -2, time.Minute)
	if incrResult.Error != nil {
		t.Fatalf("Failed to decrement: %v", incrResult.Error)
	}
	if incrResult.Result != 6 {
		t.Errorf("Expected value 6, got %d", incrResult.Result)
	}

	logx.Info("Redis increment operations test completed")
}

// Test Redis with complex data structures
func TestRedisComplexDataStructures(t *testing.T) {
	store := createRedisStore(t)
	defer store.Close()

	// Create cache with Redis store
	cache, err := cachex.New[User](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test with User struct
	ctx := context.Background()
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

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
	if getResult.Value.Email != user.Email {
		t.Errorf("Expected user email %s, got %s", user.Email, getResult.Value.Email)
	}

	logx.Info("Redis complex data structures test completed")
}
