package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStreamConfig Redis Stream 配置
type RedisStreamConfig struct {
	Addr         string // Redis 地址，如 "localhost:6379"
	Password     string // 密码
	DB           int    // 数据库编号
	ConsumerGroup string // 消费者组名
	ConsumerName string // 消费者名称
}

// RedisStreamEventBus 基于 Redis Stream 的事件总线（生产环境推荐）
// 支持消息持久化、消费者组、消息确认等特性
type RedisStreamEventBus struct {
	redis         *redis.Client
	config        RedisStreamConfig
	handlers      map[string][]EventHandler
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	stopCh        chan struct{}
	processedIDs  map[string]bool // 简单幂等：记录已处理消息ID
}

func NewRedisStreamEventBus(config RedisStreamConfig) (*RedisStreamEventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	bus := &RedisStreamEventBus{
		redis: redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
		}),
		config:        config,
		handlers:      make(map[string][]EventHandler),
		ctx:           ctx,
		cancel:        cancel,
		stopCh:        make(chan struct{}),
		processedIDs:  make(map[string]bool),
	}
	
	// 测试连接
	if err := bus.redis.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	// 创建消费者组（如果不存在）
	bus.createConsumerGroupIfNeeded(ctx)
	
	log.Printf("[EVENT] RedisStreamEventBus initialized with group: %s", config.ConsumerGroup)
	return bus, nil
}

func (b *RedisStreamEventBus) createConsumerGroupIfNeeded(ctx context.Context) {
	// 为每个主题创建消费者组
	topics := []string{
		"itsm:ticket.created",
		"itsm:ticket.assigned",
		"itsm:ticket.status.changed",
		"itsm:sla.breached",
		"itsm:approval.completed",
		"itsm:ai.triage.completed",
	}
	
	for _, topic := range topics {
		// XPENDING 用于检查消费者组是否存在
		_, err := b.redis.XPending(ctx, topic, b.config.ConsumerGroup).Result()
		if err != nil {
			// 消费者组不存在，创建它
			err = b.redis.XGroupCreateMkStream(ctx, topic, b.config.ConsumerGroup, "0").Err()
			if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
				log.Printf("[EVENT] Failed to create consumer group for %s: %v", topic, err)
			}
		}
	}
}

func (b *RedisStreamEventBus) Publish(ctx context.Context, event DomainEvent) error {
	topic := fmt.Sprintf("itsm:%s", event.EventType())
	
	// 序列化事件
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// 构建消息
	values := map[string]interface{}{
		"tenant_id": event.TenantID(),
		"event_type": event.EventType(),
		"payload":   string(payload),
		"occurred_at": event.OccurredAt().Format(time.RFC3339),
	}
	
	// 添加到 Stream
	id := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), 0)
	err = b.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: values,
		ID:     id,
	}).Err()
	
	if err != nil {
		return fmt.Errorf("failed to publish event to Redis Stream: %w", err)
	}
	
	log.Printf("[EVENT] Published %s to %s (ID: %s)", event.EventType(), topic, id)
	return nil
}

func (b *RedisStreamEventBus) Subscribe(handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	topic := handler.Topic()
	b.handlers[topic] = append(b.handlers[topic], handler)
	
	log.Printf("[EVENT] Subscribed %s to topic %s", handler.Name(), topic)
	return nil
}

func (b *RedisStreamEventBus) Unsubscribe(handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	topic := handler.Topic()
	handlers := b.handlers[topic]
	
	for i, h := range handlers {
		if h.Name() == handler.Name() {
			b.handlers[topic] = append(handlers[:i], handlers[i+1:]...)
			log.Printf("[EVENT] Unsubscribed %s from topic %s", handler.Name(), topic)
			return nil
		}
	}
	
	return nil
}

func (b *RedisStreamEventBus) Start(ctx context.Context) error {
	log.Println("[EVENT] RedisStreamEventBus started")
	
	// 为每个主题启动消费者协程
	for topic := range b.handlers {
		b.wg.Add(1)
		go b.consumeLoop(ctx, topic)
	}
	
	// 等待停止信号
	select {
	case <-ctx.Done():
		return b.Stop()
	case <-b.stopCh:
		return nil
	}
}

func (b *RedisStreamEventBus) consumeLoop(ctx context.Context, topic string) {
	defer b.wg.Done()
	
	fullTopic := fmt.Sprintf("itsm:%s", topic)
	
	for {
		select {
		case <-b.ctx.Done():
			log.Printf("[EVENT] Stopping consumer for %s", topic)
			return
		default:
		}
		
		// 阻塞读取消息
		streams, err := b.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    b.config.ConsumerGroup,
			Consumer: b.config.ConsumerName,
			Streams:  []string{fullTopic, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		
		if err == redis.Nil {
			continue // 超时，继续循环
		}
		if err != nil {
			log.Printf("[EVENT] Error reading from stream %s: %v", topic, err)
			time.Sleep(time.Second)
			continue
		}
		
		// 处理消息
		for _, stream := range streams {
			for _, message := range stream.Messages {
				b.processMessage(ctx, topic, message)
			}
		}
	}
}

func (b *RedisStreamEventBus) processMessage(ctx context.Context, topic string, message redis.XMessage) {
	// 幂等检查
	b.mu.Lock()
	if b.processedIDs[message.ID] {
		b.mu.Unlock()
		// 已处理，确认消息
		b.redis.XAck(ctx, fmt.Sprintf("itsm:%s", topic), b.config.ConsumerGroup, message.ID)
		return
	}
	b.mu.Unlock()
	
	// 获取处理器
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers[topic]))
	copy(handlers, b.handlers[topic])
	b.mu.RUnlock()
	
	if len(handlers) == 0 {
		log.Printf("[EVENT] No handlers for topic %s, acking anyway", topic)
		b.redis.XAck(ctx, fmt.Sprintf("itsm:%s", topic), b.config.ConsumerGroup, message.ID)
		return
	}
	
	// 解析事件
	eventType := message.Values["event_type"].(string)
	payload := []byte(message.Values["payload"].(string))
	
	event, err := UnmarshalEvent(payload, eventType)
	if err != nil {
		log.Printf("[EVENT] Failed to unmarshal event: %v", err)
		b.redis.XAck(ctx, fmt.Sprintf("itsm:%s", topic), b.config.ConsumerGroup, message.ID)
		return
	}
	
	// 调用处理器
	for _, handler := range handlers {
		err := handler.Handle(ctx, event)
		if err != nil {
			log.Printf("[EVENT] Handler %s failed: %v", handler.Name(), err)
			// 可以选择不 ACK，让消息重新投递
		}
	}
	
	// 标记已处理并确认
	b.mu.Lock()
	b.processedIDs[message.ID] = true
	b.mu.Unlock()
	
	b.redis.XAck(ctx, fmt.Sprintf("itsm:%s", topic), b.config.ConsumerGroup, message.ID)
}

func (b *RedisStreamEventBus) Stop() error {
	select {
	case <-b.stopCh:
		return nil
	default:
	}
	
	log.Println("[EVENT] RedisStreamEventBus stopping...")
	b.cancel()
	b.wg.Wait()
	close(b.stopCh)
	log.Println("[EVENT] RedisStreamEventBus stopped")
	return nil
}

// Close 关闭 Redis 连接
func (b *RedisStreamEventBus) Close() error {
	return b.redis.Close()
}
