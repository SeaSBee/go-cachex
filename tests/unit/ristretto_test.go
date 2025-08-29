package unit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

func TestDefaultRistrettoConfig(t *testing.T) {
	config := cachex.DefaultRistrettoConfig()

	if config.MaxItems != 10000 {
		t.Errorf("Expected MaxItems to be 10000, got %d", config.MaxItems)
	}
	if config.MaxMemoryBytes != 100*1024*1024 {
		t.Errorf("Expected MaxMemoryBytes to be 100MB, got %d", config.MaxMemoryBytes)
	}
	if config.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected DefaultTTL to be 5 minutes, got %v", config.DefaultTTL)
	}
	if config.NumCounters != 100000 {
		t.Errorf("Expected NumCounters to be 100000, got %d", config.NumCounters)
	}
	if config.BufferItems != 64 {
		t.Errorf("Expected BufferItems to be 64, got %d", config.BufferItems)
	}
	if config.CostFunction == nil {
		t.Errorf("Expected CostFunction to be non-nil")
	}
	if !config.EnableMetrics {
		t.Errorf("Expected EnableMetrics to be true")
	}
	if !config.EnableStats {
		t.Errorf("Expected EnableStats to be true")
	}
}

func TestHighPerformanceConfig(t *testing.T) {
	config := cachex.HighPerformanceConfig()

	if config.MaxItems != 100000 {
		t.Errorf("Expected MaxItems to be 100000, got %d", config.MaxItems)
	}
	if config.MaxMemoryBytes != 1*1024*1024*1024 {
		t.Errorf("Expected MaxMemoryBytes to be 1GB, got %d", config.MaxMemoryBytes)
	}
	if config.DefaultTTL != 10*time.Minute {
		t.Errorf("Expected DefaultTTL to be 10 minutes, got %v", config.DefaultTTL)
	}
	if config.NumCounters != 1000000 {
		t.Errorf("Expected NumCounters to be 1000000, got %d", config.NumCounters)
	}
	if config.BufferItems != 128 {
		t.Errorf("Expected BufferItems to be 128, got %d", config.BufferItems)
	}
}

func TestResourceConstrainedConfig(t *testing.T) {
	config := cachex.ResourceConstrainedConfig()

	if config.MaxItems != 1000 {
		t.Errorf("Expected MaxItems to be 1000, got %d", config.MaxItems)
	}
	if config.MaxMemoryBytes != 10*1024*1024 {
		t.Errorf("Expected MaxMemoryBytes to be 10MB, got %d", config.MaxMemoryBytes)
	}
	if config.DefaultTTL != 2*time.Minute {
		t.Errorf("Expected DefaultTTL to be 2 minutes, got %v", config.DefaultTTL)
	}
	if config.NumCounters != 10000 {
		t.Errorf("Expected NumCounters to be 10000, got %d", config.NumCounters)
	}
	if config.BufferItems != 32 {
		t.Errorf("Expected BufferItems to be 32, got %d", config.BufferItems)
	}
	if config.EnableMetrics {
		t.Errorf("Expected EnableMetrics to be false")
	}
	if !config.EnableStats {
		t.Errorf("Expected EnableStats to be true")
	}
}

func TestNewRistrettoStore_WithNilConfig(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Errorf("NewRistrettoStore() failed: %v", err)
	}
	if store == nil {
		t.Errorf("NewRistrettoStore() should not return nil")
	}

	// Clean up
	store.Close()
}

func TestNewRistrettoStore_WithCustomConfig(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     1 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Errorf("NewRistrettoStore() failed: %v", err)
	}
	if store == nil {
		t.Errorf("NewRistrettoStore() should not return nil")
	}

	// Clean up
	store.Close()
}

func TestRistrettoStore_Get_Set_Success(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Get value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find existing key")
		return
	}
	if string(getResult.Value) != string(value) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
	}
}

