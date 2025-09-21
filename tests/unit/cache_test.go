package unit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seasbee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCache implements the Cache interface for testing
type MockCache struct {
	mu           sync.RWMutex
	data         map[string]interface{}
	errors       map[string]error
	closed       bool
	operationLog []string
}

func NewMockCache() *MockCache {
	return &MockCache{
		data:         make(map[string]interface{}),
		errors:       make(map[string]error),
		operationLog: make([]string, 0),
	}
}

func (m *MockCache) WithContext(ctx context.Context) cachex.Cache {
	return m
}

func (m *MockCache) Get(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.RLock()
		defer m.mu.RUnlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("Get:%s", key))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		if err, exists := m.errors[key]; exists {
			result <- cachex.AsyncCacheResult[any]{Error: err}
			return
		}

		if val, exists := m.data[key]; exists {
			result <- cachex.AsyncCacheResult[any]{Value: val, Found: true}
		} else {
			result <- cachex.AsyncCacheResult[any]{Found: false}
		}
	}()

	return result
}

func (m *MockCache) Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.Lock()
		defer m.mu.Unlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("Set:%s", key))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		if err, exists := m.errors[key]; exists {
			result <- cachex.AsyncCacheResult[any]{Error: err}
			return
		}

		m.data[key] = val
		result <- cachex.AsyncCacheResult[any]{Value: val}
	}()

	return result
}

func (m *MockCache) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.RLock()
		defer m.mu.RUnlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("MGet:%v", keys))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		values := make(map[string]any)
		for _, key := range keys {
			if err, exists := m.errors[key]; exists {
				result <- cachex.AsyncCacheResult[any]{Error: err}
				return
			}
			if val, exists := m.data[key]; exists {
				values[key] = val
			}
		}

		result <- cachex.AsyncCacheResult[any]{Values: values}
	}()

	return result
}

func (m *MockCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.Lock()
		defer m.mu.Unlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("MSet:%v", items))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		for key, val := range items {
			if err, exists := m.errors[key]; exists {
				result <- cachex.AsyncCacheResult[any]{Error: err}
				return
			}
			m.data[key] = val
		}

		result <- cachex.AsyncCacheResult[any]{}
	}()

	return result
}

func (m *MockCache) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.Lock()
		defer m.mu.Unlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("Del:%v", keys))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		count := int64(0)
		for _, key := range keys {
			if _, exists := m.data[key]; exists {
				delete(m.data, key)
				count++
			}
		}

		result <- cachex.AsyncCacheResult[any]{Count: count}
	}()

	return result
}

func (m *MockCache) Exists(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.RLock()
		defer m.mu.RUnlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("Exists:%s", key))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		_, exists := m.data[key]
		result <- cachex.AsyncCacheResult[any]{Found: exists}
	}()

	return result
}

func (m *MockCache) TTL(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.RLock()
		defer m.mu.RUnlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("TTL:%s", key))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		// Mock TTL - return 1 hour for existing keys
		if _, exists := m.data[key]; exists {
			result <- cachex.AsyncCacheResult[any]{TTL: time.Hour}
		} else {
			result <- cachex.AsyncCacheResult[any]{TTL: -1}
		}
	}()

	return result
}

func (m *MockCache) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncCacheResult[any] {
	result := make(chan cachex.AsyncCacheResult[any], 1)

	go func() {
		defer close(result)
		m.mu.Lock()
		defer m.mu.Unlock()

		m.operationLog = append(m.operationLog, fmt.Sprintf("IncrBy:%s:%d", key, delta))

		if m.closed {
			result <- cachex.AsyncCacheResult[any]{Error: cachex.ErrStoreClosed}
			return
		}

		if err, exists := m.errors[key]; exists {
			result <- cachex.AsyncCacheResult[any]{Error: err}
			return
		}

		var current int64
		if val, exists := m.data[key]; exists {
			if intVal, ok := val.(int64); ok {
				current = intVal
			}
		}

		newVal := current + delta
		m.data[key] = newVal
		result <- cachex.AsyncCacheResult[any]{Int: newVal}
	}()

	return result
}

func (m *MockCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

// Helper methods for testing
func (m *MockCache) SetError(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[key] = err
}

func (m *MockCache) SetData(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MockCache) GetOperationLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.operationLog...)
}

func (m *MockCache) ClearOperationLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operationLog = make([]string, 0)
}

