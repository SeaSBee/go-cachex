package cachex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Common errors
var (
	ErrEmptyAppName     = errors.New("app name cannot be empty")
	ErrEmptyEnv         = errors.New("environment cannot be empty")
	ErrEmptySecret      = errors.New("secret cannot be empty")
	ErrEmptyEntity      = errors.New("entity cannot be empty")
	ErrEmptyID          = errors.New("id cannot be empty")
	ErrEmptySessionID   = errors.New("session id cannot be empty")
	ErrInvalidKeyFormat = errors.New("invalid key format")
	ErrInvalidKeyPrefix = errors.New("invalid key prefix")
	ErrHashFailed       = errors.New("hash operation failed")
	ErrBuilderNotValid  = errors.New("builder is not valid")
)

// Builder implements the KeyBuilder interface with namespaced key generation
// Note: This struct is not thread-safe. If concurrent access is needed,
// external synchronization should be used.
type Builder struct {
	appName string
	env     string
	secret  string
}

// NewBuilder creates a new key builder with validation
func NewBuilder(appName, env string, secret string) (*Builder, error) {
	if strings.TrimSpace(appName) == "" {
		return nil, ErrEmptyAppName
	}
	if strings.TrimSpace(env) == "" {
		return nil, ErrEmptyEnv
	}
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptySecret
	}

	return &Builder{
		appName: strings.TrimSpace(appName),
		env:     strings.TrimSpace(env),
		secret:  strings.TrimSpace(secret),
	}, nil
}

// normalizeInput normalizes and validates input strings, returning a default value if empty
func (b *Builder) normalizeInput(input, defaultValue string) string {
	if b == nil {
		return defaultValue
	}
	if trimmed := strings.TrimSpace(input); trimmed != "" {
		return trimmed
	}
	return defaultValue
}

// Build creates a namespaced entity key
// Implements KeyBuilder.Build(entity, id string) string
func (b *Builder) Build(entity, id string) string {
	if b == nil {
		return fmt.Sprintf("app:unknown:env:unknown:%s:%s",
			b.normalizeInput(entity, "unknown"),
			b.normalizeInput(id, "unknown"))
	}

	entityStr := b.normalizeInput(entity, "unknown")
	idStr := b.normalizeInput(id, "unknown")

	return fmt.Sprintf("app:%s:env:%s:%s:%s", b.appName, b.env, entityStr, idStr)
}

// BuildList creates a list key with hashed filters
// Implements KeyBuilder.BuildList(entity string, filters map[string]any) string
func (b *Builder) BuildList(entity string, filters map[string]any) string {
	if b == nil {
		return fmt.Sprintf("app:unknown:env:unknown:list:%s:all",
			b.normalizeInput(entity, "unknown"))
	}

	entityStr := b.normalizeInput(entity, "unknown")

	// Handle empty filters
	if len(filters) == 0 {
		return fmt.Sprintf("app:%s:env:%s:list:%s:all", b.appName, b.env, entityStr)
	}

	// Optimize by using a single pass through filters
	var keys []string
	var parts []string

	// Pre-allocate with estimated capacity
	keys = make([]string, 0, len(filters))
	parts = make([]string, 0, len(filters))

	// Single pass: collect valid keys and build parts simultaneously
	for k, v := range filters {
		if trimmedKey := strings.TrimSpace(k); trimmedKey != "" {
			keys = append(keys, trimmedKey)
			parts = append(parts, fmt.Sprintf("%s=%v", trimmedKey, v))
		}
	}

	// Sort filter keys for consistent hashing
	sort.Strings(keys)

	// Rebuild parts in sorted order
	if len(keys) > 0 {
		parts = make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", k, filters[k]))
		}
	}

	filterStr := strings.Join(parts, "&")

	// Hash the filter string with proper error handling
	hash, err := b.hash(filterStr)
	if err != nil {
		// Log the error for debugging while still providing a fallback
		// In a production environment, you might want to use a proper logger here
		hash = "default"
	}

	return fmt.Sprintf("app:%s:env:%s:list:%s:%s", b.appName, b.env, entityStr, hash)
}

// BuildComposite creates a composite key for related entities
// Implements KeyBuilder.BuildComposite(entityA, idA, entityB, idB string) string
func (b *Builder) BuildComposite(entityA, idA, entityB, idB string) string {
	if b == nil {
		return fmt.Sprintf("app:unknown:env:unknown:%s:%s:%s:%s",
			b.normalizeInput(entityA, "unknown"),
			b.normalizeInput(idA, "unknown"),
			b.normalizeInput(entityB, "unknown"),
			b.normalizeInput(idB, "unknown"))
	}

	entityAStr := b.normalizeInput(entityA, "unknown")
	idAStr := b.normalizeInput(idA, "unknown")
	entityBStr := b.normalizeInput(entityB, "unknown")
	idBStr := b.normalizeInput(idB, "unknown")

	return fmt.Sprintf("app:%s:env:%s:%s:%s:%s:%s", b.appName, b.env, entityAStr, idAStr, entityBStr, idBStr)
}

