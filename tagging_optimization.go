package cachex

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seasbee/go-logx"
)

// OptimizedTagManager provides high-performance tag management with optimized data structures
// and reduced allocation overhead
type OptimizedTagManager struct {
	// Optimized tag to keys mapping using sync.Map for better concurrency
	tagToKeys sync.Map
	// Optimized key to tags mapping using sync.Map for better concurrency
	keyToTags sync.Map
	// Store for persistence
	store Store
	// Configuration
	config *OptimizedTagConfig
	// Initialization flag
	initialized bool
	// Metrics with atomic operations
	metrics *OptimizedTagMetrics
	// Batch operation buffer
	batchBuffer *TagBatchBuffer
	// Persistence buffer for reduced I/O
	persistenceBuffer *PersistenceBuffer
	// Background worker for persistence
	persistenceWorker *PersistenceWorker
	// Memory usage tracking
	memoryUsage int64
	// Stop channel for graceful shutdown
	stopChan chan struct{}
	// Wait group for background operations
	wg sync.WaitGroup
}

// OptimizedTagConfig provides enhanced configuration for optimized tag manager
type OptimizedTagConfig struct {
	// Enable persistence of tag mappings
	EnablePersistence bool `yaml:"enable_persistence" json:"enable_persistence"`
	// TTL for tag mappings
	TagMappingTTL time.Duration `yaml:"tag_mapping_ttl" json:"tag_mapping_ttl"`
	// Batch size for tag operations
	BatchSize int `yaml:"batch_size" json:"batch_size"`
	// Enable statistics
	EnableStats bool `yaml:"enable_stats" json:"enable_stats"`
	// Load timeout for tag mappings
	LoadTimeout time.Duration `yaml:"load_timeout" json:"load_timeout"`
	// Enable background persistence
	EnableBackgroundPersistence bool `yaml:"enable_background_persistence" json:"enable_background_persistence"`
	// Persistence interval for background operations
	PersistenceInterval time.Duration `yaml:"persistence_interval" json:"persistence_interval"`
	// Enable memory optimization
	EnableMemoryOptimization bool `yaml:"enable_memory_optimization" json:"enable_memory_optimization"`
	// Maximum memory usage for tag mappings (in bytes)
	MaxMemoryUsage int64 `yaml:"max_memory_usage" json:"max_memory_usage"`
	// Batch flush interval
	BatchFlushInterval time.Duration `yaml:"batch_flush_interval" json:"batch_flush_interval"`
}

// DefaultOptimizedTagConfig returns optimized default configuration
func DefaultOptimizedTagConfig() *OptimizedTagConfig {
	return &OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               24 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: true,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    true,
		MaxMemoryUsage:              100 * 1024 * 1024, // 100MB
		BatchFlushInterval:          5 * time.Second,
	}
}

// OptimizedTagMetrics provides atomic metrics for better performance
type OptimizedTagMetrics struct {
	AddTagsCount         int64
	RemoveTagsCount      int64
	InvalidateByTagCount int64
	RemoveKeyCount       int64
	PersistCount         int64
	LoadCount            int64
	ErrorsCount          int64
	CacheHits            int64
	CacheMisses          int64
	BatchOperations      int64
	MemoryUsage          int64
}

// TagBatchBuffer provides efficient batch operation handling
type TagBatchBuffer struct {
	mu           sync.Mutex
	addBuffer    map[string][]string // key -> tags to add
	removeBuffer map[string][]string // key -> tags to remove
	flushSize    int
	flushTimer   *time.Timer
	stopChan     chan struct{}
}

// NewTagBatchBuffer creates a new batch buffer
func NewTagBatchBuffer(flushSize int, flushInterval time.Duration) *TagBatchBuffer {
	tbb := &TagBatchBuffer{
		addBuffer:    make(map[string][]string),
		removeBuffer: make(map[string][]string),
		flushSize:    flushSize,
		stopChan:     make(chan struct{}),
	}

	// Start timer for periodic flushing
	tbb.flushTimer = time.AfterFunc(flushInterval, func() {
		select {
		case <-tbb.stopChan:
			return
		default:
			// Timer will be reset in FlushBuffer if needed
		}
	})

	return tbb
}

// AddToBuffer adds tags to the batch buffer
func (tbb *TagBatchBuffer) AddToBuffer(key string, tags []string) {
	tbb.mu.Lock()
	defer tbb.mu.Unlock()

	if tbb.addBuffer[key] == nil {
		tbb.addBuffer[key] = make([]string, 0, len(tags))
	}
	tbb.addBuffer[key] = append(tbb.addBuffer[key], tags...)
}

