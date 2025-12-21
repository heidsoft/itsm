package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WorkflowInstance holds the schema definition for the WorkflowInstance entity.
type WorkflowInstance struct {
	ent.Schema
}

// Fields of the WorkflowInstance.
func (WorkflowInstance) Fields() []ent.Field {
	return []ent.Field{
		field.String("status").
			Comment("实例状态").
			Default("running"),
		field.String("current_step").
			Comment("当前步骤").
			Optional(),
		field.JSON("context", []byte{}).
			Comment("执行上下文").
			Optional(),
		field.Int("workflow_id").
			Comment("工作流ID").
			Positive(),
		field.Int("entity_id").
			Comment("关联实体ID").
			Positive(),
		field.String("entity_type").
			Comment("关联实体类型").
			Default("ticket"),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("started_at").
			Comment("开始时间").
			Default(time.Now),
		field.Time("completed_at").
			Comment("完成时间").
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

// Edges of the WorkflowInstance.
func (WorkflowInstance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).
			Ref("workflow_instances").
			Field("workflow_id").
			Unique().
			Required(),
	}
}
