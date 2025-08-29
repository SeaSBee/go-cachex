package unit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

func TestDefaultMemoryConfig(t *testing.T) {
	config := cachex.DefaultMemoryConfig()

	if config.MaxSize != 10000 {
		t.Errorf("Expected MaxSize to be 10000, got %d", config.MaxSize)
	}
	if config.MaxMemoryMB != 100 {
		t.Errorf("Expected MaxMemoryMB to be 100, got %d", config.MaxMemoryMB)
	}
	if config.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected DefaultTTL to be 5 minutes, got %v", config.DefaultTTL)
	}
	if config.CleanupInterval != 1*time.Minute {
		t.Errorf("Expected CleanupInterval to be 1 minute, got %v", config.CleanupInterval)
	}
	if config.EvictionPolicy != cachex.EvictionPolicyLRU {
		t.Errorf("Expected EvictionPolicy to be LRU, got %v", config.EvictionPolicy)
	}
	if !config.EnableStats {
		t.Errorf("Expected EnableStats to be true")
	}
}

func TestHighPerformanceMemoryConfig(t *testing.T) {
	config := cachex.HighPerformanceMemoryConfig()

	if config.MaxSize != 50000 {
		t.Errorf("Expected MaxSize to be 50000, got %d", config.MaxSize)
	}
	if config.MaxMemoryMB != 500 {
		t.Errorf("Expected MaxMemoryMB to be 500, got %d", config.MaxMemoryMB)
	}
	if config.DefaultTTL != 10*time.Minute {
		t.Errorf("Expected DefaultTTL to be 10 minutes, got %v", config.DefaultTTL)
	}
	if config.CleanupInterval != 30*time.Second {
		t.Errorf("Expected CleanupInterval to be 30 seconds, got %v", config.CleanupInterval)
	}
	if config.EvictionPolicy != cachex.EvictionPolicyLRU {
		t.Errorf("Expected EvictionPolicy to be LRU, got %v", config.EvictionPolicy)
	}
	if !config.EnableStats {
		t.Errorf("Expected EnableStats to be true")
	}
}

func TestResourceConstrainedMemoryConfig(t *testing.T) {
	config := cachex.ResourceConstrainedMemoryConfig()

	if config.MaxSize != 1000 {
		t.Errorf("Expected MaxSize to be 1000, got %d", config.MaxSize)
	}
	if config.MaxMemoryMB != 10 {
		t.Errorf("Expected MaxMemoryMB to be 10, got %d", config.MaxMemoryMB)
	}
	if config.DefaultTTL != 2*time.Minute {
		t.Errorf("Expected DefaultTTL to be 2 minutes, got %v", config.DefaultTTL)
	}
	if config.CleanupInterval != 2*time.Minute {
		t.Errorf("Expected CleanupInterval to be 2 minutes, got %v", config.CleanupInterval)
	}
	if config.EvictionPolicy != cachex.EvictionPolicyTTL {
		t.Errorf("Expected EvictionPolicy to be TTL, got %v", config.EvictionPolicy)
	}
	if config.EnableStats {
		t.Errorf("Expected EnableStats to be false")
	}
}

func TestNewMemoryStore_WithNilConfig(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Errorf("Expected store to be non-nil")
	}
}

func TestNewMemoryStore_WithCustomConfig(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         100,
		MaxMemoryMB:     5,
		DefaultTTL:      30 * time.Second,
		CleanupInterval: 10 * time.Second,
		EvictionPolicy:  cachex.EvictionPolicyLFU,
		EnableStats:     false,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Errorf("Expected store to be non-nil")
	}
}