// RemoveFromBuffer removes tags from the batch buffer
func (tbb *TagBatchBuffer) RemoveFromBuffer(key string, tags []string) {
	tbb.mu.Lock()
	defer tbb.mu.Unlock()

	if tbb.removeBuffer[key] == nil {
		tbb.removeBuffer[key] = make([]string, 0, len(tags))
	}
	tbb.removeBuffer[key] = append(tbb.removeBuffer[key], tags...)
}

// FlushBuffer flushes the batch buffer and returns operations
func (tbb *TagBatchBuffer) FlushBuffer() (map[string][]string, map[string][]string) {
	tbb.mu.Lock()
	defer tbb.mu.Unlock()

	addOps := make(map[string][]string, len(tbb.addBuffer))
	removeOps := make(map[string][]string, len(tbb.removeBuffer))

	for k, v := range tbb.addBuffer {
		addOps[k] = make([]string, len(v))
		copy(addOps[k], v)
	}
	for k, v := range tbb.removeBuffer {
		removeOps[k] = make([]string, len(v))
		copy(removeOps[k], v)
	}

	// Clear buffers
	tbb.addBuffer = make(map[string][]string)
	tbb.removeBuffer = make(map[string][]string)

	// Reset timer for next flush using the configured interval
	if tbb.flushTimer != nil {
		tbb.flushTimer.Reset(tbb.getFlushInterval())
	}

	return addOps, removeOps
}

// getFlushInterval returns the configured flush interval or a default
func (tbb *TagBatchBuffer) getFlushInterval() time.Duration {
	// Use a reasonable default if not configured
	if tbb.flushSize <= 0 {
		return 5 * time.Second
	}
	// Calculate interval based on batch size for better performance
	// Smaller batches get shorter intervals, larger batches get longer intervals
	if tbb.flushSize <= 10 {
		return 1 * time.Second
	} else if tbb.flushSize <= 50 {
		return 3 * time.Second
	} else if tbb.flushSize <= 100 {
		return 5 * time.Second
	} else {
		return 10 * time.Second
	}
}

// Stop stops the batch buffer
func (tbb *TagBatchBuffer) Stop() {
	close(tbb.stopChan)
	if tbb.flushTimer != nil {
		tbb.flushTimer.Stop()
	}
}

// PersistenceBuffer provides efficient persistence operations
type PersistenceBuffer struct {
	mu       sync.Mutex
	data     []byte
	dirty    bool
	lastSave time.Time
}

// NewPersistenceBuffer creates a new persistence buffer
func NewPersistenceBuffer() *PersistenceBuffer {
	return &PersistenceBuffer{
		data:  make([]byte, 0, 1024),
		dirty: false,
	}
}

// SetData sets the buffer data and marks as dirty
func (pb *PersistenceBuffer) SetData(data []byte) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.data = make([]byte, len(data))
	copy(pb.data, data)
	pb.dirty = true
}

// GetData returns the buffer data and clears dirty flag
func (pb *PersistenceBuffer) GetData() ([]byte, bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if !pb.dirty {
		return nil, false
	}

	data := make([]byte, len(pb.data))
	copy(data, pb.data)
	pb.dirty = false
	pb.lastSave = time.Now()

	return data, true
}

// PersistenceWorker handles background persistence operations
type PersistenceWorker struct {
	stopChan chan struct{}
	wg       sync.WaitGroup
	interval time.Duration
	tm       *OptimizedTagManager
}

// NewPersistenceWorker creates a new persistence worker
func NewPersistenceWorker(tm *OptimizedTagManager, interval time.Duration) *PersistenceWorker {
	return &PersistenceWorker{
		stopChan: make(chan struct{}),
		interval: interval,
		tm:       tm,
	}
}

// Start starts the persistence worker
func (pw *PersistenceWorker) Start() {
	pw.wg.Add(1)
	go pw.run()
}

// Stop stops the persistence worker
func (pw *PersistenceWorker) Stop() {
	close(pw.stopChan)
	pw.wg.Wait()
}

