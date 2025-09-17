package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

// BenchmarkMemoryPoolOptimization compares performance with and without memory pools
func BenchmarkMemoryPoolOptimization(b *testing.B) {
	// Create memory store with default configuration
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

	b.Run("Set_WithMemoryPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%1000))
			value := []byte("value" + string(rune(i%1000)))

			result := store.Set(ctx, key, value, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Set operation timed out")
			}
		}
	})

	b.Run("Get_WithMemoryPool", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			value := []byte("value" + string(rune(i)))
			result := store.Set(ctx, key, value, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Set operation timed out")
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "key" + string(rune(i%1000))
			result := store.Get(ctx, key)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Get operation timed out")
			}
		}
	})

	b.Run("MGet_WithMemoryPool", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			value := []byte("value" + string(rune(i)))
			result := store.Set(ctx, key, value, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Set operation timed out")
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = "key" + string(rune((i+j)%1000))
			}

			result := store.MGet(ctx, keys...)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("MGet operation timed out")
			}
		}
	})

	b.Run("MSet_WithMemoryPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			items := make(map[string][]byte, 10)
			for j := 0; j < 10; j++ {
				key := "key" + string(rune((i+j)%1000))
				value := []byte("value" + string(rune((i+j)%1000)))
				items[key] = value
			}

			result := store.MSet(ctx, items, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("MSet operation timed out")
			}
		}
	})

	b.Run("Del_WithMemoryPool", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			value := []byte("value" + string(rune(i)))
			result := store.Set(ctx, key, value, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Set operation timed out")
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = "key" + string(rune((i+j)%1000))
			}

			result := store.Del(ctx, keys...)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Del operation timed out")
			}
		}
	})
}

// BenchmarkMemoryPoolAllocation compares allocation patterns
func BenchmarkMemoryPoolAllocation(b *testing.B) {
	b.Run("CacheItem_Allocation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate cacheItem allocation without pool
			// Since cacheItem is not exported, we'll simulate the allocation
			value := []byte("test")
			expiresAt := time.Now().Add(5 * time.Minute)
			accessCount := int64(1)
			lastAccess := time.Now()
			size := 4
			_ = value
			_ = expiresAt
			_ = accessCount
			_ = lastAccess
			_ = size
		}
	})

	b.Run("CacheItem_Pooled", func(b *testing.B) {
		pool := cachex.NewCacheItemPool()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			item := pool.Get()
			item.Value = []byte("test")
			item.ExpiresAt = time.Now().Add(5 * time.Minute)
			item.AccessCount = 1
			item.LastAccess = time.Now()
			item.Size = 4
			pool.Put(item)
		}
	})

	b.Run("Map_Allocation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate map allocation without pool
			m := make(map[string][]byte, 16)
			m["key1"] = []byte("value1")
			m["key2"] = []byte("value2")
			_ = m
		}
	})

	b.Run("Map_Pooled", func(b *testing.B) {
		pool := cachex.NewMapPool()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m := pool.Get()
			m["key1"] = []byte("value1")
			m["key2"] = []byte("value2")
			pool.Put(m)
		}
	})

	b.Run("Buffer_Allocation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate buffer allocation without pool
			buf := make([]byte, 0, 1024)
			buf = append(buf, "test"...)
			_ = buf
		}
	})

	b.Run("Buffer_Pooled", func(b *testing.B) {
		pool := cachex.NewBufferPool()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			buf = append(buf, "test"...)
			pool.Put(&buf)
		}
	})
}

// BenchmarkConcurrentMemoryPool tests concurrent access to memory pools
func BenchmarkConcurrentMemoryPool(b *testing.B) {
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

	b.Run("Concurrent_Set", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key" + string(rune(i%1000))
				value := []byte("value" + string(rune(i%1000)))

				result := store.Set(ctx, key, value, 5*time.Minute)
				select {
				case <-result:
				case <-time.After(100 * time.Millisecond):
					b.Fatal("Set operation timed out")
				}
				i++
			}
		})
	})

	b.Run("Concurrent_Get", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			value := []byte("value" + string(rune(i)))
			result := store.Set(ctx, key, value, 5*time.Minute)
			select {
			case <-result:
			case <-time.After(100 * time.Millisecond):
				b.Fatal("Set operation timed out")
			}
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key" + string(rune(i%1000))
				result := store.Get(ctx, key)
				select {
				case <-result:
				case <-time.After(100 * time.Millisecond):
					b.Fatal("Get operation timed out")
				}
				i++
			}
		})
	})
}
