package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/cb"
	"github.com/SeaSBee/go-cachex/cachex/internal/retry"
	"github.com/SeaSBee/go-cachex/cachex/pkg/cache"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

func main() {
	fmt.Println("🚀 Go-CacheX Circuit Breaker & Retry Example")
	fmt.Println("=============================================")

	// Example 1: Circuit Breaker Standalone
	demoCircuitBreaker()

	// Example 2: Retry with Exponential Backoff
	demoRetryWithBackoff()

	// Example 3: Cache with Circuit Breaker and Retry
	demoCacheWithResilience()

	fmt.Println("\n✅ All examples completed successfully!")
}

// Example 1: Circuit Breaker Standalone
func demoCircuitBreaker() {
	fmt.Println("\n📊 Example 1: Circuit Breaker Standalone")
	fmt.Println("----------------------------------------")

	// Create circuit breaker with aggressive settings
	circuitBreaker := cb.New(cb.Config{
		Threshold:   3,               // Open after 3 failures
		Timeout:     5 * time.Second, // Stay open for 5 seconds
		HalfOpenMax: 2,               // Allow 2 requests in half-open state
	})

	fmt.Printf("Initial state: %s\n", circuitBreaker.GetState())

	// Simulate some operations
	for i := 0; i < 5; i++ {
		err := circuitBreaker.Execute(func() error {
			// Simulate intermittent failures
			if rand.Float64() < 0.7 { // 70% failure rate
				return fmt.Errorf("simulated failure %d", i+1)
			}
			return nil
		})

		stats := circuitBreaker.GetStats()
		fmt.Printf("Operation %d: State=%s, Failures=%d, Successes=%d, Error=%v\n",
			i+1, stats.State, stats.FailureCount, stats.SuccessCount, err)
	}

	// Wait for circuit to potentially close
	fmt.Println("Waiting 6 seconds for circuit to potentially close...")
	time.Sleep(6 * time.Second)

	// Try a few more operations
	for i := 0; i < 3; i++ {
		err := circuitBreaker.Execute(func() error {
			// Simulate success
			return nil
		})

		stats := circuitBreaker.GetStats()
		fmt.Printf("Recovery %d: State=%s, Failures=%d, Successes=%d, Error=%v\n",
			i+1, stats.State, stats.FailureCount, stats.SuccessCount, err)
	}

	// Show final statistics
	finalStats := circuitBreaker.GetStats()
	fmt.Printf("\nFinal Statistics:\n")
	fmt.Printf("- Total Requests: %d\n", finalStats.TotalRequests)
	fmt.Printf("- Total Failures: %d\n", finalStats.TotalFailures)
	fmt.Printf("- Total Successes: %d\n", finalStats.TotalSuccesses)
	fmt.Printf("- Failure Rate: %.2f%%\n", finalStats.FailureRate())
	fmt.Printf("- Success Rate: %.2f%%\n", finalStats.SuccessRate())
}

// Example 2: Retry with Exponential Backoff
func demoRetryWithBackoff() {
	fmt.Println("\n🔄 Example 2: Retry with Exponential Backoff")
	fmt.Println("---------------------------------------------")

	// Create different retry policies
	policies := map[string]retry.Policy{
		"Default":      retry.DefaultPolicy(),
		"Aggressive":   retry.AggressivePolicy(),
		"Conservative": retry.ConservativePolicy(),
	}

	for name, policy := range policies {
		fmt.Printf("\n%s Policy (MaxAttempts=%d, InitialDelay=%v, MaxDelay=%v):\n",
			name, policy.MaxAttempts, policy.InitialDelay, policy.MaxDelay)

		attempt := 0
		stats, err := retry.RetryWithStats(policy, func() error {
			attempt++
			fmt.Printf("  Attempt %d... ", attempt)

			// Simulate failures for first few attempts
			if attempt < 3 {
				fmt.Println("FAILED")
				return fmt.Errorf("attempt %d failed", attempt)
			}

			fmt.Println("SUCCESS")
			return nil
		})

		if err != nil {
			fmt.Printf("  Final result: FAILED after %d attempts, total delay: %v\n", stats.Attempts, stats.TotalDelay)
		} else {
			fmt.Printf("  Final result: SUCCESS after %d attempts, total delay: %v\n", stats.Attempts, stats.TotalDelay)
		}
	}
}

