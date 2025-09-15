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

func main() {
	logx.Info("Starting Go-CacheX Implementation Details Demo")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store", logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()
	logx.Info("Memory store created successfully")

	// Create cache with basic features
	c, err := cachex.New[User](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache", logx.ErrorField(err))
		return
	}
	defer c.Close()
	logx.Info("Cache created successfully", logx.String("default_ttl", "5m"))

	// Demo 1: Basic Cache Operations
	demoBasicOperations(c)

	// Demo 2: Context-Aware Operations
	demoContextAwareOperations(c)

	// Demo 3: Multi-Operations
	demoMultiOperations(c)

	logx.Info("Implementation Details Demo Complete")
}

func demoBasicOperations(c cachex.Cache[User]) {
	logx.Info("Starting Basic Cache Operations Demo")

	// Create sample user
	user := User{
		ID:        "user:1",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:1"

	ctx := context.Background()

	// Set operation
	logx.Info("Setting user in cache", logx.String("key", key), logx.String("user_name", user.Name))
	setResult := <-c.Set(ctx, key, user, 10*time.Minute)
	if setResult.Error != nil {
		logx.Error("Failed to set user", logx.String("key", key), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User stored successfully", logx.String("key", key))

	// Get operation
	logx.Info("Getting user from cache", logx.String("key", key))
	getResult := <-c.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user", logx.String("key", key), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("User retrieved successfully", logx.String("key", key), logx.String("user_name", getResult.Value.Name), logx.String("email", getResult.Value.Email))
	} else {
		logx.Warn("User not found in cache", logx.String("key", key))
	}

	// Exists operation
	logx.Info("Checking if user exists", logx.String("key", key))
	existsResult := <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence", logx.String("key", key), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("Existence check completed", logx.String("key", key), logx.Bool("exists", existsResult.Found))

	// TTL operation
	logx.Info("Getting TTL", logx.String("key", key))
	ttlResult := <-c.TTL(ctx, key)
	if ttlResult.Error != nil {
		logx.Error("Failed to get TTL", logx.String("key", key), logx.ErrorField(ttlResult.Error))
		return
	}
	logx.Info("TTL retrieved", logx.String("key", key), logx.String("ttl", ttlResult.TTL.String()))

	// Delete operation
	logx.Info("Deleting user from cache", logx.String("key", key))
	delResult := <-c.Del(ctx, key)
	if delResult.Error != nil {
		logx.Error("Failed to delete user", logx.String("key", key), logx.ErrorField(delResult.Error))
		return
	}
	logx.Info("User deleted successfully", logx.String("key", key))

	// Verify deletion
	existsResult = <-c.Exists(ctx, key)
	if existsResult.Error != nil {
		logx.Error("Failed to check existence after deletion", logx.String("key", key), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("Verification after deletion", logx.String("key", key), logx.Bool("exists", existsResult.Found))
}

func demoContextAwareOperations(c cachex.Cache[User]) {
	logx.Info("Starting Context-Aware Operations Demo")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create cache with context
	cWithCtx := c.WithContext(ctx)

	// Create sample user
	user := User{
		ID:        "user:ctx:1",
		Name:      "Context User",
		Email:     "context@example.com",
		CreatedAt: time.Now(),
	}

	key := "user:ctx:1"

	// Set with context
	logx.Info("Setting user with context", logx.String("key", key), logx.String("user_name", user.Name))
	setResult := <-cWithCtx.Set(ctx, key, user, 0)
	if setResult.Error != nil {
		logx.Error("Failed to set user with context", logx.String("key", key), logx.ErrorField(setResult.Error))
		return
	}
	logx.Info("User stored with context successfully", logx.String("key", key))

	// Get with context
	logx.Info("Getting user with context", logx.String("key", key))
	getResult := <-cWithCtx.Get(ctx, key)
	if getResult.Error != nil {
		logx.Error("Failed to get user with context", logx.String("key", key), logx.ErrorField(getResult.Error))
		return
	}

	if getResult.Found {
		logx.Info("User retrieved with context successfully", logx.String("key", key), logx.String("user_name", getResult.Value.Name), logx.String("email", getResult.Value.Email))
	} else {
		logx.Warn("User not found with context", logx.String("key", key))
	}

	// Demonstrate read-through pattern
	logx.Info("Demonstrating read-through pattern")

	// Try to get a non-existent user
	nonExistentKey := "user:non-existent"
	logx.Info("Attempting to get non-existent user", logx.String("key", nonExistentKey))
	getResult = <-cWithCtx.Get(ctx, nonExistentKey)
	if getResult.Error != nil {
		logx.Error("Failed to get non-existent user", logx.String("key", nonExistentKey), logx.ErrorField(getResult.Error))
		return
	}

	if !getResult.Found {
		logx.Info("User not found in cache, simulating database load", logx.String("key", nonExistentKey))

		// Simulate loading from database
		loadedUser := User{
			ID:        "user:non-existent",
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
		logx.Info("Loaded user stored in cache successfully", logx.String("key", nonExistentKey), logx.String("user_name", loadedUser.Name))
	}
}

func demoMultiOperations(c cachex.Cache[User]) {
	logx.Info("Starting Multi-Operations Demo")

	// Create multiple users
	users := map[string]User{
		"user:multi:1": {ID: "user:multi:1", Name: "Multi User 1", Email: "multi1@example.com", CreatedAt: time.Now()},
		"user:multi:2": {ID: "user:multi:2", Name: "Multi User 2", Email: "multi2@example.com", CreatedAt: time.Now()},
		"user:multi:3": {ID: "user:multi:3", Name: "Multi User 3", Email: "multi3@example.com", CreatedAt: time.Now()},
		"user:multi:4": {ID: "user:multi:4", Name: "Multi User 4", Email: "multi4@example.com", CreatedAt: time.Now()},
		"user:multi:5": {ID: "user:multi:5", Name: "Multi User 5", Email: "multi5@example.com", CreatedAt: time.Now()},
	}

	ctx := context.Background()

	// Multi-set operation
	logx.Info("Setting multiple users", logx.Int("count", len(users)))
	msetResult := <-c.MSet(ctx, users, 10*time.Minute)
	if msetResult.Error != nil {
		logx.Error("Failed to set multiple users", logx.ErrorField(msetResult.Error))
		return
	}
	logx.Info("Multiple users stored successfully", logx.Int("count", len(users)))

	// Multi-get operation
	keys := []string{"user:multi:1", "user:multi:2", "user:multi:3", "user:multi:4", "user:multi:5"}
	logx.Info("Getting multiple users", logx.String("keys", fmt.Sprintf("%v", keys)))
	mgetResult := <-c.MGet(ctx, keys...)
	if mgetResult.Error != nil {
		logx.Error("Failed to get multiple users", logx.ErrorField(mgetResult.Error))
		return
	}
	logx.Info("Multiple users retrieved successfully", logx.Int("requested", len(keys)), logx.Int("retrieved", len(mgetResult.Values)))

	// Display retrieved users
	logx.Info("Retrieved users details")
	for key, user := range mgetResult.Values {
		logx.Info("User details", logx.String("key", key), logx.String("name", user.Name), logx.String("email", user.Email))
	}

	// Multi-delete operation
	logx.Info("Deleting multiple users", logx.String("keys", fmt.Sprintf("%v", keys)))
	delResult := <-c.Del(ctx, keys...)
	if delResult.Error != nil {
		logx.Error("Failed to delete multiple users", logx.ErrorField(delResult.Error))
		return
	}
	logx.Info("Multiple users deleted successfully", logx.Int("count", len(keys)))

	// Verify deletion
	existsResult := <-c.Exists(ctx, "user:multi:1")
	if existsResult.Error != nil {
		logx.Error("Failed to check existence after deletion", logx.String("key", "user:multi:1"), logx.ErrorField(existsResult.Error))
		return
	}
	logx.Info("Verification after multi-deletion", logx.String("key", "user:multi:1"), logx.Bool("exists", existsResult.Found))
}
