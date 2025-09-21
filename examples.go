package cachex

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ExampleAllDataTypes demonstrates usage with all common Go data types
func ExampleAllDataTypes() {
	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create a base cache instance (no type parameter needed)
	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	// =============================================================================
	// 1. STRING TYPE
	// =============================================================================
	fmt.Println("=== STRING TYPE ===")
	stringCache := NewTypedCache[string](cache)

	// Set string value
	setResult := <-stringCache.Set(ctx, "user:name", "John Doe", time.Hour)
	if setResult.Error != nil {
		fmt.Printf("Error setting string: %v\n", setResult.Error)
	} else {
		fmt.Println("✓ String set successfully")
	}

	// Get string value
	getResult := <-stringCache.Get(ctx, "user:name")
	if getResult.Error != nil {
		fmt.Printf("Error getting string: %v\n", getResult.Error)
	} else if getResult.Found {
		fmt.Printf("✓ Retrieved string: %s\n", getResult.Value)
	}

	// =============================================================================
	// 2. INTEGER TYPES
	// =============================================================================
	fmt.Println("\n=== INTEGER TYPES ===")

	// int
	intCache := NewTypedCache[int](cache)
	<-intCache.Set(ctx, "user:age", 30, time.Hour)
	intResult := <-intCache.Get(ctx, "user:age")
	if intResult.Found {
		fmt.Printf("✓ Retrieved int: %d\n", intResult.Value)
	}

	// int64
	int64Cache := NewTypedCache[int64](cache)
	<-int64Cache.Set(ctx, "user:timestamp", time.Now().Unix(), time.Hour)
	int64Result := <-int64Cache.Get(ctx, "user:timestamp")
	if int64Result.Found {
		fmt.Printf("✓ Retrieved int64: %d\n", int64Result.Value)
	}

	// int32
	int32Cache := NewTypedCache[int32](cache)
	<-int32Cache.Set(ctx, "user:score", int32(95), time.Hour)
	int32Result := <-int32Cache.Get(ctx, "user:score")
	if int32Result.Found {
		fmt.Printf("✓ Retrieved int32: %d\n", int32Result.Value)
	}

	// uint
	uintCache := NewTypedCache[uint](cache)
	<-uintCache.Set(ctx, "user:count", uint(100), time.Hour)
	uintResult := <-uintCache.Get(ctx, "user:count")
	if uintResult.Found {
		fmt.Printf("✓ Retrieved uint: %d\n", uintResult.Value)
	}

	// =============================================================================
	// 3. FLOATING POINT TYPES
	// =============================================================================
	fmt.Println("\n=== FLOATING POINT TYPES ===")

	// float64
	float64Cache := NewTypedCache[float64](cache)
	<-float64Cache.Set(ctx, "user:balance", 1234.56, time.Hour)
	float64Result := <-float64Cache.Get(ctx, "user:balance")
	if float64Result.Found {
		fmt.Printf("✓ Retrieved float64: %.2f\n", float64Result.Value)
	}

	// float32
	float32Cache := NewTypedCache[float32](cache)
	<-float32Cache.Set(ctx, "user:rating", float32(4.5), time.Hour)
	float32Result := <-float32Cache.Get(ctx, "user:rating")
	if float32Result.Found {
		fmt.Printf("✓ Retrieved float32: %.1f\n", float32Result.Value)
	}

	// =============================================================================
	// 4. BOOLEAN TYPE
	// =============================================================================
	fmt.Println("\n=== BOOLEAN TYPE ===")
	boolCache := NewTypedCache[bool](cache)

	<-boolCache.Set(ctx, "user:active", true, time.Hour)
	boolResult := <-boolCache.Get(ctx, "user:active")
	if boolResult.Found {
		fmt.Printf("✓ Retrieved bool: %t\n", boolResult.Value)
	}

	// =============================================================================
	// 5. SLICE TYPES
	// =============================================================================
	fmt.Println("\n=== SLICE TYPES ===")

	// []string
	stringSliceCache := NewTypedCache[[]string](cache)
	hobbies := []string{"reading", "swimming", "coding"}
	<-stringSliceCache.Set(ctx, "user:hobbies", hobbies, time.Hour)
	stringSliceResult := <-stringSliceCache.Get(ctx, "user:hobbies")
	if stringSliceResult.Found {
		fmt.Printf("✓ Retrieved []string: %v\n", stringSliceResult.Value)
	}

	// []int
	intSliceCache := NewTypedCache[[]int](cache)
	scores := []int{85, 92, 78, 96}
	<-intSliceCache.Set(ctx, "user:scores", scores, time.Hour)
	intSliceResult := <-intSliceCache.Get(ctx, "user:scores")
	if intSliceResult.Found {
		fmt.Printf("✓ Retrieved []int: %v\n", intSliceResult.Value)
	}

	// =============================================================================
	// 6. MAP TYPES
	// =============================================================================
	fmt.Println("\n=== MAP TYPES ===")

	// map[string]string
	stringMapCache := NewTypedCache[map[string]string](cache)
	settings := map[string]string{
		"theme":    "dark",
		"language": "en",
		"timezone": "UTC",
	}
	<-stringMapCache.Set(ctx, "user:settings", settings, time.Hour)
	stringMapResult := <-stringMapCache.Get(ctx, "user:settings")
	if stringMapResult.Found {
		fmt.Printf("✓ Retrieved map[string]string: %v\n", stringMapResult.Value)
	}

	// map[string]int
	stringIntMapCache := NewTypedCache[map[string]int](cache)
	stats := map[string]int{
		"posts":     150,
		"followers": 1200,
		"following": 300,
	}
	<-stringIntMapCache.Set(ctx, "user:stats", stats, time.Hour)
	stringIntMapResult := <-stringIntMapCache.Get(ctx, "user:stats")
	if stringIntMapResult.Found {
		fmt.Printf("✓ Retrieved map[string]int: %v\n", stringIntMapResult.Value)
	}

	// =============================================================================
	// 7. STRUCT TYPES
	// =============================================================================
	fmt.Println("\n=== STRUCT TYPES ===")

	// Simple struct
	type User struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Email  string  `json:"email"`
		Active bool    `json:"active"`
		Score  float64 `json:"score"`
	}

	userCache := NewTypedCache[User](cache)
	user := User{
		ID:     1,
		Name:   "Jane Smith",
		Email:  "jane@example.com",
		Active: true,
		Score:  95.5,
	}
	<-userCache.Set(ctx, "user:profile", user, time.Hour)
	userResult := <-userCache.Get(ctx, "user:profile")
	if userResult.Found {
		fmt.Printf("✓ Retrieved User struct: %+v\n", userResult.Value)
	}

	// Complex nested struct
	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
		ZipCode string `json:"zip_code"`
	}

	type Company struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
		Founded int     `json:"founded"`
	}

	type Employee struct {
		ID      int      `json:"id"`
		Name    string   `json:"name"`
		Company Company  `json:"company"`
		Skills  []string `json:"skills"`
		Salary  float64  `json:"salary"`
	}

	employeeCache := NewTypedCache[Employee](cache)
	employee := Employee{
		ID:   100,
		Name: "Bob Johnson",
		Company: Company{
			Name: "Tech Corp",
			Address: Address{
				Street:  "123 Tech St",
				City:    "San Francisco",
				Country: "USA",
				ZipCode: "94105",
			},
			Founded: 2010,
		},
		Skills: []string{"Go", "Python", "JavaScript", "Docker"},
		Salary: 120000.0,
	}
	<-employeeCache.Set(ctx, "employee:100", employee, time.Hour)
	employeeResult := <-employeeCache.Get(ctx, "employee:100")
	if employeeResult.Found {
		fmt.Printf("✓ Retrieved Employee struct: %+v\n", employeeResult.Value)
	}

	// =============================================================================
	// 8. POINTER TYPES
	// =============================================================================
	fmt.Println("\n=== POINTER TYPES ===")

	// *string
	stringPtrCache := NewTypedCache[*string](cache)
	message := "Hello, World!"
	<-stringPtrCache.Set(ctx, "message:ptr", &message, time.Hour)
	stringPtrResult := <-stringPtrCache.Get(ctx, "message:ptr")
	if stringPtrResult.Found && stringPtrResult.Value != nil {
		fmt.Printf("✓ Retrieved *string: %s\n", *stringPtrResult.Value)
	}

	// *int
	intPtrCache := NewTypedCache[*int](cache)
	number := 42
	<-intPtrCache.Set(ctx, "number:ptr", &number, time.Hour)
	intPtrResult := <-intPtrCache.Get(ctx, "number:ptr")
	if intPtrResult.Found && intPtrResult.Value != nil {
		fmt.Printf("✓ Retrieved *int: %d\n", *intPtrResult.Value)
	}

	// =============================================================================
	// 9. INTERFACE TYPES
	// =============================================================================
	fmt.Println("\n=== INTERFACE TYPES ===")

	// any/interface{}
	anyCache := NewTypedCache[any](cache)

	// Store different types as any
	<-anyCache.Set(ctx, "data:mixed1", "string data", time.Hour)
	<-anyCache.Set(ctx, "data:mixed2", 123, time.Hour)
	<-anyCache.Set(ctx, "data:mixed3", true, time.Hour)
	<-anyCache.Set(ctx, "data:mixed4", []string{"a", "b", "c"}, time.Hour)

	// Retrieve and type assert
	anyResult1 := <-anyCache.Get(ctx, "data:mixed1")
	if anyResult1.Found {
		if str, ok := anyResult1.Value.(string); ok {
			fmt.Printf("✓ Retrieved any as string: %s\n", str)
		}
	}

	anyResult2 := <-anyCache.Get(ctx, "data:mixed2")
	if anyResult2.Found {
		if num, ok := anyResult2.Value.(int); ok {
			fmt.Printf("✓ Retrieved any as int: %d\n", num)
		}
	}

	// =============================================================================
	// 10. CUSTOM TYPES (TYPE ALIASES)
	// =============================================================================
	fmt.Println("\n=== CUSTOM TYPES ===")

	// Type aliases
	type UserID int
	type Status string
	type Priority int

	userIDCache := NewTypedCache[UserID](cache)
	statusCache := NewTypedCache[Status](cache)
	priorityCache := NewTypedCache[Priority](cache)

	<-userIDCache.Set(ctx, "user:id", UserID(12345), time.Hour)
	<-statusCache.Set(ctx, "user:status", Status("active"), time.Hour)
	<-priorityCache.Set(ctx, "task:priority", Priority(1), time.Hour)

	userIDResult := <-userIDCache.Get(ctx, "user:id")
	if userIDResult.Found {
		fmt.Printf("✓ Retrieved UserID: %d\n", userIDResult.Value)
	}

	statusResult := <-statusCache.Get(ctx, "user:status")
	if statusResult.Found {
		fmt.Printf("✓ Retrieved Status: %s\n", statusResult.Value)
	}

	priorityResult := <-priorityCache.Get(ctx, "task:priority")
	if priorityResult.Found {
		fmt.Printf("✓ Retrieved Priority: %d\n", priorityResult.Value)
	}
}

