package event

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// 示例：如何在现有服务中集成事件总线

/*
========== 使用示例 ==========

1. 初始化事件总线（在 main.go 或 wire 中）:

var (
	EventBusInstance EventBus
)

func initEventBus() {
	// 开发环境：使用内存事件总线
	EventBusInstance = NewInMemoryEventBus()
	
	// 或者生产环境：使用 Redis Stream
	// EventBusInstance, _ = NewRedisStreamEventBus(RedisStreamConfig{
	// 	Addr:         "localhost:6379",
	// 	ConsumerGroup: "itsm-consumers",
	// 	ConsumerName:  "worker-1",
	// })
	
	// 注册处理器
	RegisterEventHandlers(EventBusInstance)
}

func RegisterEventHandlers(bus EventBus) {
	// AI 分诊处理器
	bus.Subscribe(NewFuncHandler("ticket.created", "ai-triage", 10, func(ctx context.Context, event DomainEvent) error {
		e := event.(*TicketCreatedEvent)
		return handleAITriage(ctx, e.TenantID, e.TicketID, e.Title)
	}))
	
	// SLA 监控处理器
	bus.Subscribe(NewFuncHandler("ticket.created", "sla-monitor", 20, func(ctx context.Context, event DomainEvent) error {
		e := event.(*TicketCreatedEvent)
		return handleSLAMonitoring(ctx, e.TenantID, e.TicketID, e.Priority)
	}))
	
	// 通知处理器
	bus.Subscribe(NewFuncHandler("ticket.created", "notification", 30, func(ctx context.Context, event DomainEvent) error {
		e := event.(*TicketCreatedEvent)
		return handleNotification(ctx, e.TenantID, "ticket_created", e.TicketID)
	}))
}

2. 在领域服务中发布事件:

func (s *TicketService) CreateTicket(ctx context.Context, req CreateTicketRequest) (*Ticket, error) {
	// 1. 业务逻辑
	ticket := &Ticket{...}
	ticket, err := s.repo.Create(ctx, ticket)
	if err != nil {
		return nil, err
	}
	
	// 2. 发布领域事件
	err = EventBusInstance.Publish(ctx, NewTicketCreatedEvent(
		req.TenantID,
		ticket.ID,
		ticket.TicketNumber,
		ticket.Title,
		ticket.Priority,
		req.CreatedBy,
	))
	if err != nil {
		log.Printf("Failed to publish event: %v", err)
		// 不返回错误，确保主流程成功
	}
	
	return ticket, nil
}

3. 处理器实现示例:

func handleAITriage(ctx context.Context, tenantID, ticketID, title string) error {
	log.Printf("[AI-Triage] Processing ticket %s from tenant %s", ticketID, tenantID)
	
	// 调用 AI 服务进行分诊
	category, assignee, err := aiService.Triage(ctx, title)
	if err != nil {
		return err
	}
	
	// 更新工单分类
	repo.UpdateTicketCategory(ctx, ticketID, category)
	
	// 发布分诊完成事件
	EventBusInstance.Publish(ctx, NewAITriageCompletedEvent(
		tenantID,
		ticketID,
		category.ID,
		category.Name,
		assignee.ID,
		assignee.Confidence,
	))
	
	return nil
}
*/

// TestInMemoryEventBus 内存事件总线测试
func TestInMemoryEventBus(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	
	// 注册测试处理器
	bus.Subscribe(NewFuncHandler("ticket.created", "test-handler-1", 1, func(ctx context.Context, event DomainEvent) error {
		log.Printf("Handler 1 received: %s", event.EventType())
		return nil
	}))
	
	// 测试发布
	err := bus.Publish(ctx, NewTicketCreatedEvent("tenant-1", "ticket-1", "T-001", "Test Ticket", "high", "user-1"))
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}
	
	// 等待异步处理
	time.Sleep(100 * time.Millisecond)
	
	// 验证处理器数量
	count := bus.GetHandlerCount("ticket.created")
	if count != 1 {
		t.Errorf("Expected 1 handler, got %d", count)
	}
	
	cancel()
	bus.Stop()
}

// ExampleInMemoryEventBus 事件总线使用示例
func ExampleInMemoryEventBus() {
	bus := NewInMemoryEventBus()
	ctx := context.Background()
	
	// 订阅
	bus.Subscribe(NewFuncHandler("ticket.created", "handler-1", 1, func(ctx context.Context, event DomainEvent) error {
		e := event.(*TicketCreatedEvent)
		fmt.Printf("Processed ticket: %s\n", e.TicketNumber)
		return nil
	}))
	
	// 发布
	bus.Publish(ctx, NewTicketCreatedEvent("tenant-1", "1", "T-100", "Help", "medium", "admin"))
	
	// 等待处理
	time.Sleep(50 * time.Millisecond)
	
	bus.Stop()
	
	// Output:
	// Processed ticket: T-100
}
