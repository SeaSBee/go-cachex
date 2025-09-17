package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

// TestNewOptimizedTagManager tests the creation of optimized tag manager
func TestNewOptimizedTagManager(t *testing.T) {
	store := NewMockTagStore()

	tests := []struct {
		name        string
		config      *cachex.OptimizedTagConfig
		expectError bool
	}{
		{
			name:        "With nil config",
			config:      nil,
			expectError: false,
		},
		{
			name: "With custom config",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   50,
				EnableStats:                 true,
				LoadTimeout:                 5 * time.Second,
				EnableBackgroundPersistence: false,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    true,
				MaxMemoryUsage:              50 * 1024 * 1024,
				BatchFlushInterval:          2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "With memory optimization disabled",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   100,
				EnableStats:                 true,
				LoadTimeout:                 5 * time.Second,
				EnableBackgroundPersistence: false,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    false,
				MaxMemoryUsage:              100 * 1024 * 1024,
				BatchFlushInterval:          5 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := cachex.NewOptimizedTagManager(store, tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			if tm == nil {
				t.Errorf("OptimizedTagManager should not be nil for %s", tt.name)
				return
			}

			// Test that the manager is functional by calling GetStats
			_, err = tm.GetStats()
			if err != nil {
				t.Errorf("OptimizedTagManager should be functional for %s: %v", tt.name, err)
			}

			// Clean up
			tm.Close()
		})
	}
}

// TestOptimizedTagManager_AddTags_ThreeTierProcessing tests the three-tier batch processing logic
func TestOptimizedTagManager_AddTags_ThreeTierProcessing(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   10, // Small batch size for testing
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false, // Disable for simpler testing
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          1 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	tests := []struct {
		name           string
		tags           []string
		expectedMethod string // "direct" or "batch"
	}{
		{
			name:           "Very small operation (≤5 tags) - should use direct processing",
			tags:           []string{"tag1", "tag2", "tag3"},
			expectedMethod: "direct",
		},
		{
			name:           "Medium operation (≤batch size) - should use batch buffer",
			tags:           []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7"},
			expectedMethod: "batch",
		},
		{
			name:           "Large operation (>batch size) - should use direct processing",
			tags:           []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10", "tag11"},
			expectedMethod: "direct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := fmt.Sprintf("test-key-%s", tt.name)
			err := tm.AddTags(ctx, key, tt.tags...)
			if err != nil {
				t.Errorf("AddTags() failed: %v", err)
				return
			}

			// Verify tags were added
			resultTags, err := tm.GetTagsByKey(ctx, key)
			if err != nil {
				t.Errorf("GetTagsByKey() failed: %v", err)
				return
			}

			if len(resultTags) != len(tt.tags) {
				t.Errorf("Expected %d tags, got %d", len(tt.tags), len(resultTags))
			}

			// Verify each tag is present
			for _, expectedTag := range tt.tags {
				found := false
				for _, resultTag := range resultTags {
					if resultTag == expectedTag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tag %s not found in result", expectedTag)
				}
			}
		})
	}
}

// TestOptimizedTagManager_MemoryUsageTracking tests memory usage tracking and limits
func TestOptimizedTagManager_MemoryUsageTracking(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    true,
		MaxMemoryUsage:              1000, // Very small limit for testing
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Test memory limit enforcement
	err = tm.AddTags(ctx, "key1", "tag1", "tag2", "tag3", "tag4", "tag5")
	if err == nil {
		t.Errorf("Expected memory limit exceeded error, got nil")
	} else if !strings.Contains(fmt.Sprintf("%v", err), "memory usage limit exceeded") {
		t.Errorf("Expected memory limit error, got: %v", err)
	}

	// Test with larger limit
	config.MaxMemoryUsage = 100 * 1024 * 1024 // 100MB
	tm2, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm2.Close()

	// This should work now
	err = tm2.AddTags(ctx, "key1", "tag1", "tag2", "tag3")
	if err != nil {
		t.Errorf("AddTags() should work with larger memory limit: %v", err)
	}

	// Test memory usage tracking in stats
	stats, err := tm2.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}

	if stats.MemoryUsage <= 0 {
		t.Errorf("Memory usage should be tracked, got %d", stats.MemoryUsage)
	}
}

