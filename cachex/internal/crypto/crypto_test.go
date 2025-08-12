package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	// Test with valid config
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: true,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)
	assert.NotNil(t, encryptor)
}

func TestNewEncryptor_InvalidConfig(t *testing.T) {
	// Test with nil config
	_, err := NewEncryptor(nil)
	assert.Error(t, err)

	// Test with invalid key
	_, err = NewEncryptor(&Config{Key: "invalid"})
	assert.Error(t, err)

	// Test with short key
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = NewEncryptor(&Config{Key: shortKey})
	assert.Error(t, err)
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Test data
	testData := []byte("Hello, World!")

	// Encrypt
	encrypted, err := encryptor.Encrypt(testData)
	require.NoError(t, err)
	assert.NotEqual(t, testData, encrypted)
	assert.True(t, len(encrypted) > len(testData))

	// Decrypt
	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestEncryptor_KeyRotation(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Millisecond, // Very short for testing
		AutoRotation: true,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Encrypt with current key
	testData := []byte("test data")
	encrypted1, err := encryptor.Encrypt(testData)
	require.NoError(t, err)

	// Wait for key rotation
	time.Sleep(10 * time.Millisecond)

	// Encrypt with new key
	encrypted2, err := encryptor.Encrypt(testData)
	require.NoError(t, err)

	// Both should still be decryptable
	decrypted1, err := encryptor.Decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted1)

	decrypted2, err := encryptor.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted2)
}

func TestEncryptor_ForceRotate(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Get initial key info
	initialInfo := encryptor.GetKeyInfo()
	initialHash := initialInfo["current_key_hash"].(string)

	// Force rotate
	err = encryptor.ForceRotate()
	require.NoError(t, err)

	// Get new key info
	newInfo := encryptor.GetKeyInfo()
	newHash := newInfo["current_key_hash"].(string)

	// Keys should be different
	assert.NotEqual(t, initialHash, newHash)
	assert.True(t, newInfo["has_previous_key"].(bool))
}

func TestEncryptor_GetKeyInfo(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	info := encryptor.GetKeyInfo()

	// Check required fields
	assert.NotEmpty(t, info["current_key_hash"])
	assert.NotNil(t, info["last_rotation"])
	assert.Equal(t, 1*time.Hour, info["rotation_interval"])
	assert.False(t, info["has_previous_key"].(bool))
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	// Decode and check length
	decoded, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	assert.Equal(t, 32, len(decoded))

	// Generate multiple keys - they should be different
	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, key, key2)
}

func TestValidateKey(t *testing.T) {
	// Test valid key
	key, err := GenerateKey()
	require.NoError(t, err)
	err = ValidateKey(key)
	assert.NoError(t, err)

	// Test invalid base64
	err = ValidateKey("invalid")
	assert.Error(t, err)

	// Test wrong length
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	err = ValidateKey(shortKey)
	assert.Error(t, err)
}

func TestEncryptor_EmptyData(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Test empty data
	emptyData := []byte{}
	encrypted, err := encryptor.Encrypt(emptyData)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	// Both empty and nil should result in empty slice after encryption/decryption
	assert.Equal(t, 0, len(decrypted))

	// Test nil data
	nilData := []byte(nil)
	encrypted, err = encryptor.Encrypt(nilData)
	require.NoError(t, err)

	decrypted, err = encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	// Both empty and nil should result in empty slice after encryption/decryption
	assert.Equal(t, 0, len(decrypted))
}

func TestEncryptor_LargeData(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Test large data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := encryptor.Encrypt(largeData)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, largeData, decrypted)
}

func TestEncryptor_ConcurrentAccess(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Key:          key,
		KeyRotation:  1 * time.Hour,
		AutoRotation: false,
	}

	encryptor, err := NewEncryptor(config)
	require.NoError(t, err)

	// Test concurrent encryption/decryption
	const numGoroutines = 10
	const operationsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				data := []byte(fmt.Sprintf("data-%d-%d", id, j))

				encrypted, err := encryptor.Encrypt(data)
				require.NoError(t, err)

				decrypted, err := encryptor.Decrypt(encrypted)
				require.NoError(t, err)
				assert.Equal(t, data, decrypted)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