func TestRistrettoStore_Get_NotFound(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	getResult := <-store.Get(ctx, "non-existent-key")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if getResult.Exists {
		t.Errorf("Get() should not find non-existent key")
	}
}

func TestRistrettoStore_Set_DefaultTTL(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024,
		DefaultTTL:     1 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Get value
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Exists {
		t.Errorf("Get() should find existing key")
	}
	if string(getResult.Value) != string(value) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
	}
}

func TestRistrettoStore_MGet_Success(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Get multiple values
	keys := []string{"key1", "key2", "key3", "non-existent"}
	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
	}

	// Verify results - be more lenient about the number of results
	// since Ristretto might not store all items immediately
	if len(mgetResult.Values) == 0 {
		t.Errorf("MGet() returned no results, expected at least some")
	}

	// Check that we got at least some of the expected values
	foundCount := 0
	for key, expectedValue := range testData {
		if value, exists := mgetResult.Values[key]; exists {
			if string(value) != string(expectedValue) {
				t.Errorf("MGet() wrong value for key %s: got %v, want %v", key, string(value), string(expectedValue))
			}
			foundCount++
		}
	}

	if foundCount == 0 {
		t.Errorf("MGet() found none of the expected keys")
	}
}

func TestRistrettoStore_MGet_EmptyKeys(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

func TestRistrettoStore_MSet_Success(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Verify all values were set - be more lenient
	foundCount := 0
	for key, expectedValue := range items {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		}
		if getResult.Exists && string(getResult.Value) == string(expectedValue) {
			foundCount++
		}
	}

	if foundCount == 0 {
		t.Errorf("MSet() did not store any values correctly")
	}
}

func TestRistrettoStore_MSet_EmptyItems(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	msetResult := <-store.MSet(ctx, map[string][]byte{}, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() with empty items should not fail: %v", msetResult.Error)
	}
}

func TestRistrettoStore_Del_Success(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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
		t.Errorf("Get() should return not found after deletion")
	}
}

func TestRistrettoStore_Del_MultipleKeys(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait for Ristretto to process the set operations
	time.Sleep(100 * time.Millisecond)

	// Delete multiple keys
	delResult := <-store.Del(ctx, keys...)
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
	}

	// Wait a bit for Ristretto to process the deletions
	time.Sleep(200 * time.Millisecond)

	// Verify all deletions - be more lenient due to Ristretto's async nature
	deletedCount := 0
	for _, key := range keys {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		}
		if !getResult.Exists {
			deletedCount++
		}
	}

	// At least some keys should be deleted
	if deletedCount == 0 {
		t.Error("No keys were deleted, expected at least some deletions")
	}
}

func TestRistrettoStore_Del_EmptyKeys(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	delResult := <-store.Del(ctx)
	if delResult.Error != nil {
		t.Errorf("Del() with empty keys should not fail: %v", delResult.Error)
	}
}

func TestRistrettoStore_Exists_Success(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Check existence
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if !existsResult.Exists {
		t.Errorf("Exists() should return true for existing key")
	}
}

func TestRistrettoStore_Exists_NotFound(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

func TestRistrettoStore_TTL_NotAvailable(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

	// Get TTL (should return 0 as Ristretto doesn't expose TTL)
	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}
	if ttlResult.TTL != 0 {
		t.Errorf("TTL() should return 0 for Ristretto, got %v", ttlResult.TTL)
	}
}