// ExampleBatchOperations demonstrates batch operations with different types
func ExampleBatchOperations() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== BATCH OPERATIONS ===")

	// MSet with mixed types using base cache
	items := map[string]any{
		"config:app_name": "MyApp",
		"config:version":  "1.0.0",
		"config:debug":    true,
		"config:port":     8080,
		"config:timeout":  30.5,
		"config:features": []string{"auth", "logging", "metrics"},
		"config:limits":   map[string]int{"max_users": 1000, "max_requests": 10000},
	}

	msetResult := <-cache.MSet(ctx, items, time.Hour)
	if msetResult.Error != nil {
		fmt.Printf("Error in MSet: %v\n", msetResult.Error)
		return
	}
	fmt.Println("✓ MSet completed successfully")

	// MGet to retrieve multiple values
	keys := []string{"config:app_name", "config:version", "config:debug", "config:port", "config:timeout"}
	mgetResult := <-cache.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		fmt.Printf("Error in MGet: %v\n", mgetResult.Error)
		return
	}

	fmt.Println("✓ MGet results:")
	for key, value := range mgetResult.Values {
		fmt.Printf("  %s: %v (type: %T)\n", key, value, value)
	}
}

// ExampleUtilityOperations demonstrates utility operations
func ExampleUtilityOperations() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== UTILITY OPERATIONS ===")

	// Set some test data
	stringCache := NewTypedCache[string](cache)
	<-stringCache.Set(ctx, "test:exists", "test value", time.Hour)
	<-stringCache.Set(ctx, "test:ttl", "ttl test", 5*time.Second)

	// Exists operation
	existsResult := <-cache.Exists(ctx, "test:exists")
	if existsResult.Found {
		fmt.Println("✓ Key 'test:exists' exists")
	}

	// TTL operation
	ttlResult := <-cache.TTL(ctx, "test:ttl")
	if ttlResult.Error == nil {
		fmt.Printf("✓ TTL for 'test:ttl': %v\n", ttlResult.TTL)
	}

	// IncrBy operation
	incrResult := <-cache.IncrBy(ctx, "counter", 5, time.Hour)
	if incrResult.Error == nil {
		fmt.Printf("✓ Incremented counter by 5, new value: %d\n", incrResult.Int)
	}

	// Del operation
	delResult := <-cache.Del(ctx, "test:exists", "test:ttl")
	if delResult.Error == nil {
		fmt.Printf("✓ Deleted %d keys\n", delResult.Count)
	}
}

// ExampleWithContext demonstrates context usage
func ExampleWithContext() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	fmt.Println("\n=== CONTEXT USAGE ===")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use the cache with the context
	cacheWithCtx := cache.WithContext(ctx)

	// Create typed caches from the context-aware cache
	stringCache := NewTypedCache[string](cacheWithCtx)

	// This operation will respect the context timeout
	result := <-stringCache.Set(ctx, "context:test", "context value", time.Minute)
	if result.Error != nil {
		fmt.Printf("Error with context: %v\n", result.Error)
		return
	}

	fmt.Println("✓ Successfully set value with context")

	// Get the value back
	getResult := <-stringCache.Get(ctx, "context:test")
	if getResult.Found {
		fmt.Printf("✓ Retrieved value with context: %s\n", getResult.Value)
	}
}

// ExampleErrorHandling demonstrates error handling patterns
func ExampleErrorHandling() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== ERROR HANDLING ===")

	stringCache := NewTypedCache[string](cache)

	// Try to get a non-existent key
	getResult := <-stringCache.Get(ctx, "non:existent:key")
	if !getResult.Found {
		fmt.Println("✓ Correctly handled non-existent key")
	}

	// Try to set with empty key (should error)
	setResult := <-stringCache.Set(ctx, "", "value", time.Hour)
	if setResult.Error != nil {
		fmt.Printf("✓ Correctly caught empty key error: %v\n", setResult.Error)
	}

	// Try to get with invalid key format
	invalidResult := <-stringCache.Get(ctx, "")
	if invalidResult.Error != nil {
		fmt.Printf("✓ Correctly handled invalid key: %v\n", invalidResult.Error)
	}
}

