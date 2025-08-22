package cachex

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/seasbee/go-logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is the OpenTelemetry tracer for cache operations
var tracer = otel.Tracer("github.com/SeaSBee/go-cachex")

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Operation counters
	OperationsTotal    *prometheus.CounterVec
	OperationsDuration *prometheus.HistogramVec

	// Cache performance metrics
	CacheHits   prometheus.Counter
	CacheMisses prometheus.Counter
	CacheSets   prometheus.Counter
	CacheDels   prometheus.Counter

	// Data transfer metrics
	BytesIn  prometheus.Counter
	BytesOut prometheus.Counter

	// Error metrics
	ErrorsTotal *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerState     *prometheus.GaugeVec
	CircuitBreakerFailures  prometheus.Counter
	CircuitBreakerSuccesses prometheus.Counter

	// Worker pool metrics
	WorkerPoolActiveWorkers prometheus.Gauge
	WorkerPoolQueuedJobs    prometheus.Gauge
	WorkerPoolCompletedJobs prometheus.Counter
	WorkerPoolFailedJobs    prometheus.Counter

	// Dead letter queue metrics
	DLQFailedOperations    prometheus.Counter
	DLQRetriedOperations   prometheus.Counter
	DLQSucceededOperations prometheus.Counter
	DLQDroppedOperations   prometheus.Counter
	DLQCurrentQueueSize    prometheus.Gauge

	// Bloom filter metrics
	BloomFilterItems          prometheus.Gauge
	BloomFilterFalsePositives prometheus.Counter
	BloomFilterCapacity       prometheus.Gauge
}

// ObservabilityManager provides tracing, metrics, and logging capabilities
type ObservabilityManager struct {
	metrics *Metrics
	config  *ObservabilityManagerConfig
}

