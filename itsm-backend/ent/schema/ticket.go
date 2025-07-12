package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// TicketStatus defines the status enum for tickets
type TicketStatus string

const (
	TicketStatusDraft      TicketStatus = "draft"       // 草稿
	TicketStatusSubmitted  TicketStatus = "submitted"   // 已提交
	TicketStatusInProgress TicketStatus = "in_progress" // 处理中
	TicketStatusPending    TicketStatus = "pending"     // 待审批
	TicketStatusApproved   TicketStatus = "approved"    // 已批准
	TicketStatusRejected   TicketStatus = "rejected"    // 已拒绝
	TicketStatusClosed     TicketStatus = "closed"      // 已关闭
	TicketStatusCancelled  TicketStatus = "cancelled"   // 已取消
)

// TicketPriority defines the priority enum for tickets
type TicketPriority string

const (
	TicketPriorityLow      TicketPriority = "low"      // 低
	TicketPriorityMedium   TicketPriority = "medium"   // 中
	TicketPriorityHigh     TicketPriority = "high"     // 高
	TicketPriorityCritical TicketPriority = "critical" // 紧急
)

// Ticket holds the schema definition for the Ticket entity.
type Ticket struct {
	ent.Schema
}

// Fields of the Ticket.
func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			NotEmpty().
			MaxLen(255).
			Comment("工单标题"),
		field.Text("description").
			Optional().
			Comment("工单描述"),
		field.Enum("status").
			Values(
				string(TicketStatusDraft),
				string(TicketStatusSubmitted),
				string(TicketStatusInProgress),
				string(TicketStatusPending),
				string(TicketStatusApproved),
				string(TicketStatusRejected),
				string(TicketStatusClosed),
				string(TicketStatusCancelled),
			).
			Default(string(TicketStatusDraft)).
			Comment("工单状态"),
		field.Enum("priority").
			Values(
				string(TicketPriorityLow),
				string(TicketPriorityMedium),
				string(TicketPriorityHigh),
				string(TicketPriorityCritical),
			).
			Default(string(TicketPriorityMedium)).
			Comment("优先级"),
		field.JSON("form_fields", map[string]interface{}{}).
			Optional().
			Comment("表单字段JSON数据"),
		field.String("ticket_number").
			Unique().
			NotEmpty().
			MaxLen(50).
			Comment("工单编号"),
		field.Int("requester_id").
			Positive().
			Comment("申请人ID"),
		field.Int("assignee_id").
			Optional().
			Nillable().
			Positive().
			Comment("处理人ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges of the Ticket.
func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		// 申请人
		edge.From("requester", User.Type).
			Ref("submitted_tickets").
			Field("requester_id").
			Required().
			Unique(),
		// 处理人
		edge.From("assignee", User.Type).
			Ref("assigned_tickets").
			Field("assignee_id").
			Unique(),
		// 审批记录
		edge.To("approval_logs", ApprovalLog.Type),
		// 流程实例
		edge.To("flow_instance", FlowInstance.Type).
			Unique(),
		// 状态变更日志
		edge.To("status_logs", StatusLog.Type),
	}
}

// Indexes of the Ticket.
func (Ticket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("requester_id"),
		index.Fields("assignee_id"),
		index.Fields("created_at"),
		index.Fields("ticket_number"),
		// 复合索引
		index.Fields("status", "priority"),
		index.Fields("requester_id", "status"),
	}
}
