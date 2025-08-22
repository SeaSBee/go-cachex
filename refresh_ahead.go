package cachex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// RefreshAheadScheduler manages background refresh-ahead operations
type RefreshAheadScheduler struct {
	// Configuration
	config *RefreshAheadConfig
	// Store for cache operations
	store Store
	// Distributed lock manager
	lockManager *DistributedLockManager
	// Scheduled refresh tasks
	tasks map[string]*RefreshTask
	// Mutex for thread safety
	mu sync.RWMutex
	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	// Worker pool for background operations
	workerPool chan struct{}
}

// RefreshAheadConfig holds refresh-ahead configuration
type RefreshAheadConfig struct {
	// Enable refresh-ahead
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Default refresh before TTL
	DefaultRefreshBefore time.Duration `yaml:"default_refresh_before" json:"default_refresh_before"`
	// Maximum concurrent refresh operations
	MaxConcurrentRefreshes int `yaml:"max_concurrent_refreshes" json:"max_concurrent_refreshes"`
	// Refresh interval for background scanning
	RefreshInterval time.Duration `yaml:"refresh_interval" json:"refresh_interval"`
	// Enable distributed locking
	EnableDistributedLock bool `yaml:"enable_distributed_lock" json:"enable_distributed_lock"`
	// Lock timeout for refresh operations
	LockTimeout time.Duration `yaml:"lock_timeout" json:"lock_timeout"`
	// Enable metrics
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"`
}

// DefaultRefreshAheadConfig returns a default configuration
func DefaultRefreshAheadConfig() *RefreshAheadConfig {
	return &RefreshAheadConfig{
		Enabled:                true,
		DefaultRefreshBefore:   1 * time.Minute,
		MaxConcurrentRefreshes: 10,
		RefreshInterval:        30 * time.Second,
		EnableDistributedLock:  true,
		LockTimeout:            30 * time.Second,
		EnableMetrics:          true,
	}
}

// RefreshTask represents a scheduled refresh task
type RefreshTask struct {
	Key           string
	RefreshBefore time.Duration
	Loader        func(ctx context.Context) (interface{}, error)
	LastRefresh   time.Time
	NextRefresh   time.Time
	RefreshCount  int64
	LastError     error
	TTL           time.Duration
	mu            sync.RWMutex
}

// DistributedLockManager manages distributed locks for refresh operations
type DistributedLockManager struct {
	store Store
}

// NewDistributedLockManager creates a new distributed lock manager
func NewDistributedLockManager(store Store) *DistributedLockManager {
	return &DistributedLockManager{
		store: store,
	}
}

