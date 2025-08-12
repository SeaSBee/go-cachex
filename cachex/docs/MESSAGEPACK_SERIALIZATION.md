# MessagePack Serialization

Go-CacheX provides **MessagePack serialization** support as an alternative to JSON, offering better performance and smaller data sizes for cache operations.

## Overview

MessagePack is a binary serialization format that is:
- **Faster** than JSON for encoding/decoding
- **Smaller** than JSON in terms of data size
- **Type-preserving** for better data integrity
- **Language-agnostic** for cross-platform compatibility

## Features

- ✅ **Full MessagePack Support**: Complete implementation using `github.com/vmihailenco/msgpack/v5`
- ✅ **Pluggable Architecture**: Easy integration with existing cache operations
- ✅ **Type Safety**: Preserves Go types during serialization
- ✅ **Performance Optimized**: Faster encoding/decoding than JSON
- ✅ **Comprehensive Testing**: Extensive test coverage for all data types
- ✅ **Error Handling**: Robust error handling for serialization failures

## Installation

MessagePack support is included by default in Go-CacheX. The dependency is automatically managed:

```bash
go get github.com/SeaSBee/go-cachex
```

## Usage

### Basic Usage

```go
import (
    "time"
    "github.com/SeaSBee/go-cachex/cachex/pkg/cache"
    "github.com/SeaSBee/go-cachex/cachex/pkg/codec"
    "github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// Create MessagePack codec
msgpackCodec := codec.NewMessagePackCodec()

// Create Redis store
redisStore, err := redisstore.New(&redisstore.Config{
    Addr: "localhost:6379",
    DB:   0,
})
if err != nil {
    log.Fatal(err)
}

// Create cache with MessagePack
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCodec(msgpackCodec),
    cache.WithDefaultTTL(10*time.Minute),
)
if err != nil {
    log.Fatal(err)
}
defer c.Close()

// Use cache normally - all operations use MessagePack serialization
user := User{ID: 123, Name: "John Doe"}
err = c.Set("user:123", user, 5*time.Minute)
if err != nil {
    log.Printf("Set failed: %v", err)
}

cachedUser, found, err := c.Get("user:123")
if err != nil {
    log.Printf("Get failed: %v", err)
} else if found {
    fmt.Printf("Retrieved: %+v\n", cachedUser)
}
```

### Struct Tags

MessagePack supports struct tags for field customization:

```go
type User struct {
    ID        int       `json:"id" msgpack:"id"`
    Name      string    `json:"name" msgpack:"name"`
    Email     string    `json:"email" msgpack:"email"`
    CreatedAt time.Time `json:"created_at" msgpack:"created_at"`
    Active    bool      `json:"active" msgpack:"active"`
    Score     float64   `json:"score" msgpack:"score"`
    Metadata  map[string]interface{} `json:"metadata" msgpack:"metadata"`
}
```

### Advanced Configuration

```go
// Create MessagePack codec with custom options
msgpackCodec := codec.NewMessagePackCodecWithOptions(true) // Enable JSON tag usage

// Configure codec behavior
msgpackCodec.UseJSONTag(true)  // Use JSON tags for field names
msgpackCodec.UseJSONTag(false) // Use struct field names (default)
```

## Performance Comparison

### Data Size Comparison

MessagePack typically provides **20-40% size reduction** compared to JSON:

```go
// Test data
user := User{
    ID:        123,
    Name:      "John Doe",
    Email:     "john@example.com",
    CreatedAt: time.Now(),
    Active:    true,
    Score:     95.5,
}

// JSON encoding
jsonCodec := codec.NewJSONCodec()
jsonData, _ := jsonCodec.Encode(user)
fmt.Printf("JSON size: %d bytes\n", len(jsonData))

// MessagePack encoding
msgpackCodec := codec.NewMessagePackCodec()
msgpackData, _ := msgpackCodec.Encode(user)
fmt.Printf("MessagePack size: %d bytes\n", len(msgpackData))
fmt.Printf("Size reduction: %.2f%%\n", 
    float64(len(jsonData)-len(msgpackData))/float64(len(jsonData))*100)
```

**Example Output:**
```
JSON size: 143 bytes
MessagePack size: 99 bytes
Size reduction: 30.77%
```

### Performance Benchmarks

MessagePack provides faster encoding/decoding:

