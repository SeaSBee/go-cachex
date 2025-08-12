package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPipelineConfig(t *testing.T) {
	config := DefaultPipelineConfig()

	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 100*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, 10, config.MaxConcurrent)
	assert.Equal(t, 50*time.Millisecond, config.FlushInterval)
	assert.Equal(t, 1000, config.BufferSize)
	assert.True(t, config.RetryFailedOps)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestHighPerformancePipelineConfig(t *testing.T) {
	config := HighPerformancePipelineConfig()

	assert.Equal(t, 500, config.BatchSize)
	assert.Equal(t, 50*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, 20, config.MaxConcurrent)
	assert.Equal(t, 25*time.Millisecond, config.FlushInterval)
	assert.Equal(t, 5000, config.BufferSize)
	assert.True(t, config.RetryFailedOps)
	assert.Equal(t, 2, config.MaxRetries)
}

func TestDefaultBatchProcessorConfig(t *testing.T) {
	config := DefaultBatchProcessorConfig()

	assert.Equal(t, 100, config.MaxBatchSize)
	assert.Equal(t, 100*time.Millisecond, config.ProcessInterval)
	assert.Equal(t, 5, config.MaxWorkers)
	assert.True(t, config.RetryOnError)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestSplitIntoBatches(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	// Test splitting into batches
	items := []string{"a", "b", "c", "d", "e", "f", "g"}
	batches := pm.splitIntoBatches(items, 3)

	assert.Len(t, batches, 3)
	assert.Equal(t, []string{"a", "b", "c"}, batches[0])
	assert.Equal(t, []string{"d", "e", "f"}, batches[1])
	assert.Equal(t, []string{"g"}, batches[2])
}

func TestSplitMapIntoBatches(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	// Test splitting map into batches
	items := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
		"c": []byte("3"),
		"d": []byte("4"),
		"e": []byte("5"),
	}

	batches := pm.splitMapIntoBatches(items, 2)

	assert.Len(t, batches, 3) // Should have 3 batches with batch size 2
	assert.Len(t, batches[0], 2)
	assert.Len(t, batches[1], 2)
	assert.Len(t, batches[2], 1)
}

func TestSplitIntoBatches_EmptyInput(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	batches := pm.splitIntoBatches([]string{}, 3)
	assert.Empty(t, batches)
}

func TestSplitIntoBatches_SingleBatch(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	items := []string{"a", "b", "c"}
	batches := pm.splitIntoBatches(items, 5)

	assert.Len(t, batches, 1)
	assert.Equal(t, items, batches[0])
}

func TestSplitMapIntoBatches_EmptyInput(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	batches := pm.splitMapIntoBatches(map[string][]byte{}, 3)
	assert.Empty(t, batches)
}

func TestSplitMapIntoBatches_SingleBatch(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	items := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}

	batches := pm.splitMapIntoBatches(items, 5)

	assert.Len(t, batches, 1)
	assert.Equal(t, items, batches[0])
}

func TestSplitIntoBatches_ZeroBatchSize(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	items := []string{"a", "b", "c"}
	batches := pm.splitIntoBatches(items, 0)

	assert.Len(t, batches, 3) // Should default to batch size 1
	assert.Equal(t, []string{"a"}, batches[0])
	assert.Equal(t, []string{"b"}, batches[1])
	assert.Equal(t, []string{"c"}, batches[2])
}

func TestSplitMapIntoBatches_ZeroBatchSize(t *testing.T) {
	pm := NewPipelineManager(nil, DefaultPipelineConfig())

	items := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}

	batches := pm.splitMapIntoBatches(items, 0)

	assert.Len(t, batches, 2) // Should default to batch size 1
	assert.Len(t, batches[0], 1)
	assert.Len(t, batches[1], 1)
}
