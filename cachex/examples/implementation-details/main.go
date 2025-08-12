package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Product struct for demonstration
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

func main() {
	fmt.Println("=== Go-CacheX Implementation Details Demo ===")

	// Create Redis store
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		log.Printf("Failed to create Redis store: %v", err)
		return
	}

	// Create cache with all features enabled
	c, err := cache.New[User](
		cache.WithStore(redisStore),
		cache.WithDefaultTTL(5*time.Minute),
		cache.WithDeadLetterQueue(),
		cache.WithBloomFilter(),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Demo 1: Tagging with Inverted Index
	demoTagging(c)

	// Demo 2: Pub/Sub Invalidation
	demoPubSubInvalidation(c)

	// Demo 3: Refresh-Ahead Background Scheduler
	demoRefreshAhead(c)

	fmt.Println("\n=== Implementation Details Demo Complete ===")
}

func demoTagging(c cache.Cache[User]) {
	fmt.Println("\n1. Tagging with Inverted Index")
	fmt.Println("===============================")

	// Create sample users
	users := []User{
		{ID: "user:1", Name: "John Doe", Email: "john@example.com", CreatedAt: time.Now()},
		{ID: "user:2", Name: "Jane Smith", Email: "jane@example.com", CreatedAt: time.Now()},
		{ID: "user:3", Name: "Bob Johnson", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "user:4", Name: "Alice Brown", Email: "alice@example.com", CreatedAt: time.Now()},
	}

	// Store users with tags
	fmt.Println("✓ Storing users with tags...")
	for i, user := range users {
		key := fmt.Sprintf("user:%d", i+1)

		// Store user
		if err := c.Set(key, user, 10*time.Minute); err != nil {
			log.Printf("Failed to set user %s: %v", key, err)
			continue
		}

		// Add tags based on user properties
		var tags []string
		if i < 2 {
			tags = append(tags, "premium", "active")
		} else {
			tags = append(tags, "standard", "active")
		}

		if i%2 == 0 {
			tags = append(tags, "verified")
		}

		if err := c.AddTags(key, tags...); err != nil {
			log.Printf("Failed to add tags to %s: %v", key, err)
			continue
		}

		fmt.Printf("  - %s: tags=%v\n", key, tags)
	}

	// Demonstrate tag-based invalidation
	fmt.Println("\n✓ Demonstrating tag-based invalidation...")

	// Invalidate all premium users
	fmt.Println("  - Invalidating all premium users...")
	if err := c.InvalidateByTag("premium"); err != nil {
		log.Printf("Failed to invalidate premium users: %v", err)
	} else {
		fmt.Println("    ✓ Premium users invalidated")
	}

	// Invalidate all verified users
	fmt.Println("  - Invalidating all verified users...")
	if err := c.InvalidateByTag("verified"); err != nil {
		log.Printf("Failed to invalidate verified users: %v", err)
	} else {
		fmt.Println("    ✓ Verified users invalidated")
	}

	// Check if users still exist
	fmt.Println("\n✓ Checking user existence after invalidation...")
	for i := range users {
		key := fmt.Sprintf("user:%d", i+1)
		if _, found, err := c.Get(key); err != nil {
			log.Printf("Failed to get %s: %v", key, err)
		} else if found {
			fmt.Printf("  - %s: still exists\n", key)
		} else {
			fmt.Printf("  - %s: not found (invalidated)\n", key)
		}
	}
}

