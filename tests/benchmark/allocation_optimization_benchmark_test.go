package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

// BenchmarkAllocationOptimizationHotPaths tests allocation optimization in hot paths
func BenchmarkAllocationOptimizationHotPaths(b *testing.B) {
	// Create a memory store for testing
	config := &cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Pre-populate with some data
	testData := []byte("test-value-for-allocation-benchmark")
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		result := <-store.Set(ctx, key, testData, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Failed to set test data: %v", result.Error)
		}
	}

	b.Run("Get_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%100))
			result := <-store.Get(ctx, key)
			if result.Error != nil {
				b.Fatalf("Get failed: %v", result.Error)
			}
			if !result.Exists {
				b.Fatalf("Expected key to exist")
			}
		}
	})

	b.Run("Set_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "set-key" + string(rune(i))
			value := []byte("value" + string(rune(i)))
			result := <-store.Set(ctx, key, value, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("Set failed: %v", result.Error)
			}
		}
	})

	b.Run("MGet_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = "key" + string(rune((i+j)%100))
			}
			result := <-store.MGet(ctx, keys...)
			if result.Error != nil {
				b.Fatalf("MGet failed: %v", result.Error)
			}
			// Some keys might not exist, so we just check that we got a result
			if result.Error != nil {
				b.Fatalf("MGet failed: %v", result.Error)
			}
		}
	})

	b.Run("MSet_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			items := make(map[string][]byte, 10)
			for j := 0; j < 10; j++ {
				key := "mset-key" + string(rune(i)) + "-" + string(rune(j))
				value := []byte("value" + string(rune(i)) + "-" + string(rune(j)))
				items[key] = value
			}
			result := <-store.MSet(ctx, items, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("MSet failed: %v", result.Error)
			}
		}
	})

	b.Run("Del_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			keys := make([]string, 5)
			for j := 0; j < 5; j++ {
				keys[j] = "del-key" + string(rune(i)) + "-" + string(rune(j))
			}
			result := <-store.Del(ctx, keys...)
			if result.Error != nil {
				b.Fatalf("Del failed: %v", result.Error)
			}
		}
	})

	b.Run("IncrBy_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := "incr-key" + string(rune(i%100))
			result := <-store.IncrBy(ctx, key, 1, 5*time.Minute)
			if result.Error != nil {
				b.Fatalf("IncrBy failed: %v", result.Error)
			}
		}
	})
}

// BenchmarkAllocationOptimizationValueCopy tests optimized value copying
func BenchmarkAllocationOptimizationValueCopy(b *testing.B) {
	// Test data of different sizes
	testCases := []struct {
		name string
		data []byte
	}{
		{"Small", []byte("small-data")},
		{"Medium", make([]byte, 2048)},
		{"Large", make([]byte, 8192)},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_OptimizedCopy", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := cachex.OptimizedValueCopy(tc.data)
				if len(result) != len(tc.data) {
					b.Fatalf("Copy length mismatch: expected %d, got %d", len(tc.data), len(result))
				}
			}
		})

		b.Run(tc.name+"_StandardCopy", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := make([]byte, len(tc.data))
				copy(result, tc.data)
				if len(result) != len(tc.data) {
					b.Fatalf("Copy length mismatch: expected %d, got %d", len(tc.data), len(result))
				}
			}
		})
	}
}

// BenchmarkAllocationOptimizationStringBuilder tests optimized string building
func BenchmarkAllocationOptimizationStringBuilder(b *testing.B) {
	b.Run("OptimizedStringBuilder", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sb := cachex.NewOptimizedStringBuilder()
			sb.WriteString("prefix")
			sb.WriteByte(':')
			sb.WriteString("entity")
			sb.WriteByte(':')
			sb.WriteString("id")
			sb.WriteByte(':')
			sb.WriteString("suffix")
			result := sb.String()
			if len(result) == 0 {
				b.Fatalf("Expected non-empty string")
			}
		}
	})

	b.Run("StandardStringBuilder", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var sb cachex.StringBuilder
			sb.WriteString("prefix")
			sb.WriteByte(':')
			sb.WriteString("entity")
			sb.WriteByte(':')
			sb.WriteString("id")
			sb.WriteByte(':')
			sb.WriteString("suffix")
			result := sb.String()
			if len(result) == 0 {
				b.Fatalf("Expected non-empty string")
			}
		}
	})

	b.Run("StringConcatenation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := "prefix" + ":" + "entity" + ":" + "id" + ":" + "suffix"
			if len(result) == 0 {
				b.Fatalf("Expected non-empty string")
			}
		}
	})
}

