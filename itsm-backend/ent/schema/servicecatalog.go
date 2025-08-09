package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type ServiceCatalog struct{ ent.Schema }

func (ServiceCatalog) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("服务名称").NotEmpty(),
		field.Text("description").Comment("服务描述").Optional(),
		field.String("category").Comment("服务分类").Optional(),
		field.Float("price").Comment("价格").Optional(),
		field.Int("delivery_time").Comment("交付时间（天）").Optional(),
		field.String("status").Comment("状态").Default("active"),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (ServiceCatalog) Edges() []ent.Edge { return nil }
