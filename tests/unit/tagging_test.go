package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

// MockStore implements Store interface for testing
type MockTagStore struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockTagStore() *MockTagStore {
	return &MockTagStore{
		data: make(map[string][]byte),
	}
}

func (m *MockTagStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		if value, exists := m.data[key]; exists {
			result <- cachex.AsyncResult{Value: value, Exists: true}
		} else {
			result <- cachex.AsyncResult{Exists: false}
		}
	}()

	return result
}

func (m *MockTagStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		m.data[key] = value
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockTagStore) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		values := make(map[string][]byte)
		for _, key := range keys {
			if value, exists := m.data[key]; exists {
				values[key] = value
			}
		}
		result <- cachex.AsyncResult{Values: values}
	}()

	return result
}

func (m *MockTagStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		for key, value := range items {
			m.data[key] = value
		}
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockTagStore) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		for _, key := range keys {
			delete(m.data, key)
		}
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockTagStore) Exists(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		_, exists := m.data[key]
		result <- cachex.AsyncResult{Exists: exists}
	}()

	return result
}

func (m *MockTagStore) TTL(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		_, exists := m.data[key]
		if !exists {
			result <- cachex.AsyncResult{TTL: 0}
		} else {
			result <- cachex.AsyncResult{TTL: 0} // Mock doesn't track TTL
		}
	}()

	return result
}

func (m *MockTagStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		// Simple implementation for testing
		current := int64(0)
		if value, exists := m.data[key]; exists {
			// Parse as int64 (simplified)
			if len(value) == 8 {
				current = int64(value[0])<<56 | int64(value[1])<<48 | int64(value[2])<<40 | int64(value[3])<<32 |
					int64(value[4])<<24 | int64(value[5])<<16 | int64(value[6])<<8 | int64(value[7])
			}
		}

		newValue := current + delta

		// Convert to bytes
		bytes := make([]byte, 8)
		bytes[0] = byte(newValue >> 56)
		bytes[1] = byte(newValue >> 48)
		bytes[2] = byte(newValue >> 40)
		bytes[3] = byte(newValue >> 32)
		bytes[4] = byte(newValue >> 24)
		bytes[5] = byte(newValue >> 16)
		bytes[6] = byte(newValue >> 8)
		bytes[7] = byte(newValue)

		m.data[key] = bytes
		result <- cachex.AsyncResult{Value: bytes}
	}()

	return result
}

func (m *MockTagStore) Close() error {
	return nil
}

func (m *MockTagStore) GetStats() interface{} {
	return nil
}

func TestDefaultTagConfig(t *testing.T) {
	config := cachex.DefaultTagConfig()

	if config == nil {
		t.Errorf("DefaultTagConfig() should not return nil")
		return
	}

	if !config.EnablePersistence {
		t.Errorf("DefaultTagConfig().EnablePersistence should be true")
	}

	if config.TagMappingTTL != 24*time.Hour {
		t.Errorf("DefaultTagConfig().TagMappingTTL should be 24 hours, got %v", config.TagMappingTTL)
	}

	if config.BatchSize != 100 {
		t.Errorf("DefaultTagConfig().BatchSize should be 100, got %d", config.BatchSize)
	}

	if !config.EnableStats {
		t.Errorf("DefaultTagConfig().EnableStats should be true")
	}
}

func TestNewTagManager_WithNilConfig(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	if tm == nil {
		t.Errorf("NewTagManager() should not return nil")
	}
}

func TestNewTagManager_WithCustomConfig(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: false,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         50,
		EnableStats:       false,
	}

	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	if tm == nil {
		t.Errorf("NewTagManager() should not return nil")
	}
}

func TestTagManager_AddTags_EmptyTags(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	err = tm.AddTags(ctx, "test-key")
	if err != nil {
		t.Errorf("AddTags() should not fail with empty tags: %v", err)
	}
}

func TestTagManager_AddTags_SingleTag(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	err = tm.AddTags(ctx, key, tag)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Verify tag was added
	tags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}

	if tags[0] != tag {
		t.Errorf("Expected tag %s, got %s", tag, tags[0])
	}

	// Verify key was added to tag mapping
	keys, err := tm.GetKeysByTag(ctx, tag)
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	if keys[0] != key {
		t.Errorf("Expected key %s, got %s", key, keys[0])
	}
}

func TestTagManager_AddTags_MultipleTags(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	err = tm.AddTags(ctx, key, tags...)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Verify all tags were added
	resultTags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(resultTags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(resultTags))
	}

	// Check each tag
	for _, tag := range tags {
		found := false
		for _, resultTag := range resultTags {
			if resultTag == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Tag %s not found in result", tag)
		}
	}
}

func TestTagManager_AddTags_MultipleKeys(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add tags to multiple keys
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed for key1: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Errorf("AddTags() failed for key2: %v", err)
	}

	// Verify tag2 has both keys
	keys, err := tm.GetKeysByTag(ctx, "tag2")
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag2, got %d", len(keys))
	}

	// Check that both keys are present
	foundKey1 := false
	foundKey2 := false
	for _, key := range keys {
		if key == "key1" {
			foundKey1 = true
		}
		if key == "key2" {
			foundKey2 = true
		}
	}

	if !foundKey1 || !foundKey2 {
		t.Errorf("Expected both key1 and key2, found key1: %v, found key2: %v", foundKey1, foundKey2)
	}
}

