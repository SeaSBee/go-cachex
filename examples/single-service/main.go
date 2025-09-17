package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seasbee/go-cachex"
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
	logx.Info("Starting Go-CacheX Single Service Demo")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Fatal("Failed to create memory store", logx.ErrorField(err))
	}
	defer memoryStore.Close()

	// Create key builder
	keyBuilder, err := cachex.NewBuilder("myapp", "dev", "secret123")
	if err != nil {
		log.Fatalf("Failed to create key builder: %v", err)
	}

	// Create JSON codec
	jsonCodec := cachex.NewJSONCodec()

	// Create cache
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithCodec(jsonCodec),
		cachex.WithKeyBuilder(keyBuilder),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Fatal("Failed to create cache", logx.ErrorField(err))
	}
	defer c.Close()

	logx.Info("Cache initialized successfully")

	// Example 1: Basic Get/Set
	demoBasicGetSet(c, keyBuilder)

	// Example 2: Context-Aware Operations
	demoContextAwareOperations(c, keyBuilder)

	// Example 3: Multiple Get/Set
	demoMultipleGetSet(c, keyBuilder)

	// Example 4: Exists and TTL
	demoExistsAndTTL(c, keyBuilder)

	logx.Info("Single Service Demo Complete")
}

func demoBasicGetSet(c cachex.Cache[User], keyBuilder *cachex.Builder) {
	logx.Info("Starting Basic Get/Set demo")

	user := User{
		ID:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}
	userKey := keyBuilder.BuildUser("123")

	// Set user in cache
	ctx := context.Background()
	setResult := <-c.Set(ctx, userKey, user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", userKey), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User set in cache successfully", logx.String("key", userKey), logx.String("user_id", user.ID))

	// Get user from cache
	getResult := <-c.Get(ctx, userKey)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", userKey), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("User retrieved from cache",
			logx.String("key", userKey),
			logx.String("name", getResult.Value.Name),
			logx.String("email", getResult.Value.Email))
	} else {
		logx.Warn("User not found in cache", logx.String("key", userKey))
	}
}

func demoContextAwareOperations(c cachex.Cache[User], keyBuilder *cachex.Builder) {
	logx.Info("Starting Context-Aware Operations demo")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create cache with context
	cWithCtx := c.WithContext(ctx)

	user := User{
		ID:        "456",
		Name:      "Jane Smith",
		Email:     "jane@example.com",
		CreatedAt: time.Now(),
	}
	userKey := keyBuilder.BuildUser("456")

	// Set user with context
	setResult := <-cWithCtx.Set(ctx, userKey, user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user with context", logx.String("key", userKey), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User set with context successfully", logx.String("key", userKey), logx.String("user_id", user.ID))

	// Get user with context
	getResult := <-cWithCtx.Get(ctx, userKey)
	if getResult.Error != nil {
		logx.Error("Failed to get user with context", logx.String("key", userKey), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("User retrieved with context",
			logx.String("key", userKey),
			logx.String("name", getResult.Value.Name),
			logx.String("email", getResult.Value.Email))
	} else {
		logx.Warn("User not found with context", logx.String("key", userKey))
	}

	// Demonstrate read-through pattern
	logx.Info("Demonstrating read-through pattern")

	// Try to get a non-existent user
	nonExistentKey := keyBuilder.BuildUser("999")
	getResult = <-cWithCtx.Get(ctx, nonExistentKey)
	if getResult.Error != nil {
		logx.Error("Failed to get non-existent user", logx.String("key", nonExistentKey), logx.ErrorField(getResult.Error))
		return
	}

	if !getResult.Found {
		logx.Info("User not found in cache, loading from database", logx.String("key", nonExistentKey))

		// Simulate loading from database
		loadedUser := User{
			ID:        "999",
			Name:      "Loaded User",
			Email:     "loaded@example.com",
			CreatedAt: time.Now(),
		}

		// Store in cache for future requests
		setResult = <-cWithCtx.Set(ctx, nonExistentKey, loadedUser, 0)
		if setResult.Error != nil {
			logx.Error("Failed to store loaded user", logx.String("key", nonExistentKey), logx.ErrorField(setResult.Error))
			return
		}
		logx.Info("Loaded user stored in cache", logx.String("key", nonExistentKey), logx.String("user_id", loadedUser.ID))
	}
}

func demoMultipleGetSet(c cachex.Cache[User], keyBuilder *cachex.Builder) {
	logx.Info("Starting Multiple Get/Set demo")

	ctx := context.Background()

	users := map[string]User{
		keyBuilder.BuildUser("789"): {
			ID:        "789",
			Name:      "Bob Wilson",
			Email:     "bob@example.com",
			CreatedAt: time.Now(),
		},
		keyBuilder.BuildUser("101"): {
			ID:        "101",
			Name:      "Alice Brown",
			Email:     "alice@example.com",
			CreatedAt: time.Now(),
		},
	}

	// Set multiple users
	msetResult := <-c.MSet(ctx, users, 10*time.Minute)
	if msetResult.Error != nil {
		logx.Error("Failed to set multiple users", logx.ErrorField(msetResult.Error), logx.Int("count", len(users)))
		return
	}
	logx.Info("Multiple users set in cache", logx.Int("count", len(users)))

	// Get multiple users
	keys := []string{keyBuilder.BuildUser("789"), keyBuilder.BuildUser("101")}
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to get multiple users", logx.ErrorField(mgetResult.Error), logx.String("keys", fmt.Sprintf("%v", keys)))
		return
	}

	logx.Info("Multiple users retrieved from cache",
		logx.Int("requested_count", len(keys)),
		logx.Int("retrieved_count", len(mgetResult.Values)))

	for key, user := range mgetResult.Values {
		logx.Info("Retrieved user",
			logx.String("key", key),
			logx.String("name", user.Name),
			logx.String("email", user.Email))
	}
}

func demoExistsAndTTL(c cachex.Cache[User], keyBuilder *cachex.Builder) {
	logx.Info("Starting Exists and TTL demo")

	ctx := context.Background()
	userKey := keyBuilder.BuildUser("123")

	// Check if user exists
	existsResult := <-c.Exists(ctx, userKey)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence", logx.String("key", userKey), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("User existence check", logx.String("key", userKey), logx.Bool("exists", existsResult.Found))

	// Get TTL
	ttlResult := <-c.TTL(ctx, userKey)
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL", logx.String("key", userKey), logx.ErrorField(ttlResult.Error))
		return
	}
	logx.Info("User TTL retrieved", logx.String("key", userKey), logx.String("ttl", ttlResult.TTL.String()))

	// Delete user
	delResult := <-c.Del(ctx, userKey)
	if delResult.Error != nil {
		logx.Error("Failed to delete user", logx.String("key", userKey), logx.ErrorField(delResult.Error))
		return
	}
	logx.Info("User deleted from cache", logx.String("key", userKey))

	// Check if user still exists
	existsResult = <-c.Exists(ctx, userKey)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence after deletion", logx.String("key", userKey), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("User existence check after deletion", logx.String("key", userKey), logx.Bool("exists", existsResult.Found))
}
