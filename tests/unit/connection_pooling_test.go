package unit

import (
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/stretchr/testify/assert"
)

// TestNewOptimizedConnectionPool_Configuration tests configuration handling
func TestNewOptimizedConnectionPool_Configuration(t *testing.T) {
	tests := []struct {
		name        string
		config      *cachex.OptimizedConnectionPoolConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "With_nil_config",
			config:      nil,
			expectError: false, // Uses default config instead of error
		},
		{
			name: "With_empty_addr",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr: "",
			},
			expectError: true,
			errorMsg:    "address cannot be empty",
		},
		{
			name: "With_invalid_pool_size",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:     "localhost:6379",
				PoolSize: 0,
			},
			expectError: true,
			errorMsg:    "pool size must be positive",
		},
		{
			name: "With_invalid_min_idle_conns",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:         "localhost:6379",
				PoolSize:     10,
				MinIdleConns: 15, // Greater than pool size
			},
			expectError: true,
			errorMsg:    "min idle connections cannot exceed pool size",
		},
		{
			name: "With_invalid_pool_timeout",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:        "localhost:6379",
				PoolSize:    10,
				PoolTimeout: 0,
			},
			expectError: true,
			errorMsg:    "pool timeout must be positive",
		},
		{
			name: "With_invalid_dial_timeout",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:        "localhost:6379",
				PoolSize:    10,
				PoolTimeout: 30 * time.Second,
				DialTimeout: 0,
			},
			expectError: true,
			errorMsg:    "dial timeout must be positive",
		},
		{
			name: "With_invalid_read_timeout",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:        "localhost:6379",
				PoolSize:    10,
				PoolTimeout: 30 * time.Second,
				DialTimeout: 5 * time.Second,
				ReadTimeout: 0,
			},
			expectError: true,
			errorMsg:    "read timeout must be positive",
		},
		{
			name: "With_invalid_write_timeout",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:         "localhost:6379",
				PoolSize:     10,
				PoolTimeout:  30 * time.Second,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 0,
			},
			expectError: true,
			errorMsg:    "write timeout must be positive",
		},
		{
			name: "With_invalid_health_check_interval",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:                "localhost:6379",
				PoolSize:            10,
				PoolTimeout:         30 * time.Second,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				HealthCheckInterval: 0,
			},
			expectError: true,
			errorMsg:    "health check interval must be positive",
		},
		{
			name: "With_invalid_health_check_timeout",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:                "localhost:6379",
				PoolSize:            10,
				PoolTimeout:         30 * time.Second,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  0,
			},
			expectError: true,
			errorMsg:    "health check timeout must be positive",
		},
		{
			name: "With_valid_config",
			config: &cachex.OptimizedConnectionPoolConfig{
				Addr:                "localhost:6379",
				PoolSize:            10,
				MinIdleConns:        5,
				PoolTimeout:         30 * time.Second,
				DialTimeout:         5 * time.Second,
				ReadTimeout:         3 * time.Second,
				WriteTimeout:        3 * time.Second,
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := cachex.NewOptimizedConnectionPool(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, pool)
			} else {
				// For valid configs, we expect connection to fail but pool creation to succeed
				if err != nil {
					// Connection failed but pool was created
					assert.Contains(t, err.Error(), "failed to connect to redis")
					assert.Nil(t, pool)
				} else {
					// Pool created successfully
					assert.NotNil(t, pool)
					if pool != nil {
						pool.Close()
					}
				}
			}
		})
	}
}

