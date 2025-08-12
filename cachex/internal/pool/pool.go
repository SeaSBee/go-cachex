package pool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool provides a configurable worker pool with backpressure for batch operations
type WorkerPool struct {
	// Configuration
	config Config

	// Worker management
	workers    []*Worker
	workerPool chan *Worker
	mu         sync.RWMutex

	// Job management
	jobQueue   chan Job
	jobResults chan JobResult

	// Statistics
	stats *PoolStats

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed int32
}

// Config defines worker pool configuration
type Config struct {
	// Pool size configuration
	MinWorkers  int           // Minimum number of workers
	MaxWorkers  int           // Maximum number of workers
	QueueSize   int           // Size of job queue (backpressure control)
	IdleTimeout time.Duration // How long workers can be idle before scaling down

	// Performance tuning
	ScaleUpThreshold   int           // Number of queued jobs before scaling up
	ScaleDownThreshold int           // Number of idle workers before scaling down
	ScaleUpCooldown    time.Duration // Cooldown period between scale up operations
	ScaleDownCooldown  time.Duration // Cooldown period between scale down operations

	// Monitoring
	EnableMetrics bool // Enable detailed metrics collection
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		MinWorkers:         2,
		MaxWorkers:         10,
		QueueSize:          100,
		IdleTimeout:        30 * time.Second,
		ScaleUpThreshold:   5,
		ScaleDownThreshold: 2,
		ScaleUpCooldown:    5 * time.Second,
		ScaleDownCooldown:  10 * time.Second,
		EnableMetrics:      true,
	}
}

// HighPerformanceConfig returns a configuration optimized for high throughput
func HighPerformanceConfig() Config {
	return Config{
		MinWorkers:         5,
		MaxWorkers:         50,
		QueueSize:          1000,
		IdleTimeout:        60 * time.Second,
		ScaleUpThreshold:   10,
		ScaleDownThreshold: 3,
		ScaleUpCooldown:    2 * time.Second,
		ScaleDownCooldown:  15 * time.Second,
		EnableMetrics:      true,
	}
}

// ResourceConstrainedConfig returns a configuration for resource-constrained environments
func ResourceConstrainedConfig() Config {
	return Config{
		MinWorkers:         1,
		MaxWorkers:         5,
		QueueSize:          50,
		IdleTimeout:        15 * time.Second,
		ScaleUpThreshold:   3,
		ScaleDownThreshold: 1,
		ScaleUpCooldown:    10 * time.Second,
		ScaleDownCooldown:  5 * time.Second,
		EnableMetrics:      false,
	}
}

// Job represents a task to be executed by a worker
type Job struct {
	ID       string
	Task     func() (interface{}, error)
	Priority int // Higher number = higher priority
	Timeout  time.Duration
	Created  time.Time
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobID    string
	Result   interface{}
	Error    error
	Duration time.Duration
	WorkerID int
}

// PoolStats holds worker pool statistics
type PoolStats struct {
	ActiveWorkers  int32
	IdleWorkers    int32
	TotalWorkers   int32
	QueuedJobs     int32
	CompletedJobs  int64
	FailedJobs     int64
	TotalJobTime   time.Duration
	AverageJobTime time.Duration
	ScaleUpCount   int32
	ScaleDownCount int32
	LastScaleUp    time.Time
	LastScaleDown  time.Time
	mu             sync.RWMutex
}

// New creates a new worker pool with the given configuration
func New(config Config) (*WorkerPool, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		config:     config,
		jobQueue:   make(chan Job, config.QueueSize),
		jobResults: make(chan JobResult, config.QueueSize),
		stats:      &PoolStats{},
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize workers
	if err := pool.initializeWorkers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize workers: %w", err)
	}

	// Start background processes
	pool.startBackgroundProcesses()

	return pool, nil
}

// validateConfig validates the pool configuration
func validateConfig(config Config) error {
	if config.MinWorkers <= 0 {
		return fmt.Errorf("min workers must be positive")
	}
	if config.MaxWorkers <= 0 {
		return fmt.Errorf("max workers must be positive")
	}
	if config.MinWorkers > config.MaxWorkers {
		return fmt.Errorf("min workers cannot exceed max workers")
	}
	if config.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	if config.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}
	return nil
}

// initializeWorkers creates the initial set of workers
func (wp *WorkerPool) initializeWorkers() error {
	wp.workerPool = make(chan *Worker, wp.config.MaxWorkers)
	wp.workers = make([]*Worker, 0, wp.config.MaxWorkers)

	// Create minimum number of workers
	for i := 0; i < wp.config.MinWorkers; i++ {
		worker := wp.createWorker(i)
		wp.workers = append(wp.workers, worker)
		wp.workerPool <- worker
		atomic.AddInt32(&wp.stats.TotalWorkers, 1)
		atomic.AddInt32(&wp.stats.IdleWorkers, 1)
	}

	return nil
}

