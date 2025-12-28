package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ApprovalRecord holds the schema definition for the ApprovalRecord entity.
type ApprovalRecord struct {
	ent.Schema
}

// Fields of the ApprovalRecord.
func (ApprovalRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID").
			Positive(),
		field.String("ticket_number").
			Comment("工单编号").
			NotEmpty(),
		field.String("ticket_title").
			Comment("工单标题").
			NotEmpty(),
		field.Int("workflow_id").
			Comment("工作流ID").
			Positive(),
		field.String("workflow_name").
			Comment("工作流名称").
			NotEmpty(),
		field.Int("current_level").
			Comment("当前审批级别").
			Default(1),
		field.Int("total_levels").
			Comment("总审批级别数").
			Default(1),
		field.Int("approver_id").
			Comment("审批人ID").
			Positive(),
		field.String("approver_name").
			Comment("审批人姓名").
			NotEmpty(),
		field.Int("step_order").
			Comment("节点顺序").
			Default(1),
		field.Time("due_date").
			Comment("到期时间").
			Optional().
			Nillable(),
		field.String("status").
			Comment("状态: pending, approved, rejected, delegated, timeout").
			Default("pending"),
		field.String("action").
			Comment("操作: approve, reject, delegate").
			Optional(),
		field.Text("comment").
			Comment("审批意见").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("processed_at").
			Comment("处理时间").
			Optional(),
	}
}

// Edges of the ApprovalRecord.
func (ApprovalRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("approval_records").
			Field("ticket_id").
			Unique().
			Required().
			Comment("关联的工单"),
		edge.From("workflow", ApprovalWorkflow.Type).
			Ref("approval_records").
			Field("workflow_id").
			Unique().
			Required().
			Comment("关联的工作流"),
	}
}
