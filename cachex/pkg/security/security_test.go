package security

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	// Test with valid config
	config := &Config{
		MaxKeyLength: 256,
		MaxValueSize: 1024 * 1024,
		BlockedPatterns: []string{
			`^\.\./`,
			`[<>\"'&]`,
		},
	}

	validator, err := NewValidator(config)
	require.NoError(t, err)
	assert.NotNil(t, validator)
}

func TestNewValidator_InvalidPatterns(t *testing.T) {
	// Test with invalid regex pattern
	config := &Config{
		MaxKeyLength: 256,
		MaxValueSize: 1024 * 1024,
		BlockedPatterns: []string{
			`[invalid`, // Invalid regex
		},
	}

	_, err := NewValidator(config)
	assert.Error(t, err)
}

func TestValidator_ValidateKey(t *testing.T) {
	validator, err := NewValidator(&Config{
		MaxKeyLength: 256,
		MaxValueSize: 1024 * 1024,
		BlockedPatterns: []string{
			`^\.\./`,
			`[<>\"'&]`,
		},
	})
	require.NoError(t, err)

	// Test valid keys
	validKeys := []string{
		"user:123",
		"product:456",
		"normal-key",
		"key_with_underscores",
		"key-with-dashes",
	}

	for _, key := range validKeys {
		err := validator.ValidateKey(key)
		assert.NoError(t, err, "Key should be valid: %s", key)
	}

	// Test invalid keys
	invalidKeys := []string{
		"",                              // Empty key
		"../etc/passwd",                 // Path traversal
		"<script>alert('xss')</script>", // XSS
		"javascript:alert('xss')",       // XSS
	}

	for _, key := range invalidKeys {
		err := validator.ValidateKey(key)
		assert.Error(t, err, "Key should be invalid: %s", key)
	}

	// Test key length limit
	longKey := string(make([]byte, 300)) // 300 bytes
	err = validator.ValidateKey(longKey)
	assert.Error(t, err, "Key should be too long")
}

func TestValidator_ValidateValue(t *testing.T) {
	validator, err := NewValidator(&Config{
		MaxKeyLength: 256,
		MaxValueSize: 1024, // 1KB
	})
	require.NoError(t, err)

	// Test valid values
	validValues := [][]byte{
		[]byte("normal data"),
		[]byte(""),
		make([]byte, 512), // 512 bytes
	}

	for _, value := range validValues {
		err := validator.ValidateValue(value)
		assert.NoError(t, err, "Value should be valid")
	}

	// Test invalid values
	largeValue := make([]byte, 2048) // 2KB
	err = validator.ValidateValue(largeValue)
	assert.Error(t, err, "Value should be too large")

	// Test nil value
	err = validator.ValidateValue(nil)
	assert.Error(t, err, "Nil value should be invalid")
}

