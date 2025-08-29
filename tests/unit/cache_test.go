package unit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// MockStore implements the Store interface for testing
type MockStore struct {
	data map[string][]byte
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Time),
	}
}

func (m *MockStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		data, exists := m.data[key]
		if !exists {
			result <- cachex.AsyncResult{Exists: false}
			return
		}

		// Check TTL
		if ttl, hasTTL := m.ttl[key]; hasTTL && time.Now().After(ttl) {
			// Need to upgrade to write lock for cleanup
			m.mu.RUnlock()
			m.mu.Lock()
			// Double-check after acquiring write lock
			if ttl, hasTTL := m.ttl[key]; hasTTL && time.Now().After(ttl) {
				delete(m.data, key)
				delete(m.ttl, key)
			}
			m.mu.Unlock()
			m.mu.RLock()
			result <- cachex.AsyncResult{Exists: false}
			return
		}

		result <- cachex.AsyncResult{Value: data, Exists: true}
	}()

	return result
}

func (m *MockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
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
		if ttl > 0 {
			m.ttl[key] = time.Now().Add(ttl)
		}
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockStore) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
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
			if data, exists := m.data[key]; exists {
				// Check TTL
				if ttl, hasTTL := m.ttl[key]; hasTTL && time.Now().After(ttl) {
					continue
				}
				values[key] = data
			}
		}
		result <- cachex.AsyncResult{Values: values}
	}()

	return result
}

func (m *MockStore) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan cachex.AsyncResult {
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
			if ttl > 0 {
				m.ttl[key] = time.Now().Add(ttl)
			}
		}
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockStore) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncResult {
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
			delete(m.ttl, key)
		}
		result <- cachex.AsyncResult{}
	}()

	return result
}

func (m *MockStore) Exists(ctx context.Context, key string) <-chan cachex.AsyncResult {
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
			result <- cachex.AsyncResult{Exists: false}
			return
		}

		// Check TTL
		if ttl, hasTTL := m.ttl[key]; hasTTL && time.Now().After(ttl) {
			result <- cachex.AsyncResult{Exists: false}
			return
		}

		result <- cachex.AsyncResult{Exists: true}
	}()

	return result
}

func (m *MockStore) TTL(ctx context.Context, key string) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		ttl, exists := m.ttl[key]
		if !exists {
			result <- cachex.AsyncResult{Error: cachex.ErrNotFound}
			return
		}

		remaining := time.Until(ttl)
		if remaining <= 0 {
			result <- cachex.AsyncResult{TTL: 0}
			return
		}

		result <- cachex.AsyncResult{TTL: remaining}
	}()

	return result
}

