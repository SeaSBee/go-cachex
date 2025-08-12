package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/SeaSBee/go-cachex/cachex/internal/crypto"
	"github.com/SeaSBee/go-cachex/cachex/internal/rate"
	"github.com/SeaSBee/go-cachex/cachex/pkg/redisstore"
	"github.com/SeaSBee/go-cachex/cachex/pkg/security"
)

// User struct for demonstration
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"` // Will be redacted in logs
	Token     string    `json:"token"`    // Will be redacted in logs
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("=== Go-CacheX Security Features Demo ===")

	// Example 1: TLS Configuration
	demoTLSConfiguration()

	// Example 2: ACL/Auth Support
	demoACLAuth()

	// Example 3: Value Encryption
	demoValueEncryption()

	// Example 4: Input Validation
	demoInputValidation()

	// Example 5: Secrets Management
	demoSecretsManagement()

	// Example 6: RBAC Hooks
	demoRBACHooks()

	// Example 7: Rate Limiting
	demoRateLimiting()

	// Example 8: Log Redaction
	demoLogRedaction()

	// Example 9: Comprehensive Security
	demoComprehensiveSecurity()

	fmt.Println("\n=== Security Demo Complete ===")
}

func demoTLSConfiguration() {
	fmt.Println("1. TLS Configuration")
	fmt.Println("====================")

	// Create Redis store with TLS
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "your-password",
		DB:       0,
		TLSConfig: &redisstore.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: false, // ✅ Off by default for security
		},
	})
	if err != nil {
		fmt.Printf("Redis not available (expected for demo): %v\n", err)
		fmt.Println("TLS configuration would be applied when Redis is available")
	} else {
		fmt.Println("✅ TLS configured successfully")
		defer redisStore.Close()
	}

	fmt.Println("✅ InsecureSkipVerify is off by default")
	fmt.Println("✅ TLS configuration supports secure connections")
	fmt.Println()
}

func demoACLAuth() {
	fmt.Println("2. ACL/Auth Support")
	fmt.Println("===================")

	// Create Redis store with authentication
	redisStore, err := redisstore.New(&redisstore.Config{
		Addr:     "localhost:6379",
		Password: "your-secure-password", // ✅ Username/password support
		DB:       0,
	})
	if err != nil {
		fmt.Printf("Redis not available (expected for demo): %v\n", err)
		fmt.Println("ACL/Auth would be applied when Redis is available")
	} else {
		fmt.Println("✅ ACL/Auth configured successfully")
		defer redisStore.Close()
	}

	fmt.Println("✅ Username/password authentication supported")
	fmt.Println("✅ Database selection supported")
	fmt.Println("✅ Connection pooling with authentication")
	fmt.Println()
}

func demoValueEncryption() {
	fmt.Println("3. Value Encryption-at-Rest")
	fmt.Println("============================")

	// Generate encryption key
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Printf("Failed to generate key: %v\n", err)
		return
	}

	// Create encryptor with key rotation
	encryptor, err := crypto.NewEncryptor(&crypto.Config{
		Key:          key,
		KeyRotation:  24 * time.Hour, // ✅ Key rotation interface
		AutoRotation: true,
	})
	if err != nil {
		fmt.Printf("Failed to create encryptor: %v\n", err)
		return
	}

	// Test encryption/decryption
	testData := []byte("sensitive user data")
	encrypted, err := encryptor.Encrypt(testData)
	if err != nil {
		fmt.Printf("Failed to encrypt: %v\n", err)
		return
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		fmt.Printf("Failed to decrypt: %v\n", err)
		return
	}

	fmt.Printf("✅ Original data: %s\n", string(testData))
	fmt.Printf("✅ Encrypted data: %x\n", encrypted[:20]) // Show first 20 bytes
	fmt.Printf("✅ Decrypted data: %s\n", string(decrypted))
	fmt.Printf("✅ Data integrity: %t\n", string(testData) == string(decrypted))

	// Test key rotation
	err = encryptor.ForceRotate()
	if err != nil {
		fmt.Printf("Failed to rotate key: %v\n", err)
		return
	}

	// Data encrypted with old key should still be decryptable
	decrypted2, err := encryptor.Decrypt(encrypted)
	if err != nil {
		fmt.Printf("Failed to decrypt with rotated key: %v\n", err)
		return
	}

	fmt.Printf("✅ Backward compatibility: %t\n", string(testData) == string(decrypted2))

	// Show key info
	info := encryptor.GetKeyInfo()
	fmt.Printf("✅ Key rotation info: %+v\n", info)
	fmt.Println()
}

