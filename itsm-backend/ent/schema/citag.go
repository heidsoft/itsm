package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CITag holds the schema definition for the CITag entity.
type CITag struct {
	ent.Schema
}

// Fields of the CITag.
func (CITag) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			Comment("标签键").
			NotEmpty(),
		field.String("value").
			Comment("标签值").
			Optional(),
		field.String("color").
			Comment("标签颜色").
			Optional(),
		field.String("description").
			Comment("标签描述").
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

// Edges of the CITag.
func (CITag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("cis", ConfigurationItem.Type).
			Ref("tags"),
	}
}

// Indexes of the CITag.
func (CITag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "key", "value").
			Unique(),
	}
}
