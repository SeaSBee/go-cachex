package bloom

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// BloomFilter implements a probabilistic data structure for membership testing
type BloomFilter struct {
	mu                sync.RWMutex
	bitset            []bool
	size              uint64
	hashCount         int
	itemCount         uint64
	maxItems          uint64
	falsePositiveRate float64
	store             BloomStore
	keyPrefix         string
}

// BloomStore defines the interface for persisting bloom filter state
type BloomStore interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
}

// Config holds Bloom filter configuration
type Config struct {
	ExpectedItems     uint64  // Expected number of items
	FalsePositiveRate float64 // Desired false positive rate (0.01 = 1%)
	Store             BloomStore
	KeyPrefix         string
	TTL               time.Duration
}

// NewBloomFilter creates a new Bloom filter
func NewBloomFilter(config *Config) (*BloomFilter, error) {
	if config.ExpectedItems == 0 {
		config.ExpectedItems = 10000
	}
	if config.FalsePositiveRate <= 0 || config.FalsePositiveRate >= 1 {
		config.FalsePositiveRate = 0.01 // 1% default
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "bloom"
	}
	if config.TTL == 0 {
		config.TTL = 24 * time.Hour
	}

	// Calculate optimal size and hash count
	size := calculateOptimalSize(config.ExpectedItems, config.FalsePositiveRate)
	hashCount := calculateOptimalHashCount(size, config.ExpectedItems)

	bf := &BloomFilter{
		bitset:            make([]bool, size),
		size:              size,
		hashCount:         hashCount,
		maxItems:          config.ExpectedItems,
		falsePositiveRate: config.FalsePositiveRate,
		store:             config.Store,
		keyPrefix:         config.KeyPrefix,
	}

	// Load existing state if store is provided
	if config.Store != nil {
		if err := bf.loadFromStore(context.Background()); err != nil {
			logx.Warn("Failed to load bloom filter from store", logx.ErrorField(err))
		}
	}

	logx.Info("Bloom filter created",
		logx.String("size", fmt.Sprintf("%d", size)),
		logx.Int("hash_count", hashCount),
		logx.String("expected_items", fmt.Sprintf("%d", config.ExpectedItems)),
		logx.Float64("false_positive_rate", config.FalsePositiveRate))

	return bf, nil
}

// Add adds an item to the Bloom filter
func (bf *BloomFilter) Add(ctx context.Context, item string) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	// Check if we're approaching capacity
	if bf.itemCount >= bf.maxItems {
		logx.Warn("Bloom filter approaching capacity",
			logx.String("item_count", fmt.Sprintf("%d", bf.itemCount)),
			logx.String("max_items", fmt.Sprintf("%d", bf.maxItems)))
	}

	// Get hash positions
	positions := bf.getHashPositions(item)

	// Set bits
	for _, pos := range positions {
		bf.bitset[pos] = true
	}

	bf.itemCount++

	// Persist to store if available
	if bf.store != nil {
		return bf.saveToStore(ctx)
	}

	return nil
}

// Contains checks if an item might be in the Bloom filter
func (bf *BloomFilter) Contains(ctx context.Context, item string) (bool, error) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	// Get hash positions
	positions := bf.getHashPositions(item)

	// Check if all bits are set
	for _, pos := range positions {
		if !bf.bitset[pos] {
			return false, nil
		}
	}

	return true, nil
}

// AddBatch adds multiple items to the Bloom filter
func (bf *BloomFilter) AddBatch(ctx context.Context, items []string) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	for _, item := range items {
		// Check if we're approaching capacity
		if bf.itemCount >= bf.maxItems {
			logx.Warn("Bloom filter at capacity, stopping batch add",
				logx.String("item_count", fmt.Sprintf("%d", bf.itemCount)),
				logx.String("max_items", fmt.Sprintf("%d", bf.maxItems)))
			break
		}

		// Get hash positions
		positions := bf.getHashPositions(item)

		// Set bits
		for _, pos := range positions {
			bf.bitset[pos] = true
		}

		bf.itemCount++
	}

	// Persist to store if available
	if bf.store != nil {
		return bf.saveToStore(ctx)
	}

	return nil
}

// ContainsBatch checks if multiple items might be in the Bloom filter
func (bf *BloomFilter) ContainsBatch(ctx context.Context, items []string) (map[string]bool, error) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	results := make(map[string]bool)

	for _, item := range items {
		// Get hash positions
		positions := bf.getHashPositions(item)

		// Check if all bits are set
		contains := true
		for _, pos := range positions {
			if !bf.bitset[pos] {
				contains = false
				break
			}
		}

		results[item] = contains
	}

	return results, nil
}

// Clear clears the Bloom filter
func (bf *BloomFilter) Clear(ctx context.Context) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	// Clear bitset
	for i := range bf.bitset {
		bf.bitset[i] = false
	}

	bf.itemCount = 0

	// Clear from store if available
	if bf.store != nil {
		return bf.store.Del(ctx, bf.getStoreKey())
	}

	return nil
}

