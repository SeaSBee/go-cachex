package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore implements the Store interface for testing
type mockStore struct {
	data map[string][]byte
	ttl  map[string]time.Duration
	mu   sync.RWMutex
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Duration),
	}
}

func (m *mockStore) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	return nil, nil
}

func (m *mockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	m.ttl[key] = ttl
	return nil
}

func (m *mockStore) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]byte)
	for _, key := range keys {
		if data, exists := m.data[key]; exists {
			result[key] = data
		}
	}
	return result, nil
}

func (m *mockStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, value := range items {
		m.data[key] = value
		m.ttl[key] = ttl
	}
	return nil
}

func (m *mockStore) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.data, key)
		delete(m.ttl, key)
	}
	return nil
}

func (m *mockStore) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ttl, exists := m.ttl[key]; exists {
		return ttl, nil
	}
	return 0, nil
}

func (m *mockStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) (int64, error) {
	// Simplified implementation for testing
	return delta, nil
}

func (m *mockStore) Close() error {
	return nil
}

// mockCodec implements the Codec interface for testing
type mockCodec struct{}

func (c *mockCodec) Encode(v any) ([]byte, error) {
	// Simple encoding for testing - just convert to string
	if str, ok := v.(string); ok {
		return []byte(str), nil
	}
	return []byte("encoded"), nil
}

func (c *mockCodec) Decode(data []byte, v any) error {
	// Simple decoding for testing - just convert from string
	if str, ok := v.(*string); ok {
		*str = string(data)
	}
	return nil
}

func TestNew(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
		WithDefaultTTL(5*time.Minute),
	)

	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestGetSet(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	key := "test-key"
	value := "test-value"

	// Test Set
	err = c.Set(key, value, 10*time.Minute)
	assert.NoError(t, err)

	// Test Get
	result, found, err := c.Get(key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, result)
}

func TestMGetMSet(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// Test MSet
	err = c.MSet(items, 10*time.Minute)
	assert.NoError(t, err)

	// Test MGet
	keys := []string{"key1", "key2"}
	result, err := c.MGet(keys...)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestDel(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	key := "test-key"
	value := "test-value"

	// Set a value
	err = c.Set(key, value, 10*time.Minute)
	assert.NoError(t, err)

	// Verify it exists
	exists, err := c.Exists(key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = c.Del(key)
	assert.NoError(t, err)

	// Verify it's gone
	exists, err = c.Exists(key)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestReadThrough(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	key := "test-key"
	expectedValue := "loaded-value"

	// Test ReadThrough with loader function
	result, err := c.ReadThrough(key, 10*time.Minute, func(ctx context.Context) (string, error) {
		return expectedValue, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, result)
}

func TestTryLock(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	key := "test-lock"

	// Try to acquire lock
	unlock, acquired, err := c.TryLock(key, 30*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)
	assert.NotNil(t, unlock)

	// Release lock
	err = unlock()
	assert.NoError(t, err)
}

func TestWithContext(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	type contextKey string
	const testKey contextKey = "test"
	ctx := context.WithValue(context.Background(), testKey, "value")
	cached := c.WithContext(ctx)

	assert.NotNil(t, cached)
}

func TestClose(t *testing.T) {
	store := newMockStore()
	codec := &mockCodec{}

	c, err := New[string](
		WithStore(store),
		WithCodec(codec),
	)
	require.NoError(t, err)

	// Close should not error
	err = c.Close()
	assert.NoError(t, err)
}
