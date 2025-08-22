package unit

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

func TestCacheValidation_EmptyKey(t *testing.T) {
	// Create cache with security manager
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024, // 1MB
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test Set with empty key
	setResult := <-cache.Set(ctx, "", "value", time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for empty key in Set, got nil")
	}

	// Test Get with empty key
	getResult := <-cache.Get(ctx, "")
	if getResult.Error == nil {
		t.Error("Expected error for empty key in Get, got nil")
	}

	// Test MSet with empty key
	items := map[string]string{"": "value"}
	msetResult := <-cache.MSet(ctx, items, time.Minute)
	if msetResult.Error == nil {
		t.Error("Expected error for empty key in MSet, got nil")
	}

	// Test MGet with empty key
	mgetResult := <-cache.MGet(ctx, "")
	if mgetResult.Error == nil {
		t.Error("Expected error for empty key in MGet, got nil")
	}

	// Test Del with empty key
	delResult := <-cache.Del(ctx, "")
	if delResult.Error == nil {
		t.Error("Expected error for empty key in Del, got nil")
	}

	// Test Exists with empty key
	existsResult := <-cache.Exists(ctx, "")
	if existsResult.Error == nil {
		t.Error("Expected error for empty key in Exists, got nil")
	}

	// Test TTL with empty key
	ttlResult := <-cache.TTL(ctx, "")
	if ttlResult.Error == nil {
		t.Error("Expected error for empty key in TTL, got nil")
	}

	// Test IncrBy with empty key
	incrResult := <-cache.IncrBy(ctx, "", 1, time.Minute)
	if incrResult.Error == nil {
		t.Error("Expected error for empty key in IncrBy, got nil")
	}
}

func TestCacheValidation_NilValue(t *testing.T) {
	// Create cache with security manager
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024, // 1MB
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[[]byte](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test Set with nil value
	var nilValue []byte
	setResult := <-cache.Set(ctx, "key", nilValue, time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for nil value in Set, got nil")
	} else {
		// Check that it's an encode error (which is the correct behavior)
		expectedError := "cache set error for key key: encode error: cannot encode nil value"
		if setResult.Error.Error() != expectedError {
			t.Errorf("Expected encode error for nil value, got: %v", setResult.Error)
		}
	}

	// Test MSet with nil value
	items := map[string][]byte{"key": nil}
	msetResult := <-cache.MSet(ctx, items, time.Minute)
	if msetResult.Error == nil {
		t.Error("Expected error for nil value in MSet, got nil")
	} else {
		// Check that it's an encode error (which is the correct behavior)
		expectedError := "cache mset error for key key: encode error: cannot encode nil value"
		if msetResult.Error.Error() != expectedError {
			t.Errorf("Expected encode error for nil value, got: %v", msetResult.Error)
		}
	}
}

func TestCacheValidation_NegativeTTL(t *testing.T) {
	// Create cache with security manager
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024, // 1MB
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test Set with negative TTL
	setResult := <-cache.Set(ctx, "key", "value", -time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for negative TTL in Set, got nil")
	}

	// Test MSet with negative TTL
	items := map[string]string{"key": "value"}
	msetResult := <-cache.MSet(ctx, items, -time.Minute)
	if msetResult.Error == nil {
		t.Error("Expected error for negative TTL in MSet, got nil")
	}

	// Test IncrBy with negative TTL
	incrResult := <-cache.IncrBy(ctx, "key", 1, -time.Minute)
	if incrResult.Error == nil {
		t.Error("Expected error for negative TTL in IncrBy, got nil")
	}
}

func TestCacheValidation_KeyLength(t *testing.T) {
	// Create cache with security manager that has short key length limit
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 10, // Very short limit
			MaxValueSize: 1024 * 1024,
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test with key that's too long
	longKey := "this_key_is_much_longer_than_ten_characters"
	setResult := <-cache.Set(ctx, longKey, "value", time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for key too long, got nil")
	}
}

func TestCacheValidation_ValueSize(t *testing.T) {
	// Create cache with security manager that has small value size limit
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 100, // Very small limit
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[[]byte](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test with value that's too large
	largeValue := make([]byte, 200) // Larger than 100 bytes
	setResult := <-cache.Set(ctx, "key", largeValue, time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for value too large, got nil")
	}
}

func TestCacheValidation_KeyPatterns(t *testing.T) {
	// Create cache with security manager that has pattern restrictions
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength:    256,
			MaxValueSize:    1024 * 1024,
			AllowedPatterns: []string{"^user:[0-9]+$"}, // Only user:123 format allowed
			BlockedPatterns: []string{"^admin:.*$"},    // Block admin keys
		},
	}
	securityManager, err := cachex.NewSecurityManager(securityConfig)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[string](
		cachex.WithStore(store),
		cachex.WithSecurity(securityManager),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test with valid key pattern
	setResult := <-cache.Set(ctx, "user:123", "value", time.Minute)
	if setResult.Error != nil {
		t.Errorf("Expected no error for valid key pattern, got: %v", setResult.Error)
	}

	// Test with invalid key pattern (doesn't match allowed pattern)
	setResult = <-cache.Set(ctx, "invalid_key", "value", time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for invalid key pattern, got nil")
	}

	// Test with blocked key pattern
	setResult = <-cache.Set(ctx, "admin:config", "value", time.Minute)
	if setResult.Error == nil {
		t.Error("Expected error for blocked key pattern, got nil")
	}
}

func TestCacheValidation_WithoutSecurityManager(t *testing.T) {
	// Create cache without security manager
	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	cache, err := cachex.New[string](
		cachex.WithStore(store),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test that operations work without security manager (no validation)
	setResult := <-cache.Set(ctx, "key", "value", time.Minute)
	if setResult.Error != nil {
		t.Errorf("Expected no error without security manager, got: %v", setResult.Error)
	}

	getResult := <-cache.Get(ctx, "key")
	if getResult.Error != nil {
		t.Errorf("Expected no error without security manager, got: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Error("Expected to find value, got not found")
	}
	if getResult.Value != "value" {
		t.Errorf("Expected value 'value', got '%s'", getResult.Value)
	}
}
