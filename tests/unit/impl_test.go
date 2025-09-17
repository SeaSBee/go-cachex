package unit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

// Test data structures
type TestUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestNew_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))

	if err != nil {
		t.Errorf("New() failed: %v", err)
	}
	if cache == nil {
		t.Errorf("New() returned nil cache")
	}
}

func TestNew_NoStore(t *testing.T) {
	_, err := cachex.New[TestUser]()

	if err == nil {
		t.Errorf("New() should fail without store")
	}
	expectedError := "store is required"
	if err.Error() != expectedError {
		t.Errorf("New() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestNew_DefaultOptions(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(10*time.Minute),
		cachex.WithMaxRetries(5),
		cachex.WithRetryDelay(200*time.Millisecond),
	)

	if err != nil {
		t.Errorf("New() failed: %v", err)
	}
	if cache == nil {
		t.Errorf("New() returned nil cache")
	}
}

func TestCache_WithContext(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	type testKeyType struct{}
	type testValueType struct{}
	testKey := testKeyType{}
	testValue := testValueType{}
	ctx := context.WithValue(context.Background(), testKey, testValue)
	cacheWithCtx := cache.WithContext(ctx)

	if cacheWithCtx == nil {
		t.Errorf("WithContext() returned nil")
	}

	// Test with TODO context
	cacheWithTODO := cache.WithContext(context.TODO())
	if cacheWithTODO == nil {
		t.Errorf("WithContext(context.TODO()) returned nil")
	}
}

func TestCache_Get_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set a test value first
	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test value: %v", setResult.Error)
	}

	// Get the value
	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Errorf("Get() failed: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Errorf("Get() should find the key")
	}
	if getResult.Value.ID != user.ID || getResult.Value.Name != user.Name || getResult.Value.Email != user.Email {
		t.Errorf("Get() returned wrong value: got %+v, want %+v", getResult.Value, user)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	getResult := <-cache.Get(ctx, "nonexistent")
	if getResult.Error != nil {
		t.Errorf("Get() should not error for nonexistent key: %v", getResult.Error)
	}
	if getResult.Found {
		t.Errorf("Get() should not find nonexistent key")
	}
	if getResult.Value.ID != "" {
		t.Errorf("Get() should return zero value for nonexistent key")
	}
}

func TestCache_Get_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	getResult := <-cache.Get(ctx, "test")
	if getResult.Error == nil {
		t.Errorf("Get() should fail on closed cache")
	}
}

func TestCache_Set_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() failed: %v", setResult.Error)
	}

	// Verify it was set
	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Errorf("Get() after Set() failed: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Errorf("Get() should find the set key")
	}
	if getResult.Value.ID != user.ID {
		t.Errorf("Get() returned wrong value after Set()")
	}
}

func TestCache_Set_DefaultTTL(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](
		cachex.WithStore(store),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 0) // Use default TTL
	if setResult.Error != nil {
		t.Errorf("Set() with default TTL failed: %v", setResult.Error)
	}
}

func TestCache_Set_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error == nil {
		t.Errorf("Set() should fail on closed cache")
	}
}

func TestCache_MGet_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set test values
	users := map[string]TestUser{
		"user:1": {ID: "1", Name: "John", Email: "john@example.com"},
		"user:2": {ID: "2", Name: "Jane", Email: "jane@example.com"},
		"user:3": {ID: "3", Name: "Bob", Email: "bob@example.com"},
	}

	ctx := context.Background()
	for key, user := range users {
		setResult := <-cache.Set(ctx, key, user, 5*time.Minute)
		if setResult.Error != nil {
			t.Fatalf("Failed to set test value: %v", setResult.Error)
		}
	}

	// Get multiple values
	keys := []string{"user:1", "user:2", "user:3", "user:nonexistent"}
	mgetResult := <-cache.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		t.Errorf("MGet() failed: %v", mgetResult.Error)
	}

	if len(mgetResult.Values) != 3 {
		t.Errorf("MGet() returned %d items, want 3", len(mgetResult.Values))
	}

	for key, expectedUser := range users {
		if resultUser, found := mgetResult.Values[key]; !found {
			t.Errorf("MGet() missing key %s", key)
		} else if resultUser.ID != expectedUser.ID {
			t.Errorf("MGet() wrong value for key %s", key)
		}
	}
}