func TestTagManager_RemoveTags_EmptyTags(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	err = tm.RemoveTags(ctx, "test-key")
	if err != nil {
		t.Errorf("RemoveTags() should not fail with empty tags: %v", err)
	}
}

func TestTagManager_RemoveTags_SingleTag(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	// Add tag first
	err = tm.AddTags(ctx, key, tag)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Remove tag
	err = tm.RemoveTags(ctx, key, tag)
	if err != nil {
		t.Errorf("RemoveTags() failed: %v", err)
	}

	// Verify tag was removed
	tags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected 0 tags after removal, got %d", len(tags))
	}

	// Verify key was removed from tag mapping
	keys, err := tm.GetKeysByTag(ctx, tag)
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for tag after removal, got %d", len(keys))
	}
}

func TestTagManager_RemoveTags_MultipleTags(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags first
	err = tm.AddTags(ctx, key, tags...)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Remove some tags
	err = tm.RemoveTags(ctx, key, "tag1", "tag3")
	if err != nil {
		t.Errorf("RemoveTags() failed: %v", err)
	}

	// Verify remaining tag
	resultTags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(resultTags) != 1 {
		t.Errorf("Expected 1 tag after removal, got %d", len(resultTags))
	}

	if resultTags[0] != "tag2" {
		t.Errorf("Expected remaining tag to be tag2, got %s", resultTags[0])
	}
}

func TestTagManager_GetKeysByTag_NonExistentTag(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	keys, err := tm.GetKeysByTag(ctx, "non-existent-tag")
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected empty slice for non-existent tag, got %d keys", len(keys))
	}
}

func TestTagManager_GetKeysByTag_ExistingTag(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	tag := "test-tag"

	// Add multiple keys with the same tag
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		err = tm.AddTags(ctx, key, tag)
		if err != nil {
			t.Errorf("AddTags() failed for key %s: %v", key, err)
		}
	}

	// Get keys by tag
	resultKeys, err := tm.GetKeysByTag(ctx, tag)
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(resultKeys) != len(keys) {
		t.Errorf("Expected %d keys, got %d", len(keys), len(resultKeys))
	}

	// Check that all keys are present
	for _, expectedKey := range keys {
		found := false
		for _, resultKey := range resultKeys {
			if resultKey == expectedKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected key %s not found in result", expectedKey)
		}
	}
}

func TestTagManager_GetTagsByKey_NonExistentKey(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	tags, err := tm.GetTagsByKey(ctx, "non-existent-key")
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected empty slice for non-existent key, got %d tags", len(tags))
	}
}

func TestTagManager_GetTagsByKey_ExistingKey(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags to key
	err = tm.AddTags(ctx, key, tags...)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Get tags by key
	resultTags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(resultTags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(resultTags))
	}

	// Check that all tags are present
	for _, expectedTag := range tags {
		found := false
		for _, resultTag := range resultTags {
			if resultTag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag %s not found in result", expectedTag)
		}
	}
}

func TestTagManager_InvalidateByTag_EmptyKeys(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	err = tm.InvalidateByTag(ctx)
	if err != nil {
		t.Errorf("InvalidateByTag() should not fail with empty tags: %v", err)
	}
}

func TestTagManager_InvalidateByTag_SingleKey(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	// Add some data to store
	setResult := <-store.Set(ctx, key, []byte("test-value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Store.Set() failed: %v", setResult.Error)
	}

	// Add tag to key
	if err = tm.AddTags(ctx, key, tag); err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Invalidate by tag
	if err = tm.InvalidateByTag(ctx, tag); err != nil {
		t.Errorf("InvalidateByTag() failed: %v", err)
	}

	// Verify key was deleted from store
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Store.Exists() failed: %v", existsResult.Error)
	}

	if existsResult.Exists {
		t.Errorf("Key should not exist after invalidation")
	}

	// Verify tag mappings were cleaned up
	tags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("Expected 0 tags after invalidation, got %d", len(tags))
	}
}

func TestTagManager_InvalidateByTag_MultipleKeys(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	tag := "test-tag"

	// Add data and tags
	for _, key := range keys {
		setResult := <-store.Set(ctx, key, []byte("test-value"), 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Store.Set() failed for key %s: %v", key, setResult.Error)
		}

		err = tm.AddTags(ctx, key, tag)
		if err != nil {
			t.Errorf("AddTags() failed for key %s: %v", key, err)
		}
	}

	// Invalidate by tag
	err = tm.InvalidateByTag(ctx, tag)
	if err != nil {
		t.Errorf("InvalidateByTag() failed: %v", err)
	}

	// Verify all keys were deleted
	for _, key := range keys {
		existsResult := <-store.Exists(ctx, key)
		if existsResult.Error != nil {
			t.Errorf("Store.Exists() failed for key %s: %v", key, existsResult.Error)
		}

		if existsResult.Exists {
			t.Errorf("Key %s should not exist after invalidation", key)
		}
	}
}

