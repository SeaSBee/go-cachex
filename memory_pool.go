package cachex

import (
	"fmt"
	"sync"
	"time"
)

// CacheItemPool provides object pooling for cacheItem structures
// to reduce memory allocations and improve performance
type CacheItemPool struct {
	pool sync.Pool
}

// NewCacheItemPool creates a new cache item pool
func NewCacheItemPool() *CacheItemPool {
	return &CacheItemPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &cacheItem{
					Value:       nil,
					ExpiresAt:   time.Time{},
					AccessCount: 0,
					LastAccess:  time.Time{},
					Size:        0,
				}
			},
		},
	}
}

// Get retrieves a cacheItem from the pool
func (p *CacheItemPool) Get() *cacheItem {
	if p == nil {
		return &cacheItem{
			Value:       nil,
			ExpiresAt:   time.Time{},
			AccessCount: 0,
			LastAccess:  time.Time{},
			Size:        0,
		}
	}

	item := p.pool.Get()
	if item == nil {
		return &cacheItem{
			Value:       nil,
			ExpiresAt:   time.Time{},
			AccessCount: 0,
			LastAccess:  time.Time{},
			Size:        0,
		}
	}

	if cacheItem, ok := item.(*cacheItem); ok {
		return cacheItem
	}

	// Fallback if type assertion fails
	return &cacheItem{
		Value:       nil,
		ExpiresAt:   time.Time{},
		AccessCount: 0,
		LastAccess:  time.Time{},
		Size:        0,
	}
}

// Put returns a cacheItem to the pool after resetting its fields
func (p *CacheItemPool) Put(item *cacheItem) {
	if p == nil || item == nil {
		return // Don't put nil items back in the pool
	}

	// Reset the item to its initial state
	item.Value = nil
	item.ExpiresAt = time.Time{}
	item.AccessCount = 0
	item.LastAccess = time.Time{}
	item.Size = 0

	p.pool.Put(item)
}

// AsyncResultPool provides object pooling for AsyncResult structures
// to reduce memory allocations in async operations
type AsyncResultPool struct {
	pool sync.Pool
}

// NewAsyncResultPool creates a new async result pool
func NewAsyncResultPool() *AsyncResultPool {
	return &AsyncResultPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &AsyncResult{
					Result: 0,
					Values: nil,
					Error:  nil,
				}
			},
		},
	}
}

// Get retrieves an AsyncResult from the pool
func (p *AsyncResultPool) Get() *AsyncResult {
	if p == nil {
		return &AsyncResult{
			Result: 0,
			Values: nil,
			Error:  nil,
		}
	}

	result := p.pool.Get()
	if result == nil {
		return &AsyncResult{
			Result: 0,
			Values: nil,
			Error:  nil,
		}
	}

	if asyncResult, ok := result.(*AsyncResult); ok {
		return asyncResult
	}

	// Fallback if type assertion fails
	return &AsyncResult{
		Result: 0,
		Values: nil,
		Error:  nil,
	}
}

// Put returns an AsyncResult to the pool after resetting its fields
func (p *AsyncResultPool) Put(result *AsyncResult) {
	if p == nil || result == nil {
		return // Don't put nil results back in the pool
	}

	// Reset the result to its initial state
	result.Result = 0
	result.Values = nil
	result.Error = nil

	p.pool.Put(result)
}

// BufferPool provides object pooling for byte slices
// to reduce memory allocations for encoding/decoding operations
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024) // Pre-allocate 1KB buffer
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() []byte {
	if p == nil {
		return make([]byte, 0, 1024)
	}

	buf := p.pool.Get()
	if buf == nil {
		return make([]byte, 0, 1024)
	}

	if buffer, ok := buf.([]byte); ok {
		return buffer
	}

	// Fallback if type assertion fails
	return make([]byte, 0, 1024)
}