// TestAsyncCacheResult tests the AsyncCacheResult struct
func TestAsyncCacheResult(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		result := cachex.AsyncCacheResult[string]{}

		assert.Equal(t, "", result.Value)
		assert.Nil(t, result.Values)
		assert.False(t, result.Found)
		assert.Equal(t, time.Duration(0), result.TTL)
		assert.Equal(t, int64(0), result.Int)
		assert.Equal(t, int64(0), result.Count)
		assert.Nil(t, result.Keys)
		assert.Equal(t, int64(0), result.Size)
		assert.Nil(t, result.Error)
		assert.Nil(t, result.Metadata)
	})

	t.Run("with values", func(t *testing.T) {
		value := "test-value"
		values := map[string]string{"key1": "value1", "key2": "value2"}
		keys := []string{"key1", "key2"}
		metadata := map[string]interface{}{"source": "test"}
		err := errors.New("test error")

		result := cachex.AsyncCacheResult[string]{
			Value:    value,
			Values:   values,
			Found:    true,
			TTL:      time.Hour,
			Int:      42,
			Count:    2,
			Keys:     keys,
			Size:     1024,
			Error:    err,
			Metadata: metadata,
		}

		assert.Equal(t, value, result.Value)
		assert.Equal(t, values, result.Values)
		assert.True(t, result.Found)
		assert.Equal(t, time.Hour, result.TTL)
		assert.Equal(t, int64(42), result.Int)
		assert.Equal(t, int64(2), result.Count)
		assert.Equal(t, keys, result.Keys)
		assert.Equal(t, int64(1024), result.Size)
		assert.Equal(t, err, result.Error)
		assert.Equal(t, metadata, result.Metadata)
	})
}

// TestNewTypedCache tests the NewTypedCache constructor
func TestNewTypedCache(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		assert.NotNil(t, typedCache)
		// Note: We can't access the internal cache field directly due to encapsulation
		// The test verifies that the typed cache is created successfully
	})

	t.Run("with different types", func(t *testing.T) {
		mockCache := NewMockCache()

		// Test with string type
		stringCache := cachex.NewTypedCache[string](mockCache)
		assert.NotNil(t, stringCache)

		// Test with int type
		intCache := cachex.NewTypedCache[int](mockCache)
		assert.NotNil(t, intCache)

		// Test with struct type
		type TestStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		structCache := cachex.NewTypedCache[TestStruct](mockCache)
		assert.NotNil(t, structCache)
	})
}

// TestTypedCache_WithContext tests the WithContext method
func TestTypedCache_WithContext(t *testing.T) {
	t.Run("with valid context", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		type contextKey string
		ctx := context.WithValue(context.Background(), contextKey("test"), "value")
		newTypedCache := typedCache.WithContext(ctx)

		assert.NotNil(t, newTypedCache)
		// Note: WithContext creates a new instance, but we can't easily test this
		// without accessing internal fields. The test verifies the method works.
	})

	t.Run("with nil context", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		newTypedCache := typedCache.WithContext(context.TODO())

		assert.NotNil(t, newTypedCache)
		// Note: WithContext creates a new instance, but we can't easily test this
		// without accessing internal fields. The test verifies the method works.
	})
}

