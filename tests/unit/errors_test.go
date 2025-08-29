package unit

import (
	"errors"
	"testing"

	"github.com/SeaSBee/go-cachex"
)

func TestCommonErrors(t *testing.T) {
	// Test that all common errors are properly defined
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNotFound",
			err:      cachex.ErrNotFound,
			expected: "cache: key not found",
		},
		{
			name:     "ErrTimeout",
			err:      cachex.ErrTimeout,
			expected: "cache: operation timeout",
		},
		{
			name:     "ErrInvalidKey",
			err:      cachex.ErrInvalidKey,
			expected: "cache: invalid key",
		},
		{
			name:     "ErrInvalidValue",
			err:      cachex.ErrInvalidValue,
			expected: "cache: invalid value",
		},
		{
			name:     "ErrSerialization",
			err:      cachex.ErrSerialization,
			expected: "cache: serialization error",
		},
		{
			name:     "ErrEncryption",
			err:      cachex.ErrEncryption,
			expected: "cache: encryption error",
		},
		{
			name:     "ErrDecryption",
			err:      cachex.ErrDecryption,
			expected: "cache: decryption error",
		},
		{
			name:     "ErrLockFailed",
			err:      cachex.ErrLockFailed,
			expected: "cache: lock acquisition failed",
		},
		{
			name:     "ErrLockExpired",
			err:      cachex.ErrLockExpired,
			expected: "cache: lock expired",
		},
		{
			name:     "ErrStoreClosed",
			err:      cachex.ErrStoreClosed,
			expected: "cache: store is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("%s.Error() = %v, want %v", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestCacheError_Error(t *testing.T) {
	tests := []struct {
		name     string
		cacheErr *cachex.CacheError
		expected string
	}{
		{
			name: "with underlying error",
			cacheErr: &cachex.CacheError{
				Op:      "get",
				Key:     "test-key",
				Message: "operation failed",
				Err:     errors.New("underlying error"),
			},
			expected: "cache get error for key test-key: operation failed: underlying error",
		},
		{
			name: "without underlying error",
			cacheErr: &cachex.CacheError{
				Op:      "set",
				Key:     "test-key",
				Message: "invalid operation",
				Err:     nil,
			},
			expected: "cache set error for key test-key: invalid operation",
		},
		{
			name: "empty key",
			cacheErr: &cachex.CacheError{
				Op:      "delete",
				Key:     "",
				Message: "key is empty",
				Err:     nil,
			},
			expected: "cache delete error for key : key is empty",
		},
		{
			name: "special characters in key",
			cacheErr: &cachex.CacheError{
				Op:      "get",
				Key:     "user:123:profile",
				Message: "key contains colons",
				Err:     nil,
			},
			expected: "cache get error for key user:123:profile: key contains colons",
		},
		{
			name: "long message",
			cacheErr: &cachex.CacheError{
				Op:      "mget",
				Key:     "batch-keys",
				Message: "this is a very long error message that should be handled properly",
				Err:     nil,
			},
			expected: "cache mget error for key batch-keys: this is a very long error message that should be handled properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cacheErr.Error()
			if result != tt.expected {
				t.Errorf("CacheError.Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCacheError_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		cacheErr *cachex.CacheError
		expected error
	}{
		{
			name: "with underlying error",
			cacheErr: &cachex.CacheError{
				Op:      "get",
				Key:     "test-key",
				Message: "operation failed",
				Err:     cachex.ErrNotFound,
			},
			expected: cachex.ErrNotFound,
		},
		{
			name: "without underlying error",
			cacheErr: &cachex.CacheError{
				Op:      "set",
				Key:     "test-key",
				Message: "invalid operation",
				Err:     nil,
			},
			expected: nil,
		},
		{
			name: "with common error",
			cacheErr: &cachex.CacheError{
				Op:      "get",
				Key:     "test-key",
				Message: "not found",
				Err:     cachex.ErrNotFound,
			},
			expected: cachex.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cacheErr.Unwrap()
			if result != tt.expected {
				t.Errorf("CacheError.Unwrap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewCacheError(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		key     string
		message string
		err     error
	}{
		{
			name:    "basic error",
			op:      "get",
			key:     "test-key",
			message: "operation failed",
			err:     errors.New("underlying error"),
		},
		{
			name:    "without underlying error",
			op:      "set",
			key:     "test-key",
			message: "invalid operation",
			err:     nil,
		},
		{
			name:    "empty fields",
			op:      "",
			key:     "",
			message: "",
			err:     nil,
		},
		{
			name:    "with common error",
			op:      "lock",
			key:     "resource",
			message: "lock failed",
			err:     cachex.ErrLockFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheErr := cachex.NewCacheError(tt.op, tt.key, tt.message, tt.err)

			// For empty fields, expect default values
			expectedOp := tt.op
			expectedKey := tt.key
			expectedMessage := tt.message

			if tt.op == "" {
				expectedOp = "unknown"
			}
			if tt.key == "" {
				expectedKey = "unknown"
			}
			if tt.message == "" {
				expectedMessage = "unknown error"
			}

			if cacheErr.Op != expectedOp {
				t.Errorf("NewCacheError().Op = %v, want %v", cacheErr.Op, expectedOp)
			}
			if cacheErr.Key != expectedKey {
				t.Errorf("NewCacheError().Key = %v, want %v", cacheErr.Key, expectedKey)
			}
			if cacheErr.Message != expectedMessage {
				t.Errorf("NewCacheError().Message = %v, want %v", cacheErr.Message, expectedMessage)
			}
			if cacheErr.Err != tt.err {
				t.Errorf("NewCacheError().Err = %v, want %v", cacheErr.Err, tt.err)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct ErrNotFound",
			err:      cachex.ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrNotFound",
			err:      cachex.NewCacheError("get", "test-key", "not found", cachex.ErrNotFound),
			expected: true,
		},
		{
			name:     "different error",
			err:      cachex.ErrTimeout,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "custom error",
			err:      errors.New("custom error"),
			expected: false,
		},
		{
			name:     "wrapped custom error",
			err:      cachex.NewCacheError("get", "test-key", "custom", errors.New("custom error")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cachex.IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct ErrTimeout",
			err:      cachex.ErrTimeout,
			expected: true,
		},
		{
			name:     "wrapped ErrTimeout",
			err:      cachex.NewCacheError("get", "test-key", "timeout", cachex.ErrTimeout),
			expected: true,
		},
		{
			name:     "different error",
			err:      cachex.ErrNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "custom error",
			err:      errors.New("custom error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cachex.IsTimeout(tt.err)
			if result != tt.expected {
				t.Errorf("IsTimeout(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsLockFailed(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct ErrLockFailed",
			err:      cachex.ErrLockFailed,
			expected: true,
		},
		{
			name:     "wrapped ErrLockFailed",
			err:      cachex.NewCacheError("lock", "resource", "lock failed", cachex.ErrLockFailed),
			expected: true,
		},
		{
			name:     "different error",
			err:      cachex.ErrNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "custom error",
			err:      errors.New("custom error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cachex.IsLockFailed(tt.err)
			if result != tt.expected {
				t.Errorf("IsLockFailed(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsLockExpired(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct ErrLockExpired",
			err:      cachex.ErrLockExpired,
			expected: true,
		},
		{
			name:     "wrapped ErrLockExpired",
			err:      cachex.NewCacheError("lock", "resource", "lock expired", cachex.ErrLockExpired),
			expected: true,
		},
		{
			name:     "different error",
			err:      cachex.ErrNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "custom error",
			err:      errors.New("custom error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cachex.IsLockExpired(tt.err)
			if result != tt.expected {
				t.Errorf("IsLockExpired(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestErrorWrappingChain(t *testing.T) {
	// Test deep error wrapping
	originalErr := errors.New("original error")
	level1Err := cachex.NewCacheError("get", "key1", "level 1", originalErr)
	level2Err := cachex.NewCacheError("set", "key2", "level 2", level1Err)
	level3Err := cachex.NewCacheError("delete", "key3", "level 3", level2Err)

	// Test that we can unwrap through the chain
	if level3Err.Unwrap() != level2Err {
		t.Errorf("Level 3 unwrap should return level 2 error")
	}
	if level2Err.Unwrap() != level1Err {
		t.Errorf("Level 2 unwrap should return level 1 error")
	}
	if level1Err.Unwrap() != originalErr {
		t.Errorf("Level 1 unwrap should return original error")
	}

	// Test error message formatting
	expected := "cache delete error for key key3: level 3: cache set error for key key2: level 2: cache get error for key key1: level 1: original error"
	if level3Err.Error() != expected {
		t.Errorf("Level 3 error message = %v, want %v", level3Err.Error(), expected)
	}
}

func TestErrorComparison(t *testing.T) {
	// Test that errors.Is works correctly with wrapped errors
	err2 := cachex.NewCacheError("get", "test", "wrapped", cachex.ErrNotFound)
	err3 := cachex.NewCacheError("set", "test", "different", errors.New("different error"))

	if !errors.Is(err2, cachex.ErrNotFound) {
		t.Errorf("Wrapped error should be identified as ErrNotFound")
	}
	if errors.Is(err3, cachex.ErrNotFound) {
		t.Errorf("Different error should not be identified as ErrNotFound")
	}

	// Test utility functions
	if !cachex.IsNotFound(err2) {
		t.Errorf("IsNotFound should return true for wrapped ErrNotFound")
	}
	if cachex.IsNotFound(err3) {
		t.Errorf("IsNotFound should return false for different error")
	}
}

func TestCacheErrorNilHandling(t *testing.T) {
	// Test utility functions with nil
	if cachex.IsNotFound(nil) {
		t.Errorf("IsNotFound(nil) should return false")
	}
	if cachex.IsTimeout(nil) {
		t.Errorf("IsTimeout(nil) should return false")
	}
	if cachex.IsLockFailed(nil) {
		t.Errorf("IsLockFailed(nil) should return false")
	}
	if cachex.IsLockExpired(nil) {
		t.Errorf("IsLockExpired(nil) should return false")
	}
}

func TestCacheErrorEdgeCases(t *testing.T) {
	// Test with very long strings
	longOp := "this_is_a_very_long_operation_name_that_might_be_used_in_real_world_scenarios"
	longKey := "this_is_a_very_long_key_that_might_be_used_in_real_world_scenarios_with_many_characters"
	longMessage := "this_is_a_very_long_error_message_that_might_be_used_in_real_world_scenarios_with_many_characters_and_details"

	cacheErr := cachex.NewCacheError(longOp, longKey, longMessage, nil)

	// Should not panic and should contain all parts
	errorStr := cacheErr.Error()
	if len(errorStr) == 0 {
		t.Errorf("Error string should not be empty")
	}
	if cacheErr.Op != longOp {
		t.Errorf("Operation should be preserved")
	}
	if cacheErr.Key != longKey {
		t.Errorf("Key should be preserved")
	}
	if cacheErr.Message != longMessage {
		t.Errorf("Message should be preserved")
	}

	// Test with special characters
	specialErr := cachex.NewCacheError("op\n", "key\t", "msg\r", nil)
	_ = specialErr.Error() // Should not panic
}

func TestCacheErrorConcurrency(t *testing.T) {
	// Test that CacheError can be used safely in concurrent scenarios
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// Create and use CacheError concurrently
			err := cachex.NewCacheError("get", "key", "concurrent test", nil)
			_ = err.Error()
			_ = err.Unwrap()
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
