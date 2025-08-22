package unit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
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
	tm := cachex.NewTagManager(store, nil)

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

	tm := cachex.NewTagManager(store, config)

	if tm == nil {
		t.Errorf("NewTagManager() should not return nil")
	}
}

func TestTagManager_AddTags_EmptyTags(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	err := tm.AddTags(ctx, "test-key")
	if err != nil {
		t.Errorf("AddTags() should not fail with empty tags: %v", err)
	}
}

func TestTagManager_AddTags_SingleTag(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	err := tm.AddTags(ctx, key, tag)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	err := tm.AddTags(ctx, key, tags...)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()

	// Add tags to multiple keys
	err := tm.AddTags(ctx, "key1", "tag1", "tag2")
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	err := tm.RemoveTags(ctx, "test-key")
	if err != nil {
		t.Errorf("RemoveTags() should not fail with empty tags: %v", err)
	}
}

func TestTagManager_RemoveTags_SingleTag(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	// Add tag first
	err := tm.AddTags(ctx, key, tag)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags first
	err := tm.AddTags(ctx, key, tags...)
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
	tm := cachex.NewTagManager(store, nil)

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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	tag := "test-tag"

	// Add multiple keys with the same tag
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		err := tm.AddTags(ctx, key, tag)
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
	tm := cachex.NewTagManager(store, nil)

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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags to key
	err := tm.AddTags(ctx, key, tags...)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	err := tm.InvalidateByTag(ctx, []string{})
	if err != nil {
		t.Errorf("InvalidateByTag() should not fail with empty keys: %v", err)
	}
}

func TestTagManager_InvalidateByTag_SingleKey(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tag := "test-tag"

	// Add some data to store
	setResult := <-store.Set(ctx, key, []byte("test-value"), 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Store.Set() failed: %v", setResult.Error)
	}

	// Add tag to key
	if err := tm.AddTags(ctx, key, tag); err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Invalidate by tag
	if err := tm.InvalidateByTag(ctx, []string{key}); err != nil {
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	tag := "test-tag"

	// Add data and tags
	for _, key := range keys {
		setResult := <-store.Set(ctx, key, []byte("test-value"), 5*time.Minute)
		if setResult.Error != nil {
			t.Errorf("Store.Set() failed for key %s: %v", key, setResult.Error)
		}

		err := tm.AddTags(ctx, key, tag)
		if err != nil {
			t.Errorf("AddTags() failed for key %s: %v", key, err)
		}
	}

	// Invalidate by tag
	err := tm.InvalidateByTag(ctx, keys)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	err := tm.RemoveKey(ctx, "non-existent-key")
	if err != nil {
		t.Errorf("RemoveKey() should not fail for non-existent key: %v", err)
	}
}

func TestTagManager_RemoveKey_ExistingKey(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()
	key := "test-key"
	tags := []string{"tag1", "tag2", "tag3"}

	// Add tags to key
	err := tm.AddTags(ctx, key, tags...)
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
	tm := cachex.NewTagManager(store, nil)

	ctx := context.Background()

	// Add some tags and keys
	err := tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Get stats
	stats := tm.GetStats()
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
	tm := cachex.NewTagManager(store, nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test operations with cancelled context
	// Note: TagManager doesn't check context in read operations, so we only test write operations

	err := tm.AddTags(ctx, "test-key", "test-tag")
	if err != context.Canceled {
		t.Errorf("AddTags() should return context.Canceled, got %v", err)
	}

	err = tm.RemoveTags(ctx, "test-key", "test-tag")
	if err != context.Canceled {
		t.Errorf("RemoveTags() should return context.Canceled, got %v", err)
	}

	err = tm.RemoveKey(ctx, "test-key")
	if err != context.Canceled {
		t.Errorf("RemoveKey() should return context.Canceled, got %v", err)
	}

	err = tm.InvalidateByTag(ctx, []string{"test-key"})
	if err != context.Canceled {
		t.Errorf("InvalidateByTag() should return context.Canceled, got %v", err)
	}
}

func TestTagManager_Concurrency(t *testing.T) {
	store := NewMockTagStore()
	tm := cachex.NewTagManager(store, nil)

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
				err := tm.AddTags(ctx, key, tag)
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