// ObservabilityManagerConfig holds observability configuration
type ObservabilityManagerConfig struct {
	EnableMetrics  bool   `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing  bool   `yaml:"enable_tracing" json:"enable_tracing"`
	EnableLogging  bool   `yaml:"enable_logging" json:"enable_logging"`
	ServiceName    string `yaml:"service_name" json:"service_name"`
	ServiceVersion string `yaml:"service_version" json:"service_version"`
	Environment    string `yaml:"environment" json:"environment"`
}

// NewObservability creates a new observability instance
func NewObservability(config *ObservabilityManagerConfig) *ObservabilityManager {
	if config == nil {
		config = &ObservabilityManagerConfig{
			EnableMetrics:  true,
			EnableTracing:  true,
			EnableLogging:  true,
			ServiceName:    "cachex",
			ServiceVersion: "1.0.0",
			Environment:    "production",
		}
	}

	obs := &ObservabilityManager{
		config: config,
	}

	if config.EnableMetrics {
		obs.metrics = obs.createMetrics()
	}

	return obs
}

// createMetrics creates all Prometheus metrics
func (o *ObservabilityManager) createMetrics() *Metrics {
	return &Metrics{
		// Operation counters
		OperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_operations_total",
				Help: "Total number of cache operations",
			},
			[]string{"operation", "status", "component"},
		),
		OperationsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cache_operation_duration_seconds",
				Help:    "Cache operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "component"},
		),

		// Cache performance metrics
		CacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		CacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		CacheSets: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_sets_total",
				Help: "Total number of cache sets",
			},
		),
		CacheDels: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_dels_total",
				Help: "Total number of cache deletions",
			},
		),

		// Data transfer metrics
		BytesIn: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_bytes_in_total",
				Help: "Total bytes read from cache",
			},
		),
		BytesOut: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_bytes_out_total",
				Help: "Total bytes written to cache",
			},
		),

		// Error metrics
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_errors_total",
				Help: "Total number of cache errors",
			},
			[]string{"type", "component"},
		),

		// Circuit breaker metrics
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_circuit_breaker_state",
				Help: "Current circuit breaker state (0=closed, 1=half_open, 2=open)",
			},
			[]string{"component"},
		),
		CircuitBreakerFailures: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_circuit_breaker_failures_total",
				Help: "Total circuit breaker failures",
			},
		),
		CircuitBreakerSuccesses: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_circuit_breaker_successes_total",
				Help: "Total circuit breaker successes",
			},
		),

		// Worker pool metrics
		WorkerPoolActiveWorkers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_worker_pool_active_workers",
				Help: "Number of active workers in pool",
			},
		),
		WorkerPoolQueuedJobs: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_worker_pool_queued_jobs",
				Help: "Number of queued jobs in pool",
			},
		),
		WorkerPoolCompletedJobs: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_worker_pool_completed_jobs_total",
				Help: "Total completed jobs in pool",
			},
		),
		WorkerPoolFailedJobs: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_worker_pool_failed_jobs_total",
				Help: "Total failed jobs in pool",
			},
		),

		// Dead letter queue metrics
		DLQFailedOperations: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_dlq_failed_operations_total",
				Help: "Total failed operations in DLQ",
			},
		),
		DLQRetriedOperations: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_dlq_retried_operations_total",
				Help: "Total retried operations in DLQ",
			},
		),
		DLQSucceededOperations: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_dlq_succeeded_operations_total",
				Help: "Total succeeded operations in DLQ",
			},
		),
		DLQDroppedOperations: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_dlq_dropped_operations_total",
				Help: "Total dropped operations in DLQ",
			},
		),
		DLQCurrentQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_dlq_current_queue_size",
				Help: "Current queue size in DLQ",
			},
		),

		// Bloom filter metrics
		BloomFilterItems: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_bloom_filter_items",
				Help: "Number of items in bloom filter",
			},
		),
		BloomFilterFalsePositives: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_bloom_filter_false_positives_total",
				Help: "Total false positives in bloom filter",
			},
		),
		BloomFilterCapacity: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cache_bloom_filter_capacity",
				Help: "Capacity of bloom filter",
			},
		),
	}
}

// TraceOperation creates a span for cache operations
func (o *ObservabilityManager) TraceOperation(ctx context.Context, operation, key, namespace string, ttl time.Duration, attempt int) (context.Context, trace.Span) {
	if !o.config.EnableTracing {
		return ctx, trace.SpanFromContext(ctx)
	}

	attrs := []attribute.KeyValue{
		attribute.String("cache.operation", operation),
		attribute.String("cache.component", "cachex"),
		attribute.String("cache.service.name", o.config.ServiceName),
		attribute.String("cache.service.version", o.config.ServiceVersion),
		attribute.String("cache.environment", o.config.Environment),
	}

	if key != "" {
		attrs = append(attrs, attribute.String("cache.key", key))
	}
	if namespace != "" {
		attrs = append(attrs, attribute.String("cache.namespace", namespace))
	}
	if ttl > 0 {
		attrs = append(attrs, attribute.Int64("cache.ttl_ms", int64(ttl.Milliseconds())))
	}
	if attempt > 0 {
		attrs = append(attrs, attribute.Int("cache.attempt", attempt))
	}

	ctx, span := tracer.Start(ctx, fmt.Sprintf("cachex.%s", operation),
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

// RecordOperation records metrics for cache operations
func (o *ObservabilityManager) RecordOperation(operation, status, component string, duration time.Duration, bytesIn, bytesOut int64) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	// Record operation counter
	o.metrics.OperationsTotal.WithLabelValues(operation, status, component).Inc()

	// Record operation duration
	o.metrics.OperationsDuration.WithLabelValues(operation, component).Observe(duration.Seconds())

	// Record data transfer
	if bytesIn > 0 {
		o.metrics.BytesIn.Add(float64(bytesIn))
	}
	if bytesOut > 0 {
		o.metrics.BytesOut.Add(float64(bytesOut))
	}

	// Record specific operation types
	switch operation {
	case "get":
		if status == "hit" {
			o.metrics.CacheHits.Inc()
		} else if status == "miss" {
			o.metrics.CacheMisses.Inc()
		}
	case "set":
		o.metrics.CacheSets.Inc()
	case "del":
		o.metrics.CacheDels.Inc()
	}
}

// RecordError records error metrics
func (o *ObservabilityManager) RecordError(errorType, component string) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordCircuitBreakerState records circuit breaker state
func (o *ObservabilityManager) RecordCircuitBreakerState(component string, state int) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.CircuitBreakerState.WithLabelValues(component).Set(float64(state))
}

// RecordCircuitBreakerFailure records circuit breaker failure
func (o *ObservabilityManager) RecordCircuitBreakerFailure() {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.CircuitBreakerFailures.Inc()
}

// RecordCircuitBreakerSuccess records circuit breaker success
func (o *ObservabilityManager) RecordCircuitBreakerSuccess() {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.CircuitBreakerSuccesses.Inc()
}

// RecordWorkerPoolStats records worker pool statistics
func (o *ObservabilityManager) RecordWorkerPoolStats(activeWorkers, queuedJobs int32, completedJobs, failedJobs int64) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.WorkerPoolActiveWorkers.Set(float64(activeWorkers))
	o.metrics.WorkerPoolQueuedJobs.Set(float64(queuedJobs))
	o.metrics.WorkerPoolCompletedJobs.Add(float64(completedJobs))
	o.metrics.WorkerPoolFailedJobs.Add(float64(failedJobs))
}

// RecordDLQStats records dead letter queue statistics
func (o *ObservabilityManager) RecordDLQStats(failed, retried, succeeded, dropped int64, currentQueueSize int64) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.DLQFailedOperations.Add(float64(failed))
	o.metrics.DLQRetriedOperations.Add(float64(retried))
	o.metrics.DLQSucceededOperations.Add(float64(succeeded))
	o.metrics.DLQDroppedOperations.Add(float64(dropped))
	o.metrics.DLQCurrentQueueSize.Set(float64(currentQueueSize))
}

// RecordBloomFilterStats records bloom filter statistics
func (o *ObservabilityManager) RecordBloomFilterStats(items, capacity uint64, falsePositives int64) {
	if !o.config.EnableMetrics || o.metrics == nil {
		return
	}

	o.metrics.BloomFilterItems.Set(float64(items))
	o.metrics.BloomFilterCapacity.Set(float64(capacity))
	o.metrics.BloomFilterFalsePositives.Add(float64(falsePositives))
}

// LogOperation logs cache operations with enhanced fields
func (o *ObservabilityManager) LogOperation(level, operation, key, namespace string, ttl time.Duration, attempt int, duration time.Duration, err error, redactLogs bool) {
	if !o.config.EnableLogging {
		return
	}

	fields := []logx.Field{
		logx.String("component", "cachex"),
		logx.String("operation", operation),
		logx.String("service_name", o.config.ServiceName),
		logx.String("service_version", o.config.ServiceVersion),
		logx.String("environment", o.config.Environment),
		logx.Int("duration_ms", int(duration.Milliseconds())),
	}

	if namespace != "" {
		fields = append(fields, logx.String("namespace", namespace))
	}

	if ttl > 0 {
		fields = append(fields, logx.String("ttl", ttl.String()))
	}

	if attempt > 0 {
		fields = append(fields, logx.Int("attempt", attempt))
	}

	if redactLogs {
		fields = append(fields, logx.String("key", "[REDACTED]"))
	} else if key != "" {
		fields = append(fields, logx.String("key", key))
	}

	if err != nil {
		fields = append(fields, logx.ErrorField(err))
	}

	switch level {
	case "debug":
		logx.Debug("Cache operation completed", fields...)
	case "info":
		logx.Info("Cache operation completed", fields...)
	case "warn":
		logx.Warn("Cache operation warning", fields...)
	case "error":
		logx.Error("Cache operation failed", fields...)
	default:
		logx.Info("Cache operation completed", fields...)
	}
}

// GetMetrics returns the metrics instance
func (o *ObservabilityManager) GetMetrics() *Metrics {
	return o.metrics
}

// GetConfig returns the observability configuration
func (o *ObservabilityManager) GetConfig() *ObservabilityManagerConfig {
	return o.config
}