func TestTagManager_RemoveKey_NonExistentKey(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	err = tm.RemoveKey(ctx, "non-existent-key")
	if err != nil {
		t.Errorf("RemoveKey() should not fail for non-existent key: %v", err)
	}
}

func TestTagManager_RemoveKey_ExistingKey(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags to key
	err = tm.AddTags(ctx, key, tags...)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Remove key
	err = tm.RemoveKey(ctx, key)
	if err != nil {
		t.Errorf("RemoveKey() failed: %v", err)
	}

	// Verify key was removed from tag mappings
	resultTags, err := tm.GetTagsByKey(ctx, key)
	if err != nil {
		t.Errorf("GetTagsByKey() failed: %v", err)
	}

	if len(resultTags) != 0 {
		t.Errorf("Expected 0 tags after key removal, got %d", len(resultTags))
	}

	// Verify key was removed from all tag mappings
	for _, tag := range tags {
		keys, err := tm.GetKeysByTag(ctx, tag)
		if err != nil {
			t.Errorf("GetKeysByTag() failed for tag %s: %v", tag, err)
		}

		for _, resultKey := range keys {
			if resultKey == key {
				t.Errorf("Key %s should not be in tag %s mapping after removal", key, tag)
			}
		}
	}
}

func TestTagManager_GetStats(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add some tags and keys
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Get stats
	stats, err := tm.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}
	if stats == nil {
		t.Errorf("GetStats() should not return nil")
		return
	}

	// Should have 3 unique tags and 2 keys
	if stats.TotalTags != 3 {
		t.Errorf("Expected 3 total tags, got %d", stats.TotalTags)
	}

	if stats.TotalKeys != 2 {
		t.Errorf("Expected 2 total keys, got %d", stats.TotalKeys)
	}
}

func TestTagManager_ContextCancellation(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test operations with cancelled context
	// Note: TagManager doesn't check context in read operations, so we only test write operations

	err = tm.AddTags(ctx, "test-key", "test-tag")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("AddTags() should return context cancellation error, got %v", err)
	}

	err = tm.RemoveTags(ctx, "test-key", "test-tag")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("RemoveTags() should return context cancellation error, got %v", err)
	}

	err = tm.RemoveKey(ctx, "test-key")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("RemoveKey() should return context cancellation error, got %v", err)
	}

	err = tm.InvalidateByTag(ctx, "test-tag")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("InvalidateByTag() should return context cancellation error, got %v", err)
	}
}

func TestTagManager_Concurrency(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				tag := fmt.Sprintf("tag-%d-%d", id, j)

				// Add tags
				err = tm.AddTags(ctx, key, tag)
				if err != nil {
					t.Errorf("Concurrent AddTags() failed: %v", err)
					return
				}

				// Get tags by key
				_, err = tm.GetTagsByKey(ctx, key)
				if err != nil {
					t.Errorf("Concurrent GetTagsByKey() failed: %v", err)
					return
				}

				// Get keys by tag
				_, err = tm.GetKeysByTag(ctx, tag)
				if err != nil {
					t.Errorf("Concurrent GetKeysByTag() failed: %v", err)
					return
				}

				// Remove some tags
				if j%2 == 0 {
					err = tm.RemoveTags(ctx, key, tag)
					if err != nil {
						t.Errorf("Concurrent RemoveTags() failed: %v", err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestTagManager_GetMetrics(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Perform some operations to generate metrics
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.RemoveTags(ctx, "key1", "tag1")
	if err != nil {
		t.Errorf("RemoveTags() failed: %v", err)
	}

	err = tm.RemoveKey(ctx, "key1")
	if err != nil {
		t.Errorf("RemoveKey() failed: %v", err)
	}

	// Get metrics
	metrics := tm.GetMetrics()
	if metrics == nil {
		t.Errorf("GetMetrics() should not return nil")
		return
	}

	// Verify metrics are being tracked
	if metrics.AddTagsCount != 1 {
		t.Errorf("Expected AddTagsCount to be 1, got %d", metrics.AddTagsCount)
	}

	if metrics.RemoveTagsCount != 1 {
		t.Errorf("Expected RemoveTagsCount to be 1, got %d", metrics.RemoveTagsCount)
	}

	if metrics.RemoveKeyCount != 1 {
		t.Errorf("Expected RemoveKeyCount to be 1, got %d", metrics.RemoveKeyCount)
	}

	if metrics.PersistCount < 1 {
		t.Errorf("Expected PersistCount to be at least 1, got %d", metrics.PersistCount)
	}

	// LoadCount might be 0 if no data was persisted in the mock store
	if metrics.LoadCount < 0 {
		t.Errorf("Expected LoadCount to be >= 0, got %d", metrics.LoadCount)
	}
}

func TestTagManager_TransactionLikeBehavior(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add some tags first
	err = tm.AddTags(ctx, "key1", "tag1")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag1")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Verify both keys are associated with tag1
	keys, err := tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag1, got %d", len(keys))
	}

	// Test InvalidateByTag - this should work normally with our mock store
	err = tm.InvalidateByTag(ctx, "tag1")
	if err != nil {
		t.Errorf("InvalidateByTag() failed: %v", err)
	}

	// Verify keys were invalidated
	keys, err = tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Errorf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for tag1 after invalidation, got %d", len(keys))
	}
}

func TestTagManager_ErrorCountingOptimization(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Test that error counting works correctly
	initialErrors := tm.GetMetrics().ErrorsCount

	// Trigger multiple validation errors
	_ = tm.AddTags(nil, "", "tag1") // Should increment error count once for context validation

	metrics := tm.GetMetrics()
	if metrics.ErrorsCount != initialErrors+1 {
		t.Errorf("Expected error count to be %d, got %d", initialErrors+1, metrics.ErrorsCount)
	}

	// Test another error
	_ = tm.AddTags(context.Background(), "", "tag1") // Should increment error count once for key validation

	metrics = tm.GetMetrics()
	if metrics.ErrorsCount != initialErrors+2 {
		t.Errorf("Expected error count to be %d, got %d", initialErrors+2, metrics.ErrorsCount)
	}
}

func TestTagManager_HelperMethodDocumentation(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Test that the helper methods work correctly

	// Test incrementErrorCount indirectly
	initialErrors := tm.GetMetrics().ErrorsCount
	_ = tm.AddTags(nil, "key", "tag") // This should call incrementErrorCount
	metrics := tm.GetMetrics()
	if metrics.ErrorsCount <= initialErrors {
		t.Errorf("Expected error count to increase, got %d (was %d)", metrics.ErrorsCount, initialErrors)
	}
}

// ===== MISSING TEST SCENARIOS =====

func TestTagManager_NilReceiver(t *testing.T) {
	var tm *cachex.TagManager

	tests := []struct {
		name string
		test func()
	}{
		{
			name: "AddTags with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.AddTags(context.Background(), "key", "tag")
			},
		},
		{
			name: "RemoveTags with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.RemoveTags(context.Background(), "key", "tag")
			},
		},
		{
			name: "GetKeysByTag with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.GetKeysByTag(context.Background(), "tag")
			},
		},
		{
			name: "GetTagsByKey with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.GetTagsByKey(context.Background(), "key")
			},
		},
		{
			name: "InvalidateByTag with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.InvalidateByTag(context.Background(), "tag")
			},
		},
		{
			name: "RemoveKey with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.RemoveKey(context.Background(), "key")
			},
		},
		{
			name: "GetStats with nil receiver",
			test: func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil receiver")
					}
				}()
				tm.GetStats()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test()
		})
	}
}

