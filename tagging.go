package cachex

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// TagManager manages tag-to-keys and key-to-tags mappings
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
}

// TagConfig holds tagging configuration
type TagConfig struct {
	// Enable persistence of tag mappings
	EnablePersistence bool `yaml:"enable_persistence" json:"enable_persistence"`
	// TTL for tag mappings
	TagMappingTTL time.Duration `yaml:"tag_mapping_ttl" json:"tag_mapping_ttl"`
	// Batch size for tag operations
	BatchSize int `yaml:"batch_size" json:"batch_size"`
	// Enable statistics
	EnableStats bool `yaml:"enable_stats" json:"enable_stats"`
}

// DefaultTagConfig returns a default configuration
func DefaultTagConfig() *TagConfig {
	return &TagConfig{
		EnablePersistence: true,
		TagMappingTTL:     24 * time.Hour,
		BatchSize:         100,
		EnableStats:       true,
	}
}

// TagStats holds tagging statistics
type TagStats struct {
	TotalTags     int64
	TotalKeys     int64
	TagOperations int64
}

// NewTagManager creates a new tag manager
func NewTagManager(store Store, config *TagConfig) *TagManager {
	if config == nil {
		config = DefaultTagConfig()
	}

	tm := &TagManager{
		tagToKeys: make(map[string]map[string]bool),
		keyToTags: make(map[string]map[string]bool),
		store:     store,
		config:    config,
	}

	// Load existing tag mappings if persistence is enabled
	if config.EnablePersistence {
		tm.loadTagMappings()
	}

	return tm
}

// AddTags adds tags to a key
func (tm *TagManager) AddTags(ctx context.Context, key string, tags ...string) error {
	if len(tags) == 0 {
		return nil
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

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// RemoveTags removes tags from a key
func (tm *TagManager) RemoveTags(ctx context.Context, key string, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Remove tags from key
	for _, tag := range tags {
		if tm.keyToTags[key] != nil {
			delete(tm.keyToTags[key], tag)
		}
		if tm.tagToKeys[tag] != nil {
			delete(tm.tagToKeys[tag], key)
		}
	}

	// Clean up empty mappings
	if len(tm.keyToTags[key]) == 0 {
		delete(tm.keyToTags, key)
	}

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// GetKeysByTag returns all keys associated with a tag
func (tm *TagManager) GetKeysByTag(ctx context.Context, tag string) ([]string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if keys, exists := tm.tagToKeys[tag]; exists {
		result := make([]string, 0, len(keys))
		for key := range keys {
			result = append(result, key)
		}
		return result, nil
	}

	return []string{}, nil
}

// GetTagsByKey returns all tags associated with a key
func (tm *TagManager) GetTagsByKey(ctx context.Context, key string) ([]string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tags, exists := tm.keyToTags[key]; exists {
		result := make([]string, 0, len(tags))
		for tag := range tags {
			result = append(result, tag)
		}
		return result, nil
	}

	return []string{}, nil
}

// InvalidateByTag invalidates all keys associated with the given tags
func (tm *TagManager) InvalidateByTag(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Delete keys in batches
	for i := 0; i < len(keys); i += tm.config.BatchSize {
		end := i + tm.config.BatchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		delResult := <-tm.store.Del(ctx, batch...)
		if delResult.Error != nil {
			logx.Error("Failed to delete keys by tag",
				logx.String("batch", fmt.Sprintf("%v", batch)),
				logx.ErrorField(delResult.Error))
			return delResult.Error
		}

		// Remove from tag mappings
		tm.removeKeysFromMappings(batch)
	}

	logx.Info("Invalidated keys by tag",
		logx.Int("count", len(keys)),
		logx.String("keys", fmt.Sprintf("%v", keys)))

	return nil
}

// RemoveKey removes a key and all its tag associations
func (tm *TagManager) RemoveKey(ctx context.Context, key string) error {
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

	// Persist if enabled
	if tm.config.EnablePersistence {
		return tm.persistTagMappings(ctx)
	}

	return nil
}

// GetStats returns tagging statistics
func (tm *TagManager) GetStats() *TagStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := &TagStats{}
	stats.TotalTags = int64(len(tm.tagToKeys))
	stats.TotalKeys = int64(len(tm.keyToTags))

	return stats
}

// persistTagMappings persists tag mappings to store
func (tm *TagManager) persistTagMappings(ctx context.Context) error {
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
		keyList := make([]string, 0, len(keys))
		for key := range keys {
			keyList = append(keyList, key)
		}
		tagMappings.TagToKeys[tag] = keyList
	}

	// Convert key-to-tags mapping
	for key, tags := range tm.keyToTags {
		tagList := make([]string, 0, len(tags))
		for tag := range tags {
			tagList = append(tagList, tag)
		}
		tagMappings.KeyToTags[key] = tagList
	}

	// Serialize to JSON
	data, err := json.Marshal(tagMappings)
	if err != nil {
		return fmt.Errorf("failed to marshal tag mappings: %w", err)
	}

	// Store in Redis
	key := "cachex:tag_mappings"
	setResult := <-tm.store.Set(ctx, key, data, tm.config.TagMappingTTL)
	return setResult.Error
}

// loadTagMappings loads tag mappings from store
func (tm *TagManager) loadTagMappings() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "cachex:tag_mappings"
	getResult := <-tm.store.Get(ctx, key)
	if getResult.Error != nil {
		logx.Warn("Failed to load tag mappings", logx.ErrorField(getResult.Error))
		return
	}

	if !getResult.Exists || getResult.Value == nil {
		return // No mappings stored
	}

	// Deserialize tag mappings
	var tagMappings struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(getResult.Value, &tagMappings); err != nil {
		logx.Error("Failed to unmarshal tag mappings", logx.ErrorField(err))
		return
	}

	// Reconstruct in-memory mappings
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Reconstruct tag-to-keys mapping
	for tag, keys := range tagMappings.TagToKeys {
		tm.tagToKeys[tag] = make(map[string]bool)
		for _, key := range keys {
			tm.tagToKeys[tag][key] = true
		}
	}

	// Reconstruct key-to-tags mapping
	for key, tags := range tagMappings.KeyToTags {
		tm.keyToTags[key] = make(map[string]bool)
		for _, tag := range tags {
			tm.keyToTags[key][tag] = true
		}
	}

	logx.Info("Loaded tag mappings",
		logx.Int("tags", len(tm.tagToKeys)),
		logx.Int("keys", len(tm.keyToTags)))
}

// removeKeysFromMappings removes keys from tag mappings
func (tm *TagManager) removeKeysFromMappings(keys []string) {
	for _, key := range keys {
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
