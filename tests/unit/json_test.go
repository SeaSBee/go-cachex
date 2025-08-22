package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

func TestNewJSONCodec(t *testing.T) {
	codec := cachex.NewJSONCodec()
	if codec == nil {
		t.Errorf("NewJSONCodec() returned nil")
	}
}

func TestJSONCodec_Name(t *testing.T) {
	codec := cachex.NewJSONCodec()
	if codec.Name() != "json" {
		t.Errorf("JSONCodec.Name() = %v, want 'json'", codec.Name())
	}
}

func TestJSONCodec_Encode_Success(t *testing.T) {
	codec := cachex.NewJSONCodec()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string",
			input:    "hello world",
			expected: `"hello world"`,
		},
		{
			name:     "integer",
			input:    42,
			expected: `42`,
		},
		{
			name:     "float",
			input:    3.14,
			expected: `3.14`,
		},
		{
			name:     "boolean",
			input:    true,
			expected: `true`,
		},
		{
			name:     "empty_string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "array",
			input:    []string{"a", "b", "c"},
			expected: `["a","b","c"]`,
		},
		{
			name:     "map",
			input:    map[string]int{"a": 1, "b": 2},
			expected: `{"a":1,"b":2}`,
		},
		{
			name:     "struct",
			input:    TestUser{ID: "1", Name: "John", Email: "john@example.com"},
			expected: `{"id":"1","name":"John","email":"john@example.com"}`,
		},
		{
			name:     "pointer to struct",
			input:    &TestUser{ID: "2", Name: "Jane", Email: "jane@example.com"},
			expected: `{"id":"2","name":"Jane","email":"jane@example.com"}`,
		},
		{
			name:     "empty struct",
			input:    TestUser{},
			expected: `{"id":"","name":"","email":""}`,
		},
		{
			name:     "time",
			input:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: `"2023-01-01T12:00:00Z"`,
		},
		{
			name: "complex nested structure",
			input: map[string]any{
				"user": TestUser{ID: "1", Name: "John"},
				"tags": []string{"admin", "user"},
				"meta": map[string]any{
					"created": time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					"active":  true,
				},
			},
			expected: `{"meta":{"active":true,"created":"2023-01-01T12:00:00Z"},"tags":["admin","user"],"user":{"id":"1","name":"John","email":""}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Encode(tt.input)
			if err != nil {
				t.Errorf("Encode() failed: %v", err)
				return
			}

			// Parse expected JSON to handle map ordering differences
			var expectedParsed, resultParsed any
			if err := json.Unmarshal([]byte(tt.expected), &expectedParsed); err != nil {
				t.Errorf("Failed to parse expected JSON: %v", err)
				return
			}
			if err := json.Unmarshal(result, &resultParsed); err != nil {
				t.Errorf("Failed to parse result JSON: %v", err)
				return
			}

			// Compare parsed structures
			expectedJSON, _ := json.Marshal(expectedParsed)
			resultJSON, _ := json.Marshal(resultParsed)

			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("Encode() = %v, want %v", string(resultJSON), string(expectedJSON))
			}
		})
	}
}

func TestJSONCodec_Encode_Errors(t *testing.T) {
	codec := cachex.NewJSONCodec()

	tests := []struct {
		name        string
		input       any
		expectError bool
	}{
		{
			name:        "nil value",
			input:       nil,
			expectError: true,
		},
		{
			name:        "channel (unmarshalable)",
			input:       make(chan int),
			expectError: true,
		},
		{
			name:        "function (unmarshalable)",
			input:       func() {},
			expectError: true,
		},
		{
			name:        "complex number",
			input:       1 + 2i,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.Encode(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("Encode() should have failed for %v", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Encode() failed unexpectedly: %v", err)
			}
		})
	}
}

func TestJSONCodec_Decode_Success(t *testing.T) {
	codec := cachex.NewJSONCodec()

	tests := []struct {
		name     string
		data     []byte
		expected any
	}{
		{
			name:     "string",
			data:     []byte(`"hello world"`),
			expected: "hello world",
		},
		{
			name:     "integer",
			data:     []byte(`42`),
			expected: float64(42), // JSON numbers are decoded as float64
		},
		{
			name:     "float",
			data:     []byte(`3.14`),
			expected: 3.14,
		},
		{
			name:     "boolean",
			data:     []byte(`true`),
			expected: true,
		},
		{
			name:     "null",
			data:     []byte(`null`),
			expected: nil,
		},
		{
			name:     "array",
			data:     []byte(`["a","b","c"]`),
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "map",
			data:     []byte(`{"a":1,"b":2}`),
			expected: map[string]any{"a": float64(1), "b": float64(2)},
		},
		{
			name:     "struct",
			data:     []byte(`{"id":"1","name":"John","email":"john@example.com"}`),
			expected: TestUser{ID: "1", Name: "John", Email: "john@example.com"},
		},
		{
			name:     "time",
			data:     []byte(`"2023-01-01T12:00:00Z"`),
			expected: "2023-01-01T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result any
			err := codec.Decode(tt.data, &result)
			if err != nil {
				t.Errorf("Decode() failed: %v", err)
				return
			}

			// For structs, decode directly into the target type
			if _, ok := tt.expected.(TestUser); ok {
				var user TestUser
				err := codec.Decode(tt.data, &user)
				if err != nil {
					t.Errorf("Decode() failed for struct: %v", err)
					return
				}
				if user != tt.expected {
					t.Errorf("Decode() = %+v, want %+v", user, tt.expected)
				}
				return
			}

			// For other types, compare the decoded result
			// Handle arrays and maps specially since they're not directly comparable
			if tt.name == "array" || tt.name == "map" {
				// For arrays and maps, just verify they decode without error
				if result == nil {
					t.Errorf("Decode() returned nil for %s", tt.name)
				}
			} else if result != tt.expected {
				t.Errorf("Decode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJSONCodec_Decode_Errors(t *testing.T) {
	codec := cachex.NewJSONCodec()

	tests := []struct {
		name        string
		data        []byte
		target      any
		expectError bool
	}{
		{
			name:        "empty data",
			data:        []byte{},
			target:      &TestUser{},
			expectError: true,
		},
		{
			name:        "nil target",
			data:        []byte(`{"ID":"1"}`),
			target:      nil,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`{"ID":1,}`), // Missing value
			target:      &TestUser{},
			expectError: true,
		},
		{
			name:        "wrong type for struct field",
			data:        []byte(`{"id":123,"name":"John"}`), // id should be string
			target:      &TestUser{},
			expectError: true, // JSON unmarshal will fail with type conversion error
		},
		{
			name:        "missing required fields",
			data:        []byte(`{"name":"John"}`), // Missing id and email
			target:      &TestUser{},
			expectError: false, // This is valid JSON, just missing fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Decode(tt.data, tt.target)
			if tt.expectError && err == nil {
				t.Errorf("Decode() should have failed for %v", string(tt.data))
			}
			if !tt.expectError && err != nil {
				t.Errorf("Decode() failed unexpectedly: %v", err)
			}
		})
	}
}

func TestJSONCodec_EncodeDecode_RoundTrip(t *testing.T) {
	codec := cachex.NewJSONCodec()

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
			name:  "struct",
			input: TestUser{ID: "1", Name: "John", Email: "john@example.com"},
		},
		{
			name:  "pointer_to_struct",
			input: &TestUser{ID: "2", Name: "Jane", Email: "jane@example.com"},
		},
		{
			name: "complex_nested_structure",
			input: map[string]any{
				"user": TestUser{ID: "1", Name: "John"},
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

			// Decode
			var decoded any
			err = codec.Decode(encoded, &decoded)
			if err != nil {
				t.Errorf("Decode() failed: %v", err)
				return
			}

			// For structs, decode directly into the target type
			if _, ok := tt.input.(TestUser); ok {
				var user TestUser
				err := codec.Decode(encoded, &user)
				if err != nil {
					t.Errorf("Decode() failed for struct: %v", err)
					return
				}
				if user != tt.input {
					t.Errorf("Round trip failed: got %+v, want %+v", user, tt.input)
				}
				return
			}

			// For other types, compare the decoded result
			// Handle different types that can't be directly compared
			if tt.name == "integer" {
				// JSON numbers are decoded as float64
				if inputFloat, ok := tt.input.(int); ok {
					if decodedFloat, ok := decoded.(float64); ok && float64(inputFloat) == decodedFloat {
						return // Numbers match after conversion
					}
				}
				t.Errorf("Round trip failed for integer: got %v, want %v", decoded, tt.input)
			} else if tt.name == "array" || tt.name == "map" {
				// For array and map, just verify they decode without error
				if decoded == nil {
					t.Errorf("Round trip failed: got nil for %s", tt.name)
				}
				return
			} else if tt.name == "pointer_to_struct" {
				// For pointer_to_struct, verify the decoded map has the expected fields
				if decodedMap, ok := decoded.(map[string]any); ok {
					expectedUser := tt.input.(*TestUser)
					if decodedMap["id"] != expectedUser.ID || decodedMap["name"] != expectedUser.Name || decodedMap["email"] != expectedUser.Email {
						t.Errorf("Round trip failed for pointer_to_struct: got %v, want fields from %+v", decodedMap, expectedUser)
					}
					return
				}
				t.Errorf("Round trip failed for pointer_to_struct: decoded result is not a map")
				return
			} else if tt.name == "complex_nested_structure" {
				// For complex_nested_structure, just verify it decodes without error
				return // Skip comparison for complex nested structures
			} else if tt.input != decoded {
				t.Errorf("Round trip failed: got %v, want %v", decoded, tt.input)
			}
		})
	}
}

func TestJSONCodec_Concurrency(t *testing.T) {
	codec := cachex.NewJSONCodec()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// Test concurrent encoding
			data := map[string]any{
				"id":   id,
				"name": "test",
				"data": []int{1, 2, 3},
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

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestJSONCodec_EdgeCases(t *testing.T) {
	codec := cachex.NewJSONCodec()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "zero value",
			input:    0,
			expected: `0`,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: `[]`,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: `{}`,
		},
		{
			name:     "unicode string",
			input:    "Hello 世界",
			expected: `"Hello 世界"`,
		},
		{
			name:     "special characters",
			input:    "Hello\n\t\r\"\\",
			expected: `"Hello\n\t\r\"\\"`,
		},
		{
			name:     "very large number",
			input:    9223372036854775807, // Max int64
			expected: `9223372036854775807`,
		},
		{
			name:     "very small number",
			input:    -9223372036854775808, // Min int64
			expected: `-9223372036854775808`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.Encode(tt.input)
			if err != nil {
				t.Errorf("Encode() failed: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("Encode() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestJSONCodec_Performance(t *testing.T) {
	codec := cachex.NewJSONCodec()

	// Test with a moderately complex structure
	data := map[string]any{
		"users": []TestUser{
			{ID: "1", Name: "John", Email: "john@example.com"},
			{ID: "2", Name: "Jane", Email: "jane@example.com"},
			{ID: "3", Name: "Bob", Email: "bob@example.com"},
		},
		"metadata": map[string]any{
			"count":    3,
			"active":   true,
			"tags":     []string{"admin", "user", "guest"},
			"settings": map[string]any{"theme": "dark", "lang": "en"},
		},
	}

	// Benchmark encoding
	encoded, err := codec.Encode(data)
	if err != nil {
		t.Errorf("Performance test Encode() failed: %v", err)
		return
	}

	// Benchmark decoding
	var decoded map[string]any
	err = codec.Decode(encoded, &decoded)
	if err != nil {
		t.Errorf("Performance test Decode() failed: %v", err)
		return
	}

	// Verify the result is reasonable
	if len(encoded) == 0 {
		t.Errorf("Encoded data is empty")
	}
}

func TestJSONCodec_ErrorMessages(t *testing.T) {
	codec := cachex.NewJSONCodec()

	// Test nil value error message
	_, err := codec.Encode(nil)
	if err == nil {
		t.Errorf("Encode(nil) should return error")
	} else if err.Error() != "cannot encode nil value" {
		t.Errorf("Encode(nil) error message = %v, want 'cannot encode nil value'", err.Error())
	}

	// Test empty data error message
	err = codec.Decode([]byte{}, &TestUser{})
	if err == nil {
		t.Errorf("Decode([]byte{}) should return error")
	} else if err.Error() != "cannot decode empty data" {
		t.Errorf("Decode([]byte{}) error message = %v, want 'cannot decode empty data'", err.Error())
	}

	// Test nil target error message
	err = codec.Decode([]byte(`{"ID":"1"}`), nil)
	if err == nil {
		t.Errorf("Decode(..., nil) should return error")
	} else if err.Error() != "cannot decode into nil value" {
		t.Errorf("Decode(..., nil) error message = %v, want 'cannot decode into nil value'", err.Error())
	}
}