// TestTypedCache_Get tests the Get method
func TestTypedCache_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		expectedValue := "test-value"
		mockCache.SetData("test-key", expectedValue)

		ctx := context.Background()
		resultChan := typedCache.Get(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.True(t, result.Found)
			assert.Equal(t, expectedValue, result.Value)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Get(ctx, "non-existent-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.False(t, result.Found)
			assert.Equal(t, "", result.Value)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("type assertion failure", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up data with wrong type
		mockCache.SetData("test-key", 123) // int instead of string

		ctx := context.Background()
		resultChan := typedCache.Get(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.False(t, result.Found)
			assert.Contains(t, result.Error.Error(), "type assertion failed")
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("cache error", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up error
		expectedError := errors.New("cache error")
		mockCache.SetError("test-key", expectedError)

		ctx := context.Background()
		resultChan := typedCache.Get(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, expectedError, result.Error)
			assert.False(t, result.Found)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.Get(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
			assert.False(t, result.Found)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("with different types", func(t *testing.T) {
		mockCache := NewMockCache()

		// Test with int type
		intCache := cachex.NewTypedCache[int](mockCache)
		mockCache.SetData("int-key", 42) // Use int instead of int64

		ctx := context.Background()
		resultChan := intCache.Get(ctx, "int-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.True(t, result.Found)
			assert.Equal(t, 42, result.Value)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})
}

// TestTypedCache_Set tests the Set method
func TestTypedCache_Set(t *testing.T) {
	t.Run("successful set", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		value := "test-value"
		ttl := time.Hour

		resultChan := typedCache.Set(ctx, "test-key", value, ttl)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, value, result.Value)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Verify the value was stored
		assert.Equal(t, value, mockCache.data["test-key"])
	})

	t.Run("cache error", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up error
		expectedError := errors.New("cache error")
		mockCache.SetError("test-key", expectedError)

		ctx := context.Background()
		value := "test-value"
		ttl := time.Hour

		resultChan := typedCache.Set(ctx, "test-key", value, ttl)

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, expectedError, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		value := "test-value"
		ttl := time.Hour

		resultChan := typedCache.Set(ctx, "test-key", value, ttl)

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})

	t.Run("with different types", func(t *testing.T) {
		mockCache := NewMockCache()

		// Test with int type
		intCache := cachex.NewTypedCache[int](mockCache)
		value := 42
		ttl := time.Hour

		ctx := context.Background()
		resultChan := intCache.Set(ctx, "int-key", value, ttl)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, value, result.Value)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Verify the value was stored
		assert.Equal(t, value, mockCache.data["int-key"])
	})
}

// TestTypedCache_MGet tests the MGet method
func TestTypedCache_MGet(t *testing.T) {
	t.Run("successful mget", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("key1", "value1")
		mockCache.SetData("key2", "value2")
		mockCache.SetData("key3", "value3")

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2", "key3")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.Values)
			assert.Len(t, result.Values, 3)
			assert.Equal(t, "value1", result.Values["key1"])
			assert.Equal(t, "value2", result.Values["key2"])
			assert.Equal(t, "value3", result.Values["key3"])
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("partial results", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data - only some keys exist
		mockCache.SetData("key1", "value1")
		mockCache.SetData("key3", "value3")

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2", "key3")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.Values)
			assert.Len(t, result.Values, 2) // Only 2 keys found
			assert.Equal(t, "value1", result.Values["key1"])
			assert.Equal(t, "value3", result.Values["key3"])
			assert.NotContains(t, result.Values, "key2")
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("no keys found", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2", "key3")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.Values)
			assert.Len(t, result.Values, 0)
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("empty keys list", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.Values)
			assert.Len(t, result.Values, 0)
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("type assertion failure", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up data with wrong types
		mockCache.SetData("key1", "value1")
		mockCache.SetData("key2", 123) // int instead of string
		mockCache.SetData("key3", "value3")

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2", "key3")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.Values)
			assert.Len(t, result.Values, 2) // Only 2 keys with correct type
			assert.Equal(t, "value1", result.Values["key1"])
			assert.Equal(t, "value3", result.Values["key3"])
			assert.NotContains(t, result.Values, "key2")
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("cache error", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up error
		expectedError := errors.New("cache error")
		mockCache.SetError("key1", expectedError)

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, expectedError, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.MGet(ctx, "key1", "key2")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MGet operation timed out")
		}
	})
}

// TestTypedCache_MSet tests the MSet method
func TestTypedCache_MSet(t *testing.T) {
	t.Run("successful mset", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		items := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		ttl := time.Hour

		resultChan := typedCache.MSet(ctx, items, ttl)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MSet operation timed out")
		}

		// Verify the values were stored
		assert.Equal(t, "value1", mockCache.data["key1"])
		assert.Equal(t, "value2", mockCache.data["key2"])
		assert.Equal(t, "value3", mockCache.data["key3"])
	})

	t.Run("empty items map", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		items := map[string]string{}
		ttl := time.Hour

		resultChan := typedCache.MSet(ctx, items, ttl)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MSet operation timed out")
		}
	})

	t.Run("cache error", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up error
		expectedError := errors.New("cache error")
		mockCache.SetError("key1", expectedError)

		ctx := context.Background()
		items := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		ttl := time.Hour

		resultChan := typedCache.MSet(ctx, items, ttl)

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, expectedError, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MSet operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		items := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		ttl := time.Hour

		resultChan := typedCache.MSet(ctx, items, ttl)

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MSet operation timed out")
		}
	})

	t.Run("with different types", func(t *testing.T) {
		mockCache := NewMockCache()

		// Test with int type
		intCache := cachex.NewTypedCache[int](mockCache)
		items := map[string]int{
			"key1": 1,
			"key2": 2,
			"key3": 3,
		}
		ttl := time.Hour

		ctx := context.Background()
		resultChan := intCache.MSet(ctx, items, ttl)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("MSet operation timed out")
		}

		// Verify the values were stored
		assert.Equal(t, 1, mockCache.data["key1"])
		assert.Equal(t, 2, mockCache.data["key2"])
		assert.Equal(t, 3, mockCache.data["key3"])
	})
}

