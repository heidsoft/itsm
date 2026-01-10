package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RelationshipType holds the schema definition for the RelationshipType entity.
type RelationshipType struct {
	ent.Schema
}

// Fields of the RelationshipType.
func (RelationshipType) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("关系类型名称").
			NotEmpty(),
		field.Bool("directional").
			Comment("是否有方向").
			Default(true),
		field.String("reverse_name").
			Comment("反向名称").
			Optional(),
		field.String("description").
			Comment("描述").
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

// Indexes of the RelationshipType.
func (RelationshipType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "name").Unique(),
	}
}
