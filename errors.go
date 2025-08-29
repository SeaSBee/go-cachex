package cachex

import (
	"errors"
	"fmt"
)

// Common cache errors
var (
	ErrNotFound         = errors.New("cache: key not found")
	ErrTimeout          = errors.New("cache: operation timeout")
	ErrInvalidKey       = errors.New("cache: invalid key")
	ErrInvalidValue     = errors.New("cache: invalid value")
	ErrSerialization    = errors.New("cache: serialization error")
	ErrEncryption       = errors.New("cache: encryption error")
	ErrDecryption       = errors.New("cache: decryption error")
	ErrLockFailed       = errors.New("cache: lock acquisition failed")
	ErrLockExpired      = errors.New("cache: lock expired")
	ErrStoreClosed      = errors.New("cache: store is closed")
	ErrConnectionFailed = errors.New("cache: connection failed")
	ErrInvalidConfig    = errors.New("cache: invalid configuration")
	ErrNotSupported     = errors.New("cache: operation not supported")
)

// CacheError represents a cache-specific error with additional context
type CacheError struct {
	Op      string
	Key     string
	Message string
	Err     error
	Code    string // Error code for better categorization
}

// Error implements the error interface
func (e *CacheError) Error() string {
	if e == nil {
		return "cache: nil error"
	}

	baseMsg := fmt.Sprintf("cache %s error for key %s: %s", e.Op, e.Key, e.Message)

	if e.Err != nil {
		// Use %s instead of %v to avoid potential recursion with CacheError types
		return fmt.Sprintf("%s: %s", baseMsg, e.Err.Error())
	}
	return baseMsg
}

// Unwrap returns the underlying error
func (e *CacheError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewCacheError creates a new cache error with validation
func NewCacheError(op, key, message string, err error) *CacheError {
	// Validate required fields
	if op == "" {
		op = "unknown"
	}
	if key == "" {
		key = "unknown"
	}
	if message == "" {
		message = "unknown error"
	}

	// Allow wrapping of CacheError types to maintain error chain
	// The Error() method handles recursion properly by using %s instead of %v

	return &CacheError{
		Op:      op,
		Key:     key,
		Message: message,
		Err:     err,
		Code:    "CACHE_ERROR",
	}
}

// NewCacheErrorWithCode creates a new cache error with a specific error code
func NewCacheErrorWithCode(op, key, message, code string, err error) *CacheError {
	// Validate required fields
	if op == "" {
		op = "unknown"
	}
	if key == "" {
		key = "unknown"
	}
	if message == "" {
		message = "unknown error"
	}
	if code == "" {
		code = "CACHE_ERROR"
	}

	// Prevent circular references by ensuring err is not a CacheError
	if cacheErr, ok := err.(*CacheError); ok {
		// Use the underlying error instead of the CacheError to prevent recursion
		err = cacheErr.Err
	}

	return &CacheError{
		Op:      op,
		Key:     key,
		Message: message,
		Err:     err,
		Code:    code,
	}
}

// isErrorType is a helper function that checks if an error matches a specific error type
// Simplified to use only errors.Is which handles all error wrapping cases
func isErrorType(err error, target error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, target)
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return isErrorType(err, ErrNotFound)
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	return isErrorType(err, ErrTimeout)
}

// IsInvalidKey checks if the error is an invalid key error
func IsInvalidKey(err error) bool {
	return isErrorType(err, ErrInvalidKey)
}

// IsInvalidValue checks if the error is an invalid value error
func IsInvalidValue(err error) bool {
	return isErrorType(err, ErrInvalidValue)
}

// IsSerialization checks if the error is a serialization error
func IsSerialization(err error) bool {
	return isErrorType(err, ErrSerialization)
}

// IsEncryption checks if the error is an encryption error
func IsEncryption(err error) bool {
	return isErrorType(err, ErrEncryption)
}

// IsDecryption checks if the error is a decryption error
func IsDecryption(err error) bool {
	return isErrorType(err, ErrDecryption)
}

// IsLockFailed checks if the error is a lock failure error
func IsLockFailed(err error) bool {
	return isErrorType(err, ErrLockFailed)
}

// IsLockExpired checks if the error is a lock expiration error
func IsLockExpired(err error) bool {
	return isErrorType(err, ErrLockExpired)
}

// IsStoreClosed checks if the error is a store closed error
func IsStoreClosed(err error) bool {
	return isErrorType(err, ErrStoreClosed)
}

// IsConnectionFailed checks if the error is a connection failure error
func IsConnectionFailed(err error) bool {
	return isErrorType(err, ErrConnectionFailed)
}

// IsInvalidConfig checks if the error is an invalid configuration error
func IsInvalidConfig(err error) bool {
	return isErrorType(err, ErrInvalidConfig)
}

// IsNotSupported checks if the error is a not supported operation error
func IsNotSupported(err error) bool {
	return isErrorType(err, ErrNotSupported)
}