func TestNewMemoryStore_WithInvalidConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *cachex.MemoryConfig
		expectError bool
	}{
		{
			name: "negative_max_size",
			config: &cachex.MemoryConfig{
				MaxSize:         -1,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
			},
			expectError: true,
		},
		{
			name: "negative_max_memory",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     -1,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
			},
			expectError: true,
		},
		{
			name: "negative_default_ttl",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     100,
				DefaultTTL:      -5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
			},
			expectError: true,
		},
		{
			name: "negative_cleanup_interval",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: -1 * time.Minute,
			},
			expectError: true,
		},
		{
			name: "zero_cleanup_interval",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 0,
			},
			expectError: true,
		},
		{
			name: "invalid_eviction_policy",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
				EvictionPolicy:  "invalid",
			},
			expectError: true,
		},
		{
			name: "valid_config",
			config: &cachex.MemoryConfig{
				MaxSize:         1000,
				MaxMemoryMB:     100,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
				EvictionPolicy:  cachex.EvictionPolicyLRU,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := cachex.NewMemoryStore(tc.config)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for invalid config, but got none")
				}
				if store != nil {
					t.Errorf("Expected nil store for invalid config, but got store")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for valid config, but got: %v", err)
				}
				if store == nil {
					t.Errorf("Expected store for valid config, but got nil")
				} else {
					store.Close()
				}
			}
		})
	}
}

func TestMemoryStore_Get_Set_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value
	setResult := <-store.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Get value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the value")
		return
	}
	if string(getResult.Value) != string(value) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	getResult := <-store.Get(ctx, "non-existent-key")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Errorf("Get() should return false for non-existent key, got %v", getResult.Value)
	}
}

func TestMemoryStore_Get_Expired(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value with very short TTL
	setResult := <-store.Set(ctx, key, value, 10*time.Millisecond)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Get value (should be expired)
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Errorf("Get() should return false for expired key, got %v", getResult.Value)
	}
}

func TestMemoryStore_Set_DefaultTTL(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         100,
		DefaultTTL:      1 * time.Second,
		CleanupInterval: 500 * time.Millisecond,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value with zero TTL (should use default)
	setResult := <-store.Set(ctx, key, value, 0)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Get value immediately
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the value")
		return
	}
	if string(getResult.Value) != string(value) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
	}
}

func TestMemoryStore_Set_UpdateExisting(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value1 := []byte("value 1")
	value2 := []byte("value 2")

	// Set initial value
	setResult1 := <-store.Set(ctx, key, value1, 5*time.Minute)
	if setResult1.Error != nil {
		t.Errorf("Set() failed: %v", setResult1.Error)
	}

	// Update value
	setResult2 := <-store.Set(ctx, key, value2, 5*time.Minute)
	if setResult2.Error != nil {
		t.Errorf("Set() failed: %v", setResult2.Error)
	}

	// Get updated value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the value")
		return
	}
	if string(getResult.Value) != string(value2) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value2))
	}
}

func TestMemoryStore_MGet_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set multiple values
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for key, value := range testData {
		setResult := <-store.Set(ctx, key, value, 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Get multiple values
	keys := []string{"key1", "key2", "key3", "non-existent"}
	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
	}

	// Verify results
	if len(mgetResult.Values) != 3 {
		t.Errorf("MGet() returned wrong number of results: got %d, want 3", len(mgetResult.Values))
	}

	for key, expectedValue := range testData {
		if value, exists := mgetResult.Values[key]; !exists {
			t.Errorf("MGet() missing key %s", key)
		} else if string(value) != string(expectedValue) {
			t.Errorf("MGet() wrong value for key %s: got %v, want %v", key, string(value), string(expectedValue))
		}
	}

	// Non-existent key should not be in result
	if _, exists := mgetResult.Values["non-existent"]; exists {
		t.Errorf("MGet() should not return non-existent key")
	}
}

func TestMemoryStore_MGet_EmptyKeys(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	mgetResult := <-store.MGet(ctx)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
	}
	if len(mgetResult.Values) != 0 {
		t.Errorf("MGet() with empty keys should return empty result, got %v", mgetResult.Values)
	}
}

func TestMemoryStore_MSet_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set multiple values
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	msetResult := <-store.MSet(ctx, items, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() failed: %v", msetResult.Error)
	}

	// Verify all values were set
	for key, expectedValue := range items {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		}
		if !getResult.Exists {
			t.Errorf("Get() should find key %s", key)
		}
		if string(getResult.Value) != string(expectedValue) {
			t.Errorf("Get() wrong value for key %s: got %v, want %v", key, string(getResult.Value), string(expectedValue))
		}
	}
}

func TestMemoryStore_MSet_EmptyItems(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	msetResult := <-store.MSet(ctx, map[string][]byte{}, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() with empty items should not fail: %v", msetResult.Error)
	}
}

