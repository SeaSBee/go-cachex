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

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}

	// Create key builder
	keyBuilder := key.NewBuilder("myapp", "dev", "secret123")

	// Create JSON codec
	jsonCodec := codec.NewJSONCodec()

	// Create cache
	c, err := cache.New[User](
		cache.WithStore(redisStore),
		cache.WithCodec(jsonCodec),
		cache.WithKeyBuilder(keyBuilder),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// Example 1: Basic Get/Set
	fmt.Println("=== Basic Get/Set ===")

	user := User{ID: 123, Name: "John Doe", Email: "john@example.com"}
	userKey := keyBuilder.BuildUser("123")

	// Set user in cache
	err = c.Set(userKey, user, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
	}

	// Get user from cache
	cachedUser, found, err := c.Get(userKey)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
	} else if found {
		fmt.Printf("Found user: %+v\n", cachedUser)
	} else {
		fmt.Println("User not found in cache")
	}

	// Example 2: Read-Through Pattern
	fmt.Println("\n=== Read-Through Pattern ===")

	user2, err := c.ReadThrough(keyBuilder.BuildUser("456"), 10*time.Minute, func(ctx context.Context) (User, error) {
		// Simulate loading from database
		fmt.Println("Loading user 456 from database...")
		return User{ID: 456, Name: "Jane Smith", Email: "jane@example.com"}, nil
	})
	if err != nil {
		log.Printf("Failed to read through: %v", err)
	} else {
		fmt.Printf("User loaded: %+v\n", user2)
	}

	// Example 3: Multiple Get/Set
	fmt.Println("\n=== Multiple Get/Set ===")

	users := map[string]User{
		keyBuilder.BuildUser("789"): {ID: 789, Name: "Bob Wilson", Email: "bob@example.com"},
		keyBuilder.BuildUser("101"): {ID: 101, Name: "Alice Brown", Email: "alice@example.com"},
	}

	// Set multiple users
	err = c.MSet(users, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to set multiple users: %v", err)
	}

	// Get multiple users
	keys := []string{keyBuilder.BuildUser("789"), keyBuilder.BuildUser("101")}
	cachedUsers, err := c.MGet(keys...)
	if err != nil {
		log.Printf("Failed to get multiple users: %v", err)
	} else {
		for key, user := range cachedUsers {
			fmt.Printf("Key: %s, User: %+v\n", key, user)
		}
	}

	// Example 4: Exists and TTL
	fmt.Println("\n=== Exists and TTL ===")

	exists, err := c.Exists(userKey)
	if err != nil {
		log.Printf("Failed to check existence: %v", err)
	} else {
		fmt.Printf("User exists: %t\n", exists)
	}

	ttl, err := c.TTL(userKey)
	if err != nil {
		log.Printf("Failed to get TTL: %v", err)
	} else {
		fmt.Printf("User TTL: %v\n", ttl)
	}

	// Example 5: Increment
	fmt.Println("\n=== Increment ===")

	counterKey := "counter:visits"
	value, err := c.IncrBy(counterKey, 1, 1*time.Hour)
	if err != nil {
		log.Printf("Failed to increment: %v", err)
	} else {
		fmt.Printf("Counter value: %d\n", value)
	}

	// Example 6: Distributed Lock
	fmt.Println("\n=== Distributed Lock ===")

	unlock, acquired, err := c.TryLock("lock:resource", 30*time.Second)
	if err != nil {
		log.Printf("Failed to acquire lock: %v", err)
	} else if acquired {
		fmt.Println("Lock acquired successfully")
		defer unlock()
		// Do some work...
		time.Sleep(1 * time.Second)
		fmt.Println("Work completed, lock released")
	} else {
		fmt.Println("Failed to acquire lock")
	}

	fmt.Println("\n=== Example completed ===")
}
