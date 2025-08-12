package key

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("myapp", "prod", "secret123")

	assert.Equal(t, "myapp", builder.appName)
	assert.Equal(t, "prod", builder.env)
	assert.Equal(t, "secret123", builder.secret)
}

func TestBuild_NamespacedKeys(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test namespaced entity key
	key := builder.Build("user", "12345")
	expected := "app:seasbee:env:prod:user:12345"
	assert.Equal(t, expected, key)

	// Test with different entity and ID
	key = builder.Build("product", "67890")
	expected = "app:seasbee:env:prod:product:67890"
	assert.Equal(t, expected, key)
}

func TestBuildList_WithFilters(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test list key with filters
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"org_id": "123",
	}

	key := builder.BuildList("users", filters)

	// Should start with the expected prefix
	expectedPrefix := "app:seasbee:env:prod:list:users:"
	assert.True(t, len(key) > len(expectedPrefix))
	assert.True(t, key[:len(expectedPrefix)] == expectedPrefix)

	// Should end with a hash
	hashPart := key[len(expectedPrefix):]
	assert.Equal(t, 16, len(hashPart)) // HMAC-SHA256 truncated to 16 chars
}

func TestBuildList_WithoutFilters(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test list key without filters
	key := builder.BuildList("users", nil)
	expected := "app:seasbee:env:prod:list:users:all"
	assert.Equal(t, expected, key)

	// Test with empty filters
	key = builder.BuildList("users", map[string]any{})
	expected = "app:seasbee:env:prod:list:users:all"
	assert.Equal(t, expected, key)
}

func TestBuildList_ConsistentHashing(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test that same filters produce same hash regardless of order
	filters1 := map[string]any{
		"status": "active",
		"role":   "admin",
	}

	filters2 := map[string]any{
		"role":   "admin",
		"status": "active",
	}

	key1 := builder.BuildList("users", filters1)
	key2 := builder.BuildList("users", filters2)

	assert.Equal(t, key1, key2, "Same filters should produce same key regardless of order")
}

func TestBuildComposite_JoinKeys(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test composite key for joins
	key := builder.BuildComposite("user", "123", "order", "456")
	expected := "app:seasbee:env:prod:user:123:order:456"
	assert.Equal(t, expected, key)

	// Test with different entities
	key = builder.BuildComposite("org", "789", "product", "101")
	expected = "app:seasbee:env:prod:org:789:product:101"
	assert.Equal(t, expected, key)
}

func TestBuildSession_SessionKeys(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test session key
	key := builder.BuildSession("abc123def456")
	expected := "app:seasbee:env:prod:session:abc123def456"
	assert.Equal(t, expected, key)

	// Test with different session ID
	key = builder.BuildSession("xyz789")
	expected = "app:seasbee:env:prod:session:xyz789"
	assert.Equal(t, expected, key)
}

func TestBuildUser_HelperMethod(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test user helper method
	key := builder.BuildUser("12345")
	expected := "app:seasbee:env:prod:user:12345"
	assert.Equal(t, expected, key)

	// Should be equivalent to Build("user", "12345")
	key2 := builder.Build("user", "12345")
	assert.Equal(t, key, key2)
}

func TestBuildOrg_HelperMethod(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test org helper method
	key := builder.BuildOrg("67890")
	expected := "app:seasbee:env:prod:org:67890"
	assert.Equal(t, expected, key)

	// Should be equivalent to Build("org", "67890")
	key2 := builder.Build("org", "67890")
	assert.Equal(t, key, key2)
}

func TestBuildProduct_HelperMethod(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test product helper method
	key := builder.BuildProduct("101112")
	expected := "app:seasbee:env:prod:product:101112"
	assert.Equal(t, expected, key)

	// Should be equivalent to Build("product", "101112")
	key2 := builder.Build("product", "101112")
	assert.Equal(t, key, key2)
}

func TestBuildOrder_HelperMethod(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test order helper method
	key := builder.BuildOrder("131415")
	expected := "app:seasbee:env:prod:order:131415"
	assert.Equal(t, expected, key)

	// Should be equivalent to Build("order", "131415")
	key2 := builder.Build("order", "131415")
	assert.Equal(t, key, key2)
}

func TestBuildFeatureFlag_HelperMethod(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test feature flag helper method
	key := builder.BuildFeatureFlag("dark_mode")
	expected := "app:seasbee:env:prod:feature:dark_mode"
	assert.Equal(t, expected, key)

	// Should be equivalent to Build("feature", "dark_mode")
	key2 := builder.Build("feature", "dark_mode")
	assert.Equal(t, key, key2)
}

func TestHash_HMAC_SHA256(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test HMAC-SHA256 hashing
	data := "status=active&role=admin"
	hash := builder.hash(data)

	// Should be 16 characters (truncated HMAC-SHA256)
	assert.Equal(t, 16, len(hash))

	// Should be consistent for same input
	hash2 := builder.hash(data)
	assert.Equal(t, hash, hash2)

	// Should be different for different input
	hash3 := builder.hash("different=data")
	assert.NotEqual(t, hash, hash3)
}

