package key

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Builder implements the KeyBuilder interface with namespaced key generation
type Builder struct {
	appName string
	env     string
	secret  string
}

// NewBuilder creates a new key builder
func NewBuilder(appName, env string, secret string) *Builder {
	return &Builder{
		appName: appName,
		env:     env,
		secret:  secret,
	}
}

// Build creates a namespaced entity key
func (b *Builder) Build(entity, id string) string {
	return fmt.Sprintf("app:%s:env:%s:%s:%s", b.appName, b.env, entity, id)
}

// BuildList creates a list key with hashed filters
func (b *Builder) BuildList(entity string, filters map[string]any) string {
	if len(filters) == 0 {
		return fmt.Sprintf("app:%s:env:%s:list:%s:all", b.appName, b.env, entity)
	}

	// Sort filter keys for consistent hashing
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build filter string
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, filters[k]))
	}
	filterStr := strings.Join(parts, "&")

	// Hash the filter string
	hash := b.hash(filterStr)

	return fmt.Sprintf("app:%s:env:%s:list:%s:%s", b.appName, b.env, entity, hash)
}

// BuildComposite creates a composite key for related entities
func (b *Builder) BuildComposite(entityA, idA, entityB, idB string) string {
	return fmt.Sprintf("app:%s:env:%s:%s:%s:%s:%s", b.appName, b.env, entityA, idA, entityB, idB)
}

// BuildSession creates a session key
func (b *Builder) BuildSession(sid string) string {
	return fmt.Sprintf("app:%s:env:%s:session:%s", b.appName, b.env, sid)
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

// BuildFeatureFlag creates a feature flag key
func (b *Builder) BuildFeatureFlag(flag string) string {
	return b.Build("feature", flag)
}

// hash creates an HMAC-SHA256 hash of the input data
func (b *Builder) hash(data string) string {
	if b.secret == "" {
		// Fallback to simple hash if no secret provided
		h := sha256.New()
		h.Write([]byte(data))
		return hex.EncodeToString(h.Sum(nil))[:16]
	}

	h := hmac.New(sha256.New, []byte(b.secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// ParseKey parses a key and returns its components
func (b *Builder) ParseKey(key string) (entity, id string, err error) {
	parts := strings.Split(key, ":")
	if len(parts) < 6 {
		return "", "", fmt.Errorf("invalid key format: %s", key)
	}

	// Expected format: app:{appName}:env:{env}:{entity}:{id}
	if parts[0] != "app" || parts[2] != "env" {
		return "", "", fmt.Errorf("invalid key prefix: %s", key)
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