// TestOptimizedTagManager_BatchProcessing tests batch processing functionality
func TestOptimizedTagManager_BatchProcessing(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   10, // Large enough to allow 6 tags in batch processing
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          100 * time.Millisecond,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Add tags that should go to batch buffer (6 tags > 5, <= 10)
	key := "batch-key-1"
	tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
	err = tm.AddTags(ctx, key, tags...)
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Wait for batch processing
	time.Sleep(200 * time.Millisecond)

	// Test batch operations count in stats
	stats, err := tm.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}

	if stats.BatchOperations <= 0 {
		t.Errorf("Batch operations should be tracked, got %d", stats.BatchOperations)
	}
}

// TestOptimizedTagManager_RaceConditionFixes tests the race condition fixes
func TestOptimizedTagManager_RaceConditionFixes(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()
	numGoroutines := 20
	numOperations := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent AddTags operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				tags := []string{
					fmt.Sprintf("tag-%d", j%5),
					fmt.Sprintf("tag-%d", (j+1)%5),
					fmt.Sprintf("tag-%d", (j+2)%5),
				}
				err := tm.AddTags(ctx, key, tags...)
				if err != nil {
					t.Errorf("Concurrent AddTags() failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Test concurrent RemoveTags operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				tags := []string{fmt.Sprintf("tag-%d", j%5)}
				err := tm.RemoveTags(ctx, key, tags...)
				if err != nil {
					t.Errorf("Concurrent RemoveTags() failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	stats, err := tm.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}

	if stats.TotalKeys < 0 {
		t.Errorf("Invalid total keys after concurrent operations: %d", stats.TotalKeys)
	}
}

// TestOptimizedTagManager_ContextCancellation tests context cancellation handling
func TestOptimizedTagManager_ContextCancellation(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = tm.AddTags(ctx, "key1", "tag1")
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}

	err = tm.RemoveTags(ctx, "key1", "tag1")
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}

	err = tm.RemoveKey(ctx, "key1")
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}

	err = tm.InvalidateByTag(ctx, "tag1")
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}

	// Test with timeout context
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelTimeout()

	// Wait for timeout
	time.Sleep(2 * time.Millisecond)

	err = tm.AddTags(ctxTimeout, "key2", "tag2")
	if err == nil || !strings.Contains(fmt.Sprintf("%v", err), "context") {
		t.Errorf("Expected context timeout error, got: %v", err)
	}
}

// TestOptimizedTagManager_ConfigurationValidation tests configuration validation
func TestOptimizedTagManager_ConfigurationValidation(t *testing.T) {
	store := NewMockTagStore()

	tests := []struct {
		name        string
		config      *cachex.OptimizedTagConfig
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   100,
				EnableStats:                 true,
				LoadTimeout:                 5 * time.Second,
				EnableBackgroundPersistence: true,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    true,
				MaxMemoryUsage:              100 * 1024 * 1024,
				BatchFlushInterval:          5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Zero batch size",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   0,
				EnableStats:                 true,
				LoadTimeout:                 5 * time.Second,
				EnableBackgroundPersistence: true,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    true,
				MaxMemoryUsage:              100 * 1024 * 1024,
				BatchFlushInterval:          5 * time.Second,
			},
			expectError: false, // Should use default
		},
		{
			name: "Negative batch size",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   -10,
				EnableStats:                 true,
				LoadTimeout:                 5 * time.Second,
				EnableBackgroundPersistence: true,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    true,
				MaxMemoryUsage:              100 * 1024 * 1024,
				BatchFlushInterval:          5 * time.Second,
			},
			expectError: false, // Should use default
		},
		{
			name: "Zero timeout",
			config: &cachex.OptimizedTagConfig{
				EnablePersistence:           true,
				TagMappingTTL:               1 * time.Hour,
				BatchSize:                   100,
				EnableStats:                 true,
				LoadTimeout:                 0,
				EnableBackgroundPersistence: true,
				PersistenceInterval:         30 * time.Second,
				EnableMemoryOptimization:    true,
				MaxMemoryUsage:              100 * 1024 * 1024,
				BatchFlushInterval:          5 * time.Second,
			},
			expectError: false, // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := cachex.NewOptimizedTagManager(store, tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			if tm == nil {
				t.Errorf("OptimizedTagManager should not be nil for %s", tt.name)
				return
			}

			// Clean up
			tm.Close()
		})
	}
}