func (m *MockStore) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncResult {
	result := make(chan cachex.AsyncResult, 1)

	go func() {
		defer close(result)

		if ctx.Err() != nil {
			result <- cachex.AsyncResult{Error: ctx.Err()}
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		// This is a simplified implementation for testing
		// In a real implementation, this would handle atomic increments
		current := int64(0)
		if data, exists := m.data[key]; exists {
			// Parse current value (simplified)
			if len(data) > 0 {
				// Try to parse as string representation of number
				current = int64(len(data)) // Just for testing - using length as value
			}
		}

		newValue := current + delta
		m.data[key] = []byte(fmt.Sprintf("%d", newValue))

		if ttlIfCreate > 0 {
			m.ttl[key] = time.Now().Add(ttlIfCreate)
		}

		result <- cachex.AsyncResult{Result: newValue}
	}()

	return result
}

func (m *MockStore) Close() error {
	return nil
}

// MockCodec implements the Codec interface for testing
type MockCodec struct {
	encodeFunc func(v any) ([]byte, error)
	decodeFunc func(data []byte, v any) error
}

func NewMockCodec() *MockCodec {
	return &MockCodec{
		encodeFunc: func(v any) ([]byte, error) {
			// Simple string encoding for testing
			return []byte("encoded:" + v.(string)), nil
		},
		decodeFunc: func(data []byte, v any) error {
			// Simple string decoding for testing
			str := string(data)
			prefix := "encoded:"
			if len(str) >= len(prefix) && str[:len(prefix)] == prefix {
				*(v.(*string)) = str[len(prefix):]
				return nil
			}
			return errors.New("invalid encoded data")
		},
	}
}

func (m *MockCodec) Encode(v any) ([]byte, error) {
	return m.encodeFunc(v)
}

func (m *MockCodec) Decode(data []byte, v any) error {
	return m.decodeFunc(data, v)
}

// MockKeyBuilder implements the KeyBuilder interface for testing
type MockKeyBuilder struct {
	buildFunc          func(entity, id string) string
	buildListFunc      func(entity string, filters map[string]any) string
	buildCompositeFunc func(entityA, idA, entityB, idB string) string
	buildSessionFunc   func(sid string) string
}

func NewMockKeyBuilder() *MockKeyBuilder {
	return &MockKeyBuilder{
		buildFunc: func(entity, id string) string {
			return "mock:" + entity + ":" + id
		},
		buildListFunc: func(entity string, filters map[string]any) string {
			return "mock:list:" + entity + ":filtered"
		},
		buildCompositeFunc: func(entityA, idA, entityB, idB string) string {
			return "mock:" + entityA + ":" + idA + ":" + entityB + ":" + idB
		},
		buildSessionFunc: func(sid string) string {
			return "mock:session:" + sid
		},
	}
}

func (m *MockKeyBuilder) Build(entity, id string) string {
	return m.buildFunc(entity, id)
}

func (m *MockKeyBuilder) BuildList(entity string, filters map[string]any) string {
	return m.buildListFunc(entity, filters)
}

func (m *MockKeyBuilder) BuildComposite(entityA, idA, entityB, idB string) string {
	return m.buildCompositeFunc(entityA, idA, entityB, idB)
}

func (m *MockKeyBuilder) BuildSession(sid string) string {
	return m.buildSessionFunc(sid)
}

// MockKeyHasher implements the KeyHasher interface for testing
type MockKeyHasher struct {
	hashFunc func(data string) string
}

func NewMockKeyHasher() *MockKeyHasher {
	return &MockKeyHasher{
		hashFunc: func(data string) string {
			return "hash:" + data
		},
	}
}

func (m *MockKeyHasher) Hash(data string) string {
	return m.hashFunc(data)
}

func TestOptions_WithStore(t *testing.T) {
	store := NewMockStore()
	option := cachex.WithStore(store)

	var opts cachex.Options
	option(&opts)

	if opts.Store != store {
		t.Errorf("WithStore() did not set the store correctly")
	}
}

func TestOptions_WithCodec(t *testing.T) {
	codec := NewMockCodec()
	option := cachex.WithCodec(codec)

	var opts cachex.Options
	option(&opts)

	if opts.Codec != codec {
		t.Errorf("WithCodec() did not set the codec correctly")
	}
}

func TestOptions_WithKeyBuilder(t *testing.T) {
	builder := NewMockKeyBuilder()
	option := cachex.WithKeyBuilder(builder)

	var opts cachex.Options
	option(&opts)

	if opts.KeyBuilder != builder {
		t.Errorf("WithKeyBuilder() did not set the key builder correctly")
	}
}

func TestOptions_WithKeyHasher(t *testing.T) {
	hasher := NewMockKeyHasher()
	option := cachex.WithKeyHasher(hasher)

	var opts cachex.Options
	option(&opts)

	if opts.KeyHasher != hasher {
		t.Errorf("WithKeyHasher() did not set the key hasher correctly")
	}
}

func TestOptions_WithDefaultTTL(t *testing.T) {
	ttl := 5 * time.Minute
	option := cachex.WithDefaultTTL(ttl)

	var opts cachex.Options
	option(&opts)

	if opts.DefaultTTL != ttl {
		t.Errorf("WithDefaultTTL() did not set the TTL correctly: got %v, want %v", opts.DefaultTTL, ttl)
	}
}

func TestOptions_WithMaxRetries(t *testing.T) {
	maxRetries := 3
	option := cachex.WithMaxRetries(maxRetries)

	var opts cachex.Options
	option(&opts)

	if opts.MaxRetries != maxRetries {
		t.Errorf("WithMaxRetries() did not set the max retries correctly: got %v, want %v", opts.MaxRetries, maxRetries)
	}
}

func TestOptions_WithRetryDelay(t *testing.T) {
	retryDelay := 100 * time.Millisecond
	option := cachex.WithRetryDelay(retryDelay)

	var opts cachex.Options
	option(&opts)

	if opts.RetryDelay != retryDelay {
		t.Errorf("WithRetryDelay() did not set the retry delay correctly: got %v, want %v", opts.RetryDelay, retryDelay)
	}
}

func TestOptions_WithObservability(t *testing.T) {
	config := cachex.ObservabilityConfig{
		EnableMetrics: true,
		EnableTracing: false,
		EnableLogging: true,
	}
	option := cachex.WithObservability(config)

	var opts cachex.Options
	option(&opts)

	if opts.Observability.EnableMetrics != config.EnableMetrics {
		t.Errorf("WithObservability() did not set metrics correctly: got %v, want %v", opts.Observability.EnableMetrics, config.EnableMetrics)
	}
	if opts.Observability.EnableTracing != config.EnableTracing {
		t.Errorf("WithObservability() did not set tracing correctly: got %v, want %v", opts.Observability.EnableTracing, config.EnableTracing)
	}
	if opts.Observability.EnableLogging != config.EnableLogging {
		t.Errorf("WithObservability() did not set logging correctly: got %v, want %v", opts.Observability.EnableLogging, config.EnableLogging)
	}
}

func TestOptions_MultipleOptions(t *testing.T) {
	store := NewMockStore()
	codec := NewMockCodec()
	ttl := 10 * time.Minute

	options := []cachex.Option{
		cachex.WithStore(store),
		cachex.WithCodec(codec),
		cachex.WithDefaultTTL(ttl),
	}

	var opts cachex.Options
	for _, option := range options {
		option(&opts)
	}

	if opts.Store != store {
		t.Errorf("Store not set correctly")
	}
	if opts.Codec != codec {
		t.Errorf("Codec not set correctly")
	}
	if opts.DefaultTTL != ttl {
		t.Errorf("DefaultTTL not set correctly: got %v, want %v", opts.DefaultTTL, ttl)
	}
}

func TestMockStore_BasicOperations(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Test Set and Get
	key := "test-key"
	value := []byte("test-value")

	setResult := <-store.Set(ctx, key, value, 0)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}

	if !getResult.Exists {
		t.Errorf("Get() should return exists=true for existing key")
	}

	if string(getResult.Value) != string(value) {
		t.Errorf("Get() returned wrong value: got %v, want %v", string(getResult.Value), string(value))
	}
}

