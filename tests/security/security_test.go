package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structures
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Secret   string `json:"secret"`
}

// Helper functions
func createSecurityManager(config *cachex.SecurityConfig) *cachex.SecurityManager {
	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create security manager: %v", err))
	}
	return manager
}

func createValidator(config *cachex.Config) *cachex.Validator {
	validator, err := cachex.NewValidator(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create validator: %v", err))
	}
	return validator
}

func createRedactor(patterns []string, replacement string) *cachex.Redactor {
	redactor, err := cachex.NewRedactor(patterns, replacement)
	if err != nil {
		panic(fmt.Sprintf("Failed to create redactor: %v", err))
	}
	return redactor
}

// Validator Tests
func TestValidatorKeyValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *cachex.Config
		key         string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid key",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
			},
			key:         "user:123",
			expectError: false,
		},
		{
			name: "Empty key",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
			},
			key:         "",
			expectError: true,
			errorMsg:    "key cannot be empty",
		},
		{
			name: "Key too long",
			config: &cachex.Config{
				MaxKeyLength: 10,
				MaxValueSize: 1024 * 1024,
			},
			key:         "very_long_key_that_exceeds_limit",
			expectError: true,
			errorMsg:    "key too long",
		},
		{
			name: "Key with blocked pattern - path traversal",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
				BlockedPatterns: []string{
					`^\.\./`,
				},
			},
			key:         "../etc/passwd",
			expectError: true,
			errorMsg:    "key matches blocked pattern",
		},
		{
			name: "Key with blocked pattern - XSS",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
				BlockedPatterns: []string{
					`javascript:`,
				},
			},
			key:         "javascript:alert('xss')",
			expectError: true,
			errorMsg:    "key matches blocked pattern",
		},
		{
			name: "Key with HTML injection",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
				BlockedPatterns: []string{
					`[<>\"'&]`,
				},
			},
			key:         "user<script>alert('xss')</script>",
			expectError: true,
			errorMsg:    "key matches blocked pattern",
		},
		{
			name: "Key with allowed pattern",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
				AllowedPatterns: []string{
					`^user:\d+$`,
				},
			},
			key:         "user:123",
			expectError: false,
		},
		{
			name: "Key not matching allowed pattern",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
				AllowedPatterns: []string{
					`^user:\d+$`,
				},
			},
			key:         "invalid_key",
			expectError: true,
			errorMsg:    "key does not match any allowed pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := createValidator(tt.config)
			err := validator.ValidateKey(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatorValueValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *cachex.Config
		value       []byte
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid value",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
			},
			value:       []byte("valid data"),
			expectError: false,
		},
		{
			name: "Nil value",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
			},
			value:       nil,
			expectError: true,
			errorMsg:    "value cannot be nil",
		},
		{
			name: "Value too large",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 100,
			},
			value:       []byte(strings.Repeat("a", 200)),
			expectError: true,
			errorMsg:    "value too large",
		},
		{
			name: "Empty value",
			config: &cachex.Config{
				MaxKeyLength: 256,
				MaxValueSize: 1024 * 1024,
			},
			value:       []byte{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := createValidator(tt.config)
			err := validator.ValidateValue(tt.value)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Redactor Tests
func TestRedactorBasicRedaction(t *testing.T) {
	patterns := []string{
		`"password"\s*:\s*"[^"]*"`,
		`"token"\s*:\s*"[^"]*"`,
		`"secret"\s*:\s*"[^"]*"`,
	}
	replacement := `"[REDACTED]"`

	redactor := createRedactor(patterns, replacement)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Redact password",
			input:    `{"user": "john", "password": "secret123", "email": "john@example.com"}`,
			expected: `{"user": "john", "[REDACTED]", "email": "john@example.com"}`,
		},
		{
			name:     "Redact token",
			input:    `{"user": "jane", "token": "abc123", "role": "admin"}`,
			expected: `{"user": "jane", "[REDACTED]", "role": "admin"}`,
		},
		{
			name:     "Redact secret",
			input:    `{"user": "bob", "secret": "xyz789", "department": "IT"}`,
			expected: `{"user": "bob", "[REDACTED]", "department": "IT"}`,
		},
		{
			name:     "Redact multiple fields",
			input:    `{"user": "alice", "password": "pass123", "token": "tok456", "secret": "sec789"}`,
			expected: `{"user": "alice", "[REDACTED]", "[REDACTED]", "[REDACTED]"}`,
		},
		{
			name:     "No sensitive data",
			input:    `{"user": "alice", "email": "alice@example.com", "role": "user"}`,
			expected: `{"user": "alice", "email": "alice@example.com", "role": "user"}`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactorComplexPatterns(t *testing.T) {
	patterns := []string{
		`password\s*=\s*[^\s]+`,                               // password=value
		`api_key\s*:\s*[a-zA-Z0-9]+`,                          // api_key: value
		`secret["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,            // secret: "value" or secret="value"
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email addresses
	}
	replacement := "[REDACTED]"

	redactor := createRedactor(patterns, replacement)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Redact password with equals",
			input:    "user=john password=secret123 email=john@example.com",
			expected: "user=john [REDACTED] email=[REDACTED]",
		},
		{
			name:     "Redact API key",
			input:    "api_key: abc123xyz user: john",
			expected: "[REDACTED] user: john",
		},
		{
			name:     "Redact email addresses",
			input:    "Contact: john@example.com or jane@test.org",
			expected: "Contact: [REDACTED] or [REDACTED]",
		},
		{
			name:     "Redact secret with quotes",
			input:    `secret: "my_secret_value" user: "john"`,
			expected: `[REDACTED] user: "john"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactorInvalidPatterns(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		replacement string
		expectError bool
	}{
		{
			name:        "Invalid regex pattern",
			patterns:    []string{`[invalid`},
			replacement: "[REDACTED]",
			expectError: true,
		},
		{
			name:        "Valid patterns",
			patterns:    []string{`password\s*=\s*[^\s]+`},
			replacement: "[REDACTED]",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cachex.NewRedactor(tt.patterns, tt.replacement)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// RBAC Authorization Tests
func TestRBACAuthorization(t *testing.T) {
	authorizer := &cachex.RBACAuthorizer{}

	tests := []struct {
		name     string
		authCtx  *cachex.AuthorizationContext
		expected bool
	}{
		{
			name: "Basic authorization context",
			authCtx: &cachex.AuthorizationContext{
				UserID:    "user123",
				Roles:     []string{"user"},
				Resource:  "cache",
				Action:    "read",
				Key:       "user:123",
				Timestamp: time.Now(),
			},
			expected: true, // Default implementation always allows
		},
		{
			name: "Admin user",
			authCtx: &cachex.AuthorizationContext{
				UserID:    "admin",
				Roles:     []string{"admin", "user"},
				Resource:  "cache",
				Action:    "write",
				Key:       "system:config",
				Timestamp: time.Now(),
			},
			expected: true,
		},
		{
			name: "Empty context",
			authCtx: &cachex.AuthorizationContext{
				UserID:    "",
				Roles:     []string{},
				Resource:  "",
				Action:    "",
				Key:       "",
				Timestamp: time.Time{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authorizer.Authorize(context.Background(), tt.authCtx)
			assert.Equal(t, tt.expected, result.Allowed)
			assert.NotEmpty(t, result.Reason)
		})
	}
}

// Secrets Manager Tests
func TestSecretsManager(t *testing.T) {
	// Set up test environment variables
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set test secrets
	os.Setenv("VAULT_DB_PASSWORD", "secret_db_password")
	os.Setenv("VAULT_API_KEY", "secret_api_key")
	os.Setenv("VAULT_REDIS_PASSWORD", "secret_redis_password")
	os.Setenv("OTHER_VAR", "not_a_secret")

	tests := []struct {
		name           string
		prefix         string
		secretName     string
		expectedValue  string
		expectError    bool
		defaultValue   string
		expectedResult string
	}{
		{
			name:          "Get existing secret with default prefix",
			prefix:        "",
			secretName:    "db_password",
			expectedValue: "secret_db_password",
			expectError:   false,
		},
		{
			name:          "Get existing secret with custom prefix",
			prefix:        "VAULT_",
			secretName:    "api_key",
			expectedValue: "secret_api_key",
			expectError:   false,
		},
		{
			name:        "Get non-existent secret",
			prefix:      "VAULT_",
			secretName:  "non_existent",
			expectError: true,
		},
		{
			name:           "Get secret with default value",
			prefix:         "VAULT_",
			secretName:     "non_existent",
			defaultValue:   "default_value",
			expectedResult: "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsManager := cachex.NewSecretsManager(tt.prefix)

			if tt.defaultValue != "" {
				// Test GetSecretOrDefault
				result := secretsManager.GetSecretOrDefault(tt.secretName, tt.defaultValue)
				assert.Equal(t, tt.expectedResult, result)
			} else {
				// Test GetSecret
				value, err := secretsManager.GetSecret(tt.secretName)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedValue, value)
				}
			}
		})
	}
}

func TestSecretsManagerListSecrets(t *testing.T) {
	// Set up test environment variables
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set test secrets
	os.Setenv("VAULT_DB_PASSWORD", "secret_db_password")
	os.Setenv("VAULT_API_KEY", "secret_api_key")
	os.Setenv("VAULT_REDIS_PASSWORD", "secret_redis_password")
	os.Setenv("OTHER_VAR", "not_a_secret")

	tests := []struct {
		name            string
		prefix          string
		expectedCount   int
		expectedSecrets []string
	}{
		{
			name:          "List secrets with default prefix",
			prefix:        "",
			expectedCount: 3,
			expectedSecrets: []string{
				"db_password",
				"api_key",
				"redis_password",
			},
		},
		{
			name:          "List secrets with custom prefix",
			prefix:        "VAULT_",
			expectedCount: 3,
			expectedSecrets: []string{
				"db_password",
				"api_key",
				"redis_password",
			},
		},
		{
			name:          "List secrets with non-matching prefix",
			prefix:        "CUSTOM_",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsManager := cachex.NewSecretsManager(tt.prefix)
			secrets := secretsManager.ListSecrets()

			assert.Len(t, secrets, tt.expectedCount)

			if tt.expectedSecrets != nil {
				for _, expectedSecret := range tt.expectedSecrets {
					assert.Contains(t, secrets, expectedSecret)
				}
			}
		})
	}
}

// Security Manager Integration Tests
func TestSecurityManagerIntegration(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024,
			BlockedPatterns: []string{
				`^\.\./`,
				`javascript:`,
			},
		},
		RedactionPatterns: []string{
			`"password"\s*:\s*"[^"]*"`,
			`"token"\s*:\s*"[^"]*"`,
		},
		RedactionReplacement: `"[REDACTED]"`,
		SecretsPrefix:        "VAULT_",
	}

	// Set up test environment
	os.Setenv("VAULT_TEST_SECRET", "test_secret_value")
	defer os.Unsetenv("VAULT_TEST_SECRET")

	securityManager := createSecurityManager(config)

	t.Run("Key validation", func(t *testing.T) {
		// Valid key
		err := securityManager.ValidateKey("user:123")
		assert.NoError(t, err)

		// Invalid key - path traversal
		err = securityManager.ValidateKey("../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key matches blocked pattern")

		// Invalid key - XSS
		err = securityManager.ValidateKey("javascript:alert('xss')")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key matches blocked pattern")
	})

	t.Run("Value validation", func(t *testing.T) {
		// Valid value
		err := securityManager.ValidateValue([]byte("valid data"))
		assert.NoError(t, err)

		// Invalid value - nil
		err = securityManager.ValidateValue(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value cannot be nil")
	})

	t.Run("Redaction", func(t *testing.T) {
		input := `{"user": "john", "password": "secret123", "token": "abc123"}`
		expected := `{"user": "john", "[REDACTED]", "[REDACTED]"}`

		result := securityManager.Redact(input)
		assert.Equal(t, expected, result)
	})

	t.Run("Authorization", func(t *testing.T) {
		authCtx := &cachex.AuthorizationContext{
			UserID:    "user123",
			Roles:     []string{"user"},
			Resource:  "cache",
			Action:    "read",
			Key:       "user:123",
			Timestamp: time.Now(),
		}

		result := securityManager.Authorize(context.Background(), authCtx)
		assert.True(t, result.Allowed)
		assert.NotEmpty(t, result.Reason)
	})

	t.Run("Secrets management", func(t *testing.T) {
		// Get existing secret
		secret, err := securityManager.GetSecret("test_secret")
		assert.NoError(t, err)
		assert.Equal(t, "test_secret_value", secret)

		// Get non-existent secret
		_, err = securityManager.GetSecret("non_existent")
		assert.Error(t, err)
	})
}

// Security Configuration Tests
func TestSecurityConfigDefaults(t *testing.T) {
	config := cachex.DefaultSecurityConfig()

	// Test validation config
	assert.NotNil(t, config.Validation)
	assert.Equal(t, 256, config.Validation.MaxKeyLength)
	assert.Equal(t, 1024*1024, config.Validation.MaxValueSize)
	assert.NotEmpty(t, config.Validation.BlockedPatterns)

	// Test redaction config
	assert.NotEmpty(t, config.RedactionPatterns)
	assert.Equal(t, "[REDACTED]", config.RedactionReplacement)

	// Test secrets config
	assert.Equal(t, "VAULT_", config.SecretsPrefix)
}

func TestSecurityConfigCustom(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 512,
			AllowedPatterns: []string{
				`^user:\d+$`,
			},
			BlockedPatterns: []string{
				`^admin:`,
			},
		},
		RedactionPatterns: []string{
			`"custom_field"\s*:\s*"[^"]*"`,
		},
		RedactionReplacement: "[CUSTOM_REDACTED]",
		SecretsPrefix:        "CUSTOM_",
	}

	securityManager := createSecurityManager(config)

	t.Run("Custom validation", func(t *testing.T) {
		// Test allowed pattern
		err := securityManager.ValidateKey("user:123")
		assert.NoError(t, err)

		// Test blocked pattern
		err = securityManager.ValidateKey("admin:config")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key matches blocked pattern")

		// Test key too long
		err = securityManager.ValidateKey(strings.Repeat("a", 150))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key too long")
	})

	t.Run("Custom redaction", func(t *testing.T) {
		input := `{"user": "john", "custom_field": "sensitive_data"}`
		expected := `{"user": "john", [CUSTOM_REDACTED]}`

		result := securityManager.Redact(input)
		assert.Equal(t, expected, result)
	})
}

// Cache Integration with Security Tests
func TestCacheWithSecurity(t *testing.T) {
	// Create a Redis store for testing
	store, err := cachex.NewRedisStore(&cachex.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	// Create security manager
	securityConfig := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024,
			BlockedPatterns: []string{
				`^\.\./`,
				`javascript:`,
			},
		},
		RedactionPatterns: []string{
			`"password"\s*:\s*"[^"]*"`,
		},
		RedactionReplacement: `"[REDACTED]"`,
	}

	securityManager := createSecurityManager(securityConfig)

	// Create cache with security validation
	cache, err := cachex.New[User](cachex.WithStore(store))
	require.NoError(t, err)
	defer cache.Close()

	t.Run("Secure cache operations", func(t *testing.T) {
		// Test with valid data
		user := User{
			ID:       1,
			Name:     "John Doe",
			Email:    "john@example.com",
			Password: "secret123",
		}

		// Validate key before setting
		key := "user:1"
		err := securityManager.ValidateKey(key)
		assert.NoError(t, err)

		// Set data in cache
		setResult := <-cache.Set(context.Background(), key, user, 5*time.Minute)
		assert.NoError(t, setResult.Error)

		// Get data from cache
		getResult := <-cache.Get(context.Background(), key)
		assert.NoError(t, getResult.Error)
		assert.True(t, getResult.Found)
		retrievedUser := getResult.Value
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Name, retrievedUser.Name)

		// Test redaction on retrieved data
		userJSON, _ := json.Marshal(retrievedUser)
		redactedJSON := securityManager.Redact(string(userJSON))
		assert.Contains(t, redactedJSON, "[REDACTED]")
		assert.NotContains(t, redactedJSON, "secret123")
	})

	t.Run("Blocked key patterns", func(t *testing.T) {
		// Test path traversal attempt
		key := "../etc/passwd"
		err := securityManager.ValidateKey(key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key matches blocked pattern")

		// Test XSS attempt
		key = "javascript:alert('xss')"
		err = securityManager.ValidateKey(key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key matches blocked pattern")
	})
}

// Performance Tests for Security Components
func TestSecurityPerformance(t *testing.T) {
	config := cachex.DefaultSecurityConfig()
	securityManager := createSecurityManager(config)

	t.Run("Key validation performance", func(t *testing.T) {
		keys := []string{
			"user:123",
			"product:456",
			"order:789",
			"../etc/passwd",           // This should be blocked
			"javascript:alert('xss')", // This should be blocked
		}

		for _, key := range keys {
			start := time.Now()
			err := securityManager.ValidateKey(key)
			duration := time.Since(start)

			// Validation should be fast (< 1ms)
			assert.Less(t, duration, time.Millisecond)

			if strings.Contains(key, "../") || strings.Contains(key, "javascript:") {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		}
	})

	t.Run("Redaction performance", func(t *testing.T) {
		input := `{"user": "john", "password": "secret123", "token": "abc123", "email": "john@example.com"}`

		start := time.Now()
		result := securityManager.Redact(input)
		duration := time.Since(start)

		// Redaction should be fast (< 1ms)
		assert.Less(t, duration, time.Millisecond)
		assert.Contains(t, result, "[REDACTED]")
		assert.NotContains(t, result, "secret123")
		assert.NotContains(t, result, "abc123")
	})
}

// Error Handling Tests
func TestSecurityErrorHandling(t *testing.T) {
	t.Run("Invalid validation config", func(t *testing.T) {
		config := &cachex.SecurityConfig{
			Validation: &cachex.Config{
				BlockedPatterns: []string{
					`[invalid`, // Invalid regex
				},
			},
		}

		_, err := cachex.NewSecurityManager(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid blocked pattern")
	})

	t.Run("Invalid redaction config", func(t *testing.T) {
		config := &cachex.SecurityConfig{
			RedactionPatterns: []string{
				`[invalid`, // Invalid regex
			},
		}

		_, err := cachex.NewSecurityManager(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid redaction pattern")
	})

	t.Run("Nil config", func(t *testing.T) {
		securityManager, err := cachex.NewSecurityManager(nil)
		assert.NoError(t, err)
		assert.NotNil(t, securityManager)
	})
}
