package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// SLAMetrics holds the schema definition for the SLAMetrics entity.
type SLAMetrics struct {
	ent.Schema
}

// Fields of the SLAMetrics.
func (SLAMetrics) Fields() []ent.Field {
	return []ent.Field{
		field.String("service_type").Comment("服务类型"),
		field.String("priority").Comment("优先级"),
		field.String("impact").Comment("影响范围"),
		field.Int("total_tickets").Comment("总工单数"),
		field.Int("met_sla_tickets").Comment("达标工单数"),
		field.Int("violated_sla_tickets").Comment("违规工单数"),
		field.Float("sla_compliance_rate").Comment("SLA达标率"),
		field.Float("avg_response_time").Comment("平均响应时间（分钟）"),
		field.Float("avg_resolution_time").Comment("平均解决时间（分钟）"),
		field.String("period").Comment("统计周期：daily/weekly/monthly"),
		field.Time("period_start").Comment("周期开始时间"),
		field.Time("period_end").Comment("周期结束时间"),
		field.Int("tenant_id").Comment("租户ID"),
		field.Time("created_at").Comment("创建时间"),
		field.Time("updated_at").Comment("更新时间"),
	}
}

// Edges of the SLAMetrics.
func (SLAMetrics) Edges() []ent.Edge {
	return nil
}

// Indexes of the SLAMetrics.
func (SLAMetrics) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "service_type", "priority", "impact"),
		index.Fields("tenant_id", "period", "period_start"),
	}
}

// Mixin of the SLAMetrics.
func (SLAMetrics) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}
