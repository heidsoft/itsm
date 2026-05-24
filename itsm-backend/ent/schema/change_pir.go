package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ChangePIR holds the schema definition for the ChangePIR entity.
// ChangePIR represents Post-Implementation Review for changes (ITIL best practice)
type ChangePIR struct {
	ent.Schema
}

// Fields of the ChangePIR.
func (ChangePIR) Fields() []ent.Field {
	return []ent.Field{
		// Note: change_id is auto-generated from the 'change' edge
		field.Int("reviewer_id").
			Comment("审查人ID").
			Optional(),
		field.String("overall_result").
			Comment("总体结果: successful, partially_successful, failed").
			Default("successful"),
		field.Bool("objectives_achieved").
			Comment("目标是否达成").
			Default(false),
		field.Text("success_summary").
			Comment("成功总结").
			Optional(),
		field.Text("issues_encountered").
			Comment("遇到的问题").
			Optional(),
		field.Text("lessons_learned").
			Comment("经验教训").
			Optional(),
		field.Text("improvement_recommendations").
			Comment("改进建议").
			Optional(),
		field.Time("actual_start_time").
			Comment("实际开始时间").
			Optional(),
		field.Time("actual_end_time").
			Comment("实际结束时间").
			Optional(),
		field.Int("actual_duration_minutes").
			Comment("实际持续时间（分钟）").
			Default(0),
		field.Bool("rollback_performed").
			Comment("是否执行了回滚").
			Default(false),
		field.Text("rollback_reason").
			Comment("回滚原因").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("review_date").
			Comment("审查日期").
			Default(time.Now),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the ChangePIR.
func (ChangePIR) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("change", Change.Type).
			Ref("pir").
			Required().
			Unique().
			Comment("关联的变更"),
	}
}