func TestNewRedactor(t *testing.T) {
	// Test with valid patterns
	patterns := []string{
		`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,
		`token["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,
	}

	redactor, err := NewRedactor(patterns, "[REDACTED]")
	require.NoError(t, err)
	assert.NotNil(t, redactor)
}

func TestNewRedactor_InvalidPatterns(t *testing.T) {
	// Test with invalid regex pattern
	patterns := []string{
		`[invalid`, // Invalid regex
	}

	_, err := NewRedactor(patterns, "[REDACTED]")
	assert.Error(t, err)
}

func TestRedactor_Redact(t *testing.T) {
	redactor, err := NewRedactor([]string{
		`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,
		`token["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,
		`key["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,
	}, "[REDACTED]")
	require.NoError(t, err)

	// Test basic redaction functionality
	input := `{"user": "john", "password": "secret123"}`
	redacted := redactor.Redact(input)

	// Should contain redacted values
	assert.Contains(t, redacted, "[REDACTED]")
	assert.NotContains(t, redacted, "secret123")

	// Test that normal data is not redacted
	normalInput := `{"normal": "data", "no_sensitive": "info"}`
	normalRedacted := redactor.Redact(normalInput)
	assert.Equal(t, normalInput, normalRedacted)
}

func TestRBACAuthorizer_Authorize(t *testing.T) {
	authorizer := &RBACAuthorizer{}

	// Test authorization context
	authCtx := &AuthorizationContext{
		UserID:    "user123",
		Roles:     []string{"admin", "user"},
		Resource:  "cache",
		Action:    "read",
		Key:       "user:123",
		Timestamp: time.Now(),
	}

	result := authorizer.Authorize(context.Background(), authCtx)
	assert.True(t, result.Allowed, "Should be allowed by default")
	assert.Equal(t, "no-op default implementation", result.Reason)
}

func TestSecretsManager_GetSecret(t *testing.T) {
	// Set up test environment variables
	os.Setenv("VAULT_TEST_SECRET", "test-secret-value")
	os.Setenv("VAULT_ANOTHER_SECRET", "another-secret-value")

	secretsManager := NewSecretsManager("VAULT_")

	// Test getting existing secret
	secret, err := secretsManager.GetSecret("test_secret")
	require.NoError(t, err)
	assert.Equal(t, "test-secret-value", secret)

	// Test getting non-existent secret
	_, err = secretsManager.GetSecret("non_existent")
	assert.Error(t, err)
}

func TestSecretsManager_GetSecretOrDefault(t *testing.T) {
	secretsManager := NewSecretsManager("VAULT_")

	// Test with existing secret
	os.Setenv("VAULT_EXISTING_SECRET", "existing-value")
	value := secretsManager.GetSecretOrDefault("existing_secret", "default-value")
	assert.Equal(t, "existing-value", value)

	// Test with non-existent secret
	value = secretsManager.GetSecretOrDefault("non_existent", "default-value")
	assert.Equal(t, "default-value", value)
}

func TestSecretsManager_ListSecrets(t *testing.T) {
	// Set up test environment variables
	os.Setenv("VAULT_SECRET1", "value1")
	os.Setenv("VAULT_SECRET2", "value2")
	os.Setenv("OTHER_VAR", "other-value")

	secretsManager := NewSecretsManager("VAULT_")
	secrets := secretsManager.ListSecrets()

	// Should contain VAULT_ prefixed secrets
	assert.Contains(t, secrets, "secret1")
	assert.Contains(t, secrets, "secret2")
	assert.NotContains(t, secrets, "other_var")
}

func TestNewSecurityManager(t *testing.T) {
	config := DefaultSecurityConfig()
	securityManager, err := NewSecurityManager(config)
	require.NoError(t, err)
	assert.NotNil(t, securityManager)
}

func TestSecurityManager_ValidateKey(t *testing.T) {
	securityManager, err := NewSecurityManager(DefaultSecurityConfig())
	require.NoError(t, err)

	// Test valid key
	err = securityManager.ValidateKey("user:123")
	assert.NoError(t, err)

	// Test invalid key
	err = securityManager.ValidateKey("../etc/passwd")
	assert.Error(t, err)
}

func TestSecurityManager_ValidateValue(t *testing.T) {
	securityManager, err := NewSecurityManager(DefaultSecurityConfig())
	require.NoError(t, err)

	// Test valid value
	err = securityManager.ValidateValue([]byte("normal data"))
	assert.NoError(t, err)

	// Test large value
	largeValue := make([]byte, 2*1024*1024) // 2MB
	err = securityManager.ValidateValue(largeValue)
	assert.Error(t, err)
}

func TestSecurityManager_Redact(t *testing.T) {
	securityManager, err := NewSecurityManager(DefaultSecurityConfig())
	require.NoError(t, err)

	// Test redaction
	input := `{"user": "admin", "password": "secret123", "token": "abc123"}`
	redacted := securityManager.Redact(input)

	// Should contain redacted values
	assert.Contains(t, redacted, "[REDACTED]")
	assert.NotContains(t, redacted, "secret123")
	assert.NotContains(t, redacted, "abc123")
}

func TestSecurityManager_Authorize(t *testing.T) {
	securityManager, err := NewSecurityManager(DefaultSecurityConfig())
	require.NoError(t, err)

	authCtx := &AuthorizationContext{
		UserID: "user123", Roles: []string{"admin"}, Resource: "cache", Action: "write", Key: "user:123",
	}

	result := securityManager.Authorize(context.Background(), authCtx)
	assert.True(t, result.Allowed)
}

func TestSecurityManager_GetSecret(t *testing.T) {
	os.Setenv("VAULT_TEST_SECRET", "test-value")

	securityManager, err := NewSecurityManager(DefaultSecurityConfig())
	require.NoError(t, err)

	secret, err := securityManager.GetSecret("test_secret")
	require.NoError(t, err)
	assert.Equal(t, "test-value", secret)
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()
	assert.NotNil(t, config)
	assert.NotNil(t, config.Validation)
	assert.NotEmpty(t, config.RedactionPatterns)
	assert.Equal(t, "[REDACTED]", config.RedactionReplacement)
	assert.Equal(t, "VAULT_", config.SecretsPrefix)
}
