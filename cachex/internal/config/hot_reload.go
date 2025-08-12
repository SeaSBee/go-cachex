package config

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/seasbee/go-logx"
)

// Manager handles configuration loading and hot-reload
type Manager struct {
	config     *Config
	configFile string
	callbacks  []func(*Config)
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager creates a new configuration manager
func NewManager(configFile string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		configFile: configFile,
		callbacks:  make([]func(*Config), 0),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Load loads configuration from file and environment
func (m *Manager) Load() (*Config, error) {
	var config *Config
	var err error

	// Load from file if it exists
	if m.configFile != "" {
		config, err = LoadFromFile(m.configFile)
		if err != nil {
			logx.Warn("Failed to load config file, using defaults", logx.String("file", m.configFile), logx.ErrorField(err))
			config = DefaultConfig()
			LoadFromEnvironment(config)
		}
	} else {
		config = DefaultConfig()
		LoadFromEnvironment(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	return config, nil
}

// OnReload registers a callback to be called when configuration is reloaded
func (m *Manager) OnReload(callback func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// StartHotReload starts hot-reload monitoring
func (m *Manager) StartHotReload() error {
	if !m.config.HotReload.Enabled {
		return nil
	}

	// Start file monitoring
	if m.configFile != "" {
		go m.monitorFile()
	}

	// Start signal monitoring
	if m.config.HotReload.SignalReload {
		go m.monitorSignals()
	}

	return nil
}

// monitorFile monitors the configuration file for changes
func (m *Manager) monitorFile() {
	ticker := time.NewTicker(m.config.HotReload.CheckInterval)
	defer ticker.Stop()

	var lastModTime time.Time
	if info, err := os.Stat(m.configFile); err == nil {
		lastModTime = info.ModTime()
	}

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if info, err := os.Stat(m.configFile); err == nil {
				if info.ModTime().After(lastModTime) {
					logx.Info("Configuration file changed, reloading", logx.String("file", m.configFile))
					if err := m.reload(); err != nil {
						logx.Error("Failed to reload configuration", logx.ErrorField(err))
					}
					lastModTime = info.ModTime()
				}
			}
		}
	}
}

// monitorSignals monitors for SIGHUP signal
func (m *Manager) monitorSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-sigChan:
			logx.Info("Received SIGHUP, reloading configuration")
			if err := m.reload(); err != nil {
				logx.Error("Failed to reload configuration", logx.ErrorField(err))
			}
		}
	}
}

// reload reloads the configuration and notifies callbacks
func (m *Manager) reload() error {
	newConfig, err := m.Load()
	if err != nil {
		return err
	}

	m.mu.RLock()
	callbacks := make([]func(*Config), len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.RUnlock()

	// Notify callbacks
	for _, callback := range callbacks {
		callback(newConfig)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Stop stops the configuration manager
func (m *Manager) Stop() {
	m.cancel()
}
