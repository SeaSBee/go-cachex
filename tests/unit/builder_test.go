package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/seasbee/go-cachex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		env         string
		secret      string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid configuration",
			appName:     "myapp",
			env:         "production",
			secret:      "secret123",
			expectError: false,
		},
		{
			name:        "valid configuration with whitespace",
			appName:     "  myapp  ",
			env:         "  production  ",
			secret:      "  secret123  ",
			expectError: false,
		},
		{
			name:        "empty app name",
			appName:     "",
			env:         "production",
			secret:      "secret123",
			expectError: true,
			errorType:   cachex.ErrEmptyAppName,
		},
		{
			name:        "whitespace only app name",
			appName:     "   ",
			env:         "production",
			secret:      "secret123",
			expectError: true,
			errorType:   cachex.ErrEmptyAppName,
		},
		{
			name:        "empty environment",
			appName:     "myapp",
			env:         "",
			secret:      "secret123",
			expectError: true,
			errorType:   cachex.ErrEmptyEnv,
		},
		{
			name:        "whitespace only environment",
			appName:     "myapp",
			env:         "   ",
			secret:      "secret123",
			expectError: true,
			errorType:   cachex.ErrEmptyEnv,
		},
		{
			name:        "empty secret",
			appName:     "myapp",
			env:         "production",
			secret:      "",
			expectError: true,
			errorType:   cachex.ErrEmptySecret,
		},
		{
			name:        "whitespace only secret",
			appName:     "myapp",
			env:         "production",
			secret:      "   ",
			expectError: true,
			errorType:   cachex.ErrEmptySecret,
		},
		{
			name:        "all empty",
			appName:     "",
			env:         "",
			secret:      "",
			expectError: true,
			errorType:   cachex.ErrEmptyAppName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := cachex.NewBuilder(tt.appName, tt.env, tt.secret)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, builder)
				assert.ErrorIs(t, err, tt.errorType)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, builder)
				appName, env := builder.GetConfig()
				assert.Equal(t, "myapp", appName)
				assert.Equal(t, "production", env)
			}
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		entity   string
		id       string
		expected string
	}{
		{
			name:     "basic build",
			entity:   "user",
			id:       "123",
			expected: "app:myapp:env:production:user:123",
		},
		{
			name:     "entity with whitespace",
			entity:   "  user  ",
			id:       "123",
			expected: "app:myapp:env:production:user:123",
		},
		{
			name:     "id with whitespace",
			entity:   "user",
			id:       "  123  ",
			expected: "app:myapp:env:production:user:123",
		},
		{
			name:     "empty entity",
			entity:   "",
			id:       "123",
			expected: "app:myapp:env:production:unknown:123",
		},
		{
			name:     "empty id",
			entity:   "user",
			id:       "",
			expected: "app:myapp:env:production:user:unknown",
		},
		{
			name:     "both empty",
			entity:   "",
			id:       "",
			expected: "app:myapp:env:production:unknown:unknown",
		},
		{
			name:     "id with colons",
			entity:   "user",
			id:       "123:456:789",
			expected: "app:myapp:env:production:user:123:456:789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.Build(tt.entity, tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_BuildList(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		entity   string
		filters  map[string]any
		expected string
	}{
		{
			name:     "no filters",
			entity:   "user",
			filters:  nil,
			expected: "app:myapp:env:production:list:user:all",
		},
		{
			name:     "empty filters",
			entity:   "user",
			filters:  map[string]any{},
			expected: "app:myapp:env:production:list:user:all",
		},
		{
			name:   "single filter",
			entity: "user",
			filters: map[string]any{
				"status": "active",
			},
			expected: "app:myapp:env:production:list:user:",
		},
		{
			name:   "multiple filters",
			entity: "user",
			filters: map[string]any{
				"status": "active",
				"role":   "admin",
			},
			expected: "app:myapp:env:production:list:user:",
		},
		{
			name:   "filters with empty keys",
			entity: "user",
			filters: map[string]any{
				"status": "active",
				"":       "invalid",
				"role":   "admin",
			},
			expected: "app:myapp:env:production:list:user:",
		},
		{
			name:   "filters with whitespace keys",
			entity: "user",
			filters: map[string]any{
				"status": "active",
				"  ":     "invalid",
				"role":   "admin",
			},
			expected: "app:myapp:env:production:list:user:",
		},
		{
			name:     "entity with whitespace",
			entity:   "  user  ",
			filters:  map[string]any{"status": "active"},
			expected: "app:myapp:env:production:list:user:",
		},
		{
			name:     "empty entity",
			entity:   "",
			filters:  map[string]any{"status": "active"},
			expected: "app:myapp:env:production:list:unknown:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildList(tt.entity, tt.filters)
			// For tests with filters, we can't predict the exact hash, so we check the prefix
			if len(tt.filters) > 0 {
				assert.True(t, len(result) > len(tt.expected))
				assert.True(t, strings.HasPrefix(result, tt.expected))
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuilder_BuildComposite(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		entityA  string
		idA      string
		entityB  string
		idB      string
		expected string
	}{
		{
			name:     "basic composite",
			entityA:  "user",
			idA:      "123",
			entityB:  "org",
			idB:      "456",
			expected: "app:myapp:env:production:user:123:org:456",
		},
		{
			name:     "with whitespace",
			entityA:  "  user  ",
			idA:      "  123  ",
			entityB:  "  org  ",
			idB:      "  456  ",
			expected: "app:myapp:env:production:user:123:org:456",
		},
		{
			name:     "empty entityA",
			entityA:  "",
			idA:      "123",
			entityB:  "org",
			idB:      "456",
			expected: "app:myapp:env:production:unknown:123:org:456",
		},
		{
			name:     "empty idA",
			entityA:  "user",
			idA:      "",
			entityB:  "org",
			idB:      "456",
			expected: "app:myapp:env:production:user:unknown:org:456",
		},
		{
			name:     "ids with colons",
			entityA:  "user",
			idA:      "123:456",
			entityB:  "org",
			idB:      "789:012",
			expected: "app:myapp:env:production:user:123:456:org:789:012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildComposite(tt.entityA, tt.idA, tt.entityB, tt.idB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_BuildSession(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		sid      string
		expected string
	}{
		{
			name:     "basic session",
			sid:      "session123",
			expected: "app:myapp:env:production:session:session123",
		},
		{
			name:     "session with whitespace",
			sid:      "  session123  ",
			expected: "app:myapp:env:production:session:session123",
		},
		{
			name:     "empty session id",
			sid:      "",
			expected: "app:myapp:env:production:session:unknown",
		},
		{
			name:     "session id with colons",
			sid:      "session:123:456",
			expected: "app:myapp:env:production:session:session:123:456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildSession(tt.sid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_ConvenienceMethods(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		method   func(string) string
		id       string
		expected string
	}{
		{
			name:     "BuildUser",
			method:   builder.BuildUser,
			id:       "123",
			expected: "app:myapp:env:production:user:123",
		},
		{
			name:     "BuildOrg",
			method:   builder.BuildOrg,
			id:       "456",
			expected: "app:myapp:env:production:org:456",
		},
		{
			name:     "BuildProduct",
			method:   builder.BuildProduct,
			id:       "789",
			expected: "app:myapp:env:production:product:789",
		},
		{
			name:     "BuildOrder",
			method:   builder.BuildOrder,
			id:       "012",
			expected: "app:myapp:env:production:order:012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_HashFunctionality(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	t.Run("hash consistency through BuildList", func(t *testing.T) {
		filters := map[string]any{
			"status": "active",
			"role":   "admin",
		}

		// Test that the same filters produce the same hash
		result1 := builder.BuildList("user", filters)
		result2 := builder.BuildList("user", filters)

		assert.Equal(t, result1, result2)
		assert.True(t, strings.HasPrefix(result1, "app:myapp:env:production:list:user:"))

		// Extract the hash part (last 16 characters)
		hashPart := result1[len("app:myapp:env:production:list:user:"):]
		assert.Len(t, hashPart, 16)
	})

	t.Run("hash with empty filters", func(t *testing.T) {
		result := builder.BuildList("user", map[string]any{})
		assert.Equal(t, "app:myapp:env:production:list:user:all", result)
	})

	t.Run("hash with nil filters", func(t *testing.T) {
		result := builder.BuildList("user", nil)
		assert.Equal(t, "app:myapp:env:production:list:user:all", result)
	})

	t.Run("hash with different filter orders", func(t *testing.T) {
		filters1 := map[string]any{
			"status": "active",
			"role":   "admin",
		}

		filters2 := map[string]any{
			"role":   "admin",
			"status": "active",
		}

		result1 := builder.BuildList("user", filters1)
		result2 := builder.BuildList("user", filters2)

		// Results should be identical due to sorted keys
		assert.Equal(t, result1, result2)
	})
}

func TestBuilder_ParseKey(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name           string
		key            string
		expectError    bool
		errorType      error
		expectedEntity string
		expectedID     string
	}{
		{
			name:           "valid key",
			key:            "app:myapp:env:production:user:123",
			expectError:    false,
			expectedEntity: "user",
			expectedID:     "123",
		},
		{
			name:           "key with colons in id",
			key:            "app:myapp:env:production:user:123:456:789",
			expectError:    false,
			expectedEntity: "user",
			expectedID:     "123:456:789",
		},
		{
			name:        "empty key",
			key:         "",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "whitespace key",
			key:         "   ",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "too few parts",
			key:         "app:myapp:env:production:user",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "wrong app name",
			key:         "app:otherapp:env:production:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "wrong environment",
			key:         "app:myapp:env:development:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "missing app prefix",
			key:         "myapp:env:production:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "missing env prefix",
			key:         "app:myapp:production:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "empty part",
			key:         "app::env:production:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
		{
			name:        "nil builder",
			key:         "app:myapp:env:production:user:123",
			expectError: true,
			errorType:   cachex.ErrInvalidKeyFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testBuilder *cachex.Builder
			if tt.name == "nil builder" {
				testBuilder = nil
			} else {
				testBuilder = builder
			}

			entity, id, err := testBuilder.ParseKey(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, entity)
				assert.Empty(t, id)
				assert.ErrorIs(t, err, tt.errorType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEntity, entity)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestBuilder_IsValidKey(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid key",
			key:      "app:myapp:env:production:user:123",
			expected: true,
		},
		{
			name:     "key with colons in id",
			key:      "app:myapp:env:production:user:123:456:789",
			expected: true,
		},
		{
			name:     "invalid key - wrong app",
			key:      "app:otherapp:env:production:user:123",
			expected: false,
		},
		{
			name:     "invalid key - wrong env",
			key:      "app:myapp:env:development:user:123",
			expected: false,
		},
		{
			name:     "invalid key - too few parts",
			key:      "app:myapp:env:production:user",
			expected: false,
		},
		{
			name:     "invalid key - empty",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.IsValidKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuilder_Validate(t *testing.T) {
	tests := []struct {
		name        string
		builder     *cachex.Builder
		expectError bool
		errorType   error
	}{
		{
			name:        "valid builder",
			builder:     func() *cachex.Builder { b, _ := cachex.NewBuilder("myapp", "production", "secret123"); return b }(),
			expectError: false,
		},
		{
			name:        "nil builder",
			builder:     nil,
			expectError: true,
			errorType:   cachex.ErrBuilderNotValid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.builder.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.errorType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuilder_GetConfig(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	appName, env := builder.GetConfig()
	assert.Equal(t, "myapp", appName)
	assert.Equal(t, "production", env)

	// Test with nil builder
	var nilBuilder *cachex.Builder
	appName, env = nilBuilder.GetConfig()
	assert.Empty(t, appName)
	assert.Empty(t, env)
}

func TestBuilder_NilHandling(t *testing.T) {
	var nilBuilder *cachex.Builder

	t.Run("Build with nil builder", func(t *testing.T) {
		result := nilBuilder.Build("user", "123")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown", result)
	})

	t.Run("BuildList with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildList("user", map[string]any{"status": "active"})
		assert.Equal(t, "app:unknown:env:unknown:list:unknown:all", result)
	})

	t.Run("BuildComposite with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildComposite("user", "123", "org", "456")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown:unknown:unknown", result)
	})

	t.Run("BuildSession with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildSession("session123")
		assert.Equal(t, "app:unknown:env:unknown:session:unknown", result)
	})

	t.Run("BuildUser with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildUser("123")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown", result)
	})

	t.Run("BuildOrg with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildOrg("456")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown", result)
	})

	t.Run("BuildProduct with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildProduct("789")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown", result)
	})

	t.Run("BuildOrder with nil builder", func(t *testing.T) {
		result := nilBuilder.BuildOrder("012")
		assert.Equal(t, "app:unknown:env:unknown:unknown:unknown", result)
	})
}

func TestBuilder_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	t.Run("BuildList with complex filters", func(t *testing.T) {
		filters := map[string]any{
			"status":    "active",
			"role":      "admin",
			"age":       25,
			"is_active": true,
			"tags":      []string{"premium", "vip"},
			"metadata":  map[string]string{"source": "web"},
		}

		result := builder.BuildList("user", filters)
		assert.True(t, strings.HasPrefix(result, "app:myapp:env:production:list:user:"))
		assert.Len(t, result, len("app:myapp:env:production:list:user:")+16) // 16 char hash
	})

	t.Run("BuildList with special characters in filters", func(t *testing.T) {
		filters := map[string]any{
			"name":    "John Doe",
			"email":   "john@example.com",
			"special": "value with spaces & symbols!",
			"unicode": "测试",
		}

		result := builder.BuildList("user", filters)
		assert.True(t, strings.HasPrefix(result, "app:myapp:env:production:list:user:"))
	})

	t.Run("BuildList with sorted filter keys", func(t *testing.T) {
		// Test that filter keys are sorted consistently
		filters1 := map[string]any{
			"zebra":  "last",
			"apple":  "first",
			"banana": "middle",
		}

		filters2 := map[string]any{
			"banana": "middle",
			"apple":  "first",
			"zebra":  "last",
		}

		result1 := builder.BuildList("user", filters1)
		result2 := builder.BuildList("user", filters2)

		// Results should be identical due to sorted keys
		assert.Equal(t, result1, result2)
	})

	t.Run("Build with very long strings", func(t *testing.T) {
		longEntity := strings.Repeat("a", 1000)
		longID := strings.Repeat("b", 1000)

		result := builder.Build(longEntity, longID)
		expected := fmt.Sprintf("app:myapp:env:production:%s:%s", longEntity, longID)
		assert.Equal(t, expected, result)
	})

	t.Run("BuildList with very long filter values", func(t *testing.T) {
		longValue := strings.Repeat("x", 1000)
		filters := map[string]any{
			"long_field": longValue,
		}

		result := builder.BuildList("user", filters)
		assert.True(t, strings.HasPrefix(result, "app:myapp:env:production:list:user:"))
	})
}

func TestBuilder_ErrorTypes(t *testing.T) {
	t.Run("verify error types are defined", func(t *testing.T) {
		assert.NotNil(t, cachex.ErrEmptyAppName)
		assert.NotNil(t, cachex.ErrEmptyEnv)
		assert.NotNil(t, cachex.ErrEmptySecret)
		assert.NotNil(t, cachex.ErrEmptyEntity)
		assert.NotNil(t, cachex.ErrEmptyID)
		assert.NotNil(t, cachex.ErrEmptySessionID)
		assert.NotNil(t, cachex.ErrInvalidKeyFormat)
		assert.NotNil(t, cachex.ErrInvalidKeyPrefix)
		assert.NotNil(t, cachex.ErrHashFailed)
		assert.NotNil(t, cachex.ErrBuilderNotValid)
	})

	t.Run("verify error messages", func(t *testing.T) {
		assert.Equal(t, "app name cannot be empty", cachex.ErrEmptyAppName.Error())
		assert.Equal(t, "environment cannot be empty", cachex.ErrEmptyEnv.Error())
		assert.Equal(t, "secret cannot be empty", cachex.ErrEmptySecret.Error())
		assert.Equal(t, "entity cannot be empty", cachex.ErrEmptyEntity.Error())
		assert.Equal(t, "id cannot be empty", cachex.ErrEmptyID.Error())
		assert.Equal(t, "session id cannot be empty", cachex.ErrEmptySessionID.Error())
		assert.Equal(t, "invalid key format", cachex.ErrInvalidKeyFormat.Error())
		assert.Equal(t, "invalid key prefix", cachex.ErrInvalidKeyPrefix.Error())
		assert.Equal(t, "hash operation failed", cachex.ErrHashFailed.Error())
		assert.Equal(t, "builder is not valid", cachex.ErrBuilderNotValid.Error())
	})
}

func TestBuilder_InterfaceCompliance(t *testing.T) {
	builder, err := cachex.NewBuilder("myapp", "production", "secret123")
	require.NoError(t, err)

	// Test that Builder implements KeyBuilder interface
	var keyBuilder cachex.KeyBuilder = builder
	assert.NotNil(t, keyBuilder)

	// Test interface methods
	result := keyBuilder.Build("user", "123")
	assert.Equal(t, "app:myapp:env:production:user:123", result)

	listResult := keyBuilder.BuildList("user", map[string]any{"status": "active"})
	assert.True(t, strings.HasPrefix(listResult, "app:myapp:env:production:list:user:"))

	compositeResult := keyBuilder.BuildComposite("user", "123", "org", "456")
	assert.Equal(t, "app:myapp:env:production:user:123:org:456", compositeResult)

	sessionResult := keyBuilder.BuildSession("session123")
	assert.Equal(t, "app:myapp:env:production:session:session123", sessionResult)
}

// Benchmark tests
func BenchmarkNewBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = cachex.NewBuilder("myapp", "production", "secret123")
	}
}

