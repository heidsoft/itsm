package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// TicketAssignmentRule holds the schema definition for the TicketAssignmentRule entity.
type TicketAssignmentRule struct {
	ent.Schema
}

// Fields of the TicketAssignmentRule.
func (TicketAssignmentRule) Fields() []ent.Field {
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
			Comment("匹配条件").
			Optional(),
		field.JSON("actions", map[string]interface{}{}).
			Comment("执行动作").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.Int("execution_count").
			Comment("执行次数").
			Default(0),
		field.Time("last_executed_at").
			Comment("最后执行时间").
			Optional(),
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

// Edges of the TicketAssignmentRule.
func (TicketAssignmentRule) Edges() []ent.Edge {
	return nil
}
