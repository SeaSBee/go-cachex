package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/pool"
)

func main() {
	fmt.Println("=== Go-CacheX Worker Pool Example ===")

	// Demo different configurations
	demoDefaultConfig()
	demoHighPerformanceConfig()
	demoResourceConstrainedConfig()
	demoBackpressure()
	demoAutoScaling()
	demoBatchOperations()
}

func demoDefaultConfig() {
	fmt.Println("1. Default Configuration")
	fmt.Println("   - Min Workers: 2, Max Workers: 10")
	fmt.Println("   - Queue Size: 100")
	fmt.Println("   - Auto-scaling enabled")

	workerPool, err := pool.New(pool.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Submit some jobs
	for i := 0; i < 5; i++ {
		job := pool.Job{
			ID: fmt.Sprintf("default-job-%d", i),
			Task: func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return fmt.Sprintf("default-result-%d", i), nil
			},
		}

		err := workerPool.Submit(job)
		if err != nil {
			log.Printf("Failed to submit job: %v", err)
		}
	}

	// Collect results
	for i := 0; i < 5; i++ {
		result, err := workerPool.GetResultWithTimeout(5 * time.Second)
		if err != nil {
			log.Printf("Failed to get result: %v", err)
			continue
		}
		fmt.Printf("   - Job %s completed: %v\n", result.JobID, result.Result)
	}

	stats := workerPool.GetStats()
	fmt.Printf("   - Stats: %d workers, %d completed jobs, %d failed jobs\n\n",
		stats.TotalWorkers, stats.CompletedJobs, stats.FailedJobs)
}