// run runs the persistence worker loop
func (pw *PersistenceWorker) run() {
	defer pw.wg.Done()

	ticker := time.NewTicker(pw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-pw.stopChan:
			return
		case <-ticker.C:
			if err := pw.flushPersistence(); err != nil {
				logx.Error("Background persistence failed", logx.ErrorField(err))
				atomic.AddInt64(&pw.tm.metrics.ErrorsCount, 1)
			}
		}
	}
}

// flushPersistence flushes pending persistence operations
func (pw *PersistenceWorker) flushPersistence() error {
	if data, dirty := pw.tm.persistenceBuffer.GetData(); dirty {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		key := "cachex:optimized_tag_mappings"
		result := <-pw.tm.store.Set(ctx, key, data, pw.tm.config.TagMappingTTL)
		if result.Error != nil {
			return fmt.Errorf("background persistence failed: %w", result.Error)
		}
		atomic.AddInt64(&pw.tm.metrics.PersistCount, 1)
	}
	return nil
}

// NewOptimizedTagManager creates a new optimized tag manager
func NewOptimizedTagManager(store Store, config *OptimizedTagConfig) (*OptimizedTagManager, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	if config == nil {
		config = DefaultOptimizedTagConfig()
	}

	tm := &OptimizedTagManager{
		store:             store,
		config:            config,
		initialized:       false,
		metrics:           &OptimizedTagMetrics{},
		batchBuffer:       NewTagBatchBuffer(config.BatchSize, config.BatchFlushInterval),
		persistenceBuffer: NewPersistenceBuffer(),
		stopChan:          make(chan struct{}),
	}

	// Load existing tag mappings if persistence is enabled
	if config.EnablePersistence {
		if err := tm.loadTagMappings(); err != nil {
			logx.Warn("Failed to load tag mappings during initialization", logx.ErrorField(err))
			atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		}
	}

	// Start background persistence worker if enabled
	if config.EnableBackgroundPersistence {
		tm.persistenceWorker = NewPersistenceWorker(tm, config.PersistenceInterval)
		tm.persistenceWorker.Start()
	}

	// Start background batch processor
	tm.wg.Add(1)
	go tm.batchProcessor()

	tm.initialized = true
	return tm, nil
}

// batchProcessor handles periodic batch processing
func (tm *OptimizedTagManager) batchProcessor() {
	defer tm.wg.Done()

	ticker := time.NewTicker(tm.config.BatchFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stopChan:
			// Final flush before stopping
			tm.processBatchIfNeeded(context.Background())
			return
		case <-ticker.C:
			tm.processBatchIfNeeded(context.Background())
		}
	}
}

// AddTags adds tags to a key with optimized batch processing
func (tm *OptimizedTagManager) AddTags(ctx context.Context, key string, tags ...string) error {
	if err := tm.validateInputs(ctx, key, tags...); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return err
	}

	// Check memory usage before adding
	if tm.config.EnableMemoryOptimization {
		if err := tm.checkMemoryUsage(len(tags)); err != nil {
			return err
		}
	}

	// Optimized batch processing logic for high-throughput scenarios
	// For very small operations, use direct processing to avoid batch overhead
	// For medium operations, use batch buffer
	// For large operations, use direct processing to avoid memory pressure
	if len(tags) <= 5 {
		// Very small operations: direct processing for immediate response
		return tm.addTagsDirect(ctx, key, tags)
	} else if len(tags) <= tm.config.BatchSize {
		// Medium operations: use batch buffer for efficiency
		tm.batchBuffer.AddToBuffer(key, tags)
		// Process batch immediately to ensure tags are added
		if err := tm.processBatchIfNeeded(ctx); err != nil {
			return err
		}
		atomic.AddInt64(&tm.metrics.BatchOperations, 1)
		return nil
	} else {
		// Large operations: direct processing to avoid memory pressure
		return tm.addTagsDirect(ctx, key, tags)
	}
}

// RemoveTags removes tags from a key with optimized batch processing
func (tm *OptimizedTagManager) RemoveTags(ctx context.Context, key string, tags ...string) error {
	if err := tm.validateInputs(ctx, key, tags...); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return err
	}

	// Optimized batch processing logic for high-throughput scenarios
	// For very small operations, use direct processing to avoid batch overhead
	// For medium operations, use batch buffer
	// For large operations, use direct processing to avoid memory pressure
	if len(tags) <= 5 {
		// Very small operations: direct processing for immediate response
		return tm.removeTagsDirect(ctx, key, tags)
	} else if len(tags) <= tm.config.BatchSize {
		// Medium operations: use batch buffer for efficiency
		tm.batchBuffer.RemoveFromBuffer(key, tags)
		return nil
	} else {
		// Large operations: direct processing to avoid memory pressure
		return tm.removeTagsDirect(ctx, key, tags)
	}
}