// TestOptimizedTagManager_DefaultConfig tests the default configuration
func TestOptimizedTagManager_DefaultConfig(t *testing.T) {
	config := cachex.DefaultOptimizedTagConfig()

	if config == nil {
		t.Errorf("DefaultOptimizedTagConfig() should not return nil")
		return
	}

	if !config.EnablePersistence {
		t.Errorf("DefaultOptimizedTagConfig().EnablePersistence should be true")
	}

	if config.TagMappingTTL != 24*time.Hour {
		t.Errorf("DefaultOptimizedTagConfig().TagMappingTTL should be 24 hours, got %v", config.TagMappingTTL)
	}

	if config.BatchSize != 100 {
		t.Errorf("DefaultOptimizedTagConfig().BatchSize should be 100, got %d", config.BatchSize)
	}

	if !config.EnableStats {
		t.Errorf("DefaultOptimizedTagConfig().EnableStats should be true")
	}

	if !config.EnableBackgroundPersistence {
		t.Errorf("DefaultOptimizedTagConfig().EnableBackgroundPersistence should be true")
	}

	if config.PersistenceInterval != 30*time.Second {
		t.Errorf("DefaultOptimizedTagConfig().PersistenceInterval should be 30 seconds, got %v", config.PersistenceInterval)
	}

	if !config.EnableMemoryOptimization {
		t.Errorf("DefaultOptimizedTagConfig().EnableMemoryOptimization should be true")
	}

	if config.MaxMemoryUsage != 100*1024*1024 {
		t.Errorf("DefaultOptimizedTagConfig().MaxMemoryUsage should be 100MB, got %d", config.MaxMemoryUsage)
	}

	if config.BatchFlushInterval != 5*time.Second {
		t.Errorf("DefaultOptimizedTagConfig().BatchFlushInterval should be 5 seconds, got %v", config.BatchFlushInterval)
	}
}

// TestOptimizedTagManager_BackgroundProcessing tests background processing functionality
func TestOptimizedTagManager_BackgroundProcessing(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: true,
		PersistenceInterval:         100 * time.Millisecond, // Short interval for testing
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          100 * time.Millisecond,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}

	ctx := context.Background()

	// Add some tags to trigger persistence
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Wait for background processing
	time.Sleep(200 * time.Millisecond)

	// Verify data was persisted
	key := "cachex:optimized_tag_mappings"
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Store.Exists() failed: %v", existsResult.Error)
	}

	if !existsResult.Exists {
		t.Errorf("Tag mappings should be persisted by background worker")
	}

	// Clean up
	tm.Close()
}

// TestOptimizedTagManager_GracefulShutdown tests graceful shutdown functionality
func TestOptimizedTagManager_GracefulShutdown(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false, // Disable background persistence for immediate persistence
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}

	ctx := context.Background()

	// Add some data
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	// Close the manager
	err = tm.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify final persistence occurred
	key := "cachex:optimized_tag_mappings"
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Errorf("Store.Exists() failed: %v", existsResult.Error)
	}

	if !existsResult.Exists {
		t.Errorf("Tag mappings should be persisted on close")
	}
}