func TestCache_MGet_EmptyKeys(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	mgetResult := <-cache.MGet(ctx)
	if mgetResult.Error != nil {
		t.Errorf("MGet() with empty keys failed: %v", mgetResult.Error)
	}
	if len(mgetResult.Values) != 0 {
		t.Errorf("MGet() with empty keys should return empty map")
	}
}

func TestCache_MGet_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	mgetResult := <-cache.MGet(ctx, "test")
	if mgetResult.Error == nil {
		t.Errorf("MGet() should fail on closed cache")
	}
}

func TestCache_MSet_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	users := map[string]TestUser{
		"user:1": {ID: "1", Name: "John", Email: "john@example.com"},
		"user:2": {ID: "2", Name: "Jane", Email: "jane@example.com"},
	}

	ctx := context.Background()
	msetResult := <-cache.MSet(ctx, users, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() failed: %v", msetResult.Error)
	}

	// Verify they were set
	for key, expectedUser := range users {
		getResult := <-cache.Get(ctx, key)
		if getResult.Error != nil {
			t.Errorf("Get() after MSet() failed: %v", getResult.Error)
		}
		if !getResult.Found {
			t.Errorf("Get() should find the MSet key %s", key)
		}
		if getResult.Value.ID != expectedUser.ID {
			t.Errorf("Get() returned wrong value after MSet() for key %s", key)
		}
	}
}

func TestCache_MSet_EmptyItems(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	msetResult := <-cache.MSet(ctx, map[string]TestUser{}, 5*time.Minute)
	if msetResult.Error != nil {
		t.Errorf("MSet() with empty items failed: %v", msetResult.Error)
	}
}

func TestCache_MSet_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	users := map[string]TestUser{
		"user:1": {ID: "1", Name: "John", Email: "john@example.com"},
	}
	ctx := context.Background()
	msetResult := <-cache.MSet(ctx, users, 5*time.Minute)
	if msetResult.Error == nil {
		t.Errorf("MSet() should fail on closed cache")
	}
}

func TestCache_Del_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set a test value
	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test value: %v", setResult.Error)
	}

	// Delete it
	delResult := <-cache.Del(ctx, "user:1")
	if delResult.Error != nil {
		t.Errorf("Del() failed: %v", delResult.Error)
	}

	// Verify it was deleted
	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Errorf("Get() after Del() should not error: %v", getResult.Error)
	}
	if getResult.Found {
		t.Errorf("Get() should not find deleted key")
	}
}

func TestCache_Del_EmptyKeys(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	delResult := <-cache.Del(ctx)
	if delResult.Error != nil {
		t.Errorf("Del() with empty keys failed: %v", delResult.Error)
	}
}

func TestCache_Del_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	delResult := <-cache.Del(ctx, "test")
	if delResult.Error == nil {
		t.Errorf("Del() should fail on closed cache")
	}
}

func TestCache_Exists_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	// Test non-existent key
	existsResult := <-cache.Exists(ctx, "nonexistent")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if existsResult.Found {
		t.Errorf("Exists() should return false for nonexistent key")
	}

	// Set a test value
	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test value: %v", setResult.Error)
	}

	// Test existing key
	existsResult = <-cache.Exists(ctx, "user:1")
	if existsResult.Error != nil {
		t.Errorf("Exists() failed: %v", existsResult.Error)
	}
	if !existsResult.Found {
		t.Errorf("Exists() should return true for existing key")
	}
}

func TestCache_Exists_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	existsResult := <-cache.Exists(ctx, "test")
	if existsResult.Error == nil {
		t.Errorf("Exists() should fail on closed cache")
	}
}

