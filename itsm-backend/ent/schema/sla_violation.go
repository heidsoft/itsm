package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// SLAViolation holds the schema definition for the SLAViolation entity.
type SLAViolation struct {
	ent.Schema
}

// Fields of the SLAViolation.
func (SLAViolation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").Comment("工单ID"),
		field.String("ticket_type").Comment("工单类型：incident/service_request/problem/change"),
		field.String("violation_type").Comment("违规类型：response_time/resolution_time"),
		field.Int("sla_definition_id").Comment("SLA定义ID"),
		field.String("sla_name").Comment("SLA名称"),
		field.Int("expected_time").Comment("预期时间（分钟）"),
		field.Int("actual_time").Comment("实际时间（分钟）"),
		field.Int("overdue_minutes").Comment("超时时间（分钟）"),
		field.String("status").Comment("状态：pending/notified/escalated/resolved"),
		field.String("assigned_to").Optional().Comment("处理人"),
		field.Time("violation_occurred_at").Comment("违规发生时间"),
		field.Time("resolved_at").Optional().Comment("解决时间"),
		field.String("resolution_note").Optional().Comment("解决说明"),
		field.Int("tenant_id").Comment("租户ID"),
		field.String("created_by").Comment("创建人"),
		field.Time("created_at").Comment("创建时间"),
		field.Time("updated_at").Comment("更新时间"),
	}
}

// Edges of the SLAViolation.
func (SLAViolation) Edges() []ent.Edge {
	return nil
}

// Indexes of the SLAViolation.
func (SLAViolation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "ticket_id"),
		index.Fields("tenant_id", "status"),
		index.Fields("tenant_id", "violation_occurred_at"),
		index.Fields("tenant_id", "assigned_to"),
	}
}

// Mixin of the SLAViolation.
func (SLAViolation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}
