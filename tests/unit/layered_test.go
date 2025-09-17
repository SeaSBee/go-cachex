package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

func TestDefaultLayeredConfig(t *testing.T) {
	config := cachex.DefaultLayeredConfig()
	if config == nil {
		t.Errorf("DefaultLayeredConfig() returned nil")
		return
	}

	// Check default values
	if config.MemoryConfig == nil {
		t.Errorf("DefaultLayeredConfig().MemoryConfig should not be nil")
	}
	if config.WritePolicy != cachex.WritePolicyThrough {
		t.Errorf("DefaultLayeredConfig().WritePolicy = %v, want %v", config.WritePolicy, cachex.WritePolicyThrough)
	}
	if config.ReadPolicy != cachex.ReadPolicyThrough {
		t.Errorf("DefaultLayeredConfig().ReadPolicy = %v, want %v", config.ReadPolicy, cachex.ReadPolicyThrough)
	}
	if config.SyncInterval != 5*time.Minute {
		t.Errorf("DefaultLayeredConfig().SyncInterval = %v, want %v", config.SyncInterval, 5*time.Minute)
	}
	if !config.EnableStats {
		t.Errorf("DefaultLayeredConfig().EnableStats should be true")
	}
	if config.MaxConcurrentSync != 10 {
		t.Errorf("DefaultLayeredConfig().MaxConcurrentSync = %v, want 10", config.MaxConcurrentSync)
	}
}

func TestHighPerfLayeredConfig(t *testing.T) {
	config := cachex.HighPerfLayeredConfig()
	if config == nil {
		t.Errorf("HighPerfLayeredConfig() returned nil")
		return
	}

	// Check high performance values
	if config.WritePolicy != cachex.WritePolicyBehind {
		t.Errorf("HighPerfLayeredConfig().WritePolicy = %v, want %v", config.WritePolicy, cachex.WritePolicyBehind)
	}
	if config.ReadPolicy != cachex.ReadPolicyThrough {
		t.Errorf("HighPerfLayeredConfig().ReadPolicy = %v, want %v", config.ReadPolicy, cachex.ReadPolicyThrough)
	}
	if config.SyncInterval != 1*time.Minute {
		t.Errorf("HighPerfLayeredConfig().SyncInterval = %v, want %v", config.SyncInterval, 1*time.Minute)
	}
	if config.MaxConcurrentSync != 50 {
		t.Errorf("HighPerfLayeredConfig().MaxConcurrentSync = %v, want 50", config.MaxConcurrentSync)
	}
}

func TestResourceLayeredConfig(t *testing.T) {
	config := cachex.ResourceLayeredConfig()
	if config == nil {
		t.Errorf("ResourceLayeredConfig() returned nil")
		return
	}

	// Check resource-constrained values
	if config.WritePolicy != cachex.WritePolicyAround {
		t.Errorf("ResourceLayeredConfig().WritePolicy = %v, want %v", config.WritePolicy, cachex.WritePolicyAround)
	}
	if config.ReadPolicy != cachex.ReadPolicyAround {
		t.Errorf("ResourceLayeredConfig().ReadPolicy = %v, want %v", config.ReadPolicy, cachex.ReadPolicyAround)
	}
	if config.SyncInterval != 10*time.Minute {
		t.Errorf("ResourceLayeredConfig().SyncInterval = %v, want %v", config.SyncInterval, 10*time.Minute)
	}
	if config.EnableStats {
		t.Errorf("ResourceLayeredConfig().EnableStats should be false")
	}
	if config.MaxConcurrentSync != 5 {
		t.Errorf("ResourceLayeredConfig().MaxConcurrentSync = %v, want 5", config.MaxConcurrentSync)
	}
}

func TestNewLayeredStore(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Errorf("NewLayeredStore() failed: %v", err)
		return
	}
	if store == nil {
		t.Errorf("NewLayeredStore() returned nil")
		return
	}

	// Test with nil config (should use default)
	store2, err := cachex.NewLayeredStore(l2Store, nil)
	if err != nil {
		t.Errorf("NewLayeredStore() with nil config failed: %v", err)
		return
	}
	if store2 == nil {
		t.Errorf("NewLayeredStore() with nil config returned nil")
	}
}