func TestMockStore_Exists(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Test non-existent key
	existsResult := <-store.Exists(ctx, "non-existent")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if existsResult.Exists {
		t.Errorf("Exists() should return false for non-existent key")
	}

	// Test existing key
	key := "test-key"
	value := []byte("test-value")
	setResult := <-store.Set(ctx, key, value, 0)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	existsResult = <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if !existsResult.Exists {
		t.Errorf("Exists() should return true for existing key")
	}
}

func TestMockStore_Del(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Set a key
	key := "test-key"
	value := []byte("test-value")
	setResult := <-store.Set(ctx, key, value, 0)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Verify it exists
	existsResult := <-store.Exists(ctx, key)
	if !existsResult.Exists {
		t.Errorf("Key should exist before deletion")
	}

	// Delete it
	delResult := <-store.Del(ctx, key)
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
	}

	// Verify it's gone
	existsResult = <-store.Exists(ctx, key)
	if existsResult.Exists {
		t.Errorf("Key should not exist after deletion")
	}
}

func TestMockStore_MGet(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Set multiple keys
	keys := []string{"key1", "key2", "key3"}
	values := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for key, value := range values {
		setResult := <-store.Set(ctx, key, value, 0)
		if setResult.Error != nil {
			t.Errorf("Set() failed for key %s: %v", key, setResult.Error)
		}
	}

	// Get multiple keys
	mgetResult := <-store.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
	}

	if len(mgetResult.Values) != len(values) {
		t.Errorf("MGet() returned wrong number of items: got %d, want %d", len(mgetResult.Values), len(values))
	}

	for key, expectedValue := range values {
		if actualValue, exists := mgetResult.Values[key]; !exists {
			t.Errorf("MGet() missing key: %s", key)
		} else if string(actualValue) != string(expectedValue) {
			t.Errorf("MGet() wrong value for key %s: got %v, want %v", key, string(actualValue), string(expectedValue))
		}
	}
}