// Put returns a buffer to the pool after resetting it
func (p *BufferPool) Put(buf []byte) {
	if p == nil {
		return
	}

	// Reset the buffer to zero length but keep capacity
	if buf != nil {
		buf = buf[:0]
		p.pool.Put(buf)
	}
}

// GetWithCapacity retrieves a buffer with at least the specified capacity
func (p *BufferPool) GetWithCapacity(minCapacity int) []byte {
	if minCapacity <= 0 {
		minCapacity = 1024 // Default minimum capacity
	}

	buf := p.Get()
	if buf == nil || cap(buf) < minCapacity {
		// If the pooled buffer is too small or nil, create a new one
		if buf != nil {
			p.Put(buf)
		}
		return make([]byte, 0, minCapacity)
	}
	return buf
}

// StringSlicePool provides object pooling for string slices
// to reduce memory allocations for key collections
type StringSlicePool struct {
	pool sync.Pool
}

// NewStringSlicePool creates a new string slice pool
func NewStringSlicePool() *StringSlicePool {
	return &StringSlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]string, 0, 16) // Pre-allocate 16 strings
			},
		},
	}
}

// Get retrieves a string slice from the pool
func (p *StringSlicePool) Get() []string {
	if p == nil {
		return make([]string, 0, 16)
	}

	slice := p.pool.Get()
	if slice == nil {
		return make([]string, 0, 16)
	}

	if stringSlice, ok := slice.([]string); ok {
		return stringSlice
	}

	// Fallback if type assertion fails
	return make([]string, 0, 16)
}

// Put returns a string slice to the pool after resetting it
func (p *StringSlicePool) Put(slice []string) {
	if p == nil {
		return
	}

	// Reset the slice to zero length but keep capacity
	if slice != nil {
		slice = slice[:0]
		p.pool.Put(slice)
	}
}

// MapPool provides object pooling for map[string][]byte
// to reduce memory allocations for MGet operations
type MapPool struct {
	pool sync.Pool
}

// NewMapPool creates a new map pool
func NewMapPool() *MapPool {
	return &MapPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make(map[string][]byte, 16) // Pre-allocate 16 entries
			},
		},
	}
}

// Get retrieves a map from the pool
func (p *MapPool) Get() map[string][]byte {
	if p == nil {
		return make(map[string][]byte, 16)
	}

	m := p.pool.Get()
	if m == nil {
		return make(map[string][]byte, 16)
	}

	if mapResult, ok := m.(map[string][]byte); ok {
		return mapResult
	}

	// Fallback if type assertion fails
	return make(map[string][]byte, 16)
}

// Put returns a map to the pool after clearing it
func (p *MapPool) Put(m map[string][]byte) {
	if p == nil || m == nil {
		return
	}

	// Clear the map but keep the underlying storage
	for k := range m {
		delete(m, k)
	}
	p.pool.Put(m)
}

// GlobalPools provides access to global object pools
// These can be shared across multiple cache instances
var GlobalPools = struct {
	CacheItem   *CacheItemPool
	AsyncResult *AsyncResultPool
	Buffer      *BufferPool
	StringSlice *StringSlicePool
	Map         *MapPool
}{
	CacheItem:   NewCacheItemPool(),
	AsyncResult: NewAsyncResultPool(),
	Buffer:      NewBufferPool(),
	StringSlice: NewStringSlicePool(),
	Map:         NewMapPool(),
}

// ValidateGlobalPools ensures all global pools are properly initialized
func ValidateGlobalPools() error {
	if GlobalPools.CacheItem == nil {
		return fmt.Errorf("CacheItem pool is not initialized")
	}
	if GlobalPools.AsyncResult == nil {
		return fmt.Errorf("AsyncResult pool is not initialized")
	}
	if GlobalPools.Buffer == nil {
		return fmt.Errorf("Buffer pool is not initialized")
	}
	if GlobalPools.StringSlice == nil {
		return fmt.Errorf("StringSlice pool is not initialized")
	}
	if GlobalPools.Map == nil {
		return fmt.Errorf("Map pool is not initialized")
	}
	return nil
}
