package main

import (
	"context"
	"fmt"
	"time"

	"github.com/seasbee/go-cachex"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	OrgID     string    `json:"org_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Product struct for demonstration
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	InStock  bool    `json:"in_stock"`
}

// Order struct for demonstration
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
	fmt.Println("=== Go-CacheX Key Generation Scheme Demo ===")

	// Example 1: Basic Key Generation
	demoBasicKeyGeneration()

	// Example 2: Namespaced Keys
	demoNamespacedKeys()

	// Example 3: List Keys with Filter Hashing
	demoListKeysWithHashing()

	// Example 4: Composite Keys for Joins
	demoCompositeKeys()

	// Example 5: Session/Token Keys
	demoSessionKeys()

	// Example 6: Helper Methods
	demoHelperMethods()

	// Example 7: HMAC-SHA256 Security
	demoHMACSecurity()

	// Example 8: Multi-Environment Isolation
	demoMultiEnvironmentIsolation()

	// Example 9: Cache Integration
	demoCacheIntegration()

	fmt.Println("\n=== Key Generation Demo Complete ===")
}

func demoBasicKeyGeneration() {
	fmt.Println("1. Basic Key Generation")
	fmt.Println("=========================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// Entity keys
	userKey := builder.Build("user", "12345")
	fmt.Printf("User Key: %s\n", userKey)

	productKey := builder.Build("product", "67890")
	fmt.Printf("Product Key: %s\n", productKey)

	// List keys
	listKey := builder.BuildList("users", nil)
	fmt.Printf("Users List Key: %s\n", listKey)

	// Composite keys
	compositeKey := builder.BuildComposite("user", "123", "order", "456")
	fmt.Printf("User-Order Composite Key: %s\n", compositeKey)

	// Session keys
	sessionKey := builder.BuildSession("abc123def456")
	fmt.Printf("Session Key: %s\n", sessionKey)

	fmt.Println()
}

func demoNamespacedKeys() {
	fmt.Println("2. Namespaced Keys")
	fmt.Println("===================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// Demonstrate the exact format you specified
	userKey := builder.Build("user", "12345")
	fmt.Printf("Expected: app:seasbee:env:prod:user:12345\n")
	fmt.Printf("Actual:   %s\n", userKey)
	fmt.Printf("Match:    %t\n\n", userKey == "app:seasbee:env:prod:user:12345")

	// Different entities
	entities := []string{"user", "product", "order", "org", "feature"}
	for _, entity := range entities {
		key := builder.Build(entity, "12345")
		fmt.Printf("%s: %s\n", entity, key)
	}
	fmt.Println()
}

func demoListKeysWithHashing() {
	fmt.Println("3. List Keys with Filter Hashing")
	fmt.Println("=================================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// List key without filters
	allUsersKey := builder.BuildList("users", nil)
	fmt.Printf("All Users: %s\n", allUsersKey)

	// List key with filters
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"org_id": "123",
	}
	filteredUsersKey := builder.BuildList("users", filters)
	fmt.Printf("Filtered Users: %s\n", filteredUsersKey)

	// Demonstrate consistent hashing
	filters2 := map[string]any{
		"org_id": "123",
		"role":   "admin",
		"status": "active",
	}
	filteredUsersKey2 := builder.BuildList("users", filters2)
	fmt.Printf("Same Filters (different order): %s\n", filteredUsersKey2)
	fmt.Printf("Consistent: %t\n", filteredUsersKey == filteredUsersKey2)

	// Different filter combinations
	filterCombinations := []map[string]any{
		{"status": "active"},
		{"role": "admin"},
		{"status": "active", "role": "admin"},
		{"status": "inactive", "role": "user"},
	}

	for i, filters := range filterCombinations {
		key := builder.BuildList("users", filters)
		fmt.Printf("Filters %d: %s\n", i+1, key)
	}
	fmt.Println()
}

func demoCompositeKeys() {
	fmt.Println("4. Composite Keys for Joins")
	fmt.Println("============================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// User-Order relationships
	userOrderKey := builder.BuildComposite("user", "123", "order", "456")
	fmt.Printf("User-Order: %s\n", userOrderKey)

	// User-Product relationships
	userProductKey := builder.BuildComposite("user", "123", "product", "789")
	fmt.Printf("User-Product: %s\n", userProductKey)

	// Org-User relationships
	orgUserKey := builder.BuildComposite("org", "101", "user", "202")
	fmt.Printf("Org-User: %s\n", orgUserKey)

	// Demonstrate the exact format you specified
	expected := "app:seasbee:env:prod:user:123:order:456"
	fmt.Printf("Expected: %s\n", expected)
	fmt.Printf("Actual:   %s\n", userOrderKey)
	fmt.Printf("Match:    %t\n", userOrderKey == expected)
	fmt.Println()
}

func demoSessionKeys() {
	fmt.Println("5. Session/Token Keys")
	fmt.Println("=====================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// Session keys
	sessionID := "abc123def456ghi789"
	sessionKey := builder.BuildSession(sessionID)
	fmt.Printf("Session: %s\n", sessionKey)

	// Token keys (using session for tokens)
	tokenID := "jwt_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	tokenKey := builder.BuildSession(tokenID)
	fmt.Printf("Token: %s\n", tokenKey)

	// Demonstrate the exact format you specified
	expected := "app:seasbee:env:prod:session:" + sessionID
	fmt.Printf("Expected: %s\n", expected)
	fmt.Printf("Actual:   %s\n", sessionKey)
	fmt.Printf("Match:    %t\n", sessionKey == expected)
	fmt.Println()
}

func demoHelperMethods() {
	fmt.Println("6. Helper Methods")
	fmt.Println("=================")

	builder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create builder: %v\n", err)
		return
	}

	// All the helper methods you specified
	userKey := builder.BuildUser("12345")
	orgKey := builder.BuildOrg("67890")
	productKey := builder.BuildProduct("101112")
	orderKey := builder.BuildOrder("131415")

	fmt.Printf("UserKey(userID): %s\n", userKey)
	fmt.Printf("OrgKey(orgID): %s\n", orgKey)
	fmt.Printf("ProductKey(pid): %s\n", productKey)
	fmt.Printf("OrderKey(oid): %s\n", orderKey)

	// Verify they match the Build method
	fmt.Printf("\nVerification:\n")
	fmt.Printf("UserKey == Build('user'): %t\n", userKey == builder.Build("user", "12345"))
	fmt.Printf("OrgKey == Build('org'): %t\n", orgKey == builder.Build("org", "67890"))
	fmt.Printf("ProductKey == Build('product'): %t\n", productKey == builder.Build("product", "101112"))
	fmt.Printf("OrderKey == Build('order'): %t\n", orderKey == builder.Build("order", "131415"))
	fmt.Println()
}

func demoHMACSecurity() {
	fmt.Println("7. HMAC-SHA256 Security")
	fmt.Println("=======================")

	// Builder with secret
	builderWithSecret, err1 := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err1 != nil {
		fmt.Printf("Failed to create builder with secret: %v\n", err1)
		return
	}

	// Builder without secret
	builderWithoutSecret, err2 := cachex.NewBuilder("seasbee", "prod", "")
	if err2 != nil {
		fmt.Printf("Failed to create builder without secret: %v\n", err2)
		return
	}

	// Test data
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"org_id": "123",
	}

	// Generate keys with both builders
	keyWithSecret := builderWithSecret.BuildList("users", filters)
	keyWithoutSecret := builderWithoutSecret.BuildList("users", filters)

	fmt.Printf("With Secret: %s\n", keyWithSecret)
	fmt.Printf("Without Secret: %s\n", keyWithoutSecret)
	fmt.Printf("Different: %t\n", keyWithSecret != keyWithoutSecret)

	// Test consistency
	keyWithSecret2 := builderWithSecret.BuildList("users", filters)
	fmt.Printf("Consistent with secret: %t\n", keyWithSecret == keyWithSecret2)

	keyWithoutSecret2 := builderWithoutSecret.BuildList("users", filters)
	fmt.Printf("Consistent without secret: %t\n", keyWithoutSecret == keyWithoutSecret2)

	// Test different secrets produce different hashes
	builderWithSecret2, err3 := cachex.NewBuilder("seasbee", "prod", "different_secret")
	if err3 != nil {
		fmt.Printf("Failed to create builder with different secret: %v\n", err3)
		return
	}
	keyWithSecret3 := builderWithSecret2.BuildList("users", filters)
	fmt.Printf("Different secrets produce different keys: %t\n", keyWithSecret != keyWithSecret3)
	fmt.Println()
}

func demoMultiEnvironmentIsolation() {
	fmt.Println("8. Multi-Environment Isolation")
	fmt.Println("==============================")

	// Different environments
	environments := []string{"dev", "staging", "prod"}
	userID := "12345"

	for _, env := range environments {
		builder, err := cachex.NewBuilder("seasbee", env, "secret123")
		if err != nil {
			fmt.Printf("Failed to create builder for env %s: %v\n", env, err)
			continue
		}
		userKey := builder.BuildUser(userID)
		fmt.Printf("%s: %s\n", env, userKey)
	}

	// Different applications
	apps := []string{"app1", "app2", "seasbee"}
	env := "prod"

	fmt.Printf("\nDifferent Applications:\n")
	for _, app := range apps {
		builder, err := cachex.NewBuilder(app, env, "secret123")
		if err != nil {
			fmt.Printf("Failed to create builder for app %s: %v\n", app, err)
			continue
		}
		userKey := builder.BuildUser(userID)
		fmt.Printf("%s: %s\n", app, userKey)
	}
	fmt.Println()
}

func demoCacheIntegration() {
	fmt.Println("9. Cache Integration")
	fmt.Println("====================")

	// Create memory store for demo
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		fmt.Printf("Failed to create memory store: %v\n", err)
		return
	}
	defer memoryStore.Close()

	// Create key builder
	keyBuilder, err := cachex.NewBuilder("seasbee", "prod", "secret123")
	if err != nil {
		fmt.Printf("Failed to create key builder: %v\n", err)
		return
	}

	// Create codec
	jsonCodec := cachex.NewJSONCodec()

	// Create cache
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithCodec(jsonCodec),
		cachex.WithKeyBuilder(keyBuilder),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		fmt.Printf("Failed to create cache: %v\n", err)
		return
	}
	defer c.Close()

	// Create test user
	user := User{
		ID:        "12345",
		Name:      "John Doe",
		Email:     "john@example.com",
		Status:    "active",
		Role:      "admin",
		OrgID:     "67890",
		CreatedAt: time.Now(),
	}

	// Use key builder to generate cache key
	userKey := keyBuilder.BuildUser(user.ID)
	fmt.Printf("Generated Key: %s\n", userKey)

	// Store user in cache
	ctx := context.Background()
	setResult := <-c.Set(ctx, userKey, user, 10*time.Minute)
	if setResult.Error != nil {
		fmt.Printf("Failed to set user in cache: %v\n", setResult.Error)
		return
	}

	// Retrieve user from cache
	getResult := <-c.Get(ctx, userKey)
	if getResult.Error != nil {
		fmt.Printf("Failed to get user from cache: %v\n", getResult.Error)
		return
	}

	if getResult.Found {
		fmt.Printf("Retrieved from cache: %+v\n", getResult.Value)
		fmt.Printf("Cache integration successful!\n")
	} else {
		fmt.Printf("User not found in cache\n")
	}

	// Test list key caching
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
	}
	listKey := keyBuilder.BuildList("users", filters)
	fmt.Printf("List Key for caching: %s\n", listKey)

	// Test composite key caching
	compositeKey := keyBuilder.BuildComposite("user", "123", "order", "456")
	fmt.Printf("Composite Key for caching: %s\n", compositeKey)

	fmt.Println()
}