// Example 3: Cache with Circuit Breaker and Retry
func demoCacheWithResilience() {
	fmt.Println("\n💾 Example 3: Cache with Circuit Breaker and Retry")
	fmt.Println("--------------------------------------------------")

	// Create Redis store (this will fail if Redis is not running, but that's OK for demo)
	redisConfig := &redisstore.Config{
		Addr:         "localhost:6379",
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}

	store, err := redisstore.New(redisConfig)
	if err != nil {
		fmt.Printf("⚠️  Redis connection failed (expected if Redis is not running): %v\n", err)
		fmt.Println("   Continuing with mock operations...")
		demoCacheResilienceWithMock()
		return
	}

	// Create cache with circuit breaker and retry
	c, err := cache.New[string](
		cache.WithStore(store),
		cache.WithMaxRetries(3),
		cache.WithRetryDelay(100*time.Millisecond),
		cache.WithCircuitBreaker(cache.CircuitBreakerConfig{
			Threshold:   5,
			Timeout:     30 * time.Second,
			HalfOpenMax: 3,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// Test cache operations with resilience
	testCacheOperations(c)
}

// Demo cache resilience with mock operations
func demoCacheResilienceWithMock() {
	fmt.Println("   Simulating cache operations with circuit breaker and retry...")

	// Create a mock circuit breaker
	circuitBreaker := cb.New(cb.Config{
		Threshold:   3,
		Timeout:     2 * time.Second,
		HalfOpenMax: 2,
	})

	// Create retry policy
	retryPolicy := retry.Policy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}

	// Simulate cache operations
	operations := []string{"GET", "SET", "MGET", "DEL"}

	for _, op := range operations {
		fmt.Printf("   Testing %s operation...\n", op)

		attempt := 0
		err := retry.RetryWithContext(context.Background(), retryPolicy, func() error {
			attempt++
			fmt.Printf("     Attempt %d: ", attempt)

			// Execute with circuit breaker
			return circuitBreaker.Execute(func() error {
				// Simulate intermittent failures
				if rand.Float64() < 0.6 { // 60% failure rate
					fmt.Println("FAILED (circuit breaker protected)")
					return fmt.Errorf("simulated %s failure", op)
				}
				fmt.Println("SUCCESS")
				return nil
			})
		})

		if err != nil {
			fmt.Printf("     Final result: FAILED - %v\n", err)
		} else {
			fmt.Printf("     Final result: SUCCESS\n")
		}

		// Show circuit breaker stats
		stats := circuitBreaker.GetStats()
		fmt.Printf("     Circuit Breaker: State=%s, Failures=%d, Successes=%d\n",
			stats.State, stats.FailureCount, stats.SuccessCount)

		time.Sleep(500 * time.Millisecond) // Brief pause between operations
	}

	// Show final statistics
	finalStats := circuitBreaker.GetStats()
	fmt.Printf("\n   Final Circuit Breaker Statistics:\n")
	fmt.Printf("   - Total Requests: %d\n", finalStats.TotalRequests)
	fmt.Printf("   - Total Failures: %d\n", finalStats.TotalFailures)
	fmt.Printf("   - Total Successes: %d\n", finalStats.TotalSuccesses)
	fmt.Printf("   - Failure Rate: %.2f%%\n", finalStats.FailureRate())
	fmt.Printf("   - Success Rate: %.2f%%\n", finalStats.SuccessRate())
}

// Test cache operations with resilience
func testCacheOperations(c cache.Cache[string]) {
	fmt.Println("   Testing cache operations with resilience...")

	// Test Set operation
	fmt.Println("   Testing SET operation...")
	err := c.Set("test-key", "test-value", 5*time.Minute)
	if err != nil {
		fmt.Printf("     SET failed: %v\n", err)
	} else {
		fmt.Println("     SET successful")
	}

	// Test Get operation
	fmt.Println("   Testing GET operation...")
	value, found, err := c.Get("test-key")
	if err != nil {
		fmt.Printf("     GET failed: %v\n", err)
	} else if found {
		fmt.Printf("     GET successful: %s\n", value)
	} else {
		fmt.Println("     GET: key not found")
	}

	// Test MSet operation
	fmt.Println("   Testing MSET operation...")
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err = c.MSet(items, 5*time.Minute)
	if err != nil {
		fmt.Printf("     MSET failed: %v\n", err)
	} else {
		fmt.Println("     MSET successful")
	}

	// Test MGet operation
	fmt.Println("   Testing MGET operation...")
	results, err := c.MGet("key1", "key2", "key3")
	if err != nil {
		fmt.Printf("     MGET failed: %v\n", err)
	} else {
		fmt.Printf("     MGET successful: %v\n", results)
	}

	// Note: Circuit breaker statistics are available in the implementation
	// but not exposed in the public interface for this example
	fmt.Printf("\n   Cache operations completed successfully!\n")
}
