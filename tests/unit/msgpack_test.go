package unit

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// TestStruct for testing struct encoding/decoding
type TestStruct struct {
	ID     int     `json:"id" msgpack:"id"`
	Name   string  `json:"name" msgpack:"name"`
	Email  string  `json:"email" msgpack:"email"`
	Active bool    `json:"active" msgpack:"active"`
	Score  float64 `json:"score" msgpack:"score"`
}

// TestStructNoTags for testing struct encoding/decoding without tags
type TestStructNoTags struct {
	ID    int
	Name  string
	Email string
}

func TestNewMessagePackCodec(t *testing.T) {
	codec := cachex.NewMessagePackCodec()
	if codec == nil {
		t.Errorf("NewMessagePackCodec() should not return nil")
	}

	if codec.IsJSONTagEnabled() {
		t.Errorf("NewMessagePackCodec() should have JSON tags disabled by default")
	}

	if codec.Name() != "msgpack" {
		t.Errorf("Name() should return 'msgpack', got %s", codec.Name())
	}
}

func TestNewMessagePackCodecWithOptions(t *testing.T) {
	tests := []struct {
		name       string
		useJSONTag bool
	}{
		{
			name:       "with JSON tags enabled",
			useJSONTag: true,
		},
		{
			name:       "with JSON tags disabled",
			useJSONTag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := cachex.NewMessagePackCodecWithOptions(tt.useJSONTag)
			if codec == nil {
				t.Errorf("NewMessagePackCodecWithOptions() should not return nil")
			}

			if codec.IsJSONTagEnabled() != tt.useJSONTag {
				t.Errorf("IsJSONTagEnabled() should return %v, got %v", tt.useJSONTag, codec.IsJSONTagEnabled())
			}
		})
	}
}

func TestMessagePackCodec_Name(t *testing.T) {
	codec := cachex.NewMessagePackCodec()
	if codec.Name() != "msgpack" {
		t.Errorf("Name() should return 'msgpack', got %s", codec.Name())
	}
}

func TestMessagePackCodec_UseJSONTag(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	// Initially disabled
	if codec.IsJSONTagEnabled() {
		t.Errorf("JSON tags should be disabled by default")
	}

	// Enable JSON tags
	codec.UseJSONTag(true)
	if !codec.IsJSONTagEnabled() {
		t.Errorf("JSON tags should be enabled after UseJSONTag(true)")
	}

	// Disable JSON tags
	codec.UseJSONTag(false)
	if codec.IsJSONTagEnabled() {
		t.Errorf("JSON tags should be disabled after UseJSONTag(false)")
	}
}

func TestMessagePackCodec_Encode_Success(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name     string
		input    any
		expected bool // Just check if encoding succeeds
	}{
		{
			name:     "string",
			input:    "test string",
			expected: true,
		},
		{
			name:     "integer",
			input:    42,
			expected: true,
		},
		{
			name:     "float",
			input:    3.14159,
			expected: true,
		},
		{
			name:     "boolean true",
			input:    true,
			expected: true,
		},
		{
			name:     "boolean false",
			input:    false,
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "zero",
			input:    0,
			expected: true,
		},
		{
			name:     "negative number",
			input:    -42,
			expected: true,
		},
		{
			name:     "array",
			input:    []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "map",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			expected: true,
		},
		{
			name: "struct",
			input: TestStruct{
				ID:     1,
				Name:   "John",
				Email:  "john@example.com",
				Active: true,
				Score:  95.5,
			},
			expected: true,
		},
		{
			name: "pointer to struct",
			input: &TestStruct{
				ID:     2,
				Name:   "Jane",
				Email:  "jane@example.com",
				Active: false,
				Score:  88.0,
			},
			expected: true,
		},
		{
			name:     "empty struct",
			input:    struct{}{},
			expected: true,
		},
		{
			name:     "time",
			input:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name: "complex nested structure",
			input: map[string]any{
				"user": TestStruct{ID: 1, Name: "John", Email: "john@example.com"},
				"tags": []string{"admin", "user"},
				"meta": map[string]any{
					"active": true,
					"count":  42,
					"nested": map[string]string{"key": "value"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := codec.Encode(tt.input)
			if tt.expected && err != nil {
				t.Errorf("Encode() failed: %v", err)
			} else if !tt.expected && err == nil {
				t.Errorf("Encode() should have failed")
			}

			if tt.expected && len(data) == 0 {
				t.Errorf("Encode() returned empty data")
			}
		})
	}
}

func TestMessagePackCodec_Encode_Errors(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "nil value",
			input: nil,
		},
		{
			name:  "channel (unmarshalable)",
			input: make(chan int),
		},
		{
			name:  "function (unmarshalable)",
			input: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.Encode(tt.input)
			if err == nil {
				t.Errorf("Encode() should fail for %s", tt.name)
			}
		})
	}
}