// createWorker creates a new worker
func (wp *WorkerPool) createWorker(id int) *Worker {
	worker := &Worker{
		ID:       id,
		pool:     wp,
		jobChan:  make(chan Job, 1),
		stopChan: make(chan struct{}),
		stats:    &WorkerStats{},
	}

	wp.wg.Add(1)
	go worker.start()

	return worker
}

// startBackgroundProcesses starts background monitoring and scaling processes
func (wp *WorkerPool) startBackgroundProcesses() {
	// Start job result processor
	go wp.processJobResults()

	// Start auto-scaling if enabled
	if wp.config.MaxWorkers > wp.config.MinWorkers {
		go wp.autoScale()
	}

	// Start metrics collection if enabled
	if wp.config.EnableMetrics {
		go wp.collectMetrics()
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job Job) error {
	if atomic.LoadInt32(&wp.closed) == 1 {
		return fmt.Errorf("worker pool is closed")
	}

	// Set default values
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	if job.Created.IsZero() {
		job.Created = time.Now()
	}
	if job.Timeout == 0 {
		job.Timeout = 30 * time.Second
	}

	// Try to submit job (this provides backpressure)
	select {
	case wp.jobQueue <- job:
		atomic.AddInt32(&wp.stats.QueuedJobs, 1)
		return nil
	default:
		return fmt.Errorf("job queue is full - backpressure applied")
	}
}

// SubmitWithContext submits a job with context support
func (wp *WorkerPool) SubmitWithContext(ctx context.Context, job Job) error {
	if atomic.LoadInt32(&wp.closed) == 1 {
		return fmt.Errorf("worker pool is closed")
	}

	// Set default values
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	if job.Created.IsZero() {
		job.Created = time.Now()
	}
	if job.Timeout == 0 {
		job.Timeout = 30 * time.Second
	}

	// Try to submit job with context
	select {
	case wp.jobQueue <- job:
		atomic.AddInt32(&wp.stats.QueuedJobs, 1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
		return fmt.Errorf("job queue is full - backpressure applied")
	}
}

// SubmitBatch submits multiple jobs as a batch
func (wp *WorkerPool) SubmitBatch(jobs []Job) ([]error, error) {
	if atomic.LoadInt32(&wp.closed) == 1 {
		return nil, fmt.Errorf("worker pool is closed")
	}

	errors := make([]error, len(jobs))
	var batchError error

	for i, job := range jobs {
		if err := wp.Submit(job); err != nil {
			errors[i] = err
			if batchError == nil {
				batchError = fmt.Errorf("batch submission failed")
			}
		}
	}

	return errors, batchError
}

// GetResult retrieves a job result
func (wp *WorkerPool) GetResult() (JobResult, error) {
	select {
	case result := <-wp.jobResults:
		return result, nil
	case <-wp.ctx.Done():
		return JobResult{}, fmt.Errorf("worker pool closed")
	default:
		return JobResult{}, fmt.Errorf("no results available")
	}
}

// GetResultWithTimeout retrieves a job result with timeout
func (wp *WorkerPool) GetResultWithTimeout(timeout time.Duration) (JobResult, error) {
	select {
	case result := <-wp.jobResults:
		return result, nil
	case <-time.After(timeout):
		return JobResult{}, fmt.Errorf("timeout waiting for result")
	case <-wp.ctx.Done():
		return JobResult{}, fmt.Errorf("worker pool closed")
	}
}

// GetStats returns current pool statistics
func (wp *WorkerPool) GetStats() *PoolStats {
	wp.stats.mu.RLock()
	defer wp.stats.mu.RUnlock()

	// Update atomic values
	wp.stats.ActiveWorkers = atomic.LoadInt32(&wp.stats.ActiveWorkers)
	wp.stats.IdleWorkers = atomic.LoadInt32(&wp.stats.IdleWorkers)
	wp.stats.TotalWorkers = atomic.LoadInt32(&wp.stats.TotalWorkers)
	wp.stats.QueuedJobs = atomic.LoadInt32(&wp.stats.QueuedJobs)
	wp.stats.CompletedJobs = atomic.LoadInt64(&wp.stats.CompletedJobs)
	wp.stats.FailedJobs = atomic.LoadInt64(&wp.stats.FailedJobs)
	wp.stats.ScaleUpCount = atomic.LoadInt32(&wp.stats.ScaleUpCount)
	wp.stats.ScaleDownCount = atomic.LoadInt32(&wp.stats.ScaleDownCount)

	return wp.stats
}

// ScaleUp adds workers to the pool
func (wp *WorkerPool) ScaleUp(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	currentWorkers := len(wp.workers)
	maxNewWorkers := wp.config.MaxWorkers - currentWorkers

	if maxNewWorkers <= 0 {
		return fmt.Errorf("cannot scale up - already at maximum workers")
	}

	if count > maxNewWorkers {
		count = maxNewWorkers
	}

	for i := 0; i < count; i++ {
		workerID := currentWorkers + i
		worker := wp.createWorker(workerID)
		wp.workers = append(wp.workers, worker)
		wp.workerPool <- worker
		atomic.AddInt32(&wp.stats.TotalWorkers, 1)
		atomic.AddInt32(&wp.stats.IdleWorkers, 1)
	}

	atomic.AddInt32(&wp.stats.ScaleUpCount, 1)
	wp.stats.LastScaleUp = time.Now()

	return nil
}

// ScaleDown removes workers from the pool
func (wp *WorkerPool) ScaleDown(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	currentWorkers := len(wp.workers)
	minWorkers := wp.config.MinWorkers

	if currentWorkers <= minWorkers {
		return fmt.Errorf("cannot scale down - already at minimum workers")
	}

	maxRemovable := currentWorkers - minWorkers
	if count > maxRemovable {
		count = maxRemovable
	}

	// Stop workers from the end of the list
	for i := 0; i < count; i++ {
		workerIndex := currentWorkers - 1 - i
		worker := wp.workers[workerIndex]
		worker.stop()
		wp.workers = wp.workers[:workerIndex+1]
	}

	atomic.AddInt32(&wp.stats.ScaleDownCount, 1)
	wp.stats.LastScaleDown = time.Now()

	return nil
}

// Close gracefully shuts down the worker pool
func (wp *WorkerPool) Close() error {
	if !atomic.CompareAndSwapInt32(&wp.closed, 0, 1) {
		return fmt.Errorf("worker pool already closed")
	}

	// Cancel context to stop background processes
	wp.cancel()

	// Stop all workers
	wp.mu.Lock()
	for _, worker := range wp.workers {
		worker.stop()
	}
	wp.mu.Unlock()

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close channels
	close(wp.jobQueue)
	close(wp.jobResults)
	close(wp.workerPool)

	return nil
}

// processJobResults processes completed job results
func (wp *WorkerPool) processJobResults() {
	for {
		select {
		case result := <-wp.jobResults:
			atomic.AddInt32(&wp.stats.QueuedJobs, -1)
			if result.Error != nil {
				atomic.AddInt64(&wp.stats.FailedJobs, 1)
			} else {
				atomic.AddInt64(&wp.stats.CompletedJobs, 1)
			}

			wp.stats.mu.Lock()
			wp.stats.TotalJobTime += result.Duration
			if wp.stats.CompletedJobs > 0 {
				wp.stats.AverageJobTime = wp.stats.TotalJobTime / time.Duration(wp.stats.CompletedJobs)
			}
			wp.stats.mu.Unlock()

		case <-wp.ctx.Done():
			return
		}
	}
}

// autoScale automatically scales the worker pool based on load
func (wp *WorkerPool) autoScale() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastScaleUp, lastScaleDown time.Time

	for {
		select {
		case <-ticker.C:
			stats := wp.GetStats()

			// Check if we should scale up
			if stats.QueuedJobs >= int32(wp.config.ScaleUpThreshold) &&
				time.Since(lastScaleUp) >= wp.config.ScaleUpCooldown &&
				stats.TotalWorkers < int32(wp.config.MaxWorkers) {

				scaleUpCount := 1
				if stats.QueuedJobs >= int32(wp.config.ScaleUpThreshold*2) {
					scaleUpCount = 2
				}

				if err := wp.ScaleUp(scaleUpCount); err == nil {
					lastScaleUp = time.Now()
				}
			}

			// Check if we should scale down
			if stats.IdleWorkers >= int32(wp.config.ScaleDownThreshold) &&
				time.Since(lastScaleDown) >= wp.config.ScaleDownCooldown &&
				stats.TotalWorkers > int32(wp.config.MinWorkers) {

				scaleDownCount := 1
				if stats.IdleWorkers >= int32(wp.config.ScaleDownThreshold*2) {
					scaleDownCount = 2
				}

				if err := wp.ScaleDown(scaleDownCount); err == nil {
					lastScaleDown = time.Now()
				}
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// collectMetrics collects detailed metrics if enabled
func (wp *WorkerPool) collectMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Metrics collection logic can be added here
			// For now, we just keep the basic stats updated
		case <-wp.ctx.Done():
			return
		}
	}
}
