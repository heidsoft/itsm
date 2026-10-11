package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Menu holds the schema definition for the Menu entity.
type Menu struct {
	ent.Schema
}

// Fields of the Menu.
func (Menu) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("菜单名称").
			NotEmpty(),
		field.String("path").
			Comment("路由路径").
			NotEmpty(),
		field.String("icon").
			Comment("图标名称").
			Optional(),
		field.Int("parent_id").
			Comment("父菜单ID").
			Optional().Nillable(),
		field.String("permission_code").
			Comment("绑定权限代码").
			Optional(),
		field.Int("sort_order").
			Comment("排序").
			Default(0),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Bool("is_visible").
			Comment("是否可见").
			Default(true),
		field.Bool("is_enabled").
			Comment("是否启用").
			Default(true),
		field.String("description").
			Comment("描述").
			Optional(),
	}
}

// Edges of the Menu.
func (Menu) Edges() []ent.Edge {
	return []ent.Edge{
		// 自关联：父菜单
		edge.From("parent", Menu.Type).
			Field("parent_id").
			Ref("children").
			Unique(),
		// 子菜单
		edge.To("children", Menu.Type).
			Comment("子菜单"),
	}
}

// 注意：menus 的 (tenant_id, path) 业务唯一索引不在此声明——entc v0.14.6 对 Menu
// 自关联 edge + 组合唯一索引会组合爆炸（CPU 1200% 不终止，2026-10-11 两次复现）。
// 唯一索引由 migrations/20261011_add_menus_tenant_path_unique.sql 承担：
// bootstrap 顺序是 ent Schema.Create → SQL 迁移，全新安装同样会拿到索引。
// CreateMenu 侧以 Exist 查重 + ent.IsConstraintError 兜底防重复行。
