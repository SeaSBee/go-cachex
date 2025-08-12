package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/codec"
	"github.com/SeaSBee/go-cachex/cachex/pkg/key"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// User struct for testing MessagePack serialization
type User struct {
	ID        int                    `json:"id" msgpack:"id"`
	Name      string                 `json:"name" msgpack:"name"`
	Email     string                 `json:"email" msgpack:"email"`
	CreatedAt time.Time              `json:"created_at" msgpack:"created_at"`
	Active    bool                   `json:"active" msgpack:"active"`
	Score     float64                `json:"score" msgpack:"score"`
	Metadata  map[string]interface{} `json:"metadata" msgpack:"metadata"`
}

// Product struct for testing MessagePack serialization
type Product struct {
	ID          int      `json:"id" msgpack:"id"`
	Name        string   `json:"name" msgpack:"name"`
	Price       float64  `json:"price" msgpack:"price"`
	Category    string   `json:"category" msgpack:"category"`
	InStock     bool     `json:"in_stock" msgpack:"in_stock"`
	Tags        []string `json:"tags" msgpack:"tags"`
	Description string   `json:"description" msgpack:"description"`
}

func main() {
	// Example 1: Basic MessagePack Usage
	fmt.Println("=== Example 1: Basic MessagePack Usage ===")
	exampleBasicMessagePack()

	// Example 2: MessagePack vs JSON Comparison
	fmt.Println("\n=== Example 2: MessagePack vs JSON Comparison ===")
	exampleMessagePackVsJSON()

	// Example 3: Complex Data Structures
	fmt.Println("\n=== Example 3: Complex Data Structures ===")
	exampleComplexDataStructures()

	// Example 4: Performance Comparison
	fmt.Println("\n=== Example 4: Performance Comparison ===")
	examplePerformanceComparison()

	// Example 5: MessagePack with Cache Patterns
	fmt.Println("\n=== Example 5: MessagePack with Cache Patterns ===")
	exampleMessagePackWithCachePatterns()
}

func exampleBasicMessagePack() {
	// Create MessagePack codec
	msgpackCodec := codec.NewMessagePackCodec()

	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
	}

	// Create key builder
	keyBuilder := key.NewBuilder("myapp", "dev", "secret123")

	// Create cache with MessagePack codec
	c, err := cache.New[User](
		cache.WithStore(redisStore),
		cache.WithCodec(msgpackCodec),
		cache.WithKeyBuilder(keyBuilder),
		cache.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        123,
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

	// Set user in cache using MessagePack
	userKey := keyBuilder.BuildUser("123")
	err = c.Set(userKey, user, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
		return
	}
	fmt.Println("✓ User stored using MessagePack serialization")

	// Get user from cache
	cachedUser, found, err := c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	if found {
		fmt.Printf("✓ User retrieved using MessagePack deserialization: %+v\n", cachedUser)
		fmt.Printf("  - ID: %d\n", cachedUser.ID)
		fmt.Printf("  - Name: %s\n", cachedUser.Name)
		fmt.Printf("  - Email: %s\n", cachedUser.Email)
		fmt.Printf("  - Active: %t\n", cachedUser.Active)
		fmt.Printf("  - Score: %.2f\n", cachedUser.Score)
		fmt.Printf("  - Metadata keys: %d\n", len(cachedUser.Metadata))
	} else {
		fmt.Println("User not found in cache")
	}
}

func exampleMessagePackVsJSON() {
	// Create both codecs
	msgpackCodec := codec.NewMessagePackCodec()
	jsonCodec := codec.NewJSONCodec()

	// Test data
	user := User{
		ID:        456,
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     88.7,
		Metadata: map[string]interface{}{
			"department": "Engineering",
			"level":      "Senior",
			"skills":     []string{"Go", "Redis", "MessagePack"},
		},
	}

	// Encode with MessagePack
	msgpackData, err := msgpackCodec.Encode(user)
	if err != nil {
		log.Printf("MessagePack encode failed: %v", err)
		return
	}

	// Encode with JSON
	jsonData, err := jsonCodec.Encode(user)
	if err != nil {
		log.Printf("JSON encode failed: %v", err)
		return
	}

	// Compare sizes
	fmt.Printf("MessagePack size: %d bytes\n", len(msgpackData))
	fmt.Printf("JSON size: %d bytes\n", len(jsonData))
	fmt.Printf("Size reduction: %.2f%%\n", float64(len(jsonData)-len(msgpackData))/float64(len(jsonData))*100)

	// Decode and verify both work correctly
	var msgpackUser User
	err = msgpackCodec.Decode(msgpackData, &msgpackUser)
	if err != nil {
		log.Printf("MessagePack decode failed: %v", err)
		return
	}

	var jsonUser User
	err = jsonCodec.Decode(jsonData, &jsonUser)
	if err != nil {
		log.Printf("JSON decode failed: %v", err)
		return
	}

	// Verify data integrity
	if msgpackUser.ID == jsonUser.ID && msgpackUser.Name == jsonUser.Name {
		fmt.Println("✓ Both MessagePack and JSON maintain data integrity")
	} else {
		fmt.Println("✗ Data integrity check failed")
	}
}