func TestCache_TTL_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set a test value with TTL
	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ttl := 5 * time.Minute
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, ttl)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test value: %v", setResult.Error)
	}

	// Get TTL
	ttlResult := <-cache.TTL(ctx, "user:1")
	if ttlResult.Error != nil {
		t.Errorf("TTL() failed: %v", ttlResult.Error)
	}
	if ttlResult.TTL <= 0 || ttlResult.TTL > ttl {
		t.Errorf("TTL() returned invalid value: %v", ttlResult.TTL)
	}
}

func TestCache_TTL_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	ttlResult := <-cache.TTL(ctx, "test")
	if ttlResult.Error == nil {
		t.Errorf("TTL() should fail on closed cache")
	}
}

func TestCache_IncrBy_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	incrResult := <-cache.IncrBy(ctx, "counter", 5, 5*time.Minute)
	if incrResult.Error != nil {
		t.Errorf("IncrBy() failed: %v", incrResult.Error)
	}
	if incrResult.Int != 5 {
		t.Errorf("IncrBy() returned %d, want 5", incrResult.Int)
	}
}

func TestCache_IncrBy_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	incrResult := <-cache.IncrBy(ctx, "counter", 5, 5*time.Minute)
	if incrResult.Error == nil {
		t.Errorf("IncrBy() should fail on closed cache")
	}
}

func TestCache_ReadThrough_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test cache miss - should load from loader
	loadCalled := false
	loader := func(ctx context.Context) (TestUser, error) {
		loadCalled = true
		return TestUser{ID: "1", Name: "Loaded", Email: "loaded@example.com"}, nil
	}

	ctx := context.Background()
	readResult := <-cache.ReadThrough(ctx, "user:1", 5*time.Minute, loader)
	if readResult.Error != nil {
		t.Errorf("ReadThrough() failed: %v", readResult.Error)
	}
	if !loadCalled {
		t.Errorf("ReadThrough() should call loader on cache miss")
	}
	if readResult.Value.Name != "Loaded" {
		t.Errorf("ReadThrough() returned wrong value: %v", readResult.Value)
	}

	// Test cache hit - should not call loader
	loadCalled = false
	readResult = <-cache.ReadThrough(ctx, "user:1", 5*time.Minute, loader)
	if readResult.Error != nil {
		t.Errorf("ReadThrough() failed on second call: %v", readResult.Error)
	}
	if loadCalled {
		t.Errorf("ReadThrough() should not call loader on cache hit")
	}
	if readResult.Value.Name != "Loaded" {
		t.Errorf("ReadThrough() returned wrong cached value: %v", readResult.Value)
	}
}

func TestCache_ReadThrough_LoaderError(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	loader := func(ctx context.Context) (TestUser, error) {
		return TestUser{}, errors.New("loader error")
	}

	ctx := context.Background()
	readResult := <-cache.ReadThrough(ctx, "user:1", 5*time.Minute, loader)
	if readResult.Error == nil {
		t.Errorf("ReadThrough() should fail when loader fails")
	}
}

func TestCache_WriteThrough_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	writeCalled := false
	writer := func(ctx context.Context) error {
		writeCalled = true
		return nil
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	writeResult := <-cache.WriteThrough(ctx, "user:1", user, 5*time.Minute, writer)
	if writeResult.Error != nil {
		t.Errorf("WriteThrough() failed: %v", writeResult.Error)
	}
	if !writeCalled {
		t.Errorf("WriteThrough() should call writer")
	}

	// Verify it was cached
	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Errorf("Get() after WriteThrough() failed: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Errorf("WriteThrough() should cache the value")
	}
	if getResult.Value.ID != user.ID {
		t.Errorf("WriteThrough() cached wrong value")
	}
}

