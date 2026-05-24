package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// StandardChange holds the schema definition for the StandardChange entity.
// 标准变更模板 - 预批准的低风险变更模板库
type StandardChange struct {
	ent.Schema
}

// Fields of the StandardChange.
func (StandardChange) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("模板标题").
			NotEmpty(),
		field.Text("description").
			Comment("模板描述").
			Optional(),
		field.Text("implementation_plan").
			Comment("实施计划步骤").
			NotEmpty(),
		field.Text("rollback_plan").
			Comment("回滚计划").
			NotEmpty(),
		field.Text("justification").
			Comment("变更理由").
			Optional(),
		field.String("category").
			Comment("分类：如服务器、网络、数据库、应用").
			Default("general"),
		field.String("risk_level").
			Comment("风险等级：low/medium/high").
			Default("low"),
		field.String("impact_scope").
			Comment("影响范围：low/medium/high").
			Default("low"),
		field.Int("expected_duration").
			Comment("预计工期（分钟）").
			Default(30),
		field.Bool("approval_required").
			Comment("是否需要审批（某些标准变更仍需审批）").
			Default(false),
		field.JSON("affected_cis", []string{}).
			Comment("典型受影响的配置项类型").
			Optional(),
		field.JSON("prerequisites", []string{}).
			Comment("前置条件清单").
			Optional(),
		field.Text("remarks").
			Comment("备注说明").
			Optional(),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the StandardChange.
func (StandardChange) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("changes", Change.Type).
			Comment("基于此模板创建的标准变更"),
	}
}
