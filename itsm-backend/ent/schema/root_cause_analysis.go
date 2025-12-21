package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// RootCauseAnalysis holds the schema definition for the RootCauseAnalysis entity.
type RootCauseAnalysis struct {
	ent.Schema
}

// Fields of the RootCauseAnalysis.
func (RootCauseAnalysis) Fields() []ent.Field {
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
		field.String("analysis_date").
			Comment("分析日期").
			NotEmpty(),
		field.JSON("root_causes", []map[string]interface{}{}).
			Comment("根因列表").
			Default([]map[string]interface{}{}),
		field.Text("analysis_summary").
			Comment("分析摘要").
			Optional(),
		field.Float("confidence_score").
			Comment("置信度分数(0-1)").
			Default(0).
			Min(0).
			Max(1),
		field.String("analysis_method").
			Comment("分析方法: automatic, manual, hybrid").
			Default("automatic"),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the RootCauseAnalysis.
func (RootCauseAnalysis) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("root_cause_analyses").
			Field("ticket_id").
			Unique().
			Required().
			Comment("关联的工单"),
	}
}
