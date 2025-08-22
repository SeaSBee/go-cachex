package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SeaSBee/go-cachex"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		env     string
		secret  string
	}{
		{
			name:    "basic builder",
			appName: "testapp",
			env:     "prod",
			secret:  "secret123",
		},
		{
			name:    "empty secret",
			appName: "testapp",
			env:     "dev",
			secret:  "",
		},
		{
			name:    "empty app name",
			appName: "",
			env:     "test",
			secret:  "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := cachex.NewBuilder(tt.appName, tt.env, tt.secret)

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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
			want:   "app:testapp:env:prod::123",
		},
		{
			name:   "empty id",
			entity: "user",
			id:     "",
			want:   "app:testapp:env:prod:user:",
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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildList(tt.entity, tt.filters)

			// For empty filters, we can check exact match
			if len(tt.filters) == 0 {
				if got != tt.want {
					t.Errorf("Builder.BuildList() = %v, want %v", got, tt.want)
				}
				return
			}

			// For non-empty filters, check prefix and hash format
			if !strings.HasPrefix(got, "app:testapp:env:prod:list:"+tt.entity+":") {
				t.Errorf("Builder.BuildList() = %v, should start with app:testapp:env:prod:list:%s:", got, tt.entity)
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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")
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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
			want:    "app:testapp:env:prod::123::456",
		},
		{
			name:    "ids with colons",
			entityA: "user",
			idA:     "user:123:token:abc",
			entityB: "order",
			idB:     "order:456:status:pending",
			want:    "app:testapp:env:prod:user:user:123:token:abc:order:order:456:status:pending",
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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
			want: "app:testapp:env:prod:session:",
		},
		{
			name: "session id with special characters",
			sid:  "session@123#456",
			want: "app:testapp:env:prod:session:session@123#456",
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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
			name:   "without secret",
			secret: "",
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
			builder := cachex.NewBuilder("testapp", "prod", tt.secret)

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

	builder1 := cachex.NewBuilder("testapp", "prod", "secret1")
	builder2 := cachex.NewBuilder("testapp", "prod", "secret2")

	key1 := builder1.BuildList("users", data)
	key2 := builder2.BuildList("users", data)

	if key1 == key2 {
		t.Errorf("BuildList() should be different with different secrets: %v == %v", key1, key2)
	}
}

func TestBuilder_ParseKey(t *testing.T) {
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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
	builder := cachex.NewBuilder("testapp", "prod", "secret123")

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

	devBuilder := cachex.NewBuilder(appName, "dev", "secret")
	prodBuilder := cachex.NewBuilder(appName, "prod", "secret")

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

	app1Builder := cachex.NewBuilder("app1", env, "secret")
	app2Builder := cachex.NewBuilder("app2", env, "secret")

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