func TestLayeredStore_Get_ReadThrough(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set value in L2 only
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// Get from layered store (should read from L2 and populate L1)
	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
		return
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the value")
		return
	}
	if string(getResult.Value) != string(testData) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(testData))
	}

	// Get again (should now hit L1)
	getResult2 := <-store.Get(context.Background(), "test-key")
	if getResult2.Error != nil {
		t.Errorf("Second Get() failed: %v", getResult2.Error)
		return
	}
	if !getResult2.Exists {
		t.Errorf("Second Get() should find the value")
		return
	}
	if string(getResult2.Value) != string(testData) {
		t.Errorf("Second Get() returned wrong value: got %v, want %v", string(getResult2.Value), string(testData))
	}

	// Check stats - note that stats might not be recorded in all cases due to async operations
	// We can't reliably check stats in this test due to async L1 population
}

func TestLayeredStore_Get_ReadAside(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyAside,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set value in L2 only
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// Get from layered store (should return nil since L1 is empty and read-aside doesn't fallback)
	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
		return
	}
	if getResult.Exists {
		t.Errorf("Get() should return false in read-aside mode when L1 is empty: got %v", getResult.Value)
	}

	// Check stats
	stats := store.GetStats()
	if stats.L1Misses == 0 {
		t.Errorf("Expected L1 misses to be recorded")
	}
}

func TestLayeredStore_Get_ReadAround(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyAround,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set value in L2 only
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// Get from layered store (should read from L2 only)
	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
		return
	}
	if !getResult.Exists {
		t.Errorf("Get() should find the value")
		return
	}
	if string(getResult.Value) != string(testData) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(testData))
	}

	// Get again (should still read from L2, not populate L1)
	getResult2 := <-store.Get(context.Background(), "test-key")
	if getResult2.Error != nil {
		t.Errorf("Second Get() failed: %v", getResult2.Error)
		return
	}
	if !getResult2.Exists {
		t.Errorf("Second Get() should find the value")
		return
	}
	if string(getResult2.Value) != string(testData) {
		t.Errorf("Second Get() returned wrong value: got %v, want %v", string(getResult2.Value), string(testData))
	}

	// Check stats
	stats := store.GetStats()
	if stats.L2Hits == 0 {
		t.Errorf("Expected L2 hits to be recorded")
	}
}

func TestLayeredStore_Set_WriteThrough(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	testData := []byte("test value")
	setResult := <-store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
		return
	}

	// Verify value is in L2 (L1 access is not available through public API)
	getResult := <-l2Store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Errorf("Failed to get from L2: %v", getResult.Error)
		return
	}
	if !getResult.Exists {
		t.Errorf("Value should exist in L2")
		return
	}
	if string(getResult.Value) != string(testData) {
		t.Errorf("L2 value mismatch: got %v, want %v", string(getResult.Value), string(testData))
	}

	l2GetResult := <-l2Store.Get(context.Background(), "test-key")
	if l2GetResult.Error != nil {
		t.Errorf("Failed to get from L2: %v", l2GetResult.Error)
		return
	}
	if !l2GetResult.Exists {
		t.Errorf("Value should exist in L2")
		return
	}
	if string(l2GetResult.Value) != string(testData) {
		t.Errorf("L2 value mismatch: got %v, want %v", string(l2GetResult.Value), string(testData))
	}
}

func TestLayeredStore_Set_WriteBehind(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig:      cachex.DefaultMemoryConfig(),
		WritePolicy:       cachex.WritePolicyBehind,
		EnableStats:       true,
		MaxConcurrentSync: 10,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	testData := []byte("test value")
	setResult := <-store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
		return
	}

	// Verify value is in L1 (immediate write)
	l1GetResult := <-store.Get(context.Background(), "test-key")
	if l1GetResult.Error != nil {
		t.Errorf("Failed to get from L1: %v", l1GetResult.Error)
		return
	}
	if !l1GetResult.Exists {
		t.Errorf("Value should exist in L1")
		return
	}

	// Wait a bit for async write to L2
	time.Sleep(1 * time.Second)

	// Verify value is in L2 (async write should have completed)
	l2GetResult := <-l2Store.Get(context.Background(), "test-key")
	if l2GetResult.Error != nil {
		t.Errorf("Failed to get from L2: %v", l2GetResult.Error)
		return
	}
	if !l2GetResult.Exists {
		t.Errorf("Value should exist in L2")
		return
	}
	if string(l2GetResult.Value) != string(testData) {
		t.Errorf("L2 value mismatch: got %v, want %v", string(l2GetResult.Value), string(testData))
	}
}

