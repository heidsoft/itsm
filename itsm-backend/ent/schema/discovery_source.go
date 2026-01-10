package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiscoverySource holds the schema definition for the DiscoverySource entity.
type DiscoverySource struct {
	ent.Schema
}

// Fields of the DiscoverySource.
func (DiscoverySource) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Comment("来源ID").
			NotEmpty().
			Immutable(),
		field.String("name").
			Comment("来源名称").
			NotEmpty(),
		field.String("source_type").
			Comment("来源类型（agent/api/import/manual）").
			NotEmpty(),
		field.String("provider").
			Comment("云厂商或私有云标识").
			Optional(),
		field.Bool("enabled").
			Comment("是否启用").
			Default(true),
		field.String("description").
			Comment("描述").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Optional().
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

// Edges of the DiscoverySource.
func (DiscoverySource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("jobs", DiscoveryJob.Type),
	}
}

// Indexes of the DiscoverySource.
func (DiscoverySource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "name").Unique(),
	}
}
