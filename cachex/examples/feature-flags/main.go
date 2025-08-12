package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/codec"
	"github.com/SeaSBee/go-cachex/cachex/pkg/features"
	"github.com/SeaSBee/go-cachex/cachex/pkg/key"
	"github.com/SeaSBee/go-cachex/cachex/pkg/memorystore"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Feature Flags Integration Example ===")

	// Example 1: Basic Feature Flags Usage
	demoBasicFeatureFlags()

	// Example 2: Dynamic Feature Updates
	demoDynamicFeatureUpdates()

	// Example 3: Percentage Rollouts
	demoPercentageRollouts()

	// Example 4: Temporary Features
	demoTemporaryFeatures()

	// Example 5: Feature Observers
	demoFeatureObservers()

	fmt.Println("\n=== Feature Flags Integration Example Complete ===")
}

func demoBasicFeatureFlags() {
	fmt.Println("1. Basic Feature Flags Usage")
	fmt.Println("=============================")

	// Create feature flags manager with default configuration
	featureManager := features.New(features.DefaultConfig())
	defer featureManager.Close()

	// Create memory store for testing
	memoryStore, err := memorystore.New(memorystore.DefaultConfig())
	if err != nil {
		log.Printf("Failed to create memory store: %v", err)
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cache.New[User](
		cache.WithStore(memoryStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithKeyBuilder(key.NewBuilder("demo", "features", "secret123")),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Check feature states
	fmt.Println("Checking feature states:")
	checkFeatureState(featureManager, features.RefreshAhead, "Refresh-Ahead")
	checkFeatureState(featureManager, features.NegativeCaching, "Negative Caching")
	checkFeatureState(featureManager, features.BloomFilter, "Bloom Filter")
	checkFeatureState(featureManager, features.DeadLetterQueue, "Dead Letter Queue")
	checkFeatureState(featureManager, features.PubSubInvalidation, "Pub/Sub Invalidation")
	checkFeatureState(featureManager, features.CircuitBreaker, "Circuit Breaker")
	checkFeatureState(featureManager, features.RateLimiting, "Rate Limiting")
	checkFeatureState(featureManager, features.Encryption, "Encryption")
	checkFeatureState(featureManager, features.Compression, "Compression")
	checkFeatureState(featureManager, features.Observability, "Observability")
	checkFeatureState(featureManager, features.Tagging, "Tagging")

	// Demonstrate conditional feature usage
	fmt.Println("\nDemonstrating conditional feature usage:")

	// Refresh-Ahead feature
	if featureManager.IsEnabled(features.RefreshAhead) {
		fmt.Println("  ✓ Refresh-Ahead is enabled - setting up proactive refresh")
		user := User{ID: "feature:1", Name: "Feature User", Email: "feature@example.com", CreatedAt: time.Now()}
		key := "user:feature:1"

		if err := c.Set(key, user, 30*time.Second); err != nil {
			log.Printf("Failed to set user: %v", err)
		} else {
			// Set up refresh-ahead
			if err := c.RefreshAhead(key, 10*time.Second, func(ctx context.Context) (User, error) {
				fmt.Println("    🔄 Refresh-ahead triggered - loading fresh data")
				return User{ID: "feature:1", Name: "Feature User (Refreshed)", Email: "feature@example.com", CreatedAt: time.Now()}, nil
			}); err != nil {
				log.Printf("Failed to set up refresh-ahead: %v", err)
			} else {
				fmt.Println("    ✓ Refresh-ahead scheduled")
			}
		}
	} else {
		fmt.Println("  ✗ Refresh-Ahead is disabled - skipping proactive refresh")
	}

	// Tagging feature
	if featureManager.IsEnabled(features.Tagging) {
		fmt.Println("  ✓ Tagging is enabled - adding tags to user")
		user := User{ID: "feature:2", Name: "Tagged User", Email: "tagged@example.com", CreatedAt: time.Now()}
		key := "user:feature:2"

		if err := c.Set(key, user, 5*time.Minute); err != nil {
			log.Printf("Failed to set user: %v", err)
		} else {
			if err := c.AddTags(key, "feature-demo", "tagged", "active"); err != nil {
				log.Printf("Failed to add tags: %v", err)
			} else {
				fmt.Println("    ✓ Tags added to user")
			}
		}
	} else {
		fmt.Println("  ✗ Tagging is disabled - skipping tag operations")
	}
}

func demoDynamicFeatureUpdates() {
	fmt.Println("\n2. Dynamic Feature Updates")
	fmt.Println("===========================")

	// Create feature flags manager
	featureManager := features.New(features.DefaultConfig())
	defer featureManager.Close()

	// Create memory store
	memoryStore, err := memorystore.New(memorystore.DefaultConfig())
	if err != nil {
		log.Printf("Failed to create memory store: %v", err)
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cache.New[User](
		cache.WithStore(memoryStore),
		cache.WithCodec(codec.NewJSONCodec()),
		cache.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}
	defer c.Close()

	// Initially disable refresh-ahead
	fmt.Println("Initially disabling refresh-ahead...")
	if err := featureManager.Disable(features.RefreshAhead, "Manual disable for testing"); err != nil {
		log.Printf("Failed to disable refresh-ahead: %v", err)
		return
	}

	// Check state
	checkFeatureState(featureManager, features.RefreshAhead, "Refresh-Ahead")

	// Try to use refresh-ahead (should be disabled)
	fmt.Println("\nTrying to use refresh-ahead (should be disabled)...")
	if featureManager.IsEnabled(features.RefreshAhead) {
		fmt.Println("  ✓ Refresh-Ahead is enabled")
	} else {
		fmt.Println("  ✗ Refresh-Ahead is disabled")
	}

	// Enable refresh-ahead dynamically
	fmt.Println("\nEnabling refresh-ahead dynamically...")
	if err := featureManager.Enable(features.RefreshAhead, "Manual enable for testing"); err != nil {
		log.Printf("Failed to enable refresh-ahead: %v", err)
		return
	}

	// Check state again
	checkFeatureState(featureManager, features.RefreshAhead, "Refresh-Ahead")

	// Try to use refresh-ahead (should be enabled)
	fmt.Println("\nTrying to use refresh-ahead (should be enabled)...")
	if featureManager.IsEnabled(features.RefreshAhead) {
		fmt.Println("  ✓ Refresh-Ahead is now enabled")

		// Use the feature
		user := User{ID: "dynamic:1", Name: "Dynamic User", Email: "dynamic@example.com", CreatedAt: time.Now()}
		key := "user:dynamic:1"

		if err := c.Set(key, user, 30*time.Second); err != nil {
			log.Printf("Failed to set user: %v", err)
		} else {
			if err := c.RefreshAhead(key, 10*time.Second, func(ctx context.Context) (User, error) {
				fmt.Println("    🔄 Dynamic refresh-ahead triggered")
				return User{ID: "dynamic:1", Name: "Dynamic User (Updated)", Email: "dynamic@example.com", CreatedAt: time.Now()}, nil
			}); err != nil {
				log.Printf("Failed to set up refresh-ahead: %v", err)
			} else {
				fmt.Println("    ✓ Dynamic refresh-ahead scheduled")
			}
		}
	} else {
		fmt.Println("  ✗ Refresh-Ahead is still disabled")
	}
}

func demoPercentageRollouts() {
	fmt.Println("\n3. Percentage Rollouts")
	fmt.Println("=======================")

	// Create feature flags manager
	featureManager := features.New(features.DefaultConfig())
	defer featureManager.Close()

	// Enable encryption with 50% rollout
	fmt.Println("Enabling encryption with 50% rollout...")
	if err := featureManager.EnableWithPercentage(features.Encryption, 50.0, "Gradual rollout for encryption"); err != nil {
		log.Printf("Failed to enable encryption with percentage: %v", err)
		return
	}

	// Test with different user IDs
	userIDs := []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7", "user8", "user9", "user10"}

	fmt.Println("Testing encryption feature with different user IDs:")
	enabledCount := 0

	for _, userID := range userIDs {
		if featureManager.IsEnabledWithPercentage(features.Encryption, userID) {
			fmt.Printf("  ✓ User %s: Encryption enabled\n", userID)
			enabledCount++
		} else {
			fmt.Printf("  ✗ User %s: Encryption disabled\n", userID)
		}
	}

	fmt.Printf("\nEncryption enabled for %d out of %d users (%.1f%%)\n",
		enabledCount, len(userIDs), float64(enabledCount)/float64(len(userIDs))*100)

	// Test with different percentages
	fmt.Println("\nTesting different rollout percentages:")
	percentages := []float64{10, 25, 50, 75, 90}

	for _, percentage := range percentages {
		if err := featureManager.EnableWithPercentage(features.Compression, percentage, fmt.Sprintf("Testing %v%% rollout", percentage)); err != nil {
			log.Printf("Failed to set percentage: %v", err)
			continue
		}

		enabledCount := 0
		for _, userID := range userIDs {
			if featureManager.IsEnabledWithPercentage(features.Compression, userID) {
				enabledCount++
			}
		}

		actualPercentage := float64(enabledCount) / float64(len(userIDs)) * 100
		fmt.Printf("  %v%% rollout: %d/%d users enabled (%.1f%% actual)\n",
			percentage, enabledCount, len(userIDs), actualPercentage)
	}
}

func demoTemporaryFeatures() {
	fmt.Println("\n4. Temporary Features")
	fmt.Println("======================")

	// Create feature flags manager
	featureManager := features.New(features.DefaultConfig())
	defer featureManager.Close()

	// Enable a feature temporarily for 5 seconds
	fmt.Println("Enabling rate limiting temporarily for 5 seconds...")
	if err := featureManager.EnableTemporarily(features.RateLimiting, 5*time.Second, "Temporary test of rate limiting"); err != nil {
		log.Printf("Failed to enable rate limiting temporarily: %v", err)
		return
	}

	// Check initial state
	checkFeatureState(featureManager, features.RateLimiting, "Rate Limiting")

	// Monitor the feature for 10 seconds
	fmt.Println("Monitoring rate limiting feature for 10 seconds...")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)

		if featureManager.IsEnabled(features.RateLimiting) {
			fmt.Printf("  Second %d: Rate limiting is enabled\n", i+1)
		} else {
			fmt.Printf("  Second %d: Rate limiting is disabled (expired)\n", i+1)
			break
		}
	}

	// Check final state
	checkFeatureState(featureManager, features.RateLimiting, "Rate Limiting")
}

func demoFeatureObservers() {
	fmt.Println("\n5. Feature Observers")
	fmt.Println("=====================")

	// Create feature flags manager
	featureManager := features.New(features.DefaultConfig())
	defer featureManager.Close()

	// Add observers for different features
	fmt.Println("Adding observers for feature changes...")

	// Observer for refresh-ahead
	featureManager.AddObserver(features.RefreshAhead, func(feature features.Feature, enabled bool) {
		fmt.Printf("🔄 Refresh-Ahead feature changed: %v\n", enabled)
		if enabled {
			fmt.Println("    ✓ Setting up refresh-ahead infrastructure")
		} else {
			fmt.Println("    ✗ Shutting down refresh-ahead infrastructure")
		}
	})

	// Observer for encryption
	featureManager.AddObserver(features.Encryption, func(feature features.Feature, enabled bool) {
		fmt.Printf("🔐 Encryption feature changed: %v\n", enabled)
		if enabled {
			fmt.Println("    ✓ Initializing encryption keys")
		} else {
			fmt.Println("    ✗ Cleaning up encryption keys")
		}
	})

	// Observer for observability
	featureManager.AddObserver(features.Observability, func(feature features.Feature, enabled bool) {
		fmt.Printf("📊 Observability feature changed: %v\n", enabled)
		if enabled {
			fmt.Println("    ✓ Starting metrics collection")
		} else {
			fmt.Println("    ✗ Stopping metrics collection")
		}
	})

	// Trigger feature changes
	fmt.Println("\nTriggering feature changes to test observers...")

	// Disable refresh-ahead
	fmt.Println("Disabling refresh-ahead...")
	if err := featureManager.Disable(features.RefreshAhead, "Testing observer"); err != nil {
		log.Printf("Failed to disable refresh-ahead: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Enable encryption
	fmt.Println("Enabling encryption...")
	if err := featureManager.Enable(features.Encryption, "Testing observer"); err != nil {
		log.Printf("Failed to enable encryption: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Disable observability
	fmt.Println("Disabling observability...")
	if err := featureManager.Disable(features.Observability, "Testing observer"); err != nil {
		log.Printf("Failed to disable observability: %v", err)
	}

	time.Sleep(1 * time.Second)
	fmt.Println("Observer testing complete")
}

// Helper function to check and display feature state
func checkFeatureState(manager *features.Manager, feature features.Feature, name string) {
	state, err := manager.GetFeatureState(feature)
	if err != nil {
		fmt.Printf("  %s: Error getting state - %v\n", name, err)
		return
	}

	status := "disabled"
	if state.Enabled {
		status = "enabled"
	}

	fmt.Printf("  %s: %s (%.1f%% rollout)\n", name, status, state.Percentage)
}