func TestMessagePackCodec_Decode_Success(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name   string
		input  any
		target any
	}{
		{
			name:   "string",
			input:  "test string",
			target: new(string),
		},
		{
			name:   "integer",
			input:  42,
			target: new(int),
		},
		{
			name:   "float",
			input:  3.14159,
			target: new(float64),
		},
		{
			name:   "boolean",
			input:  true,
			target: new(bool),
		},
		{
			name:   "array",
			input:  []string{"a", "b", "c"},
			target: new([]string),
		},
		{
			name:   "map",
			input:  map[string]int{"a": 1, "b": 2},
			target: new(map[string]int),
		},
		{
			name: "struct",
			input: TestStruct{
				ID:     1,
				Name:   "John",
				Email:  "john@example.com",
				Active: true,
				Score:  95.5,
			},
			target: new(TestStruct),
		},
		{
			name:   "time",
			input:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			target: new(time.Time),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First encode the input
			data, err := codec.Encode(tt.input)
			if err != nil {
				t.Fatalf("Encode() failed: %v", err)
			}

			// Then decode it
			err = codec.Decode(data, tt.target)
			if err != nil {
				t.Errorf("Decode() failed: %v", err)
			}

			// For basic types, we can compare values
			switch v := tt.target.(type) {
			case *string:
				if *v != tt.input.(string) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *int:
				// MessagePack might decode to int64, so we need flexible comparison
				if int64(*v) != int64(tt.input.(int)) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *float64:
				if *v != tt.input.(float64) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *bool:
				if *v != tt.input.(bool) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *[]string:
				if !reflect.DeepEqual(*v, tt.input.([]string)) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *map[string]int:
				if !reflect.DeepEqual(*v, tt.input.(map[string]int)) {
					t.Errorf("Decode() returned wrong value: got %v, want %v", *v, tt.input)
				}
			case *TestStruct:
				if !reflect.DeepEqual(*v, tt.input.(TestStruct)) {
					t.Errorf("Decode() returned wrong value: got %+v, want %+v", *v, tt.input)
				}
			case *time.Time:
				// Time comparison might need some tolerance
				expected := tt.input.(time.Time)
				if !v.Equal(expected) {
					t.Errorf("Decode() returned wrong time: got %v, want %v", *v, expected)
				}
			}
		})
	}
}

func TestMessagePackCodec_Decode_Errors(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name        string
		data        []byte
		target      any
		expectError bool
	}{
		{
			name:        "empty data",
			data:        []byte{},
			target:      new(string),
			expectError: true,
		},
		{
			name:        "nil target",
			data:        []byte{0x91, 0x01}, // Valid msgpack data
			target:      nil,
			expectError: true,
		},
		{
			name:        "invalid msgpack data",
			data:        []byte{0xFF, 0xFF, 0xFF},
			target:      new(string),
			expectError: true,
		},
		{
			name: "wrong type for target",
			data: func() []byte {
				// Encode a string
				data, _ := codec.Encode("test string")
				return data
			}(),
			target:      new(int),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Decode(tt.data, tt.target)
			if tt.expectError && err == nil {
				t.Errorf("Decode() should fail for %s", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Decode() should not fail for %s: %v", tt.name, err)
			}
		})
	}
}

