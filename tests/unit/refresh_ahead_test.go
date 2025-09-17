package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

func TestDefaultRefreshAheadConfig(t *testing.T) {
	config := cachex.DefaultRefreshAheadConfig()

	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
	}
	if config.DefaultRefreshBefore != 1*time.Minute {
		t.Errorf("Expected DefaultRefreshBefore to be 1 minute, got %v", config.DefaultRefreshBefore)
	}
	if config.MaxConcurrentRefreshes != 10 {
		t.Errorf("Expected MaxConcurrentRefreshes to be 10, got %d", config.MaxConcurrentRefreshes)
	}
	if config.RefreshInterval != 30*time.Second {
		t.Errorf("Expected RefreshInterval to be 30 seconds, got %v", config.RefreshInterval)
	}
	if !config.EnableDistributedLock {
		t.Errorf("Expected EnableDistributedLock to be true, got %v", config.EnableDistributedLock)
	}
	if config.LockTimeout != 30*time.Second {
		t.Errorf("Expected LockTimeout to be 30 seconds, got %v", config.LockTimeout)
	}
	if !config.EnableMetrics {
		t.Errorf("Expected EnableMetrics to be true, got %v", config.EnableMetrics)
	}
}

func TestNewDistributedLockManager(t *testing.T) {
	mockStore := NewMockStore()
	lockManager := cachex.NewDistributedLockManager(mockStore)

	if lockManager == nil {
		t.Errorf("NewDistributedLockManager() should not return nil")
	}
	// We can't access the unexported store field, but we can test the functionality
	// by calling TryLock which uses the store
	ctx := context.Background()
	unlock, acquired, err := lockManager.TryLock(ctx, "test-key", 5*time.Minute)
	if err != nil {
		t.Errorf("TryLock() failed: %v", err)
	}
	if !acquired {
		t.Errorf("TryLock() should acquire lock successfully")
	}
	if unlock == nil {
		t.Errorf("TryLock() should return unlock function")
	}

	// Clean up
	unlock()
}

func TestDistributedLockManager_TryLock_Success(t *testing.T) {
	mockStore := NewMockStore()
	lockManager := cachex.NewDistributedLockManager(mockStore)
	ctx := context.Background()

	unlock, acquired, err := lockManager.TryLock(ctx, "test-key", 5*time.Minute)
	if err != nil {
		t.Errorf("TryLock() failed: %v", err)
	}
	if !acquired {
		t.Errorf("TryLock() should acquire lock successfully")
	}
	if unlock == nil {
		t.Errorf("TryLock() should return unlock function")
	}

	// Test unlock function
	err = unlock()
	if err != nil {
		t.Errorf("Unlock() failed: %v", err)
	}
}

func TestNewRefreshAheadScheduler_WithNilConfig(t *testing.T) {
	mockStore := NewMockStore()
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, nil)

	if scheduler == nil {
		t.Errorf("NewRefreshAheadScheduler() should not return nil")
	}

	// Test that the scheduler works by scheduling a refresh
	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	err := scheduler.ScheduleRefresh("test-key", 1*time.Minute, loader)
	if err != nil {
		t.Errorf("ScheduleRefresh() failed: %v", err)
	}

	// Clean up
	scheduler.Close()
}

func TestNewRefreshAheadScheduler_WithCustomConfig(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:                false,
		DefaultRefreshBefore:   2 * time.Minute,
		MaxConcurrentRefreshes: 5,
		RefreshInterval:        1 * time.Minute,
		EnableDistributedLock:  false,
		LockTimeout:            1 * time.Minute,
		EnableMetrics:          false,
	}

	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	if scheduler == nil {
		t.Errorf("NewRefreshAheadScheduler() should not return nil")
	}

	// Test that the scheduler respects the disabled config
	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	err := scheduler.ScheduleRefresh("test-key", 1*time.Minute, loader)
	if err == nil {
		t.Errorf("ScheduleRefresh() should fail when disabled")
	}

	// Clean up
	scheduler.Close()
}

func TestRefreshAheadScheduler_ScheduleRefresh_Enabled(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:         true,
		RefreshInterval: 30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	err := scheduler.ScheduleRefresh("test-key", 1*time.Minute, loader)
	if err != nil {
		t.Errorf("ScheduleRefresh() failed: %v", err)
	}

	// Test that we can get stats and they show the task
	stats := scheduler.GetRefreshStats()
	if stats["total_tasks"] != 1 {
		t.Errorf("Expected total_tasks to be 1, got %v", stats["total_tasks"])
	}

	// Clean up
	scheduler.Close()
}

func TestRefreshAheadScheduler_ScheduleRefresh_Disabled(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled: false,
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	err := scheduler.ScheduleRefresh("test-key", 1*time.Minute, loader)
	if err == nil {
		t.Errorf("ScheduleRefresh() should fail when disabled")
	}
}

func TestRefreshAheadScheduler_ScheduleRefresh_DefaultRefreshBefore(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:              true,
		DefaultRefreshBefore: 2 * time.Minute,
		RefreshInterval:      30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	err := scheduler.ScheduleRefresh("test-key", 0, loader)
	if err != nil {
		t.Errorf("ScheduleRefresh() failed: %v", err)
	}

	// Test that the task was scheduled
	stats := scheduler.GetRefreshStats()
	if stats["total_tasks"] != 1 {
		t.Errorf("Expected total_tasks to be 1, got %v", stats["total_tasks"])
	}

	// Clean up
	scheduler.Close()
}

