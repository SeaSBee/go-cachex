package cachex

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/vmihailenco/msgpack/v5"
)

// MessagePackCodec implements the Codec interface using MessagePack serialization
type MessagePackCodec struct {
	// Configuration options for MessagePack encoding/decoding
	// Using atomic values to prevent race conditions
	useJSONTag       atomic.Bool
	useCompactInts   atomic.Bool
	useCompactFloats atomic.Bool
	sortMapKeys      atomic.Bool
}

// NewMessagePackCodec creates a new MessagePack codec with default settings
func NewMessagePackCodec() *MessagePackCodec {
	codec := &MessagePackCodec{}
	codec.useJSONTag.Store(false)      // Use struct field names by default
	codec.useCompactInts.Store(true)   // Use compact integers by default
	codec.useCompactFloats.Store(true) // Use compact floats by default
	codec.sortMapKeys.Store(false)     // Don't sort map keys by default
	return codec
}

// NewMessagePackCodecWithOptions creates a new MessagePack codec with custom options
func NewMessagePackCodecWithOptions(useJSONTag, useCompactInts, useCompactFloats, sortMapKeys bool) *MessagePackCodec {
	codec := &MessagePackCodec{}
	codec.useJSONTag.Store(useJSONTag)
	codec.useCompactInts.Store(useCompactInts)
	codec.useCompactFloats.Store(useCompactFloats)
	codec.sortMapKeys.Store(sortMapKeys)
	return codec
}

// Encode serializes a value to MessagePack bytes
func (c *MessagePackCodec) Encode(v any) ([]byte, error) {
	// Validate input value
	if v == nil {
		return nil, fmt.Errorf("cannot encode nil value")
	}

	var buf bytes.Buffer

	// Get encoder from pool - no need to check for nil as the library handles this
	enc := msgpack.GetEncoder()
	defer msgpack.PutEncoder(enc)

	// Reset encoder with buffer
	enc.Reset(&buf)

	// Apply configuration options using atomic reads for thread safety
	if c.useJSONTag.Load() {
		enc.SetCustomStructTag("json")
	}
	if c.useCompactInts.Load() {
		enc.UseCompactInts(true)
	}
	if c.useCompactFloats.Load() {
		enc.UseCompactFloats(true)
	}
	if c.sortMapKeys.Load() {
		enc.SetSortMapKeys(true)
	}

	err := enc.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("msgpack marshal error: %w", err)
	}

	return buf.Bytes(), nil
}

// Decode deserializes MessagePack bytes to a value
func (c *MessagePackCodec) Decode(data []byte, v any) error {
	if data == nil {
		return fmt.Errorf("cannot decode nil data")
	}

	if len(data) == 0 {
		return fmt.Errorf("cannot decode empty data")
	}

	if v == nil {
		return fmt.Errorf("cannot decode into nil value")
	}

	// Get decoder from pool - no need to check for nil as the library handles this
	dec := msgpack.GetDecoder()
	defer msgpack.PutDecoder(dec)

	// Reset decoder with data
	dec.Reset(bytes.NewReader(data))

	// Apply configuration options using atomic reads for thread safety
	if c.useJSONTag.Load() {
		dec.SetCustomStructTag("json")
	}

	err := dec.Decode(v)
	if err != nil {
		return fmt.Errorf("msgpack unmarshal error: %w", err)
	}

	return nil
}

// Name returns the codec name
func (c *MessagePackCodec) Name() string {
	return "msgpack"
}

// UseJSONTag enables/disables JSON tag usage for field names
func (c *MessagePackCodec) UseJSONTag(use bool) {
	c.useJSONTag.Store(use)
}

// IsJSONTagEnabled returns whether JSON tags are being used
func (c *MessagePackCodec) IsJSONTagEnabled() bool {
	return c.useJSONTag.Load()
}

// UseCompactInts enables/disables compact integer encoding
func (c *MessagePackCodec) UseCompactInts(use bool) {
	c.useCompactInts.Store(use)
}

// IsCompactIntsEnabled returns whether compact integers are being used
func (c *MessagePackCodec) IsCompactIntsEnabled() bool {
	return c.useCompactInts.Load()
}

// UseCompactFloats enables/disables compact float encoding
func (c *MessagePackCodec) UseCompactFloats(use bool) {
	c.useCompactFloats.Store(use)
}

// IsCompactFloatsEnabled returns whether compact floats are being used
func (c *MessagePackCodec) IsCompactFloatsEnabled() bool {
	return c.useCompactFloats.Load()
}

// SortMapKeys enables/disables map key sorting
func (c *MessagePackCodec) SortMapKeys(sort bool) {
	c.sortMapKeys.Store(sort)
}

// IsMapKeysSorted returns whether map keys are being sorted
func (c *MessagePackCodec) IsMapKeysSorted() bool {
	return c.sortMapKeys.Load()
}
