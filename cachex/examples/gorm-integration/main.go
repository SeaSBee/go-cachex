package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/gormx"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// User represents a user model
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Product represents a product model
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

// Order represents an order model
type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX GORM Integration Demo ===")

	// Create cache instance
	c, err := createCache()
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// Create GORM plugin
	plugin := createGORMPlugin(c)

	// Demo 1: Model Registration and Configuration
	demoModelRegistration(plugin)

	// Demo 2: Automatic Cache Invalidation (C/U/D)
	demoAutomaticInvalidation(plugin)

	// Demo 3: Read-Through Caching
	demoReadThrough(plugin)

	// Demo 4: Query Result Caching
	demoQueryResultCaching(plugin)

	// Demo 5: Tag-Based Invalidation
	demoTagBasedInvalidation(plugin)

	fmt.Println("\n=== GORM Integration Demo Complete ===")
}

func createCache() (cache.Cache[any], error) {
	// Try Redis first, fallback to memory
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		log.Printf("Redis not available, using memory store: %v", err)
		memoryStore, err := memorystore.New(&memorystore.Config{
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create memory store: %w", err)
		}
		return cache.New[any](
			cache.WithStore(memoryStore),
			cache.WithDefaultTTL(5*time.Minute),
		)
	}

	return cache.New[any](
		cache.WithStore(redisStore),
		cache.WithDefaultTTL(5*time.Minute),
	)
}

func createGORMPlugin(c cache.Cache[any]) *gormx.Plugin {
	config := &gormx.Config{
		EnableInvalidation: true,
		EnableReadThrough:  true,
		DefaultTTL:         5 * time.Minute,
		EnableQueryCache:   true,
		KeyPrefix:          "gorm",
		EnableDebug:        true,
		BatchSize:          100,
	}

	plugin := gormx.New(c, config)
	if err := plugin.Initialize(); err != nil {
		log.Fatalf("Failed to initialize GORM plugin: %v", err)
	}

	return plugin
}

func demoModelRegistration(plugin *gormx.Plugin) {
	fmt.Println("\n1. Model Registration and Configuration")
	fmt.Println("=======================================")

	// Register User model with custom configuration
	userConfig := &gormx.ModelConfig{
		Name:        "User",
		TTL:         10 * time.Minute,
		Enabled:     true,
		ReadThrough: true,
		Tags:        []string{"users", "auth"},
	}

	if err := plugin.RegisterModel(&User{}, userConfig); err != nil {
		log.Printf("Failed to register User model: %v", err)
	}

	// Register Product model with defaults
	if err := plugin.RegisterModelWithDefaults(&Product{}, 15*time.Minute, "products", "catalog"); err != nil {
		log.Printf("Failed to register Product model: %v", err)
	}

	// Register Order model
	if err := plugin.RegisterModelWithDefaults(&Order{}, 8*time.Minute, "orders", "transactions"); err != nil {
		log.Printf("Failed to register Order model: %v", err)
	}

	// List registered models
	models := plugin.GetRegisteredModels()
	fmt.Printf("✓ Registered models: %v\n", models)
}

