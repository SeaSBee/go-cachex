package codec

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStruct for testing MessagePack serialization
type TestStruct struct {
	ID        int       `json:"id" msgpack:"id"`
	Name      string    `json:"name" msgpack:"name"`
	Email     string    `json:"email" msgpack:"email"`
	CreatedAt time.Time `json:"created_at" msgpack:"created_at"`
	Active    bool      `json:"active" msgpack:"active"`
	Score     float64   `json:"score" msgpack:"score"`
}

// TestStructWithoutTags for testing default field names
type TestStructWithoutTags struct {
	ID        int
	Name      string
	Email     string
	CreatedAt time.Time
	Active    bool
	Score     float64
}

func TestNewMessagePackCodec(t *testing.T) {
	codec := NewMessagePackCodec()
	assert.NotNil(t, codec)
	assert.Equal(t, "msgpack", codec.Name())
	assert.False(t, codec.IsJSONTagEnabled())
}

func TestNewMessagePackCodecWithOptions(t *testing.T) {
	// Test with JSON tags enabled
	codec := NewMessagePackCodecWithOptions(true)
	assert.NotNil(t, codec)
	assert.True(t, codec.IsJSONTagEnabled())

	// Test with JSON tags disabled
	codec = NewMessagePackCodecWithOptions(false)
	assert.NotNil(t, codec)
	assert.False(t, codec.IsJSONTagEnabled())
}

func TestMessagePackCodec_EncodeDecode_SimpleTypes(t *testing.T) {
	codec := NewMessagePackCodec()

	testCases := []struct {
		name  string
		value any
	}{
		{"string", "hello world"},
		{"int", 42},
		{"int64", int64(1234567890)},
		{"float64", 3.14159},
		{"bool_true", true},
		{"bool_false", false},
		{"nil_string", (*string)(nil)},
		{"empty_string", ""},
		{"zero_int", 0},
		{"negative_int", -42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			data, err := codec.Encode(tc.value)
			require.NoError(t, err)
			assert.NotNil(t, data)

			// Decode
			var result any
			err = codec.Decode(data, &result)
			require.NoError(t, err)

			// Compare - handle type differences for MessagePack
			if tc.name == "nil_string" {
				// MessagePack decodes nil pointer as nil interface
				assert.Nil(t, result)
			} else if tc.name == "int" || tc.name == "zero_int" || tc.name == "negative_int" {
				// MessagePack may preserve int types as int8/int16/int32/int64
				assert.Equal(t, fmt.Sprintf("%v", tc.value), fmt.Sprintf("%v", result))
			} else {
				assert.Equal(t, tc.value, result)
			}
		})
	}
}

func TestMessagePackCodec_EncodeDecode_Struct(t *testing.T) {
	codec := NewMessagePackCodec()

	original := TestStruct{
		ID:        123,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Active:    true,
		Score:     95.5,
	}

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result TestStruct
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Email, result.Email)
	assert.Equal(t, original.CreatedAt.Unix(), result.CreatedAt.Unix())
	assert.Equal(t, original.Active, result.Active)
	assert.Equal(t, original.Score, result.Score)
}

func TestMessagePackCodec_EncodeDecode_StructWithoutTags(t *testing.T) {
	codec := NewMessagePackCodec()

	original := TestStructWithoutTags{
		ID:        456,
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Date(2023, 2, 1, 14, 30, 0, 0, time.UTC),
		Active:    false,
		Score:     87.3,
	}

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result TestStructWithoutTags
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Email, result.Email)
	assert.Equal(t, original.CreatedAt.Unix(), result.CreatedAt.Unix())
	assert.Equal(t, original.Active, result.Active)
	assert.Equal(t, original.Score, result.Score)
}

func TestMessagePackCodec_EncodeDecode_Slices(t *testing.T) {
	codec := NewMessagePackCodec()

	original := []TestStruct{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Active: true, Score: 90.0},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Active: false, Score: 85.5},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Active: true, Score: 92.3},
	}

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result []TestStruct
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Len(t, result, len(original))
	for i := range original {
		assert.Equal(t, original[i].ID, result[i].ID)
		assert.Equal(t, original[i].Name, result[i].Name)
		assert.Equal(t, original[i].Email, result[i].Email)
		assert.Equal(t, original[i].Active, result[i].Active)
		assert.Equal(t, original[i].Score, result[i].Score)
	}
}

