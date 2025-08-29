package cachex

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/seasbee/go-logx"
)

// bufferWriter implements io.Writer for use with json.Encoder
type bufferWriter struct {
	buf []byte
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	bw.buf = append(bw.buf, p...)
	return len(p), nil
}

// JSONCodec implements the Codec interface using JSON serialization
type JSONCodec struct {
	// AllowNilValues determines whether nil values are allowed to be encoded/decoded
	AllowNilValues bool
	// EnableDebugLogging enables debug logging for nil value handling
	EnableDebugLogging bool
	// Buffer pool for reducing memory allocations
	bufferPool *BufferPool
}

// NewJSONCodec creates a new JSON codec
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{
		AllowNilValues:     false, // Default behavior: reject nil values
		EnableDebugLogging: false, // Default behavior: no debug logging
		bufferPool:         GlobalPools.Buffer,
	}
}

// NewJSONCodecWithOptions creates a new JSON codec with custom options
func NewJSONCodecWithOptions(allowNilValues, enableDebugLogging bool) *JSONCodec {
	return &JSONCodec{
		AllowNilValues:     allowNilValues,
		EnableDebugLogging: enableDebugLogging,
		bufferPool:         GlobalPools.Buffer,
	}
}

// Validate checks if the codec configuration is valid
func (c *JSONCodec) Validate() error {
	if c == nil {
		return fmt.Errorf("codec cannot be nil")
	}

	// Validate buffer pool availability
	if c.bufferPool == nil {
		return fmt.Errorf("buffer pool is not initialized")
	}

	return nil
}

// logDebug logs debug messages if debug logging is enabled
func (c *JSONCodec) logDebug(format string, args ...interface{}) {
	if c.EnableDebugLogging {
		logx.Debugf("[JSONCodec] "+format, args...)
	}
}

// Encode serializes a value to JSON bytes
func (c *JSONCodec) Encode(v any) ([]byte, error) {
	// Validate codec configuration
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("codec validation failed: %w", err)
	}

	// Check for nil values safely
	if v == nil {
		c.logDebug("Attempting to encode nil value, AllowNilValues: %t", c.AllowNilValues)
		if !c.AllowNilValues {
			return nil, fmt.Errorf("cannot encode nil value: nil values are not allowed by this codec configuration")
		}
		// Return a special marker for nil values when allowed
		c.logDebug("Encoding nil value as 'null'")
		return []byte("null"), nil
	}

	// Check for nil pointers safely
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		c.logDebug("Attempting to encode nil pointer value of type %T, AllowNilValues: %t", v, c.AllowNilValues)
		if !c.AllowNilValues {
			return nil, fmt.Errorf("cannot encode nil pointer value of type %T: nil values are not allowed by this codec configuration", v)
		}
		// Return a special marker for nil values when allowed
		c.logDebug("Encoding nil pointer value as 'null'")
		return []byte("null"), nil
	}

	c.logDebug("Encoding value of type %T", v)

	// Use standard JSON marshaling for simplicity and reliability
	data, err := json.Marshal(v)
	if err != nil {
		c.logDebug("JSON marshal failed for type %T: %v", v, err)
		return nil, fmt.Errorf("json marshal error for type %T: %w", v, err)
	}

	c.logDebug("Successfully encoded value of type %T, size: %d bytes", v, len(data))
	return data, nil
}

// Decode deserializes JSON bytes to a value
func (c *JSONCodec) Decode(data []byte, v any) error {
	// Validate codec configuration
	if err := c.Validate(); err != nil {
		return fmt.Errorf("codec validation failed: %w", err)
	}

	if len(data) == 0 {
		c.logDebug("Attempting to decode empty data")
		return fmt.Errorf("cannot decode empty data: input data is empty")
	}

	if v == nil {
		c.logDebug("Attempting to decode into nil value")
		return fmt.Errorf("cannot decode into nil value: destination must be a valid pointer")
	}

	// Check for null values first
	dataStr := string(data)
	trimmed := strings.TrimSpace(dataStr)
	if trimmed == "null" {
		c.logDebug("Decoding null value into type %T", v)

		// If nil values are not allowed, return an error
		if !c.AllowNilValues {
			c.logDebug("Nil values not allowed for type %T", v)
			return fmt.Errorf("cannot decode null value into type %T: nil values are not allowed by this codec configuration", v)
		}

		// Handle null value decoding
		return c.handleNullValue(v)
	}

	c.logDebug("Decoding data of size %d bytes into type %T", len(data), v)
	err := json.Unmarshal(data, v)
	if err != nil {
		c.logDebug("JSON unmarshal failed for type %T: %v", v, err)
		return fmt.Errorf("json unmarshal error for type %T: %w", v, err)
	}

	c.logDebug("Successfully decoded data into type %T", v)
	return nil
}

// handleNullValue safely handles decoding null values into the destination
func (c *JSONCodec) handleNullValue(v any) error {
	val := reflect.ValueOf(v)

	// Ensure the pointer is valid
	if !val.IsValid() {
		c.logDebug("Invalid pointer value for type %T", v)
		return fmt.Errorf("invalid pointer value for type %T", v)
	}

	// Must be a pointer type to set to nil
	if val.Kind() != reflect.Ptr {
		c.logDebug("Cannot decode null into non-pointer type %T", v)
		return fmt.Errorf("cannot decode null into non-pointer type %T: destination must be a pointer", v)
	}

	// Check if the pointer itself is nil (uninitialized)
	if val.IsNil() {
		c.logDebug("Cannot decode null into nil pointer of type %T", v)
		return fmt.Errorf("cannot decode null into nil pointer of type %T: destination pointer must be initialized", v)
	}

	// Get the element that the pointer points to
	elem := val.Elem()
	if !elem.IsValid() {
		c.logDebug("Invalid element for pointer type %T", v)
		return fmt.Errorf("invalid element for pointer type %T", v)
	}

	// Handle different pointer types
	switch elem.Kind() {
	case reflect.Ptr:
		// It's a pointer to a pointer (like **TestUser)
		// Set the inner pointer to nil
		c.logDebug("Setting inner pointer to nil for type %T", v)
		elem.Set(reflect.Zero(elem.Type()))
	case reflect.Interface:
		// It's a pointer to an interface
		// Set the interface to nil
		c.logDebug("Setting interface to nil for type %T", v)
		elem.Set(reflect.Zero(elem.Type()))
	default:
		// It's a pointer to a concrete type (struct, slice, etc.)
		// Set it to zero values
		c.logDebug("Setting pointer to zero values for type %T", v)
		elem.Set(reflect.Zero(elem.Type()))
	}

	return nil
}

// Name returns the codec name
func (c *JSONCodec) Name() string {
	return "json"
}
