package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/SeaSBee/go-logx"
)

// Helper function to create Redis store for msgpack testing
func createRedisStoreForMsgpack(t *testing.T) cachex.Store {
	config := cachex.DefaultRedisConfig()
	config.Addr = "localhost:6379"

	store, err := cachex.NewRedisStore(config)
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	return store
}

// Test msgpack codec with basic types
func TestMsgpackCodecBasicTypes(t *testing.T) {
	store := createRedisStoreForMsgpack(t)
	defer store.Close()

	// Create cache with msgpack codec
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with msgpack codec: %v", err)
	}
	defer cache.Close()

	// Test basic operations
	ctx := context.Background()
	key := "msgpack-basic-key"
	value := "msgpack-basic-value"

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

	logx.Info("Msgpack codec basic types test completed")
}

// Test msgpack codec with complex data structures
func TestMsgpackCodecComplexData(t *testing.T) {
	store := createRedisStoreForMsgpack(t)
	defer store.Close()

	// Create cache with msgpack codec for complex data
	cache, err := cachex.New[map[string]interface{}](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with msgpack codec: %v", err)
	}
	defer cache.Close()

	// Test complex data structure
	ctx := context.Background()
	key := "msgpack-complex-key"
	value := map[string]interface{}{
		"string": "test-string",
		"int":    42,
		"float":  3.14159,
		"bool":   true,
		"array":  []interface{}{1, 2, 3, "four"},
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

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

	// Compare values (simplified comparison)
	if len(getResult.Value) != len(value) {
		t.Errorf("Expected %d keys, got %d", len(value), len(getResult.Value))
	}

	// Check a few key values (use fmt.Sprintf for comparison to handle type differences)
	if fmt.Sprintf("%v", getResult.Value["string"]) != fmt.Sprintf("%v", value["string"]) {
		t.Errorf("Expected string %v, got %v", value["string"], getResult.Value["string"])
	}
	if fmt.Sprintf("%v", getResult.Value["int"]) != fmt.Sprintf("%v", value["int"]) {
		t.Errorf("Expected int %v, got %v", value["int"], getResult.Value["int"])
	}
	if fmt.Sprintf("%v", getResult.Value["bool"]) != fmt.Sprintf("%v", value["bool"]) {
		t.Errorf("Expected bool %v, got %v", value["bool"], getResult.Value["bool"])
	}

	logx.Info("Msgpack codec complex data test completed")
}

// Test msgpack codec with different data types
func TestMsgpackCodecDifferentTypes(t *testing.T) {
	store := createRedisStoreForMsgpack(t)
	defer store.Close()

	// Test with int
	intCache, err := cachex.New[int](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create int cache with msgpack codec: %v", err)
	}
	defer intCache.Close()

	ctx := context.Background()
	intValue := 42
	intKey := "msgpack-int-key"

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

	// Test with float64
	floatCache, err := cachex.New[float64](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create float cache with msgpack codec: %v", err)
	}
	defer floatCache.Close()

	floatValue := 3.14159
	floatKey := "msgpack-float-key"

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
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create bool cache with msgpack codec: %v", err)
	}
	defer boolCache.Close()

	boolValue := true
	boolKey := "msgpack-bool-key"

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

	logx.Info("Msgpack codec different types test completed")
}

// Test msgpack codec performance
func TestMsgpackCodecPerformance(t *testing.T) {
	store := createRedisStoreForMsgpack(t)
	defer store.Close()

	// Create cache with msgpack codec
	msgpackCache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithCodec(cachex.NewMessagePackCodec()),
	)
	if err != nil {
		t.Fatalf("Failed to create cache with msgpack codec: %v", err)
	}
	defer msgpackCache.Close()

	// Test performance
	ctx := context.Background()
	key := "performance-test-key"
	value := "performance-test-value"

	// Test msgpack codec
	msgpackSetResult := <-msgpackCache.Set(ctx, key, value, 5*time.Minute)
	if msgpackSetResult.Error != nil {
		t.Fatalf("Failed to set value with msgpack codec: %v", msgpackSetResult.Error)
	}

	msgpackGetResult := <-msgpackCache.Get(ctx, key)
	if msgpackGetResult.Error != nil {
		t.Fatalf("Failed to get value with msgpack codec: %v", msgpackGetResult.Error)
	}
	if !msgpackGetResult.Found {
		t.Fatal("Value not found with msgpack codec")
	}

	if msgpackGetResult.Value != value {
		t.Errorf("Expected value %s with msgpack codec, got %s", value, msgpackGetResult.Value)
	}

	logx.Info("Msgpack codec performance test completed")
}