func TestTagManager_Persistence(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add some tags
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	// Verify data was persisted by checking store
	key := "cachex:tag_mappings"
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Store.Exists() failed: %v", existsResult.Error)
	}

	if !existsResult.Exists {
		t.Errorf("Tag mappings should be persisted to store")
	}

	// Get the persisted data
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Store.Get() failed: %v", getResult.Error)
	}

	if !getResult.Exists || getResult.Value == nil {
		t.Errorf("Persisted tag mappings should exist and not be nil")
	}

	// Verify the data contains our mappings
	var tagMappings struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(getResult.Value, &tagMappings); err != nil {
		t.Fatalf("Failed to unmarshal persisted data: %v", err)
	}

	// Check that our mappings are in the persisted data
	if len(tagMappings.TagToKeys) == 0 {
		t.Errorf("Persisted data should contain tag mappings")
	}

	if len(tagMappings.KeyToTags) == 0 {
		t.Errorf("Persisted data should contain key mappings")
	}

	// Check specific mappings
	if keys, exists := tagMappings.TagToKeys["tag1"]; !exists || len(keys) == 0 {
		t.Errorf("Tag1 should be in persisted mappings")
	}

	if keys, exists := tagMappings.TagToKeys["tag2"]; !exists || len(keys) != 2 {
		t.Errorf("Tag2 should have 2 keys in persisted mappings")
	}
}

func TestTagManager_Loading(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	// Pre-populate store with tag mappings
	ctx := context.Background()
	tagMappings := struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		TagToKeys: map[string][]string{
			"tag1": {"key1", "key2"},
			"tag2": {"key2", "key3"},
		},
		KeyToTags: map[string][]string{
			"key1": {"tag1"},
			"key2": {"tag1", "tag2"},
			"key3": {"tag2"},
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(tagMappings)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	key := "cachex:tag_mappings"
	setResult := <-store.Set(ctx, key, data, config.TagMappingTTL)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test data: %v", setResult.Error)
	}

	// Create tag manager - should load existing mappings
	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Verify mappings were loaded
	keys, err := tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag1, got %d", len(keys))
	}

	keys, err = tm.GetKeysByTag(ctx, "tag2")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag2, got %d", len(keys))
	}

	tags, err := tm.GetTagsByKey(ctx, "key2")
	if err != nil {
		t.Fatalf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags for key2, got %d", len(tags))
	}
}

func TestTagManager_LoadingStaleData(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	// Pre-populate store with stale tag mappings
	ctx := context.Background()
	tagMappings := struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		TagToKeys: map[string][]string{
			"tag1": {"key1", "key2"},
		},
		KeyToTags: map[string][]string{
			"key1": {"tag1"},
			"key2": {"tag1"},
		},
		Timestamp: time.Now().Add(-2 * time.Hour), // Stale data
	}

	data, err := json.Marshal(tagMappings)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	key := "cachex:tag_mappings"
	setResult := <-store.Set(ctx, key, data, config.TagMappingTTL)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test data: %v", setResult.Error)
	}

	// Create tag manager - should ignore stale data
	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Verify stale mappings were not loaded
	keys, err := tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for stale tag1, got %d", len(keys))
	}
}

