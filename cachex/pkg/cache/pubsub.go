package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seasbee/go-logx"
)

// PubSubManager manages Redis pub/sub for cache invalidation
type PubSubManager struct {
	// Redis client for pub/sub
	client redis.Cmdable
	// Configuration
	config *PubSubConfig
	// Subscribers
	subscribers map[string]*Subscriber
	// Mutex for thread safety
	mu sync.RWMutex
	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// PubSubConfig holds pub/sub configuration
type PubSubConfig struct {
	// Channel names
	InvalidationChannel string
	HealthChannel       string
	// Maximum number of subscribers
	MaxSubscribers int
	// Message timeout
	MessageTimeout time.Duration
	// Enable health monitoring
	EnableHealth bool
	// Retry configuration
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultPubSubConfig returns a default configuration
func DefaultPubSubConfig() *PubSubConfig {
	return &PubSubConfig{
		InvalidationChannel: "cachex:invalidation",
		HealthChannel:       "cachex:health",
		MaxSubscribers:      100,
		MessageTimeout:      5 * time.Second,
		EnableHealth:        true,
		RetryAttempts:       3,
		RetryDelay:          100 * time.Millisecond,
	}
}

// InvalidationMessage represents a cache invalidation message
type InvalidationMessage struct {
	Keys      []string  `json:"keys"`
	Tags      []string  `json:"tags"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	MessageID string    `json:"message_id"`
	TTL       int64     `json:"ttl,omitempty"`
}

// HealthMessage represents a health check message
type HealthMessage struct {
	ServiceName    string                 `json:"service_name"`
	ServiceVersion string                 `json:"service_version"`
	Timestamp      time.Time              `json:"timestamp"`
	Status         string                 `json:"status"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
}

// Subscriber represents a pub/sub subscriber
type Subscriber struct {
	ID      string
	Channel string
	Handler func(keys ...string)
	PubSub  *redis.PubSub
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	active  bool
}

// NewPubSubManager creates a new pub/sub manager
func NewPubSubManager(client redis.Cmdable, config *PubSubConfig) *PubSubManager {
	if config == nil {
		config = DefaultPubSubConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pm := &PubSubManager{
		client:      client,
		config:      config,
		subscribers: make(map[string]*Subscriber),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start health monitoring if enabled
	if config.EnableHealth {
		go pm.startHealthMonitoring()
	}

	return pm
}

// PublishInvalidation publishes an invalidation message
func (pm *PubSubManager) PublishInvalidation(ctx context.Context, keys []string, tags []string, source string) error {
	if len(keys) == 0 && len(tags) == 0 {
		return nil
	}

	message := InvalidationMessage{
		Keys:      keys,
		Tags:      tags,
		Timestamp: time.Now(),
		Source:    source,
		MessageID: generateMessageID(),
	}

	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal invalidation message: %w", err)
	}

	// Publish to Redis
	result := pm.client.Publish(ctx, pm.config.InvalidationChannel, data)
	if result.Err() != nil {
		return fmt.Errorf("failed to publish invalidation message: %w", result.Err())
	}

	// Log the publication
	recipients := result.Val()
	logx.Info("Published invalidation message",
		logx.String("channel", pm.config.InvalidationChannel),
		logx.Int("recipients", int(recipients)),
		logx.Int("keys", len(keys)),
		logx.Int("tags", len(tags)),
		logx.String("source", source))

	return nil
}

// SubscribeInvalidations subscribes to invalidation messages
func (pm *PubSubManager) SubscribeInvalidations(ctx context.Context, handler func(keys ...string)) (string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check subscriber limit
	if len(pm.subscribers) >= pm.config.MaxSubscribers {
		return "", fmt.Errorf("maximum number of subscribers reached: %d", pm.config.MaxSubscribers)
	}

	// Create subscriber
	subscriberID := generateSubscriberID()
	subCtx, subCancel := context.WithCancel(ctx)

	subscriber := &Subscriber{
		ID:      subscriberID,
		Channel: pm.config.InvalidationChannel,
		Handler: handler,
		ctx:     subCtx,
		cancel:  subCancel,
		active:  true,
	}

	// Subscribe to Redis channel
	pubsub := pm.client.(*redis.Client).Subscribe(subCtx, pm.config.InvalidationChannel)
	// Check subscription status
	_, err := pubsub.Receive(subCtx)
	if err != nil {
		subCancel()
		return "", fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	subscriber.PubSub = pubsub

	// Store subscriber
	pm.subscribers[subscriberID] = subscriber

	// Start message processing
	go pm.processMessages(subscriber)

	logx.Info("Subscribed to invalidation messages",
		logx.String("subscriber_id", subscriberID),
		logx.String("channel", pm.config.InvalidationChannel))

	return subscriberID, nil
}

// Unsubscribe removes a subscriber
func (pm *PubSubManager) Unsubscribe(subscriberID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	subscriber, exists := pm.subscribers[subscriberID]
	if !exists {
		return fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	// Cancel subscriber context
	subscriber.cancel()

	// Close pub/sub connection
	if subscriber.PubSub != nil {
		subscriber.PubSub.Close()
	}

	// Remove from subscribers
	delete(pm.subscribers, subscriberID)

	logx.Info("Unsubscribed from invalidation messages",
		logx.String("subscriber_id", subscriberID))

	return nil
}

// PublishHealth publishes a health message
func (pm *PubSubManager) PublishHealth(ctx context.Context, serviceName, serviceVersion, status string, metrics map[string]interface{}) error {
	message := HealthMessage{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Timestamp:      time.Now(),
		Status:         status,
		Metrics:        metrics,
	}

	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal health message: %w", err)
	}

	// Publish to Redis
	result := pm.client.Publish(ctx, pm.config.HealthChannel, data)
	if result.Err() != nil {
		return fmt.Errorf("failed to publish health message: %w", result.Err())
	}

	return nil
}

// processMessages processes incoming messages for a subscriber
func (pm *PubSubManager) processMessages(subscriber *Subscriber) {
	defer func() {
		subscriber.mu.Lock()
		subscriber.active = false
		subscriber.mu.Unlock()
	}()

	for {
		select {
		case <-subscriber.ctx.Done():
			return
		default:
			// Receive message with timeout
			ctx, cancel := context.WithTimeout(subscriber.ctx, pm.config.MessageTimeout)
			msg, err := subscriber.PubSub.ReceiveMessage(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					continue // Timeout, continue listening
				}
				if err == context.Canceled {
					return // Subscriber cancelled
				}
				logx.Error("Failed to receive message",
					logx.String("subscriber_id", subscriber.ID),
					logx.ErrorField(err))
				continue
			}

			// Process message
			pm.handleMessage(subscriber, msg)
		}
	}
}

// handleMessage handles a received message
func (pm *PubSubManager) handleMessage(subscriber *Subscriber, msg *redis.Message) {
	// Parse invalidation message
	var invalidationMsg InvalidationMessage
	if err := json.Unmarshal([]byte(msg.Payload), &invalidationMsg); err != nil {
		logx.Error("Failed to unmarshal invalidation message",
			logx.String("subscriber_id", subscriber.ID),
			logx.ErrorField(err))
		return
	}

	// Check if message is not too old
	if time.Since(invalidationMsg.Timestamp) > pm.config.MessageTimeout {
		logx.Warn("Received stale invalidation message",
			logx.String("subscriber_id", subscriber.ID),
			logx.String("message_id", invalidationMsg.MessageID),
			logx.String("age", time.Since(invalidationMsg.Timestamp).String()))
		return
	}

	// Call handler with keys
	if len(invalidationMsg.Keys) > 0 {
		subscriber.Handler(invalidationMsg.Keys...)
	}

	logx.Info("Processed invalidation message",
		logx.String("subscriber_id", subscriber.ID),
		logx.String("message_id", invalidationMsg.MessageID),
		logx.Int("keys", len(invalidationMsg.Keys)),
		logx.String("source", invalidationMsg.Source))
}

// startHealthMonitoring starts health monitoring
func (pm *PubSubManager) startHealthMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			// Publish health message
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := pm.PublishHealth(ctx, "cachex", "1.0.0", "healthy", map[string]interface{}{
				"subscribers": len(pm.subscribers),
				"timestamp":   time.Now().Unix(),
			})
			cancel()

			if err != nil {
				logx.Error("Failed to publish health message", logx.ErrorField(err))
			}
		}
	}
}

// GetStats returns pub/sub statistics
func (pm *PubSubManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["subscribers"] = len(pm.subscribers)
	stats["max_subscribers"] = pm.config.MaxSubscribers
	stats["invalidation_channel"] = pm.config.InvalidationChannel
	stats["health_channel"] = pm.config.HealthChannel

	// Count active subscribers
	activeCount := 0
	for _, subscriber := range pm.subscribers {
		subscriber.mu.RLock()
		if subscriber.active {
			activeCount++
		}
		subscriber.mu.RUnlock()
	}
	stats["active_subscribers"] = activeCount

	return stats
}

// Close closes the pub/sub manager
func (pm *PubSubManager) Close() error {
	pm.cancel()

	// Close all subscribers
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for subscriberID, subscriber := range pm.subscribers {
		subscriber.cancel()
		if subscriber.PubSub != nil {
			subscriber.PubSub.Close()
		}
		delete(pm.subscribers, subscriberID)
	}

	logx.Info("Closed pub/sub manager")
	return nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// generateSubscriberID generates a unique subscriber ID
func generateSubscriberID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}