// TestTypedCache_Del tests the Del method
func TestTypedCache_Del(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("key1", "value1")
		mockCache.SetData("key2", "value2")
		mockCache.SetData("key3", "value3")

		ctx := context.Background()
		resultChan := typedCache.Del(ctx, "key1", "key2")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(2), result.Count)
		case <-time.After(time.Second):
			t.Fatal("Del operation timed out")
		}

		// Verify the keys were deleted
		assert.NotContains(t, mockCache.data, "key1")
		assert.NotContains(t, mockCache.data, "key2")
		assert.Contains(t, mockCache.data, "key3") // Should still exist
	})

	t.Run("delete non-existent keys", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Del(ctx, "key1", "key2")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(0), result.Count)
		case <-time.After(time.Second):
			t.Fatal("Del operation timed out")
		}
	})

	t.Run("empty keys list", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Del(ctx)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(0), result.Count)
		case <-time.After(time.Second):
			t.Fatal("Del operation timed out")
		}
	})

	t.Run("partial delete", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data - only some keys exist
		mockCache.SetData("key1", "value1")
		mockCache.SetData("key3", "value3")

		ctx := context.Background()
		resultChan := typedCache.Del(ctx, "key1", "key2", "key3")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(2), result.Count) // Only 2 keys existed
		case <-time.After(time.Second):
			t.Fatal("Del operation timed out")
		}

		// Verify the existing keys were deleted
		assert.NotContains(t, mockCache.data, "key1")
		assert.NotContains(t, mockCache.data, "key3")
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.Del(ctx, "key1", "key2")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Del operation timed out")
		}
	})
}

// TestTypedCache_Exists tests the Exists method
func TestTypedCache_Exists(t *testing.T) {
	t.Run("key exists", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("test-key", "test-value")

		ctx := context.Background()
		resultChan := typedCache.Exists(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.True(t, result.Found)
		case <-time.After(time.Second):
			t.Fatal("Exists operation timed out")
		}
	})

	t.Run("key does not exist", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Exists(ctx, "non-existent-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.False(t, result.Found)
		case <-time.After(time.Second):
			t.Fatal("Exists operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.Exists(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Exists operation timed out")
		}
	})
}

// TestTypedCache_TTL tests the TTL method
func TestTypedCache_TTL(t *testing.T) {
	t.Run("key with TTL", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("test-key", "test-value")

		ctx := context.Background()
		resultChan := typedCache.TTL(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, time.Hour, result.TTL) // Mock returns 1 hour
		case <-time.After(time.Second):
			t.Fatal("TTL operation timed out")
		}
	})

	t.Run("key does not exist", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.TTL(ctx, "non-existent-key")

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, time.Duration(-1), result.TTL) // Mock returns -1 for non-existent keys
		case <-time.After(time.Second):
			t.Fatal("TTL operation timed out")
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.TTL(ctx, "test-key")

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("TTL operation timed out")
		}
	})
}

// TestTypedCache_IncrBy tests the IncrBy method
func TestTypedCache_IncrBy(t *testing.T) {
	t.Run("increment existing key", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("counter", int64(10))

		ctx := context.Background()
		resultChan := typedCache.IncrBy(ctx, "counter", 5, time.Hour)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(15), result.Int)
		case <-time.After(time.Second):
			t.Fatal("IncrBy operation timed out")
		}

		// Verify the value was updated
		assert.Equal(t, int64(15), mockCache.data["counter"])
	})

	t.Run("increment non-existent key", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.IncrBy(ctx, "new-counter", 5, time.Hour)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(5), result.Int)
		case <-time.After(time.Second):
			t.Fatal("IncrBy operation timed out")
		}

		// Verify the value was created
		assert.Equal(t, int64(5), mockCache.data["new-counter"])
	})

	t.Run("decrement existing key", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("counter", int64(10))

		ctx := context.Background()
		resultChan := typedCache.IncrBy(ctx, "counter", -3, time.Hour)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(7), result.Int)
		case <-time.After(time.Second):
			t.Fatal("IncrBy operation timed out")
		}

		// Verify the value was updated
		assert.Equal(t, int64(7), mockCache.data["counter"])
	})

	t.Run("zero increment", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Set up test data
		mockCache.SetData("counter", int64(10))

		ctx := context.Background()
		resultChan := typedCache.IncrBy(ctx, "counter", 0, time.Hour)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, int64(10), result.Int)
		case <-time.After(time.Second):
			t.Fatal("IncrBy operation timed out")
		}

		// Verify the value was unchanged
		assert.Equal(t, int64(10), mockCache.data["counter"])
	})

	t.Run("closed cache", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Close the cache
		mockCache.Close()

		ctx := context.Background()
		resultChan := typedCache.IncrBy(ctx, "counter", 5, time.Hour)

		select {
		case result := <-resultChan:
			assert.Error(t, result.Error)
			assert.Equal(t, cachex.ErrStoreClosed, result.Error)
		case <-time.After(time.Second):
			t.Fatal("IncrBy operation timed out")
		}
	})
}

// TestTypedCache_Close tests the Close method
func TestTypedCache_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		err := typedCache.Close()

		assert.NoError(t, err)
		assert.True(t, mockCache.closed)
	})

	t.Run("multiple close calls", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// First close
		err1 := typedCache.Close()
		assert.NoError(t, err1)

		// Second close should also succeed
		err2 := typedCache.Close()
		assert.NoError(t, err2)
	})
}