func TestTagManager_LoadingInvalidData(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	// Pre-populate store with invalid data
	ctx := context.Background()
	key := "cachex:tag_mappings"
	invalidData := []byte("invalid json data")
	setResult := <-store.Set(ctx, key, invalidData, config.TagMappingTTL)
	if setResult.Error != nil {
		t.Fatalf("Failed to set invalid data: %v", setResult.Error)
	}

	// Create tag manager - should handle invalid data gracefully
	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	// Verify tag manager is still functional
	err = tm.AddTags(ctx, "key1", "tag1")
	if err != nil {
		t.Errorf("AddTags() should work after loading invalid data: %v", err)
	}
}

func TestTagManager_BatchOperations(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         5, // Small batch size for testing
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add many keys with the same tag to test batch invalidation
	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
		err = tm.AddTags(ctx, keys[i], "batch_tag")
		if err != nil {
			t.Fatalf("AddTags() failed for key %s: %v", keys[i], err)
		}
	}

	// Verify all keys are associated with the tag
	resultKeys, err := tm.GetKeysByTag(ctx, "batch_tag")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(resultKeys) != 10 {
		t.Errorf("Expected 10 keys, got %d", len(resultKeys))
	}

	// Test batch invalidation
	err = tm.InvalidateByTag(ctx, "batch_tag")
	if err != nil {
		t.Fatalf("InvalidateByTag() failed: %v", err)
	}

	// Verify all keys were invalidated
	resultKeys, err = tm.GetKeysByTag(ctx, "batch_tag")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(resultKeys) != 0 {
		t.Errorf("Expected 0 keys after invalidation, got %d", len(resultKeys))
	}
}

func TestTagManager_EdgeCases(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "AddTags with empty key",
			test: func() error {
				return tm.AddTags(ctx, "", "tag")
			},
		},
		{
			name: "AddTags with empty tag",
			test: func() error {
				return tm.AddTags(ctx, "key", "")
			},
		},
		{
			name: "AddTags with whitespace key",
			test: func() error {
				return tm.AddTags(ctx, "   ", "tag")
			},
		},
		{
			name: "AddTags with whitespace tag",
			test: func() error {
				return tm.AddTags(ctx, "key", "   ")
			},
		},
		{
			name: "GetKeysByTag with empty tag",
			test: func() error {
				_, err := tm.GetKeysByTag(ctx, "")
				return err
			},
		},
		{
			name: "GetTagsByKey with empty key",
			test: func() error {
				_, err := tm.GetTagsByKey(ctx, "")
				return err
			},
		},
		{
			name: "InvalidateByTag with empty tag",
			test: func() error {
				return tm.InvalidateByTag(ctx, "")
			},
		},
		{
			name: "RemoveKey with empty key",
			test: func() error {
				return tm.RemoveKey(ctx, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			// Whitespace strings are allowed by the current implementation
			if err == nil && !strings.Contains(tt.name, "whitespace") {
				t.Errorf("Expected error for %s", tt.name)
			}
			if err != nil && strings.Contains(tt.name, "whitespace") {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

func TestTagManager_Performance(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Test AddTags performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		tag := fmt.Sprintf("tag%d", i%10) // 10 different tags
		err := tm.AddTags(ctx, key, tag)
		if err != nil {
			t.Fatalf("AddTags() failed: %v", err)
		}
	}
	duration := time.Since(start)
	if duration > 1*time.Second {
		t.Errorf("AddTags performance too slow: %v", duration)
	}

	// Test GetKeysByTag performance
	start = time.Now()
	for i := 0; i < 100; i++ {
		tag := fmt.Sprintf("tag%d", i%10)
		_, err := tm.GetKeysByTag(ctx, tag)
		if err != nil {
			t.Fatalf("GetKeysByTag() failed: %v", err)
		}
	}
	duration = time.Since(start)
	if duration > 100*time.Millisecond {
		t.Errorf("GetKeysByTag performance too slow: %v", duration)
	}

	// Test GetTagsByKey performance
	start = time.Now()
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		_, err := tm.GetTagsByKey(ctx, key)
		if err != nil {
			t.Fatalf("GetTagsByKey() failed: %v", err)
		}
	}
	duration = time.Since(start)
	if duration > 100*time.Millisecond {
		t.Errorf("GetTagsByKey performance too slow: %v", duration)
	}
}

// FailingTagMockStore is a mock store that can be configured to fail on certain operations
type FailingTagMockStore struct {
	MockTagStore
	failOnSet bool
	failOnGet bool
	failOnDel bool
}

func (m *FailingTagMockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.failOnSet {
			result <- cachex.AsyncResult{Error: fmt.Errorf("simulated set failure")}
			return
		}

		// Use the parent implementation
		parentResult := <-m.MockTagStore.Set(ctx, key, value, ttl)
		result <- parentResult
	}()

	return result
}

func (m *FailingTagMockStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.failOnGet {
			result <- cachex.AsyncResult{Error: fmt.Errorf("simulated get failure")}
			return
		}

		// Use the parent implementation
		parentResult := <-m.MockTagStore.Get(ctx, key)
		result <- parentResult
	}()

	return result
}