// BuildSession creates a session key
// Implements KeyBuilder.BuildSession(sid string) string
func (b *Builder) BuildSession(sid string) string {
	if b == nil {
		return fmt.Sprintf("app:unknown:env:unknown:session:%s",
			b.normalizeInput(sid, "unknown"))
	}

	sidStr := b.normalizeInput(sid, "unknown")

	return fmt.Sprintf("app:%s:env:%s:session:%s", b.appName, b.env, sidStr)
}

// BuildUser creates a user key
func (b *Builder) BuildUser(userID string) string {
	return b.Build("user", userID)
}

// BuildOrg creates an organization key
func (b *Builder) BuildOrg(orgID string) string {
	return b.Build("org", orgID)
}

// BuildProduct creates a product key
func (b *Builder) BuildProduct(productID string) string {
	return b.Build("product", productID)
}

// BuildOrder creates an order key
func (b *Builder) BuildOrder(orderID string) string {
	return b.Build("order", orderID)
}

// hash creates an HMAC-SHA256 hash of the input data
// Returns the hash and any error that occurred during hashing
// Note: The hash is truncated to 16 characters for shorter keys, which reduces
// collision resistance but maintains a reasonable key length for caching systems.
// This is a deliberate trade-off between key length and collision resistance.
// For typical caching use cases, 16 characters provide sufficient uniqueness
// while keeping keys manageable for storage systems.
func (b *Builder) hash(data string) (string, error) {
	if b == nil {
		return "", fmt.Errorf("%w: builder is nil", ErrHashFailed)
	}

	if strings.TrimSpace(data) == "" {
		return "", fmt.Errorf("%w: empty data provided", ErrHashFailed)
	}

	// Validate secret is not empty
	if b.secret == "" {
		return "", fmt.Errorf("%w: secret is empty", ErrHashFailed)
	}

	h := hmac.New(sha256.New, []byte(b.secret))
	h.Write([]byte(data))
	hashBytes := h.Sum(nil)

	// SHA-256 hash encoded as hex is always 64 characters
	hashHex := hex.EncodeToString(hashBytes)

	// Return first 16 characters for shorter keys
	// SHA-256 hex is always 64 chars, so this is safe
	return hashHex[:16], nil
}

// ParseKey parses a key and returns its components
func (b *Builder) ParseKey(key string) (entity, id string, err error) {
	if b == nil {
		return "", "", fmt.Errorf("%w: builder is nil", ErrInvalidKeyFormat)
	}

	if strings.TrimSpace(key) == "" {
		return "", "", fmt.Errorf("%w: key is empty", ErrInvalidKeyFormat)
	}

	parts := strings.Split(key, ":")
	if len(parts) < 6 {
		return "", "", fmt.Errorf("%w: expected at least 6 parts, got %d in key '%s'", ErrInvalidKeyFormat, len(parts), key)
	}

	// Check for empty parts
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", "", fmt.Errorf("%w: part %d is empty in key '%s'", ErrInvalidKeyFormat, i, key)
		}
	}

	// Expected format: app:{appName}:env:{env}:{entity}:{id}
	if parts[0] != "app" || parts[2] != "env" {
		return "", "", fmt.Errorf("%w: expected 'app' and 'env' prefixes in key '%s'", ErrInvalidKeyPrefix, key)
	}

	// Validate that this key belongs to this builder instance
	if parts[1] != b.appName || parts[3] != b.env {
		return "", "", fmt.Errorf("%w: key does not belong to this builder instance (app: %s, env: %s)",
			ErrInvalidKeyFormat, parts[1], parts[3])
	}

	entity = parts[4]
	id = strings.Join(parts[5:], ":") // Handle IDs with colons

	return entity, id, nil
}

// IsValidKey checks if a key follows the expected format
func (b *Builder) IsValidKey(key string) bool {
	_, _, err := b.ParseKey(key)
	return err == nil
}

// Validate checks if the Builder's internal state is valid
func (b *Builder) Validate() error {
	if b == nil {
		return ErrBuilderNotValid
	}

	// Since NewBuilder already validates and trims, we only need to check if fields are empty
	// This could happen if someone manually constructs a Builder or modifies fields after creation
	if b.appName == "" {
		return ErrEmptyAppName
	}
	if b.env == "" {
		return ErrEmptyEnv
	}
	if b.secret == "" {
		return ErrEmptySecret
	}
	return nil
}

// GetConfig returns the Builder's configuration (without the secret for security)
func (b *Builder) GetConfig() (appName, env string) {
	if b == nil {
		return "", ""
	}
	return b.appName, b.env
}
