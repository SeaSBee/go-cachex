package cachex

import (
	"sync"
	"time"
)

// AllocationOptimizer provides optimized allocation patterns for hot paths
// to reduce memory allocations and improve performance
type AllocationOptimizer struct {
	// Pre-allocated buffers for common operations
	smallBufferPool  *BufferPool // For small operations (< 1KB)
	mediumBufferPool *BufferPool // For medium operations (1KB - 10KB)
	largeBufferPool  *BufferPool // For large operations (> 10KB)

	// Pre-allocated string builders for key operations
	keyBuilderPool *StringBuilderPool

	// Pre-allocated time objects for common operations
	timePool *TimePool

	// Pre-allocated error objects for common errors
	errorPool *ErrorPool

	// Pre-allocated result channels for async operations
	resultChannelPool *ResultChannelPool
}

// StringBuilderPool provides object pooling for string builders
type StringBuilderPool struct {
	pool sync.Pool
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool() *StringBuilderPool {
	return &StringBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &StringBuilder{
					buf: make([]byte, 0, 256), // Pre-allocate 256 bytes
				}
			},
		},
	}
}

// Get retrieves a string builder from the pool
func (p *StringBuilderPool) Get() *StringBuilder {
	return p.pool.Get().(*StringBuilder)
}

// Put returns a string builder to the pool
func (p *StringBuilderPool) Put(sb *StringBuilder) {
	sb.Reset()
	p.pool.Put(sb)
}

// StringBuilder provides efficient string building with minimal allocations
type StringBuilder struct {
	buf []byte
}

// Reset resets the string builder for reuse
func (sb *StringBuilder) Reset() {
	sb.buf = sb.buf[:0]
}

// Write appends bytes to the string builder
func (sb *StringBuilder) Write(p []byte) (n int, err error) {
	sb.buf = append(sb.buf, p...)
	return len(p), nil
}

// WriteString appends a string to the string builder
func (sb *StringBuilder) WriteString(s string) (n int, err error) {
	sb.buf = append(sb.buf, s...)
	return len(s), nil
}

// WriteByte appends a byte to the string builder
func (sb *StringBuilder) WriteByte(c byte) error {
	sb.buf = append(sb.buf, c)
	return nil
}

// String returns the built string
func (sb *StringBuilder) String() string {
	return string(sb.buf)
}

// Bytes returns the underlying byte slice
func (sb *StringBuilder) Bytes() []byte {
	return sb.buf
}

// Len returns the current length
func (sb *StringBuilder) Len() int {
	return len(sb.buf)
}

// Cap returns the current capacity
func (sb *StringBuilder) Cap() int {
	return cap(sb.buf)
}

// Grow ensures the buffer has at least n bytes of capacity
func (sb *StringBuilder) Grow(n int) {
	if cap(sb.buf)-len(sb.buf) < n {
		newBuf := make([]byte, len(sb.buf), 2*cap(sb.buf)+n)
		copy(newBuf, sb.buf)
		sb.buf = newBuf
	}
}

// TimePool provides object pooling for time.Time objects
type TimePool struct {
	pool sync.Pool
}

// NewTimePool creates a new time pool
func NewTimePool() *TimePool {
	return &TimePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &time.Time{}
			},
		},
	}
}

// Get retrieves a time object from the pool
func (p *TimePool) Get() *time.Time {
	return p.pool.Get().(*time.Time)
}

// Put returns a time object to the pool
func (p *TimePool) Put(t *time.Time) {
	*t = time.Time{} // Reset to zero value
	p.pool.Put(t)
}

// ErrorPool provides object pooling for common error objects
type ErrorPool struct {
	pool sync.Pool
}

// NewErrorPool creates a new error pool
func NewErrorPool() *ErrorPool {
	return &ErrorPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &CommonError{
					message: "",
					errType: ErrorTypeUnknown,
				}
			},
		},
	}
}

// Get retrieves an error object from the pool
func (p *ErrorPool) Get() *CommonError {
	return p.pool.Get().(*CommonError)
}

// Put returns an error object to the pool
func (p *ErrorPool) Put(err *CommonError) {
	err.message = ""
	err.errType = ErrorTypeUnknown
	p.pool.Put(err)
}

// CommonError represents frequently used error types
type CommonError struct {
	message string
	errType ErrorType
}

