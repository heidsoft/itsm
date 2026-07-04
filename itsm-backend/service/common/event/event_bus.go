package event

import (
	"context"
	"log"
	"sync"
	"time"
)

// EventHandler 事件处理器接口
type EventHandler interface {
	Topic() string // 关注的事件类型
	Handle(ctx context.Context, event DomainEvent) error
	Priority() int // 处理优先级，数字越小越优先
	Name() string  // 处理器名称，用于日志
}

// BaseHandler 基础处理器实现
type BaseHandler struct {
	topic    string
	priority int
	name     string
}

func (h *BaseHandler) Topic() string { return h.topic }
func (h *BaseHandler) Priority() int { return h.priority }
func (h *BaseHandler) Name() string  { return h.name }

// FuncHandler 函数式处理器 - 方便快速创建简单处理器
type FuncHandler struct {
	BaseHandler
	handler func(ctx context.Context, event DomainEvent) error
}

func NewFuncHandler(topic, name string, priority int, handler func(ctx context.Context, event DomainEvent) error) *FuncHandler {
	return &FuncHandler{
		BaseHandler: BaseHandler{topic: topic, priority: priority, name: name},
		handler:     handler,
	}
}

func (h *FuncHandler) Handle(ctx context.Context, event DomainEvent) error {
	return h.handler(ctx, event)
}

// EventBus 事件总线接口
type EventBus interface {
	// Publish 发布事件
	Publish(ctx context.Context, event DomainEvent) error
	// Subscribe 订阅事件
	Subscribe(handler EventHandler) error
	// Unsubscribe 取消订阅
	Unsubscribe(handler EventHandler) error
	// Start 启动消费者（阻塞）
	Start(ctx context.Context) error
	// Stop 停止消费者
	Stop() error
}

// InMemoryEventBus 内存事件总线实现（适用于开发/小规模）
// 生产环境建议使用 Redis Stream 或 Kafka 版本
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	closed   chan struct{}
}

func NewInMemoryEventBus() *InMemoryEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
		ctx:      ctx,
		cancel:   cancel,
		closed:   make(chan struct{}),
	}
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event DomainEvent) error {
	topic := event.EventType()

	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers[topic]))
	copy(handlers, b.handlers[topic])
	b.mu.RUnlock()

	// 按优先级排序
	for i := 0; i < len(handlers)-1; i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].Priority() > handlers[j].Priority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}

	// 异步执行所有处理器
	for _, handler := range handlers {
		h := handler // 避免闭包捕获问题
		go func() {
			start := time.Now()
			err := h.Handle(ctx, event)
			duration := time.Since(start)

			if err != nil {
				log.Printf("[EVENT] Handler %s failed for %s: %v (duration: %v)",
					h.Name(), topic, err, duration)
			} else {
				log.Printf("[EVENT] Handler %s processed %s (duration: %v)",
					h.Name(), topic, duration)
			}
		}()
	}

	return nil
}

func (b *InMemoryEventBus) Subscribe(handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	topic := handler.Topic()
	b.handlers[topic] = append(b.handlers[topic], handler)

	log.Printf("[EVENT] Subscribed %s to topic %s", handler.Name(), topic)
	return nil
}

func (b *InMemoryEventBus) Unsubscribe(handler EventHandler) error {
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

func (b *InMemoryEventBus) Start(ctx context.Context) error {
	log.Println("[EVENT] EventBus started")
	<-ctx.Done()
	return b.Stop()
}

func (b *InMemoryEventBus) Stop() error {
	select {
	case <-b.closed:
		return nil // 已经关闭
	default:
	}

	log.Println("[EVENT] EventBus stopping...")
	b.cancel()
	b.wg.Wait()
	close(b.closed)
	log.Println("[EVENT] EventBus stopped")
	return nil
}

// GetHandlerCount 获取主题的处理器数量（用于测试）
func (b *InMemoryEventBus) GetHandlerCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[topic])
}
