package security

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Validator provides input validation utilities
type Validator struct {
	maxKeyLength    int
	maxValueSize    int
	allowedPatterns []*regexp.Regexp
	blockedPatterns []*regexp.Regexp
}

// Config holds validation configuration
type Config struct {
	MaxKeyLength    int      // Maximum key length
	MaxValueSize    int      // Maximum value size in bytes
	AllowedPatterns []string // Allowed key patterns (regex)
	BlockedPatterns []string // Blocked key patterns (regex)
}

// NewValidator creates a new validator
func NewValidator(config *Config) (*Validator, error) {
	if config == nil {
		config = &Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024, // 1MB
		}
	}

	v := &Validator{
		maxKeyLength:    config.MaxKeyLength,
		maxValueSize:    config.MaxValueSize,
		allowedPatterns: make([]*regexp.Regexp, 0),
		blockedPatterns: make([]*regexp.Regexp, 0),
	}

	// Compile allowed patterns
	for _, pattern := range config.AllowedPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed pattern %s: %w", pattern, err)
		}
		v.allowedPatterns = append(v.allowedPatterns, re)
	}

	// Compile blocked patterns
	for _, pattern := range config.BlockedPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid blocked pattern %s: %w", pattern, err)
		}
		v.blockedPatterns = append(v.blockedPatterns, re)
	}

	return v, nil
}

// ValidateKey validates a cache key
func (v *Validator) ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if len(key) > v.maxKeyLength {
		return fmt.Errorf("key too long: %d > %d", len(key), v.maxKeyLength)
	}

	// Check blocked patterns first
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(key) {
			return fmt.Errorf("key matches blocked pattern: %s", pattern.String())
		}
	}

	// Check allowed patterns if any are defined
	if len(v.allowedPatterns) > 0 {
		allowed := false
		for _, pattern := range v.allowedPatterns {
			if pattern.MatchString(key) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("key does not match any allowed pattern")
		}
	}

	return nil
}

// ValidateValue validates a cache value
func (v *Validator) ValidateValue(value []byte) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}

	if len(value) > v.maxValueSize {
		return fmt.Errorf("value too large: %d > %d bytes", len(value), v.maxValueSize)
	}

	return nil
}

// Redactor provides log redaction utilities
type Redactor struct {
	patterns    []*regexp.Regexp
	replacement string
}

// NewRedactor creates a new redactor
func NewRedactor(patterns []string, replacement string) (*Redactor, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid redaction pattern %s: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}

	return &Redactor{
		patterns:    compiled,
		replacement: replacement,
	}, nil
}

// Redact redacts sensitive information from a string
func (r *Redactor) Redact(input string) string {
	result := input
	for _, pattern := range r.patterns {
		result = pattern.ReplaceAllString(result, r.replacement)
	}
	return result
}

// RBACAuthorizer provides RBAC authorization hooks
type RBACAuthorizer struct {
	// No-op default implementation
}

// AuthorizationContext holds authorization context
type AuthorizationContext struct {
	UserID    string
	Roles     []string
	Resource  string
	Action    string
	Key       string
	Timestamp time.Time
}

// AuthorizationResult holds authorization result
type AuthorizationResult struct {
	Allowed bool
	Reason  string
}

// Authorize checks if an operation is authorized
func (a *RBACAuthorizer) Authorize(ctx context.Context, authCtx *AuthorizationContext) *AuthorizationResult {
	// No-op default implementation - always allow
	return &AuthorizationResult{
		Allowed: true,
		Reason:  "no-op default implementation",
	}
}

// SecretsManager provides secrets management utilities
type SecretsManager struct {
	// Environment variable prefix for secrets
	envPrefix string
}

// NewSecretsManager creates a new secrets manager
func NewSecretsManager(envPrefix string) *SecretsManager {
	if envPrefix == "" {
		envPrefix = "VAULT_"
	}

	return &SecretsManager{
		envPrefix: envPrefix,
	}
}

// GetSecret retrieves a secret from environment variables
func (sm *SecretsManager) GetSecret(name string) (string, error) {
	envName := sm.envPrefix + strings.ToUpper(name)
	value := os.Getenv(envName)

	if value == "" {
		return "", fmt.Errorf("secret not found: %s", envName)
	}

	return value, nil
}

// GetSecretOrDefault retrieves a secret with a default value
func (sm *SecretsManager) GetSecretOrDefault(name, defaultValue string) string {
	value, err := sm.GetSecret(name)
	if err != nil {
		return defaultValue
	}
	return value
}

// ListSecrets lists all available secrets
func (sm *SecretsManager) ListSecrets() []string {
	var secrets []string

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, sm.envPrefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				secretName := strings.TrimPrefix(parts[0], sm.envPrefix)
				secrets = append(secrets, strings.ToLower(secretName))
			}
		}
	}

	return secrets
}

// SecurityManager provides comprehensive security management
type SecurityManager struct {
	validator  *Validator
	redactor   *Redactor
	authorizer *RBACAuthorizer
	secrets    *SecretsManager
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error) {
	validator, err := NewValidator(config.Validation)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	redactor, err := NewRedactor(config.RedactionPatterns, config.RedactionReplacement)
	if err != nil {
		return nil, fmt.Errorf("failed to create redactor: %w", err)
	}

	authorizer := &RBACAuthorizer{}
	secrets := NewSecretsManager(config.SecretsPrefix)

	return &SecurityManager{
		validator:  validator,
		redactor:   redactor,
		authorizer: authorizer,
		secrets:    secrets,
	}, nil
}

// ValidateKey validates a cache key
func (sm *SecurityManager) ValidateKey(key string) error {
	return sm.validator.ValidateKey(key)
}

// ValidateValue validates a cache value
func (sm *SecurityManager) ValidateValue(value []byte) error {
	return sm.validator.ValidateValue(value)
}

// Redact redacts sensitive information
func (sm *SecurityManager) Redact(input string) string {
	return sm.redactor.Redact(input)
}

// Authorize checks authorization
func (sm *SecurityManager) Authorize(ctx context.Context, authCtx *AuthorizationContext) *AuthorizationResult {
	return sm.authorizer.Authorize(ctx, authCtx)
}

// GetSecret retrieves a secret
func (sm *SecurityManager) GetSecret(name string) (string, error) {
	return sm.secrets.GetSecret(name)
}

// SecurityConfig holds comprehensive security configuration
type SecurityConfig struct {
	Validation           *Config
	RedactionPatterns    []string
	RedactionReplacement string
	SecretsPrefix        string
}

// DefaultSecurityConfig returns a default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Validation: &Config{
			MaxKeyLength: 256,
			MaxValueSize: 1024 * 1024, // 1MB
			BlockedPatterns: []string{
				`^\.\./`,         // Path traversal
				`[<>\"'&]`,       // HTML/XML injection
				`javascript:`,    // XSS
				`data:text/html`, // XSS
			},
		},
		RedactionPatterns: []string{
			`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`, // Passwords
			`token["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,    // Tokens
			`key["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,      // Keys
			`secret["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,   // Secrets
		},
		RedactionReplacement: "[REDACTED]",
		SecretsPrefix:        "VAULT_",
	}
}