func exampleComplexDataStructures() {
	// Create MessagePack codec
	msgpackCodec := codec.NewMessagePackCodec()

	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
	}

	// Create key builder
	keyBuilder := key.NewBuilder("myapp", "dev", "secret123")

	// Create cache for products
	c, err := cache.New[Product](
		cache.WithStore(redisStore),
		cache.WithCodec(msgpackCodec),
		cache.WithKeyBuilder(keyBuilder),
		cache.WithDefaultTTL(15*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Create complex product data
	product := Product{
		ID:          789,
		Name:        "Go-CacheX Library",
		Price:       99.99,
		Category:    "Software Libraries",
		InStock:     true,
		Tags:        []string{"go", "cache", "redis", "messagepack", "performance"},
		Description: "A production-grade, highly concurrent, thread-safe cache library built on top of go-redis",
	}

	// Store product
	productKey := keyBuilder.Build("product", "789")
	err = c.Set(productKey, product, 15*time.Minute)
	if err != nil {
		log.Printf("Failed to set product: %v", err)
		return
	}
	fmt.Println("✓ Complex product data stored using MessagePack")

	// Retrieve product
	cachedProduct, found, err := c.Get(productKey)
	if err != nil {
		log.Printf("Failed to get product: %v", err)
		return
	}
	if found {
		fmt.Printf("✓ Complex product data retrieved: %+v\n", cachedProduct)
		fmt.Printf("  - Name: %s\n", cachedProduct.Name)
		fmt.Printf("  - Price: $%.2f\n", cachedProduct.Price)
		fmt.Printf("  - Category: %s\n", cachedProduct.Category)
		fmt.Printf("  - In Stock: %t\n", cachedProduct.InStock)
		fmt.Printf("  - Tags: %v\n", cachedProduct.Tags)
		fmt.Printf("  - Description: %s\n", cachedProduct.Description)
	}

	// Test batch operations with complex data
	products := map[string]Product{
		keyBuilder.Build("product", "100"): {
			ID:       100,
			Name:     "Product A",
			Price:    29.99,
			Category: "Electronics",
			InStock:  true,
			Tags:     []string{"electronics", "gadget"},
		},
		keyBuilder.Build("product", "101"): {
			ID:       101,
			Name:     "Product B",
			Price:    49.99,
			Category: "Books",
			InStock:  false,
			Tags:     []string{"books", "education"},
		},
	}

	// Batch store
	err = c.MSet(products, 15*time.Minute)
	if err != nil {
		log.Printf("Failed to batch store products: %v", err)
		return
	}
	fmt.Println("✓ Batch products stored using MessagePack")

	// Batch retrieve
	keys := []string{
		keyBuilder.Build("product", "100"),
		keyBuilder.Build("product", "101"),
	}
	retrievedProducts, err := c.MGet(keys...)
	if err != nil {
		log.Printf("Failed to batch retrieve products: %v", err)
		return
	}
	fmt.Printf("✓ Batch products retrieved: %d products\n", len(retrievedProducts))
}

func examplePerformanceComparison() {
	// Create both codecs
	msgpackCodec := codec.NewMessagePackCodec()
	jsonCodec := codec.NewJSONCodec()

	// Create test data
	users := make([]User, 100)
	for i := 0; i < 100; i++ {
		users[i] = User{
			ID:        i + 1,
			Name:      fmt.Sprintf("User %d", i+1),
			Email:     fmt.Sprintf("user%d@example.com", i+1),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			Active:    i%2 == 0,
			Score:     float64(i) * 0.5,
			Metadata: map[string]interface{}{
				"index": i,
				"data":  fmt.Sprintf("metadata_%d", i),
			},
		}
	}

	// Test MessagePack performance
	start := time.Now()
	msgpackData, err := msgpackCodec.Encode(users)
	msgpackEncodeTime := time.Since(start)
	if err != nil {
		log.Printf("MessagePack encode failed: %v", err)
		return
	}

	start = time.Now()
	var msgpackUsers []User
	err = msgpackCodec.Decode(msgpackData, &msgpackUsers)
	msgpackDecodeTime := time.Since(start)
	if err != nil {
		log.Printf("MessagePack decode failed: %v", err)
		return
	}

	// Test JSON performance
	start = time.Now()
	jsonData, err := jsonCodec.Encode(users)
	jsonEncodeTime := time.Since(start)
	if err != nil {
		log.Printf("JSON encode failed: %v", err)
		return
	}

	start = time.Now()
	var jsonUsers []User
	err = jsonCodec.Decode(jsonData, &jsonUsers)
	jsonDecodeTime := time.Since(start)
	if err != nil {
		log.Printf("JSON decode failed: %v", err)
		return
	}

	// Report performance metrics
	fmt.Printf("Performance Comparison (100 users):\n")
	fmt.Printf("MessagePack:\n")
	fmt.Printf("  - Encode time: %v\n", msgpackEncodeTime)
	fmt.Printf("  - Decode time: %v\n", msgpackDecodeTime)
	fmt.Printf("  - Data size: %d bytes\n", len(msgpackData))
	fmt.Printf("JSON:\n")
	fmt.Printf("  - Encode time: %v\n", jsonEncodeTime)
	fmt.Printf("  - Decode time: %v\n", jsonDecodeTime)
	fmt.Printf("  - Data size: %d bytes\n", len(jsonData))
	fmt.Printf("Size reduction: %.2f%%\n", float64(len(jsonData)-len(msgpackData))/float64(len(jsonData))*100)
}

func exampleMessagePackWithCachePatterns() {
	// Create MessagePack codec
	msgpackCodec := codec.NewMessagePackCodec()

	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
	}

	// Create key builder
	keyBuilder := key.NewBuilder("myapp", "dev", "secret123")

	// Create cache with MessagePack
	c, err := cache.New[User](
		cache.WithStore(redisStore),
		cache.WithCodec(msgpackCodec),
		cache.WithKeyBuilder(keyBuilder),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Example 1: Read-Through Pattern with MessagePack
	fmt.Println("Testing Read-Through Pattern with MessagePack:")
	user, err := c.ReadThrough(keyBuilder.BuildUser("999"), 10*time.Minute, func(ctx context.Context) (User, error) {
		// Simulate loading from database
		fmt.Println("  Loading user 999 from database...")
		return User{
			ID:        999,
			Name:      "Read-Through User",
			Email:     "readthrough@example.com",
			CreatedAt: time.Now(),
			Active:    true,
			Score:     75.0,
		}, nil
	})
	if err != nil {
		log.Printf("Read-Through failed: %v", err)
	} else {
		fmt.Printf("  ✓ Read-Through completed: %s\n", user.Name)
	}

	// Example 2: Write-Through Pattern with MessagePack
	fmt.Println("\nTesting Write-Through Pattern with MessagePack:")
	writeUser := User{
		ID:        888,
		Name:      "Write-Through User",
		Email:     "writethrough@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     85.0,
	}
	err = c.WriteThrough(keyBuilder.BuildUser("888"), writeUser, 10*time.Minute, func(ctx context.Context) error {
		// Simulate writing to database
		fmt.Println("  Writing user 888 to database...")
		return nil
	})
	if err != nil {
		log.Printf("Write-Through failed: %v", err)
	} else {
		fmt.Println("  ✓ Write-Through completed")
	}

	// Example 3: Write-Behind Pattern with MessagePack
	fmt.Println("\nTesting Write-Behind Pattern with MessagePack:")
	behindUser := User{
		ID:        777,
		Name:      "Write-Behind User",
		Email:     "writebehind@example.com",
		CreatedAt: time.Now(),
		Active:    true,
		Score:     92.0,
	}
	err = c.WriteBehind(keyBuilder.BuildUser("777"), behindUser, 10*time.Minute, func(ctx context.Context) error {
		// Simulate async database write
		fmt.Println("  Async writing user 777 to database...")
		time.Sleep(100 * time.Millisecond) // Simulate database write time
		return nil
	})
	if err != nil {
		log.Printf("Write-Behind failed: %v", err)
	} else {
		fmt.Println("  ✓ Write-Behind completed")
	}

	fmt.Println("\n✓ All cache patterns work correctly with MessagePack serialization")
}
