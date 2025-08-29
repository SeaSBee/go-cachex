package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
)

// BenchmarkTaggingOptimization tests the performance improvements of optimized tagging
func BenchmarkTaggingOptimization(b *testing.B) {
	// Create memory store for testing
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.Run("StandardTagManager_AddTags", func(b *testing.B) {
		// Create standard tag manager
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%1000))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("AddTags failed: %v", err)
			}
		}
	})

	b.Run("OptimizedTagManager_AddTags", func(b *testing.B) {
		// Create optimized tag manager
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			BatchFlushInterval:          1 * time.Second, // Add this to prevent ticker panic
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%1000))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("AddTags failed: %v", err)
			}
		}
	})

	b.Run("StandardTagManager_GetKeysByTag", func(b *testing.B) {
		// Create standard tag manager and populate with data
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tag := "tag" + string(rune(i%3+1))
			keys, err := tm.GetKeysByTag(ctx, tag)
			if err != nil {
				b.Fatalf("GetKeysByTag failed: %v", err)
			}
			if len(keys) == 0 {
				b.Fatalf("Expected keys for tag %s", tag)
			}
		}
	})

	b.Run("OptimizedTagManager_GetKeysByTag", func(b *testing.B) {
		// Create optimized tag manager and populate with data
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			BatchFlushInterval:          1 * time.Second, // Add this to prevent ticker panic
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tag := "tag" + string(rune(i%3+1))
			keys, err := tm.GetKeysByTag(ctx, tag)
			if err != nil {
				b.Fatalf("GetKeysByTag failed: %v", err)
			}
			if len(keys) == 0 {
				b.Fatalf("Expected keys for tag %s", tag)
			}
		}
	})

	b.Run("StandardTagManager_InvalidateByTag", func(b *testing.B) {
		// Create standard tag manager and populate with data
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		// Pre-populate with data
		for i := 0; i < 100; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tag := "tag" + string(rune(i%3+1))
			err := tm.InvalidateByTag(ctx, tag)
			if err != nil {
				b.Fatalf("InvalidateByTag failed: %v", err)
			}
		}
	})

	b.Run("OptimizedTagManager_InvalidateByTag", func(b *testing.B) {
		// Create optimized tag manager and populate with data
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		// Pre-populate with data
		for i := 0; i < 100; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tag := "tag" + string(rune(i%3+1))
			err := tm.InvalidateByTag(ctx, tag)
			if err != nil {
				b.Fatalf("InvalidateByTag failed: %v", err)
			}
		}
	})
}

// BenchmarkTaggingOptimizationBatchOperations tests batch operation performance
func BenchmarkTaggingOptimizationBatchOperations(b *testing.B) {
	// Create memory store for testing
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.Run("StandardTagManager_BatchAddTags", func(b *testing.B) {
		// Create standard tag manager
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Add multiple tags to multiple keys
			for j := 0; j < 10; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
			}
		}
	})

	b.Run("OptimizedTagManager_BatchAddTags", func(b *testing.B) {
		// Create optimized tag manager
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Add multiple tags to multiple keys
			for j := 0; j < 10; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
			}
		}
	})

	b.Run("StandardTagManager_BatchRemoveTags", func(b *testing.B) {
		// Create standard tag manager and populate with data
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Remove multiple tags from multiple keys
			for j := 0; j < 10; j++ {
				key := "key" + string(rune(i%100)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2"}
				err := tm.RemoveTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("RemoveTags failed: %v", err)
				}
			}
		}
	})

	b.Run("OptimizedTagManager_BatchRemoveTags", func(b *testing.B) {
		// Create optimized tag manager and populate with data
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Remove multiple tags from multiple keys
			for j := 0; j < 10; j++ {
				key := "key" + string(rune(i%100)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2"}
				err := tm.RemoveTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("RemoveTags failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkTaggingOptimizationConcurrent tests concurrent operation performance
func BenchmarkTaggingOptimizationConcurrent(b *testing.B) {
	// Create memory store for testing
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.Run("StandardTagManager_ConcurrentAddTags", func(b *testing.B) {
		// Create standard tag manager
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key" + string(rune(i%1000))
				tags := []string{"tag1", "tag2", "tag3"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
				i++
			}
		})
	})

	b.Run("OptimizedTagManager_ConcurrentAddTags", func(b *testing.B) {
		// Create optimized tag manager
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key" + string(rune(i%1000))
				tags := []string{"tag1", "tag2", "tag3"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
				i++
			}
		})
	})

	b.Run("StandardTagManager_ConcurrentGetKeysByTag", func(b *testing.B) {
		// Create standard tag manager and populate with data
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				tag := "tag" + string(rune(i%3+1))
				keys, err := tm.GetKeysByTag(ctx, tag)
				if err != nil {
					b.Fatalf("GetKeysByTag failed: %v", err)
				}
				if len(keys) == 0 {
					b.Fatalf("Expected keys for tag %s", tag)
				}
				i++
			}
		})
	})

	b.Run("OptimizedTagManager_ConcurrentGetKeysByTag", func(b *testing.B) {
		// Create optimized tag manager and populate with data
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			tags := []string{"tag1", "tag2", "tag3"}
			err := tm.AddTags(ctx, key, tags...)
			if err != nil {
				b.Fatalf("Failed to add tags: %v", err)
			}
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				tag := "tag" + string(rune(i%3+1))
				keys, err := tm.GetKeysByTag(ctx, tag)
				if err != nil {
					b.Fatalf("GetKeysByTag failed: %v", err)
				}
				if len(keys) == 0 {
					b.Fatalf("Expected keys for tag %s", tag)
				}
				i++
			}
		})
	})
}

