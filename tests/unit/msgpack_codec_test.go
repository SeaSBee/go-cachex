package unit

import (
	"testing"

	"github.com/seasbee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessagePackCodec_Creation tests the MessagePackCodec constructor functions
func TestMessagePackCodec_Creation(t *testing.T) {
	t.Run("NewMessagePackCodec default values", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		assert.NotNil(t, codec)
		assert.False(t, codec.IsJSONTagEnabled())
		assert.True(t, codec.IsCompactIntsEnabled())
		assert.True(t, codec.IsCompactFloatsEnabled())
		assert.False(t, codec.IsMapKeysSorted())
	})

	t.Run("NewMessagePackCodecWithOptions all true", func(t *testing.T) {
		codec := cachex.NewMessagePackCodecWithOptions(true, true, true, true)

		assert.NotNil(t, codec)
		assert.True(t, codec.IsJSONTagEnabled())
		assert.True(t, codec.IsCompactIntsEnabled())
		assert.True(t, codec.IsCompactFloatsEnabled())
		assert.True(t, codec.IsMapKeysSorted())
	})

	t.Run("NewMessagePackCodecWithOptions all false", func(t *testing.T) {
		codec := cachex.NewMessagePackCodecWithOptions(false, false, false, false)

		assert.NotNil(t, codec)
		assert.False(t, codec.IsJSONTagEnabled())
		assert.False(t, codec.IsCompactIntsEnabled())
		assert.False(t, codec.IsCompactFloatsEnabled())
		assert.False(t, codec.IsMapKeysSorted())
	})

	t.Run("NewMessagePackCodecWithOptions mixed", func(t *testing.T) {
		codec := cachex.NewMessagePackCodecWithOptions(true, false, true, false)

		assert.NotNil(t, codec)
		assert.True(t, codec.IsJSONTagEnabled())
		assert.False(t, codec.IsCompactIntsEnabled())
		assert.True(t, codec.IsCompactFloatsEnabled())
		assert.False(t, codec.IsMapKeysSorted())
	})

	t.Run("Name method", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		assert.Equal(t, "msgpack", codec.Name())
	})
}

// TestMessagePackCodec_Options tests the configuration options
func TestMessagePackCodec_Options(t *testing.T) {
	t.Run("UseJSONTag", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test setting to true
		codec.UseJSONTag(true)
		assert.True(t, codec.IsJSONTagEnabled())

		// Test setting to false
		codec.UseJSONTag(false)
		assert.False(t, codec.IsJSONTagEnabled())
	})

	t.Run("UseCompactInts", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test setting to false
		codec.UseCompactInts(false)
		assert.False(t, codec.IsCompactIntsEnabled())

		// Test setting to true
		codec.UseCompactInts(true)
		assert.True(t, codec.IsCompactIntsEnabled())
	})

	t.Run("UseCompactFloats", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test setting to false
		codec.UseCompactFloats(false)
		assert.False(t, codec.IsCompactFloatsEnabled())

		// Test setting to true
		codec.UseCompactFloats(true)
		assert.True(t, codec.IsCompactFloatsEnabled())
	})

	t.Run("SortMapKeys", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test setting to true
		codec.SortMapKeys(true)
		assert.True(t, codec.IsMapKeysSorted())

		// Test setting to false
		codec.SortMapKeys(false)
		assert.False(t, codec.IsMapKeysSorted())
	})

	t.Run("concurrent option changes", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Simulate concurrent access
		done := make(chan bool, 4)

		go func() {
			for i := 0; i < 100; i++ {
				codec.UseJSONTag(i%2 == 0)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				codec.UseCompactInts(i%2 == 0)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				codec.UseCompactFloats(i%2 == 0)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				codec.SortMapKeys(i%2 == 0)
			}
			done <- true
		}()

		// Wait for all goroutines to complete
		for i := 0; i < 4; i++ {
			<-done
		}

		// Verify codec is still functional
		assert.NotNil(t, codec)
		assert.Equal(t, "msgpack", codec.Name())
	})
}

