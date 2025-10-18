package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type IncidentRule struct{ ent.Schema }

func (IncidentRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("规则名称").NotEmpty(),
		field.Text("description").Comment("规则描述").Optional(),
		field.String("rule_type").Comment("规则类型").NotEmpty(),
		field.JSON("conditions", map[string]interface{}{}).Comment("触发条件").Optional(),
		field.JSON("actions", []map[string]interface{}{}).Comment("执行动作").Optional(),
		field.String("priority").Comment("优先级").Default("medium"),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Int("execution_count").Comment("执行次数").Default(0),
		field.Time("last_executed_at").Comment("最后执行时间").Optional(),
		field.JSON("metadata", map[string]interface{}{}).Comment("元数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (IncidentRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("rule_executions", IncidentRuleExecution.Type).Comment("规则执行记录"),
	}
}