// TestCodecInterface tests the Codec interface
func TestCodecInterface(t *testing.T) {
	t.Run("interface definition", func(t *testing.T) {
		// Test that Codec interface is properly defined
		var codec cachex.Codec
		// Note: nil interface is expected, we're just testing the interface exists
		assert.Nil(t, codec) // This will be nil, which is expected for uninitialized interface

		// Test that we can assign a concrete implementation
		// This would be done with actual implementations like JSONCodec
		// For now, we just verify the interface exists
		_ = codec // Avoid unused variable warning
	})
}

// Note: Mock implementation of Codec would be added here if needed for specific tests

// TestKeyBuilderInterface tests the KeyBuilder interface
func TestKeyBuilderInterface(t *testing.T) {
	t.Run("interface definition", func(t *testing.T) {
		// Test that KeyBuilder interface is properly defined
		var keyBuilder cachex.KeyBuilder
		// Note: nil interface is expected, we're just testing the interface exists
		assert.Nil(t, keyBuilder) // This will be nil, which is expected for uninitialized interface

		// Test that we can assign a concrete implementation
		_ = keyBuilder // Avoid unused variable warning
	})

	t.Run("mock implementation", func(t *testing.T) {
		keyBuilder := &mockKeyBuilder{}

		// Test Build method
		key := keyBuilder.Build("user", "123")
		assert.Equal(t, "user:123", key)

		// Test BuildList method
		listKey := keyBuilder.BuildList("users", map[string]any{"status": "active"})
		assert.Equal(t, "list:users", listKey)

		// Test BuildComposite method
		compositeKey := keyBuilder.BuildComposite("user", "123", "post", "456")
		assert.Equal(t, "user:123:post:456", compositeKey)

		// Test BuildSession method
		sessionKey := keyBuilder.BuildSession("session123")
		assert.Equal(t, "session:session123", sessionKey)
	})
}

// Mock implementation of KeyBuilder for testing
type mockKeyBuilder struct{}

func (m *mockKeyBuilder) Build(entity, id string) string {
	return fmt.Sprintf("%s:%s", entity, id)
}

func (m *mockKeyBuilder) BuildList(entity string, filters map[string]any) string {
	return fmt.Sprintf("list:%s", entity)
}

func (m *mockKeyBuilder) BuildComposite(entityA, idA, entityB, idB string) string {
	return fmt.Sprintf("%s:%s:%s:%s", entityA, idA, entityB, idB)
}

func (m *mockKeyBuilder) BuildSession(sid string) string {
	return fmt.Sprintf("session:%s", sid)
}

// TestKeyHasherInterface tests the KeyHasher interface
func TestKeyHasherInterface(t *testing.T) {
	t.Run("interface definition", func(t *testing.T) {
		// Test that KeyHasher interface is properly defined
		var keyHasher cachex.KeyHasher
		// Note: nil interface is expected, we're just testing the interface exists
		assert.Nil(t, keyHasher) // This will be nil, which is expected for uninitialized interface

		// Test that we can assign a concrete implementation
		_ = keyHasher // Avoid unused variable warning
	})

	t.Run("mock implementation", func(t *testing.T) {
		keyHasher := &mockKeyHasher{}

		// Test Hash method
		hashed := keyHasher.Hash("test-key")
		assert.Equal(t, "hash_test-key", hashed)

		// Test with different inputs
		hashed2 := keyHasher.Hash("another-key")
		assert.Equal(t, "hash_another-key", hashed2)
	})
}

// Mock implementation of KeyHasher for testing
type mockKeyHasher struct{}