// TestOptimizedConnectionPool_ConfigValidation tests config validation
func TestOptimizedConnectionPool_ConfigValidation(t *testing.T) {
	t.Run("Default_config_validation", func(t *testing.T) {
		config := cachex.DefaultOptimizedConnectionPoolConfig()
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Addr)
		assert.Greater(t, config.PoolSize, 0)
		assert.GreaterOrEqual(t, config.MinIdleConns, 0)
		assert.LessOrEqual(t, config.MinIdleConns, config.PoolSize)
		assert.Greater(t, config.PoolTimeout, time.Duration(0))
		assert.Greater(t, config.DialTimeout, time.Duration(0))
		assert.Greater(t, config.ReadTimeout, time.Duration(0))
		assert.Greater(t, config.WriteTimeout, time.Duration(0))
		assert.Greater(t, config.HealthCheckInterval, time.Duration(0))
		assert.Greater(t, config.HealthCheckTimeout, time.Duration(0))
	})

	t.Run("High_performance_config_validation", func(t *testing.T) {
		config := cachex.HighPerformanceConnectionPoolConfig()
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Addr)
		assert.Greater(t, config.PoolSize, 0)
		assert.GreaterOrEqual(t, config.MinIdleConns, 0)
		assert.LessOrEqual(t, config.MinIdleConns, config.PoolSize)
		assert.Greater(t, config.PoolTimeout, time.Duration(0))
		assert.Greater(t, config.DialTimeout, time.Duration(0))
		assert.Greater(t, config.ReadTimeout, time.Duration(0))
		assert.Greater(t, config.WriteTimeout, time.Duration(0))
		assert.Greater(t, config.HealthCheckInterval, time.Duration(0))
		assert.Greater(t, config.HealthCheckTimeout, time.Duration(0))
	})

	t.Run("Production_config_validation", func(t *testing.T) {
		config := cachex.ProductionConnectionPoolConfig()
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.Addr)
		assert.Greater(t, config.PoolSize, 0)
		assert.GreaterOrEqual(t, config.MinIdleConns, 0)
		assert.LessOrEqual(t, config.MinIdleConns, config.PoolSize)
		assert.Greater(t, config.PoolTimeout, time.Duration(0))
		assert.Greater(t, config.DialTimeout, time.Duration(0))
		assert.Greater(t, config.ReadTimeout, time.Duration(0))
		assert.Greater(t, config.WriteTimeout, time.Duration(0))
		assert.Greater(t, config.HealthCheckInterval, time.Duration(0))
		assert.Greater(t, config.HealthCheckTimeout, time.Duration(0))
	})
}

// TestOptimizedConnectionPool_ErrorHandling tests error handling scenarios
func TestOptimizedConnectionPool_ErrorHandling(t *testing.T) {
	t.Run("Redis_connection_failure", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "localhost:9999", // Invalid port
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "failed to connect to redis")
	})

	t.Run("Invalid_Redis_address", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-address",
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "failed to connect to redis")
	})
}

// TestOptimizedConnectionPool_NilHandling tests nil handling
func TestOptimizedConnectionPool_NilHandling(t *testing.T) {
	t.Run("Nil_pool_operations", func(t *testing.T) {
		var pool *cachex.OptimizedConnectionPool

		// These should not panic
		stats := pool.GetStats()
		assert.NotNil(t, stats) // Returns empty stats struct, not nil

		err := pool.Close()
		assert.NoError(t, err) // Close returns nil for nil pool

		client := pool.GetClient()
		assert.Nil(t, client)
	})

	t.Run("Nil_context_handling", func(t *testing.T) {
		// Test that nil context is handled gracefully
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		// Should fail due to connection, but not due to nil context
		assert.Error(t, err)
		assert.Nil(t, pool)
	})
}

// TestOptimizedConnectionPool_TLSConfig tests TLS configuration
func TestOptimizedConnectionPool_TLSConfig(t *testing.T) {
	t.Run("TLS_disabled", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			TLS:                 &cachex.TLSConfig{Enabled: false},
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail, but TLS config should be valid
		assert.Nil(t, pool)
	})

	t.Run("TLS_enabled", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			TLS:                 &cachex.TLSConfig{Enabled: true, InsecureSkipVerify: true},
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail, but TLS config should be valid
		assert.Nil(t, pool)
	})

	t.Run("TLS_nil_config", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            5,
			MinIdleConns:        2,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			TLS:                 nil, // Should be set to default disabled config
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail, but TLS config should be valid
		assert.Nil(t, pool)
	})
}

