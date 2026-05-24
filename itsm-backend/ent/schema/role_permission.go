package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// RolePermission holds the schema definition for the RolePermission entity.
// This is a join table for many-to-many relationship between Role and PermissionDefinition.
type RolePermission struct {
	ent.Schema
}

// Fields of the RolePermission.
func (RolePermission) Fields() []ent.Field {
	return []ent.Field{
		field.Int("role_id").Comment("角色ID"),
		field.Int("permission_id").Comment("权限定义ID"),
	}
}

// Edges of the RolePermission.
func (RolePermission) Edges() []ent.Edge {
	return nil
}