func demoAutomaticInvalidation(plugin *gormx.Plugin) {
	fmt.Println("\n2. Automatic Cache Invalidation (C/U/D)")
	fmt.Println("=======================================")

	// Simulate user creation
	user := &User{
		ID:        "user:123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fmt.Println("✓ Creating user...")
	if err := plugin.InvalidateOnCreate(user); err != nil {
		log.Printf("Failed to invalidate on create: %v", err)
	} else {
		fmt.Println("  - Cache invalidated on user creation")
	}

	// Simulate user update
	fmt.Println("✓ Updating user...")
	user.Name = "John Smith"
	user.UpdatedAt = time.Now()
	if err := plugin.InvalidateOnUpdate(user); err != nil {
		log.Printf("Failed to invalidate on update: %v", err)
	} else {
		fmt.Println("  - Cache invalidated on user update")
	}

	// Simulate user deletion
	fmt.Println("✓ Deleting user...")
	if err := plugin.InvalidateOnDelete(user); err != nil {
		log.Printf("Failed to invalidate on delete: %v", err)
	} else {
		fmt.Println("  - Cache invalidated on user deletion")
	}
}

func demoReadThrough(plugin *gormx.Plugin) {
	fmt.Println("\n3. Read-Through Caching")
	fmt.Println("=======================")

	// Create a user to cache
	user := &User{
		ID:        "user:456",
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Cache the user first
	cacheKey := fmt.Sprintf("gorm:User:user:456")
	if err := plugin.GetCache().Set(cacheKey, user, 10*time.Minute); err != nil {
		log.Printf("Failed to cache user: %v", err)
		return
	}

	fmt.Println("✓ User cached for read-through test")

	// Simulate read-through query
	var dest User
	fmt.Println("✓ Attempting read-through query...")
	found, err := plugin.ReadThrough("User", "user:456", &dest)
	if err != nil {
		log.Printf("Read-through failed: %v", err)
		return
	}

	if found {
		fmt.Printf("  - Cache hit! Retrieved user: %s (%s)\n", dest.Name, dest.Email)
	} else {
		fmt.Println("  - Cache miss, would query database")
	}

	// Test with non-existent user
	fmt.Println("✓ Testing read-through with non-existent user...")
	var dest2 User
	found, err = plugin.ReadThrough("User", "user:999", &dest2)
	if err != nil {
		log.Printf("Read-through failed: %v", err)
		return
	}

	if found {
		fmt.Printf("  - Cache hit! Retrieved user: %s\n", dest2.Name)
	} else {
		fmt.Println("  - Cache miss, would query database")
	}
}

func demoQueryResultCaching(plugin *gormx.Plugin) {
	fmt.Println("\n4. Query Result Caching")
	fmt.Println("=======================")

	// Simulate database query result
	product := &Product{
		ID:          "product:789",
		Name:        "Laptop",
		Price:       999.99,
		Category:    "Electronics",
		Description: "High-performance laptop",
	}

	fmt.Println("✓ Caching query result...")
	if err := plugin.CacheQueryResult("Product", "product:789", product); err != nil {
		log.Printf("Failed to cache query result: %v", err)
		return
	}

	fmt.Println("  - Query result cached successfully")

	// Verify the result is cached
	fmt.Println("✓ Verifying cached result...")
	if cached, found, err := plugin.GetCache().Get("gorm:Product:product:789"); err == nil && found {
		if cachedProduct, ok := cached.(*Product); ok {
			fmt.Printf("  - Retrieved from cache: %s ($%.2f)\n", cachedProduct.Name, cachedProduct.Price)
		}
	} else {
		fmt.Println("  - Result not found in cache")
	}
}

func demoTagBasedInvalidation(plugin *gormx.Plugin) {
	fmt.Println("\n5. Tag-Based Invalidation")
	fmt.Println("==========================")

	// Create some test data with tags
	user1 := &User{ID: "user:tag1", Name: "Tag User 1", Email: "tag1@example.com"}
	user2 := &User{ID: "user:tag2", Name: "Tag User 2", Email: "tag2@example.com"}
	product1 := &Product{ID: "product:tag1", Name: "Tag Product 1", Price: 100.0}

	// Cache items with tags
	fmt.Println("✓ Caching items with tags...")
	if err := plugin.GetCache().Set("gorm:User:user:tag1", user1, 5*time.Minute); err == nil {
		plugin.GetCache().AddTags("gorm:User:user:tag1", "users", "auth")
	}
	if err := plugin.GetCache().Set("gorm:User:user:tag2", user2, 5*time.Minute); err == nil {
		plugin.GetCache().AddTags("gorm:User:user:tag2", "users", "auth")
	}
	if err := plugin.GetCache().Set("gorm:Product:product:tag1", product1, 5*time.Minute); err == nil {
		plugin.GetCache().AddTags("gorm:Product:product:tag1", "products", "catalog")
	}

	fmt.Println("  - Items cached with tags")

	// Invalidate by tag
	fmt.Println("✓ Invalidating all users...")
	if err := plugin.GetCache().InvalidateByTag("users"); err != nil {
		log.Printf("Failed to invalidate by tag: %v", err)
	} else {
		fmt.Println("  - All users invalidated")
	}

	// Check if items are still cached
	fmt.Println("✓ Checking cache status after invalidation...")
	if _, found, _ := plugin.GetCache().Get("gorm:User:user:tag1"); found {
		fmt.Println("  - User 1 still in cache (unexpected)")
	} else {
		fmt.Println("  - User 1 removed from cache (expected)")
	}

	if _, found, _ := plugin.GetCache().Get("gorm:Product:product:tag1"); found {
		fmt.Println("  - Product 1 still in cache (expected)")
	} else {
		fmt.Println("  - Product 1 removed from cache (unexpected)")
	}
}

// Helper function to simulate database operations
func simulateDatabaseQuery(modelName, id string) (interface{}, error) {
	// Simulate database delay
	time.Sleep(50 * time.Millisecond)

	switch modelName {
	case "User":
		return &User{
			ID:        id,
			Name:      "DB User",
			Email:     "db@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	case "Product":
		return &Product{
			ID:          id,
			Name:        "DB Product",
			Price:       199.99,
			Category:    "Database",
			Description: "Retrieved from database",
		}, nil
	default:
		return nil, fmt.Errorf("unknown model: %s", modelName)
	}
}