func TestCache_WriteThrough_WriterError(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	writer := func(ctx context.Context) error {
		return errors.New("writer error")
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	writeResult := <-cache.WriteThrough(ctx, "user:1", user, 5*time.Minute, writer)
	if writeResult.Error == nil {
		t.Errorf("WriteThrough() should fail when writer fails")
	}
}

func TestCache_WriteBehind_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	var writeCalled bool
	var mu sync.Mutex
	writer := func(ctx context.Context) error {
		mu.Lock()
		writeCalled = true
		mu.Unlock()
		return nil
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	writeResult := <-cache.WriteBehind(ctx, "user:1", user, 5*time.Minute, writer)
	if writeResult.Error != nil {
		t.Errorf("WriteBehind() failed: %v", writeResult.Error)
	}

	// Verify it was cached immediately
	getResult := <-cache.Get(ctx, "user:1")
	if getResult.Error != nil {
		t.Errorf("Get() after WriteBehind() failed: %v", getResult.Error)
	}
	if !getResult.Found {
		t.Errorf("WriteBehind() should cache the value immediately")
	}
	if getResult.Value.ID != user.ID {
		t.Errorf("WriteBehind() cached wrong value")
	}

	// Wait a bit for async writer to execute
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if !writeCalled {
		t.Errorf("WriteBehind() should call writer asynchronously")
	}
	mu.Unlock()
}

func TestCache_RefreshAhead_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set a value first
	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 1*time.Second) // Short TTL
	if setResult.Error != nil {
		t.Fatalf("Failed to set test value: %v", setResult.Error)
	}

	// Wait for TTL to be close to expiry
	time.Sleep(500 * time.Millisecond)

	loader := func(ctx context.Context) (TestUser, error) {
		return TestUser{ID: "1", Name: "Refreshed", Email: "refreshed@example.com"}, nil
	}

	refreshResult := <-cache.RefreshAhead(ctx, "user:1", 2*time.Second, loader)
	if refreshResult.Error != nil {
		t.Errorf("RefreshAhead() failed: %v", refreshResult.Error)
	}

	// Note: RefreshAhead behavior depends on the refresh scheduler implementation
	// This test just verifies that the method doesn't error
}

func TestCache_InvalidateByTag_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	invalidateResult := <-cache.InvalidateByTag(ctx, "user", "admin")
	if invalidateResult.Error != nil {
		t.Errorf("InvalidateByTag() failed: %v", invalidateResult.Error)
	}
}

func TestCache_InvalidateByTag_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	invalidateResult := <-cache.InvalidateByTag(ctx, "test")
	if invalidateResult.Error == nil {
		t.Errorf("InvalidateByTag() should fail on closed cache")
	}
}

func TestCache_AddTags_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	addTagsResult := <-cache.AddTags(ctx, "user:1", "user", "admin")
	if addTagsResult.Error != nil {
		t.Errorf("AddTags() failed: %v", addTagsResult.Error)
	}
}

func TestCache_AddTags_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	addTagsResult := <-cache.AddTags(ctx, "user:1", "test")
	if addTagsResult.Error == nil {
		t.Errorf("AddTags() should fail on closed cache")
	}
}

func TestCache_TryLock_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	lockResult := <-cache.TryLock(ctx, "resource:1", 5*time.Minute)
	if lockResult.Error != nil {
		t.Errorf("TryLock() failed: %v", lockResult.Error)
	}
	if !lockResult.OK {
		t.Errorf("TryLock() should acquire lock")
	}
	if lockResult.Unlock == nil {
		t.Errorf("TryLock() should return unlock function")
	}

	// Unlock
	if lockResult.Unlock != nil {
		err := lockResult.Unlock()
		if err != nil {
			t.Errorf("unlock() failed: %v", err)
		}
	}
}

func TestCache_TryLock_Closed(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	lockResult := <-cache.TryLock(ctx, "resource:1", 5*time.Minute)
	if lockResult.Error == nil {
		t.Errorf("TryLock() should fail on closed cache")
	}
}