func demoPubSubInvalidation(c cache.Cache[User]) {
	fmt.Println("\n2. Pub/Sub Invalidation")
	fmt.Println("=======================")

	// Create a second cache instance to simulate multiple services
	redisStore2, err := redisstore.New(&redisstore.Config{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		log.Printf("Failed to create second Redis store: %v", err)
		return
	}

	c2, err := cache.New[User](
		cache.WithStore(redisStore2),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create second cache: %v", err)
		return
	}
	defer c2.Close()

	// Subscribe to invalidation messages in the second cache
	fmt.Println("✓ Setting up pub/sub subscription...")
	closeSub, err := c2.SubscribeInvalidations(func(keys ...string) {
		fmt.Printf("🔄 Received invalidation for keys: %v\n", keys)

		// Process invalidated keys
		for _, key := range keys {
			fmt.Printf("    - Removing %s from local cache\n", key)
			// In a real application, you would remove from local cache here
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to invalidations: %v", err)
		return
	}
	defer closeSub()

	// Store some data in the first cache
	fmt.Println("✓ Storing data in first cache...")
	user := User{ID: "pubsub:1", Name: "PubSub User", Email: "pubsub@example.com", CreatedAt: time.Now()}
	if err := c.Set("pubsub:user:1", user, 5*time.Minute); err != nil {
		log.Printf("Failed to set user: %v", err)
		return
	}

	// Add tags
	if err := c.AddTags("pubsub:user:1", "pubsub", "demo"); err != nil {
		log.Printf("Failed to add tags: %v", err)
		return
	}

	// Publish invalidation from the first cache
	fmt.Println("✓ Publishing invalidation message...")
	if err := c.PublishInvalidation("pubsub:user:1"); err != nil {
		log.Printf("Failed to publish invalidation: %v", err)
		return
	}

	// Wait a bit for the message to be processed
	time.Sleep(2 * time.Second)
	fmt.Println("✓ Pub/sub invalidation completed")
}

func demoRefreshAhead(c cache.Cache[User]) {
	fmt.Println("\n3. Refresh-Ahead Background Scheduler")
	fmt.Println("=====================================")

	// Create a user that will be refreshed
	user := User{ID: "refresh:1", Name: "Refresh User", Email: "refresh@example.com", CreatedAt: time.Now()}
	key := "refresh:user:1"

	// Store the user with a short TTL
	fmt.Println("✓ Storing user with short TTL...")
	if err := c.Set(key, user, 30*time.Second); err != nil {
		log.Printf("Failed to set user: %v", err)
		return
	}

	// Set up refresh-ahead with a loader function
	fmt.Println("✓ Setting up refresh-ahead...")
	refreshBefore := 10 * time.Second // Refresh 10 seconds before expiry

	loader := func(ctx context.Context) (User, error) {
		// Simulate loading from database
		fmt.Println("    🔄 Loading fresh user data from database...")
		time.Sleep(100 * time.Millisecond) // Simulate DB delay

		// Return updated user data
		return User{
			ID:        "refresh:1",
			Name:      "Refresh User (Updated)",
			Email:     "refresh@example.com",
			CreatedAt: time.Now(),
		}, nil
	}

	if err := c.RefreshAhead(key, refreshBefore, loader); err != nil {
		log.Printf("Failed to set up refresh-ahead: %v", err)
		return
	}

	fmt.Printf("✓ Refresh-ahead scheduled for %s (refresh before: %v)\n", key, refreshBefore)

	// Monitor the cache for a while
	fmt.Println("✓ Monitoring cache for refresh activity...")
	for i := 0; i < 6; i++ {
		time.Sleep(5 * time.Second)

		if value, found, err := c.Get(key); err != nil {
			log.Printf("Failed to get %s: %v", key, err)
		} else if found {
			fmt.Printf("  - %s: %s (TTL: checking...)\n", key, value.Name)

			// Check TTL
			if ttl, err := c.TTL(key); err == nil {
				fmt.Printf("    TTL: %v\n", ttl)
			}
		} else {
			fmt.Printf("  - %s: not found\n", key)
		}
	}

	fmt.Println("✓ Refresh-ahead demo completed")
}

// Helper function to simulate database operations
func loadUserFromDB(ctx context.Context, id string) (User, error) {
	// Simulate database delay
	time.Sleep(50 * time.Millisecond)

	return User{
		ID:        id,
		Name:      "DB User",
		Email:     "db@example.com",
		CreatedAt: time.Now(),
	}, nil
}