func TestRefreshAheadScheduler_UnscheduleRefresh(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:         true,
		RefreshInterval: 30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	// Schedule a task first
	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}
	err := scheduler.ScheduleRefresh("test-key", 1*time.Minute, loader)
	if err != nil {
		t.Errorf("ScheduleRefresh() failed: %v", err)
	}

	// Verify task exists
	stats := scheduler.GetRefreshStats()
	if stats["total_tasks"] != 1 {
		t.Errorf("Expected total_tasks to be 1 before unscheduling, got %v", stats["total_tasks"])
	}

	// Unschedule the task
	err = scheduler.UnscheduleRefresh("test-key")
	if err != nil {
		t.Errorf("UnscheduleRefresh() failed: %v", err)
	}

	// Verify task was removed
	stats = scheduler.GetRefreshStats()
	if stats["total_tasks"] != 0 {
		t.Errorf("Expected total_tasks to be 0 after unscheduling, got %v", stats["total_tasks"])
	}

	// Clean up
	scheduler.Close()
}

func TestRefreshAheadScheduler_UnscheduleRefresh_NonExistent(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:         true,
		RefreshInterval: 30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	// Try to unschedule non-existent task
	err := scheduler.UnscheduleRefresh("non-existent-key")
	if err != nil {
		t.Errorf("UnscheduleRefresh() should not fail for non-existent key: %v", err)
	}
}

func TestRefreshAheadScheduler_GetRefreshStats(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:                true,
		MaxConcurrentRefreshes: 5,
		RefreshInterval:        30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	// Schedule some tasks
	loader := func(ctx context.Context) (interface{}, error) {
		return []byte("test-value"), nil
	}

	scheduler.ScheduleRefresh("key1", 1*time.Minute, loader)
	scheduler.ScheduleRefresh("key2", 2*time.Minute, loader)

	stats := scheduler.GetRefreshStats()

	if stats["total_tasks"] != 2 {
		t.Errorf("Expected total_tasks to be 2, got %v", stats["total_tasks"])
	}
	if stats["enabled"] != true {
		t.Errorf("Expected enabled to be true, got %v", stats["enabled"])
	}
	if stats["max_concurrent"] != 5 {
		t.Errorf("Expected max_concurrent to be 5, got %v", stats["max_concurrent"])
	}
	if stats["total_refreshes"] != int64(0) {
		t.Errorf("Expected total_refreshes to be 0, got %v", stats["total_refreshes"])
	}
	if stats["total_errors"] != int64(0) {
		t.Errorf("Expected total_errors to be 0, got %v", stats["total_errors"])
	}
}

func TestRefreshAheadScheduler_Close(t *testing.T) {
	mockStore := NewMockStore()
	config := &cachex.RefreshAheadConfig{
		Enabled:         true,
		RefreshInterval: 30 * time.Second, // Ensure positive interval
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	err := scheduler.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Test that Close() doesn't panic and returns successfully
	// We can't easily test context cancellation without accessing unexported fields
}

func TestRefreshAheadScheduler_Concurrency(t *testing.T) {
	mockStore := &MockStore{}
	config := &cachex.RefreshAheadConfig{
		Enabled:                true,
		MaxConcurrentRefreshes: 3,
		RefreshInterval:        50 * time.Millisecond,
	}
	scheduler := cachex.NewRefreshAheadScheduler(mockStore, config)

	// Schedule multiple tasks
	numTasks := 5

	for i := 0; i < numTasks; i++ {
		key := fmt.Sprintf("key-%d", i)
		loader := func(ctx context.Context) (interface{}, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return []byte(fmt.Sprintf("value-%d", i)), nil
		}

		err := scheduler.ScheduleRefresh(key, 10*time.Millisecond, loader)
		if err != nil {
			t.Errorf("ScheduleRefresh() failed for key %s: %v", key, err)
		}
	}

	// Verify all tasks were scheduled
	stats := scheduler.GetRefreshStats()
	if stats["total_tasks"] != numTasks {
		t.Errorf("Expected %d tasks, got %v", numTasks, stats["total_tasks"])
	}

	// Clean up
	scheduler.Close()
}

func TestRefreshAheadStats(t *testing.T) {
	stats := &cachex.RefreshAheadStats{
		TotalTasks:      5,
		TotalRefreshes:  10,
		TotalErrors:     2,
		LastRefreshTime: time.Now(),
	}

	if stats.TotalTasks != 5 {
		t.Errorf("Expected TotalTasks to be 5, got %d", stats.TotalTasks)
	}
	if stats.TotalRefreshes != 10 {
		t.Errorf("Expected TotalRefreshes to be 10, got %d", stats.TotalRefreshes)
	}
	if stats.TotalErrors != 2 {
		t.Errorf("Expected TotalErrors to be 2, got %d", stats.TotalErrors)
	}
	if stats.LastRefreshTime.IsZero() {
		t.Errorf("Expected LastRefreshTime to be non-zero")
	}
}
