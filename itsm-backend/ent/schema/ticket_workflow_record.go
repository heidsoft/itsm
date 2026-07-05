package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// TicketWorkflowRecord holds the schema definition for the TicketWorkflowRecord entity.
type TicketWorkflowRecord struct {
	ent.Schema
}

// Mixin of the TicketWorkflowRecord
// 注意：不使用 mixin.Time{}，因为 SQL 迁移只定义了 created_at 字段
func (TicketWorkflowRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.CreateTime{},
	}
}

// Fields of the TicketWorkflowRecord.
func (TicketWorkflowRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID"),
		field.String("action").
			Comment("操作: accept/reject/approve/resolve/close/reopen/forward/cc/withdraw/delegate"),
		field.String("from_status").
			Optional().
			Comment("变更前状态"),
		field.String("to_status").
			Optional().
			Comment("变更后状态"),
		field.Int("operator_id").
			Comment("操作人ID"),
		field.Int("from_user_id").
			Optional().
			Comment("来源用户ID"),
		field.Int("to_user_id").
			Optional().
			Comment("目标用户ID"),
		field.String("comment").
			Optional().
			Comment("操作备注"),
		field.String("reason").
			Optional().
			Comment("操作原因"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("扩展元数据"),
		field.Int("tenant_id").
			Comment("租户ID"),
	}
}

// Edges of the TicketWorkflowRecord.
func (TicketWorkflowRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("workflow_records").
			Field("ticket_id").
			Unique().
			Required().
			Comment("所属工单"),
	}
}