func TestCache_Close_Success(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	err = cache.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Should be idempotent
	err = cache.Close()
	if err != nil {
		t.Errorf("Close() should be idempotent")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	store := NewMockStore()
	cache, err := cachex.New[TestUser](cachex.WithStore(store))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				user := TestUser{
					ID:    fmt.Sprintf("%d_%d", id, j),
					Name:  fmt.Sprintf("User_%d_%d", id, j),
					Email: fmt.Sprintf("user_%d_%d@example.com", id, j),
				}
				key := fmt.Sprintf("user:%d_%d", id, j)
				ctx := context.Background()
				setResult := <-cache.Set(ctx, key, user, 5*time.Minute)
				if setResult.Error != nil {
					t.Errorf("Concurrent Set() failed: %v", setResult.Error)
				}
			}
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("user:%d_%d", id, j)
				ctx := context.Background()
				getResult := <-cache.Get(ctx, key)
				if getResult.Error != nil {
					t.Errorf("Concurrent Get() failed: %v", getResult.Error)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestDefaultCodec_EncodeDecode(t *testing.T) {
	// Use NewJSONCodec to get properly initialized codec
	codec := cachex.NewJSONCodec()

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}

	// Test encode
	data, err := codec.Encode(user)
	if err != nil {
		t.Errorf("Encode() failed: %v", err)
	}

	// Test decode
	var decoded TestUser
	err = codec.Decode(data, &decoded)
	if err != nil {
		t.Errorf("Decode() failed: %v", err)
	}

	if decoded.ID != user.ID || decoded.Name != user.Name || decoded.Email != user.Email {
		t.Errorf("Decode() returned wrong value: got %+v, want %+v", decoded, user)
	}
}

func TestCache_RetryLogic(t *testing.T) {
	// Create a store that fails initially but succeeds after retries
	failingStore := &FailingMockStore{
		MockStore:   NewMockStore(),
		failCount:   2,
		currentFail: 0,
	}

	cache, err := cachex.New[TestUser](
		cachex.WithStore(failingStore),
		cachex.WithMaxRetries(3),
		cachex.WithRetryDelay(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() with retry should succeed: %v", setResult.Error)
	}

	if failingStore.currentFail <= failingStore.failCount {
		t.Errorf("Retry logic should have been exercised")
	}
}

func TestCache_RetryLogicSimple(t *testing.T) {
	// Create a store that fails initially but succeeds after retries
	failingStore := &FailingMockStore{
		MockStore:   NewMockStore(),
		failCount:   1, // Only fail once
		currentFail: 0,
	}

	cache, err := cachex.New[TestUser](
		cachex.WithStore(failingStore),
		cachex.WithMaxRetries(1),                  // Only retry once
		cachex.WithRetryDelay(1*time.Millisecond), // Very short delay
	)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	user := TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	ctx := context.Background()
	setResult := <-cache.Set(ctx, "user:1", user, 5*time.Minute)
	if setResult.Error != nil {
		t.Errorf("Set() with retry should succeed: %v", setResult.Error)
	}

	if failingStore.currentFail <= failingStore.failCount {
		t.Errorf("Retry logic should have been exercised")
	}
}

// FailingMockStore is a mock store that fails for the first N operations
type FailingMockStore struct {
	*MockStore
	failCount   int
	currentFail int
}

func (f *FailingMockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) <-chan cachex.AsyncResult {
	f.currentFail++
	if f.currentFail <= f.failCount {
		result := make(chan cachex.AsyncResult, 1)
		result <- cachex.AsyncResult{Error: errors.New("mock failure")}
		close(result)
		return result
	}
	return f.MockStore.Set(ctx, key, value, ttl)
}

func (f *FailingMockStore) Get(ctx context.Context, key string) <-chan cachex.AsyncResult {
	f.currentFail++
	if f.currentFail <= f.failCount {
		result := make(chan cachex.AsyncResult, 1)
		result <- cachex.AsyncResult{Error: errors.New("mock failure")}
		close(result)
		return result
	}
	return f.MockStore.Get(ctx, key)
}
