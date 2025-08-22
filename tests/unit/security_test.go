package unit

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SeaSBee/go-cachex"
	"github.com/stretchr/testify/assert"
)

func TestNewValidator_WithNilConfig(t *testing.T) {
	validator, err := cachex.NewValidator(nil)
	if err != nil {
		t.Errorf("NewValidator() failed: %v", err)
	}
	if validator == nil {
		t.Errorf("NewValidator() should not return nil")
	}
}

func TestNewValidator_WithCustomConfig(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength:    100,
		MaxValueSize:    1024,
		AllowedPatterns: []string{`^[a-zA-Z0-9_]+$`},
		BlockedPatterns: []string{`^admin`},
	}

	validator, err := cachex.NewValidator(config)
	if err != nil {
		t.Errorf("NewValidator() failed: %v", err)
	}
	if validator == nil {
		t.Errorf("NewValidator() should not return nil")
	}
}

func TestNewValidator_WithInvalidAllowedPattern(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength:    100,
		MaxValueSize:    1024,
		AllowedPatterns: []string{`[invalid`}, // Invalid regex
	}

	_, err := cachex.NewValidator(config)
	if err == nil {
		t.Errorf("NewValidator() should fail with invalid allowed pattern")
	}
}

func TestNewValidator_WithInvalidBlockedPattern(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength:    100,
		MaxValueSize:    1024,
		BlockedPatterns: []string{`[invalid`}, // Invalid regex
	}

	_, err := cachex.NewValidator(config)
	if err == nil {
		t.Errorf("NewValidator() should fail with invalid blocked pattern")
	}
}

func TestValidator_ValidateKey_EmptyKey(t *testing.T) {
	validator, err := cachex.NewValidator(nil)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	err = validator.ValidateKey("")
	if err == nil {
		t.Errorf("ValidateKey() should fail for empty key")
	}
}

func TestValidator_ValidateKey_TooLong(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength: 5,
		MaxValueSize: 1024,
	}

	validator, err := cachex.NewValidator(config)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	err = validator.ValidateKey("very-long-key")
	if err == nil {
		t.Errorf("ValidateKey() should fail for too long key")
	}
}

func TestValidator_ValidateKey_BlockedPattern(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength:    100,
		MaxValueSize:    1024,
		BlockedPatterns: []string{`^admin`},
	}

	validator, err := cachex.NewValidator(config)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	err = validator.ValidateKey("admin-key")
	if err == nil {
		t.Errorf("ValidateKey() should fail for blocked pattern")
	}
}

func TestValidator_ValidateKey_AllowedPattern(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength:    100,
		MaxValueSize:    1024,
		AllowedPatterns: []string{`^[a-zA-Z0-9_]+$`},
	}

	validator, err := cachex.NewValidator(config)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	// Valid key
	err = validator.ValidateKey("valid_key123")
	if err != nil {
		t.Errorf("ValidateKey() should pass for valid key: %v", err)
	}

	// Invalid key
	err = validator.ValidateKey("invalid-key!")
	if err == nil {
		t.Errorf("ValidateKey() should fail for invalid key")
	}
}

func TestValidator_ValidateKey_NoPatterns(t *testing.T) {
	validator, err := cachex.NewValidator(nil)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	// Should pass for any non-empty key within length limit
	err = validator.ValidateKey("any-key")
	if err != nil {
		t.Errorf("ValidateKey() should pass for valid key: %v", err)
	}
}

func TestValidator_ValidateValue_NilValue(t *testing.T) {
	validator, err := cachex.NewValidator(nil)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	err = validator.ValidateValue(nil)
	if err == nil {
		t.Errorf("ValidateValue() should fail for nil value")
	}
}

func TestValidator_ValidateValue_TooLarge(t *testing.T) {
	config := &cachex.Config{
		MaxKeyLength: 100,
		MaxValueSize: 10,
	}

	validator, err := cachex.NewValidator(config)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	largeValue := make([]byte, 20)
	err = validator.ValidateValue(largeValue)
	if err == nil {
		t.Errorf("ValidateValue() should fail for too large value")
	}
}

