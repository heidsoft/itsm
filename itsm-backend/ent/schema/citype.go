package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CIType struct{ ent.Schema }

func (CIType) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("类型名称").NotEmpty(),
		field.Text("description").Comment("类型描述").Optional(),
		field.String("icon").Comment("图标").Optional(),
		field.String("color").Comment("颜色").Optional(),
		field.Text("attribute_schema").Comment("属性模式定义").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (CIType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cis", ConfigurationItem.Type),
	}
}
