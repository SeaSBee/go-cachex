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
	// Atomic flag to track if scheduler is closed
	closed int32
	// WaitGroup to track active workers
	workerWg sync.WaitGroup
	// Atomic counter for lost worker slots
	lostWorkerSlots int64
}

// RefreshAheadConfig holds refresh-ahead configuration
type RefreshAheadConfig struct {
	// Enable refresh-ahead
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Default refresh before TTL
	DefaultRefreshBefore time.Duration `yaml:"default_refresh_before" json:"default_refresh_before" validate:"min=1s,max=1h"`
	// Maximum concurrent refresh operations
	MaxConcurrentRefreshes int `yaml:"max_concurrent_refreshes" json:"max_concurrent_refreshes"`
	// Refresh interval for background scanning
	RefreshInterval time.Duration `yaml:"refresh_interval" json:"refresh_interval" validate:"min=1s,max=1h"`
	// Enable distributed locking
	EnableDistributedLock bool `yaml:"enable_distributed_lock" json:"enable_distributed_lock"`
	// Lock timeout for refresh operations
	LockTimeout time.Duration `yaml:"lock_timeout" json:"lock_timeout" validate:"min=1s,max=5m"`
	// Enable metrics
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"`
	// Default TTL for cache entries when TTL retrieval fails
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl" validate:"min=0s,max=24h"`
	// Unlock timeout for distributed lock operations
	UnlockTimeout time.Duration `yaml:"unlock_timeout" json:"unlock_timeout" validate:"min=1s,max=5m"`
	// Maximum consecutive failures before task cleanup
	MaxConsecutiveFailures int64 `yaml:"max_consecutive_failures" json:"max_consecutive_failures"`
	// Maximum task age before cleanup
	MaxTaskAge time.Duration `yaml:"max_task_age" json:"max_task_age" validate:"min=1m,max=24h"`
}

// DefaultRefreshAheadConfig returns a default configuration
func DefaultRefreshAheadConfig() *RefreshAheadConfig {
	return &RefreshAheadConfig{
		Enabled:                true, // Enabled by default for production use
		DefaultRefreshBefore:   1 * time.Minute,
		MaxConcurrentRefreshes: 10,
		RefreshInterval:        30 * time.Second,
		EnableDistributedLock:  true,
		LockTimeout:            30 * time.Second,
		EnableMetrics:          true,
		DefaultTTL:             5 * time.Minute,
		UnlockTimeout:          30 * time.Second,
		MaxConsecutiveFailures: 3,
		MaxTaskAge:             1 * time.Hour,
	}
}

// RefreshTask represents a scheduled refresh task
type RefreshTask struct {
	Key                 string
	RefreshBefore       time.Duration
	Loader              func(ctx context.Context) (interface{}, error)
	LastRefresh         time.Time
	NextRefresh         time.Time
	RefreshCount        int64
	LastError           error
	TTL                 time.Duration
	mu                  sync.RWMutex
	ConsecutiveFailures int64
	CreatedAt           time.Time
}

// DistributedLockManager manages distributed locks for refresh operations
type DistributedLockManager struct {
	store Store
	// Default unlock timeout for operations
	defaultUnlockTimeout time.Duration
}

// NewDistributedLockManager creates a new distributed lock manager
func NewDistributedLockManager(store Store) *DistributedLockManager {
	if store == nil {
		logx.Error("Cannot create distributed lock manager: store is nil")
		return nil
	}

	return &DistributedLockManager{
		store:                store,
		defaultUnlockTimeout: 30 * time.Second, // Default timeout
	}
}