// ExampleRealWorldScenarios demonstrates real-world usage patterns
func ExampleRealWorldScenarios() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== REAL-WORLD SCENARIOS ===")

	// Scenario 1: User Session Management
	fmt.Println("\n--- User Session Management ---")
	type Session struct {
		UserID    int       `json:"user_id"`
		Username  string    `json:"username"`
		LoginTime time.Time `json:"login_time"`
		ExpiresAt time.Time `json:"expires_at"`
		Roles     []string  `json:"roles"`
	}

	sessionCache := NewTypedCache[Session](cache)
	session := Session{
		UserID:    123,
		Username:  "john_doe",
		LoginTime: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Roles:     []string{"user", "admin"},
	}

	// Store session with 24-hour TTL
	sessionResult := <-sessionCache.Set(ctx, "session:abc123", session, 24*time.Hour)
	if sessionResult.Error == nil {
		fmt.Println("✓ User session stored successfully")
	}

	// Retrieve session
	retrievedSession := <-sessionCache.Get(ctx, "session:abc123")
	if retrievedSession.Found {
		fmt.Printf("✓ Retrieved session for user: %s\n", retrievedSession.Value.Username)
	}

	// Scenario 2: Application Configuration
	fmt.Println("\n--- Application Configuration ---")
	type AppConfig struct {
		DatabaseURL string            `json:"database_url"`
		APIVersion  string            `json:"api_version"`
		Features    map[string]bool   `json:"features"`
		Limits      map[string]int    `json:"limits"`
		Secrets     map[string]string `json:"secrets"`
	}

	configCache := NewTypedCache[AppConfig](cache)
	config := AppConfig{
		DatabaseURL: "postgres://localhost:5432/myapp",
		APIVersion:  "v1.2.0",
		Features: map[string]bool{
			"auth":    true,
			"logging": true,
			"metrics": false,
			"debug":   false,
		},
		Limits: map[string]int{
			"max_connections": 100,
			"rate_limit":      1000,
			"cache_ttl":       3600,
		},
		Secrets: map[string]string{
			"jwt_secret": "super-secret-key",
			"api_key":    "api-key-123",
		},
	}

	<-configCache.Set(ctx, "app:config", config, time.Hour)
	fmt.Println("✓ Application configuration cached")

	// Scenario 3: E-commerce Product Catalog
	fmt.Println("\n--- E-commerce Product Catalog ---")
	type Product struct {
		ID          int               `json:"id"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Price       float64           `json:"price"`
		Currency    string            `json:"currency"`
		Category    string            `json:"category"`
		Tags        []string          `json:"tags"`
		Attributes  map[string]string `json:"attributes"`
		InStock     bool              `json:"in_stock"`
		StockCount  int               `json:"stock_count"`
		CreatedAt   time.Time         `json:"created_at"`
		UpdatedAt   time.Time         `json:"updated_at"`
	}

	productCache := NewTypedCache[Product](cache)
	product := Product{
		ID:          1001,
		Name:        "Wireless Bluetooth Headphones",
		Description: "High-quality wireless headphones with noise cancellation",
		Price:       199.99,
		Currency:    "USD",
		Category:    "Electronics",
		Tags:        []string{"wireless", "bluetooth", "noise-cancellation", "audio"},
		Attributes: map[string]string{
			"brand":        "TechSound",
			"model":        "TS-WH1000",
			"color":        "Black",
			"battery":      "30 hours",
			"connectivity": "Bluetooth 5.0",
		},
		InStock:    true,
		StockCount: 50,
		CreatedAt:  time.Now().Add(-7 * 24 * time.Hour),
		UpdatedAt:  time.Now(),
	}

	<-productCache.Set(ctx, "product:1001", product, 2*time.Hour)
	fmt.Println("✓ Product catalog item cached")

	// Scenario 4: Analytics and Metrics
	fmt.Println("\n--- Analytics and Metrics ---")
	type Metrics struct {
		Timestamp   time.Time        `json:"timestamp"`
		PageViews   int64            `json:"page_views"`
		UniqueUsers int64            `json:"unique_users"`
		BounceRate  float64          `json:"bounce_rate"`
		TopPages    []string         `json:"top_pages"`
		UserAgents  map[string]int64 `json:"user_agents"`
		Countries   map[string]int64 `json:"countries"`
	}

	metricsCache := NewTypedCache[Metrics](cache)
	metrics := Metrics{
		Timestamp:   time.Now(),
		PageViews:   15420,
		UniqueUsers: 8930,
		BounceRate:  0.35,
		TopPages:    []string{"/home", "/products", "/about", "/contact"},
		UserAgents: map[string]int64{
			"Chrome":  8500,
			"Firefox": 3200,
			"Safari":  2100,
			"Edge":    1620,
		},
		Countries: map[string]int64{
			"US": 4500,
			"UK": 2100,
			"CA": 1800,
			"DE": 1200,
			"FR": 980,
		},
	}

	<-metricsCache.Set(ctx, "analytics:daily:2024-01-15", metrics, 7*24*time.Hour)
	fmt.Println("✓ Daily analytics metrics cached")

	// Scenario 5: Cache-Aside Pattern with Type Safety
	fmt.Println("\n--- Cache-Aside Pattern ---")

	// Define User type and create userCache for this function
	type User struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Email  string  `json:"email"`
		Active bool    `json:"active"`
		Score  float64 `json:"score"`
	}
	userCache := NewTypedCache[User](cache)

	// Simulate a function that fetches user data from database
	fetchUserFromDB := func(userID int) (User, error) {
		// Simulate database call
		time.Sleep(10 * time.Millisecond)
		return User{
			ID:     userID,
			Name:   fmt.Sprintf("User %d", userID),
			Email:  fmt.Sprintf("user%d@example.com", userID),
			Active: true,
			Score:  float64(userID * 10),
		}, nil
	}

	// Cache-aside pattern implementation
	getUserWithCache := func(userID int) (User, error) {
		// Try to get from cache first
		cacheKey := fmt.Sprintf("user:%d", userID)
		cachedResult := <-userCache.Get(ctx, cacheKey)

		if cachedResult.Found && cachedResult.Error == nil {
			fmt.Printf("✓ Cache hit for user %d\n", userID)
			return cachedResult.Value, nil
		}

		// Cache miss - fetch from database
		fmt.Printf("✗ Cache miss for user %d, fetching from DB\n", userID)
		user, err := fetchUserFromDB(userID)
		if err != nil {
			return User{}, err
		}

		// Store in cache for next time
		<-userCache.Set(ctx, cacheKey, user, time.Hour)
		fmt.Printf("✓ Stored user %d in cache\n", userID)

		return user, nil
	}

	// Test cache-aside pattern
	user1, _ := getUserWithCache(1)
	fmt.Printf("✓ Retrieved user: %s\n", user1.Name)

	// Second call should hit cache
	user1Again, _ := getUserWithCache(1)
	fmt.Printf("✓ Retrieved user again: %s\n", user1Again.Name)
}

// ExamplePerformance demonstrates performance considerations
func ExamplePerformance() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== PERFORMANCE CONSIDERATIONS ===")

	// Performance test: Batch operations vs individual operations
	fmt.Println("\n--- Batch vs Individual Operations ---")

	// Individual operations
	start := time.Now()
	stringCache := NewTypedCache[string](cache)
	for i := 0; i < 100; i++ {
		<-stringCache.Set(ctx, fmt.Sprintf("individual:%d", i), fmt.Sprintf("value%d", i), time.Hour)
	}
	individualTime := time.Since(start)
	fmt.Printf("✓ 100 individual Set operations: %v\n", individualTime)

	// Batch operations
	start = time.Now()
	batchItems := make(map[string]any)
	for i := 0; i < 100; i++ {
		batchItems[fmt.Sprintf("batch:%d", i)] = fmt.Sprintf("value%d", i)
	}
	<-cache.MSet(ctx, batchItems, time.Hour)
	batchTime := time.Since(start)
	fmt.Printf("✓ 100 batch MSet operations: %v\n", batchTime)
	fmt.Printf("✓ Batch operations are %.2fx faster\n", float64(individualTime)/float64(batchTime))

	// Memory usage consideration: Large objects
	fmt.Println("\n--- Large Object Handling ---")

	type LargeData struct {
		ID       int               `json:"id"`
		Data     []byte            `json:"data"`
		Metadata map[string]string `json:"metadata"`
	}

	largeCache := NewTypedCache[LargeData](cache)

	// Create a large object (1MB of data)
	largeData := LargeData{
		ID:   1,
		Data: make([]byte, 1024*1024), // 1MB
		Metadata: map[string]string{
			"size":    "1MB",
			"type":    "test_data",
			"created": time.Now().Format(time.RFC3339),
		},
	}

	// Fill with some data
	for i := range largeData.Data {
		largeData.Data[i] = byte(i % 256)
	}

	start = time.Now()
	<-largeCache.Set(ctx, "large:object", largeData, time.Hour)
	largeSetTime := time.Since(start)
	fmt.Printf("✓ Stored 1MB object in: %v\n", largeSetTime)

	start = time.Now()
	largeResult := <-largeCache.Get(ctx, "large:object")
	largeGetTime := time.Since(start)
	if largeResult.Found {
		fmt.Printf("✓ Retrieved 1MB object in: %v\n", largeGetTime)
		fmt.Printf("✓ Object size: %d bytes\n", len(largeResult.Value.Data))
	}
}

// ExampleAdvancedPatterns demonstrates advanced caching patterns
func ExampleAdvancedPatterns() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== ADVANCED CACHING PATTERNS ===")

	// Create typed cache for string operations
	stringCache := NewTypedCache[string](cache)

	// Define User type for this function
	type User struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Email  string  `json:"email"`
		Active bool    `json:"active"`
		Score  float64 `json:"score"`
	}

	// Pattern 1: Write-Through Cache
	fmt.Println("\n--- Write-Through Cache Pattern ---")

	type WriteThroughCache[T any] struct {
		cache     Cache
		writeFunc func(string, T) error
	}

	// Simulate database write
	writeToDB := func(key string, value string) error {
		fmt.Printf("✓ Writing to database: %s = %s\n", key, value)
		return nil
	}

	wtCache := &WriteThroughCache[string]{
		cache:     cache,
		writeFunc: writeToDB,
	}

	// Write-through implementation
	writeThrough := func(key string, value string, ttl time.Duration) error {
		// Write to cache
		result := <-cache.Set(ctx, key, value, ttl)
		if result.Error != nil {
			return result.Error
		}

		// Write to database
		return wtCache.writeFunc(key, value)
	}

	writeThrough("write:through:test", "write-through value", time.Hour)
	fmt.Println("✓ Write-through pattern completed")

	// Pattern 2: Cache Invalidation
	fmt.Println("\n--- Cache Invalidation Pattern ---")

	// Set up some related data
	userCache := NewTypedCache[User](cache)
	settingsCache := NewTypedCache[map[string]string](cache)
	user := User{ID: 1, Name: "John", Email: "john@example.com", Active: true, Score: 95.5}
	<-userCache.Set(ctx, "user:1", user, time.Hour)
	<-userCache.Set(ctx, "user:1:profile", user, time.Hour)
	<-settingsCache.Set(ctx, "user:1:settings", map[string]string{"theme": "dark"}, time.Hour)

	// Invalidate all user-related cache entries
	invalidateUserCache := func(userID int) {
		keys := []string{
			fmt.Sprintf("user:%d", userID),
			fmt.Sprintf("user:%d:profile", userID),
			fmt.Sprintf("user:%d:settings", userID),
		}
		delResult := <-cache.Del(ctx, keys...)
		if delResult.Error == nil {
			fmt.Printf("✓ Invalidated %d cache entries for user %d\n", delResult.Count, userID)
		}
	}

	invalidateUserCache(1)
	fmt.Println("✓ Cache invalidation pattern completed")

	// Pattern 3: Cache-Aside Pattern
	fmt.Println("\n--- Cache-Aside Pattern ---")

	cacheAside := func(key string) (string, error) {
		// Try to get from cache first
		result := <-stringCache.Get(ctx, key)
		if result.Found {
			fmt.Printf("✓ Cache hit for key: %s\n", key)
			return result.Value, nil
		}

		// Cache miss - fetch from database
		fmt.Printf("✗ Cache miss for key: %s, fetching from database\n", key)
		dbValue := "data from database"

		// Store in cache for next time
		<-stringCache.Set(ctx, key, dbValue, time.Hour)
		return dbValue, nil
	}

	value, err := cacheAside("cache:aside:test")
	if err == nil {
		fmt.Printf("✓ Cache-aside pattern result: %s\n", value)
	}

	// Pattern 4: Rate Limiting with Cache
	fmt.Println("\n--- Rate Limiting Pattern ---")

	rateLimit := func(userID int, limit int, window time.Duration) bool {
		key := fmt.Sprintf("rate:limit:%d", userID)

		// Get current count
		result := <-cache.IncrBy(ctx, key, 1, window)
		if result.Error != nil {
			return false
		}

		if result.Int == 1 {
			// First request in window
			fmt.Printf("✓ First request for user %d in window\n", userID)
			return true
		}

		if result.Int > int64(limit) {
			fmt.Printf("✗ Rate limit exceeded for user %d: %d/%d\n", userID, result.Int, limit)
			return false
		}

		fmt.Printf("✓ Request allowed for user %d: %d/%d\n", userID, result.Int, limit)
		return true
	}

	// Test rate limiting
	allowed1 := rateLimit(123, 5, time.Minute)
	allowed2 := rateLimit(123, 5, time.Minute)
	allowed3 := rateLimit(123, 5, time.Minute)

	fmt.Printf("✓ Rate limiting results: %t, %t, %t\n", allowed1, allowed2, allowed3)
}

// ExampleAdvancedTypes demonstrates advanced Go types
func ExampleAdvancedTypes() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== ADVANCED TYPES ===")

	// Channel types
	fmt.Println("\n--- Channel Types ---")
	ch := make(chan string, 1)
	ch <- "channel message"
	close(ch)

	// Note: Channels can't be serialized directly, but we can store their state
	channelStateCache := NewTypedCache[bool](cache)
	<-channelStateCache.Set(ctx, "channel:active", true, time.Hour)

	// Function types (stored as metadata)
	fmt.Println("\n--- Function Metadata ---")
	funcCache := NewTypedCache[map[string]any](cache)
	funcMetadata := map[string]any{
		"name":    "processData",
		"version": "1.0",
		"enabled": true,
		"timeout": 30,
	}
	<-funcCache.Set(ctx, "function:metadata", funcMetadata, time.Hour)

	// Complex nested types with interfaces
	fmt.Println("\n--- Complex Nested Types ---")

	// Note: Interface types need special handling in serialization
	configCache := NewTypedCache[map[string]any](cache)
	configData := map[string]any{
		"name":        "MyApp",
		"version":     "2.0.0",
		"environment": "production",
		"settings": map[string]any{
			"timeout": 30,
			"debug":   false,
			"host":    "localhost",
		},
	}
	<-configCache.Set(ctx, "app:config", configData, time.Hour)

	configResult := <-configCache.Get(ctx, "app:config")
	if configResult.Found {
		fmt.Printf("✓ Retrieved complex config: %+v\n", configResult.Value)
	}

	// Time types
	fmt.Println("\n--- Time Types ---")
	timeCache := NewTypedCache[time.Time](cache)
	now := time.Now()
	<-timeCache.Set(ctx, "system:startup", now, time.Hour)

	timeResult := <-timeCache.Get(ctx, "system:startup")
	if timeResult.Found {
		fmt.Printf("✓ Retrieved time: %v\n", timeResult.Value)
	}

	// Duration types
	durationCache := NewTypedCache[time.Duration](cache)
	<-durationCache.Set(ctx, "config:timeout", 30*time.Second, time.Hour)

	durationResult := <-durationCache.Get(ctx, "config:timeout")
	if durationResult.Found {
		fmt.Printf("✓ Retrieved duration: %v\n", durationResult.Value)
	}
}

// ExamplePerformancePatterns demonstrates performance optimization patterns
func ExamplePerformancePatterns() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== PERFORMANCE PATTERNS ===")

	// Pattern 1: Batch Operations for Performance
	fmt.Println("\n--- Batch Operations ---")

	// Instead of multiple individual sets, use MSet
	items := make(map[string]any)
	for i := 0; i < 100; i++ {
		items[fmt.Sprintf("batch:item:%d", i)] = fmt.Sprintf("value_%d", i)
	}

	start := time.Now()
	msetResult := <-cache.MSet(ctx, items, time.Hour)
	msetDuration := time.Since(start)

	if msetResult.Error == nil {
		fmt.Printf("✓ MSet 100 items in %v\n", msetDuration)
	}

	// Batch retrieval
	keys := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		keys = append(keys, fmt.Sprintf("batch:item:%d", i))
	}

	start = time.Now()
	mgetResult := <-cache.MGet(ctx, keys...)
	mgetDuration := time.Since(start)

	if mgetResult.Error == nil {
		fmt.Printf("✓ MGet 100 items in %v (retrieved %d items)\n", mgetDuration, len(mgetResult.Values))
	}

	// Pattern 2: Pipeline-like Operations
	fmt.Println("\n--- Pipeline-like Operations ---")

	// Simulate multiple operations that can be batched
	operations := []struct {
		key   string
		value any
		ttl   time.Duration
	}{
		{"pipeline:1", "value1", time.Hour},
		{"pipeline:2", 42, time.Hour},
		{"pipeline:3", true, time.Hour},
		{"pipeline:4", []string{"a", "b", "c"}, time.Hour},
	}

	pipelineItems := make(map[string]any)
	for _, op := range operations {
		pipelineItems[op.key] = op.value
	}

	start = time.Now()
	pipelineResult := <-cache.MSet(ctx, pipelineItems, time.Hour)
	pipelineDuration := time.Since(start)

	if pipelineResult.Error == nil {
		fmt.Printf("✓ Pipeline set %d operations in %v\n", len(operations), pipelineDuration)
	}

	// Pattern 3: Lazy Loading with Background Refresh
	fmt.Println("\n--- Lazy Loading with Background Refresh ---")

	type ExpensiveData struct {
		ID       int       `json:"id"`
		Data     string    `json:"data"`
		Computed time.Time `json:"computed"`
	}

	expensiveCache := NewTypedCache[ExpensiveData](cache)

	// Simulate expensive computation
	computeExpensiveData := func(id int) ExpensiveData {
		// Simulate computation time
		time.Sleep(10 * time.Millisecond)
		return ExpensiveData{
			ID:       id,
			Data:     fmt.Sprintf("expensive_data_%d", id),
			Computed: time.Now(),
		}
	}

	// Lazy load with background refresh
	lazyLoad := func(id int) ExpensiveData {
		key := fmt.Sprintf("expensive:%d", id)

		// Try to get from cache
		result := <-expensiveCache.Get(ctx, key)
		if result.Found {
			// Check if data is stale (older than 1 minute)
			if time.Since(result.Value.Computed) < time.Minute {
				fmt.Printf("✓ Cache hit for expensive data %d\n", id)
				return result.Value
			}
		}

		// Cache miss or stale data - compute new data
		fmt.Printf("✗ Computing expensive data for %d\n", id)
		newData := computeExpensiveData(id)

		// Store in cache
		<-expensiveCache.Set(ctx, key, newData, time.Hour)
		return newData
	}

	// Test lazy loading
	data1 := lazyLoad(1)
	data2 := lazyLoad(1) // Should be cache hit
	data3 := lazyLoad(2)

	fmt.Printf("✓ Lazy loading results: %+v, %+v, %+v\n", data1.ID, data2.ID, data3.ID)
}

// ExampleConcurrencyPatterns demonstrates concurrency patterns
func ExampleConcurrencyPatterns() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== CONCURRENCY PATTERNS ===")

	// Pattern 1: Concurrent Reads
	fmt.Println("\n--- Concurrent Reads ---")

	stringCache := NewTypedCache[string](cache)

	// Set up test data
	<-stringCache.Set(ctx, "concurrent:test", "test_value", time.Hour)

	// Concurrent reads
	numGoroutines := 10
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result := <-stringCache.Get(ctx, "concurrent:test")
			if result.Found {
				results <- fmt.Sprintf("Goroutine %d: %s", id, result.Value)
			} else {
				results <- fmt.Sprintf("Goroutine %d: not found", id)
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		fmt.Printf("✓ %s\n", result)
	}

	// Pattern 2: Read-Through with Mutex-like Behavior
	fmt.Println("\n--- Read-Through with Mutex-like Behavior ---")

	// Simulate concurrent access to the same key
	concurrentKey := "concurrent:key"
	concurrentResults := make(chan string, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			// All goroutines try to get the same key
			result := <-stringCache.Get(ctx, concurrentKey)
			if result.Found {
				concurrentResults <- fmt.Sprintf("Goroutine %d: Found existing value: %s", id, result.Value)
			} else {
				// Simulate expensive operation
				time.Sleep(100 * time.Millisecond)
				value := fmt.Sprintf("expensive-value-%d", id)
				<-stringCache.Set(ctx, concurrentKey, value, time.Hour)
				concurrentResults <- fmt.Sprintf("Goroutine %d: Set new value: %s", id, value)
			}
		}(i)
	}

	// Collect concurrent results
	for i := 0; i < 5; i++ {
		result := <-concurrentResults
		fmt.Printf("✓ %s\n", result)
	}
}

// ExamplePerformanceBenchmarks demonstrates performance testing patterns
func ExamplePerformanceBenchmarks() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== PERFORMANCE BENCHMARKS ===")

	// Benchmark 1: Single Type Operations
	fmt.Println("\n--- Single Type Operations Benchmark ---")

	stringCache := NewTypedCache[string](cache)

	// Benchmark SET operations
	start := time.Now()
	numOps := 1000
	for i := 0; i < numOps; i++ {
		<-stringCache.Set(ctx, fmt.Sprintf("bench:set:%d", i), fmt.Sprintf("value-%d", i), time.Hour)
	}
	setDuration := time.Since(start)
	fmt.Printf("✓ SET %d operations in %v (%.2f ops/sec)\n", numOps, setDuration, float64(numOps)/setDuration.Seconds())

	// Benchmark GET operations
	start = time.Now()
	for i := 0; i < numOps; i++ {
		<-stringCache.Get(ctx, fmt.Sprintf("bench:set:%d", i))
	}
	getDuration := time.Since(start)
	fmt.Printf("✓ GET %d operations in %v (%.2f ops/sec)\n", numOps, getDuration, float64(numOps)/getDuration.Seconds())

	// Benchmark 2: Mixed Type Operations
	fmt.Println("\n--- Mixed Type Operations Benchmark ---")

	intCache := NewTypedCache[int](cache)
	boolCache := NewTypedCache[bool](cache)

	start = time.Now()
	for i := 0; i < 100; i++ {
		<-stringCache.Set(ctx, fmt.Sprintf("mixed:string:%d", i), fmt.Sprintf("string-%d", i), time.Hour)
		<-intCache.Set(ctx, fmt.Sprintf("mixed:int:%d", i), i, time.Hour)
		<-boolCache.Set(ctx, fmt.Sprintf("mixed:bool:%d", i), i%2 == 0, time.Hour)
	}
	mixedDuration := time.Since(start)
	fmt.Printf("✓ Mixed types %d operations in %v (%.2f ops/sec)\n", 300, mixedDuration, float64(300)/mixedDuration.Seconds())

	// Benchmark 3: Batch Operations
	fmt.Println("\n--- Batch Operations Benchmark ---")

	// Prepare batch data
	batchItems := make(map[string]any)
	for i := 0; i < 100; i++ {
		batchItems[fmt.Sprintf("batch:%d", i)] = fmt.Sprintf("batch-value-%d", i)
	}

	start = time.Now()
	<-cache.MSet(ctx, batchItems, time.Hour)
	batchSetDuration := time.Since(start)
	fmt.Printf("✓ MSet 100 items in %v (%.2f ops/sec)\n", batchSetDuration, float64(100)/batchSetDuration.Seconds())

	// Prepare batch keys
	batchKeys := make([]string, 100)
	for i := 0; i < 100; i++ {
		batchKeys[i] = fmt.Sprintf("batch:%d", i)
	}

	start = time.Now()
	<-cache.MGet(ctx, batchKeys...)
	batchGetDuration := time.Since(start)
	fmt.Printf("✓ MGet 100 items in %v (%.2f ops/sec)\n", batchGetDuration, float64(100)/batchGetDuration.Seconds())
}

// ExampleAdvancedPatterns demonstrates advanced caching patterns
func ExampleAdvancedPatterns1() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== ADVANCED PATTERNS ===")

	// Pattern 1: Cache-Aside with Fallback
	fmt.Println("\n--- Cache-Aside with Fallback Pattern ---")

	type Product struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Price       float64 `json:"price"`
		Description string  `json:"description"`
		InStock     bool    `json:"in_stock"`
	}

	productCache := NewTypedCache[Product](cache)

	// Simulate database
	productDB := map[int]Product{
		1: {ID: 1, Name: "Laptop", Price: 999.99, Description: "High-performance laptop", InStock: true},
		2: {ID: 2, Name: "Mouse", Price: 29.99, Description: "Wireless mouse", InStock: true},
		3: {ID: 3, Name: "Keyboard", Price: 79.99, Description: "Mechanical keyboard", InStock: false},
	}

	// Cache-aside function
	getProduct := func(id int) (Product, error) {
		// Try cache first
		result := <-productCache.Get(ctx, fmt.Sprintf("product:%d", id))
		if result.Found {
			fmt.Printf("✓ Cache hit for product %d\n", id)
			return result.Value, nil
		}

		// Cache miss - get from database
		fmt.Printf("✓ Cache miss for product %d, fetching from DB\n", id)
		product, exists := productDB[id]
		if !exists {
			return Product{}, fmt.Errorf("product %d not found", id)
		}

		// Store in cache for next time
		<-productCache.Set(ctx, fmt.Sprintf("product:%d", id), product, time.Hour)
		return product, nil
	}

	// Test cache-aside pattern
	product1, _ := getProduct(1)
	fmt.Printf("✓ Retrieved product: %+v\n", product1)

	// Second call should hit cache
	product1Again, _ := getProduct(1)
	fmt.Printf("✓ Retrieved product again: %+v\n", product1Again)

	// Pattern 2: Write-Behind (Async Write)
	fmt.Println("\n--- Write-Behind Pattern ---")

	type UserProfile struct {
		ID       int       `json:"id"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
		LastSeen time.Time `json:"last_seen"`
	}

	userProfileCache := NewTypedCache[UserProfile](cache)

	// Simulate database write
	writeToDB := func(profile UserProfile) error {
		// Simulate database write delay
		time.Sleep(50 * time.Millisecond)
		fmt.Printf("✓ Written to database: %s\n", profile.Username)
		return nil
	}

	// Write-behind function
	updateUserProfile := func(profile UserProfile) error {
		// Update cache immediately
		result := <-userProfileCache.Set(ctx, fmt.Sprintf("profile:%d", profile.ID), profile, time.Hour)
		if result.Error != nil {
			return result.Error
		}

		// Write to database asynchronously
		go func() {
			if err := writeToDB(profile); err != nil {
				fmt.Printf("Error writing to database: %v\n", err)
			}
		}()

		return nil
	}

	// Test write-behind pattern
	profile := UserProfile{ID: 1, Username: "john_doe", Email: "john@example.com", LastSeen: time.Now()}
	updateUserProfile(profile)
	fmt.Println("✓ Write-behind pattern completed")

	// Pattern 3: Cache Warming
	fmt.Println("\n--- Cache Warming Pattern ---")

	// Warm up cache with frequently accessed data
	warmCache := func() {
		fmt.Println("✓ Warming up cache...")

		// Pre-load popular products
		for id, product := range productDB {
			<-productCache.Set(ctx, fmt.Sprintf("product:%d", id), product, time.Hour)
		}

		// Pre-load user profiles
		profiles := []UserProfile{
			{ID: 1, Username: "john_doe", Email: "john@example.com", LastSeen: time.Now()},
			{ID: 2, Username: "jane_smith", Email: "jane@example.com", LastSeen: time.Now()},
			{ID: 3, Username: "bob_wilson", Email: "bob@example.com", LastSeen: time.Now()},
		}

		for _, profile := range profiles {
			<-userProfileCache.Set(ctx, fmt.Sprintf("profile:%d", profile.ID), profile, time.Hour)
		}

		fmt.Println("✓ Cache warming completed")
	}

	warmCache()

	// Pattern 4: Cache Versioning
	fmt.Println("\n--- Cache Versioning Pattern ---")

	type CacheVersion struct {
		Version int       `json:"version"`
		Data    any       `json:"data"`
		Created time.Time `json:"created"`
	}

	versionedCache := NewTypedCache[CacheVersion](cache)

	// Store versioned data
	versionedData := CacheVersion{
		Version: 1,
		Data:    "initial data",
		Created: time.Now(),
	}

	<-versionedCache.Set(ctx, "versioned:key", versionedData, time.Hour)

	// Update with new version
	versionedData.Version = 2
	versionedData.Data = "updated data"
	versionedData.Created = time.Now()

	<-versionedCache.Set(ctx, "versioned:key", versionedData, time.Hour)

	// Retrieve and check version
	result := <-versionedCache.Get(ctx, "versioned:key")
	if result.Found {
		fmt.Printf("✓ Retrieved versioned data: version %d, data: %v\n", result.Value.Version, result.Value.Data)
	}
}