func TestMockStore_MSet(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Set multiple items
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	msetResult := <-store.MSet(ctx, items, 0)
	if msetResult.Error != nil {
		t.Errorf("MSet() failed: %v", msetResult.Error)
	}

	// Verify all items were set
	for key, expectedValue := range items {
		getResult := <-store.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() failed for key %s: %v", key, getResult.Error)
		} else if !getResult.Exists {
			t.Errorf("Key %s should exist after MSet", key)
		} else if string(getResult.Value) != string(expectedValue) {
			t.Errorf("Wrong value for key %s: got %v, want %v", key, string(getResult.Value), string(expectedValue))
		}
	}
}

func TestMockStore_TTL(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Test key without TTL
	key := "no-ttl-key"
	value := []byte("test-value")
	setResult := <-store.Set(ctx, key, value, 0)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	ttlResult := <-store.TTL(ctx, key)
	if ttlResult.Error == nil {
		t.Errorf("TTL() should return error for key without TTL")
	}

	// Test key with TTL
	ttlKey := "ttl-key"
	ttlValue := []byte("test-value")
	ttl := 1 * time.Hour
	setResult = <-store.Set(ctx, ttlKey, ttlValue, ttl)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	ttlResult = <-store.TTL(ctx, ttlKey)
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}

	if ttlResult.TTL <= 0 || ttlResult.TTL > ttl {
		t.Errorf("TTL() returned invalid TTL: %v", ttlResult.TTL)
	}
}

func TestMockStore_ContextCancellation(t *testing.T) {
	store := NewMockStore()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context
	cancel()

	// All operations should fail with context error
	getResult := <-store.Get(ctx, "key")
	if getResult.Error != context.Canceled {
		t.Errorf("Get() should fail with context error, got: %v", getResult.Error)
	}

	setResult := <-store.Set(ctx, "key", []byte("value"), 0)
	if setResult.Error != context.Canceled {
		t.Errorf("Set() should fail with context error, got: %v", setResult.Error)
	}

	existsResult := <-store.Exists(ctx, "key")
	if existsResult.Error != context.Canceled {
		t.Errorf("Exists() should fail with context error, got: %v", existsResult.Error)
	}
}

func TestMockCodec_EncodeDecode(t *testing.T) {
	codec := NewMockCodec()

	// Test encoding
	original := "test-string"
	encoded, err := codec.Encode(original)
	if err != nil {
		t.Errorf("Encode() failed: %v", err)
	}

	expectedEncoded := "encoded:" + original
	encodedStr := string(encoded)
	if encodedStr != expectedEncoded {
		t.Errorf("Encode() returned wrong value: got %v, want %v", encodedStr, expectedEncoded)
	}

	// Test decoding
	var decoded string
	err = codec.Decode(encoded, &decoded)
	if err != nil {
		t.Errorf("Decode() failed: %v, encoded: %q", err, encodedStr)
		return
	}

	if decoded != original {
		t.Errorf("Decode() returned wrong value: got %v, want %v", decoded, original)
	}
}

func TestMockKeyBuilder_Methods(t *testing.T) {
	builder := NewMockKeyBuilder()

	// Test Build
	key := builder.Build("user", "123")
	expected := "mock:user:123"
	if key != expected {
		t.Errorf("Build() returned wrong key: got %v, want %v", key, expected)
	}

	// Test BuildList
	listKey := builder.BuildList("users", map[string]any{"status": "active"})
	expectedList := "mock:list:users:filtered"
	if listKey != expectedList {
		t.Errorf("BuildList() returned wrong key: got %v, want %v", listKey, expectedList)
	}

	// Test BuildComposite
	compositeKey := builder.BuildComposite("user", "123", "order", "456")
	expectedComposite := "mock:user:123:order:456"
	if compositeKey != expectedComposite {
		t.Errorf("BuildComposite() returned wrong key: got %v, want %v", compositeKey, expectedComposite)
	}

	// Test BuildSession
	sessionKey := builder.BuildSession("session123")
	expectedSession := "mock:session:session123"
	if sessionKey != expectedSession {
		t.Errorf("BuildSession() returned wrong key: got %v, want %v", sessionKey, expectedSession)
	}
}