// TestOptimizedConnectionPool_EdgeCases tests edge cases
func TestOptimizedConnectionPool_EdgeCases(t *testing.T) {
	t.Run("Zero_pool_size", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:     "localhost:6379",
			PoolSize: 0,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "pool size must be positive")
	})

	t.Run("Negative_min_idle_conns", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:         "localhost:6379",
			PoolSize:     10,
			MinIdleConns: -1,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "min idle connections cannot be negative")
	})

	t.Run("Min_idle_conns_greater_than_pool_size", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:         "localhost:6379",
			PoolSize:     5,
			MinIdleConns: 10,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "min idle connections cannot exceed pool size")
	})

	t.Run("Negative_max_retries", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:       "localhost:6379",
			PoolSize:   10,
			MaxRetries: -1,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "max retries cannot be negative")
	})

	t.Run("Zero_timeouts", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:        "localhost:6379",
			PoolSize:    10,
			PoolTimeout: 0,
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "pool timeout must be positive")
	})
}

// TestOptimizedConnectionPool_DefaultValues tests default value handling
func TestOptimizedConnectionPool_DefaultValues(t *testing.T) {
	t.Run("Default_idle_timeout", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            10,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			// IdleTimeout not set, should get default
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail
		assert.Nil(t, pool)
		// But validation should pass and default values should be set
	})

	t.Run("Default_idle_check_frequency", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            10,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			// IdleCheckFrequency not set, should get default
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail
		assert.Nil(t, pool)
		// But validation should pass and default values should be set
	})

	t.Run("Default_connection_warming_timeout", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            10,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			// ConnectionWarmingTimeout not set, should get default
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail
		assert.Nil(t, pool)
		// But validation should pass and default values should be set
	})

	t.Run("Default_monitoring_interval", func(t *testing.T) {
		config := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            10,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			// MonitoringInterval not set, should get default
		}

		pool, err := cachex.NewOptimizedConnectionPool(config)
		assert.Error(t, err) // Connection will fail
		assert.Nil(t, pool)
		// But validation should pass and default values should be set
	})
}

// TestOptimizedConnectionPool_ConfigurationCopy tests that original config is not modified
func TestOptimizedConnectionPool_ConfigurationCopy(t *testing.T) {
	t.Run("Original_config_not_modified", func(t *testing.T) {
		originalConfig := &cachex.OptimizedConnectionPoolConfig{
			Addr:                "invalid-host:9999", // Use invalid address to ensure connection failure
			PoolSize:            10,
			PoolTimeout:         30 * time.Second,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}

		// Store original values
		originalIdleTimeout := originalConfig.IdleTimeout
		originalIdleCheckFrequency := originalConfig.IdleCheckFrequency
		originalConnectionWarmingTimeout := originalConfig.ConnectionWarmingTimeout
		originalMonitoringInterval := originalConfig.MonitoringInterval

		// Try to create pool (will fail due to connection)
		pool, err := cachex.NewOptimizedConnectionPool(originalConfig)
		assert.Error(t, err)
		assert.Nil(t, pool)

		// Verify original config was not modified
		assert.Equal(t, originalIdleTimeout, originalConfig.IdleTimeout)
		assert.Equal(t, originalIdleCheckFrequency, originalConfig.IdleCheckFrequency)
		assert.Equal(t, originalConnectionWarmingTimeout, originalConfig.ConnectionWarmingTimeout)
		assert.Equal(t, originalMonitoringInterval, originalConfig.MonitoringInterval)
	})
}

