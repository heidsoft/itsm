package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIRelationship CI关系实例
type CIRelationship struct {
	ent.Schema
}

func (CIRelationship) Fields() []ent.Field {
	return []ent.Field{
		field.Int("source_ci_id"),
		field.Int("target_ci_id"),
		field.Int("relationship_type_id"),
		field.JSON("properties", map[string]interface{}{}).Optional(), // 关系属性
		field.String("status").Default("active"),                      // active, inactive
		field.Time("effective_from").Default(time.Now),
		field.Time("effective_to").Optional(),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CIRelationship) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_relationships").Field("tenant_id").Unique().Required(),
		edge.From("source_ci", ConfigurationItem.Type).Ref("outgoing_relationships").Field("source_ci_id").Unique().Required(),
		edge.From("target_ci", ConfigurationItem.Type).Ref("incoming_relationships").Field("target_ci_id").Unique().Required(),
		edge.From("relationship_type", CIRelationshipType.Type).Ref("relationships").Field("relationship_type_id").Unique().Required(),
	}
}

func (CIRelationship) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("source_ci_id"),
		index.Fields("target_ci_id"),
		index.Fields("relationship_type_id"),
		index.Fields("tenant_id", "source_ci_id", "target_ci_id", "relationship_type_id").Unique(),
	}
}