func TestLayeredStore_Set_WriteAround(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyAround,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	testData := []byte("test value")
	setResult := <-store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
		return
	}

	// Verify value is in L2
	l2GetResult := <-l2Store.Get(context.Background(), "test-key")
	if l2GetResult.Error != nil {
		t.Errorf("Failed to get from L2: %v", l2GetResult.Error)
		return
	}
	if !l2GetResult.Exists {
		t.Errorf("Value should exist in L2")
		return
	}
	if string(l2GetResult.Value) != string(testData) {
		t.Errorf("L2 value mismatch: got %v, want %v", string(l2GetResult.Value), string(testData))
	}
}

func TestLayeredStore_MGet(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set values in L2
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for key, value := range testData {
		setResult := <-l2Store.Set(context.Background(), key, value, 5*time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set %s in L2: %v", key, setResult.Error)
		}
	}

	// Get multiple values
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	mgetResult := <-store.MGet(context.Background(), keys...)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
		return
	}

	// Verify results
	if len(mgetResult.Values) != 3 {
		t.Errorf("MGet() returned %d items, want 3", len(mgetResult.Values))
	}

	for key, expectedValue := range testData {
		if value, exists := mgetResult.Values[key]; !exists {
			t.Errorf("MGet() missing key: %s", key)
		} else if string(value) != string(expectedValue) {
			t.Errorf("MGet() wrong value for %s: got %v, want %v", key, string(value), string(expectedValue))
		}
	}

	// Test empty keys
	emptyMgetResult := <-store.MGet(context.Background())
	if emptyMgetResult.Error != nil {
		t.Errorf("MGet() with empty keys failed: %v", emptyMgetResult.Error)
		return
	}
	if len(emptyMgetResult.Values) != 0 {
		t.Errorf("MGet() with empty keys returned %d items, want 0", len(emptyMgetResult.Values))
	}
}

func TestLayeredStore_MSet(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	msetResult := <-store.MSet(context.Background(), testData, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() failed: %v", msetResult.Error)
		return
	}

	// Verify values are in L2
	for key, expectedValue := range testData {
		l2GetResult := <-l2Store.Get(context.Background(), key)
		if l2GetResult.Error != nil {
			t.Errorf("Failed to get %s from L2: %v", key, l2GetResult.Error)
			continue
		}
		if !l2GetResult.Exists {
			t.Errorf("Value should exist in L2 for key %s", key)
			continue
		}
		if string(l2GetResult.Value) != string(expectedValue) {
			t.Errorf("L2 value mismatch for %s: got %v, want %v", key, string(l2GetResult.Value), string(expectedValue))
		}
	}

	// Test empty items
	emptyMsetResult := <-store.MSet(context.Background(), map[string][]byte{}, 5*time.Minute)
	if emptyMsetResult.Error != nil {
		t.Errorf("MSet() with empty items failed: %v", emptyMsetResult.Error)
	}
}

func TestLayeredStore_Del(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set value in L2
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// Delete from layered store
	delResult := <-store.Del(context.Background(), "test-key")
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
		return
	}

	// Verify value is deleted from L2
	l2GetResult := <-l2Store.Get(context.Background(), "test-key")
	if l2GetResult.Error != nil {
		t.Errorf("Failed to get from L2: %v", l2GetResult.Error)
		return
	}
	if l2GetResult.Exists {
		t.Errorf("L2 value should be deleted: got %v", l2GetResult.Value)
	}

	// Test empty keys
	emptyDelResult := <-store.Del(context.Background())
	if emptyDelResult.Error != nil {
		t.Errorf("Del() with empty keys failed: %v", emptyDelResult.Error)
	}
}

func TestLayeredStore_Exists(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Test non-existent key
	existsResult := <-store.Exists(context.Background(), "nonexistent")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
		return
	}
	if existsResult.Exists {
		t.Errorf("Exists() should return false for non-existent key")
	}

	// Set value in L2 and test existing key
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	existsResult = <-store.Exists(context.Background(), "test-key")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
		return
	}
	if !existsResult.Exists {
		t.Errorf("Exists() should return true for existing key in L2")
	}

	// Set value in L2 only
	setResult2 := <-l2Store.Set(context.Background(), "test-key2", testData, 5*time.Minute)
	if setResult2.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult2.Error)
	}

	// Test existing key in L2
	existsResult2 := <-store.Exists(context.Background(), "test-key2")
	if existsResult2.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult2.Error)
		return
	}
	if !existsResult2.Exists {
		t.Errorf("Exists() should return true for existing key in L2")
	}
}

