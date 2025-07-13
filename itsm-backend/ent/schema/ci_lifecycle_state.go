package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CILifecycleState CI生命周期状态记录
type CILifecycleState struct {
	ent.Schema
}

func (CILifecycleState) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ci_id"),
		field.String("state").NotEmpty(), // planned, ordered, received, installed, deployed, active, maintenance, retired
		field.String("sub_state").Optional(),
		field.Text("reason").Optional(),
		field.String("changed_by").NotEmpty(),
		field.Time("changed_at").Default(time.Now),
		field.JSON("metadata", map[string]interface{}{}).Optional(),
		field.Int("tenant_id"),
	}
}

func (CILifecycleState) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_lifecycle_states").Field("tenant_id").Unique().Required(),
		edge.From("configuration_item", ConfigurationItem.Type).Ref("lifecycle_states").Field("ci_id").Unique().Required(),
	}
}

func (CILifecycleState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("ci_id"),
		index.Fields("state"),
		index.Fields("changed_at"),
	}
}