func TestValidator_ValidateValue_Valid(t *testing.T) {
	validator, err := cachex.NewValidator(nil)
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	validValue := []byte("valid value")
	err = validator.ValidateValue(validValue)
	if err != nil {
		t.Errorf("ValidateValue() should pass for valid value: %v", err)
	}
}

func TestNewRedactor_WithValidPatterns(t *testing.T) {
	patterns := []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`}
	replacement := "[REDACTED]"

	redactor, err := cachex.NewRedactor(patterns, replacement)
	if err != nil {
		t.Errorf("NewRedactor() failed: %v", err)
	}
	if redactor == nil {
		t.Errorf("NewRedactor() should not return nil")
	}
}

func TestNewRedactor_WithInvalidPattern(t *testing.T) {
	patterns := []string{`[invalid`} // Invalid regex
	replacement := "[REDACTED]"

	_, err := cachex.NewRedactor(patterns, replacement)
	if err == nil {
		t.Errorf("NewRedactor() should fail with invalid pattern")
	}
}

func TestRedactor_Redact_Password(t *testing.T) {
	patterns := []string{`"password"\s*:\s*"[^"]*"`}
	replacement := `"password": "[REDACTED]"`

	redactor, err := cachex.NewRedactor(patterns, replacement)
	if err != nil {
		t.Fatalf("NewRedactor() failed: %v", err)
	}

	input := `{"user": "john", "password": "secret123"}`
	result := redactor.Redact(input)

	assert.Contains(t, result, `"password": "[REDACTED]"`)
	assert.Contains(t, result, `"user": "john"`)
}

func TestRedactor_Redact_MultiplePatterns(t *testing.T) {
	// Create separate redactors for each pattern
	passwordPattern := []string{`"password"\s*:\s*"[^"]*"`}
	passwordReplacement := `"password": "[REDACTED]"`

	tokenPattern := []string{`"token"\s*:\s*"[^"]*"`}
	tokenReplacement := `"token": "[REDACTED]"`

	passwordRedactor, err := cachex.NewRedactor(passwordPattern, passwordReplacement)
	if err != nil {
		t.Fatalf("NewRedactor() failed for password: %v", err)
	}

	tokenRedactor, err := cachex.NewRedactor(tokenPattern, tokenReplacement)
	if err != nil {
		t.Fatalf("NewRedactor() failed for token: %v", err)
	}

	input := `{"user": "john", "password": "secret123", "token": "abc123"}`

	// Apply redaction in sequence
	result := passwordRedactor.Redact(input)
	result = tokenRedactor.Redact(result)

	// Check that both password and token are redacted
	assert.Contains(t, result, `"password": "[REDACTED]"`)
	assert.Contains(t, result, `"token": "[REDACTED]"`)
	assert.Contains(t, result, `"user": "john"`)
}