// ExampleBaseCacheUsage demonstrates using the cache without typed wrappers
func ExampleBaseCacheUsage() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create base cache instance (no typed wrapper)
	cache := NewRedisCache(client, nil, nil, nil)
	defer cache.Close()

	ctx := context.Background()

	fmt.Println("\n=== BASE CACHE USAGE (NO TYPED WRAPPER) ===")

	// Example 1: Basic operations with any type
	fmt.Println("\n--- Basic Operations ---")

	// Set different types directly
	<-cache.Set(ctx, "string:key", "Hello World", time.Hour)
	<-cache.Set(ctx, "int:key", 42, time.Hour)
	<-cache.Set(ctx, "bool:key", true, time.Hour)
	<-cache.Set(ctx, "float:key", 3.14, time.Hour)
	<-cache.Set(ctx, "slice:key", []string{"a", "b", "c"}, time.Hour)
	<-cache.Set(ctx, "map:key", map[string]int{"x": 1, "y": 2}, time.Hour)

	fmt.Println("✓ Stored various types directly in base cache")

	// Get values back (returns as any)
	stringResult := <-cache.Get(ctx, "string:key")
	if stringResult.Found {
		fmt.Printf("✓ Retrieved string: %v (type: %T)\n", stringResult.Value, stringResult.Value)
	}

	intResult := <-cache.Get(ctx, "int:key")
	if intResult.Found {
		fmt.Printf("✓ Retrieved int: %v (type: %T)\n", intResult.Value, intResult.Value)
	}

	// Example 2: Type assertion for type safety
	fmt.Println("\n--- Type Assertion ---")

	getString := func(key string) (string, error) {
		result := <-cache.Get(ctx, key)
		if !result.Found {
			return "", fmt.Errorf("key not found")
		}
		if result.Error != nil {
			return "", result.Error
		}

		// Type assert to string
		if str, ok := result.Value.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("value is not a string")
	}

	getInt := func(key string) (int, error) {
		result := <-cache.Get(ctx, key)
		if !result.Found {
			return 0, fmt.Errorf("key not found")
		}
		if result.Error != nil {
			return 0, result.Error
		}

		// Type assert to int
		if num, ok := result.Value.(int); ok {
			return num, nil
		}
		return 0, fmt.Errorf("value is not an int")
	}

	str, err := getString("string:key")
	if err == nil {
		fmt.Printf("✓ Type-safe string retrieval: %s\n", str)
	}

	num, err := getInt("int:key")
	if err == nil {
		fmt.Printf("✓ Type-safe int retrieval: %d\n", num)
	}

	// Example 3: Batch operations with mixed types
	fmt.Println("\n--- Batch Operations ---")

	mixedItems := map[string]any{
		"batch:string": "batch string value",
		"batch:int":    123,
		"batch:bool":   false,
		"batch:float":  99.99,
		"batch:slice":  []int{1, 2, 3, 4, 5},
		"batch:map":    map[string]string{"key1": "value1", "key2": "value2"},
	}

	msetResult := <-cache.MSet(ctx, mixedItems, time.Hour)
	if msetResult.Error == nil {
		fmt.Println("✓ MSet completed with mixed types")
	}

	// Retrieve batch
	keys := []string{"batch:string", "batch:int", "batch:bool", "batch:float"}
	mgetResult := <-cache.MGet(ctx, keys...)
	if mgetResult.Error == nil {
		fmt.Println("✓ MGet results:")
		for key, value := range mgetResult.Values {
			fmt.Printf("  %s: %v (type: %T)\n", key, value, value)
		}
	}

	// Example 4: Utility operations
	fmt.Println("\n--- Utility Operations ---")

	// Check if key exists
	existsResult := <-cache.Exists(ctx, "string:key")
	if existsResult.Found {
		fmt.Println("✓ Key 'string:key' exists")
	}

	// Get TTL
	ttlResult := <-cache.TTL(ctx, "string:key")
	if ttlResult.Error == nil {
		fmt.Printf("✓ TTL for 'string:key': %v\n", ttlResult.TTL)
	}

	// Increment counter
	incrResult := <-cache.IncrBy(ctx, "counter", 5, time.Hour)
	if incrResult.Error == nil {
		fmt.Printf("✓ Incremented counter by 5, new value: %d\n", incrResult.Int)
	}

	// Delete keys
	delResult := <-cache.Del(ctx, "string:key", "int:key", "bool:key")
	if delResult.Error == nil {
		fmt.Printf("✓ Deleted %d keys\n", delResult.Count)
	}

	// Example 5: Error handling
	fmt.Println("\n--- Error Handling ---")

	// Try to get non-existent key
	notFoundResult := <-cache.Get(ctx, "non:existent:key")
	if !notFoundResult.Found {
		fmt.Println("✓ Correctly handled non-existent key")
	}

	// Try to set with empty key
	emptyKeyResult := <-cache.Set(ctx, "", "value", time.Hour)
	if emptyKeyResult.Error != nil {
		fmt.Printf("✓ Correctly caught empty key error: %v\n", emptyKeyResult.Error)
	}

	// Example 6: Context usage
	fmt.Println("\n--- Context Usage ---")

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use cache with context
	cacheWithCtx := cache.WithContext(ctxWithTimeout)

	// Set value with context
	ctxResult := <-cacheWithCtx.Set(ctxWithTimeout, "context:key", "context value", time.Hour)
	if ctxResult.Error == nil {
		fmt.Println("✓ Successfully set value with context")
	}

	// Get value with context
	ctxGetResult := <-cacheWithCtx.Get(ctxWithTimeout, "context:key")
	if ctxGetResult.Found {
		fmt.Printf("✓ Retrieved value with context: %v\n", ctxGetResult.Value)
	}

	fmt.Println("\n✓ Base cache usage examples completed")
}

