package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIType 配置项类型定义
type CIType struct {
	ent.Schema
}

func (CIType) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().Unique(),
		field.String("display_name").NotEmpty(),
		field.Text("description").Optional(),
		field.String("category").NotEmpty(), // infrastructure, application, service, etc.
		field.String("icon").Optional(),
		field.JSON("attribute_schema", map[string]interface{}{}).Optional(), // 动态属性定义
		field.JSON("validation_rules", map[string]interface{}{}).Optional(),
		field.Bool("is_system").Default(false), // 系统预定义类型
		field.Bool("is_active").Default(true),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CIType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_types").Field("tenant_id").Unique().Required(),
		edge.To("configuration_items", ConfigurationItem.Type),
		edge.To("allowed_relationships", CIRelationshipType.Type),
		// 新增：属性定义关系
		edge.To("attribute_definitions", CIAttributeDefinition.Type),
	}
}

func (CIType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("category"),
		index.Fields("tenant_id", "name").Unique(),
	}
}