// TestOptimizedTagManager_ErrorHandling tests error handling scenarios
func TestOptimizedTagManager_ErrorHandling(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Test nil context
	err = tm.AddTags(nil, "key1", "tag1")
	if err == nil {
		t.Errorf("Expected error for nil context")
	}

	// Test empty key
	err = tm.AddTags(ctx, "", "tag1")
	if err == nil {
		t.Errorf("Expected error for empty key")
	}

	// Test empty tag
	err = tm.AddTags(ctx, "key1", "")
	if err == nil {
		t.Errorf("Expected error for empty tag")
	}

	// Test multiple empty tags
	err = tm.AddTags(ctx, "key1", "tag1", "", "tag2")
	if err == nil {
		t.Errorf("Expected error for empty tag in list")
	}

	// Test GetKeysByTag with empty tag
	_, err = tm.GetKeysByTag(ctx, "")
	if err == nil {
		t.Errorf("Expected error for empty tag in GetKeysByTag")
	}

	// Test GetTagsByKey with empty key
	_, err = tm.GetTagsByKey(ctx, "")
	if err == nil {
		t.Errorf("Expected error for empty key in GetTagsByKey")
	}

	// Test InvalidateByTag with empty tag
	err = tm.InvalidateByTag(ctx, "")
	if err == nil {
		t.Errorf("Expected error for empty tag in InvalidateByTag")
	}

	// Test RemoveKey with empty key
	err = tm.RemoveKey(ctx, "")
	if err == nil {
		t.Errorf("Expected error for empty key in RemoveKey")
	}
}

// TestOptimizedTagManager_StatsAccuracy tests statistics accuracy
func TestOptimizedTagManager_StatsAccuracy(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Get initial stats
	initialStats, err := tm.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}

	// Perform operations
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Errorf("AddTags() failed: %v", err)
	}

	err = tm.RemoveTags(ctx, "key1", "tag1")
	if err != nil {
		t.Errorf("RemoveTags() failed: %v", err)
	}

	err = tm.RemoveKey(ctx, "key1")
	if err != nil {
		t.Errorf("RemoveKey() failed: %v", err)
	}

	// Get final stats
	finalStats, err := tm.GetStats()
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
		return
	}

	// Verify stats were incremented correctly
	if finalStats.AddTagsCount != initialStats.AddTagsCount+1 {
		t.Errorf("AddTagsCount should be incremented, got %d (was %d)",
			finalStats.AddTagsCount, initialStats.AddTagsCount)
	}

	if finalStats.RemoveTagsCount != initialStats.RemoveTagsCount+1 {
		t.Errorf("RemoveTagsCount should be incremented, got %d (was %d)",
			finalStats.RemoveTagsCount, initialStats.RemoveTagsCount)
	}

	if finalStats.RemoveKeyCount != initialStats.RemoveKeyCount+1 {
		t.Errorf("RemoveKeyCount should be incremented, got %d (was %d)",
			finalStats.RemoveKeyCount, initialStats.RemoveKeyCount)
	}

	// Verify memory usage tracking
	if finalStats.MemoryUsage < 0 {
		t.Errorf("Memory usage should be non-negative, got %d", finalStats.MemoryUsage)
	}
}

// TestOptimizedTagManager_Persistence tests persistence functionality
func TestOptimizedTagManager_Persistence(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false, // Disable for testing
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()

	// Add some tags
	err = tm.AddTags(ctx, "key1", "tag1", "tag2")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	err = tm.AddTags(ctx, "key2", "tag2", "tag3")
	if err != nil {
		t.Fatalf("AddTags() failed: %v", err)
	}

	// Verify data was persisted
	key := "cachex:optimized_tag_mappings"
	existsResult := <-store.Exists(ctx, key)
	if existsResult.Error != nil {
		t.Fatalf("Store.Exists() failed: %v", existsResult.Error)
	}

	if !existsResult.Exists {
		t.Errorf("Tag mappings should be persisted to store")
	}

	// Get the persisted data
	getResult := <-store.Get(ctx, key)
	if getResult.Error != nil {
		t.Fatalf("Store.Get() failed: %v", getResult.Error)
	}

	if !getResult.Exists || getResult.Value == nil {
		t.Errorf("Persisted tag mappings should exist and not be nil")
	}

	// Verify the data contains our mappings
	var tagMappings struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(getResult.Value, &tagMappings); err != nil {
		t.Fatalf("Failed to unmarshal persisted data: %v", err)
	}

	// Check that our mappings are in the persisted data
	if len(tagMappings.TagToKeys) == 0 {
		t.Errorf("Persisted data should contain tag mappings")
	}

	if len(tagMappings.KeyToTags) == 0 {
		t.Errorf("Persisted data should contain key mappings")
	}

	// Check specific mappings
	if keys, exists := tagMappings.TagToKeys["tag1"]; !exists || len(keys) == 0 {
		t.Errorf("Tag1 should be in persisted mappings")
	}

	if keys, exists := tagMappings.TagToKeys["tag2"]; !exists || len(keys) != 2 {
		t.Errorf("Tag2 should have 2 keys in persisted mappings")
	}
}

