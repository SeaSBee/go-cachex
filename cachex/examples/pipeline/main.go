package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/pipeline"
	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("=== Go-CacheX Pipeline Example ===")

	// Create a mock Redis client for demonstration
	// In production, you would use a real Redis client
	mockClient := createMockRedisClient()

	// Demo pipeline functionality
	demoPipelineManager(mockClient)
	demoPipelineExecutor(mockClient)
	demoBatchProcessor(mockClient)
}

func createMockRedisClient() redis.Cmdable {
	// This is a simplified mock for demonstration
	// In production, you would use: redis.NewClient(&redis.Options{...})
	return nil
}

func demoPipelineManager(client redis.Cmdable) {
	fmt.Println("1. Pipeline Manager")
	fmt.Println("   - Advanced pipelined operations")
	fmt.Println("   - Automatic batching and concurrency")

	// Create pipeline manager
	config := pipeline.DefaultPipelineConfig()
	_ = pipeline.NewPipelineManager(client, config)

	fmt.Printf("   - Batch Size: %d\n", config.BatchSize)
	fmt.Printf("   - Max Concurrent: %d\n", config.MaxConcurrent)
	fmt.Printf("   - Batch Timeout: %v\n", config.BatchTimeout)
	fmt.Println()
}

func demoPipelineExecutor(client redis.Cmdable) {
	fmt.Println("2. Pipeline Executor")
	fmt.Println("   - Manual pipeline control")
	fmt.Println("   - Add commands and execute")

	// Create pipeline executor
	pe := pipeline.NewPipelineExecutor(client)

	ctx := context.Background()

	// Add commands to pipeline
	pe.AddGet(ctx, "user:1")
	pe.AddSet(ctx, "user:2", "value2", time.Minute)
	pe.AddDel(ctx, "temp:key")

	fmt.Println("   - Added GET, SET, and DEL commands to pipeline")
	fmt.Println("   - Ready to execute all commands at once")
	fmt.Println()
}

func demoBatchProcessor(client redis.Cmdable) {
	fmt.Println("3. Batch Processor")
	fmt.Println("   - High-level batch processing")
	fmt.Println("   - Automatic request batching")

	// Create pipeline manager and batch processor
	pm := pipeline.NewPipelineManager(client, pipeline.DefaultPipelineConfig())
	config := pipeline.DefaultBatchProcessorConfig()
	_ = pipeline.NewBatchProcessor(pm, config)

	fmt.Printf("   - Max Batch Size: %d\n", config.MaxBatchSize)
	fmt.Printf("   - Process Interval: %v\n", config.ProcessInterval)
	fmt.Printf("   - Max Workers: %d\n", config.MaxWorkers)
	fmt.Println("   - Automatically batches requests and processes them efficiently")
	fmt.Println()
}
