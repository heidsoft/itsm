package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Ticket holds the schema definition for the Ticket entity.
type Ticket struct {
	ent.Schema
}

// Fields of the Ticket.
func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("工单标题").
			NotEmpty(),
		field.Text("description").
			Comment("工单描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("open"),
		field.String("type").
			Comment("工单类型").
			Default("incident"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("ticket_number").
			Comment("工单编号").
			Unique().
			NotEmpty(),
		field.Int("requester_id").
			Comment("申请人ID").
			Positive(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("template_id").
			Comment("模板ID").
			Optional(),
		field.Int("category_id").
			Comment("分类ID").
			Optional(),
		field.Int("department_id").
			Comment("部门ID").
			Optional(),
		field.Int("parent_ticket_id").
			Comment("父工单ID").
			Optional(),
		field.Int("sla_definition_id").
			Comment("SLA定义ID").
			Optional(),
		field.Time("sla_response_deadline").
			Comment("SLA响应截止时间").
			Optional(),
		field.Time("sla_resolution_deadline").
			Comment("SLA解决截止时间").
			Optional(),
		field.Time("first_response_at").
			Comment("首次响应时间").
			Optional(),
		field.Time("resolved_at").
			Comment("解决时间").
			Optional(),
		field.Text("resolution").
			Comment("解决方案").
			Optional(),
		field.String("resolution_category").
			Comment("解决方案分类").
			Optional(),
		field.Time("closed_at").
			Comment("关闭时间").
			Optional().
			Nillable(),
		field.Int("rating").
			Comment("评分（1-5星）").
			Optional().
			Range(1, 5),
		field.Text("rating_comment").
			Comment("评分评论").
			Optional(),
		field.Time("rated_at").
			Comment("评分时间").
			Optional(),
		field.Int("rated_by").
			Comment("评分人ID").
			Optional(),
		field.Int("version").
			Comment("版本号（乐观锁）").
			Default(1).
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Bool("is_managed_by_msp").
			Comment("是否由MSP管理").
			Default(false),
		field.Int("msp_provider_id").
			Comment("MSP服务提供商ID").
			Optional(),
		field.Int("managed_by_user_id").
			Comment("MSP处理人ID").
			Optional(),
		field.String("msp_ticket_id").
			Comment("MSP工单ID(跨租户)").
			Optional(),
		field.Time("deleted_at").
			Comment("删除时间").
			Optional().
			Nillable(),
	}
}

// Edges of the Ticket.
func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("comments", TicketComment.Type),
		edge.To("attachments", TicketAttachment.Type),
		edge.To("tags", TicketTag.Type),
		edge.To("related_tickets", Ticket.Type).
			Comment("双向关联工单"),
		edge.To("approval_records", ApprovalRecord.Type),
		edge.To("approvals", TicketApproval.Type),
		edge.To("workflow_records", TicketWorkflowRecord.Type),
		edge.To("notifications", TicketNotification.Type),
		edge.To("cc_users", TicketCC.Type),
		edge.To("sla_violations", SLAViolation.Type),
		edge.To("sla_alert_history", SLAAlertHistory.Type),
		edge.To("root_cause_analyses", RootCauseAnalysis.Type),
		edge.To("feishu_syncs", FeishuTicketSync.Type),
		edge.From("requester", User.Type).
			Ref("tickets").
			Field("requester_id").
			Required().
			Unique(),
		edge.From("assignee", User.Type).
			Ref("assigned_tickets").
			Field("assignee_id").
			Unique(),
		edge.From("category", TicketCategory.Type).
			Ref("tickets"),
	}
}

// Indexes of the Ticket.
func (Ticket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_number").Unique(),
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("type"), // Added index for type
		index.Fields("requester_id"),
		index.Fields("assignee_id"),
		index.Fields("created_at"),
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "status"),
		index.Fields("tenant_id", "requester_id"),
		index.Fields("status", "priority"),
		index.Fields("requester_id", "status"),
	}
}
