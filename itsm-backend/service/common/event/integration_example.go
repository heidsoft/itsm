package event

import (
	"context"
	"fmt"
	"time"
	
	"itsm-backend/service/common"
	"itsm-backend/service/dto"
)

// ===== 集成示例：将现有 TicketService 改造为事件驱动 =====

/*
// 改造前（同步调用）:
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ticket.Ticket, error) {
    // 1. 创建工单
    tkt, err := s.repo.Create(ctx, params, tenantID)
    if err != nil {
        return nil, err
    }
    
    // 2. 同步计算SLA（阻塞）
    s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, ...)
    
    // 3. 同步触发审批（阻塞）
    s.approvalSvc.TriggerApproval(ctx, ...)
    
    // 4. 异步发送通知（goroutine）
    go s.notificationSvc.SendTicketCreatedNotification(ctx, tkt)
    
    return tkt, nil
}

// 改造后（事件驱动）:
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ticket.Ticket, error) {
    // 1. 创建工单
    tkt, err := s.repo.Create(ctx, params, tenantID)
    if err != nil {
        return nil, err
    }
    
    // 2. 发布领域事件（异步处理交给订阅者）
    s.eventBus.Publish(ctx, common.NewTicketCreatedEvent(
        fmt.Sprintf("%d", tenantID),  // tenantID
        fmt.Sprintf("%d", tkt.ID),   // ticketID
        tkt.TicketNumber,
        tkt.Title,
        string(tkt.Priority),
        fmt.Sprintf("%d", tkt.RequesterID),
    ))
    
    return tkt, nil
}

// 事件处理器（独立模块）：
type SLAMonitorHandler struct {
    slaService SLAService
    repo       TicketRepository
}

func (h *SLAMonitorHandler) Topic() string    { return "ticket.created" }
func (h *SLAMonitorHandler) Priority() int   { return 10 } // 高优先级
func (h *SLAMonitorHandler) Name() string    { return "sla-monitor" }

func (h *SLAMonitorHandler) Handle(ctx context.Context, event DomainEvent) error {
    e := event.(*common.TicketCreatedEvent)
    
    // 计算 SLA 截止时间
    result, err := h.slaService.CalculateSLADeadlineFromRequest(ctx, 
        e.TenantID, e.TicketNumber, e.Priority)
    if err != nil {
        return err
    }
    
    // 更新工单
    return h.repo.UpdateSLADeadlines(ctx, e.TicketID, 
        result.ResponseDeadline, result.ResolutionDeadline, tenantID)
}

type ApprovalHandler struct {
    approvalService ApprovalService
}

func (h *ApprovalHandler) Topic() string    { return "ticket.created" }
func (h *ApprovalHandler) Priority() int   { return 20 }
func (h *ApprovalHandler) Name() string    { return "approval-trigger" }

func (h *ApprovalHandler) Handle(ctx context.Context, event DomainEvent) error {
    e := event.(*common.TicketCreatedEvent)
    
    _, err := h.approvalService.TriggerApproval(ctx, &ApprovalTriggerRequest{
        TicketID:     e.TicketID,
        TicketNumber: e.TicketNumber,
        TicketTitle:  e.Title,
        TenantID:     e.TenantID,
    })
    return err
}

type NotificationHandler struct {
    notificationService NotificationService
}

func (h *NotificationHandler) Topic() string    { return "ticket.created" }
func (h *NotificationHandler) Priority() int   { return 30 }
func (h *NotificationHandler) Name() string    { return "notification" }

func (h *NotificationHandler) Handle(ctx context.Context, event DomainEvent) error {
    e := event.(*common.TicketCreatedEvent)
    return h.notificationService.SendTicketCreatedNotification(ctx, e.TicketID, e.TenantID)
}
*/

// ===== 完整集成代码示例 =====

// TicketEventService 事件驱动的工单服务
type TicketEventService struct {
	repo      TicketRepository
	eventBus  EventBus
	slaSvc    SLAService
	approvalSvc ApprovalService
	notifSvc  NotificationService
}

// NewTicketEventService 创建事件驱动的工单服务
func NewTicketEventService(repo TicketRepository, eventBus EventBus) *TicketEventService {
	svc := &TicketEventService{
		repo:     repo,
		eventBus: eventBus,
	}
	
	// 注册事件处理器
	eventBus.Subscribe(&slaMonitorHandler{svc: svc})
	eventBus.Subscribe(&approvalHandler{svc: svc})
	eventBus.Subscribe(&notificationHandler{svc: svc})
	
	return svc
}

// CreateTicket 创建工单（改造后）
func (s *TicketEventService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (interface{}, error) {
	// 1. 创建工单
	tkt, err := s.repo.Create(ctx, &ticket.CreateParams{
		Title:       req.Title,
		Description: req.Description,
		Priority:    ticket.Priority(req.Priority),
	}, tenantID)
	if err != nil {
		return nil, err
	}
	
	// 2. 发布领域事件（异步处理）
	s.eventBus.Publish(ctx, NewTicketCreatedEvent(
		fmt.Sprintf("%d", tenantID),
		fmt.Sprintf("%d", tkt.ID),
		tkt.TicketNumber,
		tkt.Title,
		string(tkt.Priority),
		fmt.Sprintf("%d", req.RequesterID),
	))
	
	return tkt, nil
}

// ===== 事件处理器实现 =====

type slaMonitorHandler struct {
	svc *TicketEventService
}

func (h *slaMonitorHandler) Topic() string     { return "ticket.created" }
func (h *slaMonitorHandler) Priority() int      { return 10 }
func (h *slaMonitorHandler) Name() string       { return "sla-monitor" }

func (h *slaMonitorHandler) Handle(ctx context.Context, event DomainEvent) error {
	e := event.(*TicketCreatedEvent)
	
	// 模拟 SLA 计算
	deadline := time.Now().Add(4 * time.Hour)
	_ = deadline // 使用
	
	return nil // 简化示例
}

type approvalHandler struct {
	svc *TicketEventService
}

func (h *approvalHandler) Topic() string     { return "ticket.created" }
func (h *approvalHandler) Priority() int      { return 20 }
func (h *approvalHandler) Name() string       { return "approval-trigger" }

func (h *approvalHandler) Handle(ctx context.Context, event DomainEvent) error {
	e := event.(*TicketCreatedEvent)
	
	// 触发审批流程
	_ = e.TicketID
	
	return nil // 简化示例
}

type notificationHandler struct {
	svc *TicketEventService
}

func (h *notificationHandler) Topic() string     { return "ticket.created" }
func (h *notificationHandler) Priority() int      { return 30 }
func (h *notificationHandler) Name() string       { return "notification" }

func (h *notificationHandler) Handle(ctx context.Context, event DomainEvent) error {
	e := event.(*TicketCreatedEvent)
	
	// 发送通知
	_ = e.TicketID
	
	return nil // 简化示例
}

// ===== 接口定义（用于依赖注入）=====

type TicketRepository interface {
	Create(ctx context.Context, params interface{}, tenantID int) (interface{}, error)
}

type SLAService interface {
	CalculateSLADeadlineFromRequest(ctx context.Context, tenantID, ticketType, priority string) (interface{}, error)
}

type ApprovalService interface {
	TriggerApproval(ctx context.Context, req interface{}) (interface{}, error)
}

type NotificationService interface {
	SendTicketCreatedNotification(ctx context.Context, ticketID, tenantID string) error
}

// Placeholder types to make the code compile
type ticket struct {
	ID           int
	TicketNumber string
	Title        string
	Priority     string
	RequesterID  int
	Type         string
}
