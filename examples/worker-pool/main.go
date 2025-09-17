package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/seasbee/go-logx"
)

// Simple worker pool implementation
type WorkerPool struct {
	workers    int
	jobQueue   chan Job
	resultChan chan JobResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

type Job struct {
	ID   string
	Task func() (interface{}, error)
}

type JobResult struct {
	JobID  string
	Result interface{}
	Error  error
}

func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:    workers,
		jobQueue:   make(chan Job, 100),
		resultChan: make(chan JobResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for {
		select {
		case job := <-wp.jobQueue:
			result, err := job.Task()
			wp.resultChan <- JobResult{
				JobID:  job.ID,
				Result: result,
				Error:  err,
			}
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobQueue <- job:
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool closed")
	default:
		return fmt.Errorf("job queue full")
	}
}

func (wp *WorkerPool) GetResult() (JobResult, error) {
	select {
	case result := <-wp.resultChan:
		return result, nil
	case <-wp.ctx.Done():
		return JobResult{}, fmt.Errorf("worker pool closed")
	}
}

func (wp *WorkerPool) Close() {
	wp.cancel()
	wp.wg.Wait()
	close(wp.jobQueue)
	close(wp.resultChan)
}

func main() {
	logx.Info("=== Go-CacheX Worker Pool Example ===")

	// Demo different configurations
	demoBasicWorkerPool()
	demoCacheWithWorkerPool()
	demoConcurrentCacheOperations()
	demoBatchCacheOperations()
}

func demoBasicWorkerPool() {
	logx.Info("1. Basic Worker Pool")
	logx.Info("====================")

	// Create worker pool
	pool := NewWorkerPool(3)
	defer pool.Close()

	logx.Info("✓ Worker pool created with 3 workers")

	// Submit some jobs
	for i := 0; i < 5; i++ {
		job := Job{
			ID: fmt.Sprintf("job-%d", i),
			Task: func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return fmt.Sprintf("result-%d", i), nil
			},
		}

		if err := pool.Submit(job); err != nil {
			logx.Error("Failed to submit job",
				logx.ErrorField(err))
		}
	}

	// Collect results
	for i := 0; i < 5; i++ {
		result, err := pool.GetResult()
		if err != nil {
			logx.Error("Failed to get result",
				logx.ErrorField(err))
			continue
		}
		logx.Info("✓ Job completed",
			logx.String("jobID", result.JobID),
			logx.Any("result", result.Result))
	}
}

func demoCacheWithWorkerPool() {
	logx.Info("2. Cache with Worker Pool")
	logx.Info("=========================")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cachex.New[string](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create worker pool
	pool := NewWorkerPool(4)
	defer pool.Close()

	logx.Info("✓ Cache and worker pool created")

	// Submit cache operations as jobs
	for i := 0; i < 6; i++ {
		key := fmt.Sprintf("user:%d", i)
		value := fmt.Sprintf("User %d", i)

		job := Job{
			ID: fmt.Sprintf("cache-job-%d", i),
			Task: func() (interface{}, error) {
				ctx := context.Background()

				// Set value in cache
				setResult := <-c.Set(ctx, key, value, 0)
				if setResult.Error != nil {
					return nil, fmt.Errorf("failed to set: %v", setResult.Error)
				}

				// Get value from cache
				getResult := <-c.Get(ctx, key)
				if getResult.Error != nil {
					return nil, fmt.Errorf("failed to get: %v", getResult.Error)
				}

				if !getResult.Found {
					return nil, fmt.Errorf("value not found")
				}

				return fmt.Sprintf("Cached: %s", getResult.Value), nil
			},
		}

		if err := pool.Submit(job); err != nil {
			logx.Error("Failed to submit job",
				logx.ErrorField(err))
		}
	}

	// Collect results
	for i := 0; i < 6; i++ {
		result, err := pool.GetResult()
		if err != nil {
			logx.Error("Failed to get result",
				logx.ErrorField(err))
			continue
		}
		logx.Info("✓ Job completed",
			logx.String("jobID", result.JobID),
			logx.Any("result", result.Result))
	}
}

func demoConcurrentCacheOperations() {
	logx.Info("3. Concurrent Cache Operations")
	logx.Info("==============================")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cachex.New[int](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create worker pool
	pool := NewWorkerPool(8)
	defer pool.Close()

	logx.Info("✓ Cache and worker pool created for concurrent operations")

	// Submit concurrent increment operations
	const numOperations = 20
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			job := Job{
				ID: fmt.Sprintf("concurrent-job-%d", id),
				Task: func() (interface{}, error) {
					ctx := context.Background()
					key := "counter"

					// Get current value
					getResult := <-c.Get(ctx, key)
					if getResult.Error != nil {
						return nil, fmt.Errorf("failed to get counter: %v", getResult.Error)
					}

					value := 0
					if getResult.Found {
						value = getResult.Value
					}

					// Increment
					value++

					// Set new value
					setResult := <-c.Set(ctx, key, value, 0)
					if setResult.Error != nil {
						return nil, fmt.Errorf("failed to set counter: %v", setResult.Error)
					}

					return value, nil
				},
			}

			if err := pool.Submit(job); err != nil {
				logx.Error("Failed to submit job",
					logx.ErrorField(err))
			}
		}(i)
	}

	wg.Wait()

	// Collect results
	results := make([]int, 0, numOperations)
	for i := 0; i < numOperations; i++ {
		result, err := pool.GetResult()
		if err != nil {
			logx.Error("Failed to get result",
				logx.ErrorField(err))
			continue
		}
		if value, ok := result.Result.(int); ok {
			results = append(results, value)
		}
	}

	duration := time.Since(start)
	logx.Info("✓ Completed concurrent operations",
		logx.Int("count", len(results)),
		logx.String("duration", duration.String()))
	logx.Info("✓ Final counter value",
		logx.Int("value", len(results)))

	// Verify final value
	ctx := context.Background()
	finalResult := <-c.Get(ctx, "counter")
	if finalResult.Error != nil {
		logx.Error("Failed to get final counter",
			logx.ErrorField(finalResult.Error))
	} else if finalResult.Found {
		logx.Info("✓ Cache shows final value",
			logx.Int("value", finalResult.Value))
	}
}

func demoBatchCacheOperations() {
	logx.Info("4. Batch Cache Operations")
	logx.Info("==========================")

	// Create memory store
	memoryStore, err := cachex.NewMemoryStore(cachex.DefaultMemoryConfig())
	if err != nil {
		logx.Error("Failed to create memory store",
			logx.ErrorField(err))
		return
	}
	defer memoryStore.Close()

	// Create cache
	c, err := cachex.New[string](
		cachex.WithStore(memoryStore),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		logx.Error("Failed to create cache",
			logx.ErrorField(err))
		return
	}
	defer c.Close()

	// Create worker pool
	pool := NewWorkerPool(5)
	defer pool.Close()

	logx.Info("✓ Cache and worker pool created for batch operations")

	// Create batch of jobs
	jobs := []Job{
		{
			ID: "batch-set-1",
			Task: func() (interface{}, error) {
				ctx := context.Background()
				setResult := <-c.Set(ctx, "user:1", "Alice", 0)
				return "Set user:1", setResult.Error
			},
		},
		{
			ID: "batch-set-2",
			Task: func() (interface{}, error) {
				ctx := context.Background()
				setResult := <-c.Set(ctx, "user:2", "Bob", 0)
				return "Set user:2", setResult.Error
			},
		},
		{
			ID: "batch-set-3",
			Task: func() (interface{}, error) {
				ctx := context.Background()
				setResult := <-c.Set(ctx, "user:3", "Charlie", 0)
				return "Set user:3", setResult.Error
			},
		},
		{
			ID: "batch-get-1",
			Task: func() (interface{}, error) {
				ctx := context.Background()
				getResult := <-c.Get(ctx, "user:1")
				if getResult.Error != nil {
					return nil, getResult.Error
				}
				if !getResult.Found {
					return nil, fmt.Errorf("user:1 not found")
				}
				return fmt.Sprintf("Retrieved: %s", getResult.Value), nil
			},
		},
		{
			ID: "batch-get-2",
			Task: func() (interface{}, error) {
				ctx := context.Background()
				getResult := <-c.Get(ctx, "user:2")
				if getResult.Error != nil {
					return nil, getResult.Error
				}
				if !getResult.Found {
					return nil, fmt.Errorf("user:2 not found")
				}
				return fmt.Sprintf("Retrieved: %s", getResult.Value), nil
			},
		},
	}

	// Submit all jobs
	for _, job := range jobs {
		if err := pool.Submit(job); err != nil {
			logx.Error("Failed to submit job",
				logx.String("jobID", job.ID),
				logx.ErrorField(err))
		}
	}

	// Collect results
	logx.Info("✓ Submitted batch jobs",
		logx.Int("count", len(jobs)))
	for i := 0; i < len(jobs); i++ {
		result, err := pool.GetResult()
		if err != nil {
			logx.Error("Failed to get result",
				logx.ErrorField(err))
			continue
		}
		logx.Info("✓ Job completed",
			logx.String("jobID", result.JobID),
			logx.Any("result", result.Result))
	}
}
