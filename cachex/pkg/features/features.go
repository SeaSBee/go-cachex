package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seasbee/go-logx"
)

// Feature represents a feature flag
type Feature string

const (
	// RefreshAhead enables/disables refresh-ahead functionality
	RefreshAhead Feature = "refresh_ahead"
	// NegativeCaching enables/disables negative caching
	NegativeCaching Feature = "negative_caching"
	// BloomFilter enables/disables bloom filter functionality
	BloomFilter Feature = "bloom_filter"
	// DeadLetterQueue enables/disables dead-letter queue
	DeadLetterQueue Feature = "dead_letter_queue"
	// PubSubInvalidation enables/disables pub/sub invalidation
	PubSubInvalidation Feature = "pubsub_invalidation"
	// CircuitBreaker enables/disables circuit breaker
	CircuitBreaker Feature = "circuit_breaker"
	// RateLimiting enables/disables rate limiting
	RateLimiting Feature = "rate_limiting"
	// Encryption enables/disables value encryption
	Encryption Feature = "encryption"
	// Compression enables/disables value compression
	Compression Feature = "compression"
	// Observability enables/disables observability features
	Observability Feature = "observability"
	// Tagging enables/disables tagging functionality
	Tagging Feature = "tagging"
)

// Manager manages feature flags
type Manager struct {
	// Feature states
	features map[Feature]*FeatureState

	// Configuration
	config *Config

	// Mutex for thread safety
	mu sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Observers for feature changes
	observers map[Feature][]FeatureObserver
}

// FeatureState represents the state of a feature
type FeatureState struct {
	Enabled     bool
	Percentage  float64 // For gradual rollouts (0-100)
	StartTime   time.Time
	EndTime     *time.Time // Optional end time for temporary features
	Description string
	mu          sync.RWMutex
}

// FeatureObserver is called when a feature state changes
type FeatureObserver func(feature Feature, enabled bool)

// Config holds feature flags configuration
type Config struct {
	// Default state for features
	DefaultEnabled map[Feature]bool
	// Enable dynamic updates
	EnableDynamicUpdates bool
	// Update interval for dynamic features
	UpdateInterval time.Duration
	// Enable logging
	EnableLogging bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultEnabled: map[Feature]bool{
			RefreshAhead:       true,
			NegativeCaching:    true,
			BloomFilter:        true,
			DeadLetterQueue:    true,
			PubSubInvalidation: true,
			CircuitBreaker:     true,
			RateLimiting:       true,
			Encryption:         false,
			Compression:        false,
			Observability:      true,
			Tagging:            true,
		},
		EnableDynamicUpdates: true,
		UpdateInterval:       30 * time.Second,
		EnableLogging:        true,
	}
}

// ProductionConfig returns a production configuration
func ProductionConfig() *Config {
	return &Config{
		DefaultEnabled: map[Feature]bool{
			RefreshAhead:       true,
			NegativeCaching:    true,
			BloomFilter:        true,
			DeadLetterQueue:    true,
			PubSubInvalidation: true,
			CircuitBreaker:     true,
			RateLimiting:       true,
			Encryption:         true,
			Compression:        true,
			Observability:      true,
			Tagging:            true,
		},
		EnableDynamicUpdates: true,
		UpdateInterval:       60 * time.Second,
		EnableLogging:        true,
	}
}

// DevelopmentConfig returns a development configuration
func DevelopmentConfig() *Config {
	return &Config{
		DefaultEnabled: map[Feature]bool{
			RefreshAhead:       false,
			NegativeCaching:    true,
			BloomFilter:        false,
			DeadLetterQueue:    false,
			PubSubInvalidation: false,
			CircuitBreaker:     false,
			RateLimiting:       false,
			Encryption:         false,
			Compression:        false,
			Observability:      true,
			Tagging:            true,
		},
		EnableDynamicUpdates: true,
		UpdateInterval:       10 * time.Second,
		EnableLogging:        true,
	}
}

// New creates a new feature flags manager
func New(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		features:  make(map[Feature]*FeatureState),
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		observers: make(map[Feature][]FeatureObserver),
	}

	// Initialize features with default states
	for feature, enabled := range config.DefaultEnabled {
		manager.features[feature] = &FeatureState{
			Enabled:     enabled,
			Percentage:  100.0, // Full rollout by default
			StartTime:   time.Now(),
			Description: fmt.Sprintf("Default state for %s", feature),
		}
	}

	// Start dynamic updates if enabled
	if config.EnableDynamicUpdates {
		go manager.startDynamicUpdates()
	}

	return manager
}

// IsEnabled checks if a feature is enabled
func (m *Manager) IsEnabled(feature Feature) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.features[feature]
	if !exists {
		return false
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	// Check if feature has expired
	if state.EndTime != nil && time.Now().After(*state.EndTime) {
		return false
	}

	return state.Enabled
}

// IsEnabledWithPercentage checks if a feature is enabled with percentage rollout
func (m *Manager) IsEnabledWithPercentage(feature Feature, identifier string) bool {
	if !m.IsEnabled(feature) {
		return false
	}

	m.mu.RLock()
	state, exists := m.features[feature]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	state.mu.RLock()
	percentage := state.Percentage
	state.mu.RUnlock()

	// Calculate hash-based percentage
	hash := hashString(identifier)
	userPercentage := float64(hash%100) + 1

	return userPercentage <= percentage
}

