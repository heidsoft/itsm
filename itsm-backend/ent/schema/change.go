package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Change holds the schema definition for the Change entity.
type Change struct {
	ent.Schema
}

// Fields of the Change.
func (Change) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("变更标题").
			NotEmpty(),
		field.Text("description").
			Comment("变更描述").
			Optional(),
		field.Text("justification").
			Comment("变更理由").
			Optional(),
		field.String("type").
			Comment("变更类型").
			Default("normal"),
		field.String("status").
			Comment("状态").
			Default("draft"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("impact_scope").
			Comment("影响范围").
			Default("medium"),
		field.String("risk_level").
			Comment("风险等级").
			Default("medium"),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("planned_start_date").
			Comment("计划开始时间").
			Optional(),
		field.Time("planned_end_date").
			Comment("计划结束时间").
			Optional(),
		field.Time("actual_start_date").
			Comment("实际开始时间").
			Optional(),
		field.Time("actual_end_date").
			Comment("实际结束时间").
			Optional(),
		field.Text("implementation_plan").
			Comment("实施计划").
			Optional(),
		field.Text("rollback_plan").
			Comment("回滚计划").
			Optional(),
		field.JSON("affected_cis", []string{}).
			Comment("受影响的配置项").
			Optional(),
		field.JSON("related_tickets", []string{}).
			Comment("相关工单").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Change.
func (Change) Edges() []ent.Edge {
	return nil
}