// ErrorType represents the type of error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeKeyEmpty
	ErrorTypeValueNil
	ErrorTypeTTLNegative
	ErrorTypeStoreNotInitialized
	ErrorTypeContextCancelled
	ErrorTypeItemExpired
	ErrorTypeItemNotFound
)

// Error returns the error message
func (e *CommonError) Error() string {
	return e.message
}

// Set sets the error message and type
func (e *CommonError) Set(message string, errType ErrorType) {
	e.message = message
	e.errType = errType
}

// ResultChannelPool provides object pooling for result channels
type ResultChannelPool struct {
	pool sync.Pool
}

// NewResultChannelPool creates a new result channel pool
func NewResultChannelPool() *ResultChannelPool {
	return &ResultChannelPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make(chan AsyncResult, 1)
			},
		},
	}
}

// Get retrieves a result channel from the pool
func (p *ResultChannelPool) Get() chan AsyncResult {
	return p.pool.Get().(chan AsyncResult)
}

// Put returns a result channel to the pool
func (p *ResultChannelPool) Put(ch chan AsyncResult) {
	// Drain the channel to prevent memory leaks
	select {
	case <-ch:
	default:
	}
	p.pool.Put(ch)
}

// NewAllocationOptimizer creates a new allocation optimizer
func NewAllocationOptimizer() *AllocationOptimizer {
	return &AllocationOptimizer{
		smallBufferPool:   NewBufferPool(),
		mediumBufferPool:  NewBufferPool(),
		largeBufferPool:   NewBufferPool(),
		keyBuilderPool:    NewStringBuilderPool(),
		timePool:          NewTimePool(),
		errorPool:         NewErrorPool(),
		resultChannelPool: NewResultChannelPool(),
	}
}

// OptimizedBufferPool provides size-optimized buffer pools
type OptimizedBufferPool struct {
	smallPool  *BufferPool // < 1KB
	mediumPool *BufferPool // 1KB - 10KB
	largePool  *BufferPool // > 10KB
}

// NewOptimizedBufferPool creates a new optimized buffer pool
func NewOptimizedBufferPool() *OptimizedBufferPool {
	return &OptimizedBufferPool{
		smallPool: &BufferPool{
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 0, 512) // 512 bytes for small operations
				},
			},
		},
		mediumPool: &BufferPool{
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 0, 4096) // 4KB for medium operations
				},
			},
		},
		largePool: &BufferPool{
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 0, 16384) // 16KB for large operations
				},
			},
		},
	}
}

// Get retrieves a buffer based on the required size
func (obp *OptimizedBufferPool) Get(size int) []byte {
	switch {
	case size <= 512:
		return obp.smallPool.Get()
	case size <= 4096:
		return obp.mediumPool.Get()
	default:
		return obp.largePool.Get()
	}
}

// Put returns a buffer to the appropriate pool
func (obp *OptimizedBufferPool) Put(buf []byte) {
	capacity := cap(buf)
	switch {
	case capacity <= 512:
		obp.smallPool.Put(buf)
	case capacity <= 4096:
		obp.mediumPool.Put(buf)
	default:
		obp.largePool.Put(buf)
	}
}

// Global allocation optimizer instance
var GlobalAllocationOptimizer *AllocationOptimizer

// init initializes the global allocation optimizer
func init() {
	GlobalAllocationOptimizer = NewAllocationOptimizer()
}

// OptimizedValueCopy provides optimized value copying with minimal allocations
func OptimizedValueCopy(src []byte) []byte {
	if src == nil {
		return nil
	}

	// Use appropriate buffer pool based on size
	var buf []byte
	switch {
	case len(src) <= 512:
		buf = GlobalAllocationOptimizer.smallBufferPool.Get()
	case len(src) <= 4096:
		buf = GlobalAllocationOptimizer.mediumBufferPool.Get()
	default:
		buf = GlobalAllocationOptimizer.largeBufferPool.Get()
	}

	// Ensure buffer has enough capacity
	if cap(buf) < len(src) {
		// If pooled buffer is too small, create a new one
		switch {
		case len(src) <= 512:
			GlobalAllocationOptimizer.smallBufferPool.Put(buf)
		case len(src) <= 4096:
			GlobalAllocationOptimizer.mediumBufferPool.Put(buf)
		default:
			GlobalAllocationOptimizer.largeBufferPool.Put(buf)
		}
		return make([]byte, len(src))
	}

	// Reset buffer and copy data
	buf = buf[:0]
	buf = append(buf, src...)
	return buf
}

