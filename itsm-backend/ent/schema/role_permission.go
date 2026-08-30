package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RolePermission holds the schema definition for the RolePermission entity.
// This is a join table for many-to-many relationship between Role and Permission.
type RolePermission struct {
	ent.Schema
}

// Fields of the RolePermission.
func (RolePermission) Fields() []ent.Field {
	return []ent.Field{
		field.Int("role_id").Comment("角色ID"),
		field.Int("permission_id").Comment("权限ID"),
		field.Int("tenant_id").Comment("租户ID"),
	}
}

// Edges of the RolePermission.
func (RolePermission) Edges() []ent.Edge {
	return nil
}

// Index of the RolePermission.
// 唯一索引防止同一角色对同一权限出现重复行（并发授权或手工 SQL 可能造成）。
// 注意：新增索引需跑 entc 重新生成（~6GB 内存前提，见 docs/entc-codegen-runbook.md）。
func (RolePermission) Index() []ent.Index {
	return []ent.Index{
		index.Fields("role_id", "permission_id", "tenant_id").Unique(),
	}
}
