package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// Test data structures
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	IsActive bool   `json:"is_active"`
}

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
}

// Helper functions
func createRedisStore() cachex.Store {
	store, err := cachex.NewRedisStore(&cachex.RedisConfig{
		Addr:                "localhost:6379",
		Password:            "",
		DB:                  0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Redis store: %v", err))
	}
	return store
}

func createMemoryStore() cachex.Store {
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Memory store: %v", err))
	}
	return store
}

func createRistrettoStore() cachex.Store {
	store, err := cachex.NewRistrettoStore(&cachex.RistrettoConfig{
		MaxItems:       100000,
		MaxMemoryBytes: 100 * 1024 * 1024, // 100MB
		DefaultTTL:     5 * time.Minute,
		NumCounters:    1000000,
		BufferItems:    64,
		CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
		EnableMetrics:  true,
		EnableStats:    true,
		BatchSize:      20,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Ristretto store: %v", err))
	}
	return store
}

func createLayeredStore() cachex.Store {
	l2Store := createRedisStore()

	store, err := cachex.NewLayeredStore(l2Store, &cachex.LayeredConfig{
		MemoryConfig: &cachex.MemoryConfig{
			MaxSize:         10000,
			MaxMemoryMB:     100,
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  cachex.EvictionPolicyLRU,
			EnableStats:     true,
		},
		WritePolicy:  cachex.WritePolicyThrough,
		ReadPolicy:   cachex.ReadPolicyThrough,
		SyncInterval: 5 * time.Minute,
		EnableStats:  true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Layered store: %v", err))
	}
	return store
}

func generateTestData(size int) []User {
	users := make([]User, size)
	for i := 0; i < size; i++ {
		users[i] = User{
			ID:       i + 1,
			Name:     fmt.Sprintf("User%d", i+1),
			Email:    fmt.Sprintf("user%d@example.com", i+1),
			Age:      20 + rand.Intn(50),
			IsActive: rand.Float32() > 0.5,
		}
	}
	return users
}

func generateProductData(size int) []Product {
	products := make([]Product, size)
	categories := []string{"Electronics", "Clothing", "Books", "Home", "Sports"}
	for i := 0; i < size; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        fmt.Sprintf("Product%d", i+1),
			Price:       10.0 + rand.Float64()*1000.0,
			Category:    categories[rand.Intn(len(categories))],
			Description: fmt.Sprintf("Description for product %d", i+1),
			InStock:     rand.Float32() > 0.3,
		}
	}
	return products
}

// Redis Store Benchmarks
func BenchmarkRedisSet(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkRedisGet(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", (i%len(users))+1)
			result := <-cache.Get(context.Background(), key)
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkRedisMSet(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			data := make(map[string]User)
			for j := 0; j < 10; j++ {
				user := users[(i+j)%len(users)]
				key := fmt.Sprintf("user:%d", user.ID)
				data[key] = user
			}
			result := <-cache.MSet(context.Background(), data, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("MSet failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkRedisMGet(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = fmt.Sprintf("user:%d", ((i+j)%len(users))+1)
			}
			result := <-cache.MGet(context.Background(), keys...)
			if result.Error != nil {
				b.Fatalf("MGet failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkRedisIncrBy(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[int64](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("counter:%d", i%10)
			result := <-cache.IncrBy(context.Background(), key, 1, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("IncrBy failed: %v", result.Error)
			}
			i++
		}
	})
}

// Memory Store Benchmarks
func BenchmarkMemorySet(b *testing.B) {
	store := createMemoryStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkMemoryGet(b *testing.B) {
	store := createMemoryStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", (i%len(users))+1)
			result := <-cache.Get(context.Background(), key)
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkMemoryMSet(b *testing.B) {
	store := createMemoryStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			data := make(map[string]User)
			for j := 0; j < 10; j++ {
				user := users[(i+j)%len(users)]
				key := fmt.Sprintf("user:%d", user.ID)
				data[key] = user
			}
			result := <-cache.MSet(context.Background(), data, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("MSet failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkMemoryMGet(b *testing.B) {
	store := createMemoryStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = fmt.Sprintf("user:%d", ((i+j)%len(users))+1)
			}
			result := <-cache.MGet(context.Background(), keys...)
			if result.Error != nil {
				b.Fatalf("MGet failed: %v", result.Error)
			}
			i++
		}
	})
}

// Ristretto Store Benchmarks
func BenchmarkRistrettoSet(b *testing.B) {
	store := createRistrettoStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkRistrettoGet(b *testing.B) {
	store := createRistrettoStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	// Allow time for async operations
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", (i%len(users))+1)
			result := <-cache.Get(context.Background(), key)
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			i++
		}
	})
}

