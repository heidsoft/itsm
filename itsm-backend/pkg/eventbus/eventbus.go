package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"itsm-backend/config"
	"itsm-backend/handlers/shared"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// WatermillEventBus implements shared.EventBus interface using Watermill with Redis Stream
type WatermillEventBus struct {
	publisher  message.Publisher
	subscriber *redisstream.Subscriber
	logger     *zap.SugaredLogger
}

// NewWatermillEventBus creates a new WatermillEventBus instance
func NewWatermillEventBus(cfg *config.RedisConfig, logger *zap.SugaredLogger) (*WatermillEventBus, error) {
	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Watermill logger
	watermillLogger := NewZapLoggerAdapter(logger)

	// Create publisher
	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: rdb,
		},
		watermillLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	// Create subscriber
	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client: rdb,
		},
		watermillLogger,
	)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	return &WatermillEventBus{
		publisher:  publisher,
		subscriber: subscriber,
		logger:     logger,
	}, nil
}

// Publish publishes an event to the event bus
func (eb *WatermillEventBus) Publish(event interface{}) error {
	// Marshal event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Get event type from the event struct name
	eventType := fmt.Sprintf("%T", event)

	// Create message
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("event_type", eventType)

	// Publish to Redis Stream
	if err := eb.publisher.Publish(eventType, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	eb.logger.Debugw("Event published", "event_type", eventType, "event_id", msg.UUID)
	return nil
}

// Subscribe subscribes to events of a specific type
func (eb *WatermillEventBus) Subscribe(eventType string, handler shared.EventHandler) error {
	// Subscribe to the topic
	messages, err := eb.subscriber.Subscribe(context.Background(), eventType)
	if err != nil {
		return fmt.Errorf("failed to subscribe to event type %s: %w", eventType, err)
	}

	// Start message processing goroutine
	go func() {
		for msg := range messages {
			// Unmarshal payload
			var event interface{}
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				eb.logger.Errorw("Failed to unmarshal event payload", "event_type", eventType, "error", err)
				msg.Nack()
				continue
			}

			// Call handler
			if err := handler.Handle(event); err != nil {
				eb.logger.Errorw("Failed to handle event", "event_type", eventType, "error", err)
				msg.Nack()
				continue
			}

			// Acknowledge message
			msg.Ack()
		}
	}()

	eb.logger.Infow("Subscribed to event type", "event_type", eventType)
	return nil
}

// Close closes the event bus
func (eb *WatermillEventBus) Close() error {
	if err := eb.publisher.Close(); err != nil {
		return err
	}
	if err := eb.subscriber.Close(); err != nil {
		return err
	}
	return nil
}

// Global event bus instance
var globalEventBus shared.EventBus

// SetGlobalEventBus sets the global event bus instance
func SetGlobalEventBus(eb shared.EventBus) {
	globalEventBus = eb
}

// GetGlobalEventBus returns the global event bus instance
func GetGlobalEventBus() shared.EventBus {
	return globalEventBus
}
