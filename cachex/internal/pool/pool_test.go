package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkerPool(t *testing.T) {
	pool, err := New(DefaultConfig())
	require.NoError(t, err)
	defer pool.Close()

	stats := pool.GetStats()
	assert.Equal(t, int32(2), stats.TotalWorkers)
	assert.Equal(t, int32(2), stats.IdleWorkers)
}

func TestWorkerPool_Submit(t *testing.T) {
	pool, err := New(DefaultConfig())
	require.NoError(t, err)
	defer pool.Close()

	job := Job{
		ID: "test-job",
		Task: func() (interface{}, error) {
			return "success", nil
		},
	}

	err = pool.Submit(job)
	assert.NoError(t, err)

	result, err := pool.GetResultWithTimeout(5 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, "test-job", result.JobID)
	assert.Equal(t, "success", result.Result)
}

func TestWorkerPool_Backpressure(t *testing.T) {
	config := DefaultConfig()
	config.QueueSize = 1
	config.MinWorkers = 1
	config.MaxWorkers = 1

	pool, err := New(config)
	require.NoError(t, err)
	defer pool.Close()

	// Submit a job
	job := Job{
		ID: "test-job",
		Task: func() (interface{}, error) {
			return "test-result", nil
		},
	}

	err = pool.Submit(job)
	assert.NoError(t, err)

	// Try to submit another job (should be rejected due to small queue)
	job2 := Job{
		ID: "test-job-2",
		Task: func() (interface{}, error) {
			return "test-result-2", nil
		},
	}

	err = pool.Submit(job2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job queue is full")
}

func TestWorkerPool_ScaleUp(t *testing.T) {
	config := DefaultConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 3

	pool, err := New(config)
	require.NoError(t, err)
	defer pool.Close()

	initialStats := pool.GetStats()
	assert.Equal(t, int32(1), initialStats.TotalWorkers)

	err = pool.ScaleUp(1)
	assert.NoError(t, err)

	scaledStats := pool.GetStats()
	assert.Equal(t, int32(2), scaledStats.TotalWorkers)
}

func TestWorkerPool_Close(t *testing.T) {
	pool, err := New(DefaultConfig())
	require.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)

	// Try to submit after closing
	job := Job{
		ID: "closed-job",
		Task: func() (interface{}, error) {
			return "result", nil
		},
	}

	err = pool.Submit(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker pool is closed")
}