func TestMessagePackCodec_EncodeDecode_RoundTrip(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "simple string",
			input: "test string",
		},
		{
			name:  "integer",
			input: 42,
		},
		{
			name:  "float",
			input: 3.14159,
		},
		{
			name:  "boolean",
			input: true,
		},
		{
			name:  "array",
			input: []string{"a", "b", "c"},
		},
		{
			name:  "map",
			input: map[string]int{"a": 1, "b": 2, "c": 3},
		},
		{
			name: "struct",
			input: TestStruct{
				ID:     1,
				Name:   "John",
				Email:  "john@example.com",
				Active: true,
				Score:  95.5,
			},
		},
		{
			name: "pointer_to_struct",
			input: &TestStruct{
				ID:     2,
				Name:   "Jane",
				Email:  "jane@example.com",
				Active: false,
				Score:  88.0,
			},
		},
		{
			name: "complex_nested_structure",
			input: map[string]any{
				"user": TestStruct{ID: 1, Name: "John", Email: "john@example.com"},
				"tags": []string{"admin", "user"},
				"meta": map[string]any{
					"active": true,
					"count":  42,
					"nested": map[string]string{"key": "value"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := codec.Encode(tt.input)
			if err != nil {
				t.Errorf("Encode() failed: %v", err)
				return
			}

			// Decode into appropriate type
			var decoded any
			switch tt.input.(type) {
			case string:
				var s string
				err = codec.Decode(encoded, &s)
				decoded = s
			case int:
				var i int
				err = codec.Decode(encoded, &i)
				decoded = i
			case float64:
				var f float64
				err = codec.Decode(encoded, &f)
				decoded = f
			case bool:
				var b bool
				err = codec.Decode(encoded, &b)
				decoded = b
			case []string:
				var arr []string
				err = codec.Decode(encoded, &arr)
				decoded = arr
			case map[string]int:
				var m map[string]int
				err = codec.Decode(encoded, &m)
				decoded = m
			case TestStruct:
				var s TestStruct
				err = codec.Decode(encoded, &s)
				decoded = s
			case *TestStruct:
				var s TestStruct
				err = codec.Decode(encoded, &s)
				decoded = s // Compare with dereferenced original
				tt.input = *(tt.input.(*TestStruct))
			default:
				// For complex types, decode into interface{}
				err = codec.Decode(encoded, &decoded)
			}

			if err != nil {
				t.Errorf("Decode() failed: %v", err)
				return
			}

			// For complex types, we can't easily compare, so just verify no error
			if tt.name == "complex_nested_structure" {
				if decoded == nil {
					t.Errorf("Decode() returned nil for complex structure")
				}
				return
			}

			// For other types, compare values
			if !reflect.DeepEqual(decoded, tt.input) {
				t.Errorf("Round trip failed: got %+v, want %+v", decoded, tt.input)
			}
		})
	}
}

func TestMessagePackCodec_JSONTagsEnabled(t *testing.T) {
	codec := cachex.NewMessagePackCodecWithOptions(true) // Enable JSON tags

	// Create a struct with JSON tags
	input := TestStruct{
		ID:     1,
		Name:   "John",
		Email:  "john@example.com",
		Active: true,
		Score:  95.5,
	}

	// Encode
	data, err := codec.Encode(input)
	if err != nil {
		t.Errorf("Encode() failed: %v", err)
		return
	}

	// Decode
	var output TestStruct
	err = codec.Decode(data, &output)
	if err != nil {
		t.Errorf("Decode() failed: %v", err)
		return
	}

	// Verify the struct was properly encoded/decoded
	if !reflect.DeepEqual(input, output) {
		t.Errorf("Round trip with JSON tags failed: got %+v, want %+v", output, input)
	}
}

func TestMessagePackCodec_JSONTagsDisabled(t *testing.T) {
	codec := cachex.NewMessagePackCodecWithOptions(false) // Disable JSON tags

	// Create a struct without JSON tags
	input := TestStructNoTags{
		ID:    1,
		Name:  "John",
		Email: "john@example.com",
	}

	// Encode
	data, err := codec.Encode(input)
	if err != nil {
		t.Errorf("Encode() failed: %v", err)
		return
	}

	// Decode
	var output TestStructNoTags
	err = codec.Decode(data, &output)
	if err != nil {
		t.Errorf("Decode() failed: %v", err)
		return
	}

	// Verify the struct was properly encoded/decoded
	if !reflect.DeepEqual(input, output) {
		t.Errorf("Round trip without JSON tags failed: got %+v, want %+v", output, input)
	}
}

