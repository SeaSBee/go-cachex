package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"
)

// Encryptor provides AES-GCM encryption with key rotation
type Encryptor struct {
	mu           sync.RWMutex
	currentKey   []byte
	previousKey  []byte
	keyRotation  time.Duration
	lastRotation time.Time
}

// Config holds encryption configuration
type Config struct {
	Key          string        // Base64 encoded 32-byte key
	KeyRotation  time.Duration // How often to rotate keys
	AutoRotation bool          // Whether to auto-rotate keys
}

// NewEncryptor creates a new encryptor with the given configuration
func NewEncryptor(config *Config) (*Encryptor, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Decode the key
	key, err := base64.StdEncoding.DecodeString(config.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid key format: %w", err)
	}

	// Validate key length (AES-256 requires 32 bytes)
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}

	// Set default rotation interval
	rotation := config.KeyRotation
	if rotation == 0 {
		rotation = 24 * time.Hour // Default to 24 hours
	}

	return &Encryptor{
		currentKey:   key,
		keyRotation:  rotation,
		lastRotation: time.Now(),
	}, nil
}

// Encrypt encrypts data using AES-GCM
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if we need to rotate keys
	if e.shouldRotate() {
		e.mu.RUnlock()
		e.rotateKey()
		e.mu.RLock()
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.currentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Try current key first
	plaintext, err := e.decryptWithKey(data, e.currentKey)
	if err == nil {
		return plaintext, nil
	}

	// Try previous key if available
	if e.previousKey != nil {
		plaintext, err := e.decryptWithKey(data, e.previousKey)
		if err == nil {
			return plaintext, nil
		}
	}

	return nil, fmt.Errorf("failed to decrypt data: %w", err)
}

// decryptWithKey attempts to decrypt data with a specific key
func (e *Encryptor) decryptWithKey(data []byte, key []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// shouldRotate checks if it's time to rotate keys
func (e *Encryptor) shouldRotate() bool {
	return time.Since(e.lastRotation) >= e.keyRotation
}

// rotateKey rotates the encryption key
func (e *Encryptor) rotateKey() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		// If we can't generate a new key, keep the current one
		return
	}

	// Move current key to previous
	e.previousKey = e.currentKey
	e.currentKey = newKey
	e.lastRotation = time.Now()
}

// ForceRotate forces a key rotation
func (e *Encryptor) ForceRotate() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Move current key to previous
	e.previousKey = e.currentKey
	e.currentKey = newKey
	e.lastRotation = time.Now()

	return nil
}

// GetKeyInfo returns information about the current key
func (e *Encryptor) GetKeyInfo() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	info := map[string]interface{}{
		"current_key_hash":  hex.EncodeToString(sha256.New().Sum(e.currentKey)),
		"last_rotation":     e.lastRotation,
		"rotation_interval": e.keyRotation,
		"has_previous_key":  e.previousKey != nil,
	}

	if e.previousKey != nil {
		info["previous_key_hash"] = hex.EncodeToString(sha256.New().Sum(e.previousKey))
	}

	return info
}

// GenerateKey generates a new random 32-byte key
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// ValidateKey validates that a key is properly formatted
func ValidateKey(key string) error {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	if len(decoded) != 32 {
		return fmt.Errorf("key must be 32 bytes, got %d", len(decoded))
	}

	return nil
}