// GetKeysByTag returns keys for a tag with optimized caching
func (tm *OptimizedTagManager) GetKeysByTag(ctx context.Context, tag string) ([]string, error) {
	if err := tm.validateTag(tag); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return nil, err
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Try to get from sync.Map
	if keysInterface, exists := tm.tagToKeys.Load(tag); exists && keysInterface != nil {
		if keys, ok := keysInterface.(map[string]bool); ok {
			atomic.AddInt64(&tm.metrics.CacheHits, 1)
			return tm.mapKeysToSlice(keys), nil
		}
	}

	atomic.AddInt64(&tm.metrics.CacheMisses, 1)
	return []string{}, nil
}

// GetTagsByKey returns tags for a key with optimized caching
func (tm *OptimizedTagManager) GetTagsByKey(ctx context.Context, key string) ([]string, error) {
	if err := tm.validateKey(key); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return nil, err
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Try to get from sync.Map
	if tagsInterface, exists := tm.keyToTags.Load(key); exists && tagsInterface != nil {
		if tags, ok := tagsInterface.(map[string]bool); ok {
			atomic.AddInt64(&tm.metrics.CacheHits, 1)
			return tm.mapKeysToSlice(tags), nil
		}
	}

	atomic.AddInt64(&tm.metrics.CacheMisses, 1)
	return []string{}, nil
}

// InvalidateByTag invalidates keys by tag with optimized batch processing
func (tm *OptimizedTagManager) InvalidateByTag(ctx context.Context, tags ...string) error {
	if err := tm.validateTags(tags...); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return err
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Collect all keys to invalidate
	keysToInvalidate := make(map[string]bool)
	for _, tag := range tags {
		if keysInterface, exists := tm.tagToKeys.Load(tag); exists && keysInterface != nil {
			if keys, ok := keysInterface.(map[string]bool); ok {
				for key := range keys {
					keysToInvalidate[key] = true
				}
			}
		}
	}

	if len(keysToInvalidate) == 0 {
		return nil
	}

	// Convert to slice
	keys := tm.mapKeysToSlice(keysToInvalidate)

	// Delete keys in batches
	err := tm.deleteKeysInBatches(ctx, keys)
	if err != nil {
		return err
	}

	// Clean up tag mappings for invalidated keys
	for key := range keysToInvalidate {
		// Remove key from all tag mappings
		if tagsInterface, exists := tm.keyToTags.Load(key); exists && tagsInterface != nil {
			if keyTags, ok := tagsInterface.(map[string]bool); ok {
				for tag := range keyTags {
					tm.removeKeyFromTag(tag, key)
				}
			}
		}
		// Remove key-to-tags mapping
		tm.keyToTags.Delete(key)
	}

	return nil
}