// ExampleCreateRedisCache demonstrates using CreateRedisCache function
func ExampleCreateRedisCache() {
	ctx := context.Background()

	fmt.Println("\n=== CREATE REDIS CACHE USAGE ===")

	// Example 1: Basic Redis cache creation with default settings
	fmt.Println("\n--- Basic Redis Cache Creation ---")

	basicCache, err := CreateRedisCache(
		"localhost:6379", // addr
		"",               // password (empty for no auth)
		0,                // db (default database)
		10,               // poolSize
		5,                // minIdleConns
		3,                // maxRetries
		5*time.Second,    // dialTimeout
		3*time.Second,    // readTimeout
		3*time.Second,    // writeTimeout
		true,             // enablePipelining
		false,            // enableMetrics
		30*time.Second,   // healthCheckInterval
		5*time.Second,    // healthCheckTimeout
		nil,              // codec (use default JSON)
		nil,              // keyBuilder (use default)
		nil,              // keyHasher (use default)
	)

	if err != nil {
		fmt.Printf("Error creating basic cache: %v\n", err)
		return
	}
	defer basicCache.Close()

	fmt.Println("✓ Basic Redis cache created successfully")

	// Test basic operations
	stringCache := NewTypedCache[string](basicCache)
	<-stringCache.Set(ctx, "basic:test", "Hello from CreateRedisCache", time.Hour)
	result := <-stringCache.Get(ctx, "basic:test")
	if result.Found {
		fmt.Printf("✓ Basic cache test: %s\n", result.Value)
	}

	// Example 2: Advanced Redis cache creation with custom settings
	fmt.Println("\n--- Advanced Redis Cache Creation ---")

	// Create advanced key builder with namespacing
	keyBuilder, err := NewBuilder("myapp", "production", "secret-key-123")
	if err != nil {
		fmt.Printf("Error creating key builder: %v\n", err)
		return
	}

	advancedCache, err := CreateRedisCache(
		"localhost:6379",    // addr
		"",                  // password
		1,                   // db (use database 1)
		20,                  // poolSize (larger pool)
		10,                  // minIdleConns
		5,                   // maxRetries
		10*time.Second,      // dialTimeout (longer timeout)
		5*time.Second,       // readTimeout
		5*time.Second,       // writeTimeout
		true,                // enablePipelining
		true,                // enableMetrics
		60*time.Second,      // healthCheckInterval
		10*time.Second,      // healthCheckTimeout
		&JSONCodec{},        // custom codec
		keyBuilder,          // advanced key builder with namespacing
		&DefaultKeyHasher{}, // custom key hasher
	)

	if err != nil {
		fmt.Printf("Error creating advanced cache: %v\n", err)
		return
	}
	defer advancedCache.Close()

	fmt.Println("✓ Advanced Redis cache created successfully")

	// Demonstrate KeyBuilder functionality
	fmt.Println("\n--- KeyBuilder Demonstration ---")

	// Build different types of keys using the advanced key builder
	userKey := keyBuilder.Build("user", "123")
	fmt.Printf("✓ User key: %s\n", userKey)

	// Build list key with filters
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
	}
	listKey := keyBuilder.BuildList("users", filters)
	fmt.Printf("✓ List key: %s\n", listKey)

	// Build composite key
	compositeKey := keyBuilder.BuildComposite("user", "123", "org", "456")
	fmt.Printf("✓ Composite key: %s\n", compositeKey)

	// Build session key
	sessionKey := keyBuilder.BuildSession("session-abc123")
	fmt.Printf("✓ Session key: %s\n", sessionKey)

	// Use convenience methods
	userKey2 := keyBuilder.BuildUser("456")
	orgKey := keyBuilder.BuildOrg("789")
	fmt.Printf("✓ Convenience user key: %s\n", userKey2)
	fmt.Printf("✓ Convenience org key: %s\n", orgKey)

	// Test advanced operations with generated keys
	intCache := NewTypedCache[int](advancedCache)
	<-intCache.Set(ctx, userKey, 42, time.Hour)
	intResult := <-intCache.Get(ctx, userKey)
	if intResult.Found {
		fmt.Printf("✓ Advanced cache test with generated key: %d\n", intResult.Value)
	}

	// Example 3: Cache creation with authentication
	fmt.Println("\n--- Redis Cache with Authentication ---")

	// Note: This example assumes Redis is running without password
	// In production, you would provide the actual password
	authCache, err := CreateRedisCache(
		"localhost:6379", // addr
		"",               // password (empty for no auth, set actual password in production)
		0,                // db
		15,               // poolSize
		8,                // minIdleConns
		3,                // maxRetries
		5*time.Second,    // dialTimeout
		3*time.Second,    // readTimeout
		3*time.Second,    // writeTimeout
		true,             // enablePipelining
		false,            // enableMetrics
		30*time.Second,   // healthCheckInterval
		5*time.Second,    // healthCheckTimeout
		nil,              // codec
		nil,              // keyBuilder
		nil,              // keyHasher
	)

	if err != nil {
		fmt.Printf("Error creating auth cache: %v\n", err)
		return
	}
	defer authCache.Close()

	fmt.Println("✓ Authenticated Redis cache created successfully")

	// Example 4: High-performance cache configuration
	fmt.Println("\n--- High-Performance Cache Configuration ---")

	perfCache, err := CreateRedisCache(
		"localhost:6379", // addr
		"",               // password
		0,                // db
		50,               // poolSize (large pool for high concurrency)
		25,               // minIdleConns (many idle connections)
		10,               // maxRetries (more retries for reliability)
		2*time.Second,    // dialTimeout (fast connection)
		1*time.Second,    // readTimeout (fast reads)
		1*time.Second,    // writeTimeout (fast writes)
		true,             // enablePipelining (for batch operations)
		true,             // enableMetrics (for monitoring)
		10*time.Second,   // healthCheckInterval (frequent checks)
		2*time.Second,    // healthCheckTimeout (fast health checks)
		nil,              // codec
		nil,              // keyBuilder
		nil,              // keyHasher
	)

	if err != nil {
		fmt.Printf("Error creating performance cache: %v\n", err)
		return
	}
	defer perfCache.Close()

	fmt.Println("✓ High-performance Redis cache created successfully")

	// Test performance with batch operations
	perfItems := make(map[string]any)
	for i := 0; i < 100; i++ {
		perfItems[fmt.Sprintf("perf:item:%d", i)] = fmt.Sprintf("value_%d", i)
	}

	start := time.Now()
	perfResult := <-perfCache.MSet(ctx, perfItems, time.Hour)
	perfDuration := time.Since(start)

	if perfResult.Error == nil {
		fmt.Printf("✓ Performance test: MSet 100 items in %v\n", perfDuration)
	}

	// Example 5: Error handling for cache creation
	fmt.Println("\n--- Error Handling for Cache Creation ---")

	// Try to create cache with invalid address
	invalidCache, err := CreateRedisCache(
		"invalid:address:6379", // invalid addr
		"",                     // password
		0,                      // db
		10,                     // poolSize
		5,                      // minIdleConns
		3,                      // maxRetries
		1*time.Second,          // dialTimeout (short timeout for quick failure)
		1*time.Second,          // readTimeout
		1*time.Second,          // writeTimeout
		true,                   // enablePipelining
		false,                  // enableMetrics
		30*time.Second,         // healthCheckInterval
		5*time.Second,          // healthCheckTimeout
		nil,                    // codec
		nil,                    // keyBuilder
		nil,                    // keyHasher
	)

	if err != nil {
		fmt.Printf("✓ Correctly caught invalid address error: %v\n", err)
	} else {
		invalidCache.Close()
	}

	// Example 6: Using CreateRedisCache with different databases
	fmt.Println("\n--- Multiple Database Usage ---")

	// Create caches for different databases
	db0Cache, err := CreateRedisCache(
		"localhost:6379", "", 0, 10, 5, 3,
		5*time.Second, 3*time.Second, 3*time.Second,
		true, false, 30*time.Second, 5*time.Second,
		nil, nil, nil,
	)
	if err != nil {
		fmt.Printf("Error creating DB 0 cache: %v\n", err)
		return
	}
	defer db0Cache.Close()

	db1Cache, err := CreateRedisCache(
		"localhost:6379", "", 1, 10, 5, 3,
		5*time.Second, 3*time.Second, 3*time.Second,
		true, false, 30*time.Second, 5*time.Second,
		nil, nil, nil,
	)
	if err != nil {
		fmt.Printf("Error creating DB 1 cache: %v\n", err)
		return
	}
	defer db1Cache.Close()

	// Store data in different databases
	db0StringCache := NewTypedCache[string](db0Cache)
	db1StringCache := NewTypedCache[string](db1Cache)

	<-db0StringCache.Set(ctx, "shared:key", "Database 0 value", time.Hour)
	<-db1StringCache.Set(ctx, "shared:key", "Database 1 value", time.Hour)

	// Retrieve from both databases
	db0Result := <-db0StringCache.Get(ctx, "shared:key")
	db1Result := <-db1StringCache.Get(ctx, "shared:key")

	if db0Result.Found && db1Result.Found {
		fmt.Printf("✓ DB 0 value: %s\n", db0Result.Value)
		fmt.Printf("✓ DB 1 value: %s\n", db1Result.Value)
		fmt.Println("✓ Successfully isolated data in different databases")
	}

	fmt.Println("\n✓ CreateRedisCache examples completed")
}

