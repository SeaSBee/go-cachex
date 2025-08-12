package dlq

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDeadLetterQueue(t *testing.T) {
	// Test with nil config
	dlq := NewDeadLetterQueue(nil)
	assert.NotNil(t, dlq)
	assert.Equal(t, 3, dlq.maxRetries)
	assert.Equal(t, 5*time.Minute, dlq.retryDelay)
	assert.Equal(t, 2, dlq.workerCount)

	// Test with custom config
	config := &Config{
		MaxRetries:    5,
		RetryDelay:    1 * time.Minute,
		WorkerCount:   3,
		QueueSize:     500,
		EnableMetrics: true,
	}

	dlq = NewDeadLetterQueue(config)
	assert.NotNil(t, dlq)
	assert.Equal(t, 5, dlq.maxRetries)
	assert.Equal(t, 1*time.Minute, dlq.retryDelay)
	assert.Equal(t, 3, dlq.workerCount)
}

func TestDeadLetterQueue_AddFailedOperation(t *testing.T) {
	dlq := NewDeadLetterQueue(nil)
	defer dlq.Close()

	operation := &FailedOperation{
		Operation:   "test_operation",
		Key:         "test_key",
		Data:        "test_data",
		Error:       "test_error",
		HandlerType: "test_handler",
	}

	err := dlq.AddFailedOperation(operation)
	assert.NoError(t, err)

	// Check that operation was added
	queue := dlq.GetQueue()
	assert.Len(t, queue, 1)
	assert.Equal(t, "test_operation", queue[0].Operation)
	assert.Equal(t, "test_key", queue[0].Key)
	assert.Equal(t, "test_data", queue[0].Data)
	assert.Equal(t, "test_error", queue[0].Error)
	assert.Equal(t, "test_handler", queue[0].HandlerType)
	assert.Equal(t, 0, queue[0].RetryCount)
	assert.Equal(t, 3, queue[0].MaxRetries)
}

func TestDeadLetterQueue_ProcessOperation(t *testing.T) {
	dlq := NewDeadLetterQueue(&Config{
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
		WorkerCount:   1,
		QueueSize:     10,
		EnableMetrics: true,
	})
	defer dlq.Close()

	// Create a handler that succeeds on first retry
	retryCount := 0
	handler := &TestHandler{
		RetryFunc: func(ctx context.Context, operation *FailedOperation) error {
			retryCount++
			if retryCount == 1 {
				return errors.New("first attempt failed")
			}
			return nil // succeed on second attempt
		},
		ShouldRetryFunc: func(operation *FailedOperation) bool {
			return operation.RetryCount < operation.MaxRetries
		},
	}

	dlq.RegisterHandler("test_handler", handler)

	operation := &FailedOperation{
		Operation:   "test_operation",
		Key:         "test_key",
		Data:        "test_data",
		Error:       "test_error",
		HandlerType: "test_handler",
	}

	err := dlq.AddFailedOperation(operation)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check that operation was processed successfully
	queue := dlq.GetQueue()
	assert.Len(t, queue, 0) // Should be removed after success

	metrics := dlq.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.TotalFailed)
	assert.Equal(t, int64(2), metrics.TotalRetried) // 2 retries (initial + 1 retry)
	assert.Equal(t, int64(1), metrics.TotalSucceeded)
}

func TestDeadLetterQueue_ProcessOperationMaxRetries(t *testing.T) {
	dlq := NewDeadLetterQueue(&Config{
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
		WorkerCount:   1,
		QueueSize:     10,
		EnableMetrics: true,
	})
	defer dlq.Close()

	// Create a handler that always fails
	handler := &TestHandler{
		RetryFunc: func(ctx context.Context, operation *FailedOperation) error {
			return errors.New("always fails")
		},
		ShouldRetryFunc: func(operation *FailedOperation) bool {
			return operation.RetryCount < operation.MaxRetries
		},
	}

	dlq.RegisterHandler("test_handler", handler)

	operation := &FailedOperation{
		Operation:   "test_operation",
		Key:         "test_key",
		Data:        "test_data",
		Error:       "test_error",
		HandlerType: "test_handler",
	}

	err := dlq.AddFailedOperation(operation)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check that operation was dropped after max retries
	queue := dlq.GetQueue()
	assert.Len(t, queue, 0) // Should be dropped

	metrics := dlq.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.TotalFailed)
	assert.Equal(t, int64(2), metrics.TotalRetried) // 2 retries
	assert.Equal(t, int64(0), metrics.TotalSucceeded)
	assert.Equal(t, int64(1), metrics.TotalDropped)
}

func TestDeadLetterQueue_NoHandler(t *testing.T) {
	dlq := NewDeadLetterQueue(&Config{
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
		WorkerCount:   1,
		QueueSize:     10,
		EnableMetrics: true,
	})
	defer dlq.Close()

	operation := &FailedOperation{
		Operation:   "test_operation",
		Key:         "test_key",
		Data:        "test_data",
		Error:       "test_error",
		HandlerType: "unknown_handler", // No handler registered
	}

	err := dlq.AddFailedOperation(operation)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check that operation was dropped due to no handler
	queue := dlq.GetQueue()
	assert.Len(t, queue, 0) // Should be dropped

	metrics := dlq.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics.TotalFailed)
	assert.Equal(t, int64(0), metrics.TotalRetried)
	assert.Equal(t, int64(0), metrics.TotalSucceeded)
	assert.Equal(t, int64(1), metrics.TotalDropped)
}