// Layered Store Benchmarks
func BenchmarkLayeredSet(b *testing.B) {
	store := createLayeredStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkLayeredGet(b *testing.B) {
	store := createLayeredStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Pre-populate cache
	users := generateTestData(100)
	for _, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)
		result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Setup Set failed: %v", result.Error)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", (i%len(users))+1)
			result := <-cache.Get(context.Background(), key)
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			i++
		}
	})
}

// Tagging Benchmarks
func BenchmarkTaggingAddTags(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	tagManager, err := cachex.NewTagManager(store, cachex.DefaultTagConfig())
	if err != nil {
		b.Fatalf("Failed to create tag manager: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key:%d", i)
			tags := []string{fmt.Sprintf("tag:%d", i%10), fmt.Sprintf("category:%d", i%5)}
			err := tagManager.AddTags(context.Background(), key, tags...)
			if err != nil {
				b.Fatalf("AddTags failed: %v", err)
			}
			i++
		}
	})
}

func BenchmarkTaggingGetKeysByTag(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	tagManager, err := cachex.NewTagManager(store, cachex.DefaultTagConfig())
	if err != nil {
		b.Fatalf("Failed to create tag manager: %v", err)
	}

	// Pre-populate tags
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		tags := []string{fmt.Sprintf("tag:%d", i%10), fmt.Sprintf("category:%d", i%5)}
		err := tagManager.AddTags(context.Background(), key, tags...)
		if err != nil {
			b.Fatalf("Setup AddTags failed: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tag := fmt.Sprintf("tag:%d", i%10)
			_, err := tagManager.GetKeysByTag(context.Background(), tag)
			if err != nil {
				b.Fatalf("GetKeysByTag failed: %v", err)
			}
			i++
		}
	})
}

// MessagePack Codec Benchmarks
func BenchmarkMessagePackEncode(b *testing.B) {
	codec := cachex.NewMessagePackCodec()
	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			_, err := codec.Encode(user)
			if err != nil {
				b.Fatalf("Encode failed: %v", err)
			}
			i++
		}
	})
}

func BenchmarkMessagePackDecode(b *testing.B) {
	codec := cachex.NewMessagePackCodec()
	users := generateTestData(100)

	// Pre-encode data
	encodedData := make([][]byte, len(users))
	for i, user := range users {
		data, err := codec.Encode(user)
		if err != nil {
			b.Fatalf("Setup Encode failed: %v", err)
		}
		encodedData[i] = data
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			var user User
			err := codec.Decode(encodedData[i%len(encodedData)], &user)
			if err != nil {
				b.Fatalf("Decode failed: %v", err)
			}
			i++
		}
	})
}

// JSON Codec Benchmarks (for comparison)
func BenchmarkJSONEncode(b *testing.B) {
	codec := cachex.NewJSONCodec()
	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			_, err := codec.Encode(user)
			if err != nil {
				b.Fatalf("Encode failed: %v", err)
			}
			i++
		}
	})
}

func BenchmarkJSONDecode(b *testing.B) {
	codec := cachex.NewJSONCodec()
	users := generateTestData(100)

	// Pre-encode data
	encodedData := make([][]byte, len(users))
	for i, user := range users {
		data, err := codec.Encode(user)
		if err != nil {
			b.Fatalf("Setup Encode failed: %v", err)
		}
		encodedData[i] = data
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			var user User
			err := codec.Decode(encodedData[i%len(encodedData)], &user)
			if err != nil {
				b.Fatalf("Decode failed: %v", err)
			}
			i++
		}
	})
}