func TestRedactor_Redact_NoMatch(t *testing.T) {
	patterns := []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`}
	replacement := "[REDACTED]"

	redactor, err := cachex.NewRedactor(patterns, replacement)
	if err != nil {
		t.Fatalf("NewRedactor() failed: %v", err)
	}

	input := `{"user": "john", "email": "john@example.com"}`
	result := redactor.Redact(input)

	if result != input {
		t.Errorf("Redact() should not change input when no pattern matches: got %s, want %s", result, input)
	}
}

func TestRBACAuthorizer_Authorize(t *testing.T) {
	authorizer := &cachex.RBACAuthorizer{}
	ctx := context.Background()
	authCtx := &cachex.AuthorizationContext{
		UserID:    "user123",
		Roles:     []string{"user"},
		Resource:  "cache",
		Action:    "read",
		Key:       "test-key",
		Timestamp: time.Now(),
	}

	result := authorizer.Authorize(ctx, authCtx)
	if result == nil {
		t.Errorf("Authorize() should not return nil")
		return
	}
	if !result.Allowed {
		t.Errorf("Authorize() should allow by default")
	}
	if result.Reason == "" {
		t.Errorf("Authorize() should provide a reason")
	}
}

func TestNewSecretsManager_WithCustomPrefix(t *testing.T) {
	prefix := "CUSTOM_"
	manager := cachex.NewSecretsManager(prefix)
	if manager == nil {
		t.Errorf("NewSecretsManager() should not return nil")
	}
}

func TestNewSecretsManager_WithEmptyPrefix(t *testing.T) {
	manager := cachex.NewSecretsManager("")
	if manager == nil {
		t.Errorf("NewSecretsManager() should not return nil")
	}
}

func TestSecretsManager_GetSecret_NotFound(t *testing.T) {
	manager := cachex.NewSecretsManager("TEST_")

	_, err := manager.GetSecret("nonexistent")
	if err == nil {
		t.Errorf("GetSecret() should fail for non-existent secret")
	}
}

func TestSecretsManager_GetSecret_Found(t *testing.T) {
	// Set environment variable for testing
	envName := "TEST_SECRET"
	expectedValue := "secret-value"
	os.Setenv(envName, expectedValue)
	defer os.Unsetenv(envName)

	manager := cachex.NewSecretsManager("TEST_")

	value, err := manager.GetSecret("secret")
	if err != nil {
		t.Errorf("GetSecret() failed: %v", err)
	}
	if value != expectedValue {
		t.Errorf("GetSecret() returned wrong value: got %s, want %s", value, expectedValue)
	}
}

func TestSecretsManager_GetSecretOrDefault_NotFound(t *testing.T) {
	manager := cachex.NewSecretsManager("TEST_")
	defaultValue := "default-value"

	value := manager.GetSecretOrDefault("nonexistent", defaultValue)
	if value != defaultValue {
		t.Errorf("GetSecretOrDefault() should return default value: got %s, want %s", value, defaultValue)
	}
}

func TestSecretsManager_GetSecretOrDefault_Found(t *testing.T) {
	// Set environment variable for testing
	envName := "TEST_SECRET"
	expectedValue := "secret-value"
	os.Setenv(envName, expectedValue)
	defer os.Unsetenv(envName)

	manager := cachex.NewSecretsManager("TEST_")
	defaultValue := "default-value"

	value := manager.GetSecretOrDefault("secret", defaultValue)
	if value != expectedValue {
		t.Errorf("GetSecretOrDefault() should return secret value: got %s, want %s", value, expectedValue)
	}
}

func TestSecretsManager_ListSecrets(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("TEST_SECRET1", "value1")
	os.Setenv("TEST_SECRET2", "value2")
	os.Setenv("OTHER_VAR", "value3")
	defer func() {
		os.Unsetenv("TEST_SECRET1")
		os.Unsetenv("TEST_SECRET2")
		os.Unsetenv("OTHER_VAR")
	}()

	manager := cachex.NewSecretsManager("TEST_")

	secrets := manager.ListSecrets()
	if len(secrets) < 2 {
		t.Errorf("ListSecrets() should return at least 2 secrets, got %d", len(secrets))
	}

	// Check that secrets are returned in lowercase
	for _, secret := range secrets {
		if secret != "secret1" && secret != "secret2" {
			t.Errorf("ListSecrets() returned unexpected secret: %s", secret)
		}
	}
}

func TestNewSecurityManager_WithNilConfig(t *testing.T) {
	manager, err := cachex.NewSecurityManager(nil)
	if err != nil {
		t.Errorf("NewSecurityManager() should not fail with nil config: %v", err)
	}
	if manager == nil {
		t.Errorf("NewSecurityManager() should return valid manager when config is nil")
	}
}

func TestNewSecurityManager_WithValidConfig(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Errorf("NewSecurityManager() failed: %v", err)
	}
	if manager == nil {
		t.Errorf("NewSecurityManager() should not return nil")
	}
}

func TestNewSecurityManager_WithInvalidValidationConfig(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength:    100,
			MaxValueSize:    1024,
			AllowedPatterns: []string{`[invalid`}, // Invalid regex
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	_, err := cachex.NewSecurityManager(config)
	if err == nil {
		t.Errorf("NewSecurityManager() should fail with invalid validation config")
	}
}

func TestNewSecurityManager_WithInvalidRedactionPattern(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`[invalid`}, // Invalid regex
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	_, err := cachex.NewSecurityManager(config)
	if err == nil {
		t.Errorf("NewSecurityManager() should fail with invalid redaction pattern")
	}
}

func TestSecurityManager_ValidateKey(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() failed: %v", err)
	}

	// Valid key
	err = manager.ValidateKey("valid-key")
	if err != nil {
		t.Errorf("ValidateKey() should pass for valid key: %v", err)
	}

	// Invalid key
	err = manager.ValidateKey("")
	if err == nil {
		t.Errorf("ValidateKey() should fail for empty key")
	}
}

func TestSecurityManager_ValidateValue(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() failed: %v", err)
	}

	// Valid value
	err = manager.ValidateValue([]byte("valid value"))
	if err != nil {
		t.Errorf("ValidateValue() should pass for valid value: %v", err)
	}

	// Invalid value
	err = manager.ValidateValue(nil)
	if err == nil {
		t.Errorf("ValidateValue() should fail for nil value")
	}
}

func TestSecurityManager_Redact(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() failed: %v", err)
	}

	input := `{"user": "john", "password": "secret123"}`
	result := manager.Redact(input)

	// The regex pattern might not match exactly, so just check that redaction occurred
	if result == input {
		t.Errorf("Redact() should modify the input, got unchanged result: %s", result)
	}

	// Check that password is redacted
	if strings.Contains(result, "secret123") {
		t.Errorf("Redact() should redact password, got: %s", result)
	}
}

func TestSecurityManager_Authorize(t *testing.T) {
	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() failed: %v", err)
	}

	ctx := context.Background()
	authCtx := &cachex.AuthorizationContext{
		UserID:    "user123",
		Roles:     []string{"user"},
		Resource:  "cache",
		Action:    "read",
		Key:       "test-key",
		Timestamp: time.Now(),
	}

	result := manager.Authorize(ctx, authCtx)
	if result == nil {
		t.Errorf("Authorize() should not return nil")
		return
	}
	if !result.Allowed {
		t.Errorf("Authorize() should allow by default")
	}
}

func TestSecurityManager_GetSecret(t *testing.T) {
	// Set environment variable for testing
	envName := "TEST_SECRET"
	expectedValue := "secret-value"
	os.Setenv(envName, expectedValue)
	defer os.Unsetenv(envName)

	config := &cachex.SecurityConfig{
		Validation: &cachex.Config{
			MaxKeyLength: 100,
			MaxValueSize: 1024,
		},
		RedactionPatterns:    []string{`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "TEST_",
	}

	manager, err := cachex.NewSecurityManager(config)
	if err != nil {
		t.Fatalf("NewSecurityManager() failed: %v", err)
	}

	value, err := manager.GetSecret("secret")
	if err != nil {
		t.Errorf("GetSecret() failed: %v", err)
	}
	if value != expectedValue {
		t.Errorf("GetSecret() returned wrong value: got %s, want %s", value, expectedValue)
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := cachex.DefaultSecurityConfig()
	if config == nil {
		t.Errorf("DefaultSecurityConfig() should not return nil")
		return
	}

	if config.Validation == nil {
		t.Errorf("DefaultSecurityConfig().Validation should not be nil")
	}

	if len(config.RedactionPatterns) == 0 {
		t.Errorf("DefaultSecurityConfig().RedactionPatterns should not be empty")
	}

	if config.RedactionReplacement == "" {
		t.Errorf("DefaultSecurityConfig().RedactionReplacement should not be empty")
	}

	if config.SecretsPrefix == "" {
		t.Errorf("DefaultSecurityConfig().SecretsPrefix should not be empty")
	}
}
