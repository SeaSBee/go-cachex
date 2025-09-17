package main

import (
	"context"
	"fmt"
	"time"

	cachex "github.com/SeaSBee/go-cachex"
	"github.com/seasbee/go-logx"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Resilience Demo ===")

	// Example 1: Circuit Breaker
	demoCircuitBreaker()

	// Example 2: Retry Logic
	demoRetryLogic()

	// Example 3: Context Handling
	demoContextHandling()

	fmt.Println("\n=== Resilience Demo Complete ===")
}

func demoCircuitBreaker() {
	fmt.Println("\n1. Circuit Breaker")
	fmt.Println("==================")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache with circuit breaker
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Simulate some operations
	fmt.Println("\nSimulating operations...")

	ctx := context.Background()

	// Successful operations
	for i := 0; i < 5; i++ {
		user := User{
			ID:        fmt.Sprintf("user:%d", i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now(),
		}

		key := fmt.Sprintf("user:%d", i)
		setResult := <-c.Set(ctx, key, user, 0)
		if setResult.Error != nil {
			logx.Error("Failed to set user", logx.String("key", key), logx.ErrorField(setResult.Error))
		} else {
			fmt.Printf("  ✓ Set user %d\n", i)
		}
	}
}

func demoRetryLogic() {
	fmt.Println("\n2. Retry Logic")
	fmt.Println("==============")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache with retry configuration
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
		cachex.WithMaxRetries(3),
		cachex.WithRetryDelay(100*time.Millisecond),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Test retry logic with successful operations
	fmt.Println("Testing retry logic with successful operations...")

	user := User{
		ID:        "retry:user",
		Name:      "Retry User",
		Email:     "retry@example.com",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	key := "user:retry"
	setResult := <-c.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", key), logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ User set successfully: %s\n", key)

	// Retrieve user
	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", key), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved successfully: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
	} else {
		fmt.Println("✗ User not found")
	}

	// Test with non-existent key (should not retry for cache misses)
	fmt.Println("\nTesting with non-existent key...")
	getResult = <-c.Get(ctx, "non:existent:key")
	if getResult.Error != nil {
		logx.Error("Failed to get non-existent key", logx.ErrorField(getResult.Error))
		return
	}

	if !getResult.Found {
		fmt.Println("✓ Non-existent key handled correctly (not found)")
	} else {
		fmt.Println("✗ Unexpected: non-existent key was found")
	}
}

func demoContextHandling() {
	fmt.Println("\n3. Context Handling")
	fmt.Println("===================")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Test with context timeout
	fmt.Println("Testing with context timeout...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create cache with context
	cWithCtx := c.WithContext(ctx)

	// Store user with context
	user := User{
		ID:        "ctx:user",
		Name:      "Context User",
		Email:     "context@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:ctx"
	setResult := <-cWithCtx.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user with context", logx.String("key", key), logx.ErrorField(setResult.Error))
		return
	}
	fmt.Printf("✓ User set with context: %s\n", key)

	// Retrieve user with context
	getResult := <-cWithCtx.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user with context", logx.String("key", key), logx.ErrorField(getResult.Error))
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
		logx.Error("Expected error with cancelled context", logx.ErrorField(getResult.Error))
		fmt.Println("✓ Context cancellation handled correctly")
	} else {
		fmt.Println("✗ Unexpected: operation succeeded with cancelled context")
	}

	// Test with background context
	fmt.Println("\nTesting with background context...")

	cWithBgCtx := c.WithContext(context.Background())

	// Try to get user with background context
	getResult = <-cWithBgCtx.Get(context.Background(), key)
	if getResult.Error != nil {
		logx.Error("Failed to get user with background context", logx.String("key", key), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		fmt.Printf("✓ User retrieved with background context: %s (%s)\n", getResult.Value.Name, getResult.Value.Email)
	} else {
		fmt.Println("✗ User not found with background context")
	}
}