func (m *mockKeyHasher) Hash(data string) string {
	return fmt.Sprintf("hash_%s", data)
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		// Test that nil context is handled gracefully
		newTypedCache := typedCache.WithContext(context.TODO())
		assert.NotNil(t, newTypedCache)
	})

	t.Run("empty string values", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()

		// Test setting empty string
		resultChan := typedCache.Set(ctx, "empty-key", "", time.Hour)
		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.Equal(t, "", result.Value)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Test getting empty string
		resultChan = typedCache.Get(ctx, "empty-key")
		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
			assert.True(t, result.Found)
			assert.Equal(t, "", result.Value)
		case <-time.After(time.Second):
			t.Fatal("Get operation timed out")
		}
	})

	t.Run("zero TTL", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Set(ctx, "zero-ttl-key", "value", 0)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})

	t.Run("negative TTL", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		resultChan := typedCache.Set(ctx, "negative-ttl-key", "value", -time.Hour)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})

	t.Run("very large TTL", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		largeTTL := time.Duration(1<<63 - 1) // Max duration
		resultChan := typedCache.Set(ctx, "large-ttl-key", "value", largeTTL)

		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)

		ctx := context.Background()
		numGoroutines := 10
		var wg sync.WaitGroup

		// Concurrent sets
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("concurrent-key-%d", i)
				value := fmt.Sprintf("value-%d", i)

				resultChan := typedCache.Set(ctx, key, value, time.Hour)
				select {
				case result := <-resultChan:
					assert.NoError(t, result.Error)
				case <-time.After(time.Second):
					t.Errorf("Set operation timed out for key %s", key)
				}
			}(i)
		}

		wg.Wait()

		// Verify all values were set
		for i := 0; i < numGoroutines; i++ {
			key := fmt.Sprintf("concurrent-key-%d", i)
			expectedValue := fmt.Sprintf("value-%d", i)

			resultChan := typedCache.Get(ctx, key)
			select {
			case result := <-resultChan:
				assert.NoError(t, result.Error)
				assert.True(t, result.Found)
				assert.Equal(t, expectedValue, result.Value)
			case <-time.After(time.Second):
				t.Errorf("Get operation timed out for key %s", key)
			}
		}
	})

	t.Run("type safety with different types", func(t *testing.T) {
		mockCache := NewMockCache()

		// Test with string type
		stringCache := cachex.NewTypedCache[string](mockCache)
		ctx := context.Background()

		resultChan := stringCache.Set(ctx, "string-key", "string-value", time.Hour)
		select {
		case result := <-resultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Test with int type
		intCache := cachex.NewTypedCache[int](mockCache)
		intResultChan := intCache.Set(ctx, "int-key", 42, time.Hour)
		select {
		case result := <-intResultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Test with bool type
		boolCache := cachex.NewTypedCache[bool](mockCache)
		boolResultChan := boolCache.Set(ctx, "bool-key", true, time.Hour)
		select {
		case result := <-boolResultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}

		// Test with struct type
		type TestStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		structCache := cachex.NewTypedCache[TestStruct](mockCache)
		testStruct := TestStruct{ID: 1, Name: "test"}
		structResultChan := structCache.Set(ctx, "struct-key", testStruct, time.Hour)
		select {
		case result := <-structResultChan:
			assert.NoError(t, result.Error)
		case <-time.After(time.Second):
			t.Fatal("Set operation timed out")
		}
	})
}

// Benchmark tests for performance measurement
func BenchmarkNewTypedCache(b *testing.B) {
	mockCache := NewMockCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cachex.NewTypedCache[string](mockCache)
	}
}

func BenchmarkTypedCache_WithContext(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = typedCache.WithContext(ctx)
	}
}

func BenchmarkTypedCache_Get(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	// Set up test data
	mockCache.SetData("bench-key", "bench-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.Get(ctx, "bench-key")
		<-resultChan
	}
}

func BenchmarkTypedCache_Set(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()
	value := "bench-value"
	ttl := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		resultChan := typedCache.Set(ctx, key, value, ttl)
		<-resultChan
	}
}

func BenchmarkTypedCache_MGet(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	// Set up test data
	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		keys[i] = key
		mockCache.SetData(key, fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.MGet(ctx, keys...)
		<-resultChan
	}
}

func BenchmarkTypedCache_MSet(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()
	ttl := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := make(map[string]string)
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("bench-key-%d-%d", i, j)
			items[key] = fmt.Sprintf("value-%d-%d", i, j)
		}
		resultChan := typedCache.MSet(ctx, items, ttl)
		<-resultChan
	}
}

func BenchmarkTypedCache_Del(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	// Set up test data
	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		keys[i] = key
		mockCache.SetData(key, fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.Del(ctx, keys...)
		<-resultChan
	}
}

func BenchmarkTypedCache_Exists(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	// Set up test data
	mockCache.SetData("bench-key", "bench-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.Exists(ctx, "bench-key")
		<-resultChan
	}
}

func BenchmarkTypedCache_TTL(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	// Set up test data
	mockCache.SetData("bench-key", "bench-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.TTL(ctx, "bench-key")
		<-resultChan
	}
}

func BenchmarkTypedCache_IncrBy(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()
	ttl := time.Hour

	// Set up test data
	mockCache.SetData("bench-counter", int64(0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan := typedCache.IncrBy(ctx, "bench-counter", 1, ttl)
		<-resultChan
	}
}

func BenchmarkTypedCache_Close(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockCache := NewMockCache()
		typedCache := cachex.NewTypedCache[string](mockCache)
		typedCache.Close()
	}
}

// Benchmark concurrent operations
func BenchmarkTypedCache_ConcurrentOperations(b *testing.B) {
	mockCache := NewMockCache()
	typedCache := cachex.NewTypedCache[string](mockCache)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("concurrent-key-%d", i)
			value := fmt.Sprintf("value-%d", i)

			// Set
			resultChan := typedCache.Set(ctx, key, value, time.Hour)
			<-resultChan

			// Get
			resultChan = typedCache.Get(ctx, key)
			<-resultChan

			// Exists
			resultChan = typedCache.Exists(ctx, key)
			<-resultChan

			i++
		}
	})
}