func TestHash_WithoutSecret(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "")

	// Test fallback to simple SHA256 when no secret provided
	data := "status=active&role=admin"
	hash := builder.hash(data)

	// Should still be 16 characters
	assert.Equal(t, 16, len(hash))

	// Should be consistent for same input
	hash2 := builder.hash(data)
	assert.Equal(t, hash, hash2)
}

func TestParseKey_ValidKeys(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test parsing valid keys
	key := "app:seasbee:env:prod:user:12345"
	entity, id, err := builder.ParseKey(key)
	require.NoError(t, err)
	assert.Equal(t, "user", entity)
	assert.Equal(t, "12345", id)

	// Test parsing composite key
	key = "app:seasbee:env:prod:user:123:order:456"
	entity, id, err = builder.ParseKey(key)
	require.NoError(t, err)
	assert.Equal(t, "user", entity)
	assert.Equal(t, "123:order:456", id) // Handles IDs with colons
}

func TestParseKey_InvalidKeys(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test parsing invalid keys
	invalidKeys := []string{
		"invalid:key",
		"app:seasbee:user:123",            // Missing env
		"app:seasbee:env:prod",            // Missing entity and id
		"wrong:seasbee:env:prod:user:123", // Wrong prefix
	}

	for _, key := range invalidKeys {
		_, _, err := builder.ParseKey(key)
		assert.Error(t, err, "Should fail for key: %s", key)
	}
}

func TestIsValidKey(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Test valid keys
	validKeys := []string{
		"app:seasbee:env:prod:user:12345",
		"app:seasbee:env:prod:product:67890",
		"app:seasbee:env:prod:user:123:order:456",
	}

	for _, key := range validKeys {
		assert.True(t, builder.IsValidKey(key), "Should be valid: %s", key)
	}

	// Test invalid keys
	invalidKeys := []string{
		"invalid:key",
		"app:seasbee:user:123",
		"app:seasbee:env:prod",
	}

	for _, key := range invalidKeys {
		assert.False(t, builder.IsValidKey(key), "Should be invalid: %s", key)
	}
}

func TestKeyGeneration_CompleteExample(t *testing.T) {
	builder := NewBuilder("seasbee", "prod", "secret123")

	// Demonstrate all key generation patterns

	// 1. Namespaced entity keys
	userKey := builder.Build("user", "12345")
	assert.Equal(t, "app:seasbee:env:prod:user:12345", userKey)

	// 2. List keys with filter hashing
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
	}
	listKey := builder.BuildList("users", filters)
	assert.True(t, len(listKey) > 0)
	assert.Contains(t, listKey, "app:seasbee:env:prod:list:users:")

	// 3. Composite keys for joins
	compositeKey := builder.BuildComposite("user", "123", "order", "456")
	assert.Equal(t, "app:seasbee:env:prod:user:123:order:456", compositeKey)

	// 4. Session keys
	sessionKey := builder.BuildSession("abc123def456")
	assert.Equal(t, "app:seasbee:env:prod:session:abc123def456", sessionKey)

	// 5. Helper methods
	userKey2 := builder.BuildUser("12345")
	orgKey := builder.BuildOrg("67890")
	productKey := builder.BuildProduct("101112")
	orderKey := builder.BuildOrder("131415")
	featureKey := builder.BuildFeatureFlag("dark_mode")

	assert.Equal(t, userKey, userKey2)
	assert.Equal(t, "app:seasbee:env:prod:org:67890", orgKey)
	assert.Equal(t, "app:seasbee:env:prod:product:101112", productKey)
	assert.Equal(t, "app:seasbee:env:prod:order:131415", orderKey)
	assert.Equal(t, "app:seasbee:env:prod:feature:dark_mode", featureKey)
}

func TestKeyGeneration_DifferentEnvironments(t *testing.T) {
	// Test key generation across different environments

	devBuilder := NewBuilder("seasbee", "dev", "secret123")
	prodBuilder := NewBuilder("seasbee", "prod", "secret123")

	// Same entity should have different keys in different environments
	devKey := devBuilder.Build("user", "12345")
	prodKey := prodBuilder.Build("user", "12345")

	assert.NotEqual(t, devKey, prodKey)
	assert.Equal(t, "app:seasbee:env:dev:user:12345", devKey)
	assert.Equal(t, "app:seasbee:env:prod:user:12345", prodKey)
}

func TestKeyGeneration_DifferentApps(t *testing.T) {
	// Test key generation across different applications

	app1Builder := NewBuilder("app1", "prod", "secret123")
	app2Builder := NewBuilder("app2", "prod", "secret123")

	// Same entity should have different keys in different apps
	app1Key := app1Builder.Build("user", "12345")
	app2Key := app2Builder.Build("user", "12345")

	assert.NotEqual(t, app1Key, app2Key)
	assert.Equal(t, "app:app1:env:prod:user:12345", app1Key)
	assert.Equal(t, "app:app2:env:prod:user:12345", app2Key)
}
