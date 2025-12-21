package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketAutomationRule holds the schema definition for the TicketAutomationRule entity.
type TicketAutomationRule struct {
	ent.Schema
}

// Fields of the TicketAutomationRule.
func (TicketAutomationRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("规则名称").
			NotEmpty(),
		field.Text("description").
			Comment("规则描述").
			Optional(),
		field.Int("priority").
			Comment("优先级（数字越大优先级越高）").
			Default(0),
		field.JSON("conditions", []map[string]interface{}{}).
			Comment("触发条件（JSON数组）").
			Optional(),
		field.JSON("actions", []map[string]interface{}{}).
			Comment("执行动作（JSON数组）").
			Optional(),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
		field.Int("execution_count").
			Comment("执行次数").
			Default(0),
		field.Time("last_executed_at").
			Comment("最后执行时间").
			Optional(),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
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

// Edges of the TicketAutomationRule.
func (TicketAutomationRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("creator", User.Type).
			Unique().
			Required().
			Field("created_by").
			Comment("创建人"),
	}
}
