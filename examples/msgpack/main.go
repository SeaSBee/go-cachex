package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// User struct for demonstration
type User struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	CreatedAt time.Time              `json:"created_at"`
	Active    bool                   `json:"active"`
	Score     float64                `json:"score"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func main() {
	logx.Info("=== Go-CacheX Codec Demo ===")

	// Example 1: JSON Codec
	exampleJSONCodec()

	// Example 2: Codec Comparison
	exampleCodecComparison()

	// Example 3: Custom Codec
	exampleCustomCodec()

	logx.Info("=== Codec Demo Complete ===")
}

func exampleJSONCodec() {
	logx.Info("1. JSON Codec")
	logx.Info("=============")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create JSON codec
	jsonCodec := cachex.NewJSONCodec()

	// Create key builder
	keyBuilder := cachex.NewBuilder("myapp", "dev", "secret123")

	// Create cache with JSON codec
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithCodec(jsonCodec),
		cachex.WithKeyBuilder(keyBuilder),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     95.5,
		Metadata: map[string]interface{}{
			"last_login": time.Now().Add(-24 * time.Hour),
			"preferences": map[string]interface{}{
				"theme": "dark",
				"lang":  "en",
			},
		},
	}

	// Set user in cache using JSON
	userKey := keyBuilder.BuildUser("123")
	ctx := context.Background()
	setResult := <-c.Set(ctx, userKey, user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", userKey), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User stored using JSON serialization", logx.String("key", userKey))

	// Get user from cache
	getResult := <-c.Get(ctx, userKey)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", userKey), logx.ErrorField(getResult.Error))
		return
	}
	if getResult.Found {
		logx.Info("User retrieved using JSON deserialization",
			logx.String("id", getResult.Value.ID),
			logx.String("name", getResult.Value.Name),
			logx.String("email", getResult.Value.Email),
			logx.Bool("active", getResult.Value.Active),
			logx.Float64("score", getResult.Value.Score),
			logx.Int("metadata_keys", len(getResult.Value.Metadata)))
	} else {
		logx.Warn("User not found in cache", logx.String("key", userKey))
	}
}

func exampleCodecComparison() {
	logx.Info("2. Codec Comparison")
	logx.Info("===================")

	// Create both codecs
	jsonCodec := cachex.NewJSONCodec()

	// Test data
	user := User{
		ID:        "456",
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     88.7,
		Metadata: map[string]interface{}{
			"department": "Engineering",
			"level":      "Senior",
			"skills":     []string{"Go", "Redis", "JSON"},
		},
	}

	// Encode with JSON
	jsonData, err := jsonCodec.Encode(user)
	if err != nil {
		logx.Error("JSON encode failed", logx.ErrorField(err))
		return
	}

	logx.Info("JSON encoded data", logx.Int("size_bytes", len(jsonData)), logx.String("codec_name", jsonCodec.Name()))

	// Decode with JSON
	var decodedUser User
	err = jsonCodec.Decode(jsonData, &decodedUser)
	if err != nil {
		logx.Error("JSON decode failed", logx.ErrorField(err))
		return
	}

	logx.Info("JSON decoded successfully",
		logx.String("name", decodedUser.Name),
		logx.String("email", decodedUser.Email),
		logx.Any("metadata_skills", decodedUser.Metadata["skills"]))

	// Performance comparison
	logx.Info("Performance comparison:")

	// JSON encoding performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := jsonCodec.Encode(user)
		if err != nil {
			logx.Error("JSON encode error", logx.ErrorField(err), logx.Int("iteration", i))
			return
		}
	}
	jsonEncodeDuration := time.Since(start)
	jsonEncodeOpsPerSec := 1000.0 / jsonEncodeDuration.Seconds()
	logx.Info("JSON encode performance",
		logx.Int("iterations", 1000),
		logx.String("duration", jsonEncodeDuration.String()),
		logx.Float64("ops_per_sec", jsonEncodeOpsPerSec))

	// JSON decoding performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		err := jsonCodec.Decode(jsonData, &decodedUser)
		if err != nil {
			logx.Error("JSON decode error", logx.ErrorField(err), logx.Int("iteration", i))
			return
		}
	}
	jsonDecodeDuration := time.Since(start)
	jsonDecodeOpsPerSec := 1000.0 / jsonDecodeDuration.Seconds()
	logx.Info("JSON decode performance",
		logx.Int("iterations", 1000),
		logx.String("duration", jsonDecodeDuration.String()),
		logx.Float64("ops_per_sec", jsonDecodeOpsPerSec))
}

func exampleCustomCodec() {
	logx.Info("3. Custom Codec")
	logx.Info("===============")

	ctx := context.Background()

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create custom codec
	customCodec := &CustomCodec{}

	// Create cache with custom codec
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithCodec(customCodec),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        "789",
		Name:      "Custom User",
		Email:     "custom@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     92.3,
		Metadata: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}

	// Set user in cache using custom codec
	key := "user:custom:789"
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", key), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User stored using custom codec", logx.String("key", key))

	// Get user from cache
	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", key), logx.ErrorField(getResult.Error))
		return
	}
	if getResult.Found {
		logx.Info("User retrieved using custom codec",
			logx.String("name", getResult.Value.Name),
			logx.String("email", getResult.Value.Email),
			logx.Any("custom_metadata", getResult.Value.Metadata["custom_field"]))
	} else {
		logx.Warn("User not found in cache", logx.String("key", key))
	}
}

// CustomCodec implements a simple custom codec
type CustomCodec struct{}

func (c *CustomCodec) Encode(v any) ([]byte, error) {
	// Use JSON codec as base for custom codec
	jsonCodec := cachex.NewJSONCodec()
	data, err := jsonCodec.Encode(v)
	if err != nil {
		logx.Error("Custom codec encode failed", logx.ErrorField(err))
		return nil, err
	}

	// Add custom prefix to identify custom codec
	customData := append([]byte("CUSTOM:"), data...)
	logx.Debug("Custom codec encoded data", logx.Int("original_size", len(data)), logx.Int("custom_size", len(customData)))
	return customData, nil
}

func (c *CustomCodec) Decode(data []byte, v any) error {
	// Check for custom prefix
	if len(data) < 7 || string(data[:7]) != "CUSTOM:" {
		logx.Error("Invalid custom codec data", logx.Int("data_length", len(data)))
		return fmt.Errorf("invalid custom codec data")
	}

	// Remove custom prefix and use JSON codec
	jsonData := data[7:]
	jsonCodec := cachex.NewJSONCodec()
	err := jsonCodec.Decode(jsonData, v)
	if err != nil {
		logx.Error("Custom codec decode failed", logx.ErrorField(err))
		return err
	}

	logx.Debug("Custom codec decoded data successfully", logx.Int("data_size", len(data)))
	return nil
}

func (c *CustomCodec) Name() string {
	return "custom"
}