func TestMemoryStore_Del_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value
	setResult := <-store.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Delete value
	delResult := <-store.Del(ctx, key)
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
	}

	// Verify deletion
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Errorf("Get() should not find key after deletion")
	}
}

func TestMemoryStore_Del_MultipleKeys(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set multiple values
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		setResult := <-store.Set(ctx, key, []byte("value"), 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Delete multiple keys
	delResult := <-store.Del(ctx, keys...)
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
	}

	// Verify all deletions
	for _, key := range keys {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		}
		if getResult.Exists {
			t.Errorf("Get() should not find deleted key %s", key)
		}
	}
}

func TestMemoryStore_Del_NonExistentKeys(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	delResult := <-store.Del(ctx, "non-existent-key")
	if delResult.Error != nil {
		t.Errorf("Del() should not fail for non-existent key: %v", delResult.Error)
	}
}

func TestMemoryStore_Del_EmptyKeys(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	delResult := <-store.Del(ctx)
	if delResult.Error != nil {
		t.Errorf("Del() with empty keys should not fail: %v", delResult.Error)
	}
}

func TestMemoryStore_Exists_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value
	setResult := <-store.Set(ctx, key, value, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Check existence
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if !existsResult.Exists {
		t.Errorf("Exists() should return true for existing key")
	}
}

func TestMemoryStore_Exists_NotFound(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	existsResult := <-store.Exists(ctx, "non-existent-key")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if existsResult.Exists {
		t.Errorf("Exists() should return false for non-existent key")
	}
}

func TestMemoryStore_Exists_Expired(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value with very short TTL
	setResult := <-store.Set(ctx, key, value, 10*time.Millisecond)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Check existence (should be expired)
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if existsResult.Exists {
		t.Errorf("Exists() should return false for expired key")
	}
}

func TestMemoryStore_TTL_Success(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")
	ttl := 1 * time.Minute

	// Set value
	setResult := <-store.Set(ctx, key, value, ttl)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Get TTL
	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 || ttlResult.TTL > ttl {
		t.Errorf("TTL() returned unexpected value: got %v, expected <= %v and > 0", ttlResult.TTL, ttl)
	}
}

func TestMemoryStore_TTL_NotFound(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ttlResult := <-store.TTL(ctx, "non-existent-key")
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}
	if ttlResult.TTL != 0 {
		t.Errorf("TTL() should return 0 for non-existent key, got %v", ttlResult.TTL)
	}
}

func TestMemoryStore_TTL_Expired(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test value")

	// Set value with very short TTL
	setResult := <-store.Set(ctx, key, value, 10*time.Millisecond)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Get TTL (should be expired)
	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}
	if ttlResult.TTL != 0 {
		t.Errorf("TTL() should return 0 for expired key, got %v", ttlResult.TTL)
	}
}

func TestMemoryStore_IncrBy_NewKey(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"
	delta := int64(5)

	incrResult := <-store.IncrBy(ctx, key, delta, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() failed: %v", incrResult.Error)
	}
	if incrResult.Result != delta {
		t.Errorf("IncrBy() returned wrong value: got %d, want %d", incrResult.Result, delta)
	}

	// Verify the value was stored
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the incremented value")
	}
	if string(getResult.Value) != "5" {
		t.Errorf("Get() returned wrong value: got %s, want 5", string(getResult.Value))
	}
}

