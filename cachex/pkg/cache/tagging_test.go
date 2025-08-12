package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTagManager_BasicOperations(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Test adding tags to a key
	err := tm.AddTags(ctx, "user:1", "premium", "active")
	assert.NoError(t, err)

	// Verify tags were added
	tags, err := tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"premium", "active"}, tags)

	// Test adding more tags
	err = tm.AddTags(ctx, "user:1", "verified")
	assert.NoError(t, err)

	tags, err = tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"premium", "active", "verified"}, tags)
}

func TestTagManager_RemoveTags(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Add tags first
	err := tm.AddTags(ctx, "user:1", "premium", "active", "verified")
	assert.NoError(t, err)

	// Remove some tags
	err = tm.RemoveTags(ctx, "user:1", "premium", "verified")
	assert.NoError(t, err)

	// Verify remaining tags
	tags, err := tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"active"}, tags)

	// Remove all tags
	err = tm.RemoveTags(ctx, "user:1", "active")
	assert.NoError(t, err)

	tags, err = tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagManager_GetKeysByTag(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Add tags to multiple keys
	err := tm.AddTags(ctx, "user:1", "premium", "active")
	assert.NoError(t, err)

	err = tm.AddTags(ctx, "user:2", "premium", "verified")
	assert.NoError(t, err)

	err = tm.AddTags(ctx, "user:3", "standard", "active")
	assert.NoError(t, err)

	// Get keys by tag
	premiumKeys, err := tm.GetKeysByTag(ctx, "premium")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:1", "user:2"}, premiumKeys)

	activeKeys, err := tm.GetKeysByTag(ctx, "active")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:1", "user:3"}, activeKeys)

	verifiedKeys, err := tm.GetKeysByTag(ctx, "verified")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:2"}, verifiedKeys)

	// Test non-existent tag
	nonExistentKeys, err := tm.GetKeysByTag(ctx, "non-existent")
	assert.NoError(t, err)
	assert.Empty(t, nonExistentKeys)
}

func TestTagManager_InvalidateByTag(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Store some data
	err := store.Set(ctx, "user:1", []byte("data1"), 5*time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "user:2", []byte("data2"), 5*time.Minute)
	assert.NoError(t, err)
	err = store.Set(ctx, "user:3", []byte("data3"), 5*time.Minute)
	assert.NoError(t, err)

	// Add tags
	err = tm.AddTags(ctx, "user:1", "premium", "active")
	assert.NoError(t, err)
	err = tm.AddTags(ctx, "user:2", "premium", "verified")
	assert.NoError(t, err)
	err = tm.AddTags(ctx, "user:3", "standard", "active")
	assert.NoError(t, err)

	// Invalidate by tag
	err = tm.InvalidateByTag(ctx, []string{"user:1", "user:2"})
	assert.NoError(t, err)

	// Verify data was deleted
	data1, err := store.Get(ctx, "user:1")
	assert.NoError(t, err)
	assert.Nil(t, data1)

	data2, err := store.Get(ctx, "user:2")
	assert.NoError(t, err)
	assert.Nil(t, data2)

	data3, err := store.Get(ctx, "user:3")
	assert.NoError(t, err)
	assert.NotNil(t, data3) // Should still exist

	// Verify tag mappings were cleaned up
	tags1, err := tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.Empty(t, tags1)

	tags2, err := tm.GetTagsByKey(ctx, "user:2")
	assert.NoError(t, err)
	assert.Empty(t, tags2)
}

func TestTagManager_RemoveKey(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Add tags to a key
	err := tm.AddTags(ctx, "user:1", "premium", "active", "verified")
	assert.NoError(t, err)

	// Remove the key
	err = tm.RemoveKey(ctx, "user:1")
	assert.NoError(t, err)

	// Verify tags were removed
	tags, err := tm.GetTagsByKey(ctx, "user:1")
	assert.NoError(t, err)
	assert.Empty(t, tags)

	// Verify key is not in tag mappings
	keys, err := tm.GetKeysByTag(ctx, "premium")
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestTagManager_GetStats(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Add some tags
	err := tm.AddTags(ctx, "user:1", "premium", "active")
	assert.NoError(t, err)
	err = tm.AddTags(ctx, "user:2", "premium", "verified")
	assert.NoError(t, err)
	err = tm.AddTags(ctx, "user:3", "standard")
	assert.NoError(t, err)

	// Get stats
	stats := tm.GetStats()
	assert.Equal(t, int64(4), stats.TotalTags) // premium, active, verified, standard
	assert.Equal(t, int64(3), stats.TotalKeys) // user:1, user:2, user:3
}

func TestTagManager_EmptyOperations(t *testing.T) {
	store := &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
	tm := NewTagManager(store, DefaultTagConfig())

	ctx := context.Background()

	// Test empty operations
	err := tm.AddTags(ctx, "user:1")
	assert.NoError(t, err)

	err = tm.RemoveTags(ctx, "user:1")
	assert.NoError(t, err)

	err = tm.InvalidateByTag(ctx, []string{})
	assert.NoError(t, err)

	err = tm.RemoveKey(ctx, "non-existent")
	assert.NoError(t, err)
}