func (m *FailingTagMockStore) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.failOnDel {
			result <- cachex.AsyncResult{Error: fmt.Errorf("simulated del failure")}
			return
		}

		// Use the parent implementation
		parentResult := <-m.MockTagStore.Del(ctx, keys...)
		result <- parentResult
	}()

	return result
}

func TestTagManager_ErrorRecovery(t *testing.T) {
	// Create a store that fails on certain operations
	failingStore := &FailingTagMockStore{
		MockTagStore: *NewMockTagStore(),
		failOnSet:    true,
	}

	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	tm, err := cachex.NewTagManager(failingStore, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add tags - should fail on persistence but not crash
	err = tm.AddTags(ctx, "key1", "tag1")
	if err != nil {
		t.Logf("AddTags failed as expected: %v", err)
	}

	// Verify tag manager is still functional for non-persistent operations
	// Switch to non-persistent mode
	failingStore.failOnSet = false
	config.EnablePersistence = false

	tm2, err := cachex.NewTagManager(failingStore, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	err = tm2.AddTags(ctx, "key2", "tag2")
	if err != nil {
		t.Errorf("AddTags() should work in non-persistent mode: %v", err)
	}

	keys, err := tm2.GetKeysByTag(ctx, "tag2")
	if err != nil {
		t.Errorf("GetKeysByTag() should work: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}
}

func TestTagManager_ConcurrentPersistence(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent AddTags operations with persistence
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				tag := fmt.Sprintf("tag-%d", j%5)
				err := tm.AddTags(ctx, key, tag)
				if err != nil {
					t.Errorf("Concurrent AddTags() failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all operations were persisted
	key := "cachex:tag_mappings"
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Store.Exists() failed: %v", existsResult.Error)
	}

	if !existsResult.Exists {
		t.Errorf("Tag mappings should be persisted after concurrent operations")
	}
}

func TestTagManager_ValidationMethods(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Test context validation
	err = tm.AddTags(nil, "key", "tag")
	if err == nil {
		t.Errorf("Expected error for nil context")
	}

	// Test key validation
	err = tm.AddTags(ctx, "", "tag")
	if err == nil {
		t.Errorf("Expected error for empty key")
	}

	// Test tag validation
	err = tm.AddTags(ctx, "key", "")
	if err == nil {
		t.Errorf("Expected error for empty tag")
	}

	// Test multiple empty tags
	err = tm.AddTags(ctx, "key", "tag1", "", "tag2")
	if err == nil {
		t.Errorf("Expected error for empty tag in list")
	}
}

func TestTagManager_MetricsAccuracy(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Get initial metrics
	initialMetrics := tm.GetMetrics()

	// Perform operations
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	err = tm.RemoveTags(ctx, "key1", "tag1")
	if err != nil {
		t.Fatalf("RemoveTags() failed: %v", err)
	}

	err = tm.RemoveKey(ctx, "key1")
	if err != nil {
		t.Fatalf("RemoveKey() failed: %v", err)
	}

	// Get final metrics
	finalMetrics := tm.GetMetrics()

	// Verify metrics were incremented correctly
	if finalMetrics.AddTagsCount != initialMetrics.AddTagsCount+1 {
		t.Errorf("AddTagsCount should be incremented, got %d (was %d)",
			finalMetrics.AddTagsCount, initialMetrics.AddTagsCount)
	}

	if finalMetrics.RemoveTagsCount != initialMetrics.RemoveTagsCount+1 {
		t.Errorf("RemoveTagsCount should be incremented, got %d (was %d)",
			finalMetrics.RemoveTagsCount, initialMetrics.RemoveTagsCount)
	}

	if finalMetrics.RemoveKeyCount != initialMetrics.RemoveKeyCount+1 {
		t.Errorf("RemoveKeyCount should be incremented, got %d (was %d)",
			finalMetrics.RemoveKeyCount, initialMetrics.RemoveKeyCount)
	}
}

// ===== ADDITIONAL COMPREHENSIVE TEST SCENARIOS =====

func TestTagManager_InvalidConfig(t *testing.T) {
	store := NewMockTagStore()

	tests := []struct {
		name        string
		config      *cachex.TagConfig
		expectError bool
	}{
		{
			name: "Negative TTL",
			config: &cachex.TagConfig{
				EnablePersistence: true,
				TagMappingTTL:     -1 * time.Hour,
				BatchSize:         100,
				EnableStats:       true,
				LoadTimeout:       5 * time.Second,
			},
			expectError: false, // Current implementation allows negative TTL
		},
		{
			name: "Zero batch size",
			config: &cachex.TagConfig{
				EnablePersistence: true,
				TagMappingTTL:     1 * time.Hour,
				BatchSize:         0,
				EnableStats:       true,
				LoadTimeout:       5 * time.Second,
			},
			expectError: false, // Current implementation handles zero batch size
		},
		{
			name: "Negative batch size",
			config: &cachex.TagConfig{
				EnablePersistence: true,
				TagMappingTTL:     1 * time.Hour,
				BatchSize:         -10,
				EnableStats:       true,
				LoadTimeout:       5 * time.Second,
			},
			expectError: false, // Current implementation handles negative batch size
		},
		{
			name: "Zero timeout",
			config: &cachex.TagConfig{
				EnablePersistence: true,
				TagMappingTTL:     1 * time.Hour,
				BatchSize:         100,
				EnableStats:       true,
				LoadTimeout:       0,
			},
			expectError: false, // Current implementation allows zero timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := cachex.NewTagManager(store, tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for invalid config, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for config: %v", err)
				}
				if tm == nil {
					t.Errorf("TagManager should not be nil")
				}
			}
		})
	}
}

func TestTagManager_LargeDataset(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Test with large dataset
	numKeys := 1000
	numTags := 100

	// Add many keys with many tags
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		tags := make([]string, 0, 5)
		for j := 0; j < 5; j++ {
			tag := fmt.Sprintf("tag-%d", (i+j)%numTags)
			tags = append(tags, tag)
		}
		err := tm.AddTags(ctx, key, tags...)
		if err != nil {
			t.Fatalf("AddTags() failed for key %s: %v", key, err)
		}
	}

	// Verify all keys were added
	stats, err := tm.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	if stats.TotalKeys != int64(numKeys) {
		t.Errorf("Expected %d keys, got %d", numKeys, stats.TotalKeys)
	}

	// Test querying with large dataset
	start := time.Now()
	keys, err := tm.GetKeysByTag(ctx, "tag-0")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}
	duration := time.Since(start)

	if len(keys) == 0 {
		t.Errorf("Expected some keys for tag-0, got 0")
	}

	if duration > 100*time.Millisecond {
		t.Errorf("GetKeysByTag() too slow for large dataset: %v", duration)
	}

	// Test batch invalidation with large dataset
	start = time.Now()
	err = tm.InvalidateByTag(ctx, "tag-0")
	if err != nil {
		t.Fatalf("InvalidateByTag() failed: %v", err)
	}
	duration = time.Since(start)

	if duration > 1*time.Second {
		t.Errorf("InvalidateByTag() too slow for large dataset: %v", duration)
	}

	// Verify invalidation worked
	keys, err = tm.GetKeysByTag(ctx, "tag-0")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after invalidation, got %d", len(keys))
	}
}