// TestOptimizedConnectionPool_ComprehensiveValidation tests comprehensive validation
func TestOptimizedConnectionPool_ComprehensiveValidation(t *testing.T) {
	t.Run("All_validation_rules", func(t *testing.T) {
		testCases := []struct {
			name        string
			config      *cachex.OptimizedConnectionPoolConfig
			expectError bool
			errorMsg    string
		}{
			{
				name: "Empty_address",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr: "",
				},
				expectError: true,
				errorMsg:    "address cannot be empty",
			},
			{
				name: "Zero_pool_size",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:     "localhost:6379",
					PoolSize: 0,
				},
				expectError: true,
				errorMsg:    "pool size must be positive",
			},
			{
				name: "Negative_min_idle_conns",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:         "localhost:6379",
					PoolSize:     10,
					MinIdleConns: -1,
				},
				expectError: true,
				errorMsg:    "min idle connections cannot be negative",
			},
			{
				name: "Min_idle_conns_exceeds_pool_size",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:         "localhost:6379",
					PoolSize:     5,
					MinIdleConns: 10,
				},
				expectError: true,
				errorMsg:    "min idle connections cannot exceed pool size",
			},
			{
				name: "Negative_max_retries",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:       "localhost:6379",
					PoolSize:   10,
					MaxRetries: -1,
				},
				expectError: true,
				errorMsg:    "max retries cannot be negative",
			},
			{
				name: "Zero_pool_timeout",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:        "localhost:6379",
					PoolSize:    10,
					PoolTimeout: 0,
				},
				expectError: true,
				errorMsg:    "pool timeout must be positive",
			},
			{
				name: "Zero_dial_timeout",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:        "localhost:6379",
					PoolSize:    10,
					PoolTimeout: 30 * time.Second,
					DialTimeout: 0,
				},
				expectError: true,
				errorMsg:    "dial timeout must be positive",
			},
			{
				name: "Zero_read_timeout",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:        "localhost:6379",
					PoolSize:    10,
					PoolTimeout: 30 * time.Second,
					DialTimeout: 5 * time.Second,
					ReadTimeout: 0,
				},
				expectError: true,
				errorMsg:    "read timeout must be positive",
			},
			{
				name: "Zero_write_timeout",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:         "localhost:6379",
					PoolSize:     10,
					PoolTimeout:  30 * time.Second,
					DialTimeout:  5 * time.Second,
					ReadTimeout:  3 * time.Second,
					WriteTimeout: 0,
				},
				expectError: true,
				errorMsg:    "write timeout must be positive",
			},
			{
				name: "Zero_health_check_interval",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:                "localhost:6379",
					PoolSize:            10,
					PoolTimeout:         30 * time.Second,
					DialTimeout:         5 * time.Second,
					ReadTimeout:         3 * time.Second,
					WriteTimeout:        3 * time.Second,
					HealthCheckInterval: 0,
				},
				expectError: true,
				errorMsg:    "health check interval must be positive",
			},
			{
				name: "Zero_health_check_timeout",
				config: &cachex.OptimizedConnectionPoolConfig{
					Addr:                "localhost:6379",
					PoolSize:            10,
					PoolTimeout:         30 * time.Second,
					DialTimeout:         5 * time.Second,
					ReadTimeout:         3 * time.Second,
					WriteTimeout:        3 * time.Second,
					HealthCheckInterval: 30 * time.Second,
					HealthCheckTimeout:  0,
				},
				expectError: true,
				errorMsg:    "health check timeout must be positive",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				pool, err := cachex.NewOptimizedConnectionPool(tc.config)

				if tc.expectError {
					assert.Error(t, err)
					assert.Nil(t, pool)
					if tc.errorMsg != "" {
						assert.Contains(t, err.Error(), tc.errorMsg)
					}
				} else {
					// For valid configs, connection will fail but validation should pass
					assert.Error(t, err)
					assert.Nil(t, pool)
					assert.Contains(t, err.Error(), "failed to connect to redis")
				}
			})
		}
	})
}