// TestOptimizedTagManager_Loading tests loading functionality
func TestOptimizedTagManager_Loading(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           true,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	// Pre-populate store with tag mappings
	ctx := context.Background()
	tagMappings := struct {
		TagToKeys map[string][]string `json:"tag_to_keys"`
		KeyToTags map[string][]string `json:"key_to_tags"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		TagToKeys: map[string][]string{
			"tag1": {"key1", "key2"},
			"tag2": {"key2", "key3"},
		},
		KeyToTags: map[string][]string{
			"key1": {"tag1"},
			"key2": {"tag1", "tag2"},
			"key3": {"tag2"},
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(tagMappings)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	key := "cachex:optimized_tag_mappings"
	setResult := <-store.Set(ctx, key, data, config.TagMappingTTL)
	if setResult.Error != nil {
		t.Fatalf("Failed to set test data: %v", setResult.Error)
	}

	// Create optimized tag manager - should load existing mappings
	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	// Verify mappings were loaded
	keys, err := tm.GetKeysByTag(ctx, "tag1")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag1, got %d", len(keys))
	}

	keys, err = tm.GetKeysByTag(ctx, "tag2")
	if err != nil {
		t.Fatalf("GetKeysByTag() failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for tag2, got %d", len(keys))
	}

	tags, err := tm.GetTagsByKey(ctx, "key2")
	if err != nil {
		t.Fatalf("GetTagsByKey() failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags for key2, got %d", len(tags))
	}
}

// TestOptimizedTagManager_StressTest tests stress scenarios
func TestOptimizedTagManager_StressTest(t *testing.T) {
	store := NewMockTagStore()
	config := &cachex.OptimizedTagConfig{
		EnablePersistence:           false,
		TagMappingTTL:               1 * time.Hour,
		BatchSize:                   100,
		EnableStats:                 true,
		LoadTimeout:                 5 * time.Second,
		EnableBackgroundPersistence: false,
		PersistenceInterval:         30 * time.Second,
		EnableMemoryOptimization:    false,
		MaxMemoryUsage:              100 * 1024 * 1024,
		BatchFlushInterval:          5 * time.Second,
	}

	tm, err := cachex.NewOptimizedTagManager(store, config)
	if err != nil {
		t.Fatalf("Failed to create optimized tag manager: %v", err)
	}
	defer tm.Close()

	ctx := context.Background()
	numOperations := 1000
	var wg sync.WaitGroup
	wg.Add(4) // 4 different operation types

	// Stress test with mixed operations
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("add-key-%d", i)
			tag := fmt.Sprintf("add-tag-%d", i%100)
			err := tm.AddTags(ctx, key, tag)
			if err != nil {
				t.Errorf("Stress test AddTags() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			tag := fmt.Sprintf("query-tag-%d", i%100)
			_, err := tm.GetKeysByTag(ctx, tag)
			if err != nil {
				t.Errorf("Stress test GetKeysByTag() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("query-key-%d", i%1000)
			_, err := tm.GetTagsByKey(ctx, key)
			if err != nil {
				t.Errorf("Stress test GetTagsByKey() failed: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("remove-key-%d", i)
			err := tm.RemoveKey(ctx, key)
			if err != nil {
				t.Errorf("Stress test RemoveKey() failed: %v", err)
				return
			}
		}
	}()

	wg.Wait()

	// Verify system is still functional
	stats, err := tm.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed after stress test: %v", err)
	}

	if stats.TotalKeys < 0 {
		t.Errorf("Invalid total keys after stress test: %d", stats.TotalKeys)
	}
}