// Benchmark different types
func BenchmarkTypedCache_DifferentTypes(b *testing.B) {
	mockCache := NewMockCache()
	ctx := context.Background()
	ttl := time.Hour

	b.Run("string", func(b *testing.B) {
		typedCache := cachex.NewTypedCache[string](mockCache)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("string-key-%d", i)
			resultChan := typedCache.Set(ctx, key, "string-value", ttl)
			<-resultChan
		}
	})

	b.Run("int", func(b *testing.B) {
		typedCache := cachex.NewTypedCache[int](mockCache)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("int-key-%d", i)
			resultChan := typedCache.Set(ctx, key, i, ttl)
			<-resultChan
		}
	})

	b.Run("bool", func(b *testing.B) {
		typedCache := cachex.NewTypedCache[bool](mockCache)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bool-key-%d", i)
			resultChan := typedCache.Set(ctx, key, i%2 == 0, ttl)
			<-resultChan
		}
	})
}

// TestKeyBuilderIntegration tests comprehensive KeyBuilder integration scenarios
func TestKeyBuilderIntegration(t *testing.T) {
	// Test with advanced Builder
	t.Run("advanced_builder_integration", func(t *testing.T) {
		// Create Redis client for this test
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		defer client.Close()

		keyBuilder, err := cachex.NewBuilder("testapp", "test", "test-secret")
		require.NoError(t, err)

		cache := cachex.NewRedisCache(client, &cachex.JSONCodec{}, keyBuilder, &cachex.DefaultKeyHasher{})
		defer cache.Close()

		// Cast to CacheWithKeyBuilder
		cacheWithKeys, ok := cache.(cachex.CacheWithKeyBuilder)
		require.True(t, ok, "Cache should implement CacheWithKeyBuilder")

		ctx := context.Background()

		// Test BuildKey method
		userKey := cacheWithKeys.BuildKey("user", "123")
		assert.NotEmpty(t, userKey)
		assert.Contains(t, userKey, "testapp")
		assert.Contains(t, userKey, "test")
		assert.Contains(t, userKey, "user")
		assert.Contains(t, userKey, "123")

		// Test BuildListKey method
		filters := map[string]any{
			"status": "active",
			"role":   "admin",
		}
		listKey := cacheWithKeys.BuildListKey("users", filters)
		assert.NotEmpty(t, listKey)
		assert.Contains(t, listKey, "testapp")
		assert.Contains(t, listKey, "test")
		assert.Contains(t, listKey, "list")
		assert.Contains(t, listKey, "users")

		// Test BuildCompositeKey method
		compositeKey := cacheWithKeys.BuildCompositeKey("user", "123", "org", "456")
		assert.NotEmpty(t, compositeKey)
		assert.Contains(t, compositeKey, "testapp")
		assert.Contains(t, compositeKey, "test")
		assert.Contains(t, compositeKey, "user")
		assert.Contains(t, compositeKey, "123")
		assert.Contains(t, compositeKey, "org")
		assert.Contains(t, compositeKey, "456")

		// Test BuildSessionKey method
		sessionKey := cacheWithKeys.BuildSessionKey("sess-123")
		assert.NotEmpty(t, sessionKey)
		assert.Contains(t, sessionKey, "testapp")
		assert.Contains(t, sessionKey, "test")
		assert.Contains(t, sessionKey, "session")
		assert.Contains(t, sessionKey, "sess-123")

		// Test GetKeyBuilder method
		retrievedBuilder := cacheWithKeys.GetKeyBuilder()
		assert.NotNil(t, retrievedBuilder)
		assert.Equal(t, keyBuilder, retrievedBuilder)

		// Test actual cache operations with generated keys
		typedCache := cachex.NewTypedCache[string](cache)

		// Set value using generated key
		setResult := <-typedCache.Set(ctx, userKey, "John Doe", time.Hour)
		assert.NoError(t, setResult.Error)

		// Get value using generated key
		getResult := <-typedCache.Get(ctx, userKey)
		assert.NoError(t, getResult.Error)
		assert.True(t, getResult.Found)
		assert.Equal(t, "John Doe", getResult.Value)
	})

	// Test with default KeyBuilder
	t.Run("default_keybuilder_integration", func(t *testing.T) {
		// Create Redis client for this test
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		defer client.Close()

		cache := cachex.NewRedisCache(client, &cachex.JSONCodec{}, &cachex.DefaultKeyBuilder{}, &cachex.DefaultKeyHasher{})
		defer cache.Close()

		// Cast to CacheWithKeyBuilder
		cacheWithKeys, ok := cache.(cachex.CacheWithKeyBuilder)
		require.True(t, ok, "Cache should implement CacheWithKeyBuilder")

		ctx := context.Background()

		// Test BuildKey method with default builder
		userKey := cacheWithKeys.BuildKey("user", "123")
		assert.Equal(t, "user:123", userKey)

		// Test BuildListKey method with default builder
		filters := map[string]any{"status": "active"}
		listKey := cacheWithKeys.BuildListKey("users", filters)
		assert.Equal(t, "list:users", listKey)

		// Test BuildCompositeKey method with default builder
		compositeKey := cacheWithKeys.BuildCompositeKey("user", "123", "org", "456")
		assert.Equal(t, "user:123:org:456", compositeKey)

		// Test BuildSessionKey method with default builder
		sessionKey := cacheWithKeys.BuildSessionKey("sess-123")
		assert.Equal(t, "session:sess-123", sessionKey)

		// Test actual cache operations
		typedCache := cachex.NewTypedCache[int](cache)

		setResult := <-typedCache.Set(ctx, userKey, 42, time.Hour)
		assert.NoError(t, setResult.Error)

		getResult := <-typedCache.Get(ctx, userKey)
		assert.NoError(t, getResult.Error)
		assert.True(t, getResult.Found)
		assert.Equal(t, 42, getResult.Value)
	})

	// Test KeyBuilder convenience methods
	t.Run("keybuilder_convenience_methods", func(t *testing.T) {
		keyBuilder, err := cachex.NewBuilder("ecommerce", "production", "secret-key")
		require.NoError(t, err)

		// Test convenience methods
		userKey := keyBuilder.BuildUser("123")
		assert.NotEmpty(t, userKey)
		assert.Contains(t, userKey, "user")
		assert.Contains(t, userKey, "123")

		orgKey := keyBuilder.BuildOrg("456")
		assert.NotEmpty(t, orgKey)
		assert.Contains(t, orgKey, "org")
		assert.Contains(t, orgKey, "456")

		productKey := keyBuilder.BuildProduct("789")
		assert.NotEmpty(t, productKey)
		assert.Contains(t, productKey, "product")
		assert.Contains(t, productKey, "789")

		orderKey := keyBuilder.BuildOrder("012")
		assert.NotEmpty(t, orderKey)
		assert.Contains(t, orderKey, "order")
		assert.Contains(t, orderKey, "012")
	})

	// Test key parsing functionality
	t.Run("key_parsing", func(t *testing.T) {
		keyBuilder, err := cachex.NewBuilder("testapp", "test", "test-secret")
		require.NoError(t, err)

		// Build a key
		originalKey := keyBuilder.Build("user", "123")
		assert.NotEmpty(t, originalKey)

		// Parse the key back
		entity, id, err := keyBuilder.ParseKey(originalKey)
		assert.NoError(t, err)
		assert.Equal(t, "user", entity)
		assert.Equal(t, "123", id)
	})

	// Test error handling
	t.Run("error_handling", func(t *testing.T) {
		// Test invalid builder creation
		_, err := cachex.NewBuilder("", "test", "secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app name cannot be empty")

		_, err = cachex.NewBuilder("app", "", "secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment cannot be empty")

		_, err = cachex.NewBuilder("app", "test", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret cannot be empty")

		// Test key parsing with invalid key
		keyBuilder, err := cachex.NewBuilder("testapp", "test", "test-secret")
		require.NoError(t, err)

		_, _, err = keyBuilder.ParseKey("invalid-key-format")
		assert.Error(t, err)
	})

	// Test concurrent access
	t.Run("concurrent_access", func(t *testing.T) {
		// Create Redis client for this test
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		defer client.Close()

		keyBuilder, err := cachex.NewBuilder("testapp", "test", "test-secret")
		require.NoError(t, err)

		cache := cachex.NewRedisCache(client, &cachex.JSONCodec{}, keyBuilder, &cachex.DefaultKeyHasher{})
		defer cache.Close()

		cacheWithKeys, ok := cache.(cachex.CacheWithKeyBuilder)
		require.True(t, ok)

		ctx := context.Background()
		typedCache := cachex.NewTypedCache[string](cache)

		// Test concurrent key building and cache operations
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Build unique keys
				userKey := cacheWithKeys.BuildKey("user", fmt.Sprintf("%d", id))
				sessionKey := cacheWithKeys.BuildSessionKey(fmt.Sprintf("sess-%d", id))

				// Perform cache operations
				setResult := <-typedCache.Set(ctx, userKey, fmt.Sprintf("User %d", id), time.Hour)
				assert.NoError(t, setResult.Error)

				getResult := <-typedCache.Get(ctx, userKey)
				assert.NoError(t, getResult.Error)
				assert.True(t, getResult.Found)
				assert.Equal(t, fmt.Sprintf("User %d", id), getResult.Value)

				// Test session key
				setResult = <-typedCache.Set(ctx, sessionKey, fmt.Sprintf("Session %d", id), time.Hour)
				assert.NoError(t, setResult.Error)
			}(i)
		}

		wg.Wait()
	})
}
