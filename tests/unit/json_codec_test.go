package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONCodec_Creation tests the JSONCodec constructor functions
func TestJSONCodec_Creation(t *testing.T) {
	t.Run("NewJSONCodec default values", func(t *testing.T) {
		codec := cachex.NewJSONCodec()

		assert.NotNil(t, codec)
		assert.False(t, codec.AllowNilValues)
		assert.False(t, codec.EnableDebugLogging)
	})

	t.Run("NewJSONCodecWithOptions", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, true)

		assert.NotNil(t, codec)
		assert.True(t, codec.AllowNilValues)
		assert.True(t, codec.EnableDebugLogging)
	})

	t.Run("NewJSONCodecWithOptions false values", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(false, false)

		assert.NotNil(t, codec)
		assert.False(t, codec.AllowNilValues)
		assert.False(t, codec.EnableDebugLogging)
	})

	t.Run("Validate valid codec", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		err := codec.Validate()

		assert.NoError(t, err)
	})

	t.Run("Validate nil codec", func(t *testing.T) {
		var codec *cachex.JSONCodec
		err := codec.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "codec cannot be nil")
	})
}

// TestJSONCodec_Encode tests the Encode method
func TestJSONCodec_Encode(t *testing.T) {
	t.Run("encode string", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := "test string"

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data is valid JSON
		var decoded string
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode integer", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := 42

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded int
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode float", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := 3.14159

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded float64
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode boolean", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := true

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded bool
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode slice", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := []string{"a", "b", "c"}

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded []string
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode map", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := map[string]int{"a": 1, "b": 2, "c": 3}

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded map[string]int
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode struct", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		type TestStruct struct {
			Name  string `json:"name"`
			Age   int    `json:"age"`
			Email string `json:"email"`
		}
		value := TestStruct{
			Name:  "John Doe",
			Age:   30,
			Email: "john@example.com",
		}

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded TestStruct
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode pointer to struct", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		value := &TestStruct{
			Name: "Jane Doe",
			Age:  25,
		}

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded TestStruct
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, *value, decoded)
	})

	t.Run("encode time", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		value := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)

		data, err := codec.Encode(value)

		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify the encoded data
		var decoded time.Time
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, value, decoded)
	})

	t.Run("encode nil value with AllowNilValues false", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(false, false)

		data, err := codec.Encode(nil)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "cannot encode nil value")
	})

	t.Run("encode nil value with AllowNilValues true", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)

		data, err := codec.Encode(nil)

		assert.NoError(t, err)
		assert.Equal(t, []byte("null"), data)
	})

	t.Run("encode nil pointer with AllowNilValues false", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(false, false)
		var ptr *string

		data, err := codec.Encode(ptr)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "cannot encode nil pointer value")
	})

	t.Run("encode nil pointer with AllowNilValues true", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var ptr *string

		data, err := codec.Encode(ptr)

		assert.NoError(t, err)
		assert.Equal(t, []byte("null"), data)
	})

	t.Run("encode with nil codec", func(t *testing.T) {
		var codec *cachex.JSONCodec

		data, err := codec.Encode("test")

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "codec validation failed")
	})
}

// TestJSONCodec_Decode tests the Decode method
func TestJSONCodec_Decode(t *testing.T) {
	t.Run("decode string", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`"test string"`)
		var result string

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, "test string", result)
	})

	t.Run("decode integer", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`42`)
		var result int

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("decode float", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`3.14159`)
		var result float64

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, 3.14159, result)
	})

	t.Run("decode boolean", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`true`)
		var result bool

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("decode slice", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`["a", "b", "c"]`)
		var result []string

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("decode map", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`{"a": 1, "b": 2, "c": 3}`)
		var result map[string]int

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3}, result)
	})

	t.Run("decode struct", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		type TestStruct struct {
			Name  string `json:"name"`
			Age   int    `json:"age"`
			Email string `json:"email"`
		}
		data := []byte(`{"name": "John Doe", "age": 30, "email": "john@example.com"}`)
		var result TestStruct

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, TestStruct{
			Name:  "John Doe",
			Age:   30,
			Email: "john@example.com",
		}, result)
	})

	t.Run("decode pointer to struct", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		data := []byte(`{"name": "Jane Doe", "age": 25}`)
		result := &TestStruct{}

		err := codec.Decode(data, result)

		assert.NoError(t, err)
		assert.Equal(t, "Jane Doe", result.Name)
		assert.Equal(t, 25, result.Age)
	})

	t.Run("decode time", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`"2023-12-25T10:30:00Z"`)
		var result time.Time

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		expected := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, expected, result)
	})

	t.Run("decode null with AllowNilValues false", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(false, false)
		data := []byte(`null`)
		var result string

		err := codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode null value")
	})

	t.Run("decode null with AllowNilValues true", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		data := []byte(`null`)
		var result *string

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("decode null into pointer to pointer", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		data := []byte(`null`)
		var result **string
		ptr := &result

		err := codec.Decode(data, ptr)

		assert.NoError(t, err)
		assert.Nil(t, *ptr)
	})

	t.Run("decode null into interface", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		data := []byte(`null`)
		var result interface{}

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("decode empty data", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(``)
		var result string

		err := codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode empty data")
	})

	t.Run("decode into nil value", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`"test"`)

		err := codec.Decode(data, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode into nil value")
	})

	t.Run("decode with nil codec", func(t *testing.T) {
		var codec *cachex.JSONCodec
		data := []byte(`"test"`)
		var result string

		err := codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "codec validation failed")
	})

	t.Run("decode invalid JSON", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`invalid json`)
		var result string

		err := codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json unmarshal error")
	})
}