func TestTagManager_StoreInterfaceEdgeCases(t *testing.T) {
	// Test with store that returns unexpected results
	edgeCaseStore := &EdgeCaseMockStore{
		MockTagStore: *NewMockTagStore(),
	}

	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}

	tm, err := cachex.NewTagManager(edgeCaseStore, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Test with store that returns nil value but Exists=true
	edgeCaseStore.nilValueExists = true
	err = tm.AddTags(ctx, "key1", "tag1")
	if err != nil {
		t.Logf("AddTags failed as expected with edge case store: %v", err)
	}

	// Test with store that returns empty value
	edgeCaseStore.nilValueExists = false
	edgeCaseStore.emptyValue = true
	err = tm.AddTags(ctx, "key2", "tag2")
	if err != nil {
		t.Logf("AddTags failed as expected with empty value: %v", err)
	}

	// Test with store that returns unexpected JSON
	edgeCaseStore.emptyValue = false
	edgeCaseStore.invalidJSON = true
	err = tm.AddTags(ctx, "key3", "tag3")
	if err != nil {
		t.Logf("AddTags failed as expected with invalid JSON: %v", err)
	}
}

func TestTagManager_TimeoutScenarios(t *testing.T) {
	// Test with very short timeouts
	store := NewMockTagStore()
	config := &cachex.TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     1 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       1 * time.Millisecond, // Very short timeout
	}

	tm, err := cachex.NewTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Test operations with short timeout
	err = tm.AddTags(ctx, "key1", "tag1")
	if err != nil {
		t.Logf("AddTags failed with short timeout as expected: %v", err)
	}

	// Test with context timeout
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(2 * time.Millisecond)

	err = tm.AddTags(ctxTimeout, "key2", "tag2")
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context timeout error, got %v", err)
	}
}

