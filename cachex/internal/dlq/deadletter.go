package dlq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// DeadLetterQueue manages failed operations with retry capabilities
type DeadLetterQueue struct {
	mu          sync.RWMutex
	queue       map[string]*FailedOperation
	maxRetries  int
	retryDelay  time.Duration
	metrics     *Metrics
	handlers    map[string]RetryHandler
	ctx         context.Context
	cancel      context.CancelFunc
	workerCount int
	workerChan  chan *FailedOperation
}

// FailedOperation represents a failed operation
type FailedOperation struct {
	ID          string                 `json:"id"`
	Operation   string                 `json:"operation"`
	Key         string                 `json:"key"`
	Data        interface{}            `json:"data"`
	Error       string                 `json:"error"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	LastRetry   time.Time              `json:"last_retry"`
	NextRetry   time.Time              `json:"next_retry"`
	Metadata    map[string]interface{} `json:"metadata"`
	HandlerType string                 `json:"handler_type"`
}

// RetryHandler defines the interface for retrying failed operations
type RetryHandler interface {
	Retry(ctx context.Context, operation *FailedOperation) error
	ShouldRetry(operation *FailedOperation) bool
}

// Metrics tracks dead-letter queue statistics
type Metrics struct {
	mu             sync.RWMutex
	TotalFailed    int64           `json:"total_failed"`
	TotalRetried   int64           `json:"total_retried"`
	TotalSucceeded int64           `json:"total_succeeded"`
	TotalDropped   int64           `json:"total_dropped"`
	CurrentQueue   int64           `json:"current_queue"`
	AverageRetries float64         `json:"average_retries"`
	LastRetryTime  time.Time       `json:"last_retry_time"`
	RetryDurations []time.Duration `json:"retry_durations"`
}

// Config holds dead-letter queue configuration
type Config struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	WorkerCount   int           `json:"worker_count"`
	QueueSize     int           `json:"queue_size"`
	EnableMetrics bool          `json:"enable_metrics"`
}

// NewDeadLetterQueue creates a new dead-letter queue
func NewDeadLetterQueue(config *Config) *DeadLetterQueue {
	if config == nil {
		config = &Config{
			MaxRetries:    3,
			RetryDelay:    5 * time.Minute,
			WorkerCount:   2,
			QueueSize:     1000,
			EnableMetrics: true,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	dlq := &DeadLetterQueue{
		queue:       make(map[string]*FailedOperation),
		maxRetries:  config.MaxRetries,
		retryDelay:  config.RetryDelay,
		metrics:     &Metrics{},
		handlers:    make(map[string]RetryHandler),
		ctx:         ctx,
		cancel:      cancel,
		workerCount: config.WorkerCount,
		workerChan:  make(chan *FailedOperation, config.QueueSize),
	}

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		go dlq.worker(i)
	}

	return dlq
}

// AddFailedOperation adds a failed operation to the dead-letter queue
func (dlq *DeadLetterQueue) AddFailedOperation(operation *FailedOperation) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Set default values
	if operation.ID == "" {
		operation.ID = fmt.Sprintf("%s-%d", operation.Key, time.Now().UnixNano())
	}
	if operation.CreatedAt.IsZero() {
		operation.CreatedAt = time.Now()
	}
	if operation.MaxRetries == 0 {
		operation.MaxRetries = dlq.maxRetries
	}
	if operation.NextRetry.IsZero() {
		operation.NextRetry = time.Now().Add(dlq.retryDelay)
	}

	// Add to queue
	dlq.queue[operation.ID] = operation

	// Update metrics
	if dlq.metrics != nil {
		dlq.metrics.mu.Lock()
		dlq.metrics.TotalFailed++
		dlq.metrics.CurrentQueue++
		dlq.metrics.mu.Unlock()
	}

	// Send to worker channel for processing
	select {
	case dlq.workerChan <- operation:
		logx.Info("Failed operation added to DLQ",
			logx.String("id", operation.ID),
			logx.String("operation", operation.Operation),
			logx.String("key", operation.Key))
	default:
		logx.Error("DLQ worker channel full, operation dropped",
			logx.String("id", operation.ID))
	}

	return nil
}

// RegisterHandler registers a retry handler for a specific operation type
func (dlq *DeadLetterQueue) RegisterHandler(operationType string, handler RetryHandler) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	dlq.handlers[operationType] = handler
}

// worker processes failed operations
func (dlq *DeadLetterQueue) worker(id int) {
	logx.Info("DLQ worker started", logx.Int("worker_id", id))

	for {
		select {
		case operation := <-dlq.workerChan:
			dlq.processOperation(operation)
		case <-dlq.ctx.Done():
			logx.Info("DLQ worker stopped", logx.Int("worker_id", id))
			return
		}
	}
}

// processOperation processes a single failed operation
func (dlq *DeadLetterQueue) processOperation(operation *FailedOperation) {
	// Wait until next retry time
	if time.Now().Before(operation.NextRetry) {
		time.Sleep(time.Until(operation.NextRetry))
	}

	// Check if we should retry
	if operation.RetryCount >= operation.MaxRetries {
		dlq.dropOperation(operation, "max retries exceeded")
		return
	}

	// Get handler for operation type
	dlq.mu.RLock()
	handler, exists := dlq.handlers[operation.HandlerType]
	dlq.mu.RUnlock()

	if !exists {
		dlq.dropOperation(operation, "no handler registered")
		return
	}

	// Check if handler thinks we should retry
	if !handler.ShouldRetry(operation) {
		dlq.dropOperation(operation, "handler rejected retry")
		return
	}

	// Attempt retry
	operation.RetryCount++
	operation.LastRetry = time.Now()
	startTime := time.Now()

	err := handler.Retry(dlq.ctx, operation)

	// Update metrics
	if dlq.metrics != nil {
		dlq.metrics.mu.Lock()
		dlq.metrics.TotalRetried++
		dlq.metrics.LastRetryTime = time.Now()
		dlq.metrics.RetryDurations = append(dlq.metrics.RetryDurations, time.Since(startTime))
		dlq.metrics.mu.Unlock()
	}

	if err == nil {
		// Success - remove from queue
		dlq.succeedOperation(operation)
	} else {
		// Failed - schedule next retry
		operation.Error = err.Error()
		operation.NextRetry = time.Now().Add(dlq.retryDelay * time.Duration(operation.RetryCount))

		logx.Error("Retry failed",
			logx.String("id", operation.ID),
			logx.String("operation", operation.Operation),
			logx.String("key", operation.Key),
			logx.Int("retry_count", operation.RetryCount),
			logx.ErrorField(err))

		// Re-queue for next retry
		select {
		case dlq.workerChan <- operation:
		default:
			dlq.dropOperation(operation, "worker channel full")
		}
	}
}

// succeedOperation marks an operation as successful
func (dlq *DeadLetterQueue) succeedOperation(operation *FailedOperation) {
	dlq.mu.Lock()
	delete(dlq.queue, operation.ID)
	dlq.mu.Unlock()

	if dlq.metrics != nil {
		dlq.metrics.mu.Lock()
		dlq.metrics.TotalSucceeded++
		dlq.metrics.CurrentQueue--
		dlq.metrics.mu.Unlock()
	}

	logx.Info("Operation retry succeeded",
		logx.String("id", operation.ID),
		logx.String("operation", operation.Operation),
		logx.String("key", operation.Key),
		logx.Int("retry_count", operation.RetryCount))
}

// dropOperation drops an operation from the queue
func (dlq *DeadLetterQueue) dropOperation(operation *FailedOperation, reason string) {
	dlq.mu.Lock()
	delete(dlq.queue, operation.ID)
	dlq.mu.Unlock()

	if dlq.metrics != nil {
		dlq.metrics.mu.Lock()
		dlq.metrics.TotalDropped++
		dlq.metrics.CurrentQueue--
		dlq.metrics.mu.Unlock()
	}

	logx.Warn("Operation dropped from DLQ",
		logx.String("id", operation.ID),
		logx.String("operation", operation.Operation),
		logx.String("key", operation.Key),
		logx.String("reason", reason),
		logx.Int("retry_count", operation.RetryCount))
}

// GetMetrics returns current metrics
func (dlq *DeadLetterQueue) GetMetrics() *Metrics {
	if dlq.metrics == nil {
		return nil
	}

	dlq.metrics.mu.RLock()
	defer dlq.metrics.mu.RUnlock()

	// Calculate average retries
	var totalRetries int64
	for _, op := range dlq.queue {
		totalRetries += int64(op.RetryCount)
	}

	avgRetries := 0.0
	if dlq.metrics.CurrentQueue > 0 {
		avgRetries = float64(totalRetries) / float64(dlq.metrics.CurrentQueue)
	}

	metrics := &Metrics{
		TotalFailed:    dlq.metrics.TotalFailed,
		TotalRetried:   dlq.metrics.TotalRetried,
		TotalSucceeded: dlq.metrics.TotalSucceeded,
		TotalDropped:   dlq.metrics.TotalDropped,
		CurrentQueue:   dlq.metrics.CurrentQueue,
		AverageRetries: avgRetries,
		LastRetryTime:  dlq.metrics.LastRetryTime,
		RetryDurations: make([]time.Duration, len(dlq.metrics.RetryDurations)),
	}
	copy(metrics.RetryDurations, dlq.metrics.RetryDurations)

	return metrics
}

// GetQueue returns all operations in the queue
func (dlq *DeadLetterQueue) GetQueue() []*FailedOperation {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	operations := make([]*FailedOperation, 0, len(dlq.queue))
	for _, op := range dlq.queue {
		operations = append(operations, op)
	}

	return operations
}

// ClearQueue clears all operations from the queue
func (dlq *DeadLetterQueue) ClearQueue() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	count := len(dlq.queue)
	dlq.queue = make(map[string]*FailedOperation)

	if dlq.metrics != nil {
		dlq.metrics.mu.Lock()
		dlq.metrics.CurrentQueue = 0
		dlq.metrics.mu.Unlock()
	}

	logx.Info("DLQ queue cleared", logx.Int("operations_cleared", count))
}

// Close closes the dead-letter queue
func (dlq *DeadLetterQueue) Close() error {
	dlq.cancel()

	// Wait a bit for workers to finish
	time.Sleep(100 * time.Millisecond)

	logx.Info("Dead-letter queue closed")
	return nil
}

// WriteBehindHandler implements RetryHandler for write-behind operations
type WriteBehindHandler struct {
	writer func(ctx context.Context, key string, data interface{}) error
}

// NewWriteBehindHandler creates a new write-behind handler
func NewWriteBehindHandler(writer func(ctx context.Context, key string, data interface{}) error) *WriteBehindHandler {
	return &WriteBehindHandler{
		writer: writer,
	}
}

// Retry retries a write-behind operation
func (h *WriteBehindHandler) Retry(ctx context.Context, operation *FailedOperation) error {
	return h.writer(ctx, operation.Key, operation.Data)
}

// ShouldRetry determines if a write-behind operation should be retried
func (h *WriteBehindHandler) ShouldRetry(operation *FailedOperation) bool {
	// Retry write-behind operations unless they've exceeded max retries
	return operation.RetryCount < operation.MaxRetries
}
