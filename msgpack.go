package cachex

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// MessagePackCodec implements the Codec interface using MessagePack serialization
type MessagePackCodec struct {
	// Configuration options for MessagePack encoding/decoding
	useJSONTag bool // Use JSON tags for field names
}

// NewMessagePackCodec creates a new MessagePack codec with default settings
func NewMessagePackCodec() *MessagePackCodec {
	return &MessagePackCodec{
		useJSONTag: false, // Use struct field names by default
	}
}

// NewMessagePackCodecWithOptions creates a new MessagePack codec with custom options
func NewMessagePackCodecWithOptions(useJSONTag bool) *MessagePackCodec {
	return &MessagePackCodec{
		useJSONTag: useJSONTag,
	}
}

// Encode serializes a value to MessagePack bytes
func (c *MessagePackCodec) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot encode nil value")
	}

	data, err := msgpack.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("msgpack marshal error: %w", err)
	}

	return data, nil
}

// Decode deserializes MessagePack bytes to a value
func (c *MessagePackCodec) Decode(data []byte, v any) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot decode empty data")
	}

	if v == nil {
		return fmt.Errorf("cannot decode into nil value")
	}

	err := msgpack.Unmarshal(data, v)
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
	c.useJSONTag = use
}

// IsJSONTagEnabled returns whether JSON tags are being used
func (c *MessagePackCodec) IsJSONTagEnabled() bool {
	return c.useJSONTag
}
