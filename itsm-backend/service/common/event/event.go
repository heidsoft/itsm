// Package common 提供通用基础设施
// 按 DDD 思想将共享能力抽取到 common 包
package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DomainEvent 领域事件接口 - 所有事件的根接口
type DomainEvent interface {
	EventType() string       // 事件类型，如 "ticket.created"
	TenantID() string        // 租户 ID，用于多租户隔离
	OccurredAt() time.Time   // 事件发生时间
	Payload() interface{}    // 事件载荷
}

// BaseEvent 基础事件实现 - 简化事件定义
type BaseEvent struct {
	eventType  string
	tenantID   string
	occurredAt time.Time
	payload    interface{}
}

func (e *BaseEvent) EventType() string  { return e.eventType }
func (e *BaseEvent) TenantID() string   { return e.tenantID }
func (e *BaseEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *BaseEvent) Payload() interface{} { return e.payload }

// NewBaseEvent 创建基础事件的工厂函数
func NewBaseEvent(eventType, tenantID string, payload interface{}) *BaseEvent {
	return &BaseEvent{
		eventType:  eventType,
		tenantID:   tenantID,
		occurredAt: time.Now(),
		payload:    payload,
	}
}

// ===== 具体事件定义 =====

// TicketCreatedEvent 工单创建事件
type TicketCreatedEvent struct {
	*BaseEvent
	TicketID     string `json:"ticket_id"`
	TicketNumber string `json:"ticket_number"`
	Title        string `json:"title"`
	Priority     string `json:"priority"`
	CreatedBy    string `json:"created_by"`
}

func NewTicketCreatedEvent(tenantID, ticketID, ticketNumber, title, priority, createdBy string) *TicketCreatedEvent {
	return &TicketCreatedEvent{
		BaseEvent:    NewBaseEvent("ticket.created", tenantID, nil),
		TicketID:     ticketID,
		TicketNumber: ticketNumber,
		Title:        title,
		Priority:     priority,
		CreatedBy:    createdBy,
	}
}

// TicketAssignedEvent 工单分配事件
type TicketAssignedEvent struct {
	*BaseEvent
	TicketID   string `json:"ticket_id"`
	AssigneeID string `json:"assignee_id"`
	Assignee   string `json:"assignee"`
}

func NewTicketAssignedEvent(tenantID, ticketID, assigneeID, assignee string) *TicketAssignedEvent {
	return &TicketAssignedEvent{
		BaseEvent:  NewBaseEvent("ticket.assigned", tenantID, nil),
		TicketID:   ticketID,
		AssigneeID: assigneeID,
		Assignee:   assignee,
	}
}

// TicketStatusChangedEvent 工单状态变更事件
type TicketStatusChangedEvent struct {
	*BaseEvent
	TicketID    string `json:"ticket_id"`
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	ChangedBy   string `json:"changed_by"`
}

func NewTicketStatusChangedEvent(tenantID, ticketID, oldStatus, newStatus, changedBy string) *TicketStatusChangedEvent {
	return &TicketStatusChangedEvent{
		BaseEvent:   NewBaseEvent("ticket.status.changed", tenantID, nil),
		TicketID:    ticketID,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		ChangedBy:   changedBy,
	}
}

// SLABreachedEvent SLA 违规事件
type SLABreachedEvent struct {
	*BaseEvent
	TicketID     string `json:"ticket_id"`
	SLAPolicyID  string `json:"sla_policy_id"`
	BreachedType string `json:"breached_type"` // "response" or "resolve"
	BreachedAt   time.Time `json:"breached_at"`
}

func NewSLABreachedEvent(tenantID, ticketID, slaPolicyID, breachedType string, breachedAt time.Time) *SLABreachedEvent {
	return &SLABreachedEvent{
		BaseEvent:    NewBaseEvent("sla.breached", tenantID, nil),
		TicketID:     ticketID,
		SLAPolicyID:  slaPolicyID,
		BreachedType: breachedType,
		BreachedAt:   breachedAt,
	}
}

// ApprovalCompletedEvent 审批完成事件
type ApprovalCompletedEvent struct {
	*BaseEvent
	ApprovalID    string `json:"approval_id"`
	EntityType   string `json:"entity_type"`   // "ticket", "change"
	EntityID      string `json:"entity_id"`
	ApprovalResult string `json:"approval_result"` // "approved", "rejected"
	ApprovedBy   string `json:"approved_by"`
}

func NewApprovalCompletedEvent(tenantID, approvalID, entityType, entityID, result, approvedBy string) *ApprovalCompletedEvent {
	return &ApprovalCompletedEvent{
		BaseEvent:      NewBaseEvent("approval.completed", tenantID, nil),
		ApprovalID:     approvalID,
		EntityType:     entityType,
		EntityID:       entityID,
		ApprovalResult: result,
		ApprovedBy:     approvedBy,
	}
}

// AITriageCompletedEvent AI 分诊完成事件
type AITriageCompletedEvent struct {
	*BaseEvent
	TicketID     string `json:"ticket_id"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	Confidence   float64 `json:"confidence"`
	SuggestedAssigneeID string `json:"suggested_assignee_id"`
}

func NewAITriageCompletedEvent(tenantID, ticketID, categoryID, categoryName, suggestedAssigneeID string, confidence float64) *AITriageCompletedEvent {
	return &AITriageCompletedEvent{
		BaseEvent:            NewBaseEvent("ai.triage.completed", tenantID, nil),
		TicketID:             ticketID,
		CategoryID:           categoryID,
		CategoryName:         categoryName,
		Confidence:           confidence,
		SuggestedAssigneeID:  suggestedAssigneeID,
	}
}

// ===== 事件序列化工具 =====

// MarshalEvent 将事件序列化为 JSON 字节
func MarshalEvent(event DomainEvent) ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalEvent 将 JSON 反序列化为事件
func UnmarshalEvent(data []byte, eventType string) (DomainEvent, error) {
	switch eventType {
	case "ticket.created":
		var e TicketCreatedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	case "ticket.assigned":
		var e TicketAssignedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	case "ticket.status.changed":
		var e TicketStatusChangedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	case "sla.breached":
		var e SLABreachedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	case "approval.completed":
		var e ApprovalCompletedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	case "ai.triage.completed":
		var e AITriageCompletedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		e.eventType = eventType
		return &e, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}