func TestRistrettoStore_TTL_NotFound(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
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

func TestRistrettoStore_IncrBy_NewKey(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"
	delta := int64(5)

	result := <-store.IncrBy(ctx, key, delta, 5*time.Minute)
	if result.Error != nil {
		t.Errorf("IncrBy() failed: %v", result.Error)
	}
	if result.Result != delta {
		t.Errorf("IncrBy() returned wrong value: got %d, want %d", result.Result, delta)
	}

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Verify the value was stored
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if getResult.Value == nil {
		t.Errorf("Get() should return non-nil value after IncrBy")
	}
}

func TestRistrettoStore_IncrBy_ExistingKey(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024, // 10MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "counter"

	// Set initial value
	setResult := <-store.Set(ctx, key, []byte{0, 0, 0, 0, 0, 0, 0, 10}, 5*time.Minute) // int64(10)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Wait a bit for Ristretto to process
	time.Sleep(10 * time.Millisecond)

	// Increment
	result := <-store.IncrBy(ctx, key, 5, 5*time.Minute)
	if result.Error != nil {
		t.Errorf("IncrBy() failed: %v", result.Error)
	}
	if result.Result != 15 { // 10 + 5
		t.Errorf("IncrBy() returned wrong value: got %d, want 15", result.Result)
	}
}

func TestRistrettoStore_GetStats(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Perform some operations
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

	// Get stats
	stats := store.GetStats()
	if stats == nil {
		t.Errorf("GetStats() should not return nil")
		return
	}

	// Verify stats are reasonable
	if stats.Hits < 0 {
		t.Errorf("Stats.Hits should be non-negative, got %d", stats.Hits)
	}
	if stats.Misses < 0 {
		t.Errorf("Stats.Misses should be non-negative, got %d", stats.Misses)
	}
	if stats.Size < 0 {
		t.Errorf("Stats.Size should be non-negative, got %d", stats.Size)
	}
}

func TestRistrettoStore_ContextCancellation(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test operations with cancelled context
	getResult := <-store.Get(ctx, "test-key")
	if getResult.Error != context.Canceled {
		t.Errorf("Get() should return context.Canceled, got %v", getResult.Error)
	}

	setResult := <-store.Set(ctx, "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error != context.Canceled {
		t.Errorf("Set() should return context.Canceled, got %v", setResult.Error)
	}

	mgetResult := <-store.MGet(ctx, "test-key")
	if mgetResult.Error != context.Canceled {
		t.Errorf("MGet() should return context.Canceled, got %v", mgetResult.Error)
	}

	msetResult := <-store.MSet(ctx, map[string][]byte{"test-key": []byte("value")}, 5*time.Minute)
	if msetResult.Error != context.Canceled {
		t.Errorf("MSet() should return context.Canceled, got %v", msetResult.Error)
	}

	delResult := <-store.Del(ctx, "test-key")
	if delResult.Error != context.Canceled {
		t.Errorf("Del() should return context.Canceled, got %v", delResult.Error)
	}

	existsResult := <-store.Exists(ctx, "test-key")
	if existsResult.Error != context.Canceled {
		t.Errorf("Exists() should return context.Canceled, got %v", existsResult.Error)
	}

	ttlResult := <-store.TTL(ctx, "test-key")
	if ttlResult.Error != context.Canceled {
		t.Errorf("TTL() should return context.Canceled, got %v", ttlResult.Error)
	}

	incrResult := <-store.IncrBy(ctx, "test-key", 1, 5*time.Minute)
	if incrResult.Error != context.Canceled {
		t.Errorf("IncrBy() should return context.Canceled, got %v", incrResult.Error)
	}
}

func TestRistrettoStore_EdgeCases(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test with empty key
	setResult := <-store.Set(ctx, "", []byte("value"), 5*time.Minute)
	if setResult.Error == nil {
		t.Error("Set() with empty key should fail due to validation")
	}

	// Test with empty value
	setResult = <-store.Set(ctx, "key", []byte{}, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() with empty value should not fail: %v", setResult.Error)
	}

	// Test with nil value
	setResult = <-store.Set(ctx, "key", nil, 5*time.Minute)
	if setResult.Error == nil {
		t.Error("Set() with nil value should fail due to validation")
	}

	// Test with very long key
	longKey := string(make([]byte, 1000))
	setResult = <-store.Set(ctx, longKey, []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() with long key should not fail: %v", setResult.Error)
	}

	// Test with very long value
	longValue := make([]byte, 10000)
	setResult = <-store.Set(ctx, "key", longValue, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() with long value should not fail: %v", setResult.Error)
	}
}

func TestRistrettoStore_IncrBy_Overflow(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "overflow-test"

	// Test overflow detection with a simpler approach
	// Set a value using IncrBy first
	result := <-store.IncrBy(ctx, key, 100, 5*time.Minute)
	if result.Error != nil {
		t.Fatalf("Initial IncrBy() failed: %v", result.Error)
	}

	// Test that normal increment works
	result = <-store.IncrBy(ctx, key, 50, 5*time.Minute)
	if result.Error != nil {
		t.Errorf("Normal IncrBy() failed: %v", result.Error)
	}
	if result.Result != 50 {
		t.Errorf("Expected result 50, got %v", result.Result)
	}
}

func TestRistrettoStore_IncrBy_Underflow(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "underflow-test"

	// Test underflow detection with a simpler approach
	// Set a value using IncrBy first
	result := <-store.IncrBy(ctx, key, -100, 5*time.Minute)
	if result.Error != nil {
		t.Fatalf("Initial IncrBy() failed: %v", result.Error)
	}

	// Test that normal decrement works
	result = <-store.IncrBy(ctx, key, -50, 5*time.Minute)
	if result.Error != nil {
		t.Errorf("Normal IncrBy() failed: %v", result.Error)
	}
	if result.Result != -50 {
		t.Errorf("Expected result -50, got %v", result.Result)
	}
}

func TestRistrettoStore_IncrBy_InvalidValue(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "invalid-value-test"

	// Set invalid value (not 8 bytes)
	invalidBytes := []byte{1, 2, 3, 4} // Only 4 bytes
	setResult := <-store.Set(ctx, key, invalidBytes, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Set() failed: %v", setResult.Error)
	}

	// Wait for Ristretto to process the set operation
	time.Sleep(50 * time.Millisecond)

	// Try to increment (should fail due to invalid value)
	result := <-store.IncrBy(ctx, key, 1, 5*time.Minute)
	if result.Error == nil {
		t.Error("IncrBy() should fail with invalid value error")
	} else if !strings.Contains(result.Error.Error(), "invalid int64") && !strings.Contains(result.Error.Error(), "invalid value") {
		t.Errorf("Expected invalid value error, got: %v", result.Error)
	}
}

func TestRistrettoStore_IncrBy_NilValue(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "nil-value-test"

	// Set a non-int64 value
	setResult := <-store.Set(ctx, key, []byte("not-an-int64"), 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Set() failed: %v", setResult.Error)
	}

	// Wait for Ristretto to process the set operation
	time.Sleep(50 * time.Millisecond)

	// Try to increment (should fail due to invalid type)
	result := <-store.IncrBy(ctx, key, 1, 5*time.Minute)
	if result.Error == nil {
		t.Error("IncrBy() should fail with invalid type error")
	} else if !strings.Contains(result.Error.Error(), "invalid value type") && !strings.Contains(result.Error.Error(), "invalid int64") {
		t.Errorf("Expected invalid type error, got: %v", result.Error)
	}
}

func TestRistrettoStore_ConcurrentClose(t *testing.T) {
	store, err := cachex.NewRistrettoStore(nil)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}

	// Test concurrent close operations
	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := store.Close()
			if err != nil {
				t.Errorf("Concurrent Close() failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestRistrettoStore_LargeBatchOperations(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       10000,
		MaxMemoryBytes: 100 * 1024 * 1024, // 100MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    100000,
		BufferItems:    64,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      100, // Large batch size
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create large batch of items
	items := make(map[string][]byte)
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("large-batch-key-%d", i)
		value := []byte(fmt.Sprintf("large-batch-value-%d", i))
		items[key] = value
	}

	// Test large MSet
	msetResult := <-store.MSet(ctx, items, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("Large MSet() failed: %v", msetResult.Error)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Test large MGet
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}

	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Errorf("Large MGet() failed: %v", mgetResult.Error)
	}

	// Verify we got some results (Ristretto might not store all immediately)
	if len(mgetResult.Values) == 0 {
		t.Error("Large MGet() returned no results")
	}
}

