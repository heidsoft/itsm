package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type CIAttributeDefinition struct{ ent.Schema }

func (CIAttributeDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("属性名称").NotEmpty(),
		field.String("display_name").Comment("显示名称").NotEmpty(),
		field.String("type").Comment("属性类型").NotEmpty(),
		field.Bool("required").Comment("是否必填").Default(false),
		field.Bool("unique").Comment("是否唯一").Default(false),
		field.Text("default_value").Comment("默认值").Optional(),
		field.Text("validation_rules").Comment("验证规则").Optional(),
		field.Int("ci_type_id").Comment("CI类型ID").Positive(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (CIAttributeDefinition) Edges() []ent.Edge { return nil }
