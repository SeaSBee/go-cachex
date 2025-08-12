package cache

import (
	"errors"
	"fmt"
)

// Common cache errors
var (
	ErrNotFound      = errors.New("cache: key not found")
	ErrTimeout       = errors.New("cache: operation timeout")
	ErrCircuitOpen   = errors.New("cache: circuit breaker open")
	ErrRateLimited   = errors.New("cache: rate limited")
	ErrInvalidKey    = errors.New("cache: invalid key")
	ErrInvalidValue  = errors.New("cache: invalid value")
	ErrSerialization = errors.New("cache: serialization error")
	ErrEncryption    = errors.New("cache: encryption error")
	ErrDecryption    = errors.New("cache: decryption error")
	ErrLockFailed    = errors.New("cache: lock acquisition failed")
	ErrLockExpired   = errors.New("cache: lock expired")
	ErrStoreClosed   = errors.New("cache: store is closed")
)

// CacheError represents a cache-specific error with additional context
type CacheError struct {
	Op      string
	Key     string
	Message string
	Err     error
}

// Error implements the error interface
func (e *CacheError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("cache %s error for key %s: %s: %v", e.Op, e.Key, e.Message, e.Err)
	}
	return fmt.Sprintf("cache %s error for key %s: %s", e.Op, e.Key, e.Message)
}

// Unwrap returns the underlying error
func (e *CacheError) Unwrap() error {
	return e.Err
}

// NewCacheError creates a new cache error
func NewCacheError(op, key, message string, err error) *CacheError {
	return &CacheError{
		Op:      op,
		Key:     key,
		Message: message,
		Err:     err,
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsCircuitOpen checks if the error is a circuit breaker open error
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

// IsRateLimited checks if the error is a rate limit error
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsLockFailed checks if the error is a lock failure error
func IsLockFailed(err error) bool {
	return errors.Is(err, ErrLockFailed)
}

// IsLockExpired checks if the error is a lock expiration error
func IsLockExpired(err error) bool {
	return errors.Is(err, ErrLockExpired)
}