// TryLock attempts to acquire a distributed lock
func (dlm *DistributedLockManager) TryLock(ctx context.Context, key string, ttl time.Duration) (func() error, bool, error) {
	lockKey := fmt.Sprintf("refresh_lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	// Try to set the lock key
	setResult := <-dlm.store.Set(ctx, lockKey, []byte(lockValue), ttl)
	if setResult.Error != nil {
		return nil, false, fmt.Errorf("failed to acquire lock: %w", setResult.Error)
	}

	// Return unlock function
	unlock := func() error {
		delResult := <-dlm.store.Del(ctx, lockKey)
		return delResult.Error
	}

	return unlock, true, nil
}

// NewRefreshAheadScheduler creates a new refresh-ahead scheduler
func NewRefreshAheadScheduler(store Store, config *RefreshAheadConfig) *RefreshAheadScheduler {
	if config == nil {
		config = DefaultRefreshAheadConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &RefreshAheadScheduler{
		config:      config,
		store:       store,
		lockManager: NewDistributedLockManager(store),
		tasks:       make(map[string]*RefreshTask),
		ctx:         ctx,
		cancel:      cancel,
		workerPool:  make(chan struct{}, config.MaxConcurrentRefreshes),
	}

	// Start background refresh loop if enabled
	if config.Enabled {
		go scheduler.startBackgroundRefresh()
	}

	return scheduler
}

// ScheduleRefresh schedules a key for refresh-ahead
func (ras *RefreshAheadScheduler) ScheduleRefresh(key string, refreshBefore time.Duration, loader func(ctx context.Context) (interface{}, error)) error {
	if !ras.config.Enabled {
		return fmt.Errorf("refresh-ahead is disabled")
	}

	if refreshBefore == 0 {
		refreshBefore = ras.config.DefaultRefreshBefore
	}

	ras.mu.Lock()
	defer ras.mu.Unlock()

	// Create or update refresh task
	task := &RefreshTask{
		Key:           key,
		RefreshBefore: refreshBefore,
		Loader:        loader,
		LastRefresh:   time.Now(),
		NextRefresh:   time.Now().Add(refreshBefore),
	}

	ras.tasks[key] = task

	logx.Info("Scheduled refresh-ahead",
		logx.String("key", key),
		logx.String("refresh_before", refreshBefore.String()),
		logx.String("next_refresh", task.NextRefresh.Format(time.RFC3339)))

	return nil
}

// UnscheduleRefresh removes a key from refresh scheduling
func (ras *RefreshAheadScheduler) UnscheduleRefresh(key string) error {
	ras.mu.Lock()
	defer ras.mu.Unlock()

	if _, exists := ras.tasks[key]; exists {
		delete(ras.tasks, key)
		logx.Info("Unscheduled refresh-ahead", logx.String("key", key))
	}

	return nil
}

// GetRefreshStats returns refresh statistics
func (ras *RefreshAheadScheduler) GetRefreshStats() map[string]interface{} {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tasks"] = len(ras.tasks)
	stats["enabled"] = ras.config.Enabled
	stats["max_concurrent"] = ras.config.MaxConcurrentRefreshes

	// Calculate task statistics
	var totalRefreshes int64
	var totalErrors int64
	for _, task := range ras.tasks {
		task.mu.RLock()
		totalRefreshes += task.RefreshCount
		if task.LastError != nil {
			totalErrors++
		}
		task.mu.RUnlock()
	}

	stats["total_refreshes"] = totalRefreshes
	stats["total_errors"] = totalErrors

	return stats
}

// startBackgroundRefresh starts the background refresh loop
func (ras *RefreshAheadScheduler) startBackgroundRefresh() {
	ticker := time.NewTicker(ras.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ras.ctx.Done():
			return
		case <-ticker.C:
			ras.processRefreshTasks()
		}
	}
}

// processRefreshTasks processes all scheduled refresh tasks
func (ras *RefreshAheadScheduler) processRefreshTasks() {
	ras.mu.RLock()
	tasks := make([]*RefreshTask, 0, len(ras.tasks))
	for _, task := range ras.tasks {
		tasks = append(tasks, task)
	}
	ras.mu.RUnlock()

	// Process tasks that need refresh
	for _, task := range tasks {
		task.mu.RLock()
		needsRefresh := time.Now().After(task.NextRefresh)
		task.mu.RUnlock()

		if needsRefresh {
			// Try to acquire worker slot
			select {
			case ras.workerPool <- struct{}{}:
				go func(t *RefreshTask) {
					defer func() { <-ras.workerPool }()
					ras.executeRefresh(t)
				}(task)
			default:
				logx.Warn("Worker pool full, skipping refresh", logx.String("key", task.Key))
			}
		}
	}
}

// executeRefresh executes a refresh operation for a task
func (ras *RefreshAheadScheduler) executeRefresh(task *RefreshTask) {
	// Use a shorter timeout for refresh operations to prevent blocking
	refreshCtx, cancel := context.WithTimeout(ras.ctx, ras.config.LockTimeout/2)
	defer cancel()

	// Try to acquire distributed lock if enabled
	var unlock func() error
	var lockAcquired bool
	var err error

	if ras.config.EnableDistributedLock {
		unlock, lockAcquired, err = ras.lockManager.TryLock(refreshCtx, task.Key, ras.config.LockTimeout)
		if err != nil {
			logx.Error("Failed to acquire refresh lock",
				logx.String("key", task.Key),
				logx.ErrorField(err))
			return
		}

		if !lockAcquired {
			logx.Debug("Refresh lock not acquired, skipping",
				logx.String("key", task.Key))
			return
		}

		defer func() {
			if unlockErr := unlock(); unlockErr != nil {
				logx.Error("Failed to release refresh lock",
					logx.String("key", task.Key),
					logx.ErrorField(unlockErr))
			}
		}()
	}

	// Execute refresh with timeout
	start := time.Now()
	value, err := task.Loader(refreshCtx)
	duration := time.Since(start)

	// Update task statistics
	task.mu.Lock()
	task.LastRefresh = time.Now()
	task.RefreshCount++
	task.LastError = err
	task.mu.Unlock()

	if err != nil {
		logx.Error("Refresh-ahead failed",
			logx.String("key", task.Key),
			logx.String("duration", duration.String()),
			logx.ErrorField(err))
		return
	}

	// Get current TTL with timeout
	ttlCtx, ttlCancel := context.WithTimeout(refreshCtx, 5*time.Second)
	defer ttlCancel()

	ttlResult := <-ras.store.TTL(ttlCtx, task.Key)
	var ttl time.Duration
	if ttlResult.Error != nil {
		logx.Warn("Failed to get TTL for refresh",
			logx.String("key", task.Key),
			logx.ErrorField(ttlResult.Error))
		ttl = 5 * time.Minute // Default TTL
	} else {
		ttl = ttlResult.TTL
	}

	// Update cache with new value (convert to bytes)
	var valueBytes []byte
	if value != nil {
		if bytes, ok := value.([]byte); ok {
			valueBytes = bytes
		} else {
			// For non-byte values, we'll skip the update
			logx.Warn("Refresh value is not bytes, skipping cache update",
				logx.String("key", task.Key),
				logx.String("value_type", fmt.Sprintf("%T", value)))
			return
		}
	}

	// Use a timeout for the final cache update
	setCtx, setCancel := context.WithTimeout(refreshCtx, 10*time.Second)
	defer setCancel()

	setResult := <-ras.store.Set(setCtx, task.Key, valueBytes, ttl)
	if setResult.Error != nil {
		logx.Error("Failed to update cache after refresh",
			logx.String("key", task.Key),
			logx.ErrorField(setResult.Error))
		return
	}

	// Schedule next refresh
	task.mu.Lock()
	task.NextRefresh = time.Now().Add(task.RefreshBefore)
	task.TTL = ttl
	task.mu.Unlock()

	logx.Info("Refresh-ahead completed",
		logx.String("key", task.Key),
		logx.String("duration", duration.String()),
		logx.String("ttl", ttl.String()),
		logx.String("next_refresh", task.NextRefresh.Format(time.RFC3339)))
}

// Close closes the refresh-ahead scheduler
func (ras *RefreshAheadScheduler) Close() error {
	ras.cancel()

	// Wait for all workers to complete
	for i := 0; i < ras.config.MaxConcurrentRefreshes; i++ {
		ras.workerPool <- struct{}{}
	}

	logx.Info("Closed refresh-ahead scheduler")
	return nil
}

// RefreshAheadStats holds refresh-ahead statistics
type RefreshAheadStats struct {
	TotalTasks      int64
	TotalRefreshes  int64
	TotalErrors     int64
	LastRefreshTime time.Time
}