func TestMessagePackCodec_Concurrency(t *testing.T) {
	codec := cachex.NewMessagePackCodec()
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Test concurrent encoding
			data := map[string]any{
				"id":   id,
				"name": fmt.Sprintf("test-%d", id),
				"data": []int{1, 2, 3, id},
			}

			encoded, err := codec.Encode(data)
			if err != nil {
				t.Errorf("Concurrent Encode() failed: %v", err)
				done <- true
				return
			}

			// Test concurrent decoding
			var decoded map[string]any
			err = codec.Decode(encoded, &decoded)
			if err != nil {
				t.Errorf("Concurrent Decode() failed: %v", err)
				done <- true
				return
			}

			// Verify basic fields
			if decoded["name"] != fmt.Sprintf("test-%d", id) {
				t.Errorf("Concurrent operation failed: expected name=test-%d, got %v", id, decoded["name"])
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestMessagePackCodec_EdgeCases(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:  "zero value int",
			input: 0,
		},
		{
			name:  "zero value float",
			input: 0.0,
		},
		{
			name:  "zero value string",
			input: "",
		},
		{
			name:  "zero value bool",
			input: false,
		},
		{
			name:  "empty array",
			input: []string{},
		},
		{
			name:  "empty map",
			input: map[string]any{},
		},
		{
			name:  "unicode string",
			input: "Hello 世界 🌍",
		},
		{
			name:  "special characters",
			input: "Hello\n\t\r\"\\",
		},
		{
			name:  "very large number",
			input: int64(9223372036854775807), // Max int64
		},
		{
			name:  "very small number",
			input: int64(-9223372036854775808), // Min int64
		},
		{
			name:  "very large float",
			input: 1.7976931348623157e+308,
		},
		{
			name:  "very small float",
			input: 2.2250738585072014e-308,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := codec.Encode(tt.input)
			if err != nil {
				t.Errorf("Encode() failed for %s: %v", tt.name, err)
				return
			}

			if len(encoded) == 0 {
				t.Errorf("Encode() returned empty data for %s", tt.name)
				return
			}

			// Decode back into appropriate type
			switch v := tt.input.(type) {
			case string:
				var decoded string
				err = codec.Decode(encoded, &decoded)
				if err != nil {
					t.Errorf("Decode() failed for %s: %v", tt.name, err)
				} else if decoded != v {
					t.Errorf("Round trip failed for %s: got %v, want %v", tt.name, decoded, v)
				}
			case int, int64:
				var decoded int64
				err = codec.Decode(encoded, &decoded)
				if err != nil {
					t.Errorf("Decode() failed for %s: %v", tt.name, err)
				} else {
					expected := reflect.ValueOf(v).Int()
					if decoded != expected {
						t.Errorf("Round trip failed for %s: got %v, want %v", tt.name, decoded, expected)
					}
				}
			case float64:
				var decoded float64
				err = codec.Decode(encoded, &decoded)
				if err != nil {
					t.Errorf("Decode() failed for %s: %v", tt.name, err)
				} else if decoded != v {
					t.Errorf("Round trip failed for %s: got %v, want %v", tt.name, decoded, v)
				}
			case bool:
				var decoded bool
				err = codec.Decode(encoded, &decoded)
				if err != nil {
					t.Errorf("Decode() failed for %s: %v", tt.name, err)
				} else if decoded != v {
					t.Errorf("Round trip failed for %s: got %v, want %v", tt.name, decoded, v)
				}
			default:
				// For complex types, just verify they decode without error
				var decoded any
				err = codec.Decode(encoded, &decoded)
				if err != nil {
					t.Errorf("Decode() failed for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestMessagePackCodec_Performance(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	// Test data
	data := map[string]any{
		"id":     12345,
		"name":   "Performance Test",
		"email":  "test@example.com",
		"active": true,
		"score":  95.7,
		"tags":   []string{"admin", "user", "tester"},
		"metadata": map[string]any{
			"created": time.Now(),
			"updated": time.Now(),
			"version": 2,
		},
	}

	// Encode multiple times
	numOperations := 1000
	start := time.Now()

	for i := 0; i < numOperations; i++ {
		_, err := codec.Encode(data)
		if err != nil {
			t.Errorf("Performance test encode failed: %v", err)
			return
		}
	}

	encodeTime := time.Since(start)

	// Encode once to get data for decode test
	encoded, err := codec.Encode(data)
	if err != nil {
		t.Errorf("Failed to encode for decode test: %v", err)
		return
	}

	// Decode multiple times
	start = time.Now()

	for i := 0; i < numOperations; i++ {
		var decoded map[string]any
		err := codec.Decode(encoded, &decoded)
		if err != nil {
			t.Errorf("Performance test decode failed: %v", err)
			return
		}
	}

	decodeTime := time.Since(start)

	t.Logf("Performance test completed:")
	t.Logf("  Encode %d operations: %v (%.2f ops/ms)", numOperations, encodeTime, float64(numOperations)/float64(encodeTime.Milliseconds()))
	t.Logf("  Decode %d operations: %v (%.2f ops/ms)", numOperations, decodeTime, float64(numOperations)/float64(decodeTime.Milliseconds()))

	// Reasonable performance expectations (adjust based on requirements)
	if encodeTime > 5*time.Second {
		t.Errorf("Encode performance too slow: %v for %d operations", encodeTime, numOperations)
	}
	if decodeTime > 5*time.Second {
		t.Errorf("Decode performance too slow: %v for %d operations", decodeTime, numOperations)
	}
}

func TestMessagePackCodec_ErrorMessages(t *testing.T) {
	codec := cachex.NewMessagePackCodec()

	tests := []struct {
		name        string
		operation   string
		input       any
		data        []byte
		target      any
		expectedMsg string
	}{
		{
			name:        "encode nil",
			operation:   "encode",
			input:       nil,
			expectedMsg: "cannot encode nil value",
		},
		{
			name:        "decode empty data",
			operation:   "decode",
			data:        []byte{},
			target:      new(string),
			expectedMsg: "cannot decode empty data",
		},
		{
			name:        "decode nil target",
			operation:   "decode",
			data:        []byte{0x91, 0x01},
			target:      nil,
			expectedMsg: "cannot decode into nil value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.operation == "encode" {
				_, err = codec.Encode(tt.input)
			} else {
				err = codec.Decode(tt.data, tt.target)
			}

			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
				return
			}

			if tt.expectedMsg != "" && !contains(err.Error(), tt.expectedMsg) {
				t.Errorf("Error message should contain '%s', got: %s", tt.expectedMsg, err.Error())
			}
		})
	}
}

func TestMessagePackCodec_CompactVsJSON(t *testing.T) {
	msgpackCodec := cachex.NewMessagePackCodec()
	jsonCodec := cachex.NewJSONCodec()

	// Test data
	data := TestStruct{
		ID:     12345,
		Name:   "Comparison Test",
		Email:  "test@example.com",
		Active: true,
		Score:  95.7,
	}

	// Encode with MessagePack
	msgpackData, err := msgpackCodec.Encode(data)
	if err != nil {
		t.Errorf("MessagePack encode failed: %v", err)
		return
	}

	// Encode with JSON
	jsonData, err := jsonCodec.Encode(data)
	if err != nil {
		t.Errorf("JSON encode failed: %v", err)
		return
	}

	t.Logf("Data size comparison:")
	t.Logf("  MessagePack: %d bytes", len(msgpackData))
	t.Logf("  JSON: %d bytes", len(jsonData))

	// MessagePack should typically be more compact than JSON
	if len(msgpackData) >= len(jsonData) {
		t.Logf("Note: MessagePack (%d bytes) is not smaller than JSON (%d bytes) for this data", len(msgpackData), len(jsonData))
	}

	// Verify both can decode correctly
	var msgpackResult TestStruct
	err = msgpackCodec.Decode(msgpackData, &msgpackResult)
	if err != nil {
		t.Errorf("MessagePack decode failed: %v", err)
	}

	var jsonResult TestStruct
	err = jsonCodec.Decode(jsonData, &jsonResult)
	if err != nil {
		t.Errorf("JSON decode failed: %v", err)
	}

	// Both should produce the same result
	if !reflect.DeepEqual(msgpackResult, jsonResult) {
		t.Errorf("MessagePack and JSON produced different results: msgpack=%+v, json=%+v", msgpackResult, jsonResult)
	}

	// Both should match original
	if !reflect.DeepEqual(data, msgpackResult) {
		t.Errorf("MessagePack result doesn't match original: got %+v, want %+v", msgpackResult, data)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
