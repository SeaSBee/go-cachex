package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/SeaSBee/go-logx"
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
	Description string  `json:"description"`
}

func main() {
	fmt.Println("=== Go-CacheX GORM Integration Demo ===")

	// Example 1: Basic GORM plugin usage
	fmt.Println("\n=== Example 1: Basic GORM Plugin Usage ===")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for GORM integration",
			logx.String("store_type", "memory"),
			logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache with GORM plugin
	cache, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache for GORM integration",
			logx.String("cache_type", "User"),
			logx.ErrorField(err))
		return
	}
	defer cache.Close()

	// Note: GORM plugin integration would be used in actual GORM applications
	// This example demonstrates core caching functionality with structured logging

	// Simulate user operations
	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Set user in cache
	setResult := <-cache.Set(ctx, "user:123", user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user in cache for GORM demo",
			logx.String("key", "user:123"),
			logx.String("user_id", user.ID),
			logx.ErrorField(setResult.Error))
	} else {
		fmt.Printf("✅ Set user: %s\n", user.Name)
	}

	// Get user from cache
	getResult := <-cache.Get(ctx, "user:123")
	if getResult.Error != nil {
		logx.Error("Failed to get user from cache for GORM demo",
			logx.String("key", "user:123"),
			logx.ErrorField(getResult.Error))
	} else if getResult.Found {
		fmt.Printf("✅ Retrieved user: %s\n", getResult.Value.Name)
		logx.Info("Successfully retrieved user from cache in GORM demo",
			logx.String("key", "user:123"),
			logx.String("user_id", getResult.Value.ID),
			logx.String("user_name", getResult.Value.Name),
			logx.Bool("found", getResult.Found))
	}

	// Check existence
	existsResult := <-cache.Exists(ctx, "user:123")
	if existsResult.Error != nil {
		logx.Error("Failed to check user existence in cache",
			logx.String("key", "user:123"),
			logx.ErrorField(existsResult.Error))
	} else {
		fmt.Printf("✅ User exists: %t\n", existsResult.Found)
	}

	// Get TTL
	ttlResult := <-cache.TTL(ctx, "user:123")
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL for user in cache",
			logx.String("key", "user:123"),
			logx.ErrorField(ttlResult.Error))
	} else {
		fmt.Printf("✅ User TTL: %v\n", ttlResult.TTL)
	}

	// Delete user
	delResult := <-cache.Del(ctx, "user:123")
	if delResult.Error != nil {
		logx.Error("Failed to delete user from cache",
			logx.String("key", "user:123"),
			logx.ErrorField(delResult.Error))
	} else {
		fmt.Printf("✅ Deleted user\n")
	}

	// Check existence after deletion
	existsResult = <-cache.Exists(ctx, "user:123")
	if existsResult.Error != nil {
		logx.Error("Failed to check user existence after deletion",
			logx.String("key", "user:123"),
			logx.ErrorField(existsResult.Error))
	} else {
		fmt.Printf("✅ User exists after deletion: %t\n", existsResult.Found)
	}

	// Example 2: Multiple model types
	fmt.Println("\n=== Example 2: Multiple Model Types ===")

	// Create separate caches for different model types
	userStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for user cache",
			logx.String("store_type", "memory"),
			logx.String("cache_purpose", "user_cache"),
			logx.ErrorField(err))
		return
	}
	defer userStore.Close()

	productStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for product cache",
			logx.String("store_type", "memory"),
			logx.String("cache_purpose", "product_cache"),
			logx.ErrorField(err))
		return
	}
	defer productStore.Close()

	userCache, err := cachex.New[User](
		cachex.WithStore(userStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create user cache",
			logx.String("cache_type", "User"),
			logx.ErrorField(err))
		return
	}
	defer userCache.Close()

	productCache, err := cachex.New[Product](
		cachex.WithStore(productStore),
		cachex.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create product cache",
			logx.String("cache_type", "Product"),
			logx.ErrorField(err))
		return
	}
	defer productCache.Close()

	// Note: GORM plugin integration would be used in actual GORM applications
	// This example demonstrates core caching functionality with structured logging

	// Simulate operations with multiple users and products
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	products := []Product{
		{ID: "1", Name: "Laptop", Price: 999.99, Description: "High-performance laptop"},
		{ID: "2", Name: "Mouse", Price: 29.99, Description: "Wireless mouse"},
		{ID: "3", Name: "Keyboard", Price: 79.99, Description: "Mechanical keyboard"},
	}

	// Set users
	for _, user := range users {
		setResult := <-userCache.Set(ctx, fmt.Sprintf("user:%s", user.ID), user, 0)
		if setResult.Error != nil {
			logx.Error("Failed to set user in multi-model demo",
				logx.String("key", fmt.Sprintf("user:%s", user.ID)),
				logx.String("user_id", user.ID),
				logx.ErrorField(setResult.Error))
		} else {
			fmt.Printf("✅ Set user: %s\n", user.Name)
		}
	}

	// Set products
	for _, product := range products {
		setResult := <-productCache.Set(ctx, fmt.Sprintf("product:%s", product.ID), product, 0)
		if setResult.Error != nil {
			logx.Error("Failed to set product in multi-model demo",
				logx.String("key", fmt.Sprintf("product:%s", product.ID)),
				logx.String("product_id", product.ID),
				logx.ErrorField(setResult.Error))
		} else {
			fmt.Printf("✅ Set product: %s\n", product.Name)
		}
	}

	// Get users
	userKeys := []string{"user:1", "user:2", "user:3"}
	userResults := <-userCache.MGet(ctx, userKeys...)
	if userResults.Error != nil {
		logx.Error("Failed to get multiple users from cache",
			logx.String("keys", fmt.Sprintf("%v", userKeys)),
			logx.ErrorField(userResults.Error))
	} else {
		fmt.Printf("✅ Retrieved %d users\n", len(userResults.Values))
		logx.Info("Successfully retrieved multiple users from cache",
			logx.Int("users_retrieved", len(userResults.Values)),
			logx.String("user_keys", fmt.Sprintf("%v", userKeys)))
	}

	// Get products
	productKeys := []string{"product:1", "product:2", "product:3"}
	productResults := <-productCache.MGet(ctx, productKeys...)
	if productResults.Error != nil {
		logx.Error("Failed to get multiple products from cache",
			logx.String("keys", fmt.Sprintf("%v", productKeys)),
			logx.ErrorField(productResults.Error))
	} else {
		fmt.Printf("✅ Retrieved %d products\n", len(productResults.Values))
		logx.Info("Successfully retrieved multiple products from cache",
			logx.Int("products_retrieved", len(productResults.Values)),
			logx.String("product_keys", fmt.Sprintf("%v", productKeys)))
	}

	// Example 3: Context-aware operations
	fmt.Println("\n=== Example 3: Context-Aware Operations ===")

	// Create cache with context support
	ctxStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for context demo",
			logx.String("store_type", "memory"),
			logx.String("demo_type", "context_aware"),
			logx.ErrorField(err))
		return
	}
	defer ctxStore.Close()

	ctxCache, err := cachex.New[User](
		cachex.WithStore(ctxStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create context-aware cache",
			logx.String("cache_type", "User"),
			logx.String("demo_type", "context_aware"),
			logx.ErrorField(err))
		return
	}
	defer ctxCache.Close()

	// Note: GORM plugin integration would be used in actual GORM applications
	// This example demonstrates context-aware caching with structured logging

	// Simulate context-aware operations
	ctxUser := User{
		ID:        "ctx-123",
		Name:      "Context User",
		Email:     "context@example.com",
		CreatedAt: time.Now(),
	}

	// Set with context
	setResult = <-ctxCache.Set(ctx, "ctx_user:123", ctxUser, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user with context",
			logx.String("key", "ctx_user:123"),
			logx.String("user_id", ctxUser.ID),
			logx.ErrorField(setResult.Error))
	} else {
		fmt.Printf("✅ Set context user: %s\n", ctxUser.Name)
	}

	// Get with context
	getResult = <-ctxCache.Get(ctx, "ctx_user:123")
	if getResult.Error != nil {
		logx.Error("Failed to get user with context",
			logx.String("key", "ctx_user:123"),
			logx.ErrorField(getResult.Error))
	} else if getResult.Found {
		fmt.Printf("✅ Retrieved context user: %s\n", getResult.Value.Name)
		logx.Info("Successfully retrieved context-aware user from cache",
			logx.String("key", "ctx_user:123"),
			logx.String("user_id", getResult.Value.ID),
			logx.String("user_name", getResult.Value.Name),
			logx.Bool("found", getResult.Found))
	}

	// Test with non-existent user
	getResult = <-ctxCache.Get(ctx, "ctx_user:999")
	if getResult.Error != nil {
		logx.Error("Failed to get non-existent user with context",
			logx.String("key", "ctx_user:999"),
			logx.ErrorField(getResult.Error))
	} else {
		fmt.Printf("✅ Non-existent user found: %t\n", getResult.Found)
	}

	// Example 4: Read-through caching
	fmt.Println("\n=== Example 4: Read-Through Caching ===")

	// Create cache with read-through support
	rtStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for read-through demo",
			logx.String("store_type", "memory"),
			logx.String("demo_type", "read_through"),
			logx.ErrorField(err))
		return
	}
	defer rtStore.Close()

	rtCache, err := cachex.New[User](
		cachex.WithStore(rtStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create read-through cache",
			logx.String("cache_type", "User"),
			logx.String("demo_type", "read_through"),
			logx.ErrorField(err))
		return
	}
	defer rtCache.Close()

	// Note: GORM plugin integration would be used in actual GORM applications
	// This example demonstrates read-through caching with structured logging

	// Simulate read-through with loader function
	rtResult := <-rtCache.ReadThrough(ctx, "rt_user:456", 10*time.Minute, func(ctx context.Context) (User, error) {
		// Simulate loading from database
		logx.Info("Loading user from database via read-through",
			logx.String("user_id", "456"),
			logx.String("operation", "read_through_loader"))

		return User{
			ID:        "456",
			Name:      "Read-Through User",
			Email:     "readthrough@example.com",
			CreatedAt: time.Now(),
		}, nil
	})
	if rtResult.Error != nil {
		logx.Error("Failed to read-through user",
			logx.String("key", "rt_user:456"),
			logx.ErrorField(rtResult.Error))
	} else {
		fmt.Printf("✅ Read-through user: %s\n", rtResult.Value.Name)
		logx.Info("Successfully loaded user via read-through caching",
			logx.String("key", "rt_user:456"),
			logx.String("user_id", rtResult.Value.ID),
			logx.String("user_name", rtResult.Value.Name))
	}

	// Example 5: Write-through and write-behind patterns
	fmt.Println("\n=== Example 5: Write-Through and Write-Behind Patterns ===")

	// Create cache for write patterns
	wtStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store for write patterns demo",
			logx.String("store_type", "memory"),
			logx.String("demo_type", "write_patterns"),
			logx.ErrorField(err))
		return
	}
	defer wtStore.Close()

	wtCache, err := cachex.New[User](
		cachex.WithStore(wtStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create write patterns cache",
			logx.String("cache_type", "User"),
			logx.String("demo_type", "write_patterns"),
			logx.ErrorField(err))
		return
	}
	defer wtCache.Close()

	// Note: GORM plugin integration would be used in actual GORM applications
	// This example demonstrates write patterns with structured logging

	// Simulate write-through
	wtUser := User{
		ID:        "789",
		Name:      "Write-Through User",
		Email:     "writethrough@example.com",
		CreatedAt: time.Now(),
	}

	wtResult := <-wtCache.WriteThrough(ctx, "wt_user:789", wtUser, 10*time.Minute, func(ctx context.Context) error {
		// Simulate writing to database
		logx.Info("Writing user to database via write-through",
			logx.String("user_id", "789"),
			logx.String("operation", "write_through_writer"))
		return nil
	})
	if wtResult.Error != nil {
		logx.Error("Failed to write-through user",
			logx.String("key", "wt_user:789"),
			logx.String("user_id", wtUser.ID),
			logx.ErrorField(wtResult.Error))
	} else {
		fmt.Printf("✅ Write-through user: %s\n", wtUser.Name)
		logx.Info("Successfully wrote user via write-through pattern",
			logx.String("key", "wt_user:789"),
			logx.String("user_id", wtUser.ID),
			logx.String("user_name", wtUser.Name))
	}

	// Simulate write-behind
	wbUser := User{
		ID:        "101",
		Name:      "Write-Behind User",
		Email:     "writebehind@example.com",
		CreatedAt: time.Now(),
	}

	wbResult := <-wtCache.WriteBehind(ctx, "wt_user:101", wbUser, 10*time.Minute, func(ctx context.Context) error {
		// Simulate writing to database
		logx.Info("Writing user to database via write-behind",
			logx.String("user_id", "101"),
			logx.String("operation", "write_behind_writer"))
		return nil
	})
	if wbResult.Error != nil {
		logx.Error("Failed to write-behind user",
			logx.String("key", "wt_user:101"),
			logx.String("user_id", wbUser.ID),
			logx.ErrorField(wbResult.Error))
	} else {
		fmt.Printf("✅ Write-behind user: %s\n", wbUser.Name)
		logx.Info("Successfully queued user for write-behind pattern",
			logx.String("key", "wt_user:101"),
			logx.String("user_id", wbUser.ID),
			logx.String("user_name", wbUser.Name))
	}

	fmt.Println("\n=== GORM Integration Demo Complete ===")
	logx.Info("GORM integration demo completed successfully",
		logx.String("demo_type", "gorm_integration"),
		logx.Int("examples_completed", 5))
}
