package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type IncidentRuleExecution struct{ ent.Schema }

func (IncidentRuleExecution) Fields() []ent.Field {
	return []ent.Field{
		field.Int("rule_id").Comment("规则ID").Positive(),
		field.Int("incident_id").Comment("事件ID").Optional(),
		field.String("status").Comment("执行状态").Default("pending"),
		field.Text("result").Comment("执行结果").Optional(),
		field.Text("error_message").Comment("错误信息").Optional(),
		field.Time("started_at").Comment("开始时间").Default(time.Now),
		field.Time("completed_at").Comment("完成时间").Optional(),
		field.Int("execution_time_ms").Comment("执行时间(毫秒)").Optional(),
		field.JSON("input_data", map[string]interface{}{}).Comment("输入数据").Optional(),
		field.JSON("output_data", map[string]interface{}{}).Comment("输出数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (IncidentRuleExecution) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("rule", IncidentRule.Type).
			Ref("rule_executions").
			Field("rule_id").
			Unique().
			Required().
			Comment("规则"),
	}
}