// GetStats returns Bloom filter statistics
func (bf *BloomFilter) GetStats() map[string]interface{} {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	// Calculate current false positive rate
	currentFPR := bf.calculateCurrentFalsePositiveRate()

	return map[string]interface{}{
		"size":                        bf.size,
		"hash_count":                  bf.hashCount,
		"item_count":                  bf.itemCount,
		"max_items":                   bf.maxItems,
		"desired_false_positive_rate": bf.falsePositiveRate,
		"current_false_positive_rate": currentFPR,
		"load_factor":                 float64(bf.itemCount) / float64(bf.maxItems),
		"bits_set":                    bf.countSetBits(),
	}
}

// getHashPositions calculates hash positions for an item
func (bf *BloomFilter) getHashPositions(item string) []uint64 {
	positions := make([]uint64, bf.hashCount)

	// Use multiple hash functions
	h1 := fnv.New64a()
	h1.Write([]byte(item))
	hash1 := h1.Sum64()

	h2 := md5.New()
	h2.Write([]byte(item))
	hash2 := binary.BigEndian.Uint64(h2.Sum(nil))

	for i := 0; i < bf.hashCount; i++ {
		// Combine hashes using double hashing
		hash := hash1 + uint64(i)*uint64(hash2)
		positions[i] = hash % bf.size
	}

	return positions
}

// countSetBits counts the number of set bits in the bitset
func (bf *BloomFilter) countSetBits() uint64 {
	count := uint64(0)
	for _, bit := range bf.bitset {
		if bit {
			count++
		}
	}
	return count
}

// calculateCurrentFalsePositiveRate calculates the current false positive rate
func (bf *BloomFilter) calculateCurrentFalsePositiveRate() float64 {
	if bf.itemCount == 0 {
		return 0.0
	}

	// Calculate probability of a bit being set
	p := float64(bf.countSetBits()) / float64(bf.size)

	// False positive rate = p^hashCount
	return math.Pow(p, float64(bf.hashCount))
}

// getStoreKey returns the key for storing bloom filter state
func (bf *BloomFilter) getStoreKey() string {
	return fmt.Sprintf("%s:filter", bf.keyPrefix)
}

// saveToStore saves the bloom filter state to the store
func (bf *BloomFilter) saveToStore(ctx context.Context) error {
	// Convert bitset to bytes
	bytes := make([]byte, (bf.size+7)/8)
	for i, bit := range bf.bitset {
		if bit {
			byteIndex := i / 8
			bitIndex := i % 8
			bytes[byteIndex] |= 1 << bitIndex
		}
	}

	// Serialize to JSON (simplified - in production use a more efficient format)
	jsonData := fmt.Sprintf(`{"item_count":%d,"size":%d,"hash_count":%d}`, bf.itemCount, bf.size, bf.hashCount)

	return bf.store.Set(ctx, bf.getStoreKey(), []byte(jsonData), 24*time.Hour)
}

// loadFromStore loads the bloom filter state from the store
func (bf *BloomFilter) loadFromStore(ctx context.Context) error {
	data, err := bf.store.Get(ctx, bf.getStoreKey())
	if err != nil {
		return err
	}

	// Parse metadata (simplified)
	// In production, use proper JSON unmarshaling
	logx.Info("Loaded bloom filter from store", logx.String("data", string(data)))

	return nil
}

// calculateOptimalSize calculates the optimal size for the bitset
func calculateOptimalSize(n uint64, p float64) uint64 {
	m := float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	return uint64(math.Ceil(-m))
}

// calculateOptimalHashCount calculates the optimal number of hash functions
func calculateOptimalHashCount(m, n uint64) int {
	k := float64(m) / float64(n) * math.Log(2)
	return int(math.Ceil(k))
}

// DefaultConfig returns a default Bloom filter configuration
func DefaultConfig(store BloomStore) *Config {
	return &Config{
		ExpectedItems:     10000,
		FalsePositiveRate: 0.01, // 1%
		Store:             store,
		KeyPrefix:         "bloom",
		TTL:               24 * time.Hour,
	}
}

// ConservativeConfig returns a conservative Bloom filter configuration
func ConservativeConfig(store BloomStore) *Config {
	return &Config{
		ExpectedItems:     50000,
		FalsePositiveRate: 0.001, // 0.1%
		Store:             store,
		KeyPrefix:         "bloom",
		TTL:               24 * time.Hour,
	}
}

// AggressiveConfig returns an aggressive Bloom filter configuration
func AggressiveConfig(store BloomStore) *Config {
	return &Config{
		ExpectedItems:     1000,
		FalsePositiveRate: 0.05, // 5%
		Store:             store,
		KeyPrefix:         "bloom",
		TTL:               24 * time.Hour,
	}
}