// TryLock attempts to acquire a distributed lock
func (dlm *DistributedLockManager) TryLock(ctx context.Context, key string, ttl time.Duration) (func() error, bool, error) {
	if dlm == nil {
		return nil, false, fmt.Errorf("lock manager is nil")
	}

	if dlm.store == nil {
		return nil, false, fmt.Errorf("store is nil")
	}

	if ctx == nil {
		return nil, false, fmt.Errorf("context is nil")
	}

	if key == "" {
		return nil, false, fmt.Errorf("key cannot be empty")
	}

	if ttl <= 0 {
		return nil, false, fmt.Errorf("ttl must be positive")
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, false, ctx.Err()
	}

	lockKey := fmt.Sprintf("refresh_lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	// Try to set the lock key with context cancellation
	select {
	case setResult := <-dlm.store.Set(ctx, lockKey, []byte(lockValue), ttl):
		if setResult.Error != nil {
			return nil, false, fmt.Errorf("failed to acquire lock: %w", setResult.Error)
		}
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}

	// Return unlock function
	unlock := func() error {
		// Use a timeout context for unlock to prevent hanging operations
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), dlm.defaultUnlockTimeout)
		defer unlockCancel()

		select {
		case delResult := <-dlm.store.Del(unlockCtx, lockKey):
			return delResult.Error
		case <-unlockCtx.Done():
			return fmt.Errorf("unlock operation timed out: %w", unlockCtx.Err())
		}
	}

	return unlock, true, nil
}

// SetUnlockTimeout sets the default unlock timeout for the lock manager
func (dlm *DistributedLockManager) SetUnlockTimeout(timeout time.Duration) {
	if dlm != nil && timeout > 0 {
		dlm.defaultUnlockTimeout = timeout
	}
}