func demoHighPerformanceConfig() {
	fmt.Println("2. High Performance Configuration")
	fmt.Println("   - Min Workers: 5, Max Workers: 50")
	fmt.Println("   - Queue Size: 1000")
	fmt.Println("   - Optimized for high throughput")

	workerPool, err := pool.New(pool.HighPerformanceConfig())
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Submit many jobs quickly
	const numJobs = 20
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			job := pool.Job{
				ID: fmt.Sprintf("perf-job-%d", id),
				Task: func() (interface{}, error) {
					// Simulate some work
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
					return fmt.Sprintf("perf-result-%d", id), nil
				},
			}

			err := workerPool.Submit(job)
			if err != nil {
				log.Printf("Failed to submit job: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Collect all results
	results := make(map[string]interface{})
	for i := 0; i < numJobs; i++ {
		result, err := workerPool.GetResultWithTimeout(10 * time.Second)
		if err != nil {
			log.Printf("Failed to get result: %v", err)
			continue
		}
		results[result.JobID] = result.Result
	}

	duration := time.Since(start)
	fmt.Printf("   - Completed %d jobs in %v\n", len(results), duration)
	fmt.Printf("   - Average time per job: %v\n\n", duration/time.Duration(len(results)))
}

func demoResourceConstrainedConfig() {
	fmt.Println("3. Resource Constrained Configuration")
	fmt.Println("   - Min Workers: 1, Max Workers: 5")
	fmt.Println("   - Queue Size: 50")
	fmt.Println("   - Optimized for limited resources")

	workerPool, err := pool.New(pool.ResourceConstrainedConfig())
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Submit jobs with limited resources
	for i := 0; i < 3; i++ {
		job := pool.Job{
			ID: fmt.Sprintf("resource-job-%d", i),
			Task: func() (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return fmt.Sprintf("resource-result-%d", i), nil
			},
		}

		err := workerPool.Submit(job)
		if err != nil {
			log.Printf("Failed to submit job: %v", err)
		}
	}

	// Collect results
	for i := 0; i < 3; i++ {
		result, err := workerPool.GetResultWithTimeout(5 * time.Second)
		if err != nil {
			log.Printf("Failed to get result: %v", err)
			continue
		}
		fmt.Printf("   - Job %s completed: %v\n", result.JobID, result.Result)
	}

	stats := workerPool.GetStats()
	fmt.Printf("   - Stats: %d workers, %d completed jobs\n\n",
		stats.TotalWorkers, stats.CompletedJobs)
}

func demoBackpressure() {
	fmt.Println("4. Backpressure Demonstration")
	fmt.Println("   - Small queue size to demonstrate backpressure")

	config := pool.DefaultConfig()
	config.QueueSize = 2
	config.MinWorkers = 1
	config.MaxWorkers = 1

	workerPool, err := pool.New(config)
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Submit a slow job that will block the queue
	slowJob := pool.Job{
		ID: "slow-job",
		Task: func() (interface{}, error) {
			fmt.Println("   - Slow job started (will take 2 seconds)")
			time.Sleep(2 * time.Second)
			fmt.Println("   - Slow job completed")
			return "slow-result", nil
		},
	}

	err = workerPool.Submit(slowJob)
	if err != nil {
		log.Printf("Failed to submit slow job: %v", err)
	}

	// Try to submit more jobs (should be rejected due to backpressure)
	for i := 0; i < 3; i++ {
		fastJob := pool.Job{
			ID: fmt.Sprintf("fast-job-%d", i),
			Task: func() (interface{}, error) {
				return "fast-result", nil
			},
		}

		err := workerPool.Submit(fastJob)
		if err != nil {
			fmt.Printf("   - Job %s rejected: %v\n", fastJob.ID, err)
		} else {
			fmt.Printf("   - Job %s accepted\n", fastJob.ID)
		}
	}

	// Wait for slow job to complete
	result, err := workerPool.GetResultWithTimeout(5 * time.Second)
	if err != nil {
		log.Printf("Failed to get result: %v", err)
	} else {
		fmt.Printf("   - Slow job result: %v\n", result.Result)
	}

	fmt.Println()
}

func demoAutoScaling() {
	fmt.Println("5. Auto-Scaling Demonstration")
	fmt.Println("   - Pool will scale up and down based on load")

	config := pool.DefaultConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 5
	config.ScaleUpThreshold = 2
	config.ScaleDownThreshold = 1
	config.ScaleUpCooldown = 100 * time.Millisecond
	config.ScaleDownCooldown = 200 * time.Millisecond

	workerPool, err := pool.New(config)
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	initialStats := workerPool.GetStats()
	fmt.Printf("   - Initial workers: %d\n", initialStats.TotalWorkers)

	// Submit jobs to trigger scale up
	for i := 0; i < 8; i++ {
		job := pool.Job{
			ID: fmt.Sprintf("scale-job-%d", i),
			Task: func() (interface{}, error) {
				time.Sleep(200 * time.Millisecond)
				return fmt.Sprintf("scale-result-%d", i), nil
			},
		}

		err := workerPool.Submit(job)
		if err != nil {
			log.Printf("Failed to submit job: %v", err)
		}
	}

	// Wait a bit for scaling to occur
	time.Sleep(300 * time.Millisecond)

	scaledStats := workerPool.GetStats()
	fmt.Printf("   - Scaled up workers: %d\n", scaledStats.TotalWorkers)
	fmt.Printf("   - Scale up count: %d\n", scaledStats.ScaleUpCount)

	// Wait for all jobs to complete
	for i := 0; i < 8; i++ {
		_, err := workerPool.GetResultWithTimeout(5 * time.Second)
		if err != nil {
			log.Printf("Failed to get result: %v", err)
		}
	}

	// Wait for scale down
	time.Sleep(500 * time.Millisecond)

	finalStats := workerPool.GetStats()
	fmt.Printf("   - Final workers: %d\n", finalStats.TotalWorkers)
	fmt.Printf("   - Scale down count: %d\n\n", finalStats.ScaleDownCount)
}

func demoBatchOperations() {
	fmt.Println("6. Batch Operations")
	fmt.Println("   - Submit multiple jobs as a batch")

	workerPool, err := pool.New(pool.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Create batch of jobs
	jobs := []pool.Job{
		{
			ID: "batch-1",
			Task: func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return "batch-result-1", nil
			},
		},
		{
			ID: "batch-2",
			Task: func() (interface{}, error) {
				time.Sleep(30 * time.Millisecond)
				return "batch-result-2", nil
			},
		},
		{
			ID: "batch-3",
			Task: func() (interface{}, error) {
				time.Sleep(70 * time.Millisecond)
				return "batch-result-3", nil
			},
		},
	}

	// Submit batch
	errors, batchErr := workerPool.SubmitBatch(jobs)
	if batchErr != nil {
		log.Printf("Batch submission failed: %v", batchErr)
	}

	fmt.Printf("   - Submitted %d jobs\n", len(jobs))
	for i, err := range errors {
		if err != nil {
			fmt.Printf("   - Job %d failed: %v\n", i, err)
		}
	}

	// Collect results
	results := make(map[string]interface{})
	for i := 0; i < len(jobs); i++ {
		result, err := workerPool.GetResultWithTimeout(5 * time.Second)
		if err != nil {
			log.Printf("Failed to get result: %v", err)
			continue
		}
		results[result.JobID] = result.Result
		fmt.Printf("   - %s: %v\n", result.JobID, result.Result)
	}

	fmt.Printf("   - Completed %d jobs\n\n", len(results))
}

func demoContextSupport() {
	fmt.Println("7. Context Support")
	fmt.Println("   - Submit jobs with context cancellation")

	workerPool, err := pool.New(pool.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}
	defer workerPool.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Submit job with context
	job := pool.Job{
		ID: "context-job",
		Task: func() (interface{}, error) {
			time.Sleep(500 * time.Millisecond)
			return "context-result", nil
		},
	}

	err = workerPool.SubmitWithContext(ctx, job)
	if err != nil {
		log.Printf("Failed to submit job with context: %v", err)
		return
	}

	// Wait for result
	result, err := workerPool.GetResultWithTimeout(5 * time.Second)
	if err != nil {
		log.Printf("Failed to get result: %v", err)
		return
	}

	fmt.Printf("   - Job completed: %v\n", result.Result)

	// Try to submit with cancelled context
	cancel()
	err = workerPool.SubmitWithContext(ctx, job)
	if err != nil {
		fmt.Printf("   - Context cancelled submission: %v\n", err)
	}

	fmt.Println()
}