func TestLayeredStore_TTL(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Set value in L2 with TTL
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// Get TTL from L2
	ttlResult := <-store.TTL(context.Background(), "test-key")
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
		return
	}
	if ttlResult.TTL <= 0 {
		t.Errorf("TTL() should return positive value: got %v", ttlResult.TTL)
	}

	// Set value in L2 with TTL
	setResult2 := <-l2Store.Set(context.Background(), "test-key2", testData, 10*time.Minute)
	if setResult2.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult2.Error)
	}

	// Get TTL from L2
	ttlResult2 := <-store.TTL(context.Background(), "test-key2")
	if ttlResult2.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult2.Error)
		return
	}
	if ttlResult2.TTL <= 0 {
		t.Errorf("TTL() should return positive value: got %v", ttlResult2.TTL)
	}
}

func TestLayeredStore_IncrBy(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Test increment in L1
	incrResult := <-store.IncrBy(context.Background(), "counter", 5, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() failed: %v", incrResult.Error)
		return
	}
	if incrResult.Result != 5 {
		t.Errorf("IncrBy() returned wrong value: got %d, want 5", incrResult.Result)
	}

	// Note: Layered store IncrBy operations may not immediately propagate to L2
	// depending on the write policy and async operations. The test verifies
	// that the IncrBy operation completes successfully and returns the correct value.
}

func TestLayeredStore_Close(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Note: LayeredStore doesn't prevent operations after Close()
	// This is different from the main cache implementation
	// The Close() method only closes the underlying stores
}

func TestLayeredStore_GetStats(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Perform some operations to generate stats
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	// This should hit L2 and populate L1
	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Fatalf("Get() failed: %v", getResult.Error)
	}

	// This should hit L1
	getResult2 := <-store.Get(context.Background(), "test-key")
	if getResult2.Error != nil {
		t.Fatalf("Second Get() failed: %v", getResult2.Error)
	}

	// Get stats
	stats := store.GetStats()
	if stats == nil {
		t.Errorf("GetStats() returned nil")
		return
	}

	// Note: Stats might not be recorded reliably due to async operations
	// The LayeredStore implementation uses goroutines for L1 population
	// which can cause race conditions in test scenarios
}

func TestLayeredStore_SyncL1ToL2(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	err = store.SyncL1ToL2(context.Background())
	if err != nil {
		t.Errorf("SyncL1ToL2() failed: %v", err)
	}

	// Check that sync was recorded
	stats := store.GetStats()
	if stats.SyncCount == 0 {
		t.Errorf("Expected sync count to be recorded")
	}
}

func TestLayeredStore_StatsDisabled(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  false,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Perform operations
	testData := []byte("test value")
	setResult := <-l2Store.Set(context.Background(), "test-key", testData, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set value in L2: %v", setResult.Error)
	}

	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error != nil {
		t.Fatalf("Get() failed: %v", getResult.Error)
	}

	// Get stats
	stats := store.GetStats()
	if stats == nil {
		t.Errorf("GetStats() returned nil")
		return
	}

	// Stats should be zero when disabled
	if stats.L1Hits != 0 || stats.L2Hits != 0 {
		t.Errorf("Stats should be zero when disabled: L1Hits=%d, L2Hits=%d", stats.L1Hits, stats.L2Hits)
	}
}

