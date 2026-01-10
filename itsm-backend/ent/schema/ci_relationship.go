package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

// CIRelationship holds the schema definition for CI relationships.
type CIRelationship struct {
	ent.Schema
}

// Fields of the CIRelationship.
func (CIRelationship) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").
			Comment("关系类型"),
		field.String("description").
			Optional().
			Comment("关系描述"),
		field.Int("parent_id").
			Comment("父CI ID"),
		field.Int("child_id").
			Comment("子CI ID"),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the CIRelationship.
func (CIRelationship) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parent", ConfigurationItem.Type).
			Ref("parent_relations").
			Unique().
			Field("parent_id").
			Required(),
		edge.From("child", ConfigurationItem.Type).
			Ref("child_relations").
			Unique().
			Field("child_id").
			Required(),
	}
}
