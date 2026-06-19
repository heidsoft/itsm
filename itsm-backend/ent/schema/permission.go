package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Permission holds the schema definition for the Permission entity.
type Permission struct {
	ent.Schema
}

// Fields of the Permission.
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("权限代码").
			NotEmpty(),
		field.String("name").
			Comment("权限名称").
			NotEmpty(),
		field.String("description").
			Comment("权限描述").
			Optional(),
		field.String("resource").
			Comment("资源类型").
			NotEmpty(),
		field.String("action").
			Comment("操作类型").
			NotEmpty(),
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

// Edges of the Permission.
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("roles", Role.Type).
			Comment("拥有此权限的角色"),
	}
}
