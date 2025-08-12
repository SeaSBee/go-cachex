package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// PipelineManager provides advanced pipelined operations for Redis
type PipelineManager struct {
	client redis.Cmdable
	config PipelineConfig
}

// PipelineConfig defines pipeline configuration
type PipelineConfig struct {
	// Batching configuration
	BatchSize     int           // Maximum items per batch
	BatchTimeout  time.Duration // Maximum time to wait for batch completion
	MaxConcurrent int           // Maximum concurrent pipeline operations

	// Performance tuning
	FlushInterval time.Duration // How often to flush pending operations
	BufferSize    int           // Size of operation buffer

	// Error handling
	RetryFailedOps bool // Whether to retry failed operations
	MaxRetries     int  // Maximum retry attempts
}

// DefaultPipelineConfig returns a default configuration
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		BatchSize:      100,
		BatchTimeout:   100 * time.Millisecond,
		MaxConcurrent:  10,
		FlushInterval:  50 * time.Millisecond,
		BufferSize:     1000,
		RetryFailedOps: true,
		MaxRetries:     3,
	}
}

// HighPerformancePipelineConfig returns a configuration optimized for high throughput
func HighPerformancePipelineConfig() PipelineConfig {
	return PipelineConfig{
		BatchSize:      500,
		BatchTimeout:   50 * time.Millisecond,
		MaxConcurrent:  20,
		FlushInterval:  25 * time.Millisecond,
		BufferSize:     5000,
		RetryFailedOps: true,
		MaxRetries:     2,
	}
}

// NewPipelineManager creates a new pipeline manager
func NewPipelineManager(client redis.Cmdable, config PipelineConfig) *PipelineManager {
	return &PipelineManager{
		client: client,
		config: config,
	}
}

// BatchGetResult represents the result of a batch get operation
type BatchGetResult struct {
	Key   string
	Value []byte
	Error error
}

// BatchSetResult represents the result of a batch set operation
type BatchSetResult struct {
	Key   string
	Error error
}

// BatchGet performs pipelined GET operations
func (pm *PipelineManager) BatchGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Split keys into batches
	batches := pm.splitIntoBatches(keys, pm.config.BatchSize)
	results := make(map[string][]byte)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	// Process batches concurrently
	semaphore := make(chan struct{}, pm.config.MaxConcurrent)

	for _, batch := range batches {
		wg.Add(1)
		go func(batchKeys []string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			batchResults := pm.executeBatchGet(ctx, batchKeys)

			mu.Lock()
			for key, value := range batchResults {
				results[key] = value
			}
			mu.Unlock()
		}(batch)
	}

	wg.Wait()
	return results, nil
}

// BatchGetAsync performs pipelined GET operations asynchronously
func (pm *PipelineManager) BatchGetAsync(ctx context.Context, keys []string) <-chan BatchGetResult {
	resultChan := make(chan BatchGetResult, len(keys))

	go func() {
		defer close(resultChan)

		if len(keys) == 0 {
			return
		}

		// Split keys into batches
		batches := pm.splitIntoBatches(keys, pm.config.BatchSize)
		var wg sync.WaitGroup

		// Process batches concurrently
		semaphore := make(chan struct{}, pm.config.MaxConcurrent)

		for _, batch := range batches {
			wg.Add(1)
			go func(batchKeys []string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				batchResults := pm.executeBatchGet(ctx, batchKeys)

				for _, key := range batchKeys {
					select {
					case resultChan <- BatchGetResult{
						Key:   key,
						Value: batchResults[key],
						Error: nil,
					}:
					case <-ctx.Done():
						return
					}
				}
			}(batch)
		}

		wg.Wait()
	}()

	return resultChan
}

