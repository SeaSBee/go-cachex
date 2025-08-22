package cachex

import (
	"encoding/json"
	"fmt"
)

// JSONCodec implements the Codec interface using JSON serialization
type JSONCodec struct{}

// NewJSONCodec creates a new JSON codec
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

// Encode serializes a value to JSON bytes
func (c *JSONCodec) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot encode nil value")
	}

	// Check for nil slices and other nil-like values
	switch val := v.(type) {
	case []byte:
		if val == nil {
			return nil, fmt.Errorf("cannot encode nil value")
		}
	case []string:
		if val == nil {
			return nil, fmt.Errorf("cannot encode nil value")
		}
	case []int:
		if val == nil {
			return nil, fmt.Errorf("cannot encode nil value")
		}
	case map[string]interface{}:
		if val == nil {
			return nil, fmt.Errorf("cannot encode nil value")
		}
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json marshal error: %w", err)
	}

	return data, nil
}

// Decode deserializes JSON bytes to a value
func (c *JSONCodec) Decode(data []byte, v any) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot decode empty data")
	}

	if v == nil {
		return fmt.Errorf("cannot decode into nil value")
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("json unmarshal error: %w", err)
	}

	return nil
}

// Name returns the codec name
func (c *JSONCodec) Name() string {
	return "json"
}
