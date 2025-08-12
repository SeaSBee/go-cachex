package cache

import (
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
)

// NewBulkheadStore creates a new bulkhead store with separate read/write pools
func NewBulkheadStore(config redisstore.BulkheadConfig) (Store, error) {
	return redisstore.NewBulkhead(&config)
}

// NewBulkheadClusterStore creates a new bulkhead store with Redis cluster
func NewBulkheadClusterStore(addrs []string, password string, config redisstore.BulkheadConfig) (Store, error) {
	return redisstore.NewBulkheadCluster(addrs, password, &config)
}

// WithBulkheadStore creates a cache with bulkhead isolation
func WithBulkheadStore(config redisstore.BulkheadConfig) Option {
	return func(o *Options) {
		store, err := NewBulkheadStore(config)
		if err != nil {
			// This is a design decision - we could panic or return an error
			// For now, we'll log the error and continue without bulkhead
			fmt.Printf("Warning: Failed to create bulkhead store: %v\n", err)
			return
		}
		o.Store = store
		o.Bulkhead = config
	}
}

// WithBulkheadClusterStore creates a cache with bulkhead isolation for Redis cluster
func WithBulkheadClusterStore(addrs []string, password string, config redisstore.BulkheadConfig) Option {
	return func(o *Options) {
		store, err := NewBulkheadClusterStore(addrs, password, config)
		if err != nil {
			fmt.Printf("Warning: Failed to create bulkhead cluster store: %v\n", err)
			return
		}
		o.Store = store
		o.Bulkhead = config
	}
}

// DefaultBulkheadConfig returns a default bulkhead configuration
func DefaultBulkheadConfig() redisstore.BulkheadConfig {
	return redisstore.BulkheadConfig{
		// Read pool - larger for high read throughput
		ReadPoolSize:     30,
		ReadMinIdleConns: 15,
		ReadMaxRetries:   3,
		ReadDialTimeout:  5 * time.Second,
		ReadTimeout:      3 * time.Second,
		ReadPoolTimeout:  4 * time.Second,

		// Write pool - smaller for write operations
		WritePoolSize:     15,
		WriteMinIdleConns: 8,
		WriteMaxRetries:   3,
		WriteDialTimeout:  5 * time.Second,
		WriteTimeout:      3 * time.Second,
		WritePoolTimeout:  4 * time.Second,

		// Common settings
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

// HighPerformanceBulkheadConfig returns a high-performance bulkhead configuration
func HighPerformanceBulkheadConfig() redisstore.BulkheadConfig {
	return redisstore.BulkheadConfig{
		// Read pool - very large for maximum read throughput
		ReadPoolSize:     50,
		ReadMinIdleConns: 25,
		ReadMaxRetries:   5,
		ReadDialTimeout:  10 * time.Second,
		ReadTimeout:      5 * time.Second,
		ReadPoolTimeout:  8 * time.Second,

		// Write pool - optimized for write operations
		WritePoolSize:     25,
		WriteMinIdleConns: 12,
		WriteMaxRetries:   5,
		WriteDialTimeout:  10 * time.Second,
		WriteTimeout:      5 * time.Second,
		WritePoolTimeout:  8 * time.Second,

		// Common settings
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

// ResourceConstrainedBulkheadConfig returns a resource-constrained bulkhead configuration
func ResourceConstrainedBulkheadConfig() redisstore.BulkheadConfig {
	return redisstore.BulkheadConfig{
		// Read pool - minimal for resource constraints
		ReadPoolSize:     10,
		ReadMinIdleConns: 5,
		ReadMaxRetries:   2,
		ReadDialTimeout:  3 * time.Second,
		ReadTimeout:      2 * time.Second,
		ReadPoolTimeout:  3 * time.Second,

		// Write pool - minimal for resource constraints
		WritePoolSize:     5,
		WriteMinIdleConns: 2,
		WriteMaxRetries:   2,
		WriteDialTimeout:  3 * time.Second,
		WriteTimeout:      2 * time.Second,
		WritePoolTimeout:  3 * time.Second,

		// Common settings
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}