```go
// Performance test with 100 users
users := make([]User, 100)
// ... populate users ...

// MessagePack performance
start := time.Now()
msgpackData, _ := msgpackCodec.Encode(users)
msgpackEncodeTime := time.Since(start)

start = time.Now()
var msgpackUsers []User
msgpackCodec.Decode(msgpackData, &msgpackUsers)
msgpackDecodeTime := time.Since(start)

// JSON performance
start = time.Now()
jsonData, _ := jsonCodec.Encode(users)
jsonEncodeTime := time.Since(start)

start = time.Now()
var jsonUsers []User
jsonCodec.Decode(jsonData, &jsonUsers)
jsonDecodeTime := time.Since(start)

fmt.Printf("MessagePack - Encode: %v, Decode: %v\n", msgpackEncodeTime, msgpackDecodeTime)
fmt.Printf("JSON - Encode: %v, Decode: %v\n", jsonEncodeTime, jsonDecodeTime)
```

## Supported Data Types

### Basic Types
- ✅ **Strings**: `string`
- ✅ **Integers**: `int`, `int8`, `int16`, `int32`, `int64`
- ✅ **Unsigned Integers**: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- ✅ **Floats**: `float32`, `float64`
- ✅ **Booleans**: `bool`
- ✅ **Nil Values**: `nil`

### Complex Types
- ✅ **Structs**: All struct types with proper field tags
- ✅ **Slices**: `[]T` for any type T
- ✅ **Maps**: `map[K]V` for any types K, V
- ✅ **Pointers**: `*T` for any type T
- ✅ **Interfaces**: `interface{}` and custom interfaces
- ✅ **Time**: `time.Time` with proper serialization

### Example with All Types

```go
type ComplexData struct {
    StringField    string                 `msgpack:"string_field"`
    IntField       int                    `msgpack:"int_field"`
    FloatField     float64                `msgpack:"float_field"`
    BoolField      bool                   `msgpack:"bool_field"`
    TimeField      time.Time              `msgpack:"time_field"`
    SliceField     []string               `msgpack:"slice_field"`
    MapField       map[string]interface{} `msgpack:"map_field"`
    PointerField   *string                `msgpack:"pointer_field"`
    InterfaceField interface{}            `msgpack:"interface_field"`
}

data := ComplexData{
    StringField:    "hello",
    IntField:       42,
    FloatField:     3.14,
    BoolField:      true,
    TimeField:      time.Now(),
    SliceField:     []string{"a", "b", "c"},
    MapField:       map[string]interface{}{"key": "value"},
    PointerField:   &pointerValue,
    InterfaceField: "any value",
}

// Serialize with MessagePack
msgpackCodec := codec.NewMessagePackCodec()
data, err := msgpackCodec.Encode(data)
if err != nil {
    log.Printf("Encode failed: %v", err)
}
```

## Integration with Cache Patterns

### Read-Through Pattern

```go
user, err := c.ReadThrough("user:123", 10*time.Minute, func(ctx context.Context) (User, error) {
    // Load from database - result will be serialized with MessagePack
    return loadUserFromDB(ctx, "123")
})
```

### Write-Through Pattern

```go
user := User{ID: 123, Name: "John Doe"}
err := c.WriteThrough("user:123", user, 10*time.Minute, func(ctx context.Context) error {
    // Write to database - user is serialized with MessagePack for cache
    return saveUserToDB(ctx, user)
})
```

### Write-Behind Pattern

```go
user := User{ID: 123, Name: "John Doe"}
err := c.WriteBehind("user:123", user, 10*time.Minute, func(ctx context.Context) error {
    // Async write to database - user is immediately available in cache with MessagePack
    return asyncSaveUserToDB(ctx, user)
})
```

### Batch Operations

```go
// Batch store with MessagePack
users := map[string]User{
    "user:1": {ID: 1, Name: "Alice"},
    "user:2": {ID: 2, Name: "Bob"},
    "user:3": {ID: 3, Name: "Charlie"},
}

err := c.MSet(users, 10*time.Minute)
if err != nil {
    log.Printf("Batch store failed: %v", err)
}

// Batch retrieve with MessagePack
keys := []string{"user:1", "user:2", "user:3"}
retrievedUsers, err := c.MGet(keys...)
if err != nil {
    log.Printf("Batch retrieve failed: %v", err)
}
```

## Error Handling

### Encoding Errors

```go
msgpackCodec := codec.NewMessagePackCodec()

// Nil value error
data, err := msgpackCodec.Encode(nil)
if err != nil {
    // Error: "cannot encode nil value"
    log.Printf("Encode error: %v", err)
}
```

### Decoding Errors

```go
// Empty data error
var result User
err := msgpackCodec.Decode([]byte{}, &result)
if err != nil {
    // Error: "cannot decode empty data"
    log.Printf("Decode error: %v", err)
}

// Invalid MessagePack data error
err = msgpackCodec.Decode([]byte{0xFF, 0xFF, 0xFF}, &result)
if err != nil {
    // Error: "msgpack unmarshal error: ..."
    log.Printf("Decode error: %v", err)
}
```

## Migration from JSON

### Before (JSON)

```go
// Old JSON-based cache
jsonCodec := codec.NewJSONCodec()
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCodec(jsonCodec),
)
```

