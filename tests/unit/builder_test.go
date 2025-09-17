package unit

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seasbee/go-cachex"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		env         string
		secret      string
		expectError bool
	}{
		{
			name:        "basic builder",
			appName:     "testapp",
			env:         "prod",
			secret:      "secret123",
			expectError: false,
		},
		{
			name:        "empty secret",
			appName:     "testapp",
			env:         "dev",
			secret:      "",
			expectError: true,
		},
		{
			name:        "empty app name",
			appName:     "",
			env:         "test",
			secret:      "secret",
			expectError: true,
		},
		{
			name:        "empty env",
			appName:     "testapp",
			env:         "",
			secret:      "secret",
			expectError: true,
		},
		{
			name:        "whitespace only app name",
			appName:     "   ",
			env:         "test",
			secret:      "secret",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := cachex.NewBuilder(tt.appName, tt.env, tt.secret)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewBuilder() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewBuilder() unexpected error = %v", err)
			}

			// Test that the builder works by using it to build a key
			key := builder.Build("test", "123")
			expectedPrefix := fmt.Sprintf("app:%s:env:%s:test:123", tt.appName, tt.env)
			if key != expectedPrefix {
				t.Errorf("NewBuilder() produced key = %v, want %v", key, expectedPrefix)
			}
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name   string
		entity string
		id     string
		want   string
	}{
		{
			name:   "basic build",
			entity: "user",
			id:     "123",
			want:   "app:testapp:env:prod:user:123",
		},
		{
			name:   "empty entity",
			entity: "",
			id:     "123",
			want:   "app:testapp:env:prod:unknown:123",
		},
		{
			name:   "empty id",
			entity: "user",
			id:     "",
			want:   "app:testapp:env:prod:user:unknown",
		},
		{
			name:   "both empty",
			entity: "",
			id:     "",
			want:   "app:testapp:env:prod:unknown:unknown",
		},
		{
			name:   "id with colons",
			entity: "session",
			id:     "user:123:token:abc",
			want:   "app:testapp:env:prod:session:user:123:token:abc",
		},
		{
			name:   "special characters",
			entity: "user",
			id:     "user@example.com",
			want:   "app:testapp:env:prod:user:user@example.com",
		},
		{
			name:   "whitespace only entity",
			entity: "   ",
			id:     "123",
			want:   "app:testapp:env:prod:unknown:123",
		},
		{
			name:   "whitespace only id",
			entity: "user",
			id:     "   ",
			want:   "app:testapp:env:prod:user:unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.Build(tt.entity, tt.id)
			if got != tt.want {
				t.Errorf("Builder.Build() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_BuildList(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name    string
		entity  string
		filters map[string]any
		want    string
	}{
		{
			name:    "empty filters",
			entity:  "users",
			filters: map[string]any{},
			want:    "app:testapp:env:prod:list:users:all",
		},
		{
			name:    "nil filters",
			entity:  "users",
			filters: nil,
			want:    "app:testapp:env:prod:list:users:all",
		},
		{
			name:   "single filter",
			entity: "users",
			filters: map[string]any{
				"status": "active",
			},
			want: "app:testapp:env:prod:list:users:",
		},
		{
			name:   "multiple filters",
			entity: "users",
			filters: map[string]any{
				"status": "active",
				"role":   "admin",
			},
			want: "app:testapp:env:prod:list:users:",
		},
		{
			name:   "filters with different types",
			entity: "users",
			filters: map[string]any{
				"status": "active",
				"age":    25,
				"active": true,
			},
			want: "app:testapp:env:prod:list:users:",
		},
		{
			name:   "empty entity",
			entity: "",
			filters: map[string]any{
				"status": "active",
			},
			want: "app:testapp:env:prod:list:unknown:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildList(tt.entity, tt.filters)

			// For empty filters, we can check exact match
			if tt.filters == nil || len(tt.filters) == 0 {
				if got != tt.want {
					t.Errorf("Builder.BuildList() = %v, want %v", got, tt.want)
				}
				return
			}

			// For non-empty filters, check prefix and hash format
			expectedEntity := tt.entity
			if strings.TrimSpace(expectedEntity) == "" {
				expectedEntity = "unknown"
			}
			if !strings.HasPrefix(got, "app:testapp:env:prod:list:"+expectedEntity+":") {
				t.Errorf("Builder.BuildList() = %v, should start with app:testapp:env:prod:list:%s:", got, expectedEntity)
			}

			// Check that the hash part is 16 characters (hex)
			parts := strings.Split(got, ":")
			if len(parts) < 5 {
				t.Errorf("Builder.BuildList() = %v, should have at least 5 parts", got)
			}
			hash := parts[len(parts)-1]
			if len(hash) != 16 {
				t.Errorf("Builder.BuildList() hash = %v, should be 16 characters", hash)
			}
		})
	}
}

func TestBuilder_BuildList_Consistency(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	filters := map[string]any{
		"status": "active",
		"role":   "admin",
		"org_id": "123",
	}

	// Test consistency
	key1 := builder.BuildList("users", filters)
	key2 := builder.BuildList("users", filters)

	if key1 != key2 {
		t.Errorf("BuildList() not consistent: %v != %v", key1, key2)
	}

	// Test different order produces same result
	filters2 := map[string]any{
		"org_id": "123",
		"status": "active",
		"role":   "admin",
	}
	key3 := builder.BuildList("users", filters2)

	if key1 != key3 {
		t.Errorf("BuildList() not order-independent: %v != %v", key1, key3)
	}
}

func TestBuilder_BuildComposite(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name    string
		entityA string
		idA     string
		entityB string
		idB     string
		want    string
	}{
		{
			name:    "basic composite",
			entityA: "user",
			idA:     "123",
			entityB: "order",
			idB:     "456",
			want:    "app:testapp:env:prod:user:123:order:456",
		},
		{
			name:    "empty entities",
			entityA: "",
			idA:     "123",
			entityB: "",
			idB:     "456",
			want:    "app:testapp:env:prod:unknown:123:unknown:456",
		},
		{
			name:    "ids with colons",
			entityA: "user",
			idA:     "user:123:token:abc",
			entityB: "order",
			idB:     "order:456:status:pending",
			want:    "app:testapp:env:prod:user:user:123:token:abc:order:order:456:status:pending",
		},
		{
			name:    "empty ids",
			entityA: "user",
			idA:     "",
			entityB: "order",
			idB:     "",
			want:    "app:testapp:env:prod:user:unknown:order:unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildComposite(tt.entityA, tt.idA, tt.entityB, tt.idB)
			if got != tt.want {
				t.Errorf("Builder.BuildComposite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_BuildSession(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name string
		sid  string
		want string
	}{
		{
			name: "basic session",
			sid:  "session123",
			want: "app:testapp:env:prod:session:session123",
		},
		{
			name: "empty session id",
			sid:  "",
			want: "app:testapp:env:prod:session:unknown",
		},
		{
			name: "session id with special characters",
			sid:  "session@123#456",
			want: "app:testapp:env:prod:session:session@123#456",
		},
		{
			name: "whitespace only session id",
			sid:  "   ",
			want: "app:testapp:env:prod:session:unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildSession(tt.sid)
			if got != tt.want {
				t.Errorf("Builder.BuildSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_HelperMethods(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name   string
		method func(string) string
		input  string
		want   string
	}{
		{
			name:   "BuildUser",
			method: builder.BuildUser,
			input:  "123",
			want:   "app:testapp:env:prod:user:123",
		},
		{
			name:   "BuildOrg",
			method: builder.BuildOrg,
			input:  "456",
			want:   "app:testapp:env:prod:org:456",
		},
		{
			name:   "BuildProduct",
			method: builder.BuildProduct,
			input:  "789",
			want:   "app:testapp:env:prod:product:789",
		},
		{
			name:   "BuildOrder",
			method: builder.BuildOrder,
			input:  "101",
			want:   "app:testapp:env:prod:order:101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.method(tt.input)
			if got != tt.want {
				t.Errorf("%s() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestBuilder_Hash_Integration(t *testing.T) {
	// Test hash functionality through BuildList which uses the hash method internally
	tests := []struct {
		name   string
		secret string
		data   map[string]any
	}{
		{
			name:   "with secret",
			secret: "secret123",
			data: map[string]any{
				"status": "active",
				"role":   "admin",
			},
		},
		{
			name:   "with different secret",
			secret: "different_secret",
			data: map[string]any{
				"status": "active",
				"role":   "admin",
			},
		},
		{
			name:   "empty data",
			secret: "secret123",
			data:   map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := cachex.NewBuilder("testapp", "prod", tt.secret)
			if err != nil {
				t.Fatalf("NewBuilder() error = %v", err)
			}

			// Test consistency through BuildList
			key1 := builder.BuildList("users", tt.data)
			key2 := builder.BuildList("users", tt.data)

			if key1 != key2 {
				t.Errorf("BuildList() not consistent: %v != %v", key1, key2)
			}
		})
	}
}

func TestBuilder_Hash_DifferentSecrets_Integration(t *testing.T) {
	// Test that different secrets produce different hashes through BuildList
	data := map[string]any{
		"status": "active",
		"role":   "admin",
	}

	builder1, err1 := cachex.NewBuilder("testapp", "prod", "secret1")
	if err1 != nil {
		t.Fatalf("NewBuilder() error = %v", err1)
	}

	builder2, err2 := cachex.NewBuilder("testapp", "prod", "secret2")
	if err2 != nil {
		t.Fatalf("NewBuilder() error = %v", err2)
	}

	key1 := builder1.BuildList("users", data)
	key2 := builder2.BuildList("users", data)

	if key1 == key2 {
		t.Errorf("BuildList() should be different with different secrets: %v == %v", key1, key2)
	}
}

func TestBuilder_ParseKey(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name       string
		key        string
		wantEntity string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "valid key",
			key:        "app:testapp:env:prod:user:123",
			wantEntity: "user",
			wantID:     "123",
			wantErr:    false,
		},
		{
			name:       "key with colons in id",
			key:        "app:testapp:env:prod:user:123:token:abc",
			wantEntity: "user",
			wantID:     "123:token:abc",
			wantErr:    false,
		},
		{
			name:    "invalid key format - too few parts",
			key:     "app:testapp:env:prod:user",
			wantErr: true,
		},
		{
			name:    "invalid key prefix",
			key:     "invalid:testapp:env:prod:user:123",
			wantErr: true,
		},
		{
			name:    "invalid key prefix 2",
			key:     "app:testapp:invalid:prod:user:123",
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, id, err := builder.ParseKey(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseKey() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseKey() unexpected error: %v", err)
				return
			}

			if entity != tt.wantEntity {
				t.Errorf("ParseKey() entity = %v, want %v", entity, tt.wantEntity)
			}

			if id != tt.wantID {
				t.Errorf("ParseKey() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestBuilder_IsValidKey(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "valid key",
			key:  "app:testapp:env:prod:user:123",
			want: true,
		},
		{
			name: "valid key with colons in id",
			key:  "app:testapp:env:prod:user:123:token:abc",
			want: true,
		},
		{
			name: "invalid key - too few parts",
			key:  "app:testapp:env:prod:user",
			want: false,
		},
		{
			name: "invalid key - wrong prefix",
			key:  "invalid:testapp:env:prod:user:123",
			want: false,
		},
		{
			name: "invalid key - wrong env prefix",
			key:  "app:testapp:invalid:prod:user:123",
			want: false,
		},
		{
			name: "empty key",
			key:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.IsValidKey(tt.key)
			if got != tt.want {
				t.Errorf("IsValidKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_RoundTrip(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		entity string
		id     string
	}{
		{"user", "123"},
		{"product", "abc"},
		{"order", "456"},
		{"session", "user:123:token:abc"},
	}

	for _, tt := range tests {
		t.Run(tt.entity+":"+tt.id, func(t *testing.T) {
			// Build the key
			key := builder.Build(tt.entity, tt.id)

			// Parse it back
			parsedEntity, parsedID, err := builder.ParseKey(key)
			if err != nil {
				t.Errorf("ParseKey() failed: %v", err)
				return
			}

			// Verify round trip
			if parsedEntity != tt.entity {
				t.Errorf("ParseKey() entity = %v, want %v", parsedEntity, tt.entity)
			}

			if parsedID != tt.id {
				t.Errorf("ParseKey() id = %v, want %v", parsedID, tt.id)
			}

			// Verify it's valid
			if !builder.IsValidKey(key) {
				t.Errorf("IsValidKey() should return true for %v", key)
			}
		})
	}
}

func TestBuilder_EnvironmentIsolation(t *testing.T) {
	appName := "testapp"
	userID := "123"

	devBuilder, err1 := cachex.NewBuilder(appName, "dev", "secret")
	if err1 != nil {
		t.Fatalf("NewBuilder() error = %v", err1)
	}

	prodBuilder, err2 := cachex.NewBuilder(appName, "prod", "secret")
	if err2 != nil {
		t.Fatalf("NewBuilder() error = %v", err2)
	}

	devKey := devBuilder.BuildUser(userID)
	prodKey := prodBuilder.BuildUser(userID)

	if devKey == prodKey {
		t.Errorf("Keys should be different for different environments: %v == %v", devKey, prodKey)
	}

	expectedDev := "app:testapp:env:dev:user:123"
	expectedProd := "app:testapp:env:prod:user:123"

	if devKey != expectedDev {
		t.Errorf("Dev key = %v, want %v", devKey, expectedDev)
	}

	if prodKey != expectedProd {
		t.Errorf("Prod key = %v, want %v", prodKey, expectedProd)
	}
}

func TestBuilder_ApplicationIsolation(t *testing.T) {
	env := "prod"
	userID := "123"

	app1Builder, err1 := cachex.NewBuilder("app1", env, "secret")
	if err1 != nil {
		t.Fatalf("NewBuilder() error = %v", err1)
	}

	app2Builder, err2 := cachex.NewBuilder("app2", env, "secret")
	if err2 != nil {
		t.Fatalf("NewBuilder() error = %v", err2)
	}

	app1Key := app1Builder.BuildUser(userID)
	app2Key := app2Builder.BuildUser(userID)

	if app1Key == app2Key {
		t.Errorf("Keys should be different for different applications: %v == %v", app1Key, app2Key)
	}

	expectedApp1 := "app:app1:env:prod:user:123"
	expectedApp2 := "app:app2:env:prod:user:123"

	if app1Key != expectedApp1 {
		t.Errorf("App1 key = %v, want %v", app1Key, expectedApp1)
	}

	if app2Key != expectedApp2 {
		t.Errorf("App2 key = %v, want %v", app2Key, expectedApp2)
	}
}

// ===== MISSING TEST SCENARIOS =====

func TestBuilder_NilReceiver(t *testing.T) {
	var builder *cachex.Builder

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "Build with nil receiver",
			test: func() error {
				_ = builder.Build("entity", "id")
				return nil // Build should not panic
			},
		},
		{
			name: "BuildList with nil receiver",
			test: func() error {
				_ = builder.BuildList("entity", map[string]any{"key": "value"})
				return nil // BuildList should not panic
			},
		},
		{
			name: "BuildComposite with nil receiver",
			test: func() error {
				_ = builder.BuildComposite("entityA", "idA", "entityB", "idB")
				return nil // BuildComposite should not panic
			},
		},
		{
			name: "BuildSession with nil receiver",
			test: func() error {
				_ = builder.BuildSession("session123")
				return nil // BuildSession should not panic
			},
		},
		{
			name: "BuildUser with nil receiver",
			test: func() error {
				_ = builder.BuildUser("user123")
				return nil // BuildUser should not panic
			},
		},
		{
			name: "BuildOrg with nil receiver",
			test: func() error {
				_ = builder.BuildOrg("org123")
				return nil // BuildOrg should not panic
			},
		},
		{
			name: "BuildProduct with nil receiver",
			test: func() error {
				_ = builder.BuildProduct("product123")
				return nil // BuildProduct should not panic
			},
		},
		{
			name: "BuildOrder with nil receiver",
			test: func() error {
				_ = builder.BuildOrder("order123")
				return nil // BuildOrder should not panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should not panic and should return default values
			err := tt.test()
			if err != nil {
				t.Errorf("Nil receiver operation should not panic: %v", err)
			}
		})
	}
}

func TestBuilder_Validate(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		env         string
		secret      string
		expectError bool
	}{
		{
			name:        "Valid builder",
			appName:     "testapp",
			env:         "prod",
			secret:      "secret123",
			expectError: false,
		},
		{
			name:        "Empty app name",
			appName:     "",
			env:         "prod",
			secret:      "secret123",
			expectError: true,
		},
		{
			name:        "Empty environment",
			appName:     "testapp",
			env:         "",
			secret:      "secret123",
			expectError: true,
		},
		{
			name:        "Empty secret",
			appName:     "testapp",
			env:         "prod",
			secret:      "",
			expectError: true,
		},
		{
			name:        "Whitespace only app name",
			appName:     "   ",
			env:         "prod",
			secret:      "secret123",
			expectError: true,
		},
		{
			name:        "Whitespace only environment",
			appName:     "testapp",
			env:         "   ",
			secret:      "secret123",
			expectError: true,
		},
		{
			name:        "Whitespace only secret",
			appName:     "testapp",
			env:         "prod",
			secret:      "   ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := cachex.NewBuilder(tt.appName, tt.env, tt.secret)
			if tt.expectError {
				if err == nil {
					t.Errorf("NewBuilder() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewBuilder() unexpected error = %v", err)
			}

			// Test Validate method
			err = builder.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestBuilder_Validate_NilReceiver(t *testing.T) {
	var builder *cachex.Builder
	err := builder.Validate()
	if err == nil {
		t.Errorf("Validate() should return error for nil receiver")
	}
}

func TestBuilder_GetConfig(t *testing.T) {
	appName := "testapp"
	env := "prod"
	secret := "secret123"

	builder, err := cachex.NewBuilder(appName, env, secret)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	retrievedAppName, retrievedEnv := builder.GetConfig()

	if retrievedAppName != appName {
		t.Errorf("GetConfig() appName = %v, want %v", retrievedAppName, appName)
	}

	if retrievedEnv != env {
		t.Errorf("GetConfig() env = %v, want %v", retrievedEnv, env)
	}
}

func TestBuilder_GetConfig_NilReceiver(t *testing.T) {
	var builder *cachex.Builder
	appName, env := builder.GetConfig()

	if appName != "" {
		t.Errorf("GetConfig() should return empty appName for nil receiver, got %v", appName)
	}

	if env != "" {
		t.Errorf("GetConfig() should return empty env for nil receiver, got %v", env)
	}
}

func TestBuilder_Hash_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name    string
		filters map[string]any
		want    string
	}{
		{
			name:    "Empty filters",
			filters: map[string]any{},
			want:    "app:testapp:env:prod:list:test:all",
		},
		{
			name:    "Nil filters",
			filters: nil,
			want:    "app:testapp:env:prod:list:test:all",
		},
		{
			name:    "Valid filters",
			filters: map[string]any{"key": "value"},
			want:    "app:testapp:env:prod:list:test:",
		},
		{
			name:    "Special characters in filters",
			filters: map[string]any{"key@#$": "value%^&"},
			want:    "app:testapp:env:prod:list:test:",
		},
		{
			name:    "Unicode in filters",
			filters: map[string]any{"key数据": "value数据"},
			want:    "app:testapp:env:prod:list:test:",
		},
		{
			name:    "Very long filter values",
			filters: map[string]any{"key": strings.Repeat("a", 1000)},
			want:    "app:testapp:env:prod:list:test:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildList("test", tt.filters)

			if tt.filters == nil || len(tt.filters) == 0 {
				if got != tt.want {
					t.Errorf("BuildList() = %v, want %v", got, tt.want)
				}
			} else {
				// For non-empty filters, check prefix and hash format
				if !strings.HasPrefix(got, "app:testapp:env:prod:list:test:") {
					t.Errorf("BuildList() = %v, should start with app:testapp:env:prod:list:test:", got)
				}

				// Check that the hash part is 16 characters (hex)
				parts := strings.Split(got, ":")
				if len(parts) < 5 {
					t.Errorf("BuildList() = %v, should have at least 5 parts", got)
				}
				hash := parts[len(parts)-1]
				if len(hash) != 16 {
					t.Errorf("BuildList() hash = %v, should be 16 characters", hash)
				}
			}
		})
	}
}

func TestBuilder_Hash_Consistency(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	filters := map[string]any{"key": "value"}
	key1 := builder.BuildList("test", filters)
	key2 := builder.BuildList("test", filters)

	if key1 != key2 {
		t.Errorf("BuildList() should be consistent: %v != %v", key1, key2)
	}
}

func TestBuilder_Hash_DifferentSecrets(t *testing.T) {
	builder1, err1 := cachex.NewBuilder("testapp", "prod", "secret1")
	if err1 != nil {
		t.Fatalf("NewBuilder() error = %v", err1)
	}

	builder2, err2 := cachex.NewBuilder("testapp", "prod", "secret2")
	if err2 != nil {
		t.Fatalf("NewBuilder() error = %v", err2)
	}

	filters := map[string]any{"key": "value"}
	key1 := builder1.BuildList("test", filters)
	key2 := builder2.BuildList("test", filters)

	if key1 == key2 {
		t.Errorf("BuildList() should be different with different secrets: %v == %v", key1, key2)
	}
}

func TestBuilder_ParseKey_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name       string
		key        string
		wantEntity string
		wantID     string
		wantErr    bool
	}{
		{
			name:    "Empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "Whitespace only key",
			key:     "   ",
			wantErr: true,
		},
		{
			name:    "Too few parts",
			key:     "app:testapp:env:prod",
			wantErr: true,
		},
		{
			name:    "Wrong app prefix",
			key:     "wrong:testapp:env:prod:user:123",
			wantErr: true,
		},
		{
			name:    "Wrong env prefix",
			key:     "app:testapp:wrong:prod:user:123",
			wantErr: true,
		},
		{
			name:    "Wrong app name",
			key:     "app:wrongapp:env:prod:user:123",
			wantErr: true,
		},
		{
			name:    "Wrong environment",
			key:     "app:testapp:env:wrong:user:123",
			wantErr: true,
		},
		{
			name:    "Empty parts",
			key:     "app::env:prod:user:123",
			wantErr: true,
		},
		{
			name:    "Whitespace parts",
			key:     "app: :env:prod:user:123",
			wantErr: true,
		},
		{
			name:       "Valid key with colons in ID",
			key:        "app:testapp:env:prod:user:123:token:abc",
			wantEntity: "user",
			wantID:     "123:token:abc",
			wantErr:    false,
		},
		{
			name:       "Valid key with special characters",
			key:        "app:testapp:env:prod:user:user@example.com",
			wantEntity: "user",
			wantID:     "user@example.com",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, id, err := builder.ParseKey(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseKey() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseKey() unexpected error: %v", err)
				return
			}

			if entity != tt.wantEntity {
				t.Errorf("ParseKey() entity = %v, want %v", entity, tt.wantEntity)
			}

			if id != tt.wantID {
				t.Errorf("ParseKey() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestBuilder_ParseKey_NilReceiver(t *testing.T) {
	var builder *cachex.Builder
	_, _, err := builder.ParseKey("app:testapp:env:prod:user:123")
	if err == nil {
		t.Errorf("ParseKey() should return error for nil receiver")
	}
}

func TestBuilder_IsValidKey_NilReceiver(t *testing.T) {
	var builder *cachex.Builder
	result := builder.IsValidKey("app:testapp:env:prod:user:123")
	if result {
		t.Errorf("IsValidKey() should return false for nil receiver")
	}
}

func TestBuilder_ConcurrentAccess(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	numGoroutines := 10
	numOperations := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent Build operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := builder.Build(fmt.Sprintf("entity-%d", id), fmt.Sprintf("id-%d", j))
				if key == "" {
					t.Errorf("Build() returned empty key")
				}
			}
		}(i)
	}

	wg.Wait()

	// Test concurrent BuildList operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				filters := map[string]any{
					"filter1": fmt.Sprintf("value-%d", j),
					"filter2": id,
				}
				key := builder.BuildList(fmt.Sprintf("entity-%d", id), filters)
				if key == "" {
					t.Errorf("BuildList() returned empty key")
				}
			}
		}(i)
	}

	wg.Wait()

	// Test concurrent BuildList operations (which use hash internally)
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				filters := map[string]any{
					"filter1": fmt.Sprintf("value-%d-%d", id, j),
					"filter2": id,
				}
				key := builder.BuildList(fmt.Sprintf("entity-%d", id), filters)
				if key == "" {
					t.Errorf("BuildList() returned empty key")
				}
				// Check that the hash part is 16 characters
				parts := strings.Split(key, ":")
				if len(parts) >= 5 {
					hash := parts[len(parts)-1]
					if len(hash) != 16 {
						t.Errorf("BuildList() hash returned wrong length: %d", len(hash))
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestBuilder_Performance(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	// Test Build performance
	start := time.Now()
	for i := 0; i < 10000; i++ {
		key := builder.Build(fmt.Sprintf("entity-%d", i), fmt.Sprintf("id-%d", i))
		if key == "" {
			t.Errorf("Build() returned empty key")
		}
	}
	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		t.Errorf("Build() performance too slow: %v", duration)
	}

	// Test BuildList performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		filters := map[string]any{
			"filter1": fmt.Sprintf("value-%d", i),
			"filter2": i,
			"filter3": true,
		}
		key := builder.BuildList(fmt.Sprintf("entity-%d", i), filters)
		if key == "" {
			t.Errorf("BuildList() returned empty key")
		}
	}
	duration = time.Since(start)
	if duration > 200*time.Millisecond {
		t.Errorf("BuildList() performance too slow: %v", duration)
	}

	// Test BuildList performance (which uses hash internally)
	start = time.Now()
	for i := 0; i < 10000; i++ {
		filters := map[string]any{
			"filter1": fmt.Sprintf("value-%d", i),
			"filter2": i,
		}
		key := builder.BuildList(fmt.Sprintf("entity-%d", i), filters)
		if key == "" {
			t.Errorf("BuildList() returned empty key")
		}
		// Check that the hash part is 16 characters
		parts := strings.Split(key, ":")
		if len(parts) >= 5 {
			hash := parts[len(parts)-1]
			if len(hash) != 16 {
				t.Errorf("BuildList() hash returned wrong length: %d", len(hash))
			}
		}
	}
	duration = time.Since(start)
	if duration > 500*time.Millisecond {
		t.Errorf("BuildList() performance too slow: %v", duration)
	}
}

func TestBuilder_BuildList_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name    string
		entity  string
		filters map[string]any
		want    string
	}{
		{
			name:    "Empty entity",
			entity:  "",
			filters: map[string]any{"key": "value"},
			want:    "app:testapp:env:prod:list:unknown:",
		},
		{
			name:    "Whitespace entity",
			entity:  "   ",
			filters: map[string]any{"key": "value"},
			want:    "app:testapp:env:prod:list:unknown:",
		},
		{
			name:    "Filters with empty keys",
			entity:  "users",
			filters: map[string]any{"": "value", "key": "value"},
			want:    "app:testapp:env:prod:list:users:",
		},
		{
			name:    "Filters with whitespace keys",
			entity:  "users",
			filters: map[string]any{"   ": "value", "key": "value"},
			want:    "app:testapp:env:prod:list:users:",
		},
		{
			name:    "Filters with nil values",
			entity:  "users",
			filters: map[string]any{"key": nil},
			want:    "app:testapp:env:prod:list:users:",
		},
		{
			name:   "Filters with complex values",
			entity: "users",
			filters: map[string]any{
				"key1": []string{"a", "b", "c"},
				"key2": map[string]int{"nested": 123},
				"key3": struct{ Name string }{"test"},
			},
			want: "app:testapp:env:prod:list:users:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildList(tt.entity, tt.filters)

			// For non-empty filters, check prefix and hash format
			if len(tt.filters) > 0 {
				expectedEntity := tt.entity
				if strings.TrimSpace(expectedEntity) == "" {
					expectedEntity = "unknown"
				}
				if !strings.HasPrefix(got, "app:testapp:env:prod:list:"+expectedEntity+":") {
					t.Errorf("BuildList() = %v, should start with app:testapp:env:prod:list:%s:", got, expectedEntity)
				}

				// Check that the hash part is 16 characters (hex)
				parts := strings.Split(got, ":")
				if len(parts) < 5 {
					t.Errorf("BuildList() = %v, should have at least 5 parts", got)
				}
				hash := parts[len(parts)-1]
				if len(hash) != 16 {
					t.Errorf("BuildList() hash = %v, should be 16 characters", hash)
				}
			} else {
				if got != tt.want {
					t.Errorf("BuildList() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBuilder_BuildComposite_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name    string
		entityA string
		idA     string
		entityB string
		idB     string
		want    string
	}{
		{
			name:    "All empty",
			entityA: "",
			idA:     "",
			entityB: "",
			idB:     "",
			want:    "app:testapp:env:prod:unknown:unknown:unknown:unknown",
		},
		{
			name:    "Mixed empty",
			entityA: "user",
			idA:     "",
			entityB: "",
			idB:     "456",
			want:    "app:testapp:env:prod:user:unknown:unknown:456",
		},
		{
			name:    "Whitespace only",
			entityA: "   ",
			idA:     "   ",
			entityB: "   ",
			idB:     "   ",
			want:    "app:testapp:env:prod:unknown:unknown:unknown:unknown",
		},
		{
			name:    "Special characters in IDs",
			entityA: "user",
			idA:     "user@example.com",
			entityB: "order",
			idB:     "order#123",
			want:    "app:testapp:env:prod:user:user@example.com:order:order#123",
		},
		{
			name:    "Unicode characters",
			entityA: "用户",
			idA:     "用户123",
			entityB: "订单",
			idB:     "订单456",
			want:    "app:testapp:env:prod:用户:用户123:订单:订单456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildComposite(tt.entityA, tt.idA, tt.entityB, tt.idB)
			if got != tt.want {
				t.Errorf("BuildComposite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_BuildSession_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name string
		sid  string
		want string
	}{
		{
			name: "Empty session ID",
			sid:  "",
			want: "app:testapp:env:prod:session:unknown",
		},
		{
			name: "Whitespace session ID",
			sid:  "   ",
			want: "app:testapp:env:prod:session:unknown",
		},
		{
			name: "Session ID with special characters",
			sid:  "session@123#456",
			want: "app:testapp:env:prod:session:session@123#456",
		},
		{
			name: "Session ID with colons",
			sid:  "session:123:token:abc",
			want: "app:testapp:env:prod:session:session:123:token:abc",
		},
		{
			name: "Unicode session ID",
			sid:  "会话123",
			want: "app:testapp:env:prod:session:会话123",
		},
		{
			name: "Very long session ID",
			sid:  strings.Repeat("a", 1000),
			want: "app:testapp:env:prod:session:" + strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildSession(tt.sid)
			if got != tt.want {
				t.Errorf("BuildSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuilder_HelperMethods_EdgeCases(t *testing.T) {
	builder, err := cachex.NewBuilder("testapp", "prod", "secret123")
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	tests := []struct {
		name   string
		method func(string) string
		input  string
		want   string
	}{
		{
			name:   "BuildUser with empty ID",
			method: builder.BuildUser,
			input:  "",
			want:   "app:testapp:env:prod:user:unknown",
		},
		{
			name:   "BuildUser with whitespace ID",
			method: builder.BuildUser,
			input:  "   ",
			want:   "app:testapp:env:prod:user:unknown",
		},
		{
			name:   "BuildOrg with empty ID",
			method: builder.BuildOrg,
			input:  "",
			want:   "app:testapp:env:prod:org:unknown",
		},
		{
			name:   "BuildProduct with empty ID",
			method: builder.BuildProduct,
			input:  "",
			want:   "app:testapp:env:prod:product:unknown",
		},
		{
			name:   "BuildOrder with empty ID",
			method: builder.BuildOrder,
			input:  "",
			want:   "app:testapp:env:prod:order:unknown",
		},
		{
			name:   "BuildUser with special characters",
			method: builder.BuildUser,
			input:  "user@example.com",
			want:   "app:testapp:env:prod:user:user@example.com",
		},
		{
			name:   "BuildOrg with unicode",
			method: builder.BuildOrg,
			input:  "组织123",
			want:   "app:testapp:env:prod:org:组织123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.method(tt.input)
			if got != tt.want {
				t.Errorf("%s() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestBuilder_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		env         string
		secret      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "All empty",
			appName:     "",
			env:         "",
			secret:      "",
			expectError: true,
			errorMsg:    "app name cannot be empty",
		},
		{
			name:        "Empty app name with whitespace env and secret",
			appName:     "",
			env:         "   ",
			secret:      "   ",
			expectError: true,
			errorMsg:    "app name cannot be empty",
		},
		{
			name:        "Valid app name, empty env",
			appName:     "testapp",
			env:         "",
			secret:      "secret",
			expectError: true,
			errorMsg:    "environment cannot be empty",
		},
		{
			name:        "Valid app name and env, empty secret",
			appName:     "testapp",
			env:         "prod",
			secret:      "",
			expectError: true,
			errorMsg:    "secret cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cachex.NewBuilder(tt.appName, tt.env, tt.secret)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewBuilder() expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("NewBuilder() error = %v, should contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewBuilder() unexpected error = %v", err)
				}
			}
		})
	}
}