// Cache with different configurations
func BenchmarkCacheWithRedis(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkCacheWithMemory(b *testing.B) {
	store := createMemoryStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

func BenchmarkCacheWithLayered(b *testing.B) {
	store := createLayeredStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

// Complex data structure benchmarks
func BenchmarkComplexDataStructures(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[Product](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	products := generateProductData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			product := products[i%len(products)]
			key := fmt.Sprintf("product:%d", product.ID)
			result := <-cache.Set(context.Background(), key, product, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

// Large data benchmarks
func BenchmarkLargeData(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[map[string]interface{}](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Generate large data
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key:%d", i)] = fmt.Sprintf("value:%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("large:%d", i)
			result := <-cache.Set(context.Background(), key, largeData, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

// TTL benchmarks
func BenchmarkTTLOperations(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)

			// Set with different TTLs
			ttl := time.Duration(1+i%60) * time.Second
			result := <-cache.Set(context.Background(), key, user, ttl)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}

			// Get TTL
			ttlResult := <-cache.TTL(context.Background(), key)
			if ttlResult.Error != nil {
				b.Fatalf("TTL failed: %v", ttlResult.Error)
			}
			i++
		}
	})
}

// Concurrent operations benchmarks
func BenchmarkConcurrentOperations(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	cache, err := cachex.New[User](cachex.WithStore(store))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)

			// Mix of operations
			switch i % 4 {
			case 0:
				result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
				if result.Error != nil {
					b.Fatalf("Set failed: %v", result.Error)
				}
			case 1:
				result := <-cache.Get(context.Background(), key)
				if result.Error != nil && result.Error.Error() != "key not found" {
					b.Fatalf("Get failed: %v", result.Error)
				}
			case 2:
				result := <-cache.Del(context.Background(), key)
				if result.Error != nil {
					b.Fatalf("Del failed: %v", result.Error)
				}
			case 3:
				result := <-cache.IncrBy(context.Background(), key, 1, 5*time.Minute)
				if result.Error != nil {
					b.Fatalf("IncrBy failed: %v", result.Error)
				}
			}
			i++
		}
	})
}

// Builder benchmarks
func BenchmarkKeyBuilder(b *testing.B) {
	builder, err := cachex.NewBuilder("test-app", "test", "secret")
	if err != nil {
		b.Fatalf("Failed to create builder: %v", err)
	}
	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := builder.Build("user", fmt.Sprintf("%d", user.ID))
			if key == "" {
				b.Fatalf("Key generation failed")
			}
			i++
		}
	})
}

// Observability benchmarks
func BenchmarkObservabilityMetrics(b *testing.B) {
	store := createRedisStore()
	defer store.Close()

	observabilityConfig := &cachex.ObservabilityConfig{
		EnableMetrics: true,
		EnableTracing: false,
		EnableLogging: false,
	}

	cache, err := cachex.New[User](cachex.WithStore(store), cachex.WithObservability(*observabilityConfig))
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	users := generateTestData(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := users[i%len(users)]
			key := fmt.Sprintf("user:%d", user.ID)
			result := <-cache.Set(context.Background(), key, user, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
			i++
		}
	})
}

// Configuration benchmarks
func BenchmarkConfigCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			config := &cachex.CacheConfig{
				Redis: &cachex.RedisConfig{
					Addr:                "localhost:6379",
					Password:            "",
					DB:                  0,
					PoolSize:            10,
					MinIdleConns:        5,
					MaxRetries:          3,
					DialTimeout:         5 * time.Second,
					ReadTimeout:         3 * time.Second,
					WriteTimeout:        3 * time.Second,
					EnablePipelining:    true,
					EnableMetrics:       true,
					HealthCheckInterval: 30 * time.Second,
					HealthCheckTimeout:  5 * time.Second,
				},
				Memory: &cachex.MemoryConfig{
					MaxSize:         10000,
					MaxMemoryMB:     100,
					DefaultTTL:      5 * time.Minute,
					CleanupInterval: 1 * time.Minute,
					EvictionPolicy:  cachex.EvictionPolicyLRU,
					EnableStats:     true,
				},
				Ristretto: &cachex.RistrettoConfig{
					MaxItems:       100000,
					MaxMemoryBytes: 100 * 1024 * 1024,
					DefaultTTL:     5 * time.Minute,
					NumCounters:    1000000,
					BufferItems:    64,
					CostFunction:   cachex.DefaultRistrettoConfig().CostFunction,
					EnableMetrics:  true,
					EnableStats:    true,
					BatchSize:      20,
				},
				Observability: &cachex.ObservabilityConfig{
					EnableMetrics: true,
					EnableTracing: false,
					EnableLogging: false,
				},
			}

			_, err := cachex.NewFromConfig[User](config)
			if err != nil {
				b.Fatalf("NewFromConfig failed: %v", err)
			}
		}
	})
}
