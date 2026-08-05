package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FieldDefinition 动态字段定义：谁（entity_type+entity_id）拥有哪些自定义字段。
// entity_type 取值：ticket_template | service_catalog_item
type FieldDefinition struct {
	ent.Schema
}

func (FieldDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.String("entity_type").Comment("字段定义归属的实体类型: ticket_template | service_catalog_item").NotEmpty(),
		field.Int("entity_id").Comment("归属实体ID（模板ID 或 服务目录项ID）").Positive(),
		field.String("name").Comment("字段key，如 office_location").NotEmpty(),
		field.String("label").Comment("显示名，如 办公地点").NotEmpty(),
		field.String("field_type").Comment("字段类型: text|textarea|number|date|select|multiselect|boolean|file").NotEmpty(),
		field.Bool("required").Comment("是否必填").Default(false),
		field.JSON("options", []interface{}{}).Comment("select/multiselect 的选项列表 [{label,value}]").Optional(),
		field.Int("sort_order").Comment("显示顺序").Default(0),
		field.JSON("config", map[string]interface{}{}).Comment("预留：校验规则/默认值/显隐条件，v1 不使用").Optional(),
		field.Bool("is_active").Comment("是否启用").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FieldDefinition) Edges() []ent.Edge { return nil }

func (FieldDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "entity_type", "entity_id", "sort_order"),
		index.Fields("tenant_id", "entity_type", "entity_id", "name").Unique(),
	}
}