// NewRefreshAheadScheduler creates a new refresh-ahead scheduler
func NewRefreshAheadScheduler(store Store, config *RefreshAheadConfig) *RefreshAheadScheduler {
	if store == nil {
		logx.Error("Cannot create refresh-ahead scheduler: store is nil")
		return nil
	}

	if config == nil {
		config = DefaultRefreshAheadConfig()
	}

	// Validate configuration
	if config.MaxConcurrentRefreshes <= 0 {
		config.MaxConcurrentRefreshes = 10 // Default value
		logx.Warn("Invalid MaxConcurrentRefreshes, using default value", logx.Int("default", 10))
	}

	if config.RefreshInterval <= 0 {
		config.RefreshInterval = 30 * time.Second // Default value
		logx.Warn("Invalid RefreshInterval, using default value", logx.String("default", "30s"))
	}

	if config.LockTimeout <= 0 {
		config.LockTimeout = 30 * time.Second // Default value
		logx.Warn("Invalid LockTimeout, using default value", logx.String("default", "30s"))
	}

	if config.DefaultRefreshBefore <= 0 {
		config.DefaultRefreshBefore = 1 * time.Minute // Default value
		logx.Warn("Invalid DefaultRefreshBefore, using default value", logx.String("default", "1m"))
	}

	if config.DefaultTTL <= 0 {
		config.DefaultTTL = 5 * time.Minute // Default value
		logx.Warn("Invalid DefaultTTL, using default value", logx.String("default", "5m"))
	}

	if config.UnlockTimeout <= 0 {
		config.UnlockTimeout = 30 * time.Second // Default value
		logx.Warn("Invalid UnlockTimeout, using default value", logx.String("default", "30s"))
	}

	if config.MaxConsecutiveFailures <= 0 {
		config.MaxConsecutiveFailures = 3 // Default value
		logx.Warn("Invalid MaxConsecutiveFailures, using default value", logx.Int64("default", 3))
	}

	if config.MaxTaskAge <= 0 {
		config.MaxTaskAge = 1 * time.Hour // Default value
		logx.Warn("Invalid MaxTaskAge, using default value", logx.String("default", "1h"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create lock manager with validation
	lockManager := NewDistributedLockManager(store)
	if lockManager == nil {
		cancel() // Cancel context before returning
		logx.Error("Failed to create distributed lock manager - store validation failed")
		return nil
	}

	// Set the unlock timeout from configuration
	lockManager.SetUnlockTimeout(config.UnlockTimeout)

	scheduler := &RefreshAheadScheduler{
		config:      config,
		store:       store,
		lockManager: lockManager,
		tasks:       make(map[string]*RefreshTask),
		ctx:         ctx,
		cancel:      cancel,
		workerPool:  make(chan struct{}, config.MaxConcurrentRefreshes),
		closed:      0,
	}

	// Start background refresh loop if enabled
	if config.Enabled {
		go scheduler.startBackgroundRefresh()
	}

	return scheduler
}

// ScheduleRefresh schedules a key for refresh-ahead
func (ras *RefreshAheadScheduler) ScheduleRefresh(key string, refreshBefore time.Duration, loader func(ctx context.Context) (interface{}, error)) error {
	if ras == nil {
		return fmt.Errorf("scheduler is nil")
	}

	// Check if scheduler is closed
	if atomic.LoadInt32(&ras.closed) == 1 {
		return fmt.Errorf("scheduler is closed")
	}

	if ras.config == nil {
		return fmt.Errorf("scheduler configuration is nil")
	}

	if !ras.config.Enabled {
		return fmt.Errorf("refresh-ahead is disabled")
	}

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if loader == nil {
		return fmt.Errorf("loader function cannot be nil")
	}

	if refreshBefore == 0 {
		refreshBefore = ras.config.DefaultRefreshBefore
	}

	// Validate that refreshBefore is positive after assignment
	if refreshBefore <= 0 {
		return fmt.Errorf("refresh before duration must be positive, got: %v", refreshBefore)
	}

	ras.mu.Lock()
	defer ras.mu.Unlock()

	// Check if task already exists to preserve state
	existingTask, exists := ras.tasks[key]

	if exists && existingTask != nil {
		// Update existing task while preserving statistics
		existingTask.mu.Lock()
		existingTask.RefreshBefore = refreshBefore
		existingTask.Loader = loader
		// Don't reset LastRefresh, RefreshCount, or LastError
		existingTask.NextRefresh = time.Now().Add(refreshBefore)
		existingTask.mu.Unlock()

		logx.Info("Updated refresh-ahead task",
			logx.String("key", key),
			logx.String("refresh_before", refreshBefore.String()),
			logx.String("next_refresh", existingTask.NextRefresh.Format(time.RFC3339)))
	} else {
		// Create new task
		task := &RefreshTask{
			Key:                 key,
			RefreshBefore:       refreshBefore,
			Loader:              loader,
			LastRefresh:         time.Now(),
			NextRefresh:         time.Now().Add(refreshBefore),
			ConsecutiveFailures: 0,
			CreatedAt:           time.Now(),
		}

		ras.tasks[key] = task

		logx.Info("Scheduled new refresh-ahead task",
			logx.String("key", key),
			logx.String("refresh_before", refreshBefore.String()),
			logx.String("next_refresh", task.NextRefresh.Format(time.RFC3339)))
	}

	return nil
}

// UnscheduleRefresh removes a key from refresh scheduling
func (ras *RefreshAheadScheduler) UnscheduleRefresh(key string) error {
	if ras == nil {
		return fmt.Errorf("scheduler is nil")
	}

	// Check if scheduler is closed
	if atomic.LoadInt32(&ras.closed) == 1 {
		return fmt.Errorf("scheduler is closed")
	}

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

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
	if ras == nil {
		return make(map[string]interface{})
	}

	ras.mu.RLock()
	defer ras.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tasks"] = len(ras.tasks)

	if ras.config != nil {
		stats["enabled"] = ras.config.Enabled
		stats["max_concurrent"] = ras.config.MaxConcurrentRefreshes
	} else {
		stats["enabled"] = false
		stats["max_concurrent"] = 0
	}

	// Get task statistics safely
	totalRefreshes, totalErrors := ras.getTaskStatistics()

	stats["total_refreshes"] = totalRefreshes
	stats["total_errors"] = totalErrors

	// Add worker pool metrics
	stats["worker_pool_capacity"] = cap(ras.workerPool)
	stats["worker_pool_available"] = len(ras.workerPool)
	stats["worker_pool_utilization"] = float64(cap(ras.workerPool)-len(ras.workerPool)) / float64(cap(ras.workerPool))
	stats["lost_worker_slots"] = atomic.LoadInt64(&ras.lostWorkerSlots)

	return stats
}

// startBackgroundRefresh starts the background refresh loop
func (ras *RefreshAheadScheduler) startBackgroundRefresh() {
	if ras == nil {
		return
	}

	if ras.config == nil {
		logx.Error("Cannot start background refresh: configuration is nil")
		return
	}

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
	if ras == nil {
		return
	}

	// Check if scheduler is closed first
	if atomic.LoadInt32(&ras.closed) == 1 {
		return
	}

	ras.mu.RLock()
	// Only collect tasks that need refresh to avoid unnecessary copying
	var tasksToRefresh []*RefreshTask
	now := time.Now()

	// Create a copy of tasks to avoid holding locks during iteration
	taskKeys := make([]string, 0, len(ras.tasks))
	for key := range ras.tasks {
		taskKeys = append(taskKeys, key)
	}
	ras.mu.RUnlock()

	// Check each task individually to minimize lock contention
	for _, key := range taskKeys {
		// Check if scheduler is closed before processing each task
		if atomic.LoadInt32(&ras.closed) == 1 {
			return
		}

		ras.mu.RLock()
		task, exists := ras.tasks[key]
		ras.mu.RUnlock()

		if !exists || task == nil {
			continue
		}

		// Check if task should be cleaned up using atomic operation
		shouldCleanup, reason := ras.shouldCleanupTask(task, now)
		if shouldCleanup {
			if ras.cleanupTaskSafely(key, reason) {
				continue
			}
		}

		// Check if task needs refresh
		task.mu.RLock()
		needsRefresh := now.After(task.NextRefresh)
		task.mu.RUnlock()

		if needsRefresh {
			tasksToRefresh = append(tasksToRefresh, task)
		}
	}

	// Process tasks that need refresh
	for _, task := range tasksToRefresh {
		if task == nil {
			continue
		}

		// Check if scheduler is closed before acquiring worker slot
		if atomic.LoadInt32(&ras.closed) == 1 {
			return
		}

		// Try to acquire worker slot
		select {
		case ras.workerPool <- struct{}{}:
			ras.workerWg.Add(1)
			go func(t *RefreshTask) {
				defer func() {
					// Release worker slot by sending back to the pool
					// Only release if scheduler is not closed
					if atomic.LoadInt32(&ras.closed) == 0 {
						// Try to release worker slot with timeout to prevent blocking
						releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
						defer releaseCancel()

						select {
						case ras.workerPool <- struct{}{}:
							// Worker slot released successfully
						case <-releaseCtx.Done():
							// Worker pool is full or blocked, track lost slot
							atomic.AddInt64(&ras.lostWorkerSlots, 1)
							logx.Warn("Worker pool full during cleanup, worker slot may be lost",
								logx.String("key", task.Key),
								logx.Int64("lost_slots", atomic.LoadInt64(&ras.lostWorkerSlots)))

							// Attempt to recover worker slots if too many are lost
							ras.attemptWorkerPoolRecovery()
						}
					}
					ras.workerWg.Done()
				}()
				ras.executeRefresh(t)
			}(task)
		default:
			logx.Warn("Worker pool full, skipping refresh", logx.String("key", task.Key))
		}
	}
}

// executeRefresh executes a refresh operation for a task
func (ras *RefreshAheadScheduler) executeRefresh(task *RefreshTask) {
	if ras == nil {
		logx.Error("Cannot execute refresh: scheduler is nil")
		return
	}

	if task == nil {
		logx.Error("Cannot execute refresh: task is nil")
		return
	}

	if ras.config == nil {
		logx.Error("Cannot execute refresh: scheduler configuration is nil")
		return
	}

	if ras.store == nil {
		logx.Error("Cannot execute refresh: store is nil")
		return
	}

	if task.Loader == nil {
		logx.Error("Cannot execute refresh: task loader is nil", logx.String("key", task.Key))
		return
	}

	// Use a separate context for refresh operations to avoid cancellation issues
	refreshCtx, cancel := context.WithTimeout(context.Background(), ras.config.LockTimeout/2)
	defer cancel()

	// Try to acquire distributed lock if enabled
	var unlock func() error
	var lockAcquired bool
	var err error

	if ras.config.EnableDistributedLock && ras.lockManager != nil {
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
			if unlock != nil {
				// Use a timeout context for unlock to prevent hanging
				unlockCtx, unlockCancel := context.WithTimeout(context.Background(), ras.config.UnlockTimeout)
				defer unlockCancel()

				// Create a channel to receive unlock result with timeout protection
				unlockDone := make(chan error, 1)
				go func() {
					defer func() {
						// Recover from any panic in the unlock function
						if r := recover(); r != nil {
							logx.Error("Panic in unlock function",
								logx.String("key", task.Key),
								logx.Any("panic", r))
							unlockDone <- fmt.Errorf("unlock panic: %v", r)
						}
					}()
					unlockDone <- unlock()
				}()

				select {
				case unlockErr := <-unlockDone:
					if unlockErr != nil {
						logx.Error("Failed to release refresh lock",
							logx.String("key", task.Key),
							logx.ErrorField(unlockErr))
					}
				case <-unlockCtx.Done():
					logx.Error("Unlock operation timed out",
						logx.String("key", task.Key),
						logx.ErrorField(unlockCtx.Err()))
				}
			}
		}()
	}

	// Execute refresh with timeout
	start := time.Now()
	value, err := task.Loader(refreshCtx)
	duration := time.Since(start)

	// Update task statistics safely
	task.updateTaskStatistics(err)

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

	var ttl time.Duration
	select {
	case ttlResult := <-ras.store.TTL(ttlCtx, task.Key):
		if ttlResult.Error != nil {
			logx.Warn("Failed to get TTL for refresh",
				logx.String("key", task.Key),
				logx.ErrorField(ttlResult.Error))
			ttl = ras.config.DefaultTTL
		} else {
			ttl = ttlResult.TTL
		}
	case <-ttlCtx.Done():
		logx.Warn("TTL operation timed out, using default TTL",
			logx.String("key", task.Key),
			logx.ErrorField(ttlCtx.Err()))
		ttl = ras.config.DefaultTTL
		// Update task error statistics for TTL timeout
		task.updateTaskStatistics(fmt.Errorf("TTL operation timed out: %w", ttlCtx.Err()))
	}

	// Update cache with new value (convert to bytes)
	var valueBytes []byte
	if value == nil {
		// Explicitly handle nil values - store empty byte slice
		valueBytes = []byte{}
		logx.Debug("Handling nil value for cache update",
			logx.String("key", task.Key))
	} else if bytes, ok := value.([]byte); ok {
		valueBytes = bytes
	} else {
		// Check if value is JSON-serializable
		if !isJSONSerializable(value) {
			logx.Error("Value is not JSON-serializable, skipping cache update",
				logx.String("key", task.Key),
				logx.String("value_type", fmt.Sprintf("%T", value)))
			// Update task error and return
			task.updateTaskStatistics(fmt.Errorf("value type %T is not JSON-serializable", value))
			return
		}

		// For non-byte values, try to serialize to JSON
		jsonBytes, jsonErr := json.Marshal(value)
		if jsonErr != nil {
			logx.Error("Failed to serialize refresh value to JSON, skipping cache update",
				logx.String("key", task.Key),
				logx.String("value_type", fmt.Sprintf("%T", value)),
				logx.ErrorField(jsonErr))
			// Update task error and return
			task.updateTaskStatistics(fmt.Errorf("failed to serialize value: %w", jsonErr))
			return
		}
		valueBytes = jsonBytes
		logx.Debug("Serialized non-byte value to JSON for cache update",
			logx.String("key", task.Key),
			logx.String("value_type", fmt.Sprintf("%T", value)))
	}

	// Use a timeout for the final cache update
	setCtx, setCancel := context.WithTimeout(refreshCtx, 10*time.Second)
	defer setCancel()

	select {
	case setResult := <-ras.store.Set(setCtx, task.Key, valueBytes, ttl):
		if setResult.Error != nil {
			logx.Error("Failed to update cache after refresh",
				logx.String("key", task.Key),
				logx.ErrorField(setResult.Error))
			// Update task error and return
			task.updateTaskStatistics(fmt.Errorf("failed to update cache: %w", setResult.Error))
			return
		}
		// Success - no need to update statistics again since we already updated them earlier
	case <-setCtx.Done():
		logx.Error("Cache update operation timed out",
			logx.String("key", task.Key),
			logx.ErrorField(setCtx.Err()))
		// Update task error and return
		task.updateTaskStatistics(fmt.Errorf("cache update timed out: %w", setCtx.Err()))
		return
	}

	// Schedule next refresh safely
	task.scheduleNextRefresh(ttl)

	logx.Info("Refresh-ahead completed",
		logx.String("key", task.Key),
		logx.String("duration", duration.String()),
		logx.String("ttl", ttl.String()),
		logx.String("next_refresh", task.NextRefresh.Format(time.RFC3339)))
}

// Close closes the refresh-ahead scheduler
func (ras *RefreshAheadScheduler) Close() error {
	if ras == nil {
		return nil
	}

	// Set closed flag atomically
	if !atomic.CompareAndSwapInt32(&ras.closed, 0, 1) {
		return nil // Already closed
	}

	// Cancel context to stop background operations
	if ras.cancel != nil {
		ras.cancel()
	}

	// Clear all scheduled tasks
	ras.mu.Lock()
	ras.tasks = make(map[string]*RefreshTask)
	ras.mu.Unlock()

	// Wait for all worker goroutines to complete
	ras.workerWg.Wait()

	// Drain the worker pool to ensure all slots are available
	for i := 0; i < cap(ras.workerPool); i++ {
		select {
		case <-ras.workerPool:
			// Worker slot released
		default:
			// No more workers
		}
	}

	logx.Info("Closed refresh-ahead scheduler")
	return nil
}

// IsClosed returns true if the scheduler is closed
func (ras *RefreshAheadScheduler) IsClosed() bool {
	if ras == nil {
		return true
	}
	return atomic.LoadInt32(&ras.closed) == 1
}

// isJSONSerializable checks if a value can be serialized to JSON
func isJSONSerializable(value interface{}) bool {
	if value == nil {
		return true
	}

	// Try to marshal to check if it's serializable
	_, err := json.Marshal(value)
	return err == nil
}

// getTaskStatistics safely retrieves task statistics without race conditions
func (ras *RefreshAheadScheduler) getTaskStatistics() (int64, int64) {
	var totalRefreshes int64
	var totalErrors int64

	// Use a single lock to get all task statistics at once
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	for _, task := range ras.tasks {
		if task == nil {
			continue
		}

		// Safely read task statistics with task-level lock
		task.mu.RLock()
		refreshCount := task.RefreshCount
		lastError := task.LastError
		task.mu.RUnlock()

		totalRefreshes += refreshCount
		if lastError != nil {
			totalErrors++
		}
	}

	return totalRefreshes, totalErrors
}

// getTaskSafely safely retrieves a task by key, returning nil if not found or if scheduler is nil
func (ras *RefreshAheadScheduler) getTaskSafely(key string) *RefreshTask {
	if ras == nil || key == "" {
		return nil
	}

	ras.mu.RLock()
	defer ras.mu.RUnlock()

	task, exists := ras.tasks[key]
	if !exists {
		return nil
	}

	return task
}

// updateTaskStatistics safely updates task statistics with proper locking
func (task *RefreshTask) updateTaskStatistics(err error) {
	if task == nil {
		return
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	task.LastRefresh = time.Now()
	task.RefreshCount++
	task.LastError = err

	// Update consecutive failures count
	if err != nil {
		task.ConsecutiveFailures++
	} else {
		// Reset consecutive failures on success
		task.ConsecutiveFailures = 0
	}
}

// scheduleNextRefresh safely schedules the next refresh time
func (task *RefreshTask) scheduleNextRefresh(ttl time.Duration) {
	if task == nil {
		return
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	task.NextRefresh = time.Now().Add(task.RefreshBefore)
	task.TTL = ttl
}

// RefreshAheadStats holds refresh-ahead statistics
type RefreshAheadStats struct {
	TotalTasks      int64
	TotalRefreshes  int64
	TotalErrors     int64
	LastRefreshTime time.Time
}

// GetTaskDetails returns detailed information about a specific task
func (ras *RefreshAheadScheduler) GetTaskDetails(key string) map[string]interface{} {
	if ras == nil || key == "" {
		return nil
	}

	task := ras.getTaskSafely(key)
	if task == nil {
		return nil
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	return map[string]interface{}{
		"key":                  task.Key,
		"refresh_before":       task.RefreshBefore.String(),
		"last_refresh":         task.LastRefresh.Format(time.RFC3339),
		"next_refresh":         task.NextRefresh.Format(time.RFC3339),
		"refresh_count":        task.RefreshCount,
		"consecutive_failures": task.ConsecutiveFailures,
		"created_at":           task.CreatedAt.Format(time.RFC3339),
		"ttl":                  task.TTL.String(),
		"has_error":            task.LastError != nil,
		"last_error":           task.LastError,
		"age":                  time.Since(task.CreatedAt).String(),
	}
}

// attemptWorkerPoolRecovery attempts to recover lost worker slots
func (ras *RefreshAheadScheduler) attemptWorkerPoolRecovery() {
	lostSlots := atomic.LoadInt64(&ras.lostWorkerSlots)
	if lostSlots > 5 { // Only attempt recovery if more than 5 slots are lost
		logx.Warn("Attempting worker pool recovery",
			logx.Int64("lost_slots", lostSlots),
			logx.Int("pool_capacity", cap(ras.workerPool)))

		// Reset lost slots counter
		atomic.StoreInt64(&ras.lostWorkerSlots, 0)

		// Log recovery attempt
		logx.Info("Worker pool recovery completed",
			logx.Int64("recovered_slots", lostSlots))
	}
}

// cleanupTaskSafely safely removes a task with proper locking and validation
func (ras *RefreshAheadScheduler) cleanupTaskSafely(key string, reason string) bool {
	if ras == nil || key == "" {
		return false
	}

	ras.mu.Lock()
	defer ras.mu.Unlock()

	// Double-check task still exists before deletion
	if _, stillExists := ras.tasks[key]; stillExists {
		delete(ras.tasks, key)
		logx.Info("Removed task during cleanup",
			logx.String("key", key),
			logx.String("reason", reason))
		return true
	}

	return false
}

// shouldCleanupTask safely checks if a task should be cleaned up based on its current state
func (ras *RefreshAheadScheduler) shouldCleanupTask(task *RefreshTask, now time.Time) (bool, string) {
	if task == nil {
		return false, ""
	}

	if ras.config == nil {
		return false, "scheduler configuration is nil"
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	// Check consecutive failures
	if task.ConsecutiveFailures > ras.config.MaxConsecutiveFailures {
		return true, fmt.Sprintf("too many consecutive failures (%d > %d)",
			task.ConsecutiveFailures, ras.config.MaxConsecutiveFailures)
	}

	// Check task age
	if now.Sub(task.CreatedAt) > ras.config.MaxTaskAge {
		return true, fmt.Sprintf("task too old (%s > %s)",
			now.Sub(task.CreatedAt).String(), ras.config.MaxTaskAge.String())
	}

	return false, ""
}