func TestMessagePackCodec_EncodeDecode_Maps(t *testing.T) {
	codec := NewMessagePackCodec()

	original := map[string]any{
		"string": "hello",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nil":    nil,
		"array":  []int{1, 2, 3},
		"nested": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result map[string]any
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare - handle type differences for MessagePack
	assert.Equal(t, original["string"], result["string"])
	assert.Equal(t, fmt.Sprintf("%v", original["int"]), fmt.Sprintf("%v", result["int"]))
	assert.Equal(t, original["float"], result["float"])
	assert.Equal(t, original["bool"], result["bool"])
	assert.Equal(t, original["nil"], result["nil"])
	// MessagePack may decode arrays as []interface{} instead of []int
	assert.Equal(t, fmt.Sprintf("%v", original["array"]), fmt.Sprintf("%v", result["array"]))
	// MessagePack may decode nested maps as map[string]interface{} instead of map[string]string
	assert.Equal(t, fmt.Sprintf("%v", original["nested"]), fmt.Sprintf("%v", result["nested"]))
}

func TestMessagePackCodec_EncodeDecode_Pointers(t *testing.T) {
	codec := NewMessagePackCodec()

	value := "pointer value"
	original := &value

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result *string
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare
	assert.NotNil(t, result)
	assert.Equal(t, *original, *result)
}

func TestMessagePackCodec_EncodeDecode_NilPointer(t *testing.T) {
	codec := NewMessagePackCodec()

	var original *string = nil

	// Encode
	data, err := codec.Encode(original)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Decode
	var result *string
	err = codec.Decode(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Nil(t, result)
}

func TestMessagePackCodec_Encode_ErrorCases(t *testing.T) {
	codec := NewMessagePackCodec()

	// Test encoding nil value
	data, err := codec.Encode(nil)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "cannot encode nil value")
}

func TestMessagePackCodec_Decode_ErrorCases(t *testing.T) {
	codec := NewMessagePackCodec()

	// Test decoding empty data
	var result string
	err := codec.Decode([]byte{}, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode empty data")

	// Test decoding into nil value
	err = codec.Decode([]byte{0x01}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode into nil value")

	// Test decoding invalid MessagePack data
	err = codec.Decode([]byte{0xFF, 0xFF, 0xFF}, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "msgpack unmarshal error")
}

func TestMessagePackCodec_UseJSONTag(t *testing.T) {
	codec := NewMessagePackCodec()

	// Initially disabled
	assert.False(t, codec.IsJSONTagEnabled())

	// Enable JSON tags
	codec.UseJSONTag(true)
	assert.True(t, codec.IsJSONTagEnabled())

	// Disable JSON tags
	codec.UseJSONTag(false)
	assert.False(t, codec.IsJSONTagEnabled())
}

func TestMessagePackCodec_Performance_Comparison(t *testing.T) {
	codec := NewMessagePackCodec()
	jsonCodec := NewJSONCodec()

	// Create test data
	testData := TestStruct{
		ID:        123,
		Name:      "Performance Test User",
		Email:     "perf@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     95.5,
	}

	// Test MessagePack encoding
	msgpackData, err := codec.Encode(testData)
	require.NoError(t, err)

	// Test JSON encoding
	jsonData, err := jsonCodec.Encode(testData)
	require.NoError(t, err)

	// MessagePack should generally be smaller than JSON
	t.Logf("MessagePack size: %d bytes", len(msgpackData))
	t.Logf("JSON size: %d bytes", len(jsonData))

	// Verify both can be decoded correctly
	var msgpackResult TestStruct
	err = codec.Decode(msgpackData, &msgpackResult)
	require.NoError(t, err)
	assert.Equal(t, testData.ID, msgpackResult.ID)

	var jsonResult TestStruct
	err = jsonCodec.Decode(jsonData, &jsonResult)
	require.NoError(t, err)
	assert.Equal(t, testData.ID, jsonResult.ID)
}

func TestMessagePackCodec_Interoperability(t *testing.T) {
	codec := NewMessagePackCodec()

	// Test that MessagePack can handle various data types
	testCases := []struct {
		name  string
		value any
	}{
		{"empty_struct", TestStruct{}},
		{"complex_map", map[string]any{
			"users": []TestStruct{
				{ID: 1, Name: "User1", Active: true},
				{ID: 2, Name: "User2", Active: false},
			},
			"metadata": map[string]any{
				"count": 2,
				"total": 100,
			},
		}},
		{"mixed_array", []any{1, "string", true, 3.14, nil}},
		{"time_value", time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			data, err := codec.Encode(tc.value)
			require.NoError(t, err)
			assert.NotNil(t, data)

			// Decode
			var result any
			err = codec.Decode(data, &result)
			require.NoError(t, err)

			// Basic verification that we got something back
			assert.NotNil(t, result)
		})
	}
}