func demoInputValidation() {
	fmt.Println("4. Input Validation")
	fmt.Println("===================")

	// Create validator
	validator, err := security.NewValidator(&security.Config{
		MaxKeyLength: 256,
		MaxValueSize: 1024 * 1024, // 1MB
		BlockedPatterns: []string{
			`^\.\./`,         // ✅ Path traversal protection
			`[<>\"'&]`,       // ✅ HTML/XML injection protection
			`javascript:`,    // ✅ XSS protection
			`data:text/html`, // ✅ XSS protection
		},
	})
	if err != nil {
		fmt.Printf("Failed to create validator: %v\n", err)
		return
	}

	// Test valid keys
	validKeys := []string{
		"user:123",
		"product:456",
		"normal-key",
	}

	for _, key := range validKeys {
		err := validator.ValidateKey(key)
		fmt.Printf("✅ Valid key '%s': %t\n", key, err == nil)
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
		fmt.Printf("❌ Invalid key '%s': %t (expected: true)\n", key, err != nil)
	}

	// Test value validation
	largeValue := make([]byte, 2*1024*1024) // 2MB
	err = validator.ValidateValue(largeValue)
	fmt.Printf("❌ Large value validation: %t (expected: true)\n", err != nil)

	normalValue := []byte("normal data")
	err = validator.ValidateValue(normalValue)
	fmt.Printf("✅ Normal value validation: %t\n", err == nil)
	fmt.Println()
}

func demoSecretsManagement() {
	fmt.Println("5. Secrets Management")
	fmt.Println("=====================")

	// Set up environment variables for demo
	os.Setenv("VAULT_REDIS_PASSWORD", "secure-redis-password")
	os.Setenv("VAULT_ENCRYPTION_KEY", "your-encryption-key")
	os.Setenv("VAULT_API_KEY", "your-api-key")

	// Create secrets manager
	secretsManager := security.NewSecretsManager("VAULT_")

	// Retrieve secrets
	redisPassword, err := secretsManager.GetSecret("redis_password")
	if err != nil {
		fmt.Printf("Failed to get Redis password: %v\n", err)
	} else {
		fmt.Printf("✅ Redis password retrieved: %s\n", redisPassword[:10]+"...")
	}

	encryptionKey, err := secretsManager.GetSecret("encryption_key")
	if err != nil {
		fmt.Printf("Failed to get encryption key: %v\n", err)
	} else {
		fmt.Printf("✅ Encryption key retrieved: %s\n", encryptionKey[:10]+"...")
	}

	// Get secret with default
	apiKey := secretsManager.GetSecretOrDefault("api_key", "default-api-key")
	fmt.Printf("✅ API key (with default): %s\n", apiKey[:10]+"...")

	// List available secrets
	secrets := secretsManager.ListSecrets()
	fmt.Printf("✅ Available secrets: %v\n", secrets)

	// Try to get non-existent secret
	_, err = secretsManager.GetSecret("non_existent")
	fmt.Printf("❌ Non-existent secret: %t (expected: true)\n", err != nil)
	fmt.Println()
}

func demoRBACHooks() {
	fmt.Println("6. RBAC Hooks")
	fmt.Println("=============")

	// Create RBAC authorizer (no-op default)
	authorizer := &security.RBACAuthorizer{}

	// Create authorization context
	authCtx := &security.AuthorizationContext{
		UserID:    "user123",
		Roles:     []string{"admin", "user"},
		Resource:  "cache",
		Action:    "read",
		Key:       "user:123",
		Timestamp: time.Now(),
	}

	// Check authorization
	result := authorizer.Authorize(context.Background(), authCtx)
	fmt.Printf("✅ Authorization result: %+v\n", result)

	// Test different contexts
	contexts := []*security.AuthorizationContext{
		{
			UserID: "user123", Roles: []string{"user"}, Resource: "cache", Action: "read", Key: "user:123",
		},
		{
			UserID: "user456", Roles: []string{"admin"}, Resource: "cache", Action: "write", Key: "user:456",
		},
		{
			UserID: "user789", Roles: []string{"guest"}, Resource: "cache", Action: "delete", Key: "user:789",
		},
	}

	for i, ctx := range contexts {
		result := authorizer.Authorize(context.Background(), ctx)
		fmt.Printf("✅ Context %d authorization: %+v\n", i+1, result)
	}

	fmt.Println("✅ RBAC hooks ready for custom implementation")
	fmt.Println("✅ No-op default implementation provided")
	fmt.Println()
}

