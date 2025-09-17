package unit

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

func TestNewObservability_WithNilConfig(t *testing.T) {
	obs := cachex.NewObservability(nil)
	if obs == nil {
		t.Errorf("NewObservability() should not return nil")
	}

	config := obs.GetConfig()
	if config == nil {
		t.Errorf("GetConfig() should not return nil")
		return
	}

	// Check default values
	if !config.EnableMetrics {
		t.Errorf("EnableMetrics should be true by default")
	}
	if !config.EnableTracing {
		t.Errorf("EnableTracing should be true by default")
	}
	if !config.EnableLogging {
		t.Errorf("EnableLogging should be true by default")
	}
	if config.ServiceName != "cachex" {
		t.Errorf("ServiceName should be 'cachex' by default, got %s", config.ServiceName)
	}
	if config.ServiceVersion != "1.0.0" {
		t.Errorf("ServiceVersion should be '1.0.0' by default, got %s", config.ServiceVersion)
	}
	if config.Environment != "production" {
		t.Errorf("Environment should be 'production' by default, got %s", config.Environment)
	}
}

func TestNewObservability_WithCustomConfig(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics:  false,
		EnableTracing:  false,
		EnableLogging:  false,
		ServiceName:    "test-service",
		ServiceVersion: "2.0.0",
		Environment:    "test",
	}

	obs := cachex.NewObservability(config)
	if obs == nil {
		t.Errorf("NewObservability() should not return nil")
	}

	retrievedConfig := obs.GetConfig()
	if retrievedConfig == nil {
		t.Errorf("GetConfig() should not return nil")
		return
	}

	// Check custom values
	if retrievedConfig.EnableMetrics != config.EnableMetrics {
		t.Errorf("EnableMetrics should match: got %v, want %v", retrievedConfig.EnableMetrics, config.EnableMetrics)
	}
	if retrievedConfig.EnableTracing != config.EnableTracing {
		t.Errorf("EnableTracing should match: got %v, want %v", retrievedConfig.EnableTracing, config.EnableTracing)
	}
	if retrievedConfig.EnableLogging != config.EnableLogging {
		t.Errorf("EnableLogging should match: got %v, want %v", retrievedConfig.EnableLogging, config.EnableLogging)
	}
	if retrievedConfig.ServiceName != config.ServiceName {
		t.Errorf("ServiceName should match: got %s, want %s", retrievedConfig.ServiceName, config.ServiceName)
	}
	if retrievedConfig.ServiceVersion != config.ServiceVersion {
		t.Errorf("ServiceVersion should match: got %s, want %s", retrievedConfig.ServiceVersion, config.ServiceVersion)
	}
	if retrievedConfig.Environment != config.Environment {
		t.Errorf("Environment should match: got %s, want %s", retrievedConfig.Environment, config.Environment)
	}
}

func TestObservabilityManager_TraceOperation_Enabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableTracing:  true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	obs := cachex.NewObservability(config)
	ctx := context.Background()

	// Test tracing with various parameters
	testCases := []struct {
		name      string
		operation string
		key       string
		namespace string
		ttl       time.Duration
		attempt   int
	}{
		{
			name:      "basic operation",
			operation: "get",
			key:       "test-key",
			namespace: "test-namespace",
			ttl:       5 * time.Minute,
			attempt:   1,
		},
		{
			name:      "operation without key",
			operation: "set",
			key:       "",
			namespace: "test-namespace",
			ttl:       0,
			attempt:   0,
		},
		{
			name:      "operation without namespace",
			operation: "del",
			key:       "test-key",
			namespace: "",
			ttl:       1 * time.Hour,
			attempt:   2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, span := obs.TraceOperation(ctx, tc.operation, tc.key, tc.namespace, tc.ttl, tc.attempt)
			if ctx == nil {
				t.Errorf("TraceOperation() should return non-nil context")
			}
			if span == nil {
				t.Errorf("TraceOperation() should return non-nil span when tracing is enabled")
			}
		})
	}
}

func TestObservabilityManager_TraceOperation_Disabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableTracing: false,
	}

	obs := cachex.NewObservability(config)
	ctx := context.Background()

	// When tracing is disabled, should return original context and no-op span
	ctx, span := obs.TraceOperation(ctx, "get", "test-key", "test-namespace", 5*time.Minute, 1)
	if ctx == nil {
		t.Errorf("TraceOperation() should return non-nil context")
	}
	if span == nil {
		t.Errorf("TraceOperation() should return non-nil span even when disabled")
	}
}

func TestObservabilityManager_RecordOperation_Disabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics: false,
	}

	obs := cachex.NewObservability(config)
	metrics := obs.GetMetrics()
	if metrics != nil {
		t.Errorf("GetMetrics() should return nil when metrics are disabled")
	}

	// Should not panic when metrics are disabled
	obs.RecordOperation("get", "hit", "memory", 10*time.Millisecond, 100, 0)
}

func TestObservabilityManager_RecordError_Disabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics: false,
	}

	obs := cachex.NewObservability(config)

	// Should not panic when metrics are disabled
	obs.RecordError("connection", "redis")
}

