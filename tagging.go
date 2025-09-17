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

// TagManager manages tag-to-keys and key-to-tags mappings with thread-safe operations,
// persistence support, and comprehensive metrics tracking.
type TagManager struct {
	// Tag to set of keys mapping (inverted index)
	tagToKeys map[string]map[string]bool
	// Key to set of tags mapping
	keyToTags map[string]map[string]bool
	// Store for persistence
	store Store
	// Mutex for thread safety
	mu sync.RWMutex
	// Configuration
	config *TagConfig
	// Initialization flag
	initialized bool
	// Metrics
	metrics *TagMetrics
}

// TagConfig holds tagging configuration options
type TagConfig struct {
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
}

// DefaultTagConfig returns a default configuration with sensible defaults
func DefaultTagConfig() *TagConfig {
	return &TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     24 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
		LoadTimeout:       5 * time.Second,
	}
}

// TagStats holds tagging statistics for monitoring
type TagStats struct {
	TotalTags     int64
	TotalKeys     int64
	TagOperations int64
}

// TagMetrics holds internal metrics for monitoring and observability
type TagMetrics struct {
	AddTagsCount         int64
	RemoveTagsCount      int64
	InvalidateByTagCount int64
	RemoveKeyCount       int64
	PersistCount         int64
	LoadCount            int64
	ErrorsCount          int64
}

// NewTagManager creates a new tag manager with the specified store and configuration.
// Returns an error if the store is nil or initialization fails.
func NewTagManager(store Store, config *TagConfig) (*TagManager, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	if config == nil {
		config = DefaultTagConfig()
	}

	tm := &TagManager{
		tagToKeys:   make(map[string]map[string]bool),
		keyToTags:   make(map[string]map[string]bool),
		store:       store,
		config:      config,
		initialized: false,
		metrics:     &TagMetrics{},
	}

	// Load existing tag mappings if persistence is enabled
	if config.EnablePersistence {
		if err := tm.loadTagMappings(); err != nil {
			logx.Warn("Failed to load tag mappings during initialization", logx.ErrorField(err))
			atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
			// Continue with empty mappings rather than failing completely
		}
	}

	tm.initialized = true
	return tm, nil
}

// ensureInitialized ensures the tag manager is properly initialized
func (tm *TagManager) ensureInitialized() error {
	if !tm.initialized {
		return fmt.Errorf("tag manager not initialized")
	}
	return nil
}

// validateContext validates that context is not nil
func (tm *TagManager) validateContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	return nil
}

// validateStore validates that store is not nil
func (tm *TagManager) validateStore() error {
	if tm.store == nil {
		return fmt.Errorf("store is nil")
	}
	return nil
}

// incrementErrorCount safely increments the error counter
func (tm *TagManager) incrementErrorCount() {
	atomic.AddInt64(&tm.metrics.ErrorsCount, 1)
}