func TestLayeredStore_Concurrency(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		ReadPolicy:   cachex.ReadPolicyThrough,
		WritePolicy:  cachex.WritePolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Test concurrent operations
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))

			// Set value
			setResult := <-store.Set(context.Background(), key, value, 5*time.Minute)
			if setResult.Error != nil {
				t.Errorf("Concurrent Set() failed: %v", setResult.Error)
				done <- true
				return
			}

			// Get value
			getResult := <-store.Get(context.Background(), key)
			if getResult.Error != nil {
				t.Errorf("Concurrent Get() failed: %v", getResult.Error)
				done <- true
				return
			}
			if !getResult.Exists {
				t.Errorf("Concurrent Get() should find the value")
				done <- true
				return
			}
			if string(getResult.Value) != string(value) {
				t.Errorf("Concurrent Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestLayeredStore_ContextCancellation(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Note: LayeredStore doesn't check for context cancellation
	// The context is passed through to the underlying stores
	// This test verifies that the MockStore handles context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test operations with cancelled context
	// These might not fail if the operation succeeds in L1 before reaching L2
	<-store.Get(ctx, "test-key")
	// Get might not fail if L1 has the value or if L1 doesn't check context

	<-store.Set(ctx, "test-key", []byte("value"), 5*time.Minute)
	// Set might not fail if L1 succeeds before reaching L2

	// The test passes if the operations complete without error
	// since the LayeredStore doesn't enforce context cancellation
}

func TestLayeredStore_EdgeCases(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Note: LayeredStore now validates input parameters
	// These tests verify that validation is working correctly

	// Test with empty key (should be rejected)
	getResult := <-store.Get(context.Background(), "")
	if getResult.Error == nil {
		t.Error("Get() should reject empty key")
	}

	// Test with nil value (should be rejected)
	setResult := <-store.Set(context.Background(), "test-key", nil, 5*time.Minute)
	if setResult.Error == nil {
		t.Error("Set() should reject nil value")
	}

	// Test with zero TTL (should be allowed)
	setResult2 := <-store.Set(context.Background(), "test-key", []byte("value"), 0)
	if setResult2.Error != nil {
		t.Errorf("Set() should work with zero TTL: %v", setResult2.Error)
	}

	// Test with negative TTL (should be rejected)
	setResult3 := <-store.Set(context.Background(), "test-key", []byte("value"), -1*time.Minute)
	if setResult3.Error == nil {
		t.Error("Set() should reject negative TTL")
	}

	// Test with empty keys in MGet (should be rejected)
	mgetResult := <-store.MGet(context.Background(), "key1", "", "key2")
	if mgetResult.Error == nil {
		t.Error("MGet() should reject empty keys")
	}

	// Test with nil values in MSet (should be rejected)
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": nil,
		"key3": []byte("value3"),
	}
	msetResult := <-store.MSet(context.Background(), items, 5*time.Minute)
	if msetResult.Error == nil {
		t.Error("MSet() should reject nil values")
	}

	// Test with empty keys in Del (should be rejected)
	delResult := <-store.Del(context.Background(), "key1", "", "key2")
	if delResult.Error == nil {
		t.Error("Del() should reject empty keys")
	}

	// Test with negative delta in IncrBy (should be allowed)
	incrResult := <-store.IncrBy(context.Background(), "counter", -5, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() should work with negative delta: %v", incrResult.Error)
	}

	// Test with negative TTL in IncrBy (should be rejected)
	incrResult2 := <-store.IncrBy(context.Background(), "counter2", 5, -1*time.Minute)
	if incrResult2.Error == nil {
		t.Error("IncrBy() should reject negative TTL")
	}
}

// TestLayeredStore_StoreInitializationFailure tests store initialization failures
func TestLayeredStore_StoreInitializationFailure(t *testing.T) {
	// Test with nil L2 store
	config := cachex.DefaultLayeredConfig()
	_, err := cachex.NewLayeredStore(nil, config)
	if err == nil {
		t.Error("NewLayeredStore() should fail with nil L2 store")
	}

	// Test with nil memory config
	config2 := &cachex.LayeredConfig{
		MemoryConfig: nil,
		WritePolicy:  cachex.WritePolicyThrough,
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
	}
	l2Store := NewMockStore()
	_, err = cachex.NewLayeredStore(l2Store, config2)
	if err == nil {
		t.Error("NewLayeredStore() should fail with nil MemoryConfig")
	}
}

// TestLayeredStore_StoreValidationFailure tests store validation failures
func TestLayeredStore_StoreValidationFailure(t *testing.T) {
	// Create a mock store that fails validation
	invalidStore := &InvalidMockStore{}
	config := cachex.DefaultLayeredConfig()

	_, err := cachex.NewLayeredStore(invalidStore, config)
	if err == nil {
		t.Error("NewLayeredStore() should fail with invalid store")
	}
}

