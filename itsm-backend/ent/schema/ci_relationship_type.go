package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIRelationshipType CI关系类型定义
type CIRelationshipType struct {
	ent.Schema
}

func (CIRelationshipType) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("display_name").NotEmpty(),
		field.Text("description").Optional(),
		field.String("direction").Default("bidirectional"),   // unidirectional, bidirectional
		field.String("cardinality").Default("many_to_many"),  // one_to_one, one_to_many, many_to_many
		field.JSON("source_ci_types", []string{}).Optional(), // 允许的源CI类型
		field.JSON("target_ci_types", []string{}).Optional(), // 允许的目标CI类型
		field.Bool("is_system").Default(false),
		field.Bool("is_active").Default(true),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CIRelationshipType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_relationship_types").Field("tenant_id").Unique().Required(),
		edge.To("relationships", CIRelationship.Type),
	}
}

func (CIRelationshipType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "name").Unique(),
	}
}
