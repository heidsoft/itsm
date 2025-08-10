package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

// Workflow holds the schema definition for the Workflow entity.
type Workflow struct {
	ent.Schema
}

// Fields of the Workflow.
func (Workflow) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("工作流名称").
			NotEmpty(),
		field.Text("description").
			Comment("工作流描述").
			Optional(),
		field.String("type").
			Comment("工作流类型").
			Default("ticket"),
		field.JSON("definition", []byte{}).
			Comment("工作流定义"),
		field.String("version").
			Comment("版本号").
			Default("1.0.0"),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
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

// Edges of the Workflow.
func (Workflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("workflow_instances", WorkflowInstance.Type).
			Comment("工作流实例"),
	}
}