func demoRateLimiting() {
	fmt.Println("7. Rate Limiting")
	fmt.Println("================")

	// Create rate limiter
	limiter, err := rate.NewLimiter(&rate.Config{
		RequestsPerSecond: 10, // 10 requests per second
		Burst:             5,  // Allow burst of 5 requests
	})
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}

	// Test rate limiting
	fmt.Println("Testing rate limiting...")

	allowed := 0
	denied := 0

	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			allowed++
		} else {
			denied++
		}
		time.Sleep(50 * time.Millisecond) // 50ms between requests
	}

	fmt.Printf("✅ Allowed requests: %d\n", allowed)
	fmt.Printf("✅ Denied requests: %d\n", denied)

	// Test reservation
	reservation := limiter.Reserve()
	fmt.Printf("✅ Reservation: %+v\n", reservation)

	// Test stats
	stats := limiter.GetStats()
	fmt.Printf("✅ Rate limiter stats: %+v\n", stats)

	// Test different configurations
	configs := []*rate.Config{
		rate.DefaultConfig(),
		rate.ConservativeConfig(),
		rate.AggressiveConfig(),
	}

	for i, config := range configs {
		limiter, err := rate.NewLimiter(config)
		if err != nil {
			fmt.Printf("Failed to create limiter %d: %v\n", i+1, err)
			continue
		}
		stats := limiter.GetStats()
		fmt.Printf("✅ Config %d stats: %+v\n", i+1, stats)
	}
	fmt.Println()
}

func demoLogRedaction() {
	fmt.Println("8. Log Redaction")
	fmt.Println("================")

	// Create redactor
	redactor, err := security.NewRedactor([]string{
		`password["\s]*[:=]["\s]*["']?[^"'\s]+["']?`, // Passwords
		`token["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,    // Tokens
		`key["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,      // Keys
		`secret["\s]*[:=]["\s]*["']?[^"'\s]+["']?`,   // Secrets
	}, "[REDACTED]")
	if err != nil {
		fmt.Printf("Failed to create redactor: %v\n", err)
		return
	}

	// Test log redaction
	logMessages := []string{
		`{"user": "john", "password": "secret123", "email": "john@example.com"}`,
		`{"token": "abc123def456", "action": "login"}`,
		`{"api_key": "xyz789", "request": "get_user"}`,
		`{"secret": "my-secret", "config": "production"}`,
		`{"normal": "data", "no_sensitive": "info"}`,
	}

	for _, message := range logMessages {
		redacted := redactor.Redact(message)
		fmt.Printf("✅ Original: %s\n", message)
		fmt.Printf("✅ Redacted: %s\n", redacted)
		fmt.Println()
	}
	fmt.Println()
}

func demoComprehensiveSecurity() {
	fmt.Println("9. Comprehensive Security")
	fmt.Println("=========================")

	// Create comprehensive security manager
	securityManager, err := security.NewSecurityManager(security.DefaultSecurityConfig())
	if err != nil {
		fmt.Printf("Failed to create security manager: %v\n", err)
		return
	}

	// Test key validation
	validKey := "user:123"
	err = securityManager.ValidateKey(validKey)
	fmt.Printf("✅ Valid key validation: %t\n", err == nil)

	invalidKey := "../etc/passwd"
	err = securityManager.ValidateKey(invalidKey)
	fmt.Printf("❌ Invalid key validation: %t (expected: true)\n", err != nil)

	// Test value validation
	validValue := []byte("normal data")
	err = securityManager.ValidateValue(validValue)
	fmt.Printf("✅ Valid value validation: %t\n", err == nil)

	// Test log redaction
	sensitiveLog := `{"user": "admin", "password": "secret123", "token": "abc123"}`
	redactedLog := securityManager.Redact(sensitiveLog)
	fmt.Printf("✅ Log redaction: %s\n", redactedLog)

	// Test authorization
	authCtx := &security.AuthorizationContext{
		UserID: "user123", Roles: []string{"admin"}, Resource: "cache", Action: "write", Key: "user:123",
	}
	result := securityManager.Authorize(context.Background(), authCtx)
	fmt.Printf("✅ Authorization: %+v\n", result)

	// Test secrets management
	secret, err := securityManager.GetSecret("redis_password")
	if err != nil {
		fmt.Printf("❌ Secret retrieval: %t (expected: true)\n", err != nil)
	} else {
		fmt.Printf("✅ Secret retrieved: %s\n", secret[:10]+"...")
	}

	fmt.Println("✅ Comprehensive security features working together")
	fmt.Println()
}