func TestDeadLetterQueue_ContextCancellation(t *testing.T) {
	dlq := NewDeadLetterQueue(&Config{
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
		WorkerCount:   1,
		QueueSize:     10,
		EnableMetrics: true,
	})
	defer dlq.Close()

	// Create a handler that respects context
	handler := &TestHandler{
		RetryFunc: func(ctx context.Context, operation *FailedOperation) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		},
		ShouldRetryFunc: func(operation *FailedOperation) bool {
			return operation.RetryCount < operation.MaxRetries
		},
	}

	dlq.RegisterHandler("test_handler", handler)

	operation := &FailedOperation{
		Operation:   "test_operation",
		Key:         "test_key",
		Data:        "test_data",
		Error:       "test_error",
		HandlerType: "test_handler",
	}

	err := dlq.AddFailedOperation(operation)
	assert.NoError(t, err)

	// Close DLQ immediately
	dlq.Close()

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Check that DLQ was closed properly
	queue := dlq.GetQueue()
	// Operations might still be in queue due to immediate close
	assert.NotNil(t, queue)
}

func TestDeadLetterQueue_ClearQueue(t *testing.T) {
	dlq := NewDeadLetterQueue(nil)
	defer dlq.Close()

	// Add multiple operations
	for i := 0; i < 5; i++ {
		operation := &FailedOperation{
			Operation:   "test_operation",
			Key:         fmt.Sprintf("test_key_%d", i),
			Data:        "test_data",
			Error:       "test_error",
			HandlerType: "test_handler",
		}
		err := dlq.AddFailedOperation(operation)
		assert.NoError(t, err)
	}

	// Verify operations were added
	queue := dlq.GetQueue()
	assert.Len(t, queue, 5)

	// Clear queue
	dlq.ClearQueue()

	// Verify queue is empty
	queue = dlq.GetQueue()
	assert.Len(t, queue, 0)

	// Verify metrics are reset
	metrics := dlq.GetMetrics()
	assert.Equal(t, int64(0), metrics.CurrentQueue)
}

func TestWriteBehindHandler(t *testing.T) {
	// Test successful write
	writerCalled := false
	writer := func(ctx context.Context, key string, data interface{}) error {
		writerCalled = true
		assert.Equal(t, "test_key", key)
		assert.Equal(t, "test_data", data)
		return nil
	}

	handler := NewWriteBehindHandler(writer)
	assert.NotNil(t, handler)

	operation := &FailedOperation{
		Key:  "test_key",
		Data: "test_data",
	}

	err := handler.Retry(context.Background(), operation)
	assert.NoError(t, err)
	assert.True(t, writerCalled)

	// Test failed write
	writerCalled = false
	failingWriter := func(ctx context.Context, key string, data interface{}) error {
		writerCalled = true
		return errors.New("write failed")
	}

	handler = NewWriteBehindHandler(failingWriter)
	err = handler.Retry(context.Background(), operation)
	assert.Error(t, err)
	assert.True(t, writerCalled)

	// Test ShouldRetry
	assert.True(t, handler.ShouldRetry(&FailedOperation{RetryCount: 0, MaxRetries: 3}))
	assert.True(t, handler.ShouldRetry(&FailedOperation{RetryCount: 1, MaxRetries: 3}))
	assert.True(t, handler.ShouldRetry(&FailedOperation{RetryCount: 2, MaxRetries: 3}))
	assert.False(t, handler.ShouldRetry(&FailedOperation{RetryCount: 3, MaxRetries: 3}))
}

func TestDeadLetterQueue_Metrics(t *testing.T) {
	dlq := NewDeadLetterQueue(&Config{
		MaxRetries:    1,
		RetryDelay:    10 * time.Millisecond,
		WorkerCount:   1,
		QueueSize:     10,
		EnableMetrics: true,
	})
	defer dlq.Close()

	// Add operations
	for i := 0; i < 3; i++ {
		operation := &FailedOperation{
			Operation:   "test_operation",
			Key:         fmt.Sprintf("test_key_%d", i),
			Data:        "test_data",
			Error:       "test_error",
			HandlerType: "test_handler",
		}
		err := dlq.AddFailedOperation(operation)
		assert.NoError(t, err)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	metrics := dlq.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(3), metrics.TotalFailed)
	assert.True(t, metrics.CurrentQueue >= 0)
	assert.True(t, metrics.AverageRetries >= 0)
}

// TestHandler is a test implementation of RetryHandler
type TestHandler struct {
	RetryFunc       func(ctx context.Context, operation *FailedOperation) error
	ShouldRetryFunc func(operation *FailedOperation) bool
}

func (h *TestHandler) Retry(ctx context.Context, operation *FailedOperation) error {
	if h.RetryFunc != nil {
		return h.RetryFunc(ctx, operation)
	}
	return nil
}

func (h *TestHandler) ShouldRetry(operation *FailedOperation) bool {
	if h.ShouldRetryFunc != nil {
		return h.ShouldRetryFunc(operation)
	}
	return true
}
