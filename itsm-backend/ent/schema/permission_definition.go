package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PermissionDefinition holds the schema definition for the PermissionDefinition entity.
type PermissionDefinition struct {
	ent.Schema
}

// Fields of the PermissionDefinition.
func (PermissionDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource").NotEmpty().Comment("资源类型，如 ticket, incident, user"),
		field.String("action").NotEmpty().Comment("操作类型，如 read, write, delete, admin"),
		field.String("description").Optional().Comment("权限描述"),
		field.String("display_name").Optional().Comment("显示名称"),
		field.Int("category").Optional().Comment("分类，用于分组"),
	}
}

// Edges of the PermissionDefinition.
func (PermissionDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role_permissions", RolePermission.Type).Comment("角色权限关联"),
	}
}

// Index of PermissionDefinition.
func (PermissionDefinition) Index() []ent.Index {
	return []ent.Index{
		index.Fields("resource", "action").Unique(),
	}
}