// Enable enables a feature
func (m *Manager) Enable(feature Feature, description string) error {
	return m.updateFeature(feature, true, 100.0, description, nil)
}

// Disable disables a feature
func (m *Manager) Disable(feature Feature, description string) error {
	return m.updateFeature(feature, false, 0.0, description, nil)
}

// EnableWithPercentage enables a feature with percentage rollout
func (m *Manager) EnableWithPercentage(feature Feature, percentage float64, description string) error {
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("percentage must be between 0 and 100")
	}
	return m.updateFeature(feature, true, percentage, description, nil)
}

// EnableTemporarily enables a feature temporarily
func (m *Manager) EnableTemporarily(feature Feature, duration time.Duration, description string) error {
	endTime := time.Now().Add(duration)
	return m.updateFeature(feature, true, 100.0, description, &endTime)
}

// GetFeatureState returns the current state of a feature
func (m *Manager) GetFeatureState(feature Feature) (*FeatureState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.features[feature]
	if !exists {
		return nil, fmt.Errorf("feature %s not found", feature)
	}

	// Return a copy to avoid race conditions
	state.mu.RLock()
	defer state.mu.RUnlock()

	return &FeatureState{
		Enabled:     state.Enabled,
		Percentage:  state.Percentage,
		StartTime:   state.StartTime,
		EndTime:     state.EndTime,
		Description: state.Description,
	}, nil
}

// GetAllFeatures returns all feature states
func (m *Manager) GetAllFeatures() map[Feature]*FeatureState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[Feature]*FeatureState)
	for feature, state := range m.features {
		state.mu.RLock()
		result[feature] = &FeatureState{
			Enabled:     state.Enabled,
			Percentage:  state.Percentage,
			StartTime:   state.StartTime,
			EndTime:     state.EndTime,
			Description: state.Description,
		}
		state.mu.RUnlock()
	}

	return result
}

// AddObserver adds an observer for feature changes
func (m *Manager) AddObserver(feature Feature, observer FeatureObserver) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.observers[feature] = append(m.observers[feature], observer)
}

// RemoveObserver removes an observer
func (m *Manager) RemoveObserver(feature Feature, observer FeatureObserver) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.observers[feature]
	if !exists {
		return
	}

	// Note: In Go, function comparison is not supported
	// This method is provided for API completeness but doesn't actually remove observers
	// To properly remove observers, you would need to store them with unique identifiers
	logx.Warn("RemoveObserver called but function comparison is not supported in Go",
		logx.String("feature", string(feature)))
}

// Close closes the feature flags manager
func (m *Manager) Close() error {
	m.cancel()
	logx.Info("Closed feature flags manager")
	return nil
}

// updateFeature updates a feature state
func (m *Manager) updateFeature(feature Feature, enabled bool, percentage float64, description string, endTime *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create feature state
	state, exists := m.features[feature]
	if !exists {
		state = &FeatureState{}
		m.features[feature] = state
	}

	// Check if state is actually changing
	state.mu.Lock()
	oldEnabled := state.Enabled
	state.Enabled = enabled
	state.Percentage = percentage
	state.Description = description
	state.EndTime = endTime
	state.mu.Unlock()

	// Notify observers if state changed
	if oldEnabled != enabled {
		m.notifyObservers(feature, enabled)
	}

	// Log the change
	if m.config.EnableLogging {
		logx.Info("Feature flag updated",
			logx.String("feature", string(feature)),
			logx.Bool("enabled", enabled),
			logx.Float64("percentage", percentage),
			logx.String("description", description))
	}

	return nil
}

// notifyObservers notifies all observers of a feature change
func (m *Manager) notifyObservers(feature Feature, enabled bool) {
	observers, exists := m.observers[feature]
	if !exists {
		return
	}

	for _, observer := range observers {
		go func(obs FeatureObserver) {
			defer func() {
				if r := recover(); r != nil {
					logx.Error("Feature observer panicked",
						logx.String("feature", string(feature)),
						logx.Any("panic", r))
				}
			}()
			obs(feature, enabled)
		}(observer)
	}
}

// startDynamicUpdates starts the dynamic update loop
func (m *Manager) startDynamicUpdates() {
	ticker := time.NewTicker(m.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupExpiredFeatures()
		}
	}
}

// cleanupExpiredFeatures removes expired features
func (m *Manager) cleanupExpiredFeatures() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for feature, state := range m.features {
		state.mu.RLock()
		endTime := state.EndTime
		state.mu.RUnlock()

		if endTime != nil && now.After(*endTime) {
			// Feature has expired, disable it
			state.mu.Lock()
			oldEnabled := state.Enabled
			state.Enabled = false
			state.mu.Unlock()

			if oldEnabled {
				m.notifyObservers(feature, false)
				if m.config.EnableLogging {
					logx.Info("Feature flag expired and disabled",
						logx.String("feature", string(feature)))
				}
			}
		}
	}
}

// hashString creates a simple hash for percentage-based rollouts
func hashString(s string) int {
	hash := 0
	for _, char := range s {
		hash = ((hash << 5) - hash) + int(char)
		hash = hash & hash // Convert to 32-bit integer
	}
	return hash
}