// BatchSet performs pipelined SET operations
func (pm *PipelineManager) BatchSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Split items into batches
	batches := pm.splitMapIntoBatches(items, pm.config.BatchSize)
	var wg sync.WaitGroup
	errorChan := make(chan error, len(batches))

	// Process batches concurrently
	semaphore := make(chan struct{}, pm.config.MaxConcurrent)

	for _, batch := range batches {
		wg.Add(1)
		go func(batchItems map[string][]byte) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			err := pm.executeBatchSet(ctx, batchItems, ttl)
			if err != nil {
				select {
				case errorChan <- err:
				default:
				}
			}
		}(batch)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// BatchSetAsync performs pipelined SET operations asynchronously
func (pm *PipelineManager) BatchSetAsync(ctx context.Context, items map[string][]byte, ttl time.Duration) <-chan BatchSetResult {
	resultChan := make(chan BatchSetResult, len(items))

	go func() {
		defer close(resultChan)

		if len(items) == 0 {
			return
		}

		// Split items into batches
		batches := pm.splitMapIntoBatches(items, pm.config.BatchSize)
		var wg sync.WaitGroup

		// Process batches concurrently
		semaphore := make(chan struct{}, pm.config.MaxConcurrent)

		for _, batch := range batches {
			wg.Add(1)
			go func(batchItems map[string][]byte) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				err := pm.executeBatchSet(ctx, batchItems, ttl)

				for key := range batchItems {
					select {
					case resultChan <- BatchSetResult{
						Key:   key,
						Error: err,
					}:
					case <-ctx.Done():
						return
					}
				}
			}(batch)
		}

		wg.Wait()
	}()

	return resultChan
}

// PipelineExecutor provides a more advanced pipeline interface
type PipelineExecutor struct {
	client redis.Cmdable
	pipe   redis.Pipeliner
	mu     sync.Mutex
}

// NewPipelineExecutor creates a new pipeline executor
func NewPipelineExecutor(client redis.Cmdable) *PipelineExecutor {
	return &PipelineExecutor{
		client: client,
		pipe:   client.Pipeline(),
	}
}

// AddGet adds a GET command to the pipeline
func (pe *PipelineExecutor) AddGet(ctx context.Context, key string) *redis.StringCmd {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.pipe.Get(ctx, key)
}

// AddSet adds a SET command to the pipeline
func (pe *PipelineExecutor) AddSet(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.pipe.Set(ctx, key, value, ttl)
}

// AddMGet adds an MGET command to the pipeline
func (pe *PipelineExecutor) AddMGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.pipe.MGet(ctx, keys...)
}

// AddMSet adds an MSET command to the pipeline
func (pe *PipelineExecutor) AddMSet(ctx context.Context, items map[string]interface{}) *redis.StatusCmd {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.pipe.MSet(ctx, items)
}

// AddDel adds a DEL command to the pipeline
func (pe *PipelineExecutor) AddDel(ctx context.Context, keys ...string) *redis.IntCmd {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.pipe.Del(ctx, keys...)
}

// Execute executes all commands in the pipeline
func (pe *PipelineExecutor) Execute(ctx context.Context) ([]redis.Cmder, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	cmds, err := pe.pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Reset pipeline for next use
	pe.pipe = pe.client.Pipeline()

	return cmds, nil
}

// executeBatchGet executes a batch of GET operations using pipeline
func (pm *PipelineManager) executeBatchGet(ctx context.Context, keys []string) map[string][]byte {
	if len(keys) == 0 {
		return make(map[string][]byte)
	}

	// Create pipeline
	pipe := pm.client.Pipeline()

	// Add GET commands to pipeline
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Return empty results on error
		return make(map[string][]byte)
	}

	// Process results
	results := make(map[string][]byte)
	for i, cmd := range cmds {
		if cmd.Err() == nil {
			results[keys[i]] = []byte(cmd.Val())
		}
	}

	return results
}

// executeBatchSet executes a batch of SET operations using pipeline
func (pm *PipelineManager) executeBatchSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Create pipeline
	pipe := pm.client.Pipeline()

	// Add SET commands to pipeline
	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// splitIntoBatches splits a slice into batches of specified size