// TestMessagePackCodec_Encode tests the Encode method
func TestMessagePackCodec_Encode(t *testing.T) {
	t.Run("encode string", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode("hello world")

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode integer", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode(42)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode float", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode(3.14)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode boolean", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode(true)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode slice", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode([]string{"a", "b", "c"})

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode map", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode(map[string]int{"a": 1, "b": 2})

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		codec := cachex.NewMessagePackCodec()
		testData := TestStruct{Name: "John", Age: 30}
		data, err := codec.Encode(testData)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode struct with JSON tags", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		codec := cachex.NewMessagePackCodecWithOptions(true, true, true, true)
		testData := TestStruct{Name: "John", Age: 30}
		data, err := codec.Encode(testData)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("encode nil value", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode(nil)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "cannot encode nil value")
	})

	t.Run("encode complex nested structure", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type Person struct {
			Name    string   `json:"name"`
			Age     int      `json:"age"`
			Address Address  `json:"address"`
			Hobbies []string `json:"hobbies"`
		}

		codec := cachex.NewMessagePackCodec()
		testData := Person{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
			Hobbies: []string{"reading", "swimming"},
		}

		data, err := codec.Encode(testData)

		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})
}

// TestMessagePackCodec_Decode tests the Decode method
func TestMessagePackCodec_Decode(t *testing.T) {
	t.Run("decode string", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := "hello world"
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result string
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode integer", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := 42
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result int
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode float", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := 3.14
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result float64
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode boolean", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := true
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result bool
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode slice", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := []string{"a", "b", "c"}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result []string
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode map", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// First encode
		original := map[string]int{"a": 1, "b": 2}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result map[string]int
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		codec := cachex.NewMessagePackCodec()

		// First encode
		original := TestStruct{Name: "John", Age: 30}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result TestStruct
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode struct with JSON tags", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		codec := cachex.NewMessagePackCodecWithOptions(true, true, true, true)

		// First encode
		original := TestStruct{Name: "John", Age: 30}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result TestStruct
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode complex nested structure", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type Person struct {
			Name    string   `json:"name"`
			Age     int      `json:"age"`
			Address Address  `json:"address"`
			Hobbies []string `json:"hobbies"`
		}

		codec := cachex.NewMessagePackCodec()

		// First encode
		original := Person{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
			Hobbies: []string{"reading", "swimming"},
		}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		// Then decode
		var result Person
		err = codec.Decode(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("decode nil data", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		var result string
		err := codec.Decode(nil, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode nil data")
	})

	t.Run("decode empty data", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		var result string
		err := codec.Decode([]byte{}, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode empty data")
	})

	t.Run("decode into nil value", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()
		data, err := codec.Encode("test")
		require.NoError(t, err)

		err = codec.Decode(data, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode into nil value")
	})
}

// TestMessagePackCodec_RoundTrip tests encode-decode round trips
func TestMessagePackCodec_RoundTrip(t *testing.T) {
	t.Run("round trip with different options", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		testData := TestStruct{Name: "John", Age: 30}

		// Test with default options
		codec1 := cachex.NewMessagePackCodec()
		data1, err := codec1.Encode(testData)
		require.NoError(t, err)

		var result1 TestStruct
		err = codec1.Decode(data1, &result1)
		assert.NoError(t, err)
		assert.Equal(t, testData, result1)

		// Test with JSON tags enabled
		codec2 := cachex.NewMessagePackCodecWithOptions(true, true, true, true)
		data2, err := codec2.Encode(testData)
		require.NoError(t, err)

		var result2 TestStruct
		err = codec2.Decode(data2, &result2)
		assert.NoError(t, err)
		assert.Equal(t, testData, result2)
	})

	t.Run("round trip with compact options", func(t *testing.T) {
		codec := cachex.NewMessagePackCodecWithOptions(false, true, true, false)

		// Test with integers
		originalInt := 12345
		data, err := codec.Encode(originalInt)
		require.NoError(t, err)

		var resultInt int
		err = codec.Decode(data, &resultInt)
		assert.NoError(t, err)
		assert.Equal(t, originalInt, resultInt)

		// Test with floats
		originalFloat := 3.14159
		data, err = codec.Encode(originalFloat)
		require.NoError(t, err)

		var resultFloat float64
		err = codec.Decode(data, &resultFloat)
		assert.NoError(t, err)
		assert.Equal(t, originalFloat, resultFloat)
	})

	t.Run("round trip with map key sorting", func(t *testing.T) {
		codec := cachex.NewMessagePackCodecWithOptions(false, false, false, true)

		original := map[string]int{"z": 1, "a": 2, "m": 3}
		data, err := codec.Encode(original)
		require.NoError(t, err)

		var result map[string]int
		err = codec.Decode(data, &result)
		assert.NoError(t, err)
		assert.Equal(t, original, result)
	})
}

