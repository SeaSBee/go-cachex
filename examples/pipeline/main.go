package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Basic Operations Demo ===")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		fmt.Printf("Failed to create memory store: %v\n", err)
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		fmt.Printf("Failed to create cache: %v\n", err)
		return
	}
	defer c.Close()

	// Example 1: Basic Operations
	demoBasicOperations(c)

	// Example 2: Batch Operations
	demoBatchOperations(c)

	// Example 3: Context Operations
	demoContextOperations(c)

	fmt.Println("\n=== Basic Operations Demo Complete ===")
}

func demoBasicOperations(c cachex.Cache[User]) {
	fmt.Println("\n1. Basic Operations")
	fmt.Println("==================")

	// Create test user
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:1"

	ctx := context.Background()

	// Set operation
	fmt.Println("Setting user in cache...")
	setResult := <-c.Set(ctx, key, user, 10*time.Minute)
	if setResult.Error != nil {
		fmt.Printf("Failed to set user: %v\n", setResult.Error)
		return
	}
	fmt.Printf("✓ User set successfully: %s\n", key)

	// Get operation
	fmt.Println("Getting user from cache...")
	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		fmt.Printf("Failed to get user: %v\n", getResult.Error)
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
	} else {
		fmt.Println("✗ User not found in cache")
	}

	// Exists operation
	fmt.Println("Checking if user exists...")
	existsResult := <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		fmt.Printf("Failed to check existence: %v\n", existsResult.Error)
		return
	}
	fmt.Printf("✓ User exists: %t\n", existsResult.Found)

	// TTL operation
	fmt.Println("Getting TTL...")
	ttlResult := <-c.TTL(ctx, key)
	if ttlResult.Error != nil {
		fmt.Printf("Failed to get TTL: %v\n", ttlResult.Error)
		return
	}
	fmt.Printf("✓ TTL: %v\n", ttlResult.TTL)

	// Delete operation
	fmt.Println("Deleting user from cache...")
	delResult := <-c.Del(ctx, key)
	if delResult.Error != nil {
		fmt.Printf("Failed to delete user: %v\n", delResult.Error)
		return
	}
	fmt.Printf("✓ User deleted successfully\n")

	// Verify deletion
	existsResult = <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		fmt.Printf("Failed to check existence after deletion: %v\n", existsResult.Error)
		return
	}
	fmt.Printf("✓ User exists after deletion: %t\n", existsResult.Found)
}

func demoBatchOperations(c cachex.Cache[User]) {
	fmt.Println("\n2. Batch Operations")
	fmt.Println("==================")

	// Create multiple users
	users := map[string]User{
		"user:batch:1": {
			ID:        "user:batch:1",
			Name:      "Batch User 1",
			Email:     "batch1@example.com",
			CreatedAt: time.Now(),
		},
		"user:batch:2": {
			ID:        "user:batch:2",
			Name:      "Batch User 2",
			Email:     "batch2@example.com",
			CreatedAt: time.Now(),
		},
		"user:batch:3": {
			ID:        "user:batch:3",
			Name:      "Batch User 3",
			Email:     "batch3@example.com",
			CreatedAt: time.Now(),
		},
	}

	ctx := context.Background()

	// Batch set operation
	fmt.Println("Setting multiple users...")
	msetResult := <-c.MSet(ctx, users, 10*time.Minute)
	if msetResult.Error != nil {
		fmt.Printf("Failed to set multiple users: %v\n", msetResult.Error)
		return
	}
	fmt.Printf("✓ %d users set successfully\n", len(users))

	// Batch get operation
	fmt.Println("Getting multiple users...")
	keys := []string{"user:batch:1", "user:batch:2", "user:batch:3"}
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		fmt.Printf("Failed to get multiple users: %v\n", mgetResult.Error)
		return
	}
	fmt.Printf("✓ Retrieved %d users\n", len(mgetResult.Values))

	// Display retrieved users
	fmt.Println("Retrieved users:")
	for key, user := range mgetResult.Values {
		fmt.Printf("  %s: %s (%s)\n", key, user.Name, user.Email)
	}

	// Batch delete operation
	fmt.Println("Deleting multiple users...")
	delResult := <-c.Del(ctx, keys...)
	if delResult.Error != nil {
		fmt.Printf("Failed to delete multiple users: %v\n", delResult.Error)
		return
	}
	fmt.Printf("✓ %d users deleted successfully\n", len(keys))

	// Verify deletion
	existsResult := <-c.Exists(ctx, "user:batch:1")
	if existsResult.Error != nil {
		fmt.Printf("Failed to check existence: %v\n", existsResult.Error)
		return
	}
	fmt.Printf("✓ First user exists after deletion: %t\n", existsResult.Found)
}

func demoContextOperations(c cachex.Cache[User]) {
	fmt.Println("\n3. Context Operations")
	fmt.Println("====================")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create cache with context
	cWithCtx := c.WithContext(ctx)

	// Create test user
	user := User{
		ID:        "user:ctx",
		Name:      "Context User",
		Email:     "context@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:ctx"

	// Set with context
	fmt.Println("Setting user with context...")
	setResult := <-cWithCtx.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		fmt.Printf("Failed to set user with context: %v\n", setResult.Error)
		return
	}
	fmt.Printf("✓ User set with context: %s\n", key)

	// Get with context
	fmt.Println("Getting user with context...")
	getResult := <-cWithCtx.Get(ctx, key)
	if getResult.Error != nil {
		fmt.Printf("Failed to get user with context: %v\n", getResult.Error)
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved with context: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
	} else {
		fmt.Println("✗ User not found with context")
	}

	// Test with cancelled context
	fmt.Println("\nTesting with cancelled context...")

	// Create context and cancel it immediately
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2() // Cancel immediately

	// Create cache with cancelled context
	cWithCancelledCtx := c.WithContext(ctx2)

	// Try to get user with cancelled context
	getResult = <-cWithCancelledCtx.Get(ctx2, key)
	if getResult.Error != nil {
		fmt.Printf("✓ Expected error with cancelled context: %v\n", getResult.Error)
	} else {
		fmt.Println("✗ Unexpected: operation succeeded with cancelled context")
	}

	// Test with background context
	fmt.Println("\nTesting with background context...")

	cWithBgCtx := c.WithContext(context.Background())

	// Try to get user with background context
	getResult = <-cWithBgCtx.Get(context.Background(), key)
	if getResult.Error != nil {
		fmt.Printf("Failed to get user with background context: %v\n", getResult.Error)
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved with background context: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
	} else {
		fmt.Println("✗ User not found with background context")
	}
}
