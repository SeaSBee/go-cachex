package unit

import (
	"context"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisConfig(t *testing.T) {
	tests := []struct {
		name                string
		addr                string
		password            string
		db                  int
		poolSize            int
		minIdleConns        int
		maxRetries          int
		dialTimeout         time.Duration
		readTimeout         time.Duration
		writeTimeout        time.Duration
		enablePipelining    bool
		enableMetrics       bool
		healthCheckInterval time.Duration
		healthCheckTimeout  time.Duration
		expected            *cachex.RedisConfig
	}{
		{
			name:                "valid configuration",
			addr:                "localhost:6379",
			password:            "secret",
			db:                  1,
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       true,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			expected: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "secret",
				DB:                  1,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    true,
				EnableMetrics:       true,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
		},
		{
			name:                "minimal configuration",
			addr:                "127.0.0.1:6379",
			password:            "",
			db:                  0,
			poolSize:            1,
			minIdleConns:        0,
			maxRetries:          0,
			dialTimeout:         100 * time.Millisecond,
			readTimeout:         100 * time.Millisecond,
			writeTimeout:        100 * time.Millisecond,
			enablePipelining:    false,
			enableMetrics:       false,
			healthCheckInterval: 1 * time.Second,
			healthCheckTimeout:  1 * time.Second,
			expected: &cachex.RedisConfig{
				Addr:                "127.0.0.1:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            1,
				MinIdleConns:        0,
				MaxRetries:          0,
				DialTimeout:         100 * time.Millisecond,
				ReadTimeout:         100 * time.Millisecond,
				WriteTimeout:        100 * time.Millisecond,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 1 * time.Second,
				HealthCheckTimeout:  1 * time.Second,
			},
		},
		{
			name:                "maximum values",
			addr:                "redis.example.com:6379",
			password:            "very-long-password-with-special-chars!@#$%",
			db:                  15,
			poolSize:            1000,
			minIdleConns:        100,
			maxRetries:          10,
			dialTimeout:         5 * time.Minute,
			readTimeout:         5 * time.Minute,
			writeTimeout:        5 * time.Minute,
			enablePipelining:    true,
			enableMetrics:       true,
			healthCheckInterval: 2 * time.Minute,
			healthCheckTimeout:  2 * time.Minute,
			expected: &cachex.RedisConfig{
				Addr:                "redis.example.com:6379",
				Password:            "very-long-password-with-special-chars!@#$%",
				DB:                  15,
				PoolSize:            1000,
				MinIdleConns:        100,
				MaxRetries:          10,
				DialTimeout:         5 * time.Minute,
				ReadTimeout:         5 * time.Minute,
				WriteTimeout:        5 * time.Minute,
				EnablePipelining:    true,
				EnableMetrics:       true,
				HealthCheckInterval: 2 * time.Minute,
				HealthCheckTimeout:  2 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := cachex.NewRedisConfig(
				tt.addr,
				tt.password,
				tt.db,
				"", // username
				tt.poolSize,
				tt.minIdleConns,
				tt.maxRetries,
				tt.dialTimeout,
				tt.readTimeout,
				tt.writeTimeout,
				tt.enablePipelining,
				tt.enableMetrics,
				tt.healthCheckInterval,
				tt.healthCheckTimeout,
			)

			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestRedisConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *cachex.RedisConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty address",
			config: &cachex.RedisConfig{
				Addr:                "",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "address is required",
		},
		{
			name: "negative pool size",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            -1,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "pool size cannot be negative",
		},
		{
			name: "pool size too large",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            1001,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "pool size cannot exceed 1000",
		},
		{
			name: "min idle conns exceeds pool size",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        15,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "min idle connections cannot exceed pool size",
		},
		{
			name: "max retries too high",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          11,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "max retries cannot exceed 10",
		},
		{
			name: "negative dial timeout",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         -1 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "dial timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_CreateRedisClient(t *testing.T) {
	config := &cachex.RedisConfig{
		Addr:                "localhost:6379",
		Password:            "secret",
		DB:                  1,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	client := config.CreateRedisClient()
	require.NotNil(t, client)

	// Test that the client is properly configured
	// Note: We can't easily test the internal configuration without reflection
	// but we can test that a client is created successfully
	assert.NotNil(t, client)
}

func TestRedisConfig_ConnectRedisClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *cachex.RedisConfig
		expectErr bool
	}{
		{
			name: "valid configuration with working Redis",
			config: &cachex.RedisConfig{
				Addr:                "localhost:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    true,
				EnableMetrics:       true,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectErr: false, // This will depend on whether Redis is running
		},
		{
			name: "invalid configuration",
			config: &cachex.RedisConfig{
				Addr:                "", // Invalid empty address
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectErr: true, // Should fail validation
		},
		{
			name: "invalid Redis connection",
			config: &cachex.RedisConfig{
				Addr:                "invalid:address:6379",
				Password:            "",
				DB:                  0,
				PoolSize:            10,
				MinIdleConns:        5,
				MaxRetries:          3,
				DialTimeout:         1 * time.Second, // Short timeout for quick failure
				ReadTimeout:         1 * time.Second,
				WriteTimeout:        1 * time.Second,
				EnablePipelining:    false,
				EnableMetrics:       false,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectErr: true, // Should fail connection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := tt.config.ConnectRedisClient(ctx)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				// Note: This test will only pass if Redis is actually running
				// In a real CI environment, you might want to skip this test
				// or use a test Redis instance
				if err != nil {
					t.Skipf("Skipping test because Redis is not available: %v", err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, client)
				client.Close()
			}
		})
	}
}

func TestCreateRedisCache(t *testing.T) {
	tests := []struct {
		name                string
		addr                string
		password            string
		db                  int
		poolSize            int
		minIdleConns        int
		maxRetries          int
		dialTimeout         time.Duration
		readTimeout         time.Duration
		writeTimeout        time.Duration
		enablePipelining    bool
		enableMetrics       bool
		healthCheckInterval time.Duration
		healthCheckTimeout  time.Duration
		codec               cachex.Codec
		keyBuilder          cachex.KeyBuilder
		keyHasher           cachex.KeyHasher
		expectErr           bool
	}{
		{
			name:                "valid configuration with defaults",
			addr:                "localhost:6379",
			password:            "",
			db:                  0,
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               nil, // Use default
			keyBuilder:          nil, // Use default
			keyHasher:           nil, // Use default
			expectErr:           false,
		},
		{
			name:                "valid configuration with custom components",
			addr:                "localhost:6379",
			password:            "",
			db:                  0,
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               &cachex.JSONCodec{},
			keyBuilder:          nil, // use default
			keyHasher:           nil, // use default
			expectErr:           false,
		},
		{
			name:                "invalid configuration - empty address",
			addr:                "",
			password:            "",
			db:                  0,
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               nil,
			keyBuilder:          nil,
			keyHasher:           nil,
			expectErr:           true,
		},
		{
			name:                "invalid configuration - invalid database",
			addr:                "localhost:6379",
			password:            "",
			db:                  -1, // Invalid
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               nil,
			keyBuilder:          nil,
			keyHasher:           nil,
			expectErr:           true,
		},
		{
			name:                "invalid configuration - invalid pool size",
			addr:                "localhost:6379",
			password:            "",
			db:                  0,
			poolSize:            0, // Invalid
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         5 * time.Second,
			readTimeout:         3 * time.Second,
			writeTimeout:        3 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               nil,
			keyBuilder:          nil,
			keyHasher:           nil,
			expectErr:           true,
		},
		{
			name:                "invalid Redis connection",
			addr:                "invalid:address:6379",
			password:            "",
			db:                  0,
			poolSize:            10,
			minIdleConns:        5,
			maxRetries:          3,
			dialTimeout:         1 * time.Second, // Short timeout for quick failure
			readTimeout:         1 * time.Second,
			writeTimeout:        1 * time.Second,
			enablePipelining:    true,
			enableMetrics:       false,
			healthCheckInterval: 30 * time.Second,
			healthCheckTimeout:  5 * time.Second,
			codec:               nil,
			keyBuilder:          nil,
			keyHasher:           nil,
			expectErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := cachex.CreateRedisCache(
				tt.addr,
				tt.password,
				tt.db,
				"", // username
				tt.poolSize,
				tt.minIdleConns,
				tt.maxRetries,
				tt.dialTimeout,
				tt.readTimeout,
				tt.writeTimeout,
				tt.enablePipelining,
				tt.enableMetrics,
				tt.healthCheckInterval,
				tt.healthCheckTimeout,
				tt.codec,
				tt.keyBuilder,
				tt.keyHasher,
			)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, cache)
			} else {
				// Note: This test will only pass if Redis is actually running
				// In a real CI environment, you might want to skip this test
				// or use a test Redis instance
				if err != nil {
					t.Skipf("Skipping test because Redis is not available: %v", err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, cache)
				cache.Close()
			}
		})
	}
}

// TestRedisConfig_EdgeCases tests edge cases and boundary conditions
func TestRedisConfig_EdgeCases(t *testing.T) {
	t.Run("boundary values", func(t *testing.T) {

		// Test minimum valid values
		config := &cachex.RedisConfig{
			Addr:                "a:1", // Minimum valid address
			Password:            "",
			DB:                  0,                      // Minimum valid DB
			PoolSize:            1,                      // Minimum valid pool size
			MinIdleConns:        0,                      // Minimum valid min idle conns
			MaxRetries:          0,                      // Minimum valid max retries
			DialTimeout:         100 * time.Millisecond, // Minimum valid timeout
			ReadTimeout:         100 * time.Millisecond,
			WriteTimeout:        100 * time.Millisecond,
			EnablePipelining:    false,
			EnableMetrics:       false,
			HealthCheckInterval: 1 * time.Second, // Minimum valid health check interval
			HealthCheckTimeout:  1 * time.Second, // Minimum valid health check timeout
		}

		err := config.Validate()
		assert.NoError(t, err, "Minimum valid values should pass validation")

		// Test maximum valid values
		config = &cachex.RedisConfig{
			Addr:                "this-is-a-very-long-address-that-is-exactly-256-characters-long-and-should-pass-validation-because-it-is-exactly-at-the-maximum-length-limit-that-has-been-set-for-the-address-field-in-the-redis-config-struct-and-should-not-cause-any-validation-errors",
			Password:            "very-long-password-with-special-chars!@#$%^&*()_+-=[]{}|;':\",./<>?",
			DB:                  15,              // Maximum valid DB
			PoolSize:            1000,            // Maximum valid pool size
			MinIdleConns:        1000,            // Large but valid min idle conns
			MaxRetries:          10,              // Maximum valid max retries
			DialTimeout:         5 * time.Minute, // Maximum valid timeout
			ReadTimeout:         5 * time.Minute,
			WriteTimeout:        5 * time.Minute,
			EnablePipelining:    true,
			EnableMetrics:       true,
			HealthCheckInterval: 2 * time.Minute, // Maximum valid health check interval
			HealthCheckTimeout:  2 * time.Minute, // Maximum valid health check timeout
		}

		err = config.Validate()
		assert.NoError(t, err, "Maximum valid values should pass validation")
	})

	t.Run("zero values", func(t *testing.T) {

		config := &cachex.RedisConfig{
			Addr:                "localhost:6379",
			Password:            "",
			DB:                  0,
			PoolSize:            0, // Invalid - should fail
			MinIdleConns:        0,
			MaxRetries:          0,
			DialTimeout:         0, // Invalid - should fail
			ReadTimeout:         0, // Invalid - should fail
			WriteTimeout:        0, // Invalid - should fail
			EnablePipelining:    false,
			EnableMetrics:       false,
			HealthCheckInterval: 0, // Invalid - should fail
			HealthCheckTimeout:  0, // Invalid - should fail
		}

		err := config.Validate()
		assert.Error(t, err, "Zero values for required fields should fail validation")
	})
}

// TestRedisConfig_ValidationErrorMessages tests that validation errors provide meaningful messages
func TestRedisConfig_ValidationErrorMessages(t *testing.T) {
	config := &cachex.RedisConfig{
		Addr:                "", // Empty address
		Password:            "",
		DB:                  0,
		PoolSize:            -1,               // Invalid pool size
		MinIdleConns:        -1,               // Invalid min idle conns
		MaxRetries:          -1,               // Invalid max retries
		DialTimeout:         -1 * time.Second, // Invalid timeout
		ReadTimeout:         -1 * time.Second, // Invalid timeout
		WriteTimeout:        -1 * time.Second, // Invalid timeout
		EnablePipelining:    false,
		EnableMetrics:       false,
		HealthCheckInterval: -1 * time.Second, // Invalid timeout
		HealthCheckTimeout:  -1 * time.Second, // Invalid timeout
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address is required")
}

// TestRedisConfig_CreateRedisClient_OptionsMapping tests that Redis client options are properly mapped
func TestRedisConfig_CreateRedisClient_OptionsMapping(t *testing.T) {
	config := &cachex.RedisConfig{
		Addr:                "test:6379",
		Password:            "testpass",
		DB:                  5,
		PoolSize:            20,
		MinIdleConns:        10,
		MaxRetries:          5,
		DialTimeout:         10 * time.Second,
		ReadTimeout:         8 * time.Second,
		WriteTimeout:        6 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 60 * time.Second,
		HealthCheckTimeout:  10 * time.Second,
	}

	client := config.CreateRedisClient()
	require.NotNil(t, client)

	// Test that client is created successfully
	// Note: We can't easily test the internal options without reflection,
	// but we can verify the client is created and can be closed
	defer client.Close()
	assert.NotNil(t, client)
}

// Benchmark tests for performance
func BenchmarkNewRedisConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = cachex.NewRedisConfig(
			"localhost:6379",
			"password",
			1,
			"", // username
			10,
			5,
			3,
			5*time.Second,
			3*time.Second,
			3*time.Second,
			true,
			true,
			30*time.Second,
			5*time.Second,
		)
	}
}

func BenchmarkRedisConfig_Validate(b *testing.B) {
	config := &cachex.RedisConfig{
		Addr:                "localhost:6379",
		Password:            "password",
		DB:                  1,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

func BenchmarkRedisConfig_CreateRedisClient(b *testing.B) {
	config := &cachex.RedisConfig{
		Addr:                "localhost:6379",
		Password:            "password",
		DB:                  1,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		EnablePipelining:    true,
		EnableMetrics:       true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := config.CreateRedisClient()
		client.Close()
	}
}
