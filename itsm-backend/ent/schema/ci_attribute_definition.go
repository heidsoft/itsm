package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIAttributeDefinition CI属性定义
type CIAttributeDefinition struct {
	ent.Schema
}

func (CIAttributeDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),                                     // 属性名称
		field.String("display_name").NotEmpty(),                             // 显示名称
		field.Text("description").Optional(),                                // 描述
		field.String("data_type").NotEmpty(),                                // string, number, boolean, date, reference, enum
		field.Bool("is_required").Default(false),                            // 是否必填
		field.Bool("is_unique").Default(false),                              // 是否唯一
		field.String("default_value").Optional(),                            // 默认值
		field.JSON("validation_rules", map[string]interface{}{}).Optional(), // 验证规则
		field.JSON("enum_values", []string{}).Optional(),                    // 枚举值（当data_type为enum时）
		field.String("reference_type").Optional(),                           // 引用类型（当data_type为reference时）
		field.Int("display_order").Default(0),                               // 显示顺序
		field.Bool("is_searchable").Default(true),                           // 是否可搜索
		field.Bool("is_system").Default(false),                              // 是否系统属性
		field.Bool("is_active").Default(true),                               // 是否激活
		field.Int("ci_type_id"),                                             // 关联的CI类型
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CIAttributeDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_attribute_definitions").Field("tenant_id").Unique().Required(),
		edge.From("ci_type", CIType.Type).Ref("attribute_definitions").Field("ci_type_id").Unique().Required(),
	}
}

func (CIAttributeDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("ci_type_id"),
		index.Fields("data_type"),
		index.Fields("tenant_id", "ci_type_id", "name").Unique(),
	}
}