// BenchmarkAllocationOptimizationMapCopy tests optimized map copying
func BenchmarkAllocationOptimizationMapCopy(b *testing.B) {
	// Create test maps of different sizes
	testCases := []struct {
		name string
		data map[string][]byte
	}{
		{"Small", map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}},
		{"Medium", func() map[string][]byte {
			m := make(map[string][]byte, 50)
			for i := 0; i < 50; i++ {
				m["key"+string(rune(i))] = []byte("value" + string(rune(i)))
			}
			return m
		}()},
		{"Large", func() map[string][]byte {
			m := make(map[string][]byte, 200)
			for i := 0; i < 200; i++ {
				m["key"+string(rune(i))] = []byte("value" + string(rune(i)))
			}
			return m
		}()},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_OptimizedCopy", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := cachex.OptimizedMapCopy(tc.data)
				if len(result) != len(tc.data) {
					b.Fatalf("Copy length mismatch: expected %d, got %d", len(tc.data), len(result))
				}
			}
		})

		b.Run(tc.name+"_StandardCopy", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := make(map[string][]byte, len(tc.data))
				for k, v := range tc.data {
					result[k] = make([]byte, len(v))
					copy(result[k], v)
				}
				if len(result) != len(tc.data) {
					b.Fatalf("Copy length mismatch: expected %d, got %d", len(tc.data), len(result))
				}
			}
		})
	}
}

// BenchmarkAllocationOptimizationStringJoin tests optimized string joining
func BenchmarkAllocationOptimizationStringJoin(b *testing.B) {
	// Test data of different sizes
	testCases := []struct {
		name string
		data []string
	}{
		{"Small", []string{"a", "b", "c", "d", "e"}},
		{"Medium", func() []string {
			s := make([]string, 50)
			for i := 0; i < 50; i++ {
				s[i] = "item" + string(rune(i))
			}
			return s
		}()},
		{"Large", func() []string {
			s := make([]string, 200)
			for i := 0; i < 200; i++ {
				s[i] = "item" + string(rune(i))
			}
			return s
		}()},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_OptimizedJoin", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := cachex.OptimizedStringJoin(tc.data, ":")
				if len(result) == 0 {
					b.Fatalf("Expected non-empty string")
				}
			}
		})

		b.Run(tc.name+"_StandardJoin", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Simulate standard string joining
				var result string
				for j, item := range tc.data {
					if j > 0 {
						result += ":"
					}
					result += item
				}
				if len(result) == 0 {
					b.Fatalf("Expected non-empty string")
				}
			}
		})
	}
}

// BenchmarkAllocationOptimizationConcurrent tests concurrent allocation optimization
func BenchmarkAllocationOptimizationConcurrent(b *testing.B) {
	// Create a memory store for testing
	config := &cachex.MemoryConfig{
		MaxSize:         10000,
		MaxMemoryMB:     100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  cachex.EvictionPolicyLRU,
		EnableStats:     true,
	}

	store, err := cachex.NewMemoryStore(config)
	if err != nil {
		b.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Pre-populate with some data
	testData := []byte("concurrent-test-data")
	for i := 0; i < 100; i++ {
		key := "concurrent-key" + string(rune(i))
		result := <-store.Set(ctx, key, testData, 5*time.Minute)
		if result.Error != nil {
			b.Fatalf("Failed to set test data: %v", result.Error)
		}
	}

	b.Run("ConcurrentGet_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "concurrent-key" + string(rune(i%100))
				result := <-store.Get(ctx, key)
				if result.Error != nil {
					b.Fatalf("Get failed: %v", result.Error)
				}
				if !result.Exists {
					b.Fatalf("Expected key to exist")
				}
				i++
			}
		})
	})

	b.Run("ConcurrentSet_WithOptimization", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "concurrent-set-key" + string(rune(i))
				value := []byte("concurrent-value" + string(rune(i)))
				result := <-store.Set(ctx, key, value, 5*time.Minute)
				if result.Error != nil {
					b.Fatalf("Set failed: %v", result.Error)
				}
				i++
			}
		})
	})
}

// BenchmarkAllocationOptimizationMemoryUsage tests memory usage patterns
func BenchmarkAllocationOptimizationMemoryUsage(b *testing.B) {
	b.Run("MemoryUsage", func(b *testing.B) {
		b.ReportAllocs()

		// Create allocation optimizer (used implicitly through global functions)
		_ = cachex.NewAllocationOptimizer()

		// Simulate intensive allocation operations
		for i := 0; i < b.N; i++ {
			// Use various optimized operations
			buf1 := cachex.OptimizedByteSlice(512)
			buf2 := cachex.OptimizedByteSlice(4096)
			buf3 := cachex.OptimizedByteSlice(16384)

			// Use string builder
			sb := cachex.NewOptimizedStringBuilder()
			sb.WriteString("test")
			sb.String()

			// Use optimized copy
			testData := []byte("test-data-for-memory-usage-benchmark")
			copied := cachex.OptimizedValueCopy(testData)

			// Use optimized map copy
			testMap := map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			}
			copiedMap := cachex.OptimizedMapCopy(testMap)

			// Use optimized string join
			testStrings := []string{"a", "b", "c", "d", "e"}
			joined := cachex.OptimizedStringJoin(testStrings, ":")

			// Prevent compiler optimization
			_ = buf1
			_ = buf2
			_ = buf3
			_ = copied
			_ = copiedMap
			_ = joined
		}
	})
}