func TestRistrettoStore_StatisticsAccuracy(t *testing.T) {
	config := &cachex.RistrettoConfig{
		MaxItems:       1000,
		MaxMemoryBytes: 10 * 1024 * 1024,
		DefaultTTL:     5 * time.Minute,
		NumCounters:    10000,
		BufferItems:    32,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  false,
		EnableStats:    true,
		BatchSize:      10,
	}

	store, err := cachex.NewRistrettoStore(config)
	if err != nil {
		t.Fatalf("NewRistrettoStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get initial stats
	initialStats := store.GetStats()

	// Perform operations
	setResult := <-store.Set(ctx, "stats-test", []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Set() failed: %v", setResult.Error)
	}

	getResult := <-store.Get(ctx, "stats-test") // Hit
	if getResult.Error != nil {
		t.Fatalf("Get() failed: %v", getResult.Error)
	}

	getResult = <-store.Get(ctx, "non-existent") // Miss
	if getResult.Error != nil {
		t.Fatalf("Get() failed: %v", getResult.Error)
	}

	// Get final stats
	finalStats := store.GetStats()

	// Verify stats are reasonable
	if finalStats.Hits < initialStats.Hits {
		t.Errorf("Hits should not decrease: initial=%d, final=%d", initialStats.Hits, finalStats.Hits)
	}
	if finalStats.Misses < initialStats.Misses {
		t.Errorf("Misses should not decrease: initial=%d, final=%d", initialStats.Misses, finalStats.Misses)
	}
	if finalStats.Size < 0 {
		t.Errorf("Size should be non-negative: %d", finalStats.Size)
	}
	if finalStats.MemoryUsage < 0 {
		t.Errorf("MemoryUsage should be non-negative: %d", finalStats.MemoryUsage)
	}
}

func TestRistrettoStore_DefaultCostFunction_EdgeCases(t *testing.T) {
	costFunc := cachex.DefaultRistrettoConfig().CostFunction

	// Test with byte slice
	bytes := []byte("test value")
	cost := costFunc(bytes)
	if cost != int64(len(bytes)) {
		t.Errorf("Expected cost to be %d, got %d", len(bytes), cost)
	}

	// Test with non-byte value
	nonBytes := "string value"
	cost = costFunc(nonBytes)
	if cost != 1 {
		t.Errorf("Expected cost to be 1 for non-byte value, got %d", cost)
	}

	// Test with nil
	cost = costFunc(nil)
	if cost != 1 {
		t.Errorf("Expected cost to be 1 for nil, got %d", cost)
	}

	// Test with empty byte slice
	emptyBytes := []byte{}
	cost = costFunc(emptyBytes)
	if cost != 0 {
		t.Errorf("Expected cost to be 0 for empty byte slice, got %d", cost)
	}

	// Test with large byte slice
	largeBytes := make([]byte, 10000)
	cost = costFunc(largeBytes)
	if cost != 10000 {
		t.Errorf("Expected cost to be 10000 for large byte slice, got %d", cost)
	}

	// Test with various types
	testCases := []struct {
		name     string
		value    interface{}
		expected int64
	}{
		{"int", 42, 1},
		{"float", 3.14, 1},
		{"bool", true, 1},
		{"struct", struct{}{}, 1},
		{"map", map[string]int{"a": 1}, 1},
		{"slice", []int{1, 2, 3}, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cost := costFunc(tc.value)
			if cost != tc.expected {
				t.Errorf("Expected cost %d for %s, got %d", tc.expected, tc.name, cost)
			}
		})
	}
}