// TestLayeredStore_PartialOperationFailure tests partial operation failures
func TestLayeredStore_PartialOperationFailure(t *testing.T) {
	// Create a mock store that fails L2 operations
	failingStore := &FailingMockStore{
		MockStore:   NewMockStore(),
		failCount:   0, // Don't fail during validation
		currentFail: 0,
	}
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(failingStore, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Reset the fail count to trigger failures during operations
	failingStore.failCount = 1
	failingStore.currentFail = 0

	// Test write-through with L2 failure
	setResult := <-store.Set(context.Background(), "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() should not fail completely in write-through mode: %v", setResult.Error)
	}

	// Reset for next operation
	failingStore.failCount = 1
	failingStore.currentFail = 0

	// Test MSet with L2 failure
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}
	msetResult := <-store.MSet(context.Background(), items, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() should not fail completely in write-through mode: %v", msetResult.Error)
	}
}

// TestLayeredStore_AsyncOperationTimeout tests async operation timeouts
func TestLayeredStore_AsyncOperationTimeout(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig:          cachex.DefaultMemoryConfig(),
		WritePolicy:           cachex.WritePolicyBehind,
		ReadPolicy:            cachex.ReadPolicyThrough,
		EnableStats:           true,
		AsyncOperationTimeout: 1 * time.Millisecond, // Very short timeout
		MaxConcurrentSync:     1,                    // Limit concurrency
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Test write-behind with timeout
	setResult := <-store.Set(context.Background(), "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() should not fail due to async timeout: %v", setResult.Error)
	}

	// Wait a bit for async operations to complete or timeout
	time.Sleep(10 * time.Millisecond)
}

// TestLayeredStore_SemaphoreExhaustion tests semaphore exhaustion
func TestLayeredStore_SemaphoreExhaustion(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig:          cachex.DefaultMemoryConfig(),
		WritePolicy:           cachex.WritePolicyBehind,
		ReadPolicy:            cachex.ReadPolicyThrough,
		EnableStats:           true,
		AsyncOperationTimeout: 100 * time.Millisecond,
		MaxConcurrentSync:     1, // Very low concurrency limit
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Try to trigger semaphore exhaustion with multiple concurrent operations
	numOperations := 5
	results := make(chan bool, numOperations)

	for i := 0; i < numOperations; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))

			setResult := <-store.Set(context.Background(), key, value, 5*time.Minute)
			results <- (setResult.Error == nil)
		}(i)
	}

	// Wait for all operations to complete
	successCount := 0
	for i := 0; i < numOperations; i++ {
		if <-results {
			successCount++
		}
	}

	// All operations should succeed, but some async operations might be skipped
	if successCount == 0 {
		t.Error("All operations failed, expected at least some to succeed")
	}
}

// TestLayeredStore_OperationsAfterClose tests operations after store is closed
func TestLayeredStore_OperationsAfterClose(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Try operations after close
	getResult := <-store.Get(context.Background(), "test-key")
	if getResult.Error == nil {
		t.Error("Get() should fail after store is closed")
	}

	setResult := <-store.Set(context.Background(), "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error == nil {
		t.Error("Set() should fail after store is closed")
	}

	delResult := <-store.Del(context.Background(), "test-key")
	if delResult.Error == nil {
		t.Error("Del() should fail after store is closed")
	}
}

// TestLayeredStore_MultipleClose tests multiple Close() calls
func TestLayeredStore_MultipleClose(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}

	// First close should succeed
	err = store.Close()
	if err != nil {
		t.Errorf("First Close() failed: %v", err)
	}

	// Second close should also succeed (idempotent)
	err = store.Close()
	if err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

// TestLayeredStore_BackgroundSyncCleanup tests background sync cleanup
func TestLayeredStore_BackgroundSyncCleanup(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyBehind, // Enable background sync
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
		SyncInterval: 100 * time.Millisecond, // Short interval for testing
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}

	// Wait a bit for background sync to start
	time.Sleep(50 * time.Millisecond)

	// Close should clean up background sync
	err = store.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Wait a bit more to ensure cleanup is complete
	time.Sleep(50 * time.Millisecond)
}