func (pm *PipelineManager) splitIntoBatches(items []string, batchSize int) [][]string {
	if batchSize <= 0 {
		batchSize = 1
	}

	var batches [][]string
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	return batches
}

// splitMapIntoBatches splits a map into batches of specified size
func (pm *PipelineManager) splitMapIntoBatches(items map[string][]byte, batchSize int) []map[string][]byte {
	if batchSize <= 0 {
		batchSize = 1
	}

	var batches []map[string][]byte
	currentBatch := make(map[string][]byte)
	count := 0

	for key, value := range items {
		currentBatch[key] = value
		count++

		if count >= batchSize {
			batches = append(batches, currentBatch)
			currentBatch = make(map[string][]byte)
			count = 0
		}
	}

	// Add remaining items
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// BatchProcessor provides a high-level interface for batch processing
type BatchProcessor struct {
	pipelineManager *PipelineManager
	config          BatchProcessorConfig
}

// BatchProcessorConfig defines batch processor configuration
type BatchProcessorConfig struct {
	// Processing configuration
	MaxBatchSize    int           // Maximum items per batch
	ProcessInterval time.Duration // How often to process batches
	MaxWorkers      int           // Maximum number of worker goroutines

	// Error handling
	RetryOnError bool          // Whether to retry on error
	RetryDelay   time.Duration // Delay between retries
	MaxRetries   int           // Maximum retry attempts
}

// DefaultBatchProcessorConfig returns a default configuration
func DefaultBatchProcessorConfig() BatchProcessorConfig {
	return BatchProcessorConfig{
		MaxBatchSize:    100,
		ProcessInterval: 100 * time.Millisecond,
		MaxWorkers:      5,
		RetryOnError:    true,
		RetryDelay:      1 * time.Second,
		MaxRetries:      3,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(pipelineManager *PipelineManager, config BatchProcessorConfig) *BatchProcessor {
	return &BatchProcessor{
		pipelineManager: pipelineManager,
		config:          config,
	}
}

// ProcessGetRequests processes GET requests in batches
func (bp *BatchProcessor) ProcessGetRequests(ctx context.Context, requests <-chan string, results chan<- BatchGetResult) {
	defer close(results)

	var batch []string
	ticker := time.NewTicker(bp.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case key, ok := <-requests:
			if !ok {
				// Process remaining batch
				if len(batch) > 0 {
					bp.processGetBatch(ctx, batch, results)
				}
				return
			}

			batch = append(batch, key)

			// Process batch if it reaches max size
			if len(batch) >= bp.config.MaxBatchSize {
				bp.processGetBatch(ctx, batch, results)
				batch = batch[:0] // Reset batch
			}

		case <-ticker.C:
			// Process batch on timer
			if len(batch) > 0 {
				bp.processGetBatch(ctx, batch, results)
				batch = batch[:0] // Reset batch
			}

		case <-ctx.Done():
			return
		}
	}
}

// processGetBatch processes a batch of GET requests
func (bp *BatchProcessor) processGetBatch(ctx context.Context, keys []string, results chan<- BatchGetResult) {
	// Retry logic
	for attempt := 0; attempt <= bp.config.MaxRetries; attempt++ {
		batchResults, err := bp.pipelineManager.BatchGet(ctx, keys)

		if err == nil {
			// Send results
			for _, key := range keys {
				select {
				case results <- BatchGetResult{
					Key:   key,
					Value: batchResults[key],
					Error: nil,
				}:
				case <-ctx.Done():
					return
				}
			}
			return
		}

		// Retry on error
		if attempt < bp.config.MaxRetries && bp.config.RetryOnError {
			select {
			case <-time.After(bp.config.RetryDelay):
			case <-ctx.Done():
				return
			}
			continue
		}

		// Send error results
		for _, key := range keys {
			select {
			case results <- BatchGetResult{
				Key:   key,
				Value: nil,
				Error: fmt.Errorf("failed to get key %s after %d attempts: %w", key, attempt+1, err),
			}:
			case <-ctx.Done():
				return
			}
		}
	}
}
