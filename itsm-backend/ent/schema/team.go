package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Team holds the schema definition for the Team entity.
type Team struct {
	ent.Schema
}

// Fields of the Team.
func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("团队名称").
			NotEmpty(),
		field.String("code").
			Comment("团队代码").
			Unique().
			NotEmpty(),
		field.Text("description").
			Comment("团队描述").
			Optional(),
		field.String("status").
			Comment("状态: active, inactive").
			Default("active"),
		field.Int("manager_id").
			Comment("负责人ID").
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

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type).
			Comment("团队成员"),
		edge.To("tags", Tag.Type).
			Comment("团队标签"),
	}
}