func BenchmarkBuilder_Build(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = builder.Build("user", "123")
	}
}

func BenchmarkBuilder_BuildList(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"age":    25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.BuildList("user", filters)
	}
}

func BenchmarkBuilder_BuildComposite(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = builder.BuildComposite("user", "123", "org", "456")
	}
}

func BenchmarkBuilder_BuildSession(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = builder.BuildSession("session123")
	}
}

func BenchmarkBuilder_HashThroughBuildList(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"age":    25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.BuildList("user", filters)
	}
}

func BenchmarkBuilder_ParseKey(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	key := "app:myapp:env:production:user:123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = builder.ParseKey(key)
	}
}

func BenchmarkBuilder_IsValidKey(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	key := "app:myapp:env:production:user:123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.IsValidKey(key)
	}
}

func BenchmarkBuilder_ConvenienceMethods(b *testing.B) {
	builder, _ := cachex.NewBuilder("myapp", "production", "secret123")
	b.ResetTimer()

	b.Run("BuildUser", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = builder.BuildUser("123")
		}
	})

	b.Run("BuildOrg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = builder.BuildOrg("456")
		}
	})

	b.Run("BuildProduct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = builder.BuildProduct("789")
		}
	})

	b.Run("BuildOrder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = builder.BuildOrder("012")
		}
	})
}
