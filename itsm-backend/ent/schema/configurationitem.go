package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ConfigurationItem holds the schema definition for the ConfigurationItem entity.
type ConfigurationItem struct {
	ent.Schema
}

// Fields of the ConfigurationItem.
func (ConfigurationItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("配置项名称").
			NotEmpty(),
		field.Text("description").
			Comment("配置项描述").
			Optional(),
		field.String("type").
			Comment("配置项类型").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("active"),
		field.String("location").
			Comment("位置").
			Optional(),
		field.String("serial_number").
			Comment("序列号").
			Optional(),
		field.String("model").
			Comment("型号").
			Optional(),
		field.String("vendor").
			Comment("厂商").
			Optional(),
		field.Int("ci_type_id").
			Comment("CI类型ID").
			Positive(),
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

// Edges of the ConfigurationItem.
func (ConfigurationItem) Edges() []ent.Edge {
	return nil
}
