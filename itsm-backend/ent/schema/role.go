package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Role holds the schema definition for the Role entity.
type Role struct {
	ent.Schema
}

// Fields of the Role.
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("角色名称").
			NotEmpty(),
		field.String("code").
			Comment("角色代码").
			NotEmpty(),
		field.String("description").
			Comment("角色描述").
			Optional(),
		field.Bool("is_system").
			Comment("是否系统角色").
			Default(false),
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

// Edges of the Role.
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("permissions", Permission.Type).
			Comment("角色拥有的权限"),
		edge.From("users", User.Type).
			Ref("roles").
			Comment("拥有此角色的用户"),
	}
}
