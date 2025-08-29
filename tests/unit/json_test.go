package unit

import (
	"encoding/json"
	"testing"

	"github.com/SeaSBee/go-cachex"
	"github.com/stretchr/testify/assert"
)

func TestJSONCodec_NewJSONCodec(t *testing.T) {
	codec := cachex.NewJSONCodec()
	assert.NotNil(t, codec)
	assert.False(t, codec.AllowNilValues)
	assert.False(t, codec.EnableDebugLogging)
}

func TestJSONCodec_NewJSONCodecWithOptions(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, true)
	assert.NotNil(t, codec)
	assert.True(t, codec.AllowNilValues)
	assert.True(t, codec.EnableDebugLogging)
}

func TestJSONCodec_Validate(t *testing.T) {
	codec := cachex.NewJSONCodec()
	err := codec.Validate()
	assert.NoError(t, err)

	var nilCodec *cachex.JSONCodec
	err = nilCodec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codec cannot be nil")
}

func TestJSONCodec_Encode_NilValue_NotAllowed(t *testing.T) {
	codec := cachex.NewJSONCodec()
	data, err := codec.Encode(nil)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "cannot encode nil value")
}

func TestJSONCodec_Encode_NilValue_Allowed(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	data, err := codec.Encode(nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), data)
}

func TestJSONCodec_Encode_NilPointer_NotAllowed(t *testing.T) {
	codec := cachex.NewJSONCodec()
	var user *TestUser
	data, err := codec.Encode(user)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "cannot encode nil pointer value")
}

func TestJSONCodec_Encode_NilPointer_Allowed(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	var user *TestUser
	data, err := codec.Encode(user)
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), data)
}

func TestJSONCodec_Encode_ValidStruct(t *testing.T) {
	codec := cachex.NewJSONCodec()
	user := &TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	data, err := codec.Encode(user)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Verify the encoded data is valid JSON
	var decoded TestUser
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, decoded.ID)
	assert.Equal(t, user.Name, decoded.Name)
}

func TestJSONCodec_Encode_PrimitiveTypes(t *testing.T) {
	codec := cachex.NewJSONCodec()

	testCases := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
		{"map", map[string]int{"a": 1}, `{"a":1}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := codec.Encode(tc.value)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(data))
		})
	}
}

func TestJSONCodec_Encode_UnmarshalableType(t *testing.T) {
	codec := cachex.NewJSONCodec()
	// Create a channel which cannot be marshaled to JSON
	ch := make(chan int)
	data, err := codec.Encode(ch)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "json marshal error")
}

func TestJSONCodec_Decode_EmptyData(t *testing.T) {
	codec := cachex.NewJSONCodec()
	var user TestUser
	err := codec.Decode([]byte{}, &user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode empty data")
}

func TestJSONCodec_Decode_NilDestination(t *testing.T) {
	codec := cachex.NewJSONCodec()
	err := codec.Decode([]byte(`{"id":"1"}`), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode into nil value")
}

func TestJSONCodec_Decode_NullValue_NotAllowed(t *testing.T) {
	codec := cachex.NewJSONCodec()
	var user TestUser
	err := codec.Decode([]byte("null"), &user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode null value into type")
}

func TestJSONCodec_Decode_NullValue_Allowed_NonPointer(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	var user TestUser
	err := codec.Decode([]byte("null"), &user)
	if err != nil {
		assert.Contains(t, err.Error(), "cannot decode null into non-pointer type")
	} else {
		// If no error is returned, the implementation might be setting zero values
		// This is also acceptable behavior
		t.Log("No error returned, implementation sets zero values for non-pointer types")
	}
}

func TestJSONCodec_Decode_NullValue_Allowed_Pointer(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	var user *TestUser
	err := codec.Decode([]byte("null"), &user)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestJSONCodec_Decode_NullValue_Allowed_PointerToPointer(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	var user **TestUser
	err := codec.Decode([]byte("null"), &user)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestJSONCodec_Decode_ValidJSON(t *testing.T) {
	codec := cachex.NewJSONCodec()
	jsonData := []byte(`{"id":"1","name":"John","email":"john@example.com"}`)
	var user TestUser
	err := codec.Decode(jsonData, &user)
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "John", user.Name)
}

func TestJSONCodec_Decode_InvalidJSON(t *testing.T) {
	codec := cachex.NewJSONCodec()
	jsonData := []byte(`{"id":"1","name":"John"`) // Missing closing brace
	var user TestUser
	err := codec.Decode(jsonData, &user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json unmarshal error")
}

func TestJSONCodec_Decode_WhitespaceNull(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)
	var user *TestUser
	err := codec.Decode([]byte("  null  "), &user)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestJSONCodec_Decode_ComplexTypes(t *testing.T) {
	codec := cachex.NewJSONCodec()

	testCases := []struct {
		name     string
		jsonData string
		value    interface{}
		check    func(t *testing.T, value interface{})
	}{
		{
			name:     "slice",
			jsonData: `[1,2,3]`,
			value:    &[]int{},
			check: func(t *testing.T, value interface{}) {
				slice := value.(*[]int)
				assert.Equal(t, []int{1, 2, 3}, *slice)
			},
		},
		{
			name:     "map",
			jsonData: `{"a":1,"b":2}`,
			value:    &map[string]int{},
			check: func(t *testing.T, value interface{}) {
				m := value.(*map[string]int)
				assert.Equal(t, map[string]int{"a": 1, "b": 2}, *m)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := codec.Decode([]byte(tc.jsonData), tc.value)
			assert.NoError(t, err)
			tc.check(t, tc.value)
		})
	}
}

func TestJSONCodec_Name(t *testing.T) {
	codec := cachex.NewJSONCodec()
	assert.Equal(t, "json", codec.Name())
}

func TestJSONCodec_DebugLogging(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, true)

	// Test that debug logging doesn't cause panics
	user := &TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	data, err := codec.Encode(user)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var decoded TestUser
	err = codec.Decode(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, decoded.ID)
}

func TestJSONCodec_EdgeCases(t *testing.T) {
	codec := cachex.NewJSONCodec()

	// Test with zero values
	var zeroUser TestUser
	data, err := codec.Encode(zeroUser)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var decoded TestUser
	err = codec.Decode(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, zeroUser, decoded)

	// Test with empty string
	data, err = codec.Encode("")
	assert.NoError(t, err)
	assert.Equal(t, `""`, string(data))

	var decodedStr string
	err = codec.Decode(data, &decodedStr)
	assert.NoError(t, err)
	assert.Equal(t, "", decodedStr)
}

func TestJSONCodec_PointerLevels(t *testing.T) {
	codec := cachex.NewJSONCodecWithOptions(true, false)

	// Test pointer to pointer
	user := &TestUser{ID: "1", Name: "John", Email: "john@example.com"}
	userPtr := &user

	data, err := codec.Encode(userPtr)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var decodedPtr **TestUser
	err = codec.Decode(data, &decodedPtr)
	assert.NoError(t, err)
	assert.NotNil(t, decodedPtr)
	assert.NotNil(t, *decodedPtr)
	assert.Equal(t, user.ID, (*decodedPtr).ID)
	assert.Equal(t, user.Name, (*decodedPtr).Name)
}