// BenchmarkTaggingOptimizationMemoryUsage tests memory usage patterns
func BenchmarkTaggingOptimizationMemoryUsage(b *testing.B) {
	// Create memory store for testing
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.Run("StandardTagManager_MemoryUsage", func(b *testing.B) {
		// Create standard tag manager
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate intensive tag operations
			for j := 0; j < 100; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}

				// Get keys by tag
				keys, err := tm.GetKeysByTag(ctx, "tag1")
				if err != nil {
					b.Fatalf("GetKeysByTag failed: %v", err)
				}
				_ = keys

				// Remove some tags
				err = tm.RemoveTags(ctx, key, "tag1", "tag2")
				if err != nil {
					b.Fatalf("RemoveTags failed: %v", err)
				}
			}
		}
	})

	b.Run("OptimizedTagManager_MemoryUsage", func(b *testing.B) {
		// Create optimized tag manager
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate intensive tag operations
			for j := 0; j < 100; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}

				// Get keys by tag
				keys, err := tm.GetKeysByTag(ctx, "tag1")
				if err != nil {
					b.Fatalf("GetKeysByTag failed: %v", err)
				}
				_ = keys

				// Remove some tags
				err = tm.RemoveTags(ctx, key, "tag1", "tag2")
				if err != nil {
					b.Fatalf("RemoveTags failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkTaggingOptimizationPersistence tests persistence performance
func BenchmarkTaggingOptimizationPersistence(b *testing.B) {
	// Create memory store for testing
	store, err := cachex.NewMemoryStore(&cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	})
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.Run("StandardTagManager_Persistence", func(b *testing.B) {
		// Create standard tag manager
		config := &cachex.TagConfig{
			EnablePersistence: true,
			TagMappingTTL:     1 * time.Hour,
			BatchSize:         100,
			EnableStats:       true,
			LoadTimeout:       5 * time.Second,
		}
		tm, err := cachex.NewTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create tag manager: %v", err)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Add tags to trigger persistence
			for j := 0; j < 50; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
			}
		}
	})

	b.Run("OptimizedTagManager_Persistence", func(b *testing.B) {
		// Create optimized tag manager
		config := &cachex.OptimizedTagConfig{
			EnablePersistence:           true,
			TagMappingTTL:               1 * time.Hour,
			BatchSize:                   100,
			EnableStats:                 true,
			LoadTimeout:                 5 * time.Second,
			EnableBackgroundPersistence: false, // Disable for benchmarks
			PersistenceInterval:         30 * time.Second,
			EnableMemoryOptimization:    true,
			MaxMemoryUsage:              100 * 1024 * 1024,
		}
		tm, err := cachex.NewOptimizedTagManager(store, config)
		if err != nil {
			b.Fatalf("Failed to create optimized tag manager: %v", err)
		}
		defer tm.Close()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Add tags to trigger persistence
			for j := 0; j < 50; j++ {
				key := "key" + string(rune(i)) + "-" + string(rune(j))
				tags := []string{"tag1", "tag2", "tag3"}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					b.Fatalf("AddTags failed: %v", err)
				}
			}
		}
	})
}
