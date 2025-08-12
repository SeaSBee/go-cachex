package pool

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Worker represents a worker in the pool
type Worker struct {
	ID       int
	pool     *WorkerPool
	jobChan  chan Job
	stopChan chan struct{}
	stats    *WorkerStats
	active   int32
}

// WorkerStats holds worker-specific statistics
type WorkerStats struct {
	JobsProcessed int64
	JobsFailed    int64
	TotalJobTime  time.Duration
	LastJobTime   time.Time
	IdleTime      time.Duration
	LastIdleStart time.Time
}

// start starts the worker loop
func (w *Worker) start() {
	defer w.pool.wg.Done()

	for {
		select {
		case job := <-w.pool.jobQueue:
			// Mark worker as active
			atomic.StoreInt32(&w.active, 1)
			atomic.AddInt32(&w.pool.stats.ActiveWorkers, 1)
			atomic.AddInt32(&w.pool.stats.IdleWorkers, -1)

			// Process the job
			result := w.processJob(job)

			// Send result back to pool
			select {
			case w.pool.jobResults <- result:
			case <-w.pool.ctx.Done():
				return
			}

			// Mark worker as idle
			atomic.StoreInt32(&w.active, 0)
			atomic.AddInt32(&w.pool.stats.ActiveWorkers, -1)
			atomic.AddInt32(&w.pool.stats.IdleWorkers, 1)

			// Return worker to pool
			select {
			case w.pool.workerPool <- w:
			case <-w.pool.ctx.Done():
				return
			}

		case <-w.stopChan:
			return

		case <-w.pool.ctx.Done():
			return
		}
	}
}

// processJob processes a single job
func (w *Worker) processJob(job Job) JobResult {
	start := time.Now()
	result := JobResult{
		JobID:    job.ID,
		WorkerID: w.ID,
	}

	// Create context with timeout
	ctx := context.Background()
	if job.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, job.Timeout)
		defer cancel()
	}

	// Execute the job
	jobResult, err := w.executeJob(ctx, job)
	result.Result = jobResult
	result.Error = err
	result.Duration = time.Since(start)

	// Update worker statistics
	atomic.AddInt64(&w.stats.JobsProcessed, 1)
	if err != nil {
		atomic.AddInt64(&w.stats.JobsFailed, 1)
	}
	w.stats.TotalJobTime += result.Duration
	w.stats.LastJobTime = time.Now()

	return result
}

// executeJob executes the job with proper error handling
func (w *Worker) executeJob(ctx context.Context, job Job) (interface{}, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before execution")
	default:
	}

	// Execute the job
	done := make(chan struct{})
	var result interface{}
	var err error

	go func() {
		defer close(done)
		result, err = job.Task()
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return result, err
	case <-ctx.Done():
		return nil, fmt.Errorf("job execution timeout: %w", ctx.Err())
	}
}

// stop stops the worker
func (w *Worker) stop() {
	select {
	case w.stopChan <- struct{}{}:
	default:
		// Worker might already be stopped
	}
}

// IsActive returns true if the worker is currently processing a job
func (w *Worker) IsActive() bool {
	return atomic.LoadInt32(&w.active) == 1
}

// GetStats returns the worker's statistics
func (w *Worker) GetStats() WorkerStats {
	return *w.stats
}