func TestMemoryStore_IncrBy_ExistingKey(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"

	// Set initial value
	setResult := <-store.Set(ctx, key, []byte("10"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Increment
	incrResult := <-store.IncrBy(ctx, key, 5, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() failed: %v", incrResult.Error)
	}
	if incrResult.Result != 15 {
		t.Errorf("IncrBy() returned wrong value: got %d, want 15", incrResult.Result)
	}

	// Verify the value was updated
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the incremented value")
	}
	if string(getResult.Value) != "15" {
		t.Errorf("Get() returned wrong value: got %s, want 15", string(getResult.Value))
	}
}

func TestMemoryStore_IncrBy_ExpiredKey(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"

	// Set value with very short TTL
	setResult := <-store.Set(ctx, key, []byte("10"), 10*time.Millisecond)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Try to increment (should create new value since key is expired)
	incrResult := <-store.IncrBy(ctx, key, 5, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() should not fail for expired key: %v", incrResult.Error)
	}
	if incrResult.Result != 5 {
		t.Errorf("IncrBy() should create new value for expired key: got %d, want 5", incrResult.Result)
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Calling Close again should not fail
	err = store.Close()
	if err != nil {
		t.Errorf("Second Close() should not fail: %v", err)
	}
}

func TestMemoryStore_GetStats(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Initial stats
	stats := store.GetStats()
	if stats == nil {
		t.Errorf("GetStats() should not return nil")
	}

	// Set some values and perform operations
	setResult := <-store.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	getResult1 := <-store.Get(ctx, "key1") // Hit
	if getResult1.Error != nil {
		t.Errorf("Get() failed: %v", getResult1.Error)
	}

	getResult2 := <-store.Get(ctx, "non-existent") // Miss
	if getResult2.Error != nil {
		t.Errorf("Get() failed: %v", getResult2.Error)
	}

	// Check stats
	stats = store.GetStats()
	if stats.Hits == 0 {
		t.Errorf("Expected hits to be recorded")
	}
	if stats.Misses == 0 {
		t.Errorf("Expected misses to be recorded")
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set some values
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := []byte(fmt.Sprintf("value%d", i))
		setResult := <-store.Set(ctx, key, value, 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Clear the store
	store.Clear()

	// Verify all keys are gone
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		}
		if getResult.Exists {
			t.Errorf("Get() should return not found after Clear() for key %s", key)
		}
	}

	// Check stats are reset
	stats := store.GetStats()
	if stats.Size != 0 {
		t.Errorf("Expected size to be 0 after Clear(), got %d", stats.Size)
	}
	if stats.MemoryUsage != 0 {
		t.Errorf("Expected memory usage to be 0 after Clear(), got %d", stats.MemoryUsage)
	}
}

func TestMemoryStore_EvictionLRU(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         3, // Small capacity to trigger eviction
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Fill up to capacity
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("key%d", i)
		value := []byte(fmt.Sprintf("value%d", i))
		setResult := <-store.Set(ctx, key, value, 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Access key2 to make it more recently used than key1
	getResult := <-store.Get(ctx, "key2")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}

	// Add another key, should evict key1 (least recently used)
	setResult := <-store.Set(ctx, "key4", []byte("value4"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// key1 should be evicted
	result := <-store.Get(ctx, "key1")
	if result.Error != nil {
		t.Errorf("Get() failed: %v", result.Error)
	}
	if result.Exists {
		t.Errorf("key1 should have been evicted")
	}

	// key2, key3, key4 should still exist
	for _, key := range []string{"key2", "key3", "key4"} {
		result := <-store.Get(ctx, key)
		if result.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, result.Error)
		}
		if !result.Exists {
			t.Errorf("key %s should still exist", key)
		}
	}
}

func TestMemoryStore_EvictionLFU(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         3, // Small capacity to trigger eviction
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLFU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Fill up to capacity
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("key%d", i)
		value := []byte(fmt.Sprintf("value%d", i))
		setResult := <-store.Set(ctx, key, value, 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Access key2 and key3 multiple times
	for i := 0; i < 3; i++ {
		getResult := <-store.Get(ctx, "key2")
		if getResult.Error != nil {
			t.Errorf("Get() failed: %v", getResult.Error)
		}
		getResult = <-store.Get(ctx, "key3")
		if getResult.Error != nil {
			t.Errorf("Get() failed: %v", getResult.Error)
		}
	}

	// Add another key, should evict key1 (least frequently used)
	setResult := <-store.Set(ctx, "key4", []byte("value4"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// key1 should be evicted (it only has 1 access from Set)
	result := <-store.Get(ctx, "key1")
	if result.Error != nil {
		t.Errorf("Get() failed: %v", result.Error)
	}
	if result.Exists {
		t.Errorf("key1 should have been evicted (LFU)")
	}
}

func TestMemoryStore_EvictionTTL(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         3, // Small capacity to trigger eviction
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyTTL,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set keys with different TTLs
	setResult := <-store.Set(ctx, "key1", []byte("value1"), 1*time.Minute) // Shortest TTL
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	setResult = <-store.Set(ctx, "key2", []byte("value2"), 3*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	setResult = <-store.Set(ctx, "key3", []byte("value3"), 5*time.Minute) // Longest TTL
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Add another key, should evict key1 (shortest TTL)
	setResult = <-store.Set(ctx, "key4", []byte("value4"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// key1 should be evicted
	result := <-store.Get(ctx, "key1")
	if result.Error != nil {
		t.Errorf("Get() failed: %v", result.Error)
	}
	if result.Exists {
		t.Errorf("key1 should have been evicted (TTL)")
	}
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent read/write operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))

				// Set
				setResult := <-store.Set(ctx, key, value, 5*time.Minute)
				if setResult.Error != nil {
					t.Errorf("Concurrent Set() failed: %v", setResult.Error)
					return
				}

				// Get
				getResult := <-store.Get(ctx, key)
				if getResult.Error != nil {
					t.Errorf("Concurrent Get() failed: %v", getResult.Error)
					return
				}
				if !getResult.Exists || string(getResult.Value) != string(value) {
					t.Errorf("Concurrent Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
					return
				}

				// Delete some keys
				if j%2 == 0 {
					delResult := <-store.Del(ctx, key)
					if delResult.Error != nil {
						t.Errorf("Concurrent Del() failed: %v", delResult.Error)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestMemoryStore_CleanupExpiredItems(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 100 * time.Millisecond, // Fast cleanup for testing
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Set items with short TTL
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := []byte(fmt.Sprintf("value%d", i))
		setResult := <-store.Set(ctx, key, value, 50*time.Millisecond)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// All items should be expired and cleaned up
	stats := store.GetStats()
	if stats.Expirations == 0 {
		t.Errorf("Expected expirations to be recorded")
	}

	// Verify items are gone
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		result := <-store.Get(ctx, key)
		if result.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, result.Error)
		}
		if result.Exists {
			t.Errorf("Expired item %s should have been cleaned up", key)
		}
	}
}

func TestMemoryStore_StatsDisabled(t *testing.T) {
	config := &cachex.MemoryConfig{
		MaxSize:         100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     false, // Disable stats
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Perform operations
	setResult := <-store.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	getResult := <-store.Get(ctx, "key1") // Hit
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}

	getResult = <-store.Get(ctx, "non-existent") // Miss
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}

	// Stats should not be recorded when disabled
	stats := store.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Stats should not be recorded when disabled: hits=%d, misses=%d", stats.Hits, stats.Misses)
	}
}

func TestMemoryStore_EdgeCases(t *testing.T) {
	store, err := cachex.NewMemoryStore(nil)
	if err != nil {
		t.Fatalf("NewMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		key         string
		value       []byte
		expectError bool
	}{
		{
			name:        "empty key",
			key:         "",
			value:       []byte("value"),
			expectError: true, // Now rejected due to validation
		},
		{
			name:        "empty value",
			key:         "key",
			value:       []byte{},
			expectError: false, // Empty slice is allowed
		},
		{
			name:        "nil value",
			key:         "key",
			value:       nil,
			expectError: true, // Now rejected due to validation
		},
		{
			name:        "very long key",
			key:         string(make([]byte, 1000)),
			value:       []byte("value"),
			expectError: false, // Long keys are allowed (no length limit in store layer)
		},
		{
			name:        "very long value",
			key:         "key",
			value:       make([]byte, 10000),
			expectError: false, // Long values are allowed (no size limit in store layer)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setResult := <-store.Set(ctx, tt.key, tt.value, 5*time.Minute)
			if tt.expectError && setResult.Error == nil {
				t.Errorf("Set() should fail for %s", tt.name)
			} else if !tt.expectError && setResult.Error != nil {
				t.Errorf("Set() should not fail for %s: %v", tt.name, setResult.Error)
			}

			if !tt.expectError {
				getResult := <-store.Get(ctx, tt.key)
				if getResult.Error != nil {
					t.Errorf("Get() failed for %s: %v", tt.name, getResult.Error)
				}
				if !getResult.Exists || string(getResult.Value) != string(tt.value) {
					t.Errorf("Get() returned wrong value for %s: got %v, want %v", tt.name, string(getResult.Value), string(tt.value))
				}
			}
		})
	}
}