// TestJSONCodec_NilHandling tests nil value handling
func TestJSONCodec_NilHandling(t *testing.T) {
	t.Run("handleNullValue with pointer to string", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var result *string
		ptr := &result

		err := codec.Decode([]byte(`null`), ptr)

		assert.NoError(t, err)
		assert.Nil(t, *ptr)
	})

	t.Run("handleNullValue with pointer to struct", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		type TestStruct struct {
			Name string
		}
		var result *TestStruct
		ptr := &result

		err := codec.Decode([]byte(`null`), ptr)

		assert.NoError(t, err)
		assert.Nil(t, *ptr)
	})

	t.Run("handleNullValue with pointer to slice", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var result *[]string
		ptr := &result

		err := codec.Decode([]byte(`null`), ptr)

		assert.NoError(t, err)
		assert.Nil(t, *ptr)
	})

	t.Run("handleNullValue with pointer to map", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var result *map[string]int
		ptr := &result

		err := codec.Decode([]byte(`null`), ptr)

		assert.NoError(t, err)
		assert.Nil(t, *ptr)
	})

	t.Run("handleNullValue with non-pointer", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var result string

		err := codec.Decode([]byte(`null`), result) // Pass non-pointer value

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "destination must be a pointer")
	})

	t.Run("handleNullValue with nil pointer", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		var result *string

		err := codec.Decode([]byte(`null`), &result)

		// The codec should handle nil pointers by setting them to nil
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestJSONCodec_EdgeCases tests edge cases and error conditions
func TestJSONCodec_EdgeCases(t *testing.T) {
	t.Run("encode unsupported type", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		// Create a channel which cannot be JSON marshaled
		ch := make(chan int)

		data, err := codec.Encode(ch)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "json marshal error")
	})

	t.Run("decode into wrong type", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		data := []byte(`"string"`)
		var result int

		err := codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json unmarshal error")
	})

	t.Run("decode whitespace around null", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		data := []byte(`  null  `)
		var result *string

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("decode with tabs and newlines", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		data := []byte("\t\n  null  \t\n")
		var result *string

		err := codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("round trip encoding and decoding", func(t *testing.T) {
		codec := cachex.NewJSONCodec()
		type TestStruct struct {
			Name    string            `json:"name"`
			Age     int               `json:"age"`
			Tags    []string          `json:"tags"`
			Details map[string]string `json:"details"`
		}

		original := TestStruct{
			Name:    "Test User",
			Age:     30,
			Tags:    []string{"tag1", "tag2", "tag3"},
			Details: map[string]string{"key1": "value1", "key2": "value2"},
		}

		// Encode
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Decode
		var decoded TestStruct
		err = codec.Decode(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("round trip with nil values", func(t *testing.T) {
		codec := cachex.NewJSONCodecWithOptions(true, false)
		type TestStruct struct {
			Name    *string           `json:"name"`
			Age     int               `json:"age"`
			Tags    []string          `json:"tags"`
			Details map[string]string `json:"details"`
		}

		original := TestStruct{
			Name:    nil,
			Age:     30,
			Tags:    nil,
			Details: nil,
		}

		// Encode
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Decode
		var decoded TestStruct
		err = codec.Decode(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})
}

// TestJSONCodec_Name tests the Name method
func TestJSONCodec_Name(t *testing.T) {
	codec := cachex.NewJSONCodec()
	name := codec.Name()

	assert.Equal(t, "json", name)
}

// TestJSONCodec_InterfaceCompliance tests that JSONCodec implements the Codec interface
func TestJSONCodec_InterfaceCompliance(t *testing.T) {
	var codec cachex.Codec = cachex.NewJSONCodec()

	// Test that we can call interface methods
	data, err := codec.Encode("test")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var result string
	err = codec.Decode(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result)
}