// OptimizedKeyBuilder provides optimized key building
func OptimizedKeyBuilder() *StringBuilder {
	return GlobalAllocationOptimizer.keyBuilderPool.Get()
}

// OptimizedTime provides optimized time operations
func OptimizedTime() *time.Time {
	return GlobalAllocationOptimizer.timePool.Get()
}

// OptimizedError provides optimized error creation
func OptimizedError(message string, errType ErrorType) *CommonError {
	err := GlobalAllocationOptimizer.errorPool.Get()
	err.Set(message, errType)
	return err
}

// OptimizedResultChannel provides optimized result channel creation
func OptimizedResultChannel() chan AsyncResult {
	return GlobalAllocationOptimizer.resultChannelPool.Get()
}

// OptimizedMapCopy provides optimized map copying
func OptimizedMapCopy(src map[string][]byte) map[string][]byte {
	if src == nil {
		return nil
	}

	// Use map pool for the result
	result := GlobalPools.Map.Get()

	// Copy values using optimized copy
	for k, v := range src {
		result[k] = OptimizedValueCopy(v)
	}

	return result
}

// OptimizedStringSlice provides optimized string slice operations
func OptimizedStringSlice(initialCapacity int) []string {
	if initialCapacity <= 16 {
		return GlobalPools.StringSlice.Get()
	}
	return make([]string, 0, initialCapacity)
}

// OptimizedByteSlice provides optimized byte slice operations
func OptimizedByteSlice(initialCapacity int) []byte {
	switch {
	case initialCapacity <= 512:
		return GlobalAllocationOptimizer.smallBufferPool.Get()
	case initialCapacity <= 4096:
		return GlobalAllocationOptimizer.mediumBufferPool.Get()
	default:
		return GlobalAllocationOptimizer.largeBufferPool.Get()
	}
}

// OptimizedStringJoin provides optimized string joining
func OptimizedStringJoin(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	if len(elems) == 1 {
		return elems[0]
	}

	// Calculate total length
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	// Use optimized buffer
	buf := OptimizedByteSlice(n)
	defer func() {
		// Return buffer to appropriate pool
		switch {
		case cap(buf) <= 512:
			GlobalAllocationOptimizer.smallBufferPool.Put(buf)
		case cap(buf) <= 4096:
			GlobalAllocationOptimizer.mediumBufferPool.Put(buf)
		default:
			GlobalAllocationOptimizer.largeBufferPool.Put(buf)
		}
	}()

	// Build the string
	for i, elem := range elems {
		if i > 0 {
			buf = append(buf, sep...)
		}
		buf = append(buf, elem...)
	}

	return string(buf)
}

// OptimizedStringBuilder provides optimized string building with automatic cleanup
type OptimizedStringBuilder struct {
	sb *StringBuilder
}

// NewOptimizedStringBuilder creates a new optimized string builder
func NewOptimizedStringBuilder() *OptimizedStringBuilder {
	return &OptimizedStringBuilder{
		sb: OptimizedKeyBuilder(),
	}
}

// Write appends bytes to the builder
func (osb *OptimizedStringBuilder) Write(p []byte) (n int, err error) {
	return osb.sb.Write(p)
}

// WriteString appends a string to the builder
func (osb *OptimizedStringBuilder) WriteString(s string) (n int, err error) {
	return osb.sb.WriteString(s)
}

// WriteByte appends a byte to the builder
func (osb *OptimizedStringBuilder) WriteByte(c byte) error {
	return osb.sb.WriteByte(c)
}

// String returns the built string and returns the builder to the pool
func (osb *OptimizedStringBuilder) String() string {
	result := osb.sb.String()
	GlobalAllocationOptimizer.keyBuilderPool.Put(osb.sb)
	return result
}

// Bytes returns the underlying bytes and returns the builder to the pool
func (osb *OptimizedStringBuilder) Bytes() []byte {
	result := osb.sb.Bytes()
	GlobalAllocationOptimizer.keyBuilderPool.Put(osb.sb)
	return result
}

// Reset resets the builder for reuse
func (osb *OptimizedStringBuilder) Reset() {
	osb.sb.Reset()
}

// Close returns the builder to the pool
func (osb *OptimizedStringBuilder) Close() {
	GlobalAllocationOptimizer.keyBuilderPool.Put(osb.sb)
}