// ExampleMain demonstrates how to run all examples
func ExampleMain() {
	fmt.Println("🚀 Go-CacheX Comprehensive Examples")
	fmt.Println("=====================================")

	// Run all examples
	ExampleAllDataTypes()
	ExampleBatchOperations()
	ExampleUtilityOperations()
	ExampleWithContext()
	ExampleErrorHandling()
	ExampleRealWorldScenarios()
	ExampleConcurrencyPatterns()
	ExamplePerformanceBenchmarks()
	ExampleAdvancedPatterns()
	ExampleAdvancedPatterns1()
	ExampleBaseCacheUsage()
	ExampleCreateRedisCache()
	ExampleKeyBuilderIntegration()

	fmt.Println("\n✅ All examples completed successfully!")
	fmt.Println("=====================================")
}

// ExampleKeyBuilderIntegration demonstrates comprehensive KeyBuilder usage
func ExampleKeyBuilderIntegration() {
	fmt.Println("=== KeyBuilder Integration Example ===")

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Create advanced key builder with namespacing
	keyBuilder, err := NewBuilder("ecommerce", "production", "secure-secret-2024")
	if err != nil {
		fmt.Printf("Error creating key builder: %v\n", err)
		return
	}

	// Create cache with advanced key builder
	cache := NewRedisCache(client, &JSONCodec{}, keyBuilder, &DefaultKeyHasher{})
	defer cache.Close()

	// Cast to CacheWithKeyBuilder to access helper methods
	cacheWithKeys, ok := cache.(CacheWithKeyBuilder)
	if !ok {
		fmt.Println("Cache does not support KeyBuilder helper methods")
		return
	}

	ctx := context.Background()

	// =============================================================================
	// 1. USER MANAGEMENT WITH KEYBUILDER
	// =============================================================================
	fmt.Println("\n--- User Management ---")

	// Build user keys using helper methods
	userKey := cacheWithKeys.BuildKey("user", "12345")
	userProfileKey := cacheWithKeys.BuildKey("user_profile", "12345")
	userSettingsKey := cacheWithKeys.BuildKey("user_settings", "12345")

	fmt.Printf("✓ User key: %s\n", userKey)
	fmt.Printf("✓ User profile key: %s\n", userProfileKey)
	fmt.Printf("✓ User settings key: %s\n", userSettingsKey)

	// Store user data
	userCache := NewTypedCache[map[string]any](cache)
	userData := map[string]any{
		"id":      "12345",
		"name":    "John Doe",
		"email":   "john@example.com",
		"role":    "customer",
		"status":  "active",
		"created": "2024-01-15",
	}

	<-userCache.Set(ctx, userKey, userData, 24*time.Hour)
	fmt.Println("✓ User data stored")

	// =============================================================================
	// 2. PRODUCT CATALOG WITH LIST KEYS
	// =============================================================================
	fmt.Println("\n--- Product Catalog ---")

	// Build list keys with different filters
	activeProductsKey := cacheWithKeys.BuildListKey("products", map[string]any{
		"status":   "active",
		"category": "electronics",
	})

	featuredProductsKey := cacheWithKeys.BuildListKey("products", map[string]any{
		"featured": true,
		"in_stock": true,
	})

	fmt.Printf("✓ Active products key: %s\n", activeProductsKey)
	fmt.Printf("✓ Featured products key: %s\n", featuredProductsKey)

	// Store product lists
	productListCache := NewTypedCache[[]map[string]any](cache)
	products := []map[string]any{
		{"id": "p1", "name": "Laptop", "price": 999.99},
		{"id": "p2", "name": "Phone", "price": 699.99},
	}

	<-productListCache.Set(ctx, activeProductsKey, products, 2*time.Hour)
	fmt.Println("✓ Product list stored")

	// =============================================================================
	// 3. RELATIONSHIP MAPPING WITH COMPOSITE KEYS
	// =============================================================================
	fmt.Println("\n--- Relationship Mapping ---")

	// Build composite keys for relationships
	userOrderKey := cacheWithKeys.BuildCompositeKey("user", "12345", "order", "ord-001")
	userCartKey := cacheWithKeys.BuildCompositeKey("user", "12345", "cart", "current")
	productCategoryKey := cacheWithKeys.BuildCompositeKey("product", "p1", "category", "electronics")

	fmt.Printf("✓ User-order key: %s\n", userOrderKey)
	fmt.Printf("✓ User-cart key: %s\n", userCartKey)
	fmt.Printf("✓ Product-category key: %s\n", productCategoryKey)

	// Store relationship data
	orderData := map[string]any{
		"order_id": "ord-001",
		"user_id":  "12345",
		"items":    []string{"p1", "p2"},
		"total":    1699.98,
		"status":   "pending",
	}

	<-userCache.Set(ctx, userOrderKey, orderData, 7*24*time.Hour)
	fmt.Println("✓ Order relationship stored")

	// =============================================================================
	// 4. SESSION MANAGEMENT
	// =============================================================================
	fmt.Println("\n--- Session Management ---")

	// Build session keys
	sessionKey := cacheWithKeys.BuildSessionKey("sess-abc123def456")
	userSessionKey := cacheWithKeys.BuildSessionKey("user-sess-12345")

	fmt.Printf("✓ Session key: %s\n", sessionKey)
	fmt.Printf("✓ User session key: %s\n", userSessionKey)

	// Store session data
	sessionData := map[string]any{
		"session_id": "sess-abc123def456",
		"user_id":    "12345",
		"login_time": "2024-01-15T10:30:00Z",
		"expires_at": "2024-01-15T22:30:00Z",
		"ip_address": "192.168.1.100",
	}

	<-userCache.Set(ctx, sessionKey, sessionData, 12*time.Hour)
	fmt.Println("✓ Session data stored")

	// =============================================================================
	// 5. RETRIEVAL AND VALIDATION
	// =============================================================================
	fmt.Println("\n--- Data Retrieval ---")

	// Retrieve and validate stored data
	retrievedUser := <-userCache.Get(ctx, userKey)
	if retrievedUser.Found {
		fmt.Printf("✓ Retrieved user: %v\n", retrievedUser.Value["name"])
	}

	retrievedOrder := <-userCache.Get(ctx, userOrderKey)
	if retrievedOrder.Found {
		fmt.Printf("✓ Retrieved order total: $%.2f\n", retrievedOrder.Value["total"])
	}

	retrievedSession := <-userCache.Get(ctx, sessionKey)
	if retrievedSession.Found {
		fmt.Printf("✓ Retrieved session for user: %v\n", retrievedSession.Value["user_id"])
	}

	// =============================================================================
	// 6. KEY PARSING AND VALIDATION
	// =============================================================================
	fmt.Println("\n--- Key Parsing ---")

	// Parse keys back to components
	entity, id, err := keyBuilder.ParseKey(userKey)
	if err == nil {
		fmt.Printf("✓ Parsed user key - Entity: %s, ID: %s\n", entity, id)
	}

	// =============================================================================
	// 7. CONVENIENCE METHODS DEMONSTRATION
	// =============================================================================
	fmt.Println("\n--- Convenience Methods ---")

	// Use convenience methods for common entities
	convenienceUserKey := keyBuilder.BuildUser("67890")
	convenienceOrgKey := keyBuilder.BuildOrg("org-001")
	convenienceProductKey := keyBuilder.BuildProduct("p3")
	convenienceOrderKey := keyBuilder.BuildOrder("ord-002")

	fmt.Printf("✓ Convenience user key: %s\n", convenienceUserKey)
	fmt.Printf("✓ Convenience org key: %s\n", convenienceOrgKey)
	fmt.Printf("✓ Convenience product key: %s\n", convenienceProductKey)
	fmt.Printf("✓ Convenience order key: %s\n", convenienceOrderKey)

	fmt.Println("\n✓ KeyBuilder integration example completed successfully")
}