### After (MessagePack)

```go
// New MessagePack-based cache
msgpackCodec := codec.NewMessagePackCodec()
c, err := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCodec(msgpackCodec),
)
```

### Gradual Migration

You can migrate gradually by using different codecs for different cache instances:

```go
// Keep JSON for existing data
jsonCache, _ := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCodec(codec.NewJSONCodec()),
)

// Use MessagePack for new data
msgpackCache, _ := cache.New[User](
    cache.WithStore(redisStore),
    cache.WithCodec(codec.NewMessagePackCodec()),
)
```

## Best Practices

### 1. Struct Tags

Always use MessagePack tags for consistent serialization:

```go
type User struct {
    ID   int    `msgpack:"id"`   // Good
    Name string `msgpack:"name"` // Good
    // Bad: no tags
}
```

### 2. Type Consistency

Be consistent with types across your application:

```go
// Good: Consistent types
type Config struct {
    Port     int    `msgpack:"port"`
    Host     string `msgpack:"host"`
    Timeout  int    `msgpack:"timeout"`
}

// Avoid: Mixed types for similar data
type Config struct {
    Port     string `msgpack:"port"`     // Bad: port as string
    Host     string `msgpack:"host"`
    Timeout  int    `msgpack:"timeout"`
}
```

### 3. Performance Optimization

For high-performance scenarios:

```go
// Reuse codec instances
msgpackCodec := codec.NewMessagePackCodec()

// Create multiple caches with the same codec
cache1, _ := cache.New[User](cache.WithCodec(msgpackCodec))
cache2, _ := cache.New[Product](cache.WithCodec(msgpackCodec))
cache3, _ := cache.New[Order](cache.WithCodec(msgpackCodec))
```

### 4. Error Handling

Always handle serialization errors:

```go
user := User{ID: 123, Name: "John"}
data, err := msgpackCodec.Encode(user)
if err != nil {
    log.Printf("Serialization failed: %v", err)
    // Handle error appropriately
    return
}

var decodedUser User
err = msgpackCodec.Decode(data, &decodedUser)
if err != nil {
    log.Printf("Deserialization failed: %v", err)
    // Handle error appropriately
    return
}
```

## Troubleshooting

### Common Issues

1. **Type Mismatch Errors**
   ```go
   // Problem: MessagePack preserves exact types
   var result int
   err := msgpackCodec.Decode(data, &result)
   // May fail if data was encoded as int8/int16/int32/int64
   
   // Solution: Use interface{} for flexible decoding
   var result interface{}
   err := msgpackCodec.Decode(data, &result)
   ```

2. **Nil Pointer Handling**
   ```go
   // Problem: MessagePack handles nil pointers differently
   var ptr *string = nil
   data, _ := msgpackCodec.Encode(ptr)
   
   var result *string
   msgpackCodec.Decode(data, &result) // result will be nil
   ```

3. **Map Type Differences**
   ```go
   // Problem: MessagePack may decode maps as map[string]interface{}
   original := map[string]string{"key": "value"}
   data, _ := msgpackCodec.Encode(original)
   
   var result map[string]string
   err := msgpackCodec.Decode(data, &result) // May fail
   
   // Solution: Use map[string]interface{} for decoding
   var result map[string]interface{}
   err := msgpackCodec.Decode(data, &result)
   ```

### Debug Information

```go
// Get codec information
msgpackCodec := codec.NewMessagePackCodec()
fmt.Printf("Codec name: %s\n", msgpackCodec.Name())
fmt.Printf("JSON tags enabled: %t\n", msgpackCodec.IsJSONTagEnabled())

// Test serialization
testData := "hello world"
data, err := msgpackCodec.Encode(testData)
if err != nil {
    fmt.Printf("Encode error: %v\n", err)
} else {
    fmt.Printf("Encoded size: %d bytes\n", len(data))
}
```

## Examples

See the [MessagePack examples](../examples/msgpack/main.go) for complete working examples demonstrating:

- Basic MessagePack usage
- Performance comparison with JSON
- Complex data structures
- Cache patterns with MessagePack
- Error handling scenarios

## Performance Considerations

### Memory Usage

- MessagePack typically uses **20-40% less memory** than JSON
- Binary format reduces memory allocation overhead
- Type preservation reduces conversion overhead

### CPU Usage

- **Faster encoding/decoding** than JSON
- Binary format reduces parsing overhead
- Optimized for high-throughput scenarios

### Network Usage

- **Smaller payload sizes** reduce network bandwidth
- **Faster transmission** due to reduced data size
- **Lower latency** for cache operations

### Storage Usage

- **Reduced storage requirements** in Redis
- **Better compression ratios** if using Redis compression
- **Lower memory usage** in Redis instances
