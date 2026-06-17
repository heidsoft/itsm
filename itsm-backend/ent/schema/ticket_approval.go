package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketApproval holds the schema definition for the TicketApproval entity.
type TicketApproval struct {
	ent.Schema
}

// Fields of the TicketApproval.
func (TicketApproval) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID"),
		field.Int("level").
			Comment("审批级别（1=一级,2=二级...）"),
		field.String("level_name").
			Comment("级别名称"),
		field.Int("approver_id").
			Comment("审批人ID"),
		field.String("status").
			Default("pending").
			Comment("审批状态: pending/approved/rejected/cancelled"),
		field.String("action").
			Optional().
			Comment("审批动作: approve/reject/delegate"),
		field.String("comment").
			Optional().
			Comment("审批意见"),
		field.Int("delegate_to_user_id").
			Optional().
			Comment("委派目标用户ID"),
		field.Int("tenant_id").
			Comment("租户ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
		field.Time("processed_at").
			Optional().
			Nillable().
			Comment("处理时间"),
	}
}

// Edges of the TicketApproval.
func (TicketApproval) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("approvals").
			Field("ticket_id").
			Unique().
			Required().
			Comment("所属工单"),
	}
}
