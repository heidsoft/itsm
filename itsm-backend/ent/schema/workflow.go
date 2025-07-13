package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

type Workflow struct {
	ent.Schema
}

func (Workflow) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),                    // ticket, change, incident, problem
		field.JSON("definition", map[string]interface{}{}), // 工作流定义
		field.String("status").Default("active"),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Workflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("workflows").Field("tenant_id").Unique().Required(),
		edge.To("flow_instances", FlowInstance.Type),
	}
}