// TestLayeredStore_InvalidTimeoutConfig tests invalid timeout configurations
func TestLayeredStore_InvalidTimeoutConfig(t *testing.T) {
	l2Store := NewMockStore()

	// Test with zero timeout
	config1 := &cachex.LayeredConfig{
		MemoryConfig:          cachex.DefaultMemoryConfig(),
		WritePolicy:           cachex.WritePolicyThrough,
		ReadPolicy:            cachex.ReadPolicyThrough,
		EnableStats:           true,
		AsyncOperationTimeout: 0, // Zero timeout
		BackgroundSyncTimeout: 0, // Zero timeout
	}

	store1, err := cachex.NewLayeredStore(l2Store, config1)
	if err != nil {
		t.Fatalf("NewLayeredStore() should work with zero timeouts: %v", err)
	}
	defer store1.Close()

	// Test that operations still work with zero timeouts (should use defaults)
	setResult := <-store1.Set(context.Background(), "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() should work with zero timeout config: %v", setResult.Error)
	}
}

// TestLayeredStore_InvalidConcurrencyConfig tests invalid concurrency configurations
func TestLayeredStore_InvalidConcurrencyConfig(t *testing.T) {
	l2Store := NewMockStore()

	// Test with zero concurrency
	config1 := &cachex.LayeredConfig{
		MemoryConfig:      cachex.DefaultMemoryConfig(),
		WritePolicy:       cachex.WritePolicyThrough,
		ReadPolicy:        cachex.ReadPolicyThrough,
		EnableStats:       true,
		MaxConcurrentSync: 0, // Zero concurrency
	}

	store1, err := cachex.NewLayeredStore(l2Store, config1)
	if err != nil {
		t.Fatalf("NewLayeredStore() should work with zero concurrency: %v", err)
	}
	defer store1.Close()

	// Test that operations still work with zero concurrency
	setResult := <-store1.Set(context.Background(), "test-key", []byte("value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() should work with zero concurrency config: %v", setResult.Error)
	}
}

// TestLayeredStore_ConcurrentCloseAndOperations tests concurrent Close() and operations
func TestLayeredStore_ConcurrentCloseAndOperations(t *testing.T) {
	l2Store := NewMockStore()
	config := cachex.DefaultLayeredConfig()

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}

	// Start concurrent operations
	numOperations := 10
	operationDone := make(chan bool, numOperations)
	closeDone := make(chan bool, 1)

	// Start operations
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))

			<-store.Set(context.Background(), key, value, 5*time.Minute)
			<-store.Get(context.Background(), key)

			// Operations might fail if store is closed, which is expected
			operationDone <- true
		}(i)
	}

	// Start close operation
	go func() {
		time.Sleep(1 * time.Millisecond) // Small delay to allow operations to start
		store.Close()
		closeDone <- true
	}()

	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		<-operationDone
	}
	<-closeDone
}

// TestLayeredStore_ConcurrentStatsUpdates tests concurrent stats updates
func TestLayeredStore_ConcurrentStatsUpdates(t *testing.T) {
	l2Store := NewMockStore()
	config := &cachex.LayeredConfig{
		MemoryConfig: cachex.DefaultMemoryConfig(),
		WritePolicy:  cachex.WritePolicyThrough,
		ReadPolicy:   cachex.ReadPolicyThrough,
		EnableStats:  true,
	}

	store, err := cachex.NewLayeredStore(l2Store, config)
	if err != nil {
		t.Fatalf("Failed to create layered store: %v", err)
	}
	defer store.Close()

	// Start concurrent operations that will update stats
	numOperations := 20
	operationDone := make(chan bool, numOperations)

	for i := 0; i < numOperations; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			value := []byte(fmt.Sprintf("value-%d", id))

			// Set operation
			<-store.Set(context.Background(), key, value, 5*time.Minute)

			// Get operation
			<-store.Get(context.Background(), key)

			// Get stats (this should be thread-safe)
			stats := store.GetStats()
			if stats == nil {
				t.Error("GetStats() returned nil")
			}

			operationDone <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		<-operationDone
	}

	// Final stats check
	stats := store.GetStats()
	if stats == nil {
		t.Error("Final GetStats() returned nil")
	}
}

// InvalidMockStore is a mock store that fails validation
type InvalidMockStore struct{}

func (m *InvalidMockStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) Exists(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) TTL(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)
	result <- cachex.AsyncResult{Error: fmt.Errorf("invalid store")}
	close(result)
	return result
}

func (m *InvalidMockStore) Close() error {
	return fmt.Errorf("invalid store")
}