func TestMockKeyHasher_Hash(t *testing.T) {
	hasher := NewMockKeyHasher()

	data := "test-data"
	hash := hasher.Hash(data)
	expected := "hash:" + data

	if hash != expected {
		t.Errorf("Hash() returned wrong hash: got %v, want %v", hash, expected)
	}
}

func TestObservabilityConfig_DefaultValues(t *testing.T) {
	config := cachex.ObservabilityConfig{}

	// Test default values (should be false)
	if config.EnableMetrics {
		t.Errorf("EnableMetrics should be false by default")
	}
	if config.EnableTracing {
		t.Errorf("EnableTracing should be false by default")
	}
	if config.EnableLogging {
		t.Errorf("EnableLogging should be false by default")
	}
}

func TestOptions_DefaultValues(t *testing.T) {
	var opts cachex.Options

	// Test default values
	if opts.Store != nil {
		t.Errorf("Store should be nil by default")
	}
	if opts.Codec != nil {
		t.Errorf("Codec should be nil by default")
	}
	if opts.KeyBuilder != nil {
		t.Errorf("KeyBuilder should be nil by default")
	}
	if opts.KeyHasher != nil {
		t.Errorf("KeyHasher should be nil by default")
	}
	if opts.DefaultTTL != 0 {
		t.Errorf("DefaultTTL should be 0 by default")
	}
	if opts.MaxRetries != 0 {
		t.Errorf("MaxRetries should be 0 by default")
	}
	if opts.RetryDelay != 0 {
		t.Errorf("RetryDelay should be 0 by default")
	}
}

func TestCacheInterface_Completeness(t *testing.T) {
	// This test ensures that the Cache interface is complete
	// by checking that all expected methods are present
	// This is a compile-time check rather than a runtime test

	var _ cachex.Cache[string] = (*MockCache)(nil)
}

// MockCache implements Cache interface for interface completeness testing
type MockCache struct{}

func (m *MockCache) WithContext(ctx context.Context) cachex.Cache[string] { return m }
func (m *MockCache) Get(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) Set(ctx context.Context, key string, val string, ttl time.Duration) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) MSet(ctx context.Context, items map[string]string, ttl time.Duration) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) Exists(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) TTL(ctx context.Context, key string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (string, error)) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) WriteThrough(ctx context.Context, key string, val string, ttl time.Duration, writer func(ctx context.Context) error) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) WriteBehind(ctx context.Context, key string, val string, ttl time.Duration, writer func(ctx context.Context) error) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (string, error)) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) InvalidateByTag(ctx context.Context, tags ...string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) AddTags(ctx context.Context, key string, tags ...string) <-chan cachex.AsyncCacheResult[string] {
	result := make(chan cachex.AsyncCacheResult[string], 1)
	result <- cachex.AsyncCacheResult[string]{}
	close(result)
	return result
}
func (m *MockCache) TryLock(ctx context.Context, key string, ttl time.Duration) <-chan cachex.AsyncLockResult {
	result := make(chan cachex.AsyncLockResult, 1)
	result <- cachex.AsyncLockResult{
		OK: true,
		Unlock: func() error {
			return nil
		},
	}
	close(result)
	return result
}
func (m *MockCache) GetStats() map[string]any { return nil }
func (m *MockCache) Close() error             { return nil }

func TestAnyCacheInterface_Completeness(t *testing.T) {
	// This test ensures that the AnyCache interface is complete
	// by checking that all expected methods are present
	// This is a compile-time check rather than a runtime test

	var _ cachex.AnyCache = (*MockAnyCache)(nil)
}

// MockAnyCache implements AnyCache interface for interface completeness testing
type MockAnyCache struct{}

