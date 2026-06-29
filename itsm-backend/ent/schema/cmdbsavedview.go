package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CMDBSavedView holds the schema definition for the CMDBSavedView entity.
type CMDBSavedView struct {
	ent.Schema
}

// Fields of the CMDBSavedView.
func (CMDBSavedView) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("视图名称").
			NotEmpty(),
		field.String("description").
			Comment("视图描述").
			Optional(),
		field.JSON("filters", map[string]interface{}{}).
			Comment("保存的过滤条件").
			Optional(),
		field.String("sort_by").
			Comment("排序字段").
			Optional(),
		field.String("sort_order").
			Comment("排序方向：asc/desc").
			Default("desc"),
		field.Bool("is_public").
			Comment("是否公开").
			Default(false),
		field.Int("creator_id").
			Comment("创建人ID").
			Positive(),
		field.String("creator_name").
			Comment("创建人名称").
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

// Edges of the CMDBSavedView.
func (CMDBSavedView) Edges() []ent.Edge {
	return nil
}

// Indexes of the CMDBSavedView.
func (CMDBSavedView) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "name", "creator_id").
			Unique(),
		index.Fields("tenant_id"),
		index.Fields("creator_id"),
		index.Fields("is_public"),
	}
}