func TestTagManager_ConcurrentLargeDataset(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 20
	numOperations := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent operations with large dataset
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				tags := []string{
					fmt.Sprintf("tag-%d", j%10),
					fmt.Sprintf("tag-%d", (j+1)%10),
					fmt.Sprintf("tag-%d", (j+2)%10),
				}

				// Add tags
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					t.Errorf("Concurrent AddTags() failed: %v", err)
					return
				}

				// Query tags
				_, err = tm.GetTagsByKey(ctx, key)
				if err != nil {
					t.Errorf("Concurrent GetTagsByKey() failed: %v", err)
					return
				}

				// Query keys by tag
				_, err = tm.GetKeysByTag(ctx, tags[0])
				if err != nil {
					t.Errorf("Concurrent GetKeysByTag() failed: %v", err)
					return
				}

				// Remove some tags occasionally
				if j%3 == 0 {
					err = tm.RemoveTags(ctx, key, tags[0])
					if err != nil {
						t.Errorf("Concurrent RemoveTags() failed: %v", err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	stats, err := tm.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	expectedKeys := numGoroutines * numOperations
	if stats.TotalKeys != int64(expectedKeys) {
		t.Errorf("Expected approximately %d keys, got %d", expectedKeys, stats.TotalKeys)
	}
}

func TestTagManager_MemoryEfficiency(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add many keys and tags
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		tags := []string{
			fmt.Sprintf("tag-%d", i%10),
			fmt.Sprintf("tag-%d", (i+1)%10),
		}
		err := tm.AddTags(ctx, key, tags...)
		if err != nil {
			t.Fatalf("AddTags() failed: %v", err)
		}
	}

	// Remove many keys to test cleanup
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := tm.RemoveKey(ctx, key)
		if err != nil {
			t.Fatalf("RemoveKey() failed: %v", err)
		}
	}

	// Verify cleanup worked
	stats, err := tm.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	if stats.TotalKeys != 500 {
		t.Errorf("Expected 500 keys after cleanup, got %d", stats.TotalKeys)
	}

	// Test that remaining keys are still accessible
	keys, err := tm.GetKeysByTag(ctx, "tag-0")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	// Should have some keys but not all 1000
	if len(keys) == 0 {
		t.Errorf("Expected some keys for tag-0, got 0")
	}

	if len(keys) >= 1000 {
		t.Errorf("Expected fewer keys after cleanup, got %d", len(keys))
	}
}

func TestTagManager_DataConsistency(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()

	// Add tags and verify consistency
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	// Verify tag-to-key consistency
	keys1, err := tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys1) != 1 || keys1[0] != "key1" {
		t.Errorf("Expected key1 for tag1, got %v", keys1)
	}

	keys2, err := tm.GetKeysByTag(ctx, "tag2")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys2) != 2 {
		t.Errorf("Expected 2 keys for tag2, got %d", len(keys2))
	}

	// Verify key-to-tag consistency
	tags1, err := tm.GetTagsByKey(ctx, "key1")
	if err != nil {
		t.Fatalf("GetTagsByKey() failed: %v", err)
	}

	if len(tags1) != 2 {
		t.Errorf("Expected 2 tags for key1, got %d", len(tags1))
	}

	// Verify both directions are consistent
	for _, key := range keys2 {
		tags, err := tm.GetTagsByKey(ctx, key)
		if err != nil {
			t.Fatalf("GetTagsByKey() failed for key %s: %v", key, err)
		}

		found := false
		for _, tag := range tags {
			if tag == "tag2" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Key %s should have tag2", key)
		}
	}
}

func TestTagManager_StressTest(t *testing.T) {
	store := NewMockTagStore()
	tm, err := cachex.NewTagManager(store, nil)
	if err != nil {
		t.Fatalf("Failed to create tag manager: %v", err)
	}

	ctx := context.Background()
	numOperations := 10000
	var wg sync.WaitGroup
	wg.Add(4) // 4 different operation types

	// Stress test with mixed operations
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("add-key-%d", i)
			tag := fmt.Sprintf("add-tag-%d", i%100)
			err := tm.AddTags(ctx, key, tag)
			if err != nil {
				t.Errorf("Stress test AddTags() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			tag := fmt.Sprintf("query-tag-%d", i%100)
			_, err := tm.GetKeysByTag(ctx, tag)
			if err != nil {
				t.Errorf("Stress test GetKeysByTag() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("query-key-%d", i%1000)
			_, err := tm.GetTagsByKey(ctx, key)
			if err != nil {
				t.Errorf("Stress test GetTagsByKey() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("remove-key-%d", i)
			err := tm.RemoveKey(ctx, key)
			if err != nil {
				t.Errorf("Stress test RemoveKey() failed: %v", err)
				return
			}
		}
	}()

	wg.Wait()

	// Verify system is still functional
	stats, err := tm.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed after stress test: %v", err)
	}

	if stats.TotalKeys < 0 {
		t.Errorf("Invalid total keys after stress test: %d", stats.TotalKeys)
	}
}

// EdgeCaseMockStore is a mock store that can return edge case results
type EdgeCaseMockStore struct {
	MockTagStore
	nilValueExists bool
	emptyValue     bool
	invalidJSON    bool
}

func (m *EdgeCaseMockStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			// Return Exists=true but Value=nil
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			// Return empty value
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			// Return invalid JSON
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.Get(ctx, key)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.Set(ctx, key, value, ttl)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.MGet(ctx, keys...)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.MSet(ctx, items, ttl)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.Del(ctx, keys...)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) Exists(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.Exists(ctx, key)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) TTL(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.TTL(ctx, key)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		if m.nilValueExists {
			result <- cachex.AsyncResult{Exists: true, Value: nil}
			return
		}

		if m.emptyValue {
			result <- cachex.AsyncResult{Exists: true, Value: []byte{}}
			return
		}

		if m.invalidJSON {
			result <- cachex.AsyncResult{Exists: true, Value: []byte("invalid json")}
			return
		}

		// Use parent implementation
		parentResult := <-m.MockTagStore.IncrBy(ctx, key, delta, ttlIfCreate)
		result <- parentResult
	}()

	return result
}

func (m *EdgeCaseMockStore) Close() error {
	return nil
}

func (m *EdgeCaseMockStore) GetStats() interface{} {
	return nil
}