// RemoveKey removes a key and all its tag associations
func (tm *OptimizedTagManager) RemoveKey(ctx context.Context, key string) error {
	if err := tm.validateKey(key); err != nil {
		atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
		return err
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get tags for this key
	var tags []string
	if tagsInterface, exists := tm.keyToTags.Load(key); exists && tagsInterface != nil {
		if tagMap, ok := tagsInterface.(map[string]bool); ok {
			tags = tm.mapKeysToSlice(tagMap)
		}
	}

	// Remove from tag mappings using atomic operations
	for _, tag := range tags {
		tm.removeKeyFromTag(tag, key)
	}

	// Remove key mapping
	tm.keyToTags.Delete(key)

	atomic.AddInt64(&tm.metrics.RemoveKeyCount, 1)

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// GetStats returns optimized tag statistics
func (tm *OptimizedTagManager) GetStats() (*OptimizedTagStats, error) {
	if !tm.initialized {
		return nil, fmt.Errorf("tag manager not initialized")
	}

	// Update memory usage with actual calculation if stats are enabled
	if tm.config.EnableStats {
		tm.updateMemoryUsage()
	}

	stats := &OptimizedTagStats{
		TotalTags:       tm.countMapEntries(&tm.tagToKeys),
		TotalKeys:       tm.countMapEntries(&tm.keyToTags),
		AddTagsCount:    atomic.LoadInt64(&tm.metrics.AddTagsCount),
		RemoveTagsCount: atomic.LoadInt64(&tm.metrics.RemoveTagsCount),
		InvalidateCount: atomic.LoadInt64(&tm.metrics.InvalidateByTagCount),
		RemoveKeyCount:  atomic.LoadInt64(&tm.metrics.RemoveKeyCount),
		CacheHits:       atomic.LoadInt64(&tm.metrics.CacheHits),
		CacheMisses:     atomic.LoadInt64(&tm.metrics.CacheMisses),
		BatchOperations: atomic.LoadInt64(&tm.metrics.BatchOperations),
		MemoryUsage:     atomic.LoadInt64(&tm.memoryUsage),
	}

	return stats, nil
}

// OptimizedTagStats provides comprehensive statistics
type OptimizedTagStats struct {
	TotalTags       int64
	TotalKeys       int64
	AddTagsCount    int64
	RemoveTagsCount int64
	InvalidateCount int64
	RemoveKeyCount  int64
	CacheHits       int64
	CacheMisses     int64
	BatchOperations int64
	MemoryUsage     int64
}

// Close closes the optimized tag manager
func (tm *OptimizedTagManager) Close() error {
	// Signal shutdown
	close(tm.stopChan)

	// Stop batch buffer
	if tm.batchBuffer != nil {
		tm.batchBuffer.Stop()
	}

	// Stop persistence worker
	if tm.persistenceWorker != nil {
		tm.persistenceWorker.Stop()
	}

	// Wait for background operations to complete
	tm.wg.Wait()

	// Final persistence flush
	if tm.config.EnablePersistence {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// Helper methods for optimized operations

func (tm *OptimizedTagManager) validateInputs(ctx context.Context, key string, tags ...string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if !tm.initialized {
		return fmt.Errorf("tag manager not initialized")
	}
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	for _, tag := range tags {
		if tag == "" {
			return fmt.Errorf("tag cannot be empty")
		}
	}
	return nil
}

func (tm *OptimizedTagManager) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}

func (tm *OptimizedTagManager) validateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	return nil
}

func (tm *OptimizedTagManager) validateTags(tags ...string) error {
	for _, tag := range tags {
		if tag == "" {
			return fmt.Errorf("tag cannot be empty")
		}
	}
	return nil
}

func (tm *OptimizedTagManager) checkMemoryUsage(additionalSize int) error {
	if !tm.config.EnableMemoryOptimization {
		return nil // Memory optimization disabled
	}

	currentUsage := atomic.LoadInt64(&tm.memoryUsage)

	// Additional overhead for sync.Map and other structures
	estimatedAdditionalUsage := int64(additionalSize * 250) // Increased from 175 to 250 bytes per item
	estimatedUsage := currentUsage + estimatedAdditionalUsage

	if estimatedUsage > tm.config.MaxMemoryUsage {
		return fmt.Errorf("memory usage limit exceeded: estimated %d bytes would exceed limit of %d bytes", estimatedUsage, tm.config.MaxMemoryUsage)
	}

	return nil
}

// updateMemoryUsage updates the memory usage based on actual data structures
func (tm *OptimizedTagManager) updateMemoryUsage() {
	var totalUsage int64

	// Calculate memory usage from tag-to-keys mapping
	tm.tagToKeys.Range(func(key, value interface{}) bool {
		if key != nil && value != nil {
			// Key string memory
			if keyStr, ok := key.(string); ok {
				totalUsage += int64(len(keyStr))
			}

			// Value map memory
			if keys, ok := value.(map[string]bool); ok {
				totalUsage += int64(len(keys) * 100) // Rough estimate for map overhead
				for k := range keys {
					totalUsage += int64(len(k))
				}
			}
		}
		return true
	})

	// Calculate memory usage from key-to-tags mapping
	tm.keyToTags.Range(func(key, value interface{}) bool {
		if key != nil && value != nil {
			// Key string memory
			if keyStr, ok := key.(string); ok {
				totalUsage += int64(len(keyStr))
			}

			// Value map memory
			if tags, ok := value.(map[string]bool); ok {
				totalUsage += int64(len(tags) * 100) // Rough estimate for map overhead
				for t := range tags {
					totalUsage += int64(len(t))
				}
			}
		}
		return true
	})

	// Add overhead for sync.Map structures
	totalUsage += 1024 // Base overhead for sync.Map

	atomic.StoreInt64(&tm.memoryUsage, totalUsage)
}

func (tm *OptimizedTagManager) addTagsDirect(ctx context.Context, key string, tags []string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get or create key-to-tags mapping with atomic operation
	var keyTags map[string]bool
	if keyTagsInterface, exists := tm.keyToTags.Load(key); exists && keyTagsInterface != nil {
		if existingTags, ok := keyTagsInterface.(map[string]bool); ok {
			// Create a copy to avoid race conditions
			keyTags = make(map[string]bool, len(existingTags)+len(tags))
			for k, v := range existingTags {
				keyTags[k] = v
			}
		}
	}
	if keyTags == nil {
		keyTags = make(map[string]bool)
	}

	// Add tags to key
	for _, tag := range tags {
		keyTags[tag] = true

		// Get or create tag-to-keys mapping with atomic operation
		var tagKeys map[string]bool
		if tagKeysInterface, exists := tm.tagToKeys.Load(tag); exists && tagKeysInterface != nil {
			if existingKeys, ok := tagKeysInterface.(map[string]bool); ok {
				// Create a copy to avoid race conditions
				tagKeys = make(map[string]bool, len(existingKeys)+1)
				for k, v := range existingKeys {
					tagKeys[k] = v
				}
			}
		}
		if tagKeys == nil {
			tagKeys = make(map[string]bool)
		}

		tagKeys[key] = true
		tm.tagToKeys.Store(tag, tagKeys)
	}

	tm.keyToTags.Store(key, keyTags)
	atomic.AddInt64(&tm.metrics.AddTagsCount, 1)

	// Update memory usage after adding tags
	if tm.config.EnableMemoryOptimization {
		// Calculate actual memory usage based on what was added
		actualUsage := int64(len(tags) * 175) // 50 + 100 + 25 bytes per item
		atomic.AddInt64(&tm.memoryUsage, actualUsage)
	}

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

func (tm *OptimizedTagManager) removeTagsDirect(ctx context.Context, key string, tags []string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get key-to-tags mapping
	if keyTagsInterface, exists := tm.keyToTags.Load(key); exists && keyTagsInterface != nil {
		if existingTags, ok := keyTagsInterface.(map[string]bool); ok {
			// Create a copy to avoid race conditions
			keyTags := make(map[string]bool, len(existingTags))
			for k, v := range existingTags {
				keyTags[k] = v
			}

			// Remove tags from key
			for _, tag := range tags {
				delete(keyTags, tag)
				tm.removeKeyFromTag(tag, key)
			}

			// Update or remove key mapping
			if len(keyTags) == 0 {
				tm.keyToTags.Delete(key)
			} else {
				tm.keyToTags.Store(key, keyTags)
			}
		}
	}

	atomic.AddInt64(&tm.metrics.RemoveTagsCount, 1)

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

func (tm *OptimizedTagManager) removeKeyFromTag(tag, key string) {
	if tagKeysInterface, exists := tm.tagToKeys.Load(tag); exists && tagKeysInterface != nil {
		if existingKeys, ok := tagKeysInterface.(map[string]bool); ok {
			// Create a copy to avoid race conditions
			tagKeys := make(map[string]bool, len(existingKeys))
			for k, v := range existingKeys {
				tagKeys[k] = v
			}

			delete(tagKeys, key)
			if len(tagKeys) == 0 {
				tm.tagToKeys.Delete(tag)
			} else {
				tm.tagToKeys.Store(tag, tagKeys)
			}
		}
	}
}

func (tm *OptimizedTagManager) processBatchIfNeeded(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if batch buffer needs flushing
	addOps, removeOps := tm.batchBuffer.FlushBuffer()

	if len(addOps) == 0 && len(removeOps) == 0 {
		return nil
	}

	atomic.AddInt64(&tm.metrics.BatchOperations, 1)

	// Process add operations
	for key, tags := range addOps {
		if err := tm.addTagsDirect(ctx, key, tags); err != nil {
			return err
		}
	}

	// Process remove operations
	for key, tags := range removeOps {
		if err := tm.removeTagsDirect(ctx, key, tags); err != nil {
			return err
		}
	}

	return nil
}

func (tm *OptimizedTagManager) mapKeysToSlice(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}

	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func (tm *OptimizedTagManager) countMapEntries(m *sync.Map) int64 {
	var count int64
	m.Range(func(key, value interface{}) bool {
		if key != nil && value != nil {
			count++
		}
		return true
	})
	return count
}

func (tm *OptimizedTagManager) deleteKeysInBatches(ctx context.Context, keys []string) error {
	for i := 0; i < len(keys); i += tm.config.BatchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + tm.config.BatchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		result := <-tm.store.Del(ctx, batch...)
		if result.Error != nil {
			return fmt.Errorf("failed to delete keys batch: %w", result.Error)
		}
	}

	atomic.AddInt64(&tm.metrics.InvalidateByTagCount, 1)
	return nil
}

func (tm *OptimizedTagManager) persistTagMappings(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Serialize tag mappings efficiently
	tagMappings := struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		TagToKeys: make(map[string][]string),
		KeyToTags: make(map[string][]string),
		Timestamp: time.Now(),
	}

	// Convert tag-to-keys mapping with safe iteration
	tagToKeysCopy := make(map[string]map[string]bool)
	tm.tagToKeys.Range(func(key, value interface{}) bool {
		if key == nil || value == nil {
			return true
		}

		tag, ok := key.(string)
		if !ok {
			return true
		}

		if keys, ok := value.(map[string]bool); ok {
			keysCopy := make(map[string]bool, len(keys))
			for k := range keys {
				keysCopy[k] = true
			}
			tagToKeysCopy[tag] = keysCopy
		}
		return true
	})

	// Convert key-to-tags mapping with safe iteration
	keyToTagsCopy := make(map[string]map[string]bool)
	tm.keyToTags.Range(func(key, value interface{}) bool {
		if key == nil || value == nil {
			return true
		}

		k, ok := key.(string)
		if !ok {
			return true
		}

		if tags, ok := value.(map[string]bool); ok {
			tagsCopy := make(map[string]bool, len(tags))
			for t := range tags {
				tagsCopy[t] = true
			}
			keyToTagsCopy[k] = tagsCopy
		}
		return true
	})

	// Now safely convert to string slices
	for tag, keys := range tagToKeysCopy {
		keyList := tm.mapKeysToSlice(keys)
		tagMappings.TagToKeys[tag] = keyList
	}

	for key, tags := range keyToTagsCopy {
		tagList := tm.mapKeysToSlice(tags)
		tagMappings.KeyToTags[key] = tagList
	}

	// Serialize to JSON
	data, err := json.Marshal(tagMappings)
	if err != nil {
		return fmt.Errorf("failed to marshal tag mappings: %w", err)
	}

	// Use persistence buffer for background operations
	if tm.config.EnableBackgroundPersistence {
		tm.persistenceBuffer.SetData(data)
		return nil
	}

	// Direct persistence for immediate operations
	key := "cachex:optimized_tag_mappings"
	result := <-tm.store.Set(ctx, key, data, tm.config.TagMappingTTL)
	if result.Error != nil {
		return fmt.Errorf("failed to persist tag mappings: %w", result.Error)
	}

	atomic.AddInt64(&tm.metrics.PersistCount, 1)
	return nil
}

func (tm *OptimizedTagManager) loadTagMappings() error {
	ctx, cancel := context.WithTimeout(context.Background(), tm.config.LoadTimeout)
	defer cancel()

	key := "cachex:optimized_tag_mappings"
	result := <-tm.store.Get(ctx, key)
	if result.Error != nil {
		return fmt.Errorf("failed to get tag mappings from store: %w", result.Error)
	}

	if !result.Exists || result.Value == nil {
		return nil // No mappings stored
	}

	// Deserialize tag mappings
	var tagMappings struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(result.Value, &tagMappings); err != nil {
		return fmt.Errorf("failed to unmarshal tag mappings: %w", err)
	}

	// Check if data is stale
	if time.Since(tagMappings.Timestamp) > tm.config.TagMappingTTL {
		logx.Warn("Loaded tag mappings are stale, ignoring")
		return nil
	}

	// Reconstruct in-memory mappings
	for tag, keys := range tagMappings.TagToKeys {
		if tag == "" {
			continue
		}
		keyMap := make(map[string]bool)
		for _, key := range keys {
			if key != "" {
				keyMap[key] = true
			}
		}
		tm.tagToKeys.Store(tag, keyMap)
	}

	for key, tags := range tagMappings.KeyToTags {
		if key == "" {
			continue
		}
		tagMap := make(map[string]bool)
		for _, tag := range tags {
			if tag != "" {
				tagMap[tag] = true
			}
		}
		tm.keyToTags.Store(key, tagMap)
	}

	atomic.AddInt64(&tm.metrics.LoadCount, 1)
	return nil
}
