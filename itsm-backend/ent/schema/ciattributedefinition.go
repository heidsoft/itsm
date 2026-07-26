package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CIAttributeDefinition struct{ ent.Schema }

func (CIAttributeDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("属性名称").NotEmpty(),
		field.String("display_name").Comment("显示名称").NotEmpty(),
		field.Text("description").Comment("属性说明").Optional(),
		field.String("type").Comment("属性类型").NotEmpty(),
		field.Bool("required").Comment("是否必填").Default(false),
		field.Bool("unique").Comment("是否唯一").Default(false),
		field.Text("default_value").Comment("默认值").Optional(),
		field.Text("validation_rules").Comment("验证规则").Optional(),
		field.JSON("enum_values", []string{}).Comment("枚举选项").Optional(),
		field.String("reference_type").Comment("引用目标类型").Optional(),
		field.Int("display_order").Comment("显示顺序").Default(0),
		field.String("group_name").Comment("属性分组").Optional(),
		field.String("placeholder").Comment("输入提示").Optional(),
		field.Text("help_text").Comment("帮助文本").Optional(),
		field.Bool("is_searchable").Comment("是否进入属性检索索引").Default(false),
		field.Bool("is_system").Comment("是否系统属性").Default(false),
		field.Int("ci_type_id").Comment("CI类型ID").Positive(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (CIAttributeDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ci_type", CIType.Type).
			Ref("attribute_definitions").
			Unique().
			Field("ci_type_id").
			Required(),
	}
}

func (CIAttributeDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "ci_type_id", "name").Unique(),
		index.Fields("tenant_id", "ci_type_id", "display_order"),
		index.Fields("tenant_id", "is_searchable"),
	}
}