func TestObservabilityManager_LogOperation_Enabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableLogging:  true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	obs := cachex.NewObservability(config)

	testCases := []struct {
		name       string
		level      string
		operation  string
		key        string
		namespace  string
		ttl        time.Duration
		attempt    int
		duration   time.Duration
		err        error
		redactLogs bool
	}{
		{
			name:       "successful operation",
			level:      "info",
			operation:  "get",
			key:        "test-key",
			namespace:  "test-namespace",
			ttl:        5 * time.Minute,
			attempt:    1,
			duration:   10 * time.Millisecond,
			err:        nil,
			redactLogs: false,
		},
		{
			name:       "failed operation",
			level:      "error",
			operation:  "set",
			key:        "test-key",
			namespace:  "test-namespace",
			ttl:        0,
			attempt:    2,
			duration:   50 * time.Millisecond,
			err:        cachex.ErrTimeout,
			redactLogs: false,
		},
		{
			name:       "redacted logs",
			level:      "debug",
			operation:  "del",
			key:        "sensitive-key",
			namespace:  "test-namespace",
			ttl:        1 * time.Hour,
			attempt:    1,
			duration:   5 * time.Millisecond,
			err:        nil,
			redactLogs: true,
		},
		{
			name:       "operation without key",
			level:      "warn",
			operation:  "exists",
			key:        "",
			namespace:  "test-namespace",
			ttl:        0,
			attempt:    0,
			duration:   2 * time.Millisecond,
			err:        nil,
			redactLogs: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			obs.LogOperation(tc.level, tc.operation, tc.key, tc.namespace, tc.ttl, tc.attempt, tc.duration, tc.err, tc.redactLogs)
		})
	}
}

func TestObservabilityManager_LogOperation_Disabled(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableLogging: false,
	}

	obs := cachex.NewObservability(config)

	// Should not panic when logging is disabled
	obs.LogOperation("info", "get", "test-key", "test-namespace", 5*time.Minute, 1, 10*time.Millisecond, nil, false)
}

func TestObservabilityManager_GetMetrics(t *testing.T) {
	// Test with metrics disabled (to avoid Prometheus conflicts)
	configDisabled := &cachex.ObservabilityManagerConfig{
		EnableMetrics: false,
	}

	obsDisabled := cachex.NewObservability(configDisabled)
	metricsDisabled := obsDisabled.GetMetrics()
	if metricsDisabled != nil {
		t.Errorf("GetMetrics() should return nil when metrics are disabled")
	}
}

func TestObservabilityManager_GetConfig(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics:  false, // Disable metrics to avoid Prometheus conflicts
		EnableTracing:  false,
		EnableLogging:  true,
		ServiceName:    "test-service",
		ServiceVersion: "2.0.0",
		Environment:    "staging",
	}

	obs := cachex.NewObservability(config)
	retrievedConfig := obs.GetConfig()

	if retrievedConfig == nil {
		t.Errorf("GetConfig() should not return nil")
		return
	}

	// Verify all fields match
	if retrievedConfig.EnableMetrics != config.EnableMetrics {
		t.Errorf("EnableMetrics mismatch: got %v, want %v", retrievedConfig.EnableMetrics, config.EnableMetrics)
	}
	if retrievedConfig.EnableTracing != config.EnableTracing {
		t.Errorf("EnableTracing mismatch: got %v, want %v", retrievedConfig.EnableTracing, config.EnableTracing)
	}
	if retrievedConfig.EnableLogging != config.EnableLogging {
		t.Errorf("EnableLogging mismatch: got %v, want %v", retrievedConfig.EnableLogging, config.EnableLogging)
	}
	if retrievedConfig.ServiceName != config.ServiceName {
		t.Errorf("ServiceName mismatch: got %s, want %s", retrievedConfig.ServiceName, config.ServiceName)
	}
	if retrievedConfig.ServiceVersion != config.ServiceVersion {
		t.Errorf("ServiceVersion mismatch: got %s, want %s", retrievedConfig.ServiceVersion, config.ServiceVersion)
	}
	if retrievedConfig.Environment != config.Environment {
		t.Errorf("Environment mismatch: got %s, want %s", retrievedConfig.Environment, config.Environment)
	}
}

func TestObservabilityManager_EdgeCases(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics: false, // Disable metrics to avoid Prometheus conflicts
		EnableTracing: true,
		EnableLogging: true,
	}

	obs := cachex.NewObservability(config)

	// Test with empty strings and zero values
	obs.RecordOperation("", "", "", 0, 0, 0)
	obs.RecordError("", "")
	obs.LogOperation("", "", "", "", 0, 0, 0, nil, false)
}

func TestObservabilityManager_Concurrency(t *testing.T) {
	config := &cachex.ObservabilityManagerConfig{
		EnableMetrics: false, // Disable metrics to avoid Prometheus conflicts
		EnableTracing: true,
		EnableLogging: true,
	}

	obs := cachex.NewObservability(config)
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Test concurrent operations
			obs.RecordOperation("get", "hit", "memory", 10*time.Millisecond, 100, 0)
			obs.RecordError("timeout", "redis")
			obs.LogOperation("info", "get", "test-key", "test-namespace", 5*time.Minute, 1, 10*time.Millisecond, nil, false)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