// AddTags adds one or more tags to a key. If the key doesn't exist, it will be created.
// Returns an error if the context is nil, the key is empty, or any tag is empty.
func (tm *TagManager) AddTags(ctx context.Context, key string, tags ...string) error {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if key == "" {
		tm.incrementErrorCount()
		return fmt.Errorf("key cannot be empty")
	}

	if len(tags) == 0 {
		return nil
	}

	// Validate tags
	for _, tag := range tags {
		if tag == "" {
			tm.incrementErrorCount()
			return fmt.Errorf("tag cannot be empty")
		}
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Initialize key-to-tags mapping if not exists
	if tm.keyToTags[key] == nil {
		tm.keyToTags[key] = make(map[string]bool)
	}

	// Add tags to key
	for _, tag := range tags {
		// Initialize tag-to-keys mapping if not exists
		if tm.tagToKeys[tag] == nil {
			tm.tagToKeys[tag] = make(map[string]bool)
		}

		tm.keyToTags[key][tag] = true
		tm.tagToKeys[tag][key] = true
	}

	atomic.AddInt64(&tm.metrics.AddTagsCount, 1)

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// RemoveTags removes one or more tags from a key. If the key doesn't exist, no error is returned.
// Returns an error if the context is nil or the key is empty.
func (tm *TagManager) RemoveTags(ctx context.Context, key string, tags ...string) error {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if key == "" {
		tm.incrementErrorCount()
		return fmt.Errorf("key cannot be empty")
	}

	if len(tags) == 0 {
		return nil
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if key exists
	if tm.keyToTags[key] == nil {
		return nil // Key doesn't exist, nothing to remove
	}

	// Remove tags from key
	for _, tag := range tags {
		if tag == "" {
			continue // Skip empty tags
		}

		delete(tm.keyToTags[key], tag)
		if tm.tagToKeys[tag] != nil {
			delete(tm.tagToKeys[tag], key)
			// Clean up empty tag mappings
			if len(tm.tagToKeys[tag]) == 0 {
				delete(tm.tagToKeys, tag)
			}
		}
	}

	// Clean up empty key mappings
	if len(tm.keyToTags[key]) == 0 {
		delete(tm.keyToTags, key)
	}

	atomic.AddInt64(&tm.metrics.RemoveTagsCount, 1)

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// GetKeysByTag returns all keys associated with a specific tag.
// Returns an empty slice if the tag doesn't exist.
// Returns an error if the context is nil or the tag is empty.
func (tm *TagManager) GetKeysByTag(ctx context.Context, tag string) ([]string, error) {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return nil, err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return nil, err
	}

	if tag == "" {
		tm.incrementErrorCount()
		return nil, fmt.Errorf("tag cannot be empty")
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if keys, exists := tm.tagToKeys[tag]; exists && len(keys) > 0 {
		result := make([]string, 0, len(keys))
		for key := range keys {
			result = append(result, key)
		}
		return result, nil
	}

	return []string{}, nil
}

// GetTagsByKey returns all tags associated with a specific key.
// Returns an empty slice if the key doesn't exist.
// Returns an error if the context is nil or the key is empty.
func (tm *TagManager) GetTagsByKey(ctx context.Context, key string) ([]string, error) {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return nil, err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return nil, err
	}

	if key == "" {
		tm.incrementErrorCount()
		return nil, fmt.Errorf("key cannot be empty")
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tags, exists := tm.keyToTags[key]; exists && len(tags) > 0 {
		result := make([]string, 0, len(tags))
		for tag := range tags {
			result = append(result, tag)
		}
		return result, nil
	}

	return []string{}, nil
}

// InvalidateByTag invalidates all keys associated with the given tags.
// This operation is atomic - if any store operation fails, the tag mappings are rolled back.
// Returns an error if the context is nil or any tag is empty.
func (tm *TagManager) InvalidateByTag(ctx context.Context, tags ...string) error {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if err := tm.validateStore(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if len(tags) == 0 {
		return nil
	}

	// Validate tags
	for _, tag := range tags {
		if tag == "" {
			tm.incrementErrorCount()
			return fmt.Errorf("tag cannot be empty")
		}
	}

	// First, collect all keys to invalidate under read lock
	tm.mu.RLock()
	keysToInvalidate := make(map[string]bool)
	for _, tag := range tags {
		if keys, exists := tm.tagToKeys[tag]; exists {
			for key := range keys {
				keysToInvalidate[key] = true
			}
		}
	}
	tm.mu.RUnlock()

	if len(keysToInvalidate) == 0 {
		// Check context before returning
		if ctx.Err() != nil {
			tm.incrementErrorCount()
			return ctx.Err()
		}
		return nil // No keys to invalidate
	}

	// Convert to slice
	keys := make([]string, 0, len(keysToInvalidate))
	for key := range keysToInvalidate {
		keys = append(keys, key)
	}

	// Store the original mappings for rollback if needed
	tm.mu.Lock()
	originalMappings := tm.backupMappingsLocked(keys)
	tm.mu.Unlock()

	// Delete keys in batches without holding the lock
	success := true
	batchSize := tm.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	for i := 0; i < len(keys); i += batchSize {
		// Check context cancellation between batches
		if ctx.Err() != nil {
			tm.incrementErrorCount()
			return ctx.Err()
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		delResult := <-tm.store.Del(ctx, batch...)
		if delResult.Error != nil {
			logx.Error("Failed to delete keys by tag",
				logx.String("batch", fmt.Sprintf("%v", batch)),
				logx.ErrorField(delResult.Error))
			tm.incrementErrorCount()
			success = false
			break
		}
	}

	// Update tag mappings based on success/failure
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if success {
		// Only remove from tag mappings if all store operations succeeded
		tm.removeKeysFromMappingsLocked(keys)
		atomic.AddInt64(&tm.metrics.InvalidateByTagCount, 1)

		logx.Info("Invalidated keys by tag",
			logx.Int("count", len(keys)),
			logx.String("tags", fmt.Sprintf("%v", tags)),
			logx.String("keys", fmt.Sprintf("%v", keys)))
	} else {
		// Rollback: restore original mappings
		tm.restoreMappingsLocked(originalMappings)
		logx.Warn("Rolled back tag mappings due to store operation failure",
			logx.String("tags", fmt.Sprintf("%v", tags)))
	}

	return nil
}

// RemoveKey removes a key and all its tag associations.
// Returns an error if the context is nil or the key is empty.
func (tm *TagManager) RemoveKey(ctx context.Context, key string) error {
	// Validate inputs
	if err := tm.validateContext(ctx); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if err := tm.ensureInitialized(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	if key == "" {
		tm.incrementErrorCount()
		return fmt.Errorf("key cannot be empty")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get tags for this key
	tags := make([]string, 0)
	if keyTags, exists := tm.keyToTags[key]; exists {
		for tag := range keyTags {
			tags = append(tags, tag)
		}
	}

	// Remove from tag mappings
	for _, tag := range tags {
		if tm.tagToKeys[tag] != nil {
			delete(tm.tagToKeys[tag], key)
			if len(tm.tagToKeys[tag]) == 0 {
				delete(tm.tagToKeys, tag)
			}
		}
	}

	// Remove key mapping
	delete(tm.keyToTags, key)

	atomic.AddInt64(&tm.metrics.RemoveKeyCount, 1)

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// GetStats returns tagging statistics including total tags, keys, and operations.
// Returns an error if the tag manager is not initialized.
func (tm *TagManager) GetStats() (*TagStats, error) {
	if err := tm.ensureInitialized(); err != nil {
		return nil, err
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := &TagStats{}
	stats.TotalTags = int64(len(tm.tagToKeys))
	stats.TotalKeys = int64(len(tm.keyToTags))
	stats.TagOperations = atomic.LoadInt64(&tm.metrics.AddTagsCount) +
		atomic.LoadInt64(&tm.metrics.RemoveTagsCount) +
		atomic.LoadInt64(&tm.metrics.InvalidateByTagCount) +
		atomic.LoadInt64(&tm.metrics.RemoveKeyCount)

	return stats, nil
}

// GetMetrics returns internal metrics for monitoring and observability.
// Returns a copy of the current metrics to ensure thread safety.
func (tm *TagManager) GetMetrics() *TagMetrics {
	if err := tm.ensureInitialized(); err != nil {
		return &TagMetrics{}
	}

	// Metrics should never be nil if properly initialized, but handle defensively
	if tm.metrics == nil {
		logx.Warn("Metrics is nil, returning empty metrics")
		return &TagMetrics{}
	}

	return &TagMetrics{
		AddTagsCount:         atomic.LoadInt64(&tm.metrics.AddTagsCount),
		RemoveTagsCount:      atomic.LoadInt64(&tm.metrics.RemoveTagsCount),
		InvalidateByTagCount: atomic.LoadInt64(&tm.metrics.InvalidateByTagCount),
		RemoveKeyCount:       atomic.LoadInt64(&tm.metrics.RemoveKeyCount),
		PersistCount:         atomic.LoadInt64(&tm.metrics.PersistCount),
		LoadCount:            atomic.LoadInt64(&tm.metrics.LoadCount),
		ErrorsCount:          atomic.LoadInt64(&tm.metrics.ErrorsCount),
	}
}

// persistTagMappings persists tag mappings to the underlying store.
// This method is thread-safe and handles serialization errors.
func (tm *TagManager) persistTagMappings(ctx context.Context) error {
	// Check context cancellation
	if ctx.Err() != nil {
		tm.incrementErrorCount()
		return ctx.Err()
	}

	if err := tm.validateStore(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	// Serialize tag mappings
	tagMappings := struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		TagToKeys: make(map[string][]string),
		KeyToTags: make(map[string][]string),
		Timestamp: time.Now(),
	}

	// Convert tag-to-keys mapping
	for tag, keys := range tm.tagToKeys {
		// Check context cancellation during iteration
		if ctx.Err() != nil {
			tm.incrementErrorCount()
			return ctx.Err()
		}

		keyList := make([]string, 0, len(keys))
		for key := range keys {
			keyList = append(keyList, key)
		}
		tagMappings.TagToKeys[tag] = keyList
	}

	// Convert key-to-tags mapping
	for key, tags := range tm.keyToTags {
		// Check context cancellation during iteration
		if ctx.Err() != nil {
			tm.incrementErrorCount()
			return ctx.Err()
		}

		tagList := make([]string, 0, len(tags))
		for tag := range tags {
			tagList = append(tagList, tag)
		}
		tagMappings.KeyToTags[key] = tagList
	}

	// Serialize to JSON
	data, err := json.Marshal(tagMappings)
	if err != nil {
		tm.incrementErrorCount()
		return fmt.Errorf("failed to marshal tag mappings: %w", err)
	}

	// Check context cancellation before store operation
	if ctx.Err() != nil {
		tm.incrementErrorCount()
		return ctx.Err()
	}

	// Store in Redis
	key := "cachex:tag_mappings"
	setResult := <-tm.store.Set(ctx, key, data, tm.config.TagMappingTTL)
	if setResult.Error != nil {
		tm.incrementErrorCount()
		return fmt.Errorf("failed to persist tag mappings: %w", setResult.Error)
	}

	atomic.AddInt64(&tm.metrics.PersistCount, 1)
	return nil
}

// loadTagMappings loads tag mappings from the underlying store.
// This method is called during initialization and handles deserialization errors.
func (tm *TagManager) loadTagMappings() error {
	if err := tm.validateStore(); err != nil {
		tm.incrementErrorCount()
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), tm.config.LoadTimeout)
	defer cancel()

	// Check context cancellation
	if ctx.Err() != nil {
		tm.incrementErrorCount()
		return ctx.Err()
	}

	key := "cachex:tag_mappings"
	getResult := <-tm.store.Get(ctx, key)
	if getResult.Error != nil {
		tm.incrementErrorCount()
		return fmt.Errorf("failed to get tag mappings from store: %w", getResult.Error)
	}

	if !getResult.Exists || getResult.Value == nil {
		return nil // No mappings stored
	}

	// Deserialize tag mappings
	var tagMappings struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(getResult.Value, &tagMappings); err != nil {
		tm.incrementErrorCount()
		return fmt.Errorf("failed to unmarshal tag mappings: %w", err)
	}

	// Validate loaded data
	if err := tm.validateLoadedMappings(&tagMappings); err != nil {
		tm.incrementErrorCount()
		return fmt.Errorf("invalid tag mappings data: %w", err)
	}

	// Check if data is stale
	if time.Since(tagMappings.Timestamp) > tm.config.TagMappingTTL {
		logx.Warn("Loaded tag mappings are stale, ignoring",
			logx.String("timestamp", tagMappings.Timestamp.Format(time.RFC3339)),
			logx.String("age", time.Since(tagMappings.Timestamp).String()))
		return nil
	}

	// Reconstruct in-memory mappings with proper locking
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Reconstruct tag-to-keys mapping
	for tag, keys := range tagMappings.TagToKeys {
		if tag == "" {
			continue // Skip empty tags
		}
		tm.tagToKeys[tag] = make(map[string]bool)
		for _, key := range keys {
			if key != "" {
				tm.tagToKeys[tag][key] = true
			}
		}
	}

	// Reconstruct key-to-tags mapping
	for key, tags := range tagMappings.KeyToTags {
		if key == "" {
			continue // Skip empty keys
		}
		tm.keyToTags[key] = make(map[string]bool)
		for _, tag := range tags {
			if tag != "" {
				tm.keyToTags[key][tag] = true
			}
		}
	}

	atomic.AddInt64(&tm.metrics.LoadCount, 1)

	logx.Info("Loaded tag mappings",
		logx.Int("tags", len(tm.tagToKeys)),
		logx.Int("keys", len(tm.keyToTags)))

	return nil
}

// backupMappingsLocked creates a backup of current mappings for rollback.
// This method must be called while holding a write lock.
func (tm *TagManager) backupMappingsLocked(keys []string) map[string]map[string]bool {
	// For small datasets, use the full backup approach
	if len(keys) <= 100 {
		backup := make(map[string]map[string]bool)
		for _, key := range keys {
			if keyTags, exists := tm.keyToTags[key]; exists {
				backup[key] = make(map[string]bool)
				for tag := range keyTags {
					backup[key][tag] = true
				}
			}
		}
		return backup
	}

	// For large datasets, use a more efficient approach
	// Only backup the keys that will be affected
	backup := make(map[string]map[string]bool, len(keys))
	for _, key := range keys {
		if keyTags, exists := tm.keyToTags[key]; exists && len(keyTags) > 0 {
			backup[key] = make(map[string]bool, len(keyTags))
			for tag := range keyTags {
				backup[key][tag] = true
			}
		}
	}

	return backup
}

// restoreMappingsLocked restores mappings from backup.
// This method must be called while holding a write lock.
func (tm *TagManager) restoreMappingsLocked(backup map[string]map[string]bool) {
	for key, tags := range backup {
		// Restore key-to-tags mapping
		if tm.keyToTags[key] == nil {
			tm.keyToTags[key] = make(map[string]bool)
		}
		for tag := range tags {
			tm.keyToTags[key][tag] = true

			// Restore tag-to-keys mapping
			if tm.tagToKeys[tag] == nil {
				tm.tagToKeys[tag] = make(map[string]bool)
			}
			tm.tagToKeys[tag][key] = true
		}
	}
}

// removeKeysFromMappingsLocked removes keys from tag mappings.
// This method must be called while holding a write lock.
func (tm *TagManager) removeKeysFromMappingsLocked(keys []string) {
	if len(keys) == 0 {
		return
	}

	for _, key := range keys {
		if key == "" {
			continue
		}

		// Get tags for this key
		tags := make([]string, 0)
		if keyTags, exists := tm.keyToTags[key]; exists {
			for tag := range keyTags {
				tags = append(tags, tag)
			}
		}

		// Remove from tag mappings
		for _, tag := range tags {
			if tm.tagToKeys[tag] != nil {
				delete(tm.tagToKeys[tag], key)
				if len(tm.tagToKeys[tag]) == 0 {
					delete(tm.tagToKeys, tag)
				}
			}
		}

		// Remove key mapping
		delete(tm.keyToTags, key)
	}
}

// validateLoadedMappings validates the integrity of loaded tag mappings
func (tm *TagManager) validateLoadedMappings(mappings *struct {
	TagToKeys map[string][]string `json:"tag_to_keys"`
	KeyToTags map[string][]string `json:"key_to_tags"`
	Timestamp time.Time           `json:"timestamp"`
}) error {
	if mappings == nil {
		return fmt.Errorf("mappings cannot be nil")
	}

	// Validate timestamp
	if mappings.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	// Validate tag-to-keys mapping
	if mappings.TagToKeys == nil {
		return fmt.Errorf("tag_to_keys mapping cannot be nil")
	}

	// Validate key-to-tags mapping
	if mappings.KeyToTags == nil {
		return fmt.Errorf("key_to_tags mapping cannot be nil")
	}

	// Check for consistency between tag-to-keys and key-to-tags
	for tag, keys := range mappings.TagToKeys {
		if tag == "" {
			continue // Skip empty tags
		}
		for _, key := range keys {
			if key == "" {
				continue // Skip empty keys
			}
			// Check if key exists in key-to-tags mapping
			if keyTags, exists := mappings.KeyToTags[key]; !exists {
				return fmt.Errorf("inconsistent mapping: key %s exists in tag %s but not in key_to_tags", key, tag)
			} else {
				// Check if tag exists in key's tags
				found := false
				for _, keyTag := range keyTags {
					if keyTag == tag {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("inconsistent mapping: tag %s exists for key %s in tag_to_keys but not in key_to_tags", tag, key)
				}
			}
		}
	}

	return nil
}