func (m *MockAnyCache) WithContext(ctx context.Context) cachex.AnyCache { return m }
func (m *MockAnyCache) Get(ctx context.Context, key string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) Set(ctx context.Context, key string, val any, ttl time.Duration) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) MGet(ctx context.Context, keys ...string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) Del(ctx context.Context, keys ...string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) Exists(ctx context.Context, key string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) TTL(ctx context.Context, key string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) IncrBy(ctx context.Context, key string, delta int64, ttlIfCreate time.Duration) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) ReadThrough(ctx context.Context, key string, ttl time.Duration, loader func(ctx context.Context) (any, error)) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) WriteThrough(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) WriteBehind(ctx context.Context, key string, val any, ttl time.Duration, writer func(ctx context.Context) error) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) RefreshAhead(ctx context.Context, key string, refreshBefore time.Duration, loader func(ctx context.Context) (any, error)) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) InvalidateByTag(ctx context.Context, tags ...string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) AddTags(ctx context.Context, key string, tags ...string) <-chan cachex.AsyncAnyCacheResult {
	result := make(chan cachex.AsyncAnyCacheResult, 1)
	result <- cachex.AsyncAnyCacheResult{}
	close(result)
	return result
}
func (m *MockAnyCache) TryLock(ctx context.Context, key string, ttl time.Duration) <-chan cachex.AsyncLockResult {
	result := make(chan cachex.AsyncLockResult, 1)
	result <- cachex.AsyncLockResult{
		OK: true,
		Unlock: func() error {
			return nil
		},
	}
	close(result)
	return result
}
func (m *MockAnyCache) GetStats() map[string]any { return nil }
func (m *MockAnyCache) Close() error             { return nil }

func TestCache_ConcurrentClose(t *testing.T) {
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

	// Test concurrent close operations
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- cache.Close()
		}()
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Close operation %d failed: %v", i, err)
		}
	}

	// Verify that subsequent close calls return nil
	err = cache.Close()
	if err != nil {
		t.Errorf("Subsequent close call should return nil, got: %v", err)
	}
}

func TestCache_WithContextResourceSharing(t *testing.T) {
	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}

	originalCache, err := cachex.New[string](
		cachex.WithStore(store),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer originalCache.Close()

	ctx1 := context.WithValue(context.Background(), "test", "value1")
	ctx2 := context.WithValue(context.Background(), "test", "value2")

	// Create context-specific caches
	cache1 := originalCache.WithContext(ctx1)
	cache2 := originalCache.WithContext(ctx2)

	// Test that context caches work before closing
	setResult := <-cache1.Set(ctx1, "test-key", "test-value", time.Minute)
	if setResult.Error != nil {
		t.Errorf("Failed to set value in context cache: %v", setResult.Error)
	}

	getResult := <-cache1.Get(ctx1, "test-key")
	if getResult.Error != nil {
		t.Errorf("Failed to get value from context cache: %v", getResult.Error)
	}
	if getResult.Value != "test-value" {
		t.Errorf("Expected 'test-value', got: %v", getResult.Value)
	}

	// Verify they share the same underlying resources by checking that
	// closing the original cache affects all instances
	err = originalCache.Close()
	if err != nil {
		t.Errorf("Failed to close original cache: %v", err)
	}

	// Verify that operations on context-specific caches fail after original is closed
	result := <-cache1.Get(ctx1, "test-key")
	if result.Error == nil {
		t.Error("Expected error when using context cache after original is closed")
	}

	result = <-cache2.Get(ctx2, "test-key")
	if result.Error == nil {
		t.Error("Expected error when using context cache after original is closed")
	}
}

func TestCache_WithNilValues(t *testing.T) {
	store, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}

	// Test cache with nil values allowed
	cache, err := cachex.New[*string](
		cachex.WithStore(store),
		cachex.WithNilValues(true),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test setting nil value (should work when allowed)
	var nilValue *string
	setResult := <-cache.Set(ctx, "nil-key", nilValue, time.Minute)
	if setResult.Error != nil {
		t.Errorf("Expected nil value to be allowed, got error: %v", setResult.Error)
	}

	// Test getting nil value back
	getResult := <-cache.Get(ctx, "nil-key")
	if getResult.Error != nil {
		t.Errorf("Expected to get nil value back, got error: %v", getResult.Error)
	}
	if getResult.Value != nil {
		t.Errorf("Expected nil value, got: %v", getResult.Value)
	}

	// Test cache with nil values not allowed (default behavior)
	cacheNoNil, err := cachex.New[*string](
		cachex.WithStore(store),
		cachex.WithNilValues(false),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheNoNil.Close()

	// Test setting nil value (should fail when not allowed)
	setResult = <-cacheNoNil.Set(ctx, "nil-key-2", nilValue, time.Minute)
	t.Logf("Set result: err=%v", setResult.Error)
	if setResult.Error == nil {
		t.Error("Expected error when setting nil value with nil values not allowed")
	} else {
		t.Logf("Got expected error: %v", setResult.Error)
	}
}