// TestMessagePackCodec_EdgeCases tests edge cases and error conditions
func TestMessagePackCodec_EdgeCases(t *testing.T) {
	t.Run("encode unsupported type", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Create a channel (unsupported by MessagePack)
		ch := make(chan int)
		data, err := codec.Encode(ch)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "msgpack marshal error")
	})

	t.Run("decode invalid data", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Try to decode invalid MessagePack data
		invalidData := []byte("invalid msgpack data")
		var result string
		err := codec.Decode(invalidData, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "msgpack unmarshal error")
	})

	t.Run("decode into wrong type", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Encode a string
		data, err := codec.Encode("hello")
		require.NoError(t, err)

		// Try to decode into an int
		var result int
		err = codec.Decode(data, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "msgpack unmarshal error")
	})

	t.Run("large data encoding", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Create a large slice
		largeSlice := make([]int, 10000)
		for i := range largeSlice {
			largeSlice[i] = i
		}

		data, err := codec.Encode(largeSlice)
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)

		// Decode it back
		var result []int
		err = codec.Decode(data, &result)
		assert.NoError(t, err)
		assert.Equal(t, largeSlice, result)
	})

	t.Run("empty slice and map", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test empty slice
		emptySlice := []string{}
		data, err := codec.Encode(emptySlice)
		assert.NoError(t, err)

		var resultSlice []string
		err = codec.Decode(data, &resultSlice)
		assert.NoError(t, err)
		assert.Equal(t, emptySlice, resultSlice)

		// Test empty map
		emptyMap := map[string]int{}
		data, err = codec.Encode(emptyMap)
		assert.NoError(t, err)

		var resultMap map[string]int
		err = codec.Decode(data, &resultMap)
		assert.NoError(t, err)
		assert.Equal(t, emptyMap, resultMap)
	})

	t.Run("special values", func(t *testing.T) {
		codec := cachex.NewMessagePackCodec()

		// Test zero values
		zeroValues := struct {
			String string
			Int    int
			Float  float64
			Bool   bool
		}{}

		data, err := codec.Encode(zeroValues)
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})
}

// Benchmark tests for MessagePack codec
func BenchmarkMessagePackCodec_Encode(b *testing.B) {
	codec := cachex.NewMessagePackCodec()
	testData := map[string]interface{}{
		"string": "test value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"array":  []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessagePackCodec_Decode(b *testing.B) {
	codec := cachex.NewMessagePackCodec()
	testData := map[string]interface{}{
		"string": "test value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"array":  []int{1, 2, 3, 4, 5},
	}

	data, err := codec.Encode(testData)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := codec.Decode(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessagePackCodec_EncodeDecode(b *testing.B) {
	codec := cachex.NewMessagePackCodec()
	testData := map[string]interface{}{
		"string": "test value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"array":  []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := codec.Encode(testData)
		if err != nil {
			b.Fatal(err)
		}

		var result map[string]interface{}
		err = codec.Decode(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessagePackCodec_WithOptions(b *testing.B) {
	codec := cachex.NewMessagePackCodecWithOptions(true, true, true, true)
	testData := map[string]interface{}{
		"string": "test value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"array":  []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := codec.Encode(testData)
		if err != nil {
			b.Fatal(err)
		}

		var result map[string]interface{}
		err = codec.Decode(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}
